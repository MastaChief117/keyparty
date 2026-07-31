# Security

We take this seriously — not "trust me bro" seriously, actually seriously. Like, we-wrote-tests-and-did-a-full-audit seriously.

## Protections

| Protection | Implementation |
|-----------|----------------|
| **Encryption at Rest** | AES-256-GCM via `golang.org/x/crypto`. All API keys encrypted in SQLite. Decrypted only in memory during request handling. |
| **Key File** | `.gateway.key` stored with `0600` permissions. 256-bit random value generated on first run. Lost key = encrypted data is gone forever. |
| **Guardrails** | PII detection (SSN, email, phone), prompt injection blocking via regex patterns, custom rules configurable per-deployment. |
| **SSRF Protection** | Uses `net.ParseIP()` with `IsLoopback()`, `IsPrivate()`, `IsLinkLocalUnicast()`, `IsUnspecified()` — not string matching. Blocks IPv6-mapped, decimal IPs, hex IPs, and cloud metadata endpoints. |
| **Admin Auth** | Bearer token auth on all `/admin/*` endpoints. Refuses to start without a password (returns 503). |
| **Rate Limiting** | Per-IP brute-force protection on admin auth (5 attempts/min lockout). |
| **Request Logging** | Full audit trail — who requested what, when, which provider, cost, latency. |
| **Panic Recovery** | All handlers wrapped in recovery middleware. One bad request won't crash the server. |
| **Cache Isolation** | Response cache includes virtual key ID in hash. Different users can't see each other's cached responses. |
| **Budget Enforcement** | Atomic budget deduction via SQL `WHERE` guard. No race conditions on concurrent requests. |
| **CORS** | Defaults to deny when no origin configured. `Vary: Origin` header set correctly. |
| **Connection Pooling** | Custom HTTP transport with 100 max idle connections, 20 per host. No connection exhaustion. |

## Threat Model

**Designed for:**
- Solo devs or small teams (< 5 people)
- Self-hosted on a single machine
- Trusted network environment (home, personal VPS)
- Non-critical workloads (prototyping, personal projects, hackathons)

**NOT designed for:**
- Public-facing production with untrusted users
- Enterprises needing SOC2/HIPAA compliance
- Multi-tenant SaaS with strict isolation requirements
- Environments where the host machine is untrusted

## Known Limitations

- **SQLite contention** at high concurrency (100+ concurrent requests). It's a feature, not a bug. (It's a bug.)
- **No TLS termination** — use a reverse proxy (nginx, Caddy, Cloudflare Tunnel).
- **No authentication on `/v1/chat/completions`** unless you create virtual keys. Guard the unified key with your life.
- **Admin password visible in `ps` output** — use a reverse proxy with its own auth if this matters.
- **No audit log tamper protection** — if someone gets root, they can modify the SQLite database.

## Audit Status

- **31 vulnerabilities found and fixed** in security audits (7 critical, 6 high, 8 medium, 6 low).
- **SSRF blocklist upgraded** from naive string prefix matching to proper `net.ParseIP()` validation.
- **Brute-force protection** added to admin auth (5 attempts/min lockout).
- **Crypto primitives are standard** — AES-256-GCM, well-studied and used by the Go standard library.
- **No custom crypto** — we use `golang.org/x/crypto`, not hand-rolled anything.
- **If you find a vulnerability**, open a GitHub issue. Please don't tweet about it first.
