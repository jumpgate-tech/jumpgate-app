//! The operator admin HTTP API: keys, pricing, and the audit log. Also the
//! relay's one route: `/internal/authenticate`.
//!
//! This is step 4 of the billing build (docs/design/billing-admin.md section 5),
//! extended per the keyed-access design
//! (docs/superpowers/specs/2026-08-15-relay-keyed-access-design.md section 7).
//! The store owns the SQL, the key manager owns the crypto, and the price book
//! owns the pricing map. This module is the thin HTTP shell over the three.
//!
//! Security shape (findings S9, S10, S11, and the keyed-access design):
//!   - [`serve`] binds loopback TCP and refuses a non-loopback address. This
//!     is the fallback transport for a platform with no reliable unix socket
//!     (Windows).
//!   - [`serve_unix`] binds a unix socket instead. This is the transport a
//!     same-box relay uses. A TCP port on loopback is squattable by any local
//!     process before this service starts; a unix socket inside a
//!     service-owned, `0700` directory is not. See `serve_unix` for the full
//!     posture, including the peer credential check.
//!   - Two bearer tokens, not one. `JUMPGATE_ADMIN_TOKEN` gates every
//!     `/admin/*` route: create, rotate, revoke, pricing, audit. A second,
//!     separate `JUMPGATE_RELAY_TOKEN` gates only `/internal/authenticate`.
//!     The admin token does NOT work on `/internal/*`, and the relay token
//!     does NOT work on `/admin/*` — each route group's middleware checks
//!     only its own token. A relay credential leak then buys an attacker the
//!     ability to test whether a key is valid, not the ability to mint or
//!     revoke one. Both compares run in constant time. A missing or empty
//!     token, for either credential, is a startup fault: the server refuses
//!     to start, so neither gate can open by default.
//!   - `GET /healthz` is the one open route. It touches no secret.
//!   - The raw key, the key hash, and both tokens never appear in a response
//!     body or an error.
//!
//! TODO: mutual TLS is the design's target for the admin plane (billing-admin.md
//! section 2, finding S9). This step ships loopback plus a bearer token, the
//! sanctioned scope trim. mTLS is the next hardening step: terminate a client
//! certificate here and pin the operator's certificate.
//!
//! Note on the actor: every mutation writes an audit row under the constant
//! "operator" actor (see `keys.rs` and `pricing.rs`). A later step threads a real
//! operator identity from the authenticated request into the audit row.
//!
//! Note on concurrency: `rusqlite::Connection` is `Send` but not `Sync`, so
//! `KeyManager` and `PriceBook` are `Send + !Sync`. Axum state must be
//! `Clone + Send + Sync + 'static`, so each sits behind a `std::sync::Mutex`. A
//! handler locks, does the synchronous database work, drops the guard, then
//! builds the response — it never holds a guard across an `.await`. The admin
//! plane is low volume, so one mutex per manager is fine. The future RPC hot path
//! (step 6) is high volume and will need a lock-free authentication read path
//! instead; do not copy this mutex arrangement onto that path.

use std::net::SocketAddr;
use std::path::Path as FsPath;
use std::sync::{Arc, Mutex};

use axum::extract::{Path, Query, Request, State};
use axum::http::{header, StatusCode};
use axum::middleware::{self, Next};
use axum::response::{IntoResponse, Response};
use axum::routing::{get, post, put};
use axum::{Json, Router};
use serde::{Deserialize, Serialize};
use serde_json::json;
use subtle::ConstantTimeEq;

use crate::error::Error;
use crate::keys::{AuthOutcome, KeyConfig, KeyManager, KeyPatch, KeyRecord, Rate};
use crate::pricing::PriceBook;
use crate::store::{AuditRow, Store};
use crate::Result;

/// The default number of audit rows a read returns when the caller sets no limit.
const DEFAULT_AUDIT_LIMIT: i64 = 100;
/// The hard ceiling on an audit read, so one request cannot pull the whole trail.
const MAX_AUDIT_LIMIT: i64 = 1000;

/// The shared application state. It is `Clone`; the inner state is shared behind
/// `Arc`. Each manager sits behind a `std::sync::Mutex` because it is not `Sync`.
#[derive(Clone)]
pub struct AppState {
    keys: Arc<Mutex<KeyManager>>,
    prices: Arc<Mutex<PriceBook>>,
    /// The credit ledger's connection. Unlike `keys` and `prices`, this holds
    /// the raw [`Store`] with no in-memory cache layer in front of it. The
    /// design (keyed-access design doc, section 8) is explicit that a credit
    /// balance must never be cached — a stale balance is money — so there is
    /// no `Accounts` wrapper here mirroring `KeyManager` or `PriceBook`, only
    /// the database itself, read and written fresh on every call.
    accounts: Arc<Mutex<Store>>,
    /// The admin bearer token, for every `/admin/*` route. It never appears
    /// in a response or a log.
    token: Arc<String>,
    /// The relay bearer token, for `/internal/*` routes only. Kept apart from
    /// `token` so the two credentials can never be confused: each route
    /// group's middleware reads only the field it owns.
    relay_token: Arc<String>,
}

impl AppState {
    /// Build the state. It refuses an empty token for either credential, so
    /// neither gate can open by default (finding S9, and the same posture for
    /// the relay credential). The caller reads `token` from
    /// `JUMPGATE_ADMIN_TOKEN` and `relay_token` from `JUMPGATE_RELAY_TOKEN`,
    /// and must fail closed when either is unset.
    pub fn new(
        keys: KeyManager,
        prices: PriceBook,
        accounts: Store,
        token: String,
        relay_token: String,
    ) -> Result<AppState> {
        if token.is_empty() {
            return Err(Error::MissingAdminToken);
        }
        if relay_token.is_empty() {
            return Err(Error::MissingRelayToken);
        }
        Ok(AppState {
            keys: Arc::new(Mutex::new(keys)),
            prices: Arc::new(Mutex::new(prices)),
            accounts: Arc::new(Mutex::new(accounts)),
            token: Arc::new(token),
            relay_token: Arc::new(relay_token),
        })
    }
}

/// Build the router: the open `GET /healthz`, the admin-token-gated
/// `/admin/*` routes, and the relay-token-gated `/internal/*` routes.
///
/// Each guarded group carries its own `route_layer`, bound to its own
/// middleware function reading its own token field. That is what makes the
/// separation a fact of the wiring: the admin middleware never even looks at
/// `relay_token`, and the relay middleware never looks at `token`, so a future
/// edit cannot accidentally let one credential open both doors.
pub fn build_router(state: AppState) -> Router {
    let admin = Router::new()
        .route("/admin/keys", post(create_key).get(list_keys))
        .route(
            "/admin/keys/{id}",
            axum::routing::patch(update_key).delete(revoke_key),
        )
        .route("/admin/keys/{id}/rotate", post(rotate_key))
        .route("/admin/keys/{id}/usage", get(key_usage))
        .route("/admin/pricing", get(list_pricing))
        .route("/admin/pricing/{method}", put(set_pricing))
        .route("/admin/audit", get(read_audit))
        .route_layer(middleware::from_fn_with_state(
            state.clone(),
            require_bearer,
        ));

    let internal = Router::new()
        .route("/internal/authenticate", post(authenticate_key))
        .route("/internal/reserve", post(reserve_credits))
        .route("/internal/settle", post(settle_credits))
        .route("/internal/price", get(price_for_relay))
        .route_layer(middleware::from_fn_with_state(
            state.clone(),
            require_relay_bearer,
        ));

    Router::new()
        .route("/healthz", get(healthz))
        .merge(admin)
        .merge(internal)
        .with_state(state)
}

