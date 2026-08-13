//! Method prices in credits, plus the in-memory price book and the getLogs math.
//!
//! Credits are relative weights. The weight follows the cost a method puts on
//! the upstream. A cached read sits near the floor. A standard read sits at the
//! base unit of 20. A wide scan or a trace costs far more. The deposit rail sets
//! the credit-to-USD rate, so the operator prices in credits here.
//!
//! The table below seeds the store on first boot, so the box meters correctly
//! with no configuration. The operator can edit any row later.
//!
//! Two prices carry runtime logic that a static row cannot hold:
//!   - `eth_getLogs` bills the base below plus a block-range surcharge, and the
//!     proxy rejects a range wider than the cap.
//!   - `eth_subscribe` bills per outbound notification, not once per call.

/// The global default when no row matches. Kept equal to the monorepo peg.
pub const DEFAULT_CU: i64 = 20;

/// (method, credits). All seed rows use chain_id 0, meaning any chain.
/// `"*"` is the editable default row.
pub const DEFAULT_PRICES: &[(&str, i64)] = &[
    // default row
    ("*", DEFAULT_CU),
    // trivial: cached or static
    ("eth_chainId", 1),
    ("net_version", 1),
    ("web3_clientVersion", 1),
    ("eth_syncing", 1),
    ("eth_unsubscribe", 1),
    // light reads
    ("eth_blockNumber", 5),
    ("eth_gasPrice", 5),
    ("eth_getBalance", 5),
    ("eth_getTransactionCount", 5),
    ("eth_getCode", 5),
    ("eth_feeHistory", 5),
    ("eth_maxPriorityFeePerGas", 5),
    // standard: the base unit
    ("eth_call", 20),
    ("eth_estimateGas", 20),
    ("eth_getBlockByNumber", 20),
    ("eth_getBlockByHash", 20),
    ("eth_getTransactionByHash", 20),
    ("eth_getTransactionReceipt", 20),
    ("eth_getStorageAt", 20),
    ("eth_getBlockReceipts", 20),
    ("eth_getBlockTransactionCountByNumber", 20),
    // write
    ("eth_sendRawTransaction", 20),
    // filters and subscriptions
    ("eth_newFilter", 20),
    ("eth_newBlockFilter", 20),
    ("eth_getFilterChanges", 20),
    ("eth_getFilterLogs", 20),
    ("eth_subscribe", 5), // per notification, applied at runtime
    // heavy scan (plus a runtime range surcharge)
    ("eth_getLogs", 75),
    // trace and debug: heavy, and blocked unless the key sets allow_trace
    ("debug_traceTransaction", 500),
    ("debug_traceCall", 500),
    ("debug_traceBlockByNumber", 500),
    ("debug_traceBlockByHash", 500),
    ("trace_transaction", 500),
    ("trace_block", 500),
    ("trace_call", 500),
    ("trace_filter", 500),
    ("trace_replayTransaction", 500),
    ("trace_replayBlockTransactions", 500),
];

// --- eth_getLogs range surcharge (finding S7) ---------------------------------

/// One credit is added for each complete 1,000-block window of range. A wide log
/// scan is the classic way to turn one cheap request into heavy upstream work,
/// so the range costs on top of the base price.
pub const GETLOGS_SURCHARGE_PER_1000_BLOCKS: i64 = 1;

/// The hard cap on a getLogs block range. The proxy rejects a wider scan before
/// it reaches the upstream. The range is `to_block - from_block`.
pub const GETLOGS_MAX_RANGE_BLOCKS: u64 = 100_000;

/// Price one `eth_getLogs` call: the base price plus a range surcharge.
///
/// This is a pure helper. The proxy parses the getLogs params and passes the
/// base (the table's `eth_getLogs` price) and the block range here. The math and
/// the cap live in one place.
///
/// The rule: one credit per complete 1,000-block window of range. The range is
/// `to_block - from_block`, so a scan up to 1,000 blocks wide keeps the base
/// price, and each further 1,000 blocks adds one credit. The function rejects an
/// inverted range and a range wider than [`GETLOGS_MAX_RANGE_BLOCKS`].
pub fn getlogs_credits(base: i64, from_block: u64, to_block: u64) -> crate::Result<i64> {
    if to_block < from_block {
        return Err(crate::Error::RangeInverted {
            from: from_block,
            to: to_block,
        });
    }
    let span = to_block - from_block;
    if span > GETLOGS_MAX_RANGE_BLOCKS {
        return Err(crate::Error::RangeTooWide {
            span,
            cap: GETLOGS_MAX_RANGE_BLOCKS,
        });
    }
    let windows = (span / 1000) as i64;
    Ok(base + windows * GETLOGS_SURCHARGE_PER_1000_BLOCKS)
}

