//! The SQLite store: open, migrate, seed default pricing, and the price lookup.
//!
//! This is step 1 of the billing build (docs/design/billing-admin.md section 6).
//! It owns the schema and the read side of pricing. The key manager, the admin
//! API, and the hot path build on top of it.

use std::path::Path;
use std::time::{Duration, SystemTime, UNIX_EPOCH};

use rusqlite::{params, Connection, OptionalExtension};

use crate::error::Result;
use crate::pricing;

/// One `project_key` row, with the hash. The store owns the hash; higher layers
/// hold the public id instead. Never log or return `key_hash`.
#[derive(Debug, Clone)]
pub struct StoredKey {
    pub id: String,
    pub key_hash: Vec<u8>,
    pub label: String,
    pub account_address: Option<String>,
    pub credit_exempt: bool,
    pub allow_trace: bool,
    pub rate_unlimited: bool,
    pub per_second_limit: Option<i64>,
    pub per_day_limit: Option<i64>,
    pub created_at: i64,
    pub disabled_at: Option<i64>,
    pub expires_at: Option<i64>,
}

/// One `audit_log` row, newest-first when read. Safe to return to the admin
/// surface: it holds no secret and no key hash.
#[derive(Debug, Clone)]
pub struct AuditRow {
    pub id: i64,
    pub ts: i64,
    pub actor: String,
    pub action: String,
    pub target: Option<String>,
    pub detail: Option<String>,
}

/// The current unix time in seconds.
pub(crate) fn unix_now() -> i64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|d| d.as_secs() as i64)
        .unwrap_or(0)
}

/// Bump this when the schema changes. It is written to `PRAGMA user_version`.
const SCHEMA_VERSION: i64 = 1;
const SCHEMA: &str = include_str!("schema.sql");

/// A handle to the billing database.
pub struct Store {
    conn: Connection,
}

impl Store {
    /// Open the store at `path`, run the schema, and seed default pricing.
    /// The file is set to owner-only (0600) on unix.
    pub fn open(path: &Path) -> Result<Store> {
        let conn = Connection::open(path)?;
        let store = Store::init(conn)?;
        set_owner_only(path)?;
        Ok(store)
    }

    /// Open an in-memory store. Used by tests.
    pub fn open_in_memory() -> Result<Store> {
        Store::init(Connection::open_in_memory()?)
    }

    fn init(conn: Connection) -> Result<Store> {
        // WAL keeps reads and the single writer from blocking each other.
        // synchronous=NORMAL is durable enough with WAL and much faster.
        conn.execute_batch(
            "PRAGMA journal_mode = WAL;\
             PRAGMA synchronous = NORMAL;\
             PRAGMA foreign_keys = ON;",
        )?;
        conn.busy_timeout(Duration::from_secs(5))?;
        conn.execute_batch(SCHEMA)?;
        conn.pragma_update(None, "user_version", SCHEMA_VERSION)?;
        let store = Store { conn };
        store.seed_default_pricing()?;
        Ok(store)
    }

    /// Insert the default price rows. `INSERT OR IGNORE` keeps an operator edit,
    /// so re-running on boot never overwrites a changed price.
    fn seed_default_pricing(&self) -> Result<()> {
        let mut stmt = self.conn.prepare(
            "INSERT OR IGNORE INTO method_pricing (method, chain_id, credits_per_request) \
             VALUES (?1, 0, ?2)",
        )?;
        for (method, credits) in pricing::DEFAULT_PRICES {
            stmt.execute(params![method, credits])?;
        }
        Ok(())
    }

    /// The price of one method on one chain, in credits.
    /// Precedence: exact (method, chain), then (method, any), then the default
    /// row for the chain, then the global default, then `DEFAULT_CU`.
    pub fn price_of(&self, method: &str, chain_id: i64) -> Result<i64> {
        let lookups = [(method, chain_id), (method, 0), ("*", chain_id), ("*", 0)];
        for (m, c) in lookups {
            if let Some(price) = self.lookup_price(m, c)? {
                return Ok(price);
            }
        }
        Ok(pricing::DEFAULT_CU)
    }

