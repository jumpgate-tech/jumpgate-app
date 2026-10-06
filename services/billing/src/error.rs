use thiserror::Error;

/// One error type for the whole crate.
#[derive(Debug, Error)]
pub enum Error {
    #[error("sqlite: {0}")]
    Sqlite(#[from] rusqlite::Error),
    #[error("io: {0}")]
    Io(#[from] std::io::Error),
    #[error("key not found: {0}")]
    KeyNotFound(String),
    #[error("the key pepper is missing or empty")]
    MissingPepper,
    #[error("invalid price: credits must be positive, got {0}")]
    InvalidPrice(i64),
    #[error("block range inverted: to {to} is below from {from}")]
    RangeInverted { from: u64, to: u64 },
    #[error("block range too wide: {span} blocks exceeds the cap of {cap}")]
    RangeTooWide { span: u64, cap: u64 },
    #[error("admin address {0} is not a loopback address; the admin API binds loopback only")]
    AddrNotLoopback(std::net::SocketAddr),
    #[error("the admin bearer token is missing or empty; set JUMPGATE_ADMIN_TOKEN")]
    MissingAdminToken,
    #[error("the relay bearer token is missing or empty; set JUMPGATE_RELAY_TOKEN")]
    MissingRelayToken,
    #[error("account not found: {0}")]
    AccountNotFound(String),
    #[error("invalid credits: must be positive, got {0}")]
    InvalidCredits(i64),
    #[error("invalid settle: spent {spent} must be between 0 and reserved {reserved}")]
    InvalidSettle { spent: i64, reserved: i64 },
    #[error("settle of {reserved} exceeds the {held} credits this account holds in reserve")]
    SettleExceedsReservation { reserved: i64, held: i64 },
    #[error("settle id {0} was already used for a different settle")]
    SettleIdReused(String),
    #[error("refusing unix socket path {path}: {reason}")]
    UnsafeSocketPath {
        path: std::path::PathBuf,
        reason: String,
    },
}

pub type Result<T> = std::result::Result<T, Error>;
