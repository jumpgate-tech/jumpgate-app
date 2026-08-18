//! A thin CLI for the billing store. Step 1 shipped `init`. Step 2 adds a
//! `keys` command group: create, list, revoke, and rotate keys.
//!
//! The key commands need the HMAC pepper. It comes from the environment variable
//! `JUMPGATE_KEY_PEPPER` and never has a default; an empty pepper is a hard
//! error. The pepper lives beside the signer secret, never in the database.

use std::net::SocketAddr;
use std::path::PathBuf;
use std::process::exit;

use billing::admin::{self, AppState};
use billing::keys::{KeyConfig, KeyManager, Rate};
use billing::pricing::PriceBook;
use billing::store::Store;

const DEFAULT_DB: &str = "jumpgate-billing.db";
const PEPPER_ENV: &str = "JUMPGATE_KEY_PEPPER";
const ADMIN_TOKEN_ENV: &str = "JUMPGATE_ADMIN_TOKEN";
const RELAY_TOKEN_ENV: &str = "JUMPGATE_RELAY_TOKEN";
const DEFAULT_ADMIN_ADDR: &str = "127.0.0.1:8787";

fn main() {
    if let Err(e) = run() {
        eprintln!("error: {e}");
        exit(1);
    }
}

fn run() -> billing::Result<()> {
    let mut args = std::env::args().skip(1);
    match args.next().as_deref() {
        Some("init") => cmd_init(args),
        Some("keys") => cmd_keys(args),
        Some("price") => cmd_price(args),
        Some("serve") => cmd_serve(args),
        _ => {
            usage();
            exit(2);
        }
    }
}

fn cmd_init(mut args: impl Iterator<Item = String>) -> billing::Result<()> {
    let path = args
        .next()
        .map(PathBuf::from)
        .unwrap_or_else(|| PathBuf::from(DEFAULT_DB));
    let store = Store::open(&path)?;
    println!("initialised {}", path.display());
    println!("  {} pricing rows seeded", store.price_count()?);
    println!(
        "  eth_call        = {} credits",
        store.price_of("eth_call", 1)?
    );
    println!(
        "  eth_getLogs     = {} credits",
        store.price_of("eth_getLogs", 1)?
    );
    println!(
        "  unknown method  = {} credits (default)",
        store.price_of("foo_bar", 1)?
    );
    Ok(())
}

fn cmd_keys(mut args: impl Iterator<Item = String>) -> billing::Result<()> {
    let sub = args.next();
    let rest: Vec<String> = args.collect();
    match sub.as_deref() {
        Some("create") => keys_create(&rest),
        Some("list") => keys_list(&rest),
        Some("revoke") => keys_revoke(&rest),
        Some("rotate") => keys_rotate(&rest),
        _ => {
            usage();
            exit(2);
        }
    }
}

/// Read the pepper from the environment. Fail hard when it is unset or empty;
/// never default it to empty, or the keyed hash becomes an unkeyed one.
fn pepper() -> billing::Result<Vec<u8>> {
    match std::env::var(PEPPER_ENV) {
        Ok(v) if !v.is_empty() => Ok(v.into_bytes()),
        _ => {
            eprintln!("error: set {PEPPER_ENV} to a non-empty value");
            exit(1);
        }
    }
}

/// Build the manager for a keys command from the shared flags.
fn open_manager(flags: &Flags) -> billing::Result<KeyManager> {
    let pepper = pepper()?;
    let store = Store::open(&flags.db)?;
    KeyManager::new(store, &pepper)
}

fn keys_create(args: &[String]) -> billing::Result<()> {
    let flags = Flags::parse(args);
    let km = open_manager(&flags)?;

    let rate = match (flags.per_second, flags.per_day) {
        (Some(per_second), Some(per_day)) => Rate::Limited {
            per_second,
            per_day,
        },
        _ => Rate::Unlimited,
    };
    let config = KeyConfig {
        label: flags.label.unwrap_or_default(),
        account_address: flags.account,
        credit_exempt: flags.exempt,
        allow_trace: flags.allow_trace,
        rate,
        expires_at: flags.expires_at,
        ..Default::default()
    };

    let (id, raw) = km.create(config)?;
    println!("created {id}");
    println!("  raw key (shown once): {raw}");
    println!("  keys in map: {}", km.key_count());
    Ok(())
}

fn keys_list(args: &[String]) -> billing::Result<()> {
    let flags = Flags::parse(args);
    let km = open_manager(&flags)?;
    let records = km.list()?;
    println!("{} key(s):", records.len());
    for r in records {
        let state = match r.disabled_at {
            Some(ts) => format!("revoked@{ts}"),
            None => "active".to_string(),
        };
        let rate = match r.rate {
            Rate::Unlimited => "unlimited".to_string(),
            Rate::Limited {
                per_second,
                per_day,
            } => format!("{per_second}/s {per_day}/day"),
        };
        println!(
            "  {id}  {state:<16}  rate={rate:<14}  exempt={exempt}  trace={trace}  label={label:?}",
            id = r.id,
            exempt = r.credit_exempt,
            trace = r.allow_trace,
            label = r.label,
        );
    }
    Ok(())
}