/// Bind the loopback address and serve the admin API. It refuses a non-loopback
/// address before it binds (finding S9/S3 posture). The bind and serve I/O errors
/// map to [`Error::Io`].
///
/// This is the fallback transport (design doc section 7): `AF_UNIX` exists on
/// Windows but is patchy in both toolchains, so a Windows build keeps using
/// TCP. Every other target should prefer [`serve_unix`].
pub async fn serve(addr: SocketAddr, state: AppState) -> Result<()> {
    if !addr.ip().is_loopback() {
        return Err(Error::AddrNotLoopback(addr));
    }
    let listener = tokio::net::TcpListener::bind(addr).await?;
    let local = listener.local_addr()?;
    println!("jumpgate admin API listening on http://{local}");
    axum::serve(listener, build_router(state)).await?;
    Ok(())
}

/// Bind a unix socket and serve the admin API over it. This is the relay's
/// transport (design doc section 7): a unix socket in a service-owned
/// directory, not a TCP port on loopback. Binding an unused high port on
/// loopback needs no privilege on Linux or macOS, so a TCP port is squattable
/// by any local process that starts before this service does; a unix socket
/// inside a `0700` directory is not, because only the owning user can even
/// see the directory's contents.
///
/// Three things happen before the bind, all load-bearing:
///
/// 1. The parent directory is created at `0700` if missing. This is the real
///    protection — on Linux and macOS, directory permissions gate whether
///    another user's process can reach the socket at all, so both the
///    directory mode and the socket file mode are set, not just one.
/// 2. Any file already at `path` is unlinked first. A crashed process leaves
///    its socket file behind; binding to an existing path then fails with
///    `EADDRINUSE` unless the stale file is removed first. A dangling unix
///    socket file carries no data, so removing it is safe.
/// 3. The socket file itself is set to `0600` once it exists, so a
///    same-directory, different-user read (were the directory mode ever
///    loosened by mistake) still hits a permission error.
///
/// Every accepted connection also passes a peer credential check — see
/// [`PeerCheckedUnixListener`] for what it does and does not stop.
#[cfg(unix)]
pub async fn serve_unix(path: &FsPath, state: AppState) -> Result<()> {
    use std::os::unix::fs::MetadataExt;

    let listener = bind_unix(path)?;
    println!("jumpgate admin API listening on unix:{}", path.display());

    // The socket file we just created is owned by our own effective uid, so
    // its metadata is a dependency-free way to read that uid back — no need
    // for `libc::getuid()` or a new crate just for this.
    let expected_uid = std::fs::metadata(path)?.uid();

    let checked = PeerCheckedUnixListener {
        inner: listener,
        expected_uid,
    };
    axum::serve(checked, build_router(state)).await?;
    Ok(())
}

/// The synchronous half of [`serve_unix`]: prepare the directory, clear a
/// stale socket file, bind, and lock the socket file down. Split out so a
/// test can drive it without running the accept loop.
#[cfg(unix)]
fn bind_unix(path: &FsPath) -> Result<tokio::net::UnixListener> {
    use std::os::unix::fs::PermissionsExt;

    if let Some(parent) = path.parent().filter(|p| !p.as_os_str().is_empty()) {
        std::fs::create_dir_all(parent)?;
        std::fs::set_permissions(parent, std::fs::Permissions::from_mode(0o700))?;
    }

    // Ignore the error: the common case is "no such file", which is exactly
    // the state we want before a bind. A real removal failure (e.g. a
    // permission problem) still surfaces at the `bind` call below.
    let _ = std::fs::remove_file(path);

    let listener = tokio::net::UnixListener::bind(path)?;
    std::fs::set_permissions(path, std::fs::Permissions::from_mode(0o600))?;
    Ok(listener)
}

/// A unix listener that only hands `axum::serve` a connection from the
/// expected uid. `axum::serve`'s `Listener::accept` must always return a
/// connection — it has no way to reject one — so this loops past a bad peer
/// instead of returning it.
///
/// **What this stops:** a process running as a different local user from
/// reaching the socket, as a backstop in case the directory or file mode is
/// ever loosened by mistake (design doc section 7).
///
/// **What this does NOT stop:** an impostor process running under the SAME
/// uid as the billing service. `peer_cred()` reports the kernel's view of the
/// connecting process's real uid, which a same-uid impostor legitimately has.
/// Stopping that needs process isolation — a separate service account, or a
/// container boundary — which is outside what any socket-level check can do.
#[cfg(unix)]
struct PeerCheckedUnixListener {
    inner: tokio::net::UnixListener,
    expected_uid: u32,
}

#[cfg(unix)]
impl axum::serve::Listener for PeerCheckedUnixListener {
    type Io = tokio::net::UnixStream;
    type Addr = tokio::net::unix::SocketAddr;

    async fn accept(&mut self) -> (Self::Io, Self::Addr) {
        loop {
            let (stream, addr) = match self.inner.accept().await {
                Ok(pair) => pair,
                Err(e) => {
                    eprintln!("unix accept error: {e}");
                    tokio::time::sleep(std::time::Duration::from_millis(100)).await;
                    continue;
                }
            };
            match stream.peer_cred() {
                Ok(cred) if peer_uid_ok(cred.uid(), self.expected_uid) => return (stream, addr),
                Ok(cred) => eprintln!(
                    "rejected a unix peer with uid {}, expected {}",
                    cred.uid(),
                    self.expected_uid
                ),
                Err(e) => eprintln!("could not read the unix peer credential: {e}"),
            }
            // Drop `stream` here and loop for the next connection. Closing it
            // with no response is the correct refusal: the peer gets a
            // connection reset, not a hint about why.
        }
    }

    fn local_addr(&self) -> std::io::Result<Self::Addr> {
        self.inner.local_addr()
    }
}

/// The peer credential decision, pulled out as a plain comparison so it is
/// testable with ordinary numbers and needs no real socket.
#[cfg(unix)]
fn peer_uid_ok(peer_uid: u32, expected_uid: u32) -> bool {
    peer_uid == expected_uid
}

// --- auth ---------------------------------------------------------------------

/// True when the request carries `Bearer <expected>`. The compare runs in
/// constant time: `ct_eq` on slices returns false at once for a length
/// mismatch and takes the same time for any equal-length guess, so neither
/// token leaks through a timing side channel.
fn bearer_matches(req: &Request, expected: &str) -> bool {
    let presented = req
        .headers()
        .get(header::AUTHORIZATION)
        .and_then(|v| v.to_str().ok())
        .and_then(|v| v.strip_prefix("Bearer "));

    match presented {
        Some(token) => token.as_bytes().ct_eq(expected.as_bytes()).into(),
        None => false,
    }
}

/// The bearer gate on every `/admin/*` route. A missing or wrong token
/// returns 401 with a small JSON error and no detail. This checks `token`
/// only — the relay token never opens this door.
async fn require_bearer(State(state): State<AppState>, req: Request, next: Next) -> Response {
    if bearer_matches(&req, &state.token) {
        next.run(req).await
    } else {
        unauthorized()
    }
}

/// The bearer gate on `/internal/*`. This checks `relay_token` only — the
/// admin token never opens this door, which is the whole point of splitting
/// the credential (design doc section 7: least privilege).
async fn require_relay_bearer(
    State(state): State<AppState>,
    req: Request,
    next: Next,
) -> Response {
    if bearer_matches(&req, &state.relay_token) {
        next.run(req).await
    } else {
        unauthorized()
    }
}

/// The 401 response. It carries a small JSON error and leaks no detail.
fn unauthorized() -> Response {
    (
        StatusCode::UNAUTHORIZED,
        Json(json!({ "error": "unauthorized" })),
    )
        .into_response()
}

