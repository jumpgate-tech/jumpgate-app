# Extract `internal/wsrpc`

**Date:** 2026-07-27
**Backlog item:** 3 — "Extract `internal/wsrpc`", unblocked by the `wsframe_test.go` triplet
landed 2026-07-27.

## Why

The RFC 6455 client is written three times: `internal/chainlist`, `internal/capabilities`,
`internal/setup`. The duplication is not theoretical debt — the three copies have already
drifted apart in ways that matter:

| | `chainlist` | `capabilities` | `setup/tlsverify` |
|---|---|---|---|
| handshake signature | `(conn, br, *url.URL)` | `(conn, br, *url.URL)` | `(w, br, host, path)` |
| non-101 error | status only | status only, `errWSRefused` | status **+ response body** |
| frame cap | 64 KB (`maxProbeBytes`) | 1 MB (`maxFrameBytes`) | 8 MB (inline `8<<20`) |
| oversize wording | "oversized" | "oversized" | "implausible" |
| reader shape | `wsReadMessage` + `wsReadFrame` | same | one fused function |
| dial | own TCP + opportunistic TLS | own TCP + opportunistic TLS | caller's **pinned** `tls.Dialer` |
| lifetime | one round trip | one round trip | **persistent** — reads `newHeads` after the reply |

This code is load-bearing. It is how an endpoint that advertises `wss://` but cannot speak it
gets caught — true of `rpc.pulsechain.com` and of all four published chain-943 endpoints — and
it is the only check that catches an eRPC gateway whose upstream scheme makes `eth_chainId`
work while every `eth_subscribe` is refused.

## Decisions

Settled with the owner before writing code:

1. **`MaxMessageBytes` is an option, defaulting to 1 MB; all three call sites take the
   default.** The knob exists for a future caller with a real need, but nothing passes a
   different value today, so the drift cannot silently recur. 64 KB was sized for an
   `eth_chainId` reply and 8 MB for a `newHeads` header; a real `newHeads` is a few KB, so
   both were wrong by an order of magnitude in opposite directions.
2. **`wsrpc` serves both dial styles.** `Dial` for a caller that just has a URL;
   `Handshake` for a caller that has already dialled — which `setup` must, because it pins
   the connection to a verified cert pool and a fixed address. All frame code is shared, so
   all three copies go.
3. **Every refusal carries the body and a sentinel.** `setup` reads a truncated response body
   on a non-101 and the other two do not; that body is how the eRPC "HTTP 500 whenever the
   client sends `Accept-Encoding: gzip`" behaviour was diagnosed. `capabilities` has
   `errWSRefused`/`errWSNoAnswer` sentinels and the other two do not; the difference between
   "this endpoint refuses WebSocket" and "we could not tell" is the whole point of the probe.
   Unifying takes the best of each, so every call site ends up strictly more diagnosable.

## API

```go
package wsrpc

var ErrRefused  error // the server answered the upgrade and said no
var ErrNoAnswer error // the upgrade succeeded and then nothing came back

type Options struct {
    Dialer          *net.Dialer  // nil = a default dialer
    TLSConfig       *tls.Config  // nil = ServerName from the URL
    MaxMessageBytes int64        // 0 = DefaultMaxMessageBytes (1 MB)
    HostHeader      string       // "" = the URL's host
}

func Dial(ctx context.Context, rawURL string, opt *Options) (*Conn, error)
func Handshake(ctx context.Context, conn net.Conn, host, path string, opt *Options) (*Conn, error)
func RoundTrip(ctx context.Context, rawURL string, payload []byte, opt *Options) ([]byte, error)

func (c *Conn) WriteText(payload []byte) error
func (c *Conn) ReadMessage() ([]byte, error)
func (c *Conn) SetDeadline(t time.Time) error
func (c *Conn) Close() error
```

`Dial` owns the connection it opened, so it installs the context watchdog the current
`chainlist`/`capabilities` copies have — a goroutine that closes the socket on cancellation so
a blocked read unblocks — and `Conn.Close` stops it. `Handshake` does not: the caller dialled,
so the caller's deadline discipline governs. That is exactly how `setup` already works.

## Steps

1. **Move the spec.** `wsframe_test.go` → `internal/wsrpc/frame_test.go`, unchanged in
   substance: same frame builder, same cases. `maxProbeBytes`/`wsFrameCap` become
   `DefaultMaxMessageBytes`, and the oversize assertion settles on "oversized" (the wording two
   of the three copies already use).
2. **Write `internal/wsrpc`** against that spec: `frame.go` (read/write/mask/cap) and
   `wsrpc.go` (`Dial`, `Handshake`, `RoundTrip`, `Conn`, sentinels).
3. **Add the handshake tests the triplet never had** — nothing currently covers the accept
   digest, the non-101 body, or the sentinels, and all three are changing. Drive them off an
   `httptest` server plus a hand-written 101 responder.
4. **Rewire `chainlist`**: `probeWS` becomes `wsrpc.RoundTrip` + `parseChainID`.
5. **Rewire `capabilities`**: `wsRoundTrip` becomes `wsrpc.RoundTrip`; `errWSRefused`/
   `errWSNoAnswer` become the package sentinels; `unwrapRefusal` follows.
6. **Rewire `setup`**: keep the pinned `tls.Dialer`, hand the live conn to `wsrpc.Handshake`,
   keep the `newHeads` read loop on `Conn.ReadMessage`.
7. **Delete** all three `wsHandshake`/`wsWriteFrame`/`wsWriteText`/`wsReadMessage`/
   `wsReadFrame` copies, the three `wsGUID` constants, and the three `wsframe_test.go` files.

## Verification

- `go build ./... && go test ./...` green.
- `grep -c "Sec-WebSocket-Key" internal/**/*.go` — 1, in `wsrpc`.
- **Mutation, not inspection.** This repo's worst bugs report success while broken, so each
  guard is checked by breaking it: remove the cap and the oversize test must fail; skip the
  accept-digest check and the digest test must fail; drop the mask and the write test must
  fail. A guard whose test has never been seen to fail is not evidence of anything.
- Coverage of `internal/wsrpc` at or above the 83.2% repo total.
