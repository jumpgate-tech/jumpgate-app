//! The SQLite store: open, migrate, seed default pricing, and the price lookup.
//!
//! This is step 1 of the billing build (docs/design/billing-admin.md section 6).
//! It owns the schema and the read side of pricing. The key manager, the admin
//! API, and the hot path build on top of it.

use std::path::Path;
use std::time::Duration;

use rusqlite::{params, Connection, OptionalExtension};

use crate::error::Result;
use crate::pricing;

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
        let lookups = [
            (method, chain_id),
            (method, 0),
            ("*", chain_id),
            ("*", 0),
        ];
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
        assert_eq!(store.price_count().unwrap(), pricing::DEFAULT_PRICES.len() as i64);
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
        assert!(bad.is_err(), "the CHECK constraint must reject a zero price");
    }
}