    fn lookup_price(&self, method: &str, chain_id: i64) -> Result<Option<i64>> {
        let price = self
            .conn
            .query_row(
                "SELECT credits_per_request FROM method_pricing \
                 WHERE method = ?1 AND chain_id = ?2",
                params![method, chain_id],
                |row| row.get::<_, i64>(0),
            )
            .optional()?;
        Ok(price)
    }

    /// The number of price rows in the store.
    pub fn price_count(&self) -> Result<i64> {
        let n = self
            .conn
            .query_row("SELECT COUNT(*) FROM method_pricing", [], |row| row.get(0))?;
        Ok(n)
    }

    /// Insert one price row, or overwrite the credits when the (method, chain_id)
    /// row exists. The caller rejects a non-positive price first; the CHECK is the
    /// last line of defence, not the gate.
    pub fn upsert_price(&self, method: &str, chain_id: i64, credits: i64) -> Result<()> {
        self.conn.execute(
            "INSERT INTO method_pricing (method, chain_id, credits_per_request) \
             VALUES (?1, ?2, ?3) \
             ON CONFLICT(method, chain_id) \
             DO UPDATE SET credits_per_request = excluded.credits_per_request",
            params![method, chain_id, credits],
        )?;
        Ok(())
    }

    /// Load every price row as (method, chain_id, credits). The price book builds
    /// its in-memory map from this at boot.
    pub fn all_prices(&self) -> Result<Vec<(String, i64, i64)>> {
        let mut stmt = self
            .conn
            .prepare("SELECT method, chain_id, credits_per_request FROM method_pricing")?;
        let rows = stmt.query_map([], |row| {
            Ok((
                row.get::<_, String>(0)?,
                row.get::<_, i64>(1)?,
                row.get::<_, i64>(2)?,
            ))
        })?;
        let mut out = Vec::new();
        for row in rows {
            out.push(row?);
        }
        Ok(out)
    }

    // --- API keys ---------------------------------------------------------

    /// Insert one key row. The caller supplies the hash; the store never sees
    /// the raw key.
    pub fn insert_key(&self, k: &StoredKey) -> Result<()> {
        self.conn.execute(
            "INSERT INTO project_key \
             (id, key_hash, label, account_address, credit_exempt, allow_trace, \
              rate_unlimited, per_second_limit, per_day_limit, created_at, \
              disabled_at, expires_at) \
             VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9, ?10, ?11, ?12)",
            params![
                k.id,
                k.key_hash,
                k.label,
                k.account_address,
                k.credit_exempt,
                k.allow_trace,
                k.rate_unlimited,
                k.per_second_limit,
                k.per_day_limit,
                k.created_at,
                k.disabled_at,
                k.expires_at,
            ],
        )?;
        Ok(())
    }

    /// Load every key row. The key manager builds its in-memory map from this at
    /// boot.
    pub fn load_all_keys(&self) -> Result<Vec<StoredKey>> {
        let mut stmt = self.conn.prepare(
            "SELECT id, key_hash, label, account_address, credit_exempt, allow_trace, \
                    rate_unlimited, per_second_limit, per_day_limit, created_at, \
                    disabled_at, expires_at \
             FROM project_key",
        )?;
        let rows = stmt.query_map([], row_to_key)?;
        let mut keys = Vec::new();
        for row in rows {
            keys.push(row?);
        }
        Ok(keys)
    }

    /// Load one key row by its keyed hash. The key manager's hot map already
    /// answers "is this key active", so this exists only for the failure path
    /// of an authenticate check, to tell "no such key" from "this key is
    /// revoked" (see `keys::KeyManager::authenticate_status`).
    pub fn find_key_by_hash(&self, key_hash: &[u8]) -> Result<Option<StoredKey>> {
        let key = self
            .conn
            .query_row(
                "SELECT id, key_hash, label, account_address, credit_exempt, allow_trace, \
                        rate_unlimited, per_second_limit, per_day_limit, created_at, \
                        disabled_at, expires_at \
                 FROM project_key WHERE key_hash = ?1",
                params![key_hash],
                row_to_key,
            )
            .optional()?;
        Ok(key)
    }