// --- the in-memory price book (the hot-path cache) ----------------------------

use std::collections::HashMap;
use std::sync::RwLock;

use crate::error::{Error, Result};
use crate::store::Store;

/// The actor recorded on audit rows from the price book. The admin API (a later
/// step) will carry a real operator identity; for now every price change is the
/// operator at the box.
const ACTOR: &str = "operator";

/// The in-memory price book. It owns the store and resolves a price without
/// touching the database, so the hot path stays off disk.
///
/// The book mirrors the store's lookup precedence in memory:
/// exact (method, chain), then (method, 0), then ("*", chain), then ("*", 0),
/// then [`DEFAULT_CU`]. It also normalizes a method name before the lookup, so a
/// case or whitespace variant cannot bypass a priced method (finding S8).
pub struct PriceBook {
    store: Store,
    /// (method, chain_id) -> credits. Read on the hot path, updated on every set.
    prices: RwLock<HashMap<(String, i64), i64>>,
    /// lowercase method -> canonical method. Closes the casing bypass (S8).
    canonical: RwLock<HashMap<String, String>>,
}

impl PriceBook {
    /// Build the book and load every price row into memory at boot. It also
    /// builds a lowercase-to-canonical index from the known method rows.
    pub fn new(store: Store) -> Result<PriceBook> {
        let mut prices = HashMap::new();
        let mut canonical = HashMap::new();
        for (method, chain_id, credits) in store.all_prices()? {
            // The "*" default row is not a real method name; keep it out of the
            // canonical index so it never resolves a normalized lookup.
            if method != "*" {
                canonical
                    .entry(method.to_lowercase())
                    .or_insert_with(|| method.clone());
            }
            prices.insert((method, chain_id), credits);
        }
        Ok(PriceBook {
            store,
            prices: RwLock::new(prices),
            canonical: RwLock::new(canonical),
        })
    }

    /// The price of one method on one chain, in credits. This mirrors
    /// [`Store::price_of`] exactly, but reads memory. It does not normalize; an
    /// unknown method falls through to the "*" default row.
    pub fn price_of(&self, method: &str, chain_id: i64) -> i64 {
        let prices = self.prices.read().expect("price map lock");
        let lookups = [(method, chain_id), (method, 0), ("*", chain_id), ("*", 0)];
        for (m, c) in lookups {
            if let Some(&credits) = prices.get(&(m.to_owned(), c)) {
                return credits;
            }
        }
        DEFAULT_CU
    }

    /// Resolve a raw method name to its canonical priced name. It trims
    /// surrounding whitespace and matches case-insensitively, so `"ETH_getLogs"`,
    /// `" eth_getlogs "`, and `"eth_getLogs"` all return `Some("eth_getLogs")`.
    /// It returns `None` when no known method matches (finding S8).
    pub fn normalize(&self, method: &str) -> Option<String> {
        let key = method.trim().to_lowercase();
        self.canonical
            .read()
            .expect("canonical map lock")
            .get(&key)
            .cloned()
    }

    /// Resolve a raw method name and its price in one call. It returns the
    /// canonical name (or `None` for an unknown method) and the credits.
    ///
    /// A `None` name means the method is not in the table. The caller decides
    /// what to do: reject the request, or charge the default-high "*" price. The
    /// credits returned here are the "*" default for an unknown method, so a
    /// caller that charges rather than rejects still gets a safe number.
    pub fn price_of_normalized(&self, method: &str, chain_id: i64) -> (Option<String>, i64) {
        match self.normalize(method) {
            Some(canonical) => {
                let credits = self.price_of(&canonical, chain_id);
                (Some(canonical), credits)
            }
            None => (None, self.price_of(method, chain_id)),
        }
    }

