//! The API-key manager: create, list, authenticate, revoke, and rotate keys.
//!
//! This is step 2 of the billing build (docs/design/billing-admin.md section 3).
//! The store owns the SQL; this module owns the crypto and the hot-path map.
//!
//! Security shape (findings S9, S11, S12, S14):
//!   - A raw key carries 128 bits of CSPRNG entropy and the `jg_` prefix.
//!   - The store holds only `HMAC-SHA256(pepper, raw)`, never the raw key and
//!     never a plain SHA-256. The pepper lives outside the database.
//!   - The public id (`k_...`) is distinct from the secret and safe to log.
//!     Admin routes and audit rows reference the id, never the raw key.
//!   - Authentication reads an in-memory map, not the database. Revoke evicts
//!     the key from that map in the same call, so the gate fails closed at once.

use std::collections::HashMap;
use std::sync::RwLock;

use hmac::{Hmac, Mac};
use rand::Rng;
use sha2::Sha256;

use crate::error::{Error, Result};
use crate::store::{unix_now, Store, StoredKey};

type HmacSha256 = Hmac<Sha256>;

/// A hashed key. This is the map key and the value stored in `key_hash`.
type KeyHash = [u8; 32];

/// The actor recorded on audit rows from this manager. The admin API (a later
/// step) will carry a real operator identity; for now every mutation is the
/// operator at the box.
const ACTOR: &str = "operator";

/// The rate policy for a key. Operator keys default to `Unlimited`, because the
/// prepaid balance bounds them, not a throttle (design section 3.2).
#[derive(Debug, Clone, PartialEq, Eq, Default)]
pub enum Rate {
    #[default]
    Unlimited,
    Limited {
        per_second: i64,
        per_day: i64,
    },
}

/// The input to `create`. Sensible defaults: no throttle, not exempt, no trace.
#[derive(Debug, Clone, Default)]
pub struct KeyConfig {
    pub label: String,
    pub account_address: Option<String>,
    pub credit_exempt: bool,
    pub allow_trace: bool,
    pub rate: Rate,
    pub expires_at: Option<i64>,
    // Constraints persisted to the `key_constraint` table.
    pub origins: Vec<String>,
    pub method_allow: Vec<String>,
    pub method_block: Vec<String>,
    pub networks: Vec<String>,
    pub ip_allow: Vec<String>,
    pub ip_deny: Vec<String>,
}

/// A key as seen by admin reads and the hot path. It holds no secret and no
/// hash. The hash is the map key; it never leaves the manager.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct KeyRecord {
    pub id: String,
    pub label: String,
    pub account_address: Option<String>,
    pub credit_exempt: bool,
    pub allow_trace: bool,
    pub rate: Rate,
    pub created_at: i64,
    pub disabled_at: Option<i64>,
    pub expires_at: Option<i64>,
}

impl KeyRecord {
    /// Build a record from a stored row. Drops the hash.
    fn from_stored(s: &StoredKey) -> KeyRecord {
        let rate = if s.rate_unlimited {
            Rate::Unlimited
        } else {
            Rate::Limited {
                per_second: s.per_second_limit.unwrap_or(0),
                per_day: s.per_day_limit.unwrap_or(0),
            }
        };
        KeyRecord {
            id: s.id.clone(),
            label: s.label.clone(),
            account_address: s.account_address.clone(),
            credit_exempt: s.credit_exempt,
            allow_trace: s.allow_trace,
            rate,
            created_at: s.created_at,
            disabled_at: s.disabled_at,
            expires_at: s.expires_at,
        }
    }

    /// A key is usable when it is not revoked and not past its expiry.
    fn is_active(&self, now: i64) -> bool {
        self.disabled_at.is_none() && self.expires_at.is_none_or(|e| e > now)
    }
}

/// The key manager. It owns the store and the in-memory key map.
pub struct KeyManager {
    store: Store,
    pepper: Vec<u8>,
    /// hash -> record. Read on the hot path, updated on every mutation.
    /// The lock is here from the start, because the admin API is multi-threaded.
    map: RwLock<HashMap<KeyHash, KeyRecord>>,
}

