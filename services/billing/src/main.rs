//! A thin CLI for the billing store. Step 1 shipped `init`. Step 2 adds a
//! `keys` command group: create, list, revoke, and rotate keys.
//!
//! The key commands need the HMAC pepper. It comes from the environment variable
//! `JUMPGATE_KEY_PEPPER` and never has a default; an empty pepper is a hard
//! error. The pepper lives beside the signer secret, never in the database.

use std::path::PathBuf;
use std::process::exit;

use billing::keys::{KeyConfig, KeyManager, Rate};
use billing::store::Store;

const DEFAULT_DB: &str = "jumpgate-billing.db";
const PEPPER_ENV: &str = "JUMPGATE_KEY_PEPPER";

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

/// The parsed flags shared across the keys commands. Hand-rolled to keep the
/// dependency set minimal, matching the existing CLI style.
#[derive(Default)]
struct Flags {
    db: PathBuf,
    positional: Option<String>,
    label: Option<String>,
    account: Option<String>,
    exempt: bool,
    allow_trace: bool,
    per_second: Option<i64>,
    per_day: Option<i64>,
    expires_at: Option<i64>,
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
                other if other.starts_with("--") => {
                    eprintln!("error: unknown flag {other}");
                    exit(2);
                }
                other => f.positional = Some(other.to_string()),
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
    eprintln!("  ({PEPPER_ENV} must be set for every keys command)");
}
