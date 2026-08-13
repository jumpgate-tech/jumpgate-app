-- Jumpgate billing store. One embedded SQLite file, WAL mode.
-- All timestamps are unix seconds. The store holds no raw API key — only the
-- HMAC hash of one. See docs/design/metered-api-keys.md section 6.

-- API keys. There is no cap on the row count; a key is one cheap row.
CREATE TABLE IF NOT EXISTS project_key (
  id               TEXT    PRIMARY KEY,          -- public id, never the secret
  key_hash         BLOB    NOT NULL UNIQUE,      -- HMAC-SHA256(pepper, raw key)
  label            TEXT    NOT NULL DEFAULT '',
  account_address  TEXT,                          -- funding account, null until bound
  credit_exempt    INTEGER NOT NULL DEFAULT 0,    -- 1 = free key (keeps a hard cap)
  allow_trace      INTEGER NOT NULL DEFAULT 0,    -- 1 = trace/debug namespaces allowed
  rate_unlimited   INTEGER NOT NULL DEFAULT 1,    -- 1 = no throttle; balance is the limit
  per_second_limit INTEGER,                        -- null when rate_unlimited = 1
  per_day_limit    INTEGER,
  created_at       INTEGER NOT NULL,
  disabled_at      INTEGER,                        -- null = active; set = revoked
  expires_at       INTEGER
);

-- Per-key rules: origins, method allow/block, networks, IP ranges.
CREATE TABLE IF NOT EXISTS key_constraint (
  id      INTEGER PRIMARY KEY,
  key_id  TEXT    NOT NULL REFERENCES project_key(id) ON DELETE CASCADE,
  kind    TEXT    NOT NULL,   -- origin | method_allow | method_block | network | ip_allow | ip_deny
  value   TEXT    NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_key_constraint_key ON key_constraint(key_id);

-- Method prices in credits. The row method = '*' is the default. chain_id = 0
-- means any chain. Lookup precedence lives in code: (method, chain) ->
-- (method, 0) -> ('*', chain) -> ('*', 0). A price must be positive.
CREATE TABLE IF NOT EXISTS method_pricing (
  method              TEXT    NOT NULL,
  chain_id            INTEGER NOT NULL DEFAULT 0,
  credits_per_request INTEGER NOT NULL CHECK (credits_per_request > 0),
  PRIMARY KEY (method, chain_id)
);

-- Prepaid balances. The escrow ceiling bounds a customer's exposure.
CREATE TABLE IF NOT EXISTS account (
  address           TEXT    PRIMARY KEY,
  credits_remaining INTEGER NOT NULL DEFAULT 0,
  credits_reserved  INTEGER NOT NULL DEFAULT 0,
  escrow_ceiling    INTEGER NOT NULL DEFAULT 0,
  updated_at        INTEGER NOT NULL DEFAULT 0
);

-- Idempotent deposit ledger. A unique tx_hash makes a repeated poll safe.
-- token_amount is a decimal string, because a token amount can exceed i64.
CREATE TABLE IF NOT EXISTS credit_deposit (
  tx_hash               TEXT    PRIMARY KEY,
  address               TEXT    NOT NULL,
  token_amount          TEXT    NOT NULL,
  credits_issued        INTEGER NOT NULL,
  rate_token_per_credit TEXT,
  rate_source           TEXT,
  created_at            INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_credit_deposit_addr ON credit_deposit(address);

-- Append-only audit trail. Every admin mutation writes one row.
CREATE TABLE IF NOT EXISTS audit_log (
  id      INTEGER PRIMARY KEY AUTOINCREMENT,
  ts      INTEGER NOT NULL,
  actor   TEXT    NOT NULL,
  action  TEXT    NOT NULL,
  target  TEXT,
  detail  TEXT
);