impl KeyManager {
    /// Build the manager and load the map from the database. Rejects an empty
    /// pepper, because a keyed hash with no key is not keyed at all.
    pub fn new(store: Store, pepper: &[u8]) -> Result<KeyManager> {
        if pepper.is_empty() {
            return Err(Error::MissingPepper);
        }
        let mut map = HashMap::new();
        for stored in store.load_all_keys()? {
            // Load only usable-by-revoke keys into the hot map. Expiry is
            // checked live at authenticate time.
            if stored.disabled_at.is_some() {
                continue;
            }
            let hash = to_hash(&stored.key_hash);
            map.insert(hash, KeyRecord::from_stored(&stored));
        }
        Ok(KeyManager {
            store,
            pepper: pepper.to_vec(),
            map: RwLock::new(map),
        })
    }

    /// Create a key. Writes the row, its constraints, and an audit row, then
    /// adds it to the map. Returns the public id and the raw key. The raw key is
    /// shown once and never stored.
    pub fn create(&self, config: KeyConfig) -> Result<(String, String)> {
        let id = new_public_id();
        let raw = new_raw_key();
        let hash = self.hash(&raw);
        let now = unix_now();

        let (rate_unlimited, per_second_limit, per_day_limit) = match config.rate {
            Rate::Unlimited => (true, None, None),
            Rate::Limited {
                per_second,
                per_day,
            } => (false, Some(per_second), Some(per_day)),
        };

        let stored = StoredKey {
            id: id.clone(),
            key_hash: hash.to_vec(),
            label: config.label.clone(),
            account_address: config.account_address.clone(),
            credit_exempt: config.credit_exempt,
            allow_trace: config.allow_trace,
            rate_unlimited,
            per_second_limit,
            per_day_limit,
            created_at: now,
            disabled_at: None,
            expires_at: config.expires_at,
        };

        // Database first, then the map. A failed write must not leave a phantom
        // entry in the map.
        self.store.insert_key(&stored)?;
        for (kind, value) in config.constraints() {
            self.store.insert_constraint(&id, kind, &value)?;
        }
        self.store
            .append_audit(ACTOR, "key.create", Some(&id), Some(&config.label))?;

        let record = KeyRecord::from_stored(&stored);
        self.map.write().expect("key map lock").insert(hash, record);

        Ok((id, raw))
    }

    /// List every key, including revoked ones, for the admin surface. Reads the
    /// database, not the hot map, so revoked keys still show. Returns no secret
    /// and no hash.
    pub fn list(&self) -> Result<Vec<KeyRecord>> {
        let keys = self
            .store
            .load_all_keys()?
            .iter()
            .map(KeyRecord::from_stored)
            .collect();
        Ok(keys)
    }

    /// Resolve a raw key to its record. This is the hot path: it hashes the key
    /// and reads the in-memory map, never the database. Returns a record only
    /// when the key is active.
    pub fn authenticate(&self, raw_key: &str) -> Option<KeyRecord> {
        let hash = self.hash(raw_key);
        let map = self.map.read().expect("key map lock");
        let record = map.get(&hash)?;
        if record.is_active(unix_now()) {
            Some(record.clone())
        } else {
            None
        }
    }

    /// Revoke a key. Sets `disabled_at`, writes an audit row, and evicts the key
    /// from the map in the same call. The next `authenticate` fails closed.
    pub fn revoke(&self, id: &str) -> Result<()> {
        let now = unix_now();
        if self.store.set_key_disabled(id, now)? == 0 {
            return Err(Error::KeyNotFound(id.to_string()));
        }
        self.store
            .append_audit(ACTOR, "key.revoke", Some(id), None)?;
        self.evict_id(id);
        Ok(())
    }

    /// Rotate a key. Issues a new raw key, swaps the stored hash, refreshes the
    /// map, and writes an audit row. Returns the new raw key.
    ///
    // TODO: the design wants a short dual-validity grace TTL so a client can
    // roll a key with no outage (billing-admin.md section 3.3). This is an
    // immediate swap: the old key stops working at once. Adding the grace TTL
    // needs a schema column for the prior hash and its expiry; do not change the
    // schema for it in this step.
    pub fn rotate(&self, id: &str) -> Result<String> {
        let raw = new_raw_key();
        let hash = self.hash(&raw);

        if self.store.update_key_hash(id, &hash)? == 0 {
            return Err(Error::KeyNotFound(id.to_string()));
        }
        self.store
            .append_audit(ACTOR, "key.rotate", Some(id), None)?;

        // Reload the row so the map matches the database exactly.
        let stored = self
            .store
            .load_key(id)?
            .ok_or_else(|| Error::KeyNotFound(id.to_string()))?;
        let record = KeyRecord::from_stored(&stored);

        let mut map = self.map.write().expect("key map lock");
        remove_by_id(&mut map, id);
        map.insert(hash, record);

        Ok(raw)
    }

