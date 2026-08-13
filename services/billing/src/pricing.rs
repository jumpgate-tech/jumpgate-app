//! Default method prices in credits.
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
