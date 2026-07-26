<div align="center">

# 🔀 AI Gateway

**One API to rule them all. Route LLM requests across 10 providers like a boss.**

*Failover when quota dies. Encrypt your keys. Race providers. Guard your prompts. All from one dashboard.*

**[Docs](#wtf-is-this) • [Quick Start](#quick-start-in-60-seconds) • [Features](#the-stuff) • [API](#api-nonsense)**

*Bugs Fixed: 3 • Features Added: 12 • Sleep Deprived: Absolutely*

</div>

---

## WTF is this

So basically you have API keys for like 5 different AI providers and you're tired of:
- OpenAI rate limiting you at 2am when you're vibing
- Anthropic eating your credits like pacman
- Having to change your code every time you want to switch providers
- Your API keys sitting in plain text like a clown

**This gateway solves all of that.** You point ONE endpoint at it, and it:
- Routes to whichever provider is available
- Rotates across your keys automatically
- Fails over to another provider when one dies (with your chat history intact)
- Encrypts everything at rest so nobody can steal your keys
- Blocks prompt injection and PII from leaking
- Lets you race providers against each other for the fastest response

It's like a load balancer but for AI. And it has a dashboard. Because everything needs a dashboard.

## Quick Start (in 60 seconds)

```bash
# Clone the thing
git clone https://github.com/MastaChief117/ai-gateway.git
cd ai-gateway

# Build it (needs Go 1.24+)
go build -o ai-gateway .

# Run it
./ai-gateway -port 8080 -admin-pass your-password-here
```

Dashboard: **http://localhost:8080**

That's it. No Docker. No Kubernetes. No npm install. Just a single binary and vibes.

## The Stuff

| Feature | What it does |
|---------|-------------|
| 🔀 **Multi-Provider Routing** | OpenAI, Anthropic, Gemini, Groq, NVIDIA NIM, Together, DeepSeek, OpenRouter, Fireworks, Mistral — one endpoint |
| 🔄 **Round-Robin Rotation** | Spreads requests across multiple keys per provider |
| 💥 **Compaction Failover** | When quota dies (402), summarizes your chat history and sends it to a backup provider. Nothing is lost. |
| ⚡ **Provider Race Mode** | Sends your message to multiple providers simultaneously, returns the fastest response |
| 🛡️ **Guardrails** | PII detection, prompt injection blocking, custom regex rules |
| 🔑 **Virtual Keys** | Give users their own API keys with budgets, rate limits, and model allowlists |
| 🏷️ **Model Aliases** | Map `smart` → `claude-sonnet-4-5`, `fast` → `gpt-4o-mini` |
| 🔒 **Encryption at Rest** | AES-256-GCM encryption for all API keys. Your keys are safe. |
| 📊 **Request Logging** | Full audit trail with token usage and cost tracking |
| 💰 **Cost Tracking** | Per-key and per-provider cost estimation |
| 📦 **Response Caching** | SHA-256 deduplication. Same request? Cached. |
| 🎛️ **Admin Dashboard** | Web UI with tabs for everything. Dark mode because light mode is a war crime. |

## The Failover Thing (it's cool ok)

Here's how compaction failover works:

1. You send a request to OpenAI
2. OpenAI says "lol you're out of credits" (HTTP 402)
3. Gateway goes "no worries bro" and summarizes your entire chat history
4. Sends that summary to Groq (or whatever backup you configured)
5. You get a response like nothing happened
6. Response has `X-Gateway-Failover: true` header so you know it happened

**Your conversation survives.** Even if the provider dies. That's the whole point.

Configure it in the dashboard or via API:

```bash
# Enable failover
curl -X POST http://localhost:8080/admin/failover \
  -H "Authorization: Bearer admin-password" \
  -H "Content-Type: application/json" \
  -d '{"key":"enabled","value":"true"}'

# Pick your backup provider
curl -X POST http://localhost:8080/admin/failover \
  -H "Authorization: Bearer admin-password" \
  -H "Content-Type: application/json" \
  -d '{"key":"provider","value":"groq"}'

curl -X POST http://localhost:8080/admin/failover \
  -H "Authorization: Bearer admin-password" \
  -H "Content-Type: application/json" \
  -d '{"key":"model","value":"llama-3.3-70b-versatile"}'
```

## API Nonsense

### Chat Completions (OpenAI-compatible, obviously)

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer YOUR_UNIFIED_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o",
    "messages": [{"role": "user", "content": "say hi"}]
  }'
```

### Provider Race Mode (who's fastest?)

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

### All Admin Endpoints

| Method | Endpoint | What it does |
|--------|----------|-------------|
| GET | `/admin/unified-key` | Get your unified API key |
| POST | `/admin/unified-key` | Regenerate it (update all your apps!) |
| GET/POST | `/admin/keys` | List / Add provider API keys |
| DELETE | `/admin/keys/:id` | Delete a key |
| POST | `/admin/keys/:id/toggle` | Enable/disable a key |
| GET/POST | `/admin/failover` | Failover config |
| GET | `/admin/failover/logs` | See when failover happened |
| GET | `/admin/virtual-keys` | Consumer-facing keys |
| POST | `/admin/virtual-keys` | Create virtual key |
| GET | `/admin/guardrails` | List guardrails |
| POST | `/admin/guardrails` | Add guardrail |
| GET | `/admin/aliases` | Model aliases |
| POST | `/admin/aliases` | Add alias |
| GET | `/admin/logs` | Request logs |
| GET | `/admin/stats` | Dashboard stats |
| POST | `/admin/race` | Provider race |

All admin endpoints need `Authorization: Bearer <your-admin-password>`

## Config Stuff

### CLI Flags (because env vars are for cowards)

| Flag | Default | What it does |
|------|---------|-------------|
| `-port` | `8080` | Server port |
| `-admin-pass` | (required) | Admin dashboard password |
| `-db` | `gateway.db` | SQLite database path |
| `-cache-ttl` | `60` | Cache TTL in minutes |
| `-cors-origin` | (empty) | Allowed CORS origin |

### Environment Variables (for the cowards)

| Variable | What it does |
|----------|-------------|
| `ADMIN_PASSWORD` | Admin password (alternative to `-admin-pass`) |

## Security (we take this seriously)

- 🔐 **API Key Encryption** — All keys encrypted at rest with AES-256-GCM
- 🔑 **Key File** — `.gateway.key` with `0600` permissions (only root/gateway can read)
- 🛡️ **Guardrails** — PII detection and prompt injection blocking
- 🚫 **SSRF Protection** — Blocks requests to localhost, private IPs, metadata endpoints

## Tech Stuff

- **Language:** Go 1.24+ (single binary, no deps, 32MB base memory)
- **Database:** SQLite (it just works)
- **Encryption:** AES-256-GCM (military grade or whatever)
- **Frontend:** Vanilla HTML/CSS/JS (no React, no Vue, no suffering)
- **Tunnel:** Cloudflare Quick Tunnel support (for when you're too lazy to set up nginx)

## Roadmap

- [x] Make gateway
- [x] Add failover with compaction
- [x] Add encryption
- [x] Make dashboard
- [x] Add race mode
- [x] Regret nothing
- [ ] Add semantic caching (like the fancy projects do)
- [ ] Add MCP support
- [ ] Add Webhooks
- [ ] Take over the AI gateway market
- [ ] Buy a real domain
- [ ] Retire

---

<div align="center">

**If you star this repo I will personally thank you**

**[AI Gateway](https://github.com/MastaChief117/ai-gateway)** ← Click here to be cool

*Made by a dude who should've been sleeping*
*Last updated: Whenever I remember*

</div>