    /// The number of keys in the hot map. There is no cap on the key count; this
    /// exposes the memory cost so the operator can see it (design section 3.1).
    pub fn key_count(&self) -> usize {
        self.map.read().expect("key map lock").len()
    }

    /// HMAC-SHA256 of the raw key under the pepper. This is the only key
    /// fingerprint the service ever stores or compares.
    fn hash(&self, raw_key: &str) -> KeyHash {
        let mut mac = HmacSha256::new_from_slice(&self.pepper).expect("HMAC takes any key length");
        mac.update(raw_key.as_bytes());
        mac.finalize().into_bytes().into()
    }

    /// Evict the entry for a public id from the map, holding the write lock.
    fn evict_id(&self, id: &str) {
        let mut map = self.map.write().expect("key map lock");
        remove_by_id(&mut map, id);
    }
}

/// The constraint (kind, value) pairs for a config, in the schema's vocabulary.
impl KeyConfig {
    fn constraints(&self) -> Vec<(&'static str, String)> {
        let mut out = Vec::new();
        for v in &self.origins {
            out.push(("origin", v.clone()));
        }
        for v in &self.method_allow {
            out.push(("method_allow", v.clone()));
        }
        for v in &self.method_block {
            out.push(("method_block", v.clone()));
        }
        for v in &self.networks {
            out.push(("network", v.clone()));
        }
        for v in &self.ip_allow {
            out.push(("ip_allow", v.clone()));
        }
        for v in &self.ip_deny {
            out.push(("ip_deny", v.clone()));
        }
        out
    }
}

/// Remove the map entry whose record has this id. The map is keyed by hash, so
/// this scans for the id. It runs off the hot path (revoke and rotate only).
fn remove_by_id(map: &mut HashMap<KeyHash, KeyRecord>, id: &str) {
    if let Some(hash) = map.iter().find(|(_, r)| r.id == id).map(|(h, _)| *h) {
        map.remove(&hash);
    }
}

/// Copy a 32-byte hash slice into a fixed array for the map key.
fn to_hash(bytes: &[u8]) -> KeyHash {
    let mut out = [0u8; 32];
    out.copy_from_slice(bytes);
    out
}

/// A new raw key: `jg_` plus base58 of 16 CSPRNG bytes (128-bit entropy, S12).
fn new_raw_key() -> String {
    let mut bytes = [0u8; 16];
    fill_random(&mut bytes);
    format!("jg_{}", bs58::encode(bytes).into_string())
}

/// A new public id: `k_` plus base58 of 6 CSPRNG bytes. Safe to log.
fn new_public_id() -> String {
    let mut bytes = [0u8; 6];
    fill_random(&mut bytes);
    format!("k_{}", bs58::encode(bytes).into_string())
}

/// Fill a buffer from the thread CSPRNG. `ThreadRng` is a ChaCha-based generator
/// seeded from the operating system, so it is cryptographically secure.
fn fill_random(buf: &mut [u8]) {
    rand::rng().fill_bytes(buf);
}

#[cfg(test)]
mod tests {
    use super::*;
    use sha2::{Digest, Sha256};

    const PEPPER: &[u8] = b"unit-test-pepper";

    fn manager() -> KeyManager {
        let store = Store::open_in_memory().unwrap();
        KeyManager::new(store, PEPPER).unwrap()
    }

    #[test]
    fn create_then_authenticate_round_trips() {
        let km = manager();
        let cfg = KeyConfig {
            label: "round-trip".into(),
            ..Default::default()
        };
        let (id, raw) = km.create(cfg).unwrap();

        assert!(raw.starts_with("jg_"), "raw key must carry the jg_ prefix");
        assert!(id.starts_with("k_"), "public id must carry the k_ prefix");

        let record = km.authenticate(&raw).expect("the fresh key must resolve");
        assert_eq!(record.id, id);
        assert_eq!(record.label, "round-trip");
    }