/// The 403 response for a disabled (revoked) key. It carries a small JSON
/// error and leaks no detail beyond the status the caller already knows.
fn forbidden() -> Response {
    (StatusCode::FORBIDDEN, Json(json!({ "error": "disabled" }))).into_response()
}

// --- handlers -----------------------------------------------------------------

/// Liveness. The one open route. It touches no secret and no database.
async fn healthz() -> impl IntoResponse {
    (StatusCode::OK, Json(json!({ "status": "ok" })))
}

/// Create a key. Returns the public id and the raw key once.
async fn create_key(
    State(state): State<AppState>,
    Json(req): Json<CreateKeyRequest>,
) -> std::result::Result<Response, ApiError> {
    let config = req.into_config();
    let (id, raw) = {
        let km = state.keys.lock().expect("keys lock");
        km.create(config)?
    };
    Ok((StatusCode::CREATED, Json(CreatedKey { id, key: raw })).into_response())
}

/// List every key, revoked ones included. Returns no secret and no hash.
async fn list_keys(
    State(state): State<AppState>,
) -> std::result::Result<Json<Vec<KeyView>>, ApiError> {
    let records = {
        let km = state.keys.lock().expect("keys lock");
        km.list()?
    };
    Ok(Json(records.iter().map(KeyView::from).collect()))
}

/// Update the mutable fields of a key. Returns the updated record.
async fn update_key(
    State(state): State<AppState>,
    Path(id): Path<String>,
    Json(req): Json<PatchKeyRequest>,
) -> std::result::Result<Json<KeyView>, ApiError> {
    let patch = req.into_patch();
    let record = {
        let km = state.keys.lock().expect("keys lock");
        km.update(&id, patch)?
    };
    Ok(Json(KeyView::from(&record)))
}

/// Rotate a key. Returns the new raw key once.
async fn rotate_key(
    State(state): State<AppState>,
    Path(id): Path<String>,
) -> std::result::Result<Json<CreatedKey>, ApiError> {
    let raw = {
        let km = state.keys.lock().expect("keys lock");
        km.rotate(&id)?
    };
    Ok(Json(CreatedKey { id, key: raw }))
}

/// Revoke a key. Returns 204 with no body.
async fn revoke_key(
    State(state): State<AppState>,
    Path(id): Path<String>,
) -> std::result::Result<StatusCode, ApiError> {
    {
        let km = state.keys.lock().expect("keys lock");
        km.revoke(&id)?;
    }
    Ok(StatusCode::NO_CONTENT)
}

/// The relay's one operation: resolve a raw key and read its record. This is
/// the least-privilege route (design doc section 7) — it can only look a key
/// up, never create, rotate, revoke, price, or read audit.
///
/// It never returns the key hash or the raw key, because [`KeyRecord`] never
/// carries them past the manager in the first place (see `keys.rs`); there is
/// no field to accidentally serialize.
async fn authenticate_key(
    State(state): State<AppState>,
    Json(req): Json<AuthenticateRequest>,
) -> std::result::Result<Response, ApiError> {
    let outcome = {
        let km = state.keys.lock().expect("keys lock");
        km.authenticate_status(&req.key)?
    };
    let record = match outcome {
        AuthOutcome::Active(record) => record,
        AuthOutcome::Disabled => return Ok(forbidden()),
        AuthOutcome::Unknown => return Ok(unauthorized()),
    };
    let constraints = {
        let km = state.keys.lock().expect("keys lock");
        km.constraints(&record.id)?
    };
    Ok((StatusCode::OK, Json(AuthenticateView::build(record, constraints))).into_response())
}

/// Reserve up to `credits` credits for an account (design doc section 8: the
/// relay leases credits rather than caching a balance). Grants what is
/// available when the account holds less than asked — a partial grant, never
/// a failure — so the response always carries a `granted` count, 0 included.
/// An unknown account is a 404; a non-positive `credits` is a 400, checked
/// before the store is touched.
async fn reserve_credits(
    State(state): State<AppState>,
    Json(req): Json<ReserveRequest>,
) -> std::result::Result<Json<ReserveView>, ApiError> {
    if req.credits <= 0 {
        return Err(ApiError::from(Error::InvalidCredits(req.credits)));
    }
    let granted = {
        let accounts = state.accounts.lock().expect("accounts lock");
        accounts.reserve(&req.account, req.credits)?
    };
    match granted {
        Some(granted) => Ok(Json(ReserveView { granted })),
        None => Err(ApiError::from(Error::AccountNotFound(req.account))),
    }
}

/// Settle a previous reservation: the relay reports how much of `reserved` it
/// actually spent, and this returns the rest to `credits_remaining`. Rejects
/// `spent > reserved` and a negative amount with a 400 before the store is
/// touched — the relay cannot spend more than it held. An unknown account is
/// a 404, the same as `reserve_credits`.
async fn settle_credits(
    State(state): State<AppState>,
    Json(req): Json<SettleRequest>,
) -> std::result::Result<Json<SettleView>, ApiError> {
    if req.spent < 0 || req.reserved < 0 || req.spent > req.reserved {
        return Err(ApiError::from(Error::InvalidSettle {
            spent: req.spent,
            reserved: req.reserved,
        }));
    }
    let settled = {
        let accounts = state.accounts.lock().expect("accounts lock");
        accounts.settle(&req.account, req.spent, req.reserved)?
    };
    match settled {
        Some((credits_remaining, credits_reserved)) => Ok(Json(SettleView {
            credits_remaining,
            credits_reserved,
        })),
        None => Err(ApiError::from(Error::AccountNotFound(req.account))),
    }
}

/// The price of one method on one chain, for the relay to charge the right
/// amount per request.
///
/// This is a separate route rather than a field folded into
/// `/internal/authenticate`, because the two answers change on different
/// clocks. A key's authenticate record is worth caching for a short TTL
/// (section 8): it rarely changes mid-session. A price applies per request,
/// per method, per chain — a single key calls many methods across a session —
/// so baking one price into the once-per-TTL authenticate response would
/// answer the wrong question. `price_of_normalized` is a `RwLock` read with
/// no database I/O, so a dedicated lookup here costs nothing extra.
async fn price_for_relay(
    State(state): State<AppState>,
    Query(q): Query<PriceQuery>,
) -> Json<PriceForRelayView> {
    let (canonical, credits) = {
        let pb = state.prices.lock().expect("prices lock");
        pb.price_of_normalized(&q.method, q.chain)
    };
    let known = canonical.is_some();
    let method = canonical.unwrap_or(q.method);
    Json(PriceForRelayView {
        method,
        chain_id: q.chain,
        credits,
        known,
    })
}

/// Per-key usage needs the metering hot path, which is not built. Return 501 and
/// a JSON note. Do not fabricate usage numbers.
async fn key_usage(Path(_id): Path<String>) -> impl IntoResponse {
    (
        StatusCode::NOT_IMPLEMENTED,
        Json(json!({ "note": "metering not yet wired" })),
    )
}

/// List every price row.
async fn list_pricing(State(state): State<AppState>) -> Json<Vec<PriceView>> {
    let rows = {
        let pb = state.prices.lock().expect("prices lock");
        pb.all_prices()
    };
    Json(
        rows.into_iter()
            .map(|(method, chain_id, credits)| PriceView {
                method,
                chain_id,
                credits,
            })
            .collect(),
    )
}

