//! The operator admin HTTP API: keys, pricing, and the audit log.
//!
//! This is step 4 of the billing build (docs/design/billing-admin.md section 5).
//! The store owns the SQL, the key manager owns the crypto, and the price book
//! owns the pricing map. This module is the thin HTTP shell over the three.
//!
//! Security shape (findings S9, S10, S11):
//!   - The API binds loopback only. [`serve`] refuses a non-loopback address.
//!   - Every `/admin/*` route needs the bearer token from `JUMPGATE_ADMIN_TOKEN`.
//!     The compare runs in constant time. A missing or wrong token returns 401
//!     with a small JSON error and no detail. The server refuses to start with an
//!     empty token, so the gate cannot open by default.
//!   - `GET /healthz` is the one open route. It touches no secret.
//!   - The raw key and the token never appear in a response body or an error.
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
use crate::keys::{KeyConfig, KeyManager, KeyPatch, KeyRecord, Rate};
use crate::pricing::PriceBook;
use crate::store::AuditRow;
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
    /// The admin bearer token. It never appears in a response or a log.
    token: Arc<String>,
}

impl AppState {
    /// Build the state. It refuses an empty token, so the gate cannot open by
    /// default (finding S9). The caller reads the token from
    /// `JUMPGATE_ADMIN_TOKEN` and must fail closed when it is unset.
    pub fn new(keys: KeyManager, prices: PriceBook, token: String) -> Result<AppState> {
        if token.is_empty() {
            return Err(Error::MissingAdminToken);
        }
        Ok(AppState {
            keys: Arc::new(Mutex::new(keys)),
            prices: Arc::new(Mutex::new(prices)),
            token: Arc::new(token),
        })
    }
}

/// Build the router: the open `GET /healthz` plus the guarded `/admin/*` routes.
/// The bearer middleware is a route layer on the admin routes only, so healthz
/// stays open.
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

    Router::new()
        .route("/healthz", get(healthz))
        .merge(admin)
        .with_state(state)
}

/// Bind the loopback address and serve the admin API. It refuses a non-loopback
/// address before it binds (finding S9/S3 posture). The bind and serve I/O errors
/// map to [`Error::Io`].
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

// --- auth ---------------------------------------------------------------------

/// The bearer gate on every `/admin/*` route. It reads the `Authorization`
/// header, expects `Bearer <token>`, and compares the token in constant time. A
/// missing or wrong token returns 401 with a small JSON error and no detail.
async fn require_bearer(State(state): State<AppState>, req: Request, next: Next) -> Response {
    let presented = req
        .headers()
        .get(header::AUTHORIZATION)
        .and_then(|v| v.to_str().ok())
        .and_then(|v| v.strip_prefix("Bearer "));

    let ok = match presented {
        // `ct_eq` on slices returns false at once for a length mismatch and runs
        // in constant time for an equal length, so it does not leak the token by
        // timing.
        Some(token) => token.as_bytes().ct_eq(state.token.as_bytes()).into(),
        None => false,
    };

    if ok {
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
            Error::KeyNotFound(_) => StatusCode::NOT_FOUND,
            Error::InvalidPrice(_) | Error::RangeInverted { .. } | Error::RangeTooWide { .. } => {
                StatusCode::BAD_REQUEST
            }
            // A missing pepper is a startup fault surfaced at request time.
            Error::MissingPepper
            | Error::MissingAdminToken
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
        let state = AppState::new(km, pb, TOKEN.to_string()).unwrap();
        (build_router(state), path)
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
}