    /// Set the price of one method on one chain. It rejects a non-positive price
    /// with [`Error::InvalidPrice`] before any write, so the SQL CHECK never has
    /// to fire. On success it upserts the row, writes an audit row, and updates
    /// the in-memory maps.
    pub fn set_price(&self, method: &str, chain_id: i64, credits: i64) -> Result<()> {
        if credits <= 0 {
            return Err(Error::InvalidPrice(credits));
        }
        // Database first, then memory. A failed write must not leave the map
        // ahead of the store.
        self.store.upsert_price(method, chain_id, credits)?;
        self.store
            .append_audit(ACTOR, "price.set", Some(method), Some(&credits.to_string()))?;

        self.prices
            .write()
            .expect("price map lock")
            .insert((method.to_string(), chain_id), credits);
        if method != "*" {
            self.canonical
                .write()
                .expect("canonical map lock")
                .entry(method.to_lowercase())
                .or_insert_with(|| method.to_string());
        }
        Ok(())
    }

    /// Every price row as (method, chain_id, credits), sorted. The CLI reads this
    /// for `price list`. It reads memory, not the database.
    pub fn all_prices(&self) -> Vec<(String, i64, i64)> {
        let prices = self.prices.read().expect("price map lock");
        let mut out: Vec<(String, i64, i64)> = prices
            .iter()
            .map(|((method, chain), credits)| (method.clone(), *chain, *credits))
            .collect();
        out.sort();
        out
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::store::Store;

    /// A temp-file database path, so a second `PriceBook` can reopen the store.
    fn temp_db_path(tag: &str) -> std::path::PathBuf {
        let mut path = std::env::temp_dir();
        path.push(format!(
            "jumpgate-price-{}-{}-{}.db",
            tag,
            std::process::id(),
            crate::store::unix_now()
        ));
        path
    }

    fn remove_db(path: &std::path::Path) {
        let _ = std::fs::remove_file(path);
        let _ = std::fs::remove_file(format!("{}-wal", path.display()));
        let _ = std::fs::remove_file(format!("{}-shm", path.display()));
    }

    #[test]
    fn cache_matches_store_for_known_and_unknown_methods() {
        let store = Store::open_in_memory().unwrap();
        // Capture the store's answers before the book takes ownership.
        let call = store.price_of("eth_call", 1).unwrap();
        let logs = store.price_of("eth_getLogs", 1).unwrap();
        let unknown = store.price_of("foo_bar", 1).unwrap();

        let book = PriceBook::new(store).unwrap();
        assert_eq!(book.price_of("eth_call", 1), call);
        assert_eq!(book.price_of("eth_getLogs", 1), logs);
        assert_eq!(book.price_of("foo_bar", 1), unknown);
        assert_eq!(unknown, DEFAULT_CU);
    }

    #[test]
    fn cache_honours_an_exact_chain_override() {
        let store = Store::open_in_memory().unwrap();
        store.upsert_price("eth_call", 369, 40).unwrap();
        let base_chain = store.price_of("eth_call", 1).unwrap();

        let book = PriceBook::new(store).unwrap();
        assert_eq!(book.price_of("eth_call", 369), 40); // exact row wins
        assert_eq!(book.price_of("eth_call", 1), base_chain); // other chains keep 20
    }

    #[test]
    fn normalize_closes_the_casing_bypass() {
        let book = PriceBook::new(Store::open_in_memory().unwrap()).unwrap();
        for m in ["ETH_getLogs", "eth_getlogs", "  eth_getLogs  "] {
            assert_eq!(book.normalize(m).as_deref(), Some("eth_getLogs"));
            let (canonical, credits) = book.price_of_normalized(m, 1);
            assert_eq!(canonical.as_deref(), Some("eth_getLogs"));
            assert_eq!(credits, 75);
        }
        // A truly unknown method normalizes to None and prices at the default.
        assert_eq!(book.normalize("totally_unknown"), None);
        let (canonical, credits) = book.price_of_normalized("totally_unknown", 1);
        assert_eq!(canonical, None);
        assert_eq!(credits, DEFAULT_CU);
    }

    #[test]
    fn set_price_updates_db_and_cache_and_audit() {
        let path = temp_db_path("set");

        {
            let store = Store::open(&path).unwrap();
            let book = PriceBook::new(store).unwrap();
            book.set_price("eth_getLogs", 0, 90).unwrap();
            // The in-memory cache updates in the same call.
            assert_eq!(book.price_of("eth_getLogs", 1), 90);
        }

        // A fresh book reloads from the database and sees the new price.
        let store = Store::open(&path).unwrap();
        let book = PriceBook::new(store).unwrap();
        assert_eq!(book.price_of("eth_getLogs", 1), 90);

        // The audit trail holds the price.set row, with the new credits as detail.
        let conn = rusqlite::Connection::open(&path).unwrap();
        let detail: String = conn
            .query_row(
                "SELECT detail FROM audit_log \
                 WHERE action = 'price.set' AND target = 'eth_getLogs' \
                 ORDER BY id DESC LIMIT 1",
                [],
                |row| row.get(0),
            )
            .unwrap();
        assert_eq!(detail, "90");
        drop(conn);

        remove_db(&path);
    }

    #[test]
    fn set_price_rejects_non_positive_and_keeps_the_old_price() {
        let store = Store::open_in_memory().unwrap();
        let book = PriceBook::new(store).unwrap();
        let before = book.price_of("eth_getLogs", 1);

        assert!(matches!(
            book.set_price("eth_getLogs", 0, 0),
            Err(Error::InvalidPrice(0))
        ));
        assert!(matches!(
            book.set_price("eth_getLogs", 0, -5),
            Err(Error::InvalidPrice(-5))
        ));
        // The rejected sets must leave the price untouched.
        assert_eq!(book.price_of("eth_getLogs", 1), before);
    }

    #[test]
    fn getlogs_credits_math_cap_and_inverted() {
        let base = 75;
        // One credit per complete 1,000-block window of range (to - from).
        assert_eq!(getlogs_credits(base, 0, 0).unwrap(), base);
        assert_eq!(getlogs_credits(base, 0, 999).unwrap(), base);
        assert_eq!(getlogs_credits(base, 0, 1000).unwrap(), base + 1);
        assert_eq!(getlogs_credits(base, 0, 1999).unwrap(), base + 1);
        assert_eq!(getlogs_credits(base, 0, 2000).unwrap(), base + 2);
        // Note: the task's example lists 0..2500 -> base+3. That value cannot
        // coexist with 0..999 -> base under any per-1,000 step rule, so it is an
        // arithmetic slip. This rule (complete windows) matches the deliberate
        // 999/1000 boundary and yields base+2 here.
        assert_eq!(getlogs_credits(base, 0, 2500).unwrap(), base + 2);

        // An inverted range is rejected.
        assert!(matches!(
            getlogs_credits(base, 10, 5),
            Err(Error::RangeInverted { from: 10, to: 5 })
        ));
        // A range wider than the cap is rejected; exactly the cap is allowed.
        assert!(getlogs_credits(base, 0, GETLOGS_MAX_RANGE_BLOCKS).is_ok());
        assert!(matches!(
            getlogs_credits(base, 0, GETLOGS_MAX_RANGE_BLOCKS + 1),
            Err(Error::RangeTooWide { .. })
        ));
    }

    #[test]
    fn getlogs_cap_boundary_reports_span_and_cap() {
        let base = 75;
        // Exactly the cap succeeds and carries the full-window surcharge.
        let at_cap = getlogs_credits(base, 0, GETLOGS_MAX_RANGE_BLOCKS).unwrap();
        assert_eq!(at_cap, base + (GETLOGS_MAX_RANGE_BLOCKS / 1000) as i64);

        // One block past the cap reports the exact span and cap it rejected.
        match getlogs_credits(base, 0, GETLOGS_MAX_RANGE_BLOCKS + 1) {
            Err(Error::RangeTooWide { span, cap }) => {
                assert_eq!(span, GETLOGS_MAX_RANGE_BLOCKS + 1);
                assert_eq!(cap, GETLOGS_MAX_RANGE_BLOCKS);
            }
            other => panic!("expected RangeTooWide, got {other:?}"),
        }

        // An inverted range reports the from and to it rejected.
        match getlogs_credits(base, 100, 50) {
            Err(Error::RangeInverted { from, to }) => {
                assert_eq!(from, 100);
                assert_eq!(to, 50);
            }
            other => panic!("expected RangeInverted, got {other:?}"),
        }
    }
}