/// Set a method price. The chain comes from `?chain=N` (default 0). The price
/// book rejects a non-positive price with a 400.
async fn set_pricing(
    State(state): State<AppState>,
    Path(method): Path<String>,
    Query(q): Query<ChainQuery>,
    Json(req): Json<SetPriceRequest>,
) -> std::result::Result<Json<PriceView>, ApiError> {
    let chain_id = q.chain;
    {
        let pb = state.prices.lock().expect("prices lock");
        pb.set_price(&method, chain_id, req.credits)?;
    }
    Ok(Json(PriceView {
        method,
        chain_id,
        credits: req.credits,
    }))
}

/// Read the audit trail, newest first, capped at a sane default.
async fn read_audit(
    State(state): State<AppState>,
    Query(q): Query<AuditQuery>,
) -> std::result::Result<Json<Vec<AuditView>>, ApiError> {
    let limit = q
        .limit
        .unwrap_or(DEFAULT_AUDIT_LIMIT)
        .clamp(1, MAX_AUDIT_LIMIT);
    let rows = {
        let km = state.keys.lock().expect("keys lock");
        km.read_audit(limit)?
    };
    Ok(Json(rows.iter().map(AuditView::from).collect()))
}

// --- request and response DTOs ------------------------------------------------
//
// The DTOs carry serde, so `keys.rs` and `pricing.rs` stay serde-free. Each maps
// to or from a core type.

/// The rate wire form. `"unlimited"` or `{ "limited": { per_second, per_day } }`.
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
enum RateDto {
    #[default]
    Unlimited,
    Limited {
        per_second: i64,
        per_day: i64,
    },
}

impl From<RateDto> for Rate {
    fn from(r: RateDto) -> Rate {
        match r {
            RateDto::Unlimited => Rate::Unlimited,
            RateDto::Limited {
                per_second,
                per_day,
            } => Rate::Limited {
                per_second,
                per_day,
            },
        }
    }
}

impl From<&Rate> for RateDto {
    fn from(r: &Rate) -> RateDto {
        match r {
            Rate::Unlimited => RateDto::Unlimited,
            Rate::Limited {
                per_second,
                per_day,
            } => RateDto::Limited {
                per_second: *per_second,
                per_day: *per_day,
            },
        }
    }
}

/// The body of `POST /admin/keys`.
#[derive(Debug, Deserialize)]
struct CreateKeyRequest {
    label: String,
    account_address: Option<String>,
    #[serde(default)]
    credit_exempt: bool,
    #[serde(default)]
    allow_trace: bool,
    #[serde(default)]
    rate: RateDto,
    expires_at: Option<i64>,
    #[serde(default)]
    origins: Vec<String>,
    #[serde(default)]
    method_allow: Vec<String>,
    #[serde(default)]
    method_block: Vec<String>,
    #[serde(default)]
    networks: Vec<String>,
    #[serde(default)]
    ip_allow: Vec<String>,
    #[serde(default)]
    ip_deny: Vec<String>,
}

impl CreateKeyRequest {
    fn into_config(self) -> KeyConfig {
        KeyConfig {
            label: self.label,
            account_address: self.account_address,
            credit_exempt: self.credit_exempt,
            allow_trace: self.allow_trace,
            rate: self.rate.into(),
            expires_at: self.expires_at,
            origins: self.origins,
            method_allow: self.method_allow,
            method_block: self.method_block,
            networks: self.networks,
            ip_allow: self.ip_allow,
            ip_deny: self.ip_deny,
        }
    }
}

/// The reply to a create or a rotate: the public id and the raw key, shown once.
#[derive(Debug, Serialize)]
struct CreatedKey {
    id: String,
    key: String,
}

/// The body of `PATCH /admin/keys/:id`. A missing field is left unchanged.
#[derive(Debug, Deserialize)]
struct PatchKeyRequest {
    label: Option<String>,
    credit_exempt: Option<bool>,
    allow_trace: Option<bool>,
    rate: Option<RateDto>,
    expires_at: Option<i64>,
}

impl PatchKeyRequest {
    fn into_patch(self) -> KeyPatch {
        KeyPatch {
            label: self.label,
            credit_exempt: self.credit_exempt,
            allow_trace: self.allow_trace,
            rate: self.rate.map(Rate::from),
            expires_at: self.expires_at,
        }
    }
}

/// A key as returned to the admin surface. It holds no secret and no hash.
#[derive(Debug, Serialize)]
struct KeyView {
    id: String,
    label: String,
    account_address: Option<String>,
    credit_exempt: bool,
    allow_trace: bool,
    rate: RateDto,
    created_at: i64,
    disabled_at: Option<i64>,
    expires_at: Option<i64>,
}

impl From<&KeyRecord> for KeyView {
    fn from(r: &KeyRecord) -> KeyView {
        KeyView {
            id: r.id.clone(),
            label: r.label.clone(),
            account_address: r.account_address.clone(),
            credit_exempt: r.credit_exempt,
            allow_trace: r.allow_trace,
            rate: RateDto::from(&r.rate),
            created_at: r.created_at,
            disabled_at: r.disabled_at,
            expires_at: r.expires_at,
        }
    }
}

/// The body of `POST /internal/authenticate`.
#[derive(Debug, Deserialize)]
struct AuthenticateRequest {
    key: String,
}

/// A key's constraints, grouped by kind and named to match
/// `CreateKeyRequest`'s fields. The relay then reads the same shape it would
/// have sent to create the key, instead of an untyped (kind, value) list.
#[derive(Debug, Default, Serialize)]
struct ConstraintsView {
    origins: Vec<String>,
    method_allow: Vec<String>,
    method_block: Vec<String>,
    networks: Vec<String>,
    ip_allow: Vec<String>,
    ip_deny: Vec<String>,
}

impl ConstraintsView {
    /// Group the store's flat (kind, value) rows into their named buckets. An
    /// unrecognised kind is dropped rather than failing the whole read — the
    /// schema's vocabulary is closed, so this should never hit in practice.
    fn from_pairs(pairs: Vec<(String, String)>) -> ConstraintsView {
        let mut view = ConstraintsView::default();
        for (kind, value) in pairs {
            match kind.as_str() {
                "origin" => view.origins.push(value),
                "method_allow" => view.method_allow.push(value),
                "method_block" => view.method_block.push(value),
                "network" => view.networks.push(value),
                "ip_allow" => view.ip_allow.push(value),
                "ip_deny" => view.ip_deny.push(value),
                _ => {}
            }
        }
        view
    }
}

/// The reply to `POST /internal/authenticate`. It holds no secret: no key
/// hash, no raw key. `enabled` is always `true` here — `authenticate_key`
/// only ever builds this view for [`AuthOutcome::Active`] — but the field
/// stays explicit because the relay's contract names it.
#[derive(Debug, Serialize)]
struct AuthenticateView {
    id: String,
    label: String,
    enabled: bool,
    account_address: Option<String>,
    credit_exempt: bool,
    allow_trace: bool,
    rate: RateDto,
    constraints: ConstraintsView,
}

impl AuthenticateView {
    fn build(record: KeyRecord, constraint_pairs: Vec<(String, String)>) -> AuthenticateView {
        AuthenticateView {
            id: record.id,
            label: record.label,
            enabled: record.disabled_at.is_none(),
            account_address: record.account_address,
            credit_exempt: record.credit_exempt,
            allow_trace: record.allow_trace,
            rate: RateDto::from(&record.rate),
            constraints: ConstraintsView::from_pairs(constraint_pairs),
        }
    }
}

/// The body of `POST /internal/reserve`.
#[derive(Debug, Deserialize)]
struct ReserveRequest {
    account: String,
    credits: i64,
}

