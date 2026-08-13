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
}

pub type Result<T> = std::result::Result<T, Error>;