    /// Load one key row by public id.
    pub fn load_key(&self, id: &str) -> Result<Option<StoredKey>> {
        let key = self
            .conn
            .query_row(
                "SELECT id, key_hash, label, account_address, credit_exempt, allow_trace, \
                        rate_unlimited, per_second_limit, per_day_limit, created_at, \
                        disabled_at, expires_at \
                 FROM project_key WHERE id = ?1",
                params![id],
                row_to_key,
            )
            .optional()?;
        Ok(key)
    }

    /// Mark a key revoked. Returns the number of rows changed, so the caller can
    /// tell an unknown id (0) from a real revoke (1).
    pub fn set_key_disabled(&self, id: &str, ts: i64) -> Result<usize> {
        let n = self.conn.execute(
            "UPDATE project_key SET disabled_at = ?2 WHERE id = ?1",
            params![id, ts],
        )?;
        Ok(n)
    }

    /// Update the mutable fields of one key row. It never touches the id, the
    /// hash, `created_at`, or `disabled_at`; those change through their own paths
    /// (rotate, revoke). Returns the number of rows changed, so the caller can
    /// tell an unknown id (0) from a real update (1).
    pub fn update_key_fields(&self, k: &StoredKey) -> Result<usize> {
        let n = self.conn.execute(
            "UPDATE project_key SET \
                 label = ?2, account_address = ?3, credit_exempt = ?4, \
                 allow_trace = ?5, rate_unlimited = ?6, per_second_limit = ?7, \
                 per_day_limit = ?8, expires_at = ?9 \
             WHERE id = ?1",
            params![
                k.id,
                k.label,
                k.account_address,
                k.credit_exempt,
                k.allow_trace,
                k.rate_unlimited,
                k.per_second_limit,
                k.per_day_limit,
                k.expires_at,
            ],
        )?;
        Ok(n)
    }

    /// Swap the stored hash for a rotated key. Returns the number of rows
    /// changed.
    pub fn update_key_hash(&self, id: &str, key_hash: &[u8]) -> Result<usize> {
        let n = self.conn.execute(
            "UPDATE project_key SET key_hash = ?2 WHERE id = ?1",
            params![id, key_hash],
        )?;
        Ok(n)
    }

    /// Insert one per-key constraint row (origin, method rule, network, IP rule).
    pub fn insert_constraint(&self, key_id: &str, kind: &str, value: &str) -> Result<()> {
        self.conn.execute(
            "INSERT INTO key_constraint (key_id, kind, value) VALUES (?1, ?2, ?3)",
            params![key_id, kind, value],
        )?;
        Ok(())
    }

    /// List the (kind, value) constraints for a key.
    pub fn list_key_constraints(&self, key_id: &str) -> Result<Vec<(String, String)>> {
        let mut stmt = self
            .conn
            .prepare("SELECT kind, value FROM key_constraint WHERE key_id = ?1")?;
        let rows = stmt.query_map(params![key_id], |row| {
            Ok((row.get::<_, String>(0)?, row.get::<_, String>(1)?))
        })?;
        let mut out = Vec::new();
        for row in rows {
            out.push(row?);
        }
        Ok(out)
    }

    /// Append one audit row. Every key mutation writes one. The trail is
    /// append-only.
    pub fn append_audit(
        &self,
        actor: &str,
        action: &str,
        target: Option<&str>,
        detail: Option<&str>,
    ) -> Result<()> {
        self.conn.execute(
            "INSERT INTO audit_log (ts, actor, action, target, detail) \
             VALUES (?1, ?2, ?3, ?4, ?5)",
            params![unix_now(), actor, action, target, detail],
        )?;
        Ok(())
    }