fn keys_revoke(args: &[String]) -> billing::Result<()> {
    let flags = Flags::parse(args);
    let id = flags.positional.clone().unwrap_or_else(|| {
        eprintln!("usage: billing keys revoke <id> [--db path]");
        exit(2);
    });
    let km = open_manager(&flags)?;
    km.revoke(&id)?;
    println!("revoked {id}");
    Ok(())
}

fn keys_rotate(args: &[String]) -> billing::Result<()> {
    let flags = Flags::parse(args);
    let id = flags.positional.clone().unwrap_or_else(|| {
        eprintln!("usage: billing keys rotate <id> [--db path]");
        exit(2);
    });
    let km = open_manager(&flags)?;
    let raw = km.rotate(&id)?;
    println!("rotated {id}");
    println!("  new raw key (shown once): {raw}");
    Ok(())
}

fn cmd_price(mut args: impl Iterator<Item = String>) -> billing::Result<()> {
    let sub = args.next();
    let rest: Vec<String> = args.collect();
    match sub.as_deref() {
        Some("list") => price_list(&rest),
        Some("get") => price_get(&rest),
        Some("set") => price_set(&rest),
        _ => {
            usage();
            exit(2);
        }
    }
}

/// Open the price book for a price command. Unlike the key commands, pricing
/// needs no pepper; it only reads and writes the store.
fn open_book(flags: &Flags) -> billing::Result<PriceBook> {
    let store = Store::open(&flags.db)?;
    PriceBook::new(store)
}

fn price_list(args: &[String]) -> billing::Result<()> {
    let flags = Flags::parse(args);
    let book = open_book(&flags)?;
    let rows = book.all_prices();
    println!("{} price row(s):", rows.len());
    for (method, chain, credits) in rows {
        let scope = if chain == 0 {
            "any".to_string()
        } else {
            chain.to_string()
        };
        println!("  {method:<40}  chain={scope:<8}  {credits} credits");
    }
    Ok(())
}

fn price_get(args: &[String]) -> billing::Result<()> {
    let flags = Flags::parse(args);
    let method = flags.positionals.first().cloned().unwrap_or_else(|| {
        eprintln!("usage: billing price get <method> [--chain N] [--db path]");
        exit(2);
    });
    let chain = flags.chain.unwrap_or(0);
    let book = open_book(&flags)?;
    let (canonical, credits) = book.price_of_normalized(&method, chain);
    match canonical {
        Some(name) => println!("{name} (chain {chain}) = {credits} credits"),
        None => {
            println!("{method} is not a known priced method; default = {credits} credits")
        }
    }
    Ok(())
}

fn price_set(args: &[String]) -> billing::Result<()> {
    let flags = Flags::parse(args);
    if flags.positionals.len() < 2 {
        eprintln!("usage: billing price set <method> <credits> [--chain N] [--db path]");
        exit(2);
    }
    let method = flags.positionals[0].clone();
    let credits = parse_int(&flags.positionals[1]);
    let chain = flags.chain.unwrap_or(0);
    let book = open_book(&flags)?;
    book.set_price(&method, chain, credits)?;
    println!("set {method} (chain {chain}) = {credits} credits");
    Ok(())
}

/// Read the admin bearer token from the environment. Fail hard when it is unset
/// or empty; never default it, or the admin gate opens by default (finding S9).
fn admin_token() -> billing::Result<String> {
    match std::env::var(ADMIN_TOKEN_ENV) {
        Ok(v) if !v.is_empty() => Ok(v),
        _ => {
            eprintln!("error: set {ADMIN_TOKEN_ENV} to a non-empty value");
            exit(1);
        }
    }
}

/// Read the relay bearer token from the environment. Fail hard when it is
/// unset or empty, the same posture as [`admin_token`] and the pepper: a
/// missing credential must never widen into "no credential needed".
fn relay_token() -> billing::Result<String> {
    match std::env::var(RELAY_TOKEN_ENV) {
        Ok(v) if !v.is_empty() => Ok(v),
        _ => {
            eprintln!("error: set {RELAY_TOKEN_ENV} to a non-empty value");
            exit(1);
        }
    }
}

