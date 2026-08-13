//! Jumpgate metered API-key and billing service.
//!
//! Step 1 ships the store: the SQLite schema, seeded default pricing, and the
//! price lookup. Later steps add the key manager, the admin API, and the
//! metering hot path. See docs/design/billing-admin.md.

pub mod admin;
pub mod error;
pub mod keys;
pub mod pricing;
pub mod store;

pub use error::{Error, Result};