    /// Read the audit trail, newest first, capped at `limit` rows. The admin
    /// surface reads this. It returns no secret.
    pub fn read_audit(&self, limit: i64) -> Result<Vec<AuditRow>> {
        let mut stmt = self.conn.prepare(
            "SELECT id, ts, actor, action, target, detail \
             FROM audit_log ORDER BY id DESC LIMIT ?1",
        )?;
        let rows = stmt.query_map(params![limit], |row| {
            Ok(AuditRow {
                id: row.get(0)?,
                ts: row.get(1)?,
                actor: row.get(2)?,
                action: row.get(3)?,
                target: row.get(4)?,
                detail: row.get(5)?,
            })
        })?;
        let mut out = Vec::new();
        for row in rows {
            out.push(row?);
        }
        Ok(out)
    }

    // --- accounts: reserve and settle a credit lease -----------------------
    //
    // The design (keyed-access design doc, section 8) forbids caching a credit
    // balance: a stale balance is money. So the relay leases a block of
    // credits instead — it reserves up to N up front, spends them at wire
    // speed with no further store round trip, then settles back what it did
    // not use. `reserve` and `settle` are that lease's two verbs. Both run as
    // one SQL statement each, so two concurrent reserves against the same
    // account can never both grant credits that were already granted to the
    // other; there is no read-then-write gap for a race to land in.

    /// Insert or update an account's balance row. The reserve and settle
    /// routes only ever adjust an existing row; they never create one, so
    /// this is how a row starts to exist. Used by tests today; a deposit step
    /// will call it too once that lands.
    pub fn upsert_account(
        &self,
        address: &str,
        credits_remaining: i64,
        credits_reserved: i64,
        escrow_ceiling: i64,
    ) -> Result<()> {
        self.conn.execute(
            "INSERT INTO account \
                 (address, credits_remaining, credits_reserved, escrow_ceiling, updated_at) \
             VALUES (?1, ?2, ?3, ?4, ?5) \
             ON CONFLICT(address) DO UPDATE SET \
                 credits_remaining = excluded.credits_remaining, \
                 credits_reserved = excluded.credits_reserved, \
                 escrow_ceiling = excluded.escrow_ceiling, \
                 updated_at = excluded.updated_at",
            params![address, credits_remaining, credits_reserved, escrow_ceiling, unix_now()],
        )?;
        Ok(())
    }

    /// Read one account's balance row, for tests and for a caller that wants
    /// the raw numbers rather than a lease outcome.
    pub fn get_account(&self, address: &str) -> Result<Option<(i64, i64, i64)>> {
        let row = self
            .conn
            .query_row(
                "SELECT credits_remaining, credits_reserved, escrow_ceiling \
                 FROM account WHERE address = ?1",
                params![address],
                |row| {
                    Ok((
                        row.get::<_, i64>(0)?,
                        row.get::<_, i64>(1)?,
                        row.get::<_, i64>(2)?,
                    ))
                },
            )
            .optional()?;
        Ok(row)
    }