    #[test]
    fn authenticate_rejects_an_unknown_key() {
        let km = manager();
        km.create(KeyConfig::default()).unwrap();
        assert!(km.authenticate("jg_not_a_real_key").is_none());
        assert!(km.authenticate("garbage").is_none());
        assert!(km.authenticate("").is_none());
    }

    #[test]
    fn revoke_fails_closed_on_the_next_call() {
        let km = manager();
        let (id, raw) = km.create(KeyConfig::default()).unwrap();
        assert!(km.authenticate(&raw).is_some());

        km.revoke(&id).unwrap();
        // The very next authenticate must fail. Eviction is synchronous.
        assert!(km.authenticate(&raw).is_none());
        assert_eq!(km.key_count(), 0);
    }

    #[test]
    fn revoke_unknown_id_errors() {
        let km = manager();
        let err = km.revoke("k_missing").unwrap_err();
        assert!(matches!(err, Error::KeyNotFound(_)));
    }

    #[test]
    fn rotate_swaps_the_valid_key() {
        let km = manager();
        let (id, old_raw) = km.create(KeyConfig::default()).unwrap();

        let new_raw = km.rotate(&id).unwrap();
        assert_ne!(old_raw, new_raw);
        assert!(new_raw.starts_with("jg_"));

        // The old key stops working; the new key works. Same id throughout.
        assert!(km.authenticate(&old_raw).is_none());
        let record = km.authenticate(&new_raw).expect("new key must resolve");
        assert_eq!(record.id, id);
        assert_eq!(km.key_count(), 1);
    }

    #[test]
    fn stored_hash_is_hmac_not_raw_and_not_plain_sha256() {
        let store = Store::open_in_memory().unwrap();
        let km = KeyManager::new(store, PEPPER).unwrap();
        let (id, raw) = km.create(KeyConfig::default()).unwrap();

        let stored = km.store.load_key(&id).unwrap().unwrap();

        // The expected keyed hash.
        let mut mac = HmacSha256::new_from_slice(PEPPER).unwrap();
        mac.update(raw.as_bytes());
        let expected: [u8; 32] = mac.finalize().into_bytes().into();
        assert_eq!(stored.key_hash.as_slice(), expected.as_slice());

        // Not the raw key.
        assert_ne!(stored.key_hash.as_slice(), raw.as_bytes());

        // Not a plain SHA-256 of the raw key.
        let plain = Sha256::digest(raw.as_bytes());
        assert_ne!(
            stored.key_hash.as_slice(),
            plain.as_slice(),
            "the hash must be HMAC, not a plain SHA-256"
        );
    }

    #[test]
    fn reboot_reloads_keys_from_the_database() {
        // A temp file DB, so a second manager can reopen the same store.
        let mut path = std::env::temp_dir();
        let unique = format!(
            "jumpgate-keys-test-{}-{}.db",
            std::process::id(),
            unix_now()
        );
        path.push(unique);

        let (id, raw);
        {
            let store = Store::open(&path).unwrap();
            let km = KeyManager::new(store, PEPPER).unwrap();
            let out = km.create(KeyConfig::default()).unwrap();
            id = out.0;
            raw = out.1;
        }

        // Reopen the same file with a fresh manager.
        let store = Store::open(&path).unwrap();
        let km = KeyManager::new(store, PEPPER).unwrap();
        assert_eq!(km.key_count(), 1, "the reloaded map must hold the key");
        let record = km.authenticate(&raw).expect("reloaded key must resolve");
        assert_eq!(record.id, id);

        let _ = std::fs::remove_file(&path);
        let _ = std::fs::remove_file(format!("{}-wal", path.display()));
        let _ = std::fs::remove_file(format!("{}-shm", path.display()));
    }

    #[test]
    fn a_created_key_defaults_to_unlimited_rate() {
        let km = manager();
        let (_, raw) = km.create(KeyConfig::default()).unwrap();
        let record = km.authenticate(&raw).unwrap();
        assert_eq!(record.rate, Rate::Unlimited);
    }

    #[test]
    fn empty_pepper_is_rejected() {
        let store = Store::open_in_memory().unwrap();
        match KeyManager::new(store, b"") {
            Err(Error::MissingPepper) => {}
            _ => panic!("an empty pepper must be rejected"),
        }
    }
}
