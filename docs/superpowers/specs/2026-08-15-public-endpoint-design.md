# Slice A — Public branded endpoint (design)

Date: 2026-08-15. Part of [self-hosted metered RPC](../plans/self-hosted-metered-rpc.md).
Status: design for review, not yet built.

## Goal

Give a gateway a third trust tier: a **Public** mode where the operator sets
their own domain and the front serves a real, publicly-trusted certificate. No
wallet has to install anything. This is the foundation the metered slices (B–D)
sit on.

## Scope

**In:** a new `acme` cert source; the operator's real domain as the site
address; stock Caddy obtaining a Let's Encrypt certificate over HTTP-01 /
TLS-ALPN; binding `:80` for the ACME challenge; the settings UI; validation; the
health-probe change for a public name.

**Out (later slices):** the `/rpc/<key>/...` path shape, key validation, and any
metering. Slice A serves the gateway's EXISTING path (`/<project>/evm/<chainId>`)
on the public domain. The `/rpc` prefix and the key belong to the relay (slice
B), because eRPC will not read a key from the path and something must translate
it — that something is the relay, which does not exist yet.

**Refinement to the roadmap:** the roadmap table says slice A "strips `/rpc`".
That is wrong — stripping `/rpc/<key>` only makes sense once the relay lifts the
key. Slice A does no path rewriting. I will correct the roadmap table.

**Safety note:** slice A alone yields an OPEN public endpoint — anyone who
learns the URL can use it for free. That is acceptable for the foundation, but
the operator must not put it in front of real customers until slice B adds the
key gate. The UI must say so.

## The three tiers, expressed in existing fields

No new "exposure mode" enum is needed. The tiers already fall out of two fields
on `catalog.GatewayTLS`:

| Tier | Expressed as |
|------|--------------|
| Local | `TLS.Enabled = false` — no front, loopback http (the wallet URL already shipped) |
| Private | `TLS.Enabled = true`, `CertSource = "internal"` — self-signed, manual trust |
| Public | `TLS.Enabled = true`, `CertSource = "acme"` — real cert, trusted everywhere |

`CertACME = "acme"` already exists as a reserved constant
(`internal/catalog/caddy.go:45`); slice A implements it. This is the thin choice:
one new cert-source value, not a new dimension.

## Design

### 1. Certificate acquisition
Stock `caddy:2-alpine` (`DefaultCaddyImage`, caddy.go:51) does ACME natively —
HTTP-01 and TLS-ALPN-01, auto-HTTPS on by default for a public site address. No
image change. The `/data` volume already mounted (docker.go, `CaddyDataVolume`)
persists the ACME account key and issued certs, so recreating the container does
NOT re-issue — this matters for Let's Encrypt rate limits.

Requirements the operator must meet (documented in the UI, not enforceable by
us): the box is publicly reachable on :80 and :443, and a DNS `A`/`AAAA` record
for the domain points at it. HTTP-01 needs :80 reachable from the internet.

### 2. Caddyfile rendering (`internal/catalog/caddy.go`)
Current template (caddy.go:294):
```
{
	auto_https disable_redirects
}
{{.Hostname}} {
{{.TLSDirective}}
	reverse_proxy {{.Upstream}}
}
```
For `acme`:
- The global `auto_https disable_redirects` must be **omitted** — it is what
  stops Caddy binding :80. A public front needs :80 for the HTTP-01 challenge and
  the http→https redirect. Make the global block conditional on cert source.
- `{{.Hostname}}` is already the site address — for `acme` it is the operator's
  real domain. Caddy triggers ACME automatically for a public name.
- `TLSDirective`: for `acme`, emit either nothing (Caddy auto-HTTPS) or, if the
  operator gave an email, `tls <email>`. Add an `acme` arm to the `switch` in
  `RenderCaddyfile` (caddy.go:~377) beside `internal` and `files`.

### 3. Port binding (`internal/ops/docker.go`)
`CaddyRunArgs` (docker.go:1106) publishes only `:443` today
(`publishSpec(bind, hostPort, CaddyHTTPSPort)`, CaddyHTTPSPort=443). For `acme`:
- Add `CaddyHTTPPort = 80` beside `CaddyHTTPSPort` (docker.go:1052).
- Publish a second mapping `-p <bind>:80:80` when the cert source is `acme`.
  Gate it on cert source so Private/Local gateways do not grab :80.
- Bind stays `0.0.0.0` (already the default for the Caddy front).