    /// Reserve up to `credits` credits for `address`, moving them from
    /// `credits_remaining` into `credits_reserved`. Returns the amount
    /// actually granted — 0 or more, never more than `credits` — or `None`
    /// when the account does not exist. A 0 grant is a normal answer: it
    /// means the account is out of credits, not an error.
    ///
    /// The grant is capped two ways at once: by what `credits_remaining`
    /// actually holds, and by the escrow ceiling's headroom
    /// (`escrow_ceiling - credits_reserved`) when `escrow_ceiling` is set.
    /// `escrow_ceiling = 0` means "no ceiling", matching the same
    /// zero-means-any-chain convention `method_pricing` already uses, so a
    /// freshly seeded account (ceiling 0 by schema default) is not
    /// accidentally reservation-locked at zero.
    ///
    /// The `before` CTE is `MATERIALIZED` on purpose, not a style choice.
    /// SQLite evaluates a `RETURNING` expression against the row's
    /// post-update state, so a plain (non-materialized) CTE referencing this
    /// same table would be re-run there and see the balance this very
    /// statement already debited — recomputing "granted" from the wrong,
    /// already-spent number. Materializing snapshots `before` once, at the
    /// pre-update state, so the `SET` clause and the `RETURNING` clause agree
    /// on the same granted amount. This was verified against a live SQLite
    /// database before landing, not just reasoned about.
    pub fn reserve(&self, address: &str, credits: i64) -> Result<Option<i64>> {
        let granted = self
            .conn
            .query_row(
                "WITH before AS MATERIALIZED ( \
                     SELECT credits_remaining AS remaining, credits_reserved AS reserved, \
                            escrow_ceiling AS ceiling \
                     FROM account WHERE address = ?2 \
                 ) \
                 UPDATE account \
                 SET credits_remaining = credits_remaining - ( \
                         SELECT MIN(?1, remaining, \
                             CASE WHEN ceiling = 0 THEN ?1 ELSE MAX(ceiling - reserved, 0) END) \
                         FROM before), \
                     credits_reserved = credits_reserved + ( \
                         SELECT MIN(?1, remaining, \
                             CASE WHEN ceiling = 0 THEN ?1 ELSE MAX(ceiling - reserved, 0) END) \
                         FROM before), \
                     updated_at = ?3 \
                 WHERE address = ?2 \
                 RETURNING ( \
                     SELECT MIN(?1, remaining, \
                         CASE WHEN ceiling = 0 THEN ?1 ELSE MAX(ceiling - reserved, 0) END) \
                     FROM before)",
                params![credits, address, unix_now()],
                |row| row.get::<_, i64>(0),
            )
            .optional()?;
        Ok(granted)
    }

    /// Settle a previous reservation: remove `reserved` from
    /// `credits_reserved`, and return the unspent remainder
    /// (`reserved - spent`) to `credits_remaining`. The caller (the admin API
    /// handler) has already rejected `spent > reserved` and any negative
    /// amount before this runs; this is the write, not the validation.
    ///
    /// `credits_reserved` is clamped at 0 with `MAX(...)`. A relay that
    /// settles more than the account currently shows reserved — a double
    /// settle, or a bug upstream — must not drive the reservation negative.
    /// The unspent credit still returns to `credits_remaining` regardless, so
    /// a mismatched settle loses no credit; it only stops double-draining
    /// `credits_reserved`.
    ///
    /// Returns the new `(credits_remaining, credits_reserved)`, or `None`
    /// when the account does not exist.
    pub fn settle(&self, address: &str, spent: i64, reserved: i64) -> Result<Option<(i64, i64)>> {
        let row = self
            .conn
            .query_row(
                "UPDATE account \
                 SET credits_reserved = MAX(credits_reserved - ?1, 0), \
                     credits_remaining = credits_remaining + (?1 - ?2), \
                     updated_at = ?4 \
                 WHERE address = ?3 \
                 RETURNING credits_remaining, credits_reserved",
                params![reserved, spent, address, unix_now()],
                |row| Ok((row.get::<_, i64>(0)?, row.get::<_, i64>(1)?)),
            )
            .optional()?;
        Ok(row)
    }
}

/// Map one SQL row to a `StoredKey`.
fn row_to_key(row: &rusqlite::Row) -> rusqlite::Result<StoredKey> {
    Ok(StoredKey {
        id: row.get(0)?,
        key_hash: row.get(1)?,
        label: row.get(2)?,
        account_address: row.get(3)?,
        credit_exempt: row.get(4)?,
        allow_trace: row.get(5)?,
        rate_unlimited: row.get(6)?,
        per_second_limit: row.get(7)?,
        per_day_limit: row.get(8)?,
        created_at: row.get(9)?,
        disabled_at: row.get(10)?,
        expires_at: row.get(11)?,
    })
}

/// Restrict a file to the owner (0600). A co-tenant process must not read the DB.
#[cfg(unix)]
fn set_owner_only(path: &Path) -> Result<()> {
    use std::os::unix::fs::PermissionsExt;
    std::fs::set_permissions(path, std::fs::Permissions::from_mode(0o600))?;
    Ok(())
}

