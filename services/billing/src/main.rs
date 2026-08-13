//! A thin CLI for the billing store. Step 1 exposes one command: `init`.
//! It creates or migrates the database and prints a short summary, so the
//! store is verifiable by running it.

use std::path::PathBuf;
use std::process::exit;

use billing::store::Store;

fn main() {
    if let Err(e) = run() {
        eprintln!("error: {e}");
        exit(1);
    }
}

fn run() -> billing::Result<()> {
    let mut args = std::env::args().skip(1);
    match args.next().as_deref() {
        Some("init") => {
            let path = args
                .next()
                .map(PathBuf::from)
                .unwrap_or_else(|| PathBuf::from("jumpgate-billing.db"));
            let store = Store::open(&path)?;
            println!("initialised {}", path.display());
            println!("  {} pricing rows seeded", store.price_count()?);
            println!("  eth_call        = {} credits", store.price_of("eth_call", 1)?);
            println!("  eth_getLogs     = {} credits", store.price_of("eth_getLogs", 1)?);
            println!("  unknown method  = {} credits (default)", store.price_of("foo_bar", 1)?);
            Ok(())
        }
        _ => {
            eprintln!("usage: billing init [db-path]");
            exit(2);
        }
    }
}