### 4. Port preflight (`internal/setup/gateway.go`)
`checkPortFree` (gateway.go:~468) probes only the HTTPS port for a fronted
gateway, as the one deliberate non-reclaiming exception. For `acme`, add the
same `reclaim=false` probe for :80. A busy :80 must fail preflight with a clear
message, because ACME will silently fail without it.

### 5. Config + validation
- `catalog.GatewayTLS` (caddy.go:165): add optional `ACMEEmail string`. Reuse
  `Hostname` for the public domain — no separate field.
- Relax the ACME rejection in `CaddyConfig.Validate()` (caddy.go:~361). Replace
  "not implemented" with real checks for `acme`: `Hostname` must be a public
  FQDN — it must contain a dot, must NOT end in `.localhost-valaxy.com`, and must
  not be a bare IP. An email, if given, must look like one.
- Thread through `handleGatewayCreate` / `handleGatewayPutConfig`
  (`internal/server/gateways.go`) — they already call `ValidateSettings()`, so
  the new checks ride the existing seam.

### 6. Settings UI (`cmd/valve-node-app/web/src/screens/Rpc/SettingsBlock.tsx`)
- Add a third option to the cert-source `<select>` (SettingsBlock.tsx:214):
  `<option value="acme">A public domain — a real certificate, trusted everywhere</option>`.
- When `acme` is selected, label the existing hostname input as the public
  domain (e.g. `rpc.your-company.com`) and reveal an optional email input.
- Show a plain note: the box must be reachable on ports 80 and 443, a DNS record
  must point at it, and — until slice B — the endpoint is open to anyone.
- Mirror `ACMEEmail` in `web/src/api.ts` `GatewayTLS` (api.ts:~665).
- `web/src/lib/gatewaySetup.ts` `internalTLSConfig` (gatewaySetup.ts:21):
  add a public variant, or branch on the chosen source.
- Rebuild dist (`npx vite build`) — CI-enforced.

### 7. Health probe (`internal/setup/gateway.go`)
`probeCommand` (gateway.go:~1141) pins `--resolve` to loopback and passes the
internal CA as `--cacert`. For `acme`, both are wrong: a real Let's Encrypt cert
verifies against the system trust store, and `--resolve` to loopback is
meaningless once the name is genuinely public. For `acme`, verify against system
trust (drop `--cacert`) and drop the `--resolve` pin. The existing
`|| attempt("")` fallthrough already covers the system-trust case.

## Testing

Renderer + arg tests (pure, fast — the reliable core):
- `RenderCaddyfile` for `acme`: no `auto_https disable_redirects`; site address
  is the real domain; `tls <email>` present only when an email is set.
- `CaddyRunArgs` for `acme`: publishes both `:443` and `:80`; a non-acme gateway
  publishes only `:443`.
- `Validate` for `acme`: accepts a public FQDN; rejects `*.localhost-valaxy.com`,
  a bare IP, and a single-label host.
- `checkPortFree` adds the :80 probe for `acme`.

verify-by-running-it — HONEST LIMIT: a full ACME issuance needs a real public
domain with DNS pointing at a reachable box, which the build box does not have.
So end-to-end issuance cannot be tested here. Two real checks are still worth
running: (1) start a Caddy container with a rendered `acme` config against Let's
Encrypt **staging** for a throwaway domain if one is available, and confirm the
container comes up and attempts issuance; (2) assert the generated Caddyfile is
what a public front needs by diffing it against a hand-verified fixture. State
plainly in the PR that live issuance was not exercised on the build box.

## Open forks — decided

- **Separate exposure enum?** No. Cert source + `Enabled` already express the
  three tiers. Thinner, fewer fields to validate.
- **Separate public-domain field?** No. Reuse `Hostname`.
- **Strip `/rpc` in slice A?** No. That needs the relay (slice B).

## Files touched (summary)

- `internal/catalog/caddy.go` — `ACMEEmail` field; relax ACME rejection; `acme`
  arm in `RenderCaddyfile`; conditional global block.
- `internal/ops/docker.go` — `CaddyHTTPPort`; publish :80 for `acme`.
- `internal/setup/gateway.go` — :80 preflight; `acme` health-probe branch.
- `internal/server/gateways.go` — validation rides existing `ValidateSettings`.
- `cmd/valve-node-app/web/src/screens/Rpc/SettingsBlock.tsx`, `web/src/api.ts`,
  `web/src/lib/gatewaySetup.ts` — the `acme` option + domain/email inputs; dist
  rebuild.
</content>