#[cfg(not(unix))]
fn set_owner_only(_path: &Path) -> Result<()> {
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn seeds_the_default_table() {
        let store = Store::open_in_memory().unwrap();
        // one row per default entry
        assert_eq!(
            store.price_count().unwrap(),
            pricing::DEFAULT_PRICES.len() as i64
        );
    }

    #[test]
    fn prices_known_methods_from_the_table() {
        let store = Store::open_in_memory().unwrap();
        assert_eq!(store.price_of("eth_call", 1).unwrap(), 20);
        assert_eq!(store.price_of("eth_chainId", 1).unwrap(), 1);
        assert_eq!(store.price_of("eth_getLogs", 1).unwrap(), 75);
        assert_eq!(store.price_of("debug_traceTransaction", 1).unwrap(), 500);
    }

    #[test]
    fn unknown_method_falls_back_to_the_default() {
        let store = Store::open_in_memory().unwrap();
        assert_eq!(store.price_of("foo_bar", 1).unwrap(), pricing::DEFAULT_CU);
    }

    #[test]
    fn exact_chain_price_beats_the_any_chain_row() {
        let store = Store::open_in_memory().unwrap();
        // an operator override for eth_call on chain 369
        store
            .conn
            .execute(
                "INSERT INTO method_pricing (method, chain_id, credits_per_request) \
                 VALUES ('eth_call', 369, 40)",
                [],
            )
            .unwrap();
        assert_eq!(store.price_of("eth_call", 369).unwrap(), 40); // exact row wins
        assert_eq!(store.price_of("eth_call", 1).unwrap(), 20); // other chains keep the default
    }

    #[test]
    fn rejects_a_non_positive_price() {
        let store = Store::open_in_memory().unwrap();
        let bad = store.conn.execute(
            "INSERT INTO method_pricing (method, chain_id, credits_per_request) \
             VALUES ('eth_call', 5, 0)",
            [],
        );
        assert!(
            bad.is_err(),
            "the CHECK constraint must reject a zero price"
        );
    }

    #[test]
    fn find_key_by_hash_locates_the_row() {
        let store = Store::open_in_memory().unwrap();
        let key = StoredKey {
            id: "k_findme".into(),
            key_hash: vec![7u8; 32],
            label: "findable".into(),
            account_address: None,
            credit_exempt: false,
            allow_trace: false,
            rate_unlimited: true,
            per_second_limit: None,
            per_day_limit: None,
            created_at: unix_now(),
            disabled_at: None,
            expires_at: None,
        };
        store.insert_key(&key).unwrap();

        let found = store
            .find_key_by_hash(&[7u8; 32])
            .unwrap()
            .expect("the row must be found by its hash");
        assert_eq!(found.id, "k_findme");
    }

    #[test]
    fn find_key_by_hash_returns_none_for_an_unknown_hash() {
        let store = Store::open_in_memory().unwrap();
        assert!(store.find_key_by_hash(&[9u8; 32]).unwrap().is_none());
    }

    #[test]
    fn read_audit_is_newest_first_and_honours_the_limit() {
        let store = Store::open_in_memory().unwrap();
        // Append in a known order. `id` is the tie-breaker the read sorts on.
        store
            .append_audit("operator", "a.one", Some("t1"), None)
            .unwrap();
        store
            .append_audit("operator", "a.two", Some("t2"), None)
            .unwrap();
        store
            .append_audit("operator", "a.three", Some("t3"), None)
            .unwrap();

        // Newest first: the last write leads, and the ids strictly descend.
        let all = store.read_audit(100).unwrap();
        assert_eq!(all.len(), 3);
        assert_eq!(all[0].action, "a.three");
        assert_eq!(all[1].action, "a.two");
        assert_eq!(all[2].action, "a.one");
        assert!(
            all[0].id > all[1].id && all[1].id > all[2].id,
            "ids must descend newest-first"
        );

        // The limit truncates to the newest rows.
        let two = store.read_audit(2).unwrap();
        assert_eq!(two.len(), 2);
        assert_eq!(two[0].action, "a.three");
        assert_eq!(two[1].action, "a.two");

        // A limit above the row count returns every row, not an error.
        assert_eq!(store.read_audit(1000).unwrap().len(), 3);
    }

    // --- accounts: reserve and settle --------------------------------------

    fn temp_account_db_path(tag: &str) -> std::path::PathBuf {
        let mut path = std::env::temp_dir();
        path.push(format!(
            "jumpgate-account-{}-{}-{}.db",
            tag,
            std::process::id(),
            unix_now()
        ));
        path
    }

    fn remove_db(path: &std::path::Path) {
        let _ = std::fs::remove_file(path);
        let _ = std::fs::remove_file(format!("{}-wal", path.display()));
        let _ = std::fs::remove_file(format!("{}-shm", path.display()));
    }

    #[test]
    fn reserve_grants_in_full_when_enough_is_available() {
        let store = Store::open_in_memory().unwrap();
        store.upsert_account("0xabc", 100, 0, 0).unwrap();

        let granted = store.reserve("0xabc", 30).unwrap();
        assert_eq!(granted, Some(30));
        assert_eq!(store.get_account("0xabc").unwrap(), Some((70, 30, 0)));
    }

    #[test]
    fn reserve_grants_a_partial_amount_when_short() {
        let store = Store::open_in_memory().unwrap();
        store.upsert_account("0xshort", 10, 0, 0).unwrap();

        // Requests 30, only 10 is there: a partial grant, not a failure.
        let granted = store.reserve("0xshort", 30).unwrap();
        assert_eq!(granted, Some(10));
        assert_eq!(store.get_account("0xshort").unwrap(), Some((0, 10, 0)));
    }

    #[test]
    fn reserve_grants_zero_when_the_account_is_empty() {
        let store = Store::open_in_memory().unwrap();
        store.upsert_account("0xempty", 0, 0, 0).unwrap();

        // A 0 grant is a normal answer ("out of credits"), not an error.
        let granted = store.reserve("0xempty", 30).unwrap();
        assert_eq!(granted, Some(0));
        assert_eq!(store.get_account("0xempty").unwrap(), Some((0, 0, 0)));
    }

    #[test]
    fn reserve_returns_none_for_an_unknown_account() {
        let store = Store::open_in_memory().unwrap();
        assert_eq!(store.reserve("0xnosuch", 10).unwrap(), None);
    }

    #[test]
    fn reserve_respects_a_nonzero_escrow_ceiling() {
        let store = Store::open_in_memory().unwrap();
        // Plenty remaining, but 40 already reserved against a ceiling of 50:
        // only 10 more headroom exists no matter how much is requested.
        store.upsert_account("0xceil", 1000, 40, 50).unwrap();

        let granted = store.reserve("0xceil", 30).unwrap();
        assert_eq!(granted, Some(10));
        assert_eq!(store.get_account("0xceil").unwrap(), Some((990, 50, 50)));
    }

    #[test]
    fn reserve_zero_ceiling_means_unbounded() {
        let store = Store::open_in_memory().unwrap();
        // The schema default for escrow_ceiling is 0. That must mean "no
        // ceiling", not "reserve nothing", the same way method_pricing's
        // chain_id = 0 means "any chain".
        store.upsert_account("0xnoceil", 500, 0, 0).unwrap();
        let granted = store.reserve("0xnoceil", 500).unwrap();
        assert_eq!(granted, Some(500));
    }

    #[test]
    fn settle_returns_the_unspent_remainder() {
        let store = Store::open_in_memory().unwrap();
        store.upsert_account("0xsettle", 100, 0, 0).unwrap();
        store.reserve("0xsettle", 40).unwrap();

        // Spent 15 of the 40 reserved; 25 comes back to credits_remaining.
        let result = store.settle("0xsettle", 15, 40).unwrap();
        assert_eq!(result, Some((85, 0)));
        assert_eq!(store.get_account("0xsettle").unwrap(), Some((85, 0, 0)));
    }

    #[test]
    fn settle_clamps_credits_reserved_at_zero() {
        let store = Store::open_in_memory().unwrap();
        // Only 10 is actually reserved, but the caller reports settling 40 —
        // a double settle or an upstream bug. credits_reserved must not go
        // negative; the unspent credit still returns to credits_remaining.
        store.upsert_account("0xover", 50, 10, 0).unwrap();

        let result = store.settle("0xover", 0, 40).unwrap();
        assert_eq!(result, Some((90, 0)), "reserved clamps at 0, not -30");
    }

    #[test]
    fn settle_returns_none_for_an_unknown_account() {
        let store = Store::open_in_memory().unwrap();
        assert_eq!(store.settle("0xnosuch", 0, 10).unwrap(), None);
    }

    #[test]
    fn reserve_then_settle_round_trip_is_invariant_when_nothing_is_spent() {
        let store = Store::open_in_memory().unwrap();
        store.upsert_account("0xroundtrip", 200, 0, 0).unwrap();
        let before_total = 200;

        let granted = store.reserve("0xroundtrip", 75).unwrap().unwrap();
        let (remaining_mid, reserved_mid, _) = store.get_account("0xroundtrip").unwrap().unwrap();
        assert_eq!(remaining_mid + reserved_mid, before_total);

        // Settling with spent = 0 must return the whole reservation, leaving
        // the account exactly where it started.
        store.settle("0xroundtrip", 0, granted).unwrap();
        let (remaining_after, reserved_after, _) =
            store.get_account("0xroundtrip").unwrap().unwrap();
        assert_eq!(remaining_after + reserved_after, before_total);
        assert_eq!((remaining_after, reserved_after), (200, 0));
    }

    #[test]
    fn reserve_then_settle_conserves_total_minus_what_was_spent() {
        let store = Store::open_in_memory().unwrap();
        store.upsert_account("0xspend", 200, 0, 0).unwrap();

        let granted = store.reserve("0xspend", 75).unwrap().unwrap();
        store.settle("0xspend", 20, granted).unwrap();

        // The total (remaining + reserved) must fall by exactly what was
        // spent — no more (a leak) and no less (a double credit).
        let (remaining, reserved, _) = store.get_account("0xspend").unwrap().unwrap();
        assert_eq!(remaining + reserved, 200 - 20);
        assert_eq!((remaining, reserved), (180, 0));
    }

    #[test]
    fn concurrent_reserves_never_grant_more_than_the_account_had() {
        // This is the load-bearing correctness test: over-granting is the bug
        // that lets a customer spend money that does not exist. Each thread
        // opens its OWN connection to the same on-disk file, so this
        // exercises SQLite's real locking, not an in-process mutex that would
        // serialize the calls trivially and prove nothing.
        let path = temp_account_db_path("concurrent");
        {
            let seed = Store::open(&path).unwrap();
            seed.upsert_account("0xconcurrent", 55, 0, 0).unwrap();
        }

        let threads: Vec<_> = (0..20)
            .map(|_| {
                let path = path.clone();
                std::thread::spawn(move || {
                    let store = Store::open(&path).unwrap();
                    store.reserve("0xconcurrent", 10).unwrap().unwrap()
                })
            })
            .collect();

        let total_granted: i64 = threads.into_iter().map(|h| h.join().unwrap()).sum();

        assert_eq!(
            total_granted, 55,
            "20 requests of 10 against a balance of 55 must grant exactly 55 in total"
        );

        let verify = Store::open(&path).unwrap();
        let (remaining, reserved, _) = verify.get_account("0xconcurrent").unwrap().unwrap();
        assert_eq!(remaining, 0, "every available credit was reserved");
        assert_eq!(reserved, 55, "the reservation total matches what was granted");
        remove_db(&path);
    }
}
