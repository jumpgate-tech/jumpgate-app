# API-key templates: one mechanism for valve, Infura and whoever comes next

**Date:** 2026-07-28
**Status:** approved, implementing
**Supersedes:** §5 of `2026-07-28-rpc-resilience-readout-design.md` (the per-chain valve key)

The app already knows what an API-key endpoint is. `chainlist.isTemplated()` matches
`${`, and every URL carrying a placeholder is rejected with *"API-key template (contains
${...}); requires a provider account"*. What it cannot do is the other half: **fill one
in.** So a provider slot is only ever a thing to discard.

Meanwhile valve's endpoint — which is an API-key endpoint of exactly that shape — was
built as a special case: the key baked into the URL by `catalog.KnownSet`, its own
`Config.ValveKeys` store, its own explanatory copy. Two mechanisms for one idea, and the
general one is the half that does nothing.

This makes templates first-class. valve stops being special, and an operator with an
Infura key stops throwing two live Ethereum endpoints away.

## What is true today

- `internal/chainlist/chainlist.go:383` — `isTemplated(raw) = strings.Contains(raw, "${")`.
  Deliberately broad, so a new provider's placeholder is caught without a code change.
- `chainlist.go:360-366` — templated URLs get `StatusRejected`. The comment calls the
  filter **mandatory**: *"chain 1 alone carries two."*
- Those two, discarded from the live feed on 2026-07-28:
  `https://mainnet.infura.io/v3/${INFURA_API_KEY}` and
  `wss://mainnet.infura.io/ws/v3/${INFURA_API_KEY}` — one of them a WebSocket endpoint,
  on the chain where WebSocket coverage matters most.
- `internal/catalog/knownset.go` resolves valve's key itself, per chain, from
  `Config.ValveKeys map[int]string`.
- `GET /api/gateways/{gid}/knownset/{chainId}` returns `key` — the secret itself — in the
  response body.
- **The settings page already has the right convention for a secret.** `aiKey` is stored
  and never returned; the API exposes `aiKeySet bool`, and the input reads *"•••••••• (leave
  blank to keep)"*. Provider keys are the same class of thing and get the same treatment.
- **`vk_demo` is not what I claimed.** Measured 2026-07-28: `x-valve-tier: FULL`, no
  rate-limit headers, 20/20 sequential requests answered, and a bogus key gets `401` — so
  it is a real credential on the full tier. The earlier "shared quota, runs dry" wording
  was inference from the word *demo*, never verified, and is withdrawn.

## 1. A template is a URL with a placeholder, and placeholders have names

`${NAME}` is the shape the feed uses, so it is the shape the app adopts. The name is the
identity: `VALVE_API_KEY`, `INFURA_API_KEY`, `ALCHEMY_API_KEY`. Detection stays exactly as
it is — broad, matching `${` — because a provider the app has never heard of must still be
recognised as templated on the day it appears in the feed.

## 2. Keys are stored per placeholder, not per chain

`Config.ProviderKeys map[string]string`, keyed by placeholder name.

This replaces `Config.ValveKeys map[int]string`. Per-chain was justified only by valve's
key sitting in the URL path; as a general rule it is wrong — an Infura key is an account,
not a chain. A per-chain override remains expressible later if a real case turns up, but
it is not built now, because a mechanism nobody has asked for is a mechanism nobody has
tested.

**Migration:** any existing `valveKeys` entries collapse to a single `VALVE_API_KEY`. If
entries disagree, the first by chain id wins and the rest are reported to the operator
rather than silently dropped — the same stance the orphan record takes.

**Keys are secrets and follow the `aiKey` convention:** stored, never returned. The API
exposes which placeholders are set, never their values. This corrects
`GET /api/gateways/{gid}/knownset/{chainId}`, which currently returns the key itself.

## 3. Resolve what we can, reject what we cannot — with the reason naming the key

`isTemplated` keeps its job. A new step follows it:

- placeholder has a stored value, or a shipped default → substitute, and the URL becomes a
  normal candidate, **probed like any other**. A resolved template is not trusted because
  it resolved; it is trusted because it answered.
- placeholder has no value → `StatusRejected`, as today, but the reason names the
  placeholder: *"needs INFURA_API_KEY — add it in Settings"*. Today's message says an
  account is required without saying which, which is the difference between a dead end and
  a next step.

## 4. valve becomes a template with a default

`catalog.KnownSet` returns `https://one.valve.city/rpc/${VALVE_API_KEY}/evm/1`, unresolved.
`VALVE_API_KEY` ships with the default value `vk_demo`, so the set still works with no
setup — that property is preserved exactly.

`KnownSet` stops taking a key argument and stops resolving anything. Resolution happens at
one seam every endpoint crosses, whether it came from the set or the feed, so the two
paths cannot drift.

The modal's copy stops describing a shared quota. What is true and worth saying: which
key a template resolved with, and that a key of your own can replace the shipped default.

## 5. One place to enter a key

Settings gains a provider-keys section: one row per placeholder the app has seen — from
the known sets and from the feed for the chains configured — each a password input
following the `aiKey` pattern, showing whether a key is set rather than its value.

A placeholder is listed because something wants it, so the list is evidence of what would
become usable, not a catalogue of providers.

## Delivery

1. `ProviderKeys` + the migration off `ValveKeys`, and the secret-handling correction.
2. Resolution in `chainlist`, with the named rejection reason.
3. `KnownSet` returns templates; the one resolution seam.
4. The settings section.

§1–§3 are the mechanism; §4 is what makes it reachable. §1 must not ship alone, because
between §1 and §3 the set would carry an unresolved placeholder.

## Verification

- Go tests: substitution, a placeholder with no key, the `ValveKeys` migration including
  the disagreement case, and that a resolved template is still probed rather than assumed.
- **The check that proves it:** with no Infura key, `evm:1` reports two rejected endpoints
  naming `INFURA_API_KEY`. Add a key in settings; the same two become live candidates and
  appear in the picker. That is one operator action turning discarded feed entries into
  usable upstreams, and it is the whole point.
- The known set must still add successfully **with no keys configured at all**, resolving
  `VALVE_API_KEY` to `vk_demo` — the zero-setup path is a regression risk here, not a
  given.

## Explicitly not in scope

- Per-chain key overrides.
- Validating a key against its provider before storing it; a wrong key fails its probe,
  which is the same signal every other endpoint gets.
- Any change to how eRPC scores or orders upstreams.
