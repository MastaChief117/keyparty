# AI Gateway

A production-grade, self-hosted AI gateway written in Go. One OpenAI-compatible API endpoint for multiple LLM providers with intelligent failover, context compaction, encryption at rest, and a built-in admin dashboard.

## Features

- **Multi-Provider Routing** — Route to OpenAI, Anthropic, Gemini, Groq, NVIDIA NIM, Together, DeepSeek, OpenRouter, Fireworks, and Mistral through a single endpoint
- **Round-Robin Rotation** — Automatically distribute requests across multiple API keys per provider
- **Compaction Failover** — When a provider's quota expires (402), chat history is summarized and forwarded to a backup provider so nothing is lost
- **Response Caching** — SHA-256 based request deduplication with configurable TTL
- **Guardrails** — PII detection (email, phone, SSN, credit cards), prompt injection blocking, and custom regex rules
- **Virtual Keys** — Consumer-facing API keys with per-key budgets, rate limits, and model allowlists
- **Model Aliases** — Map friendly names to real model IDs (e.g. `smart` → `claude-sonnet-4-5`)
- **Provider Race Mode** — Send requests to multiple providers simultaneously, return the fastest response
- **Encryption at Rest** — AES-256-GCM encryption for all API keys with a machine-specific key file
- **Request Logging** — Full audit trail of all requests with token usage and cost tracking
- **Cost Tracking** — Per-key and per-provider cost estimation
- **Admin Dashboard** — Web-based UI with tabs for API Keys, Virtual Keys, Guardrails, Aliases, Failover, and Request Logs

## Quick Start

```bash
# Clone the repo
git clone https://github.com/MastaChief117/ai-gateway.git
cd ai-gateway

# Build
go build -o ai-gateway .

# Run
./ai-gateway -port 8080 -admin-pass your-password
```

Dashboard available at `http://localhost:8080`

## Configuration

### CLI Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-port` | `8080` | Server port |
| `-admin-pass` | (required) | Admin dashboard password |
| `-db` | `gateway.db` | SQLite database path |
| `-cache-ttl` | `60` | Cache TTL in minutes |
| `-cors-origin` | (empty) | Allowed CORS origin |

### Environment Variables

| Variable | Description |
|----------|-------------|
| `ADMIN_PASSWORD` | Admin password (alternative to `-admin-pass`) |

## API

### Chat Completions (OpenAI-compatible)

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer YOUR_UNIFIED_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o",
    "messages": [{"role": "user", "content": "Hello"}]
  }'
```

### Provider Race Mode

```bash
curl http://localhost:8080/admin/race \
  -H "Authorization: Bearer YOUR_UNIFIED_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Hello, which model are you?",
    "providers": "groq,nvidia",
    "model": "llama-3.3-70b-versatile"
  }'
```

### Admin Endpoints

All admin endpoints require `Authorization: Bearer <admin-password>`.

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/admin/unified-key` | Get unified API key |
| POST | `/admin/unified-key` | Regenerate unified API key |
| GET/POST | `/admin/keys` | List / Add API keys |
| DELETE | `/admin/keys/:id` | Delete API key |
| POST | `/admin/keys/:id/toggle` | Enable/disable key |
| GET/POST | `/admin/failover` | Get / Set failover config |
| GET | `/admin/failover/logs` | Failover event logs |
| GET | `/admin/virtual-keys` | List virtual keys |
| GET | `/admin/guardrails` | List guardrails |
| GET | `/admin/aliases` | List model aliases |
| GET | `/admin/logs` | Request logs |
| GET | `/admin/stats` | Dashboard statistics |

## Failover & Compaction

The gateway supports automatic failover when a provider returns a 402 (quota exhausted) or other configured error codes. Optionally, chat history can be compacted (summarized) before forwarding to the backup provider.

**How it works:**
1. Request is sent to the primary provider
2. If the provider returns 402 (or other trigger code), the gateway intercepts the error
3. If compaction is enabled, the full chat history is summarized via a cheap model
4. The compacted (or full) history is sent to the configured failover provider
5. The response is returned to the client with `X-Gateway-Failover: true` header

Configure via the dashboard's **Failover** tab or via the API:

```bash
# Enable failover
curl -X POST http://localhost:8080/admin/failover \
  -H "Authorization: Bearer admin-password" \
  -H "Content-Type: application/json" \
  -d '{"key":"enabled","value":"true"}'

# Set failover target
curl -X POST http://localhost:8080/admin/failover \
  -H "Authorization: Bearer admin-password" \
  -H "Content-Type: application/json" \
  -d '{"key":"provider","value":"groq"}'

curl -X POST http://localhost:8080/admin/failover \
  -H "Authorization: Bearer admin-password" \
  -H "Content-Type: application/json" \
  -d '{"key":"model","value":"llama-3.3-70b-versatile"}'
```

## Security

- **API Key Encryption** — All provider API keys and the unified key are encrypted at rest using AES-256-GCM
- **Encryption Key** — Stored in `.gateway.key` with `0600` permissions (only the gateway process owner can read)
- **Guardrails** — PII detection and prompt injection blocking built-in
- **SSRF Protection** — Blocks requests to localhost, private IPs, and metadata endpoints

## Tech Stack

- **Language:** Go 1.24+
- **Database:** SQLite (via `go-sqlite`)
- **Encryption:** AES-256-GCM
- **Frontend:** Vanilla HTML/CSS/JS (embedded via `go:embed`)
- **Tunnel:** Cloudflare Quick Tunnel support

## License

MIT