/// Serve the admin API. This is the one async command, so `main` stays
/// synchronous: it builds a tokio runtime and blocks on the server. It reads
/// the pepper (for the key manager) and both bearer tokens (fail closed).
///
/// `--addr` (TCP, loopback only) and `--socket` (a unix socket) pick the
/// transport. They are mutually exclusive — passing both leaves no single
/// answer for which one the operator meant, so it is a usage error, not a
/// silent pick of one. `--socket` is the transport the relay uses (design doc
/// section 7); `--addr` stays for a platform with no reliable `AF_UNIX`.
fn cmd_serve(args: impl Iterator<Item = String>) -> billing::Result<()> {
    let rest: Vec<String> = args.collect();
    let flags = Flags::parse(&rest);

    if flags.addr.is_some() && flags.socket.is_some() {
        eprintln!("error: --addr and --socket are mutually exclusive");
        exit(2);
    }

    let pepper = pepper()?;
    let token = admin_token()?;
    let relay = relay_token()?;

    // The key manager and the price book each need their own connection to the
    // same file. WAL mode lets the two connections share the database.
    let km = KeyManager::new(Store::open(&flags.db)?, &pepper)?;
    let pb = PriceBook::new(Store::open(&flags.db)?)?;
    let state = AppState::new(km, pb, token, relay)?;

    let runtime = tokio::runtime::Runtime::new()?;

    if let Some(socket_str) = &flags.socket {
        #[cfg(unix)]
        {
            let path = PathBuf::from(socket_str);
            return runtime.block_on(admin::serve_unix(&path, state));
        }
        #[cfg(not(unix))]
        {
            eprintln!(
                "error: --socket needs a unix target; this build has no AF_UNIX, use --addr"
            );
            exit(2);
        }
    }

    let addr_str = flags.addr.as_deref().unwrap_or(DEFAULT_ADMIN_ADDR);
    let addr: SocketAddr = match addr_str.parse() {
        Ok(a) => a,
        Err(_) => {
            eprintln!("error: --addr must be an IP:port, got {addr_str:?}");
            exit(2);
        }
    };
    runtime.block_on(admin::serve(addr, state))
}

/// The parsed flags shared across the keys commands. Hand-rolled to keep the
/// dependency set minimal, matching the existing CLI style.
#[derive(Default)]
struct Flags {
    db: PathBuf,
    positional: Option<String>,
    positionals: Vec<String>,
    chain: Option<i64>,
    label: Option<String>,
    account: Option<String>,
    exempt: bool,
    allow_trace: bool,
    per_second: Option<i64>,
    per_day: Option<i64>,
    expires_at: Option<i64>,
    addr: Option<String>,
    socket: Option<String>,
}

impl Flags {
    fn parse(args: &[String]) -> Flags {
        let mut f = Flags {
            db: PathBuf::from(DEFAULT_DB),
            ..Default::default()
        };
        let mut it = args.iter();
        while let Some(arg) = it.next() {
            match arg.as_str() {
                "--db" => f.db = PathBuf::from(next_value(&mut it, "--db")),
                "--label" => f.label = Some(next_value(&mut it, "--label")),
                "--account" => f.account = Some(next_value(&mut it, "--account")),
                "--exempt" => f.exempt = true,
                "--allow-trace" => f.allow_trace = true,
                "--per-second" => {
                    f.per_second = Some(parse_int(&next_value(&mut it, "--per-second")))
                }
                "--per-day" => f.per_day = Some(parse_int(&next_value(&mut it, "--per-day"))),
                "--expires-at" => {
                    f.expires_at = Some(parse_int(&next_value(&mut it, "--expires-at")))
                }
                "--chain" => f.chain = Some(parse_int(&next_value(&mut it, "--chain"))),
                "--addr" => f.addr = Some(next_value(&mut it, "--addr")),
                "--socket" => f.socket = Some(next_value(&mut it, "--socket")),
                other if other.starts_with("--") => {
                    eprintln!("error: unknown flag {other}");
                    exit(2);
                }
                other => {
                    // Keep the first positional in `positional` for the key
                    // commands, and collect every positional for the price
                    // commands (method, credits).
                    if f.positional.is_none() {
                        f.positional = Some(other.to_string());
                    }
                    f.positionals.push(other.to_string());
                }
            }
        }
        f
    }
}

fn next_value<'a>(it: &mut impl Iterator<Item = &'a String>, flag: &str) -> String {
    match it.next() {
        Some(v) => v.clone(),
        None => {
            eprintln!("error: {flag} needs a value");
            exit(2);
        }
    }
}

fn parse_int(s: &str) -> i64 {
    match s.parse() {
        Ok(n) => n,
        Err(_) => {
            eprintln!("error: expected an integer, got {s:?}");
            exit(2);
        }
    }
}

fn usage() {
    eprintln!("usage:");
    eprintln!("  billing init [db-path]");
    eprintln!("  billing keys create [--db path] [--label L] [--account A] [--exempt] \\");
    eprintln!(
        "                      [--allow-trace] [--per-second N --per-day N] [--expires-at TS]"
    );
    eprintln!("  billing keys list   [--db path]");
    eprintln!("  billing keys revoke <id> [--db path]");
    eprintln!("  billing keys rotate <id> [--db path]");
    eprintln!("  billing price list  [--db path]");
    eprintln!("  billing price get <method> [--chain N] [--db path]");
    eprintln!("  billing price set <method> <credits> [--chain N] [--db path]");
    eprintln!("  billing serve [--db path] [--addr 127.0.0.1:8787] [--socket path]");
    eprintln!("                (--addr and --socket are mutually exclusive)");
    eprintln!("  ({PEPPER_ENV} must be set for every keys command and for serve)");
    eprintln!("  ({ADMIN_TOKEN_ENV} must be set for serve; it gates every /admin/* route)");
    eprintln!(
        "  ({RELAY_TOKEN_ENV} must be set for serve; it gates /internal/authenticate only)"
    );
}