/// The reply to `POST /internal/reserve`. `granted` may be 0 — that is a
/// normal "out of credits" answer, not an error.
#[derive(Debug, Serialize)]
struct ReserveView {
    granted: i64,
}

/// The body of `POST /internal/settle`.
#[derive(Debug, Deserialize)]
struct SettleRequest {
    account: String,
    spent: i64,
    reserved: i64,
}

/// The reply to `POST /internal/settle`: the account's balance after the
/// settle, so the relay can confirm the arithmetic without a second read.
#[derive(Debug, Serialize)]
struct SettleView {
    credits_remaining: i64,
    credits_reserved: i64,
}

/// The query for `GET /internal/price`. `chain` defaults to 0 (any chain),
/// matching `method_pricing`'s own convention.
#[derive(Debug, Deserialize)]
struct PriceQuery {
    method: String,
    #[serde(default)]
    chain: i64,
}

/// The reply to `GET /internal/price`. `known` is false when `method` matched
/// no priced row and `credits` is therefore the "*" default, not a specific
/// price for that method — the relay can use this to tell "priced at the
/// default" from "priced because we recognise this method".
#[derive(Debug, Serialize)]
struct PriceForRelayView {
    method: String,
    chain_id: i64,
    credits: i64,
    known: bool,
}

/// The query for a per-chain price. `?chain=N`, default 0 (any chain).
#[derive(Debug, Deserialize)]
struct ChainQuery {
    #[serde(default)]
    chain: i64,
}

/// The body of `PUT /admin/pricing/:method`.
#[derive(Debug, Deserialize)]
struct SetPriceRequest {
    credits: i64,
}

/// One price row on the wire.
#[derive(Debug, Serialize)]
struct PriceView {
    method: String,
    chain_id: i64,
    credits: i64,
}

/// The query for an audit read. `?limit=N`, clamped to a sane range.
#[derive(Debug, Deserialize)]
struct AuditQuery {
    limit: Option<i64>,
}

/// One audit row on the wire.
#[derive(Debug, Serialize)]
struct AuditView {
    id: i64,
    ts: i64,
    actor: String,
    action: String,
    target: Option<String>,
    detail: Option<String>,
}

impl From<&AuditRow> for AuditView {
    fn from(r: &AuditRow) -> AuditView {
        AuditView {
            id: r.id,
            ts: r.ts,
            actor: r.actor.clone(),
            action: r.action.clone(),
            target: r.target.clone(),
            detail: r.detail.clone(),
        }
    }
}

// --- error mapping ------------------------------------------------------------

/// An HTTP error: a status and a safe message. It renders as `{ "error": ... }`.
struct ApiError {
    status: StatusCode,
    message: String,
}

impl IntoResponse for ApiError {
    fn into_response(self) -> Response {
        (self.status, Json(json!({ "error": self.message }))).into_response()
    }
}

