# What `jumpgate lock` removes from memory

Locking the vault (`jumpgate lock`, the UI's lock button, or the idle
auto-lock) makes the running jumpgate stop using its secrets. It does not
wipe every copy of every secret from the process. This page says which is
which, so you can decide what a lock is worth on your machine.

## Cleared on lock

- **The vault's data key.** Its bytes are overwritten with zeros and the
  reference is dropped. Nothing can be decrypted again until the next unlock.
- **Every entry value the vault held.** The decrypted values inside the
  vault object are overwritten with zeros and dropped.
- **The controller signer, when the vault holds the controller key.** The
  server forgets its signer, so nothing new is signed until the next unlock.
  Requests that need it answer "locked" instead of falling back to anything
  else.

## Not cleared on lock

- **Strings made from secrets.** Code that reads a secret to use it (a
  provider API key, the AI key, a VPN config that was resolved to bring a
  tunnel up) turns it into a Go string. Go strings cannot be overwritten. They
  stay in memory until the garbage collector reuses that memory, which can be
  long after the lock.
- **The controller key object.** The server drops its reference, but the key
  is not overwritten. A plan or request that had already fetched the signer
  before the lock still holds it until it finishes, and the memory lingers
  until garbage collection.
- **SSH transport signers.** Each connection to a gate gets its own
  `ssh.Signer` for the controller's transport key, parsed out of the vault.
  A connection that is already open keeps its signer, and the key material is
  not overwritten when it is dropped.
- **The running relay's tokens.** The in-process relay reads the relay and
  admin tokens once, when it starts after the first unlock. It keeps using
  them while the vault is locked. A token change reaches the relay only
  after jumpgate restarts.
- **Running VPN tunnels.** A tunnel brought up while unlocked stays up. Its
  config already went to the tunnel process.
- **Other copies.** Anything else that briefly held a secret, such as HTTP
  request and response buffers, JSON encoding buffers, or the terminal or
  clipboard you typed the passphrase from, is outside the vault's control.

## Swap and crash dumps

Nothing is locked into RAM (there is no `mlock`), so the data key and secrets
can be written to swap or the page file. Encrypt swap if that matters.

Before deriving any key, jumpgate turns off core dumps:

- **Linux:** the process is marked not dumpable, so no core file is written
  and other processes of the same user cannot attach to it with ptrace.
- **macOS:** the core size limit is set to zero, and the process refuses
  debuggers (`PT_DENY_ATTACH`). Running jumpgate under a debugger stops
  working after the first unlock.
- **FreeBSD and other Unix systems:** the core size limit is set to zero.
  Other processes of the same user can still attach a debugger.
- **Windows:** nothing is done. Windows Error Reporting may still write a
  crash dump that holds secrets, and processes of the same user can read this
  process's memory.

## Old copies of the vault file

Changing the passphrase or removing a key slot gives the vault a new data
key. After that, the old passphrase, the old recovery code and any old sealed
key no longer open the current file. The change also removes the recovery
slot and every machine-held slot (systemd-creds, keychain and so on), because
their keys can't be re-wrapped without the original material. Run
`jumpgate vault add-recovery` and `jumpgate vault seal` again to add them back.

A copy or backup of the vault file taken before the change is a separate
file. It still opens with the credentials it had when it was copied. If an
old passphrase or recovery code might be known to someone else, destroy the
old copies too.

## What a lock is good for

A lock stops jumpgate from signing or decrypting anything new. It also
removes the data key, so a later memory dump can't be used to decrypt the
vault file. It is not a guarantee that no secret remains anywhere in the
process. For that, stop jumpgate.
