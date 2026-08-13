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
}

pub type Result<T> = std::result::Result<T, Error>;