impl From<Error> for ApiError {
    /// Map a core error to a status. A 500 returns a generic message, so a
    /// sqlite or io detail never leaks. A 400 or 404 message is safe: it names
    /// the public id or the price rule, never a secret.
    fn from(e: Error) -> ApiError {
        let status = match &e {
            Error::KeyNotFound(_) | Error::AccountNotFound(_) => StatusCode::NOT_FOUND,
            Error::InvalidPrice(_)
            | Error::RangeInverted { .. }
            | Error::RangeTooWide { .. }
            | Error::InvalidCredits(_)
            | Error::InvalidSettle { .. } => StatusCode::BAD_REQUEST,
            // A missing pepper is a startup fault surfaced at request time.
            Error::MissingPepper
            | Error::MissingAdminToken
            | Error::MissingRelayToken
            | Error::AddrNotLoopback(_)
            | Error::Sqlite(_)
            | Error::Io(_) => StatusCode::INTERNAL_SERVER_ERROR,
        };
        let message = if status == StatusCode::INTERNAL_SERVER_ERROR {
            "internal error".to_string()
        } else {
            e.to_string()
        };
        ApiError { status, message }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use axum::body::Body;
    use axum::http::Request as HttpRequest;
    use http_body_util::BodyExt;
    use tower::ServiceExt;

    use crate::keys::KeyManager;
    use crate::pricing::PriceBook;
    use crate::store::{unix_now, Store};

    const TOKEN: &str = "test-admin-token";
    const RELAY_TOKEN: &str = "test-relay-token";
    const PEPPER: &[u8] = b"test-pepper";

    /// A temp-file database path, so both the key manager and the price book can
    /// open the same store.
    fn temp_db_path(tag: &str) -> std::path::PathBuf {
        let mut path = std::env::temp_dir();
        path.push(format!(
            "jumpgate-admin-{}-{}-{}.db",
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

    /// Build a router over a fresh temp-file store. Returns the router and the DB
    /// path, so the test can clean up.
    fn test_app(tag: &str) -> (Router, std::path::PathBuf) {
        let path = temp_db_path(tag);
        let km = KeyManager::new(Store::open(&path).unwrap(), PEPPER).unwrap();
        let pb = PriceBook::new(Store::open(&path).unwrap()).unwrap();
        let state = AppState::new(km, pb, TOKEN.to_string(), RELAY_TOKEN.to_string()).unwrap();
        (build_router(state), path)
    }

    /// Build the `AppState` directly (not the router), for the `serve_unix`
    /// tests, which need to drive the transport rather than call `.oneshot()`.
    fn test_state(tag: &str) -> (AppState, std::path::PathBuf) {
        let path = temp_db_path(tag);
        let km = KeyManager::new(Store::open(&path).unwrap(), PEPPER).unwrap();
        let pb = PriceBook::new(Store::open(&path).unwrap()).unwrap();
        let state = AppState::new(km, pb, TOKEN.to_string(), RELAY_TOKEN.to_string()).unwrap();
        (state, path)
    }

    /// Send a request and return the status and the parsed JSON body (or null for
    /// an empty body).
    async fn send(app: &Router, req: HttpRequest<Body>) -> (StatusCode, serde_json::Value) {
        let res = app.clone().oneshot(req).await.unwrap();
        let status = res.status();
        let bytes = res.into_body().collect().await.unwrap().to_bytes();
        let body = if bytes.is_empty() {
            serde_json::Value::Null
        } else {
            serde_json::from_slice(&bytes).unwrap()
        };
        (status, body)
    }

    fn get(uri: &str, token: Option<&str>) -> HttpRequest<Body> {
        let mut b = HttpRequest::builder().method("GET").uri(uri);
        if let Some(t) = token {
            b = b.header(header::AUTHORIZATION, format!("Bearer {t}"));
        }
        b.body(Body::empty()).unwrap()
    }

    fn json_req(
        method: &str,
        uri: &str,
        token: &str,
        body: serde_json::Value,
    ) -> HttpRequest<Body> {
        HttpRequest::builder()
            .method(method)
            .uri(uri)
            .header(header::AUTHORIZATION, format!("Bearer {token}"))
            .header(header::CONTENT_TYPE, "application/json")
            .body(Body::from(serde_json::to_vec(&body).unwrap()))
            .unwrap()
    }

    /// Create one key through the API and return its id and raw key.
    async fn create_key(app: &Router, label: &str) -> (String, String) {
        let (status, body) = send(
            app,
            json_req("POST", "/admin/keys", TOKEN, json!({ "label": label })),
        )
        .await;
        assert_eq!(status, StatusCode::CREATED);
        let id = body["id"].as_str().unwrap().to_string();
        let key = body["key"].as_str().unwrap().to_string();
        (id, key)
    }

    #[tokio::test]
    async fn missing_auth_is_rejected() {
        let (app, path) = test_app("noauth");
        let (status, body) = send(&app, get("/admin/keys", None)).await;
        assert_eq!(status, StatusCode::UNAUTHORIZED);
        assert_eq!(body["error"], "unauthorized");
        remove_db(&path);
    }

    #[tokio::test]
    async fn wrong_token_is_rejected() {
        let (app, path) = test_app("wrongtoken");
        let (status, _) = send(&app, get("/admin/keys", Some("not-the-token"))).await;
        assert_eq!(status, StatusCode::UNAUTHORIZED);
        remove_db(&path);
    }

    #[tokio::test]
    async fn correct_token_lists_keys() {
        let (app, path) = test_app("righttoken");
        let (status, body) = send(&app, get("/admin/keys", Some(TOKEN))).await;
        assert_eq!(status, StatusCode::OK);
        assert!(body.is_array());
        remove_db(&path);
    }

    #[tokio::test]
    async fn healthz_needs_no_auth() {
        let (app, path) = test_app("healthz");
        let (status, body) = send(&app, get("/healthz", None)).await;
        assert_eq!(status, StatusCode::OK);
        assert_eq!(body["status"], "ok");
        remove_db(&path);
    }

    #[tokio::test]
    async fn create_returns_raw_key_and_it_lists() {
        let (app, path) = test_app("create");
        let (id, key) = create_key(&app, "first-key").await;
        assert!(key.starts_with("jg_"), "raw key must carry the jg_ prefix");
        assert!(id.starts_with("k_"), "public id must carry the k_ prefix");

        // The key now shows in the list.
        let (status, body) = send(&app, get("/admin/keys", Some(TOKEN))).await;
        assert_eq!(status, StatusCode::OK);
        let found = body
            .as_array()
            .unwrap()
            .iter()
            .any(|k| k["id"] == id && k["label"] == "first-key");
        assert!(found, "the created key must appear in the list");
        remove_db(&path);
    }

    #[tokio::test]
    async fn delete_revokes_and_list_shows_it_disabled() {
        let (app, path) = test_app("delete");
        let (id, _) = create_key(&app, "to-revoke").await;

        let del = HttpRequest::builder()
            .method("DELETE")
            .uri(format!("/admin/keys/{id}"))
            .header(header::AUTHORIZATION, format!("Bearer {TOKEN}"))
            .body(Body::empty())
            .unwrap();
        let (status, _) = send(&app, del).await;
        assert_eq!(status, StatusCode::NO_CONTENT);

        let (_, body) = send(&app, get("/admin/keys", Some(TOKEN))).await;
        let row = body
            .as_array()
            .unwrap()
            .iter()
            .find(|k| k["id"] == id)
            .expect("the revoked key still lists");
        assert!(
            !row["disabled_at"].is_null(),
            "a revoked key must carry disabled_at"
        );
        remove_db(&path);
    }

    #[tokio::test]
    async fn rotate_returns_a_new_key() {
        let (app, path) = test_app("rotate");
        let (id, old_key) = create_key(&app, "to-rotate").await;

        let req = HttpRequest::builder()
            .method("POST")
            .uri(format!("/admin/keys/{id}/rotate"))
            .header(header::AUTHORIZATION, format!("Bearer {TOKEN}"))
            .body(Body::empty())
            .unwrap();
        let (status, body) = send(&app, req).await;
        assert_eq!(status, StatusCode::OK);
        let new_key = body["key"].as_str().unwrap();
        assert!(new_key.starts_with("jg_"));
        assert_ne!(new_key, old_key, "rotate must issue a different key");
        remove_db(&path);
    }

    #[tokio::test]
    async fn patch_changes_the_label() {
        let (app, path) = test_app("patch");
        let (id, _) = create_key(&app, "old-label").await;

        let (status, body) = send(
            &app,
            json_req(
                "PATCH",
                &format!("/admin/keys/{id}"),
                TOKEN,
                json!({ "label": "new-label" }),
            ),
        )
        .await;
        assert_eq!(status, StatusCode::OK);
        assert_eq!(body["label"], "new-label");

        // Read it back through the list.
        let (_, body) = send(&app, get("/admin/keys", Some(TOKEN))).await;
        let row = body
            .as_array()
            .unwrap()
            .iter()
            .find(|k| k["id"] == id)
            .unwrap();
        assert_eq!(row["label"], "new-label");
        remove_db(&path);
    }

    #[tokio::test]
    async fn put_pricing_sets_and_rejects_non_positive() {
        let (app, path) = test_app("pricing");

        // A valid price change reads back through the pricing list.
        let (status, body) = send(
            &app,
            json_req(
                "PUT",
                "/admin/pricing/eth_getLogs?chain=0",
                TOKEN,
                json!({ "credits": 123 }),
            ),
        )
        .await;
        assert_eq!(status, StatusCode::OK);
        assert_eq!(body["credits"], 123);

        let (_, body) = send(&app, get("/admin/pricing", Some(TOKEN))).await;
        let row = body
            .as_array()
            .unwrap()
            .iter()
            .find(|p| p["method"] == "eth_getLogs" && p["chain_id"] == 0)
            .expect("the price row must list");
        assert_eq!(row["credits"], 123);

        // A non-positive price is rejected with a 400.
        let (status, _) = send(
            &app,
            json_req(
                "PUT",
                "/admin/pricing/eth_getLogs",
                TOKEN,
                json!({ "credits": 0 }),
            ),
        )
        .await;
        assert_eq!(status, StatusCode::BAD_REQUEST);
        remove_db(&path);
    }

    #[tokio::test]
    async fn audit_shows_the_mutations() {
        let (app, path) = test_app("audit");
        let (id, _) = create_key(&app, "audited").await;

        // A price change and a revoke add more audit rows.
        send(
            &app,
            json_req(
                "PUT",
                "/admin/pricing/eth_call",
                TOKEN,
                json!({ "credits": 25 }),
            ),
        )
        .await;
        let del = HttpRequest::builder()
            .method("DELETE")
            .uri(format!("/admin/keys/{id}"))
            .header(header::AUTHORIZATION, format!("Bearer {TOKEN}"))
            .body(Body::empty())
            .unwrap();
        send(&app, del).await;

        let (status, body) = send(&app, get("/admin/audit", Some(TOKEN))).await;
        assert_eq!(status, StatusCode::OK);
        let actions: Vec<&str> = body
            .as_array()
            .unwrap()
            .iter()
            .map(|r| r["action"].as_str().unwrap())
            .collect();
        assert!(actions.contains(&"key.create"), "audit shows key.create");
        assert!(actions.contains(&"price.set"), "audit shows price.set");
        assert!(actions.contains(&"key.revoke"), "audit shows key.revoke");
        // Newest first: the revoke is the last mutation, so it leads.
        assert_eq!(actions.first(), Some(&"key.revoke"));
        remove_db(&path);
    }

    #[tokio::test]
    async fn unknown_id_mutations_return_404_and_leak_no_secret() {
        let (app, path) = test_app("notfound");

        // Rotate an unknown id.
        let rot = HttpRequest::builder()
            .method("POST")
            .uri("/admin/keys/k_missing/rotate")
            .header(header::AUTHORIZATION, format!("Bearer {TOKEN}"))
            .body(Body::empty())
            .unwrap();
        let (status, body) = send(&app, rot).await;
        assert_eq!(status, StatusCode::NOT_FOUND);
        let text = body.to_string();
        assert!(
            !text.contains(TOKEN),
            "the 404 body must not leak the token"
        );
        assert!(
            !text.contains("jg_"),
            "the 404 body must not leak a raw key"
        );

        // Revoke an unknown id.
        let del = HttpRequest::builder()
            .method("DELETE")
            .uri("/admin/keys/k_missing")
            .header(header::AUTHORIZATION, format!("Bearer {TOKEN}"))
            .body(Body::empty())
            .unwrap();
        let (status, _) = send(&app, del).await;
        assert_eq!(status, StatusCode::NOT_FOUND);

        // Patch an unknown id.
        let (status, _) = send(
            &app,
            json_req(
                "PATCH",
                "/admin/keys/k_missing",
                TOKEN,
                json!({ "label": "x" }),
            ),
        )
        .await;
        assert_eq!(status, StatusCode::NOT_FOUND);

        remove_db(&path);
    }

    #[tokio::test]
    async fn non_positive_price_is_a_400() {
        let (app, path) = test_app("badprice");
        // A negative price is rejected before any write, mapped to 400.
        let (status, _) = send(
            &app,
            json_req(
                "PUT",
                "/admin/pricing/eth_call",
                TOKEN,
                json!({ "credits": -5 }),
            ),
        )
        .await;
        assert_eq!(status, StatusCode::BAD_REQUEST);
        remove_db(&path);
    }

    #[tokio::test]
    async fn malformed_json_is_a_client_error_not_a_500() {
        let (app, path) = test_app("badjson");
        let req = HttpRequest::builder()
            .method("POST")
            .uri("/admin/keys")
            .header(header::AUTHORIZATION, format!("Bearer {TOKEN}"))
            .header(header::CONTENT_TYPE, "application/json")
            .body(Body::from("{ not valid json "))
            .unwrap();
        // The rejection body is plain text, so read it directly rather than
        // through the JSON-parsing `send` helper.
        let res = app.clone().oneshot(req).await.unwrap();
        let status = res.status();
        let bytes = res.into_body().collect().await.unwrap().to_bytes();
        let text = String::from_utf8_lossy(&bytes);
        assert!(
            status.is_client_error(),
            "malformed JSON must be a 4xx, got {status}"
        );
        assert_ne!(status, StatusCode::INTERNAL_SERVER_ERROR);
        assert!(
            !text.contains(TOKEN),
            "the error body must not leak the token"
        );
        remove_db(&path);
    }

    #[tokio::test]
    async fn set_pricing_for_a_specific_chain_via_query() {
        let (app, path) = test_app("chainprice");

        // Set a per-chain override for chain 369.
        let (status, body) = send(
            &app,
            json_req(
                "PUT",
                "/admin/pricing/eth_call?chain=369",
                TOKEN,
                json!({ "credits": 44 }),
            ),
        )
        .await;
        assert_eq!(status, StatusCode::OK);
        assert_eq!(body["chain_id"], 369);
        assert_eq!(body["credits"], 44);

        // The per-chain row lists, and the any-chain default stays at 20.
        let (_, body) = send(&app, get("/admin/pricing", Some(TOKEN))).await;
        let rows = body.as_array().unwrap();
        let per_chain = rows
            .iter()
            .find(|p| p["method"] == "eth_call" && p["chain_id"] == 369)
            .expect("the per-chain row must list");
        assert_eq!(per_chain["credits"], 44);
        let any_chain = rows
            .iter()
            .find(|p| p["method"] == "eth_call" && p["chain_id"] == 0)
            .expect("the any-chain default row must still list");
        assert_eq!(
            any_chain["credits"], 20,
            "the per-chain set must not touch the any-chain default"
        );

        remove_db(&path);
    }

    #[tokio::test]
    async fn audit_limit_is_clamped() {
        let (app, path) = test_app("auditclamp");
        // Two creates leave at least two audit rows.
        create_key(&app, "one").await;
        create_key(&app, "two").await;

        // limit=0 clamps up to 1, so exactly the newest row returns.
        let (status, body) = send(&app, get("/admin/audit?limit=0", Some(TOKEN))).await;
        assert_eq!(status, StatusCode::OK);
        assert_eq!(
            body.as_array().unwrap().len(),
            1,
            "limit=0 must clamp up to 1"
        );

        // A limit above the hard cap is accepted (clamped) and still returns rows.
        let (status, body) = send(&app, get("/admin/audit?limit=99999", Some(TOKEN))).await;
        assert_eq!(status, StatusCode::OK);
        assert!(body.as_array().unwrap().len() >= 2);

        remove_db(&path);
    }

    #[tokio::test]
    async fn usage_is_not_implemented() {
        let (app, path) = test_app("usage");
        let (id, _) = create_key(&app, "usage-key").await;
        let (status, body) = send(&app, get(&format!("/admin/keys/{id}/usage"), Some(TOKEN))).await;
        assert_eq!(status, StatusCode::NOT_IMPLEMENTED);
        assert_eq!(body["note"], "metering not yet wired");
        remove_db(&path);
    }

    // --- least-privilege split: /internal/authenticate ------------------------

    #[tokio::test]
    async fn internal_authenticate_returns_the_record_for_a_valid_key() {
        let (app, path) = test_app("authok");
        let (id, raw) = create_key(&app, "relay-target").await;

        let (status, body) = send(
            &app,
            json_req(
                "POST",
                "/internal/authenticate",
                RELAY_TOKEN,
                json!({ "key": raw }),
            ),
        )
        .await;
        assert_eq!(status, StatusCode::OK);
        assert_eq!(body["id"], id);
        assert_eq!(body["label"], "relay-target");
        assert_eq!(body["enabled"], true);
        assert_eq!(body["credit_exempt"], false);
        assert_eq!(body["allow_trace"], false);
        assert_eq!(body["rate"], "unlimited");
        assert!(body["account_address"].is_null());
        assert!(body["constraints"].is_object(), "constraints must be present");
        remove_db(&path);
    }

    #[tokio::test]
    async fn internal_authenticate_never_leaks_the_hash_or_the_raw_key() {
        let (app, path) = test_app("authnoleak");
        let (_, raw) = create_key(&app, "no-leak").await;

        let (status, body) = send(
            &app,
            json_req(
                "POST",
                "/internal/authenticate",
                RELAY_TOKEN,
                json!({ "key": raw }),
            ),
        )
        .await;
        assert_eq!(status, StatusCode::OK);
        let text = body.to_string();
        assert!(
            !text.contains(&raw),
            "the response must not echo the raw key back"
        );
        assert!(
            body.get("key_hash").is_none(),
            "the response must not carry a key_hash field"
        );
        assert!(
            body.get("key").is_none(),
            "the response must not carry a raw key field"
        );
        remove_db(&path);
    }

    #[tokio::test]
    async fn internal_authenticate_rejects_an_unknown_key() {
        let (app, path) = test_app("authunknown");
        let (status, _) = send(
            &app,
            json_req(
                "POST",
                "/internal/authenticate",
                RELAY_TOKEN,
                json!({ "key": "jg_not_a_real_key" }),
            ),
        )
        .await;
        assert_eq!(status, StatusCode::UNAUTHORIZED);
        remove_db(&path);
    }

    #[tokio::test]
    async fn internal_authenticate_rejects_a_disabled_key() {
        let (app, path) = test_app("authdisabled");
        let (id, raw) = create_key(&app, "to-disable").await;

        let del = HttpRequest::builder()
            .method("DELETE")
            .uri(format!("/admin/keys/{id}"))
            .header(header::AUTHORIZATION, format!("Bearer {TOKEN}"))
            .body(Body::empty())
            .unwrap();
        send(&app, del).await;

        let (status, _) = send(
            &app,
            json_req(
                "POST",
                "/internal/authenticate",
                RELAY_TOKEN,
                json!({ "key": raw }),
            ),
        )
        .await;
        assert_eq!(status, StatusCode::FORBIDDEN);
        remove_db(&path);
    }

    #[tokio::test]
    async fn internal_authenticate_reads_back_the_constraints() {
        let (app, path) = test_app("authconstraints");
        let (_, body) = send(
            &app,
            json_req(
                "POST",
                "/admin/keys",
                TOKEN,
                json!({
                    "label": "constrained",
                    "method_allow": ["eth_call"],
                    "ip_deny": ["10.0.0.2"],
                }),
            ),
        )
        .await;
        let raw = body["key"].as_str().unwrap().to_string();

        let (status, body) = send(
            &app,
            json_req(
                "POST",
                "/internal/authenticate",
                RELAY_TOKEN,
                json!({ "key": raw }),
            ),
        )
        .await;
        assert_eq!(status, StatusCode::OK);
        assert_eq!(body["constraints"]["method_allow"], json!(["eth_call"]));
        assert_eq!(body["constraints"]["ip_deny"], json!(["10.0.0.2"]));
        assert_eq!(body["constraints"]["origins"], json!([]));
        remove_db(&path);
    }

    #[tokio::test]
    async fn admin_token_does_not_open_internal_authenticate() {
        let (app, path) = test_app("crossadmin");
        let (_, raw) = create_key(&app, "cross-check").await;

        // The admin token is valid — just not for this route.
        let (status, _) = send(
            &app,
            json_req(
                "POST",
                "/internal/authenticate",
                TOKEN,
                json!({ "key": raw }),
            ),
        )
        .await;
        assert_eq!(
            status,
            StatusCode::UNAUTHORIZED,
            "the admin token must not authorize /internal/*"
        );
        remove_db(&path);
    }

    #[tokio::test]
    async fn relay_token_does_not_open_admin_routes() {
        let (app, path) = test_app("crossrelay");

        let (status, _) = send(&app, get("/admin/keys", Some(RELAY_TOKEN))).await;
        assert_eq!(
            status,
            StatusCode::UNAUTHORIZED,
            "the relay token must not authorize /admin/*"
        );
        remove_db(&path);
    }

    #[tokio::test]
    async fn internal_authenticate_needs_a_token_at_all() {
        let (app, path) = test_app("authnotoken");
        let (_, raw) = create_key(&app, "needs-token").await;

        let req = HttpRequest::builder()
            .method("POST")
            .uri("/internal/authenticate")
            .header(header::CONTENT_TYPE, "application/json")
            .body(Body::from(serde_json::to_vec(&json!({ "key": raw })).unwrap()))
            .unwrap();
        let (status, _) = send(&app, req).await;
        assert_eq!(status, StatusCode::UNAUTHORIZED);
        remove_db(&path);
    }

    // --- the unix transport -----------------------------------------------

    #[test]
    fn peer_uid_ok_matches_only_the_expected_uid() {
        assert!(peer_uid_ok(1000, 1000));
        assert!(!peer_uid_ok(1000, 1001));
        assert!(!peer_uid_ok(0, 1000));
    }

    /// A temp directory path for a unix-socket test. Nested one level under a
    /// throwaway parent, so the test also exercises `serve_unix` creating that
    /// parent directory at `0700`.
    ///
    /// This uses `/tmp` directly, not `std::env::temp_dir()`. A unix socket
    /// path is capped at roughly 100 bytes by the kernel's `sockaddr_un`
    /// (the exact cap varies, macOS is 104), and `std::env::temp_dir()` on
    /// macOS already returns a long per-process path that leaves too little
    /// room for a name plus `/billing.sock`.
    fn temp_socket_path(tag: &str) -> std::path::PathBuf {
        let mut dir = std::path::PathBuf::from("/tmp");
        dir.push(format!(
            "jg-{tag}-{}-{}",
            std::process::id(),
            unix_now() % 100_000
        ));
        dir.push("b.sock");
        dir
    }

    /// Retry-connect to a unix socket path, bounded, so a test never hangs if
    /// `serve_unix` fails to bind. Connecting, not just checking the path
    /// exists, is the right readiness signal: the stale-socket test leaves a
    /// plain file at the path before `serve_unix` even starts, and a plain
    /// file "exists" too — only a real, bound socket accepts a connection.
    async fn wait_for_socket(path: &std::path::Path) -> tokio::net::UnixStream {
        for _ in 0..200 {
            match tokio::net::UnixStream::connect(path).await {
                Ok(stream) => return stream,
                Err(_) => tokio::time::sleep(std::time::Duration::from_millis(10)).await,
            }
        }
        panic!("could not connect to {}", path.display());
    }

    /// Send one raw HTTP/1.1 request over a connected unix stream and return
    /// the response text. No hyper client needed: the request is tiny and
    /// fixed, so a hand-written request line is clearer than pulling in a
    /// client stack for one call.
    async fn raw_http_get(stream: &mut tokio::net::UnixStream, path: &str) -> String {
        use tokio::io::{AsyncReadExt, AsyncWriteExt};
        let req = format!(
            "GET {path} HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n"
        );
        stream.write_all(req.as_bytes()).await.unwrap();
        let mut out = Vec::new();
        stream.read_to_end(&mut out).await.unwrap();
        String::from_utf8_lossy(&out).into_owned()
    }

    #[tokio::test]
    async fn serve_unix_binds_owner_only_and_serves_a_request() {
        use std::os::unix::fs::PermissionsExt;

        let (state, dbpath) = test_state("unixserve");
        let sock_path = temp_socket_path("unixserve");
        let dir = sock_path.parent().unwrap().to_path_buf();

        let sock_path_for_task = sock_path.clone();
        let handle = tokio::spawn(async move {
            let _ = serve_unix(&sock_path_for_task, state).await;
        });

        let mut stream = wait_for_socket(&sock_path).await;

        let dir_mode = std::fs::metadata(&dir).unwrap().permissions().mode() & 0o777;
        assert_eq!(dir_mode, 0o700, "the socket's parent directory must be 0700");
        let sock_mode = std::fs::metadata(&sock_path).unwrap().permissions().mode() & 0o777;
        assert_eq!(sock_mode, 0o600, "the socket file must be 0600");

        let response = raw_http_get(&mut stream, "/healthz").await;
        assert!(
            response.starts_with("HTTP/1.1 200"),
            "expected a 200 from /healthz over the unix socket, got: {response}"
        );
        assert!(response.contains("\"status\":\"ok\""));

        handle.abort();
        let _ = std::fs::remove_dir_all(&dir);
        remove_db(&dbpath);
    }

    #[tokio::test]
    async fn serve_unix_removes_a_stale_socket_file_before_binding() {
        let (state, dbpath) = test_state("unixstale");
        let sock_path = temp_socket_path("unixstale");
        let dir = sock_path.parent().unwrap().to_path_buf();

        // Simulate a crashed process: the directory and a plain file already
        // sit at the socket path. `UnixListener::bind` refuses to bind over an
        // existing path, so this only works if `serve_unix` unlinks it first.
        std::fs::create_dir_all(&dir).unwrap();
        std::fs::write(&sock_path, b"stale").unwrap();

        let sock_path_for_task = sock_path.clone();
        let handle = tokio::spawn(async move {
            let _ = serve_unix(&sock_path_for_task, state).await;
        });

        let mut stream = wait_for_socket(&sock_path).await;
        let response = raw_http_get(&mut stream, "/healthz").await;
        assert!(
            response.starts_with("HTTP/1.1 200"),
            "the service must bind and serve even with a stale file at the path, got: {response}"
        );

        handle.abort();
        let _ = std::fs::remove_dir_all(&dir);
        remove_db(&dbpath);
    }
}
