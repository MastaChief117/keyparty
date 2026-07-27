<div align="center">

# 🔀 KeyParty

**One endpoint. Ten providers. Zero budget.**

*For the broke dev juggling free tiers at 3AM, praying one of them doesn't rate-limit them mid-demo.*

**[What is this](#what-is-this) • [Quick Start](#quick-start-in-60-seconds) • [The Stuff](#the-stuff) • [FAQ (it's funny)](#faqs)**

*Features: 28 • Sleep Deprived: Immensely • Free Tier Credits: Scattered Across 5 Providers*

</div>

---

## What is this

You're building an AI agent. You sign up for free tiers on Groq, Nvidia, DeepSeek, maybe grab an OpenAI trial. Now you've got:

- 5 different API endpoints
- 5 different auth methods
- 5 different rate limits
- 5 different ways to get screwed when one goes down

And you have $4 across all of them.

**This gateway solves all of that.** You point ONE endpoint at it, and it:

- Routes to whichever provider is breathing
- Rotates across your keys automatically (like a DJ but for API keys)
- Fails over to another provider when one dies (your chat history survives because we're not monsters)
- **Compacts your chat history** to fit within cheaper provider limits when quota dies
- Encrypts everything at rest so nobody can steal your keys
- Blocks prompt injection and PII from leaking
- Tracks every penny you spend across all providers
- Lets you race providers, roast each other, rap battle, and more

It's like a load balancer but for AI. And it has a dashboard. Because everything needs a dashboard.

Most API gateways are built for enterprises with Kubernetes clusters and DevOps teams. **This one is built for the dev who has $4 and a dream.**

---

## Quick Start (in 60 seconds)

```bash
# Clone the thing
git clone https://github.com/MastaChief117/keyparty.git
cd keyparty

# Build it (needs Go 1.24+)
go build -o keyparty .

# Run it
./keyparty -port 8080 -admin-pass your-password-here
```

Dashboard: **http://localhost:8080**

That's it. No Docker. No Kubernetes. No npm install. No node_modules folder the size of Texas. Just a single binary and vibes.

---

## The Stuff

### Core Features

| Feature | What it does | Why you care |
|---------|-------------|-------------|
| 🔀 **Multi-Provider Routing** | OpenAI, Anthropic, Gemini, Groq, NVIDIA, Together, DeepSeek, OpenRouter, Fireworks, Mistral — one endpoint | Stop changing your code when a provider dies |
| 🔄 **Round-Robin Rotation** | Spreads requests across multiple keys per provider | Your 3 Groq free tier keys work as one |
| 💥 **Compaction Failover** | When quota dies (402), summarizes your chat history and sends it to a backup provider | Your $4 just got you through a demo |
| ⚡ **Provider Race Mode** | Sends your message to multiple providers simultaneously, returns the fastest | Find out which free tier is actually fastest |
| 🛡️ **Guardrails** | PII detection, prompt injection blocking, custom regex rules | Your SSN stays home |
| 🔑 **Virtual Keys** | Give users their own API keys with budgets, rate limits, and model allowlists | Share without sharing your real keys |
| 🏷️ **Model Aliases** | Map `smart` → `claude-sonnet-4-5`, `fast` → `gpt-4o-mini` | Sound smart without knowing model names |
| 🔒 **Encryption at Rest** | AES-256-GCM encryption for all API keys | Your keys are safe. We promise. |
| 📊 **Request Logging** | Full audit trail with token usage and cost tracking | Know where your money goes |
| 💰 **Cost Tracking** | Per-key and per-provider cost estimation | Your wallet will thank you |
| 📦 **Response Caching** | SHA-256 deduplication. Same request? Cached. | Your rate limits thank you |
| 🎛️ **Admin Dashboard** | Web UI with tabs for everything. Dark mode. | Because light mode is a war crime |

### Fun Features

| Feature | What it does | Vibe |
|---------|-------------|------|
| 🥊 **AI Rumble** | Two providers roast each other in real-time via SSE streaming | Boxing match but for LLMs |
| 🎤 **Rap Battle** | Two AIs drop bars with context memory across rounds | 8 Mile but make it AI |
| 🎰 **Model Roulette** | Random provider picks your message | Surprise me mode |
| 🌊 **Vibe Check** | Send a message, get a random vibe rating | Important metrics |
| 🧠 **AI Therapist** | Sarcastic therapy for when your code won't compile | We've all been there |
| 🔥 **Roast My Logs** | AI roasts your API usage patterns | Painful but funny |
| 🗳️ **AI Poll** | Same prompt to multiple providers, compare answers | Democratic AI |

### Pro Tools

| Feature | What it does | Power level |
|---------|-------------|-------------|
| 📊 **VK Usage Dashboard** | Per virtual key usage breakdown with cost, latency, errors | Know who's spending what |
| 🔄 **Auto-Rotate Keys** | Pick the healthiest key per provider automatically | Never hit a dead key again |
| 🧪 **Chat Playground** | Full chat UI with SSE streaming, provider/model picker | Test prompts without leaving the dashboard |
| 📋 **Weekly Recap** | Stats summary + AI-generated snarky reports | Your usage, but make it funny |
| 📈 **Cost Analytics** | Cost by provider/day/model with CSS bar charts | See where your credits go |
| 🪝 **Webhooks** | Notify URLs on events (request, error, failover) | Get alerted when things break |
| 📝 **Prompt Templates** | Save and manage system prompts | Reuse without copy-pasting |
| ⏱️ **Rate Limit Tiers** | Tier-based rate limiting management (free/premium) | Give friends different access levels |
| 💰 **Budget Alerts** | Threshold alerts per virtual key | Don't let anyone burn the group's credits |
| 🔍 **Search Logs** | Filter by provider/model/status/virtual key | Find that one request that broke everything |

---

## The Failover Thing (it's cool ok)

Here's how compaction failover works:

1. You send a request to OpenAI
2. OpenAI says "lol you're out of credits" (HTTP 402)
3. Gateway goes "no worries bro" and summarizes your entire chat history
4. Sends that summary to Groq (or whatever backup you configured)
5. You get a response like nothing happened
6. Response has `X-Gateway-Failover: true` header so you know it happened
7. You feel like a genius for setting this up

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

---

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

### Provider Race Mode (who's fastest? let them fight)

```bash
curl http://localhost:8080/admin/race \
  -H "Authorization: Bearer YOUR_ADMIN_PASSWORD" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Hello, which model are you?",
    "providers": ["groq", "nvidia"]
  }'
```

### AI Rumble (roast battle)

```bash
curl http://localhost:8080/admin/rumble \
  -H "Authorization: Bearer YOUR_ADMIN_PASSWORD" \
  -H "Content-Type: application/json" \
  -d '{"rounds": 5}' | stream
```

### Model Roulette (surprise me)

```bash
curl http://localhost:8080/admin/roulette \
  -H "Authorization: Bearer YOUR_ADMIN_PASSWORD" \
  -H "Content-Type: application/json" \
  -d '{"message": "Say something chaotic"}'
```

### All Admin Endpoints

| Method | Endpoint | What it does |
|--------|----------|-------------|
| GET | `/admin/unified-key` | Get your unified API key |
| POST | `/admin/unified-key` | Regenerate it |
| GET/POST | `/admin/keys` | List / Add provider API keys |
| DELETE | `/admin/keys/:id` | Delete a key |
| POST | `/admin/keys/:id/toggle` | Enable/disable a key |
| GET/POST | `/admin/failover` | Failover config |
| GET | `/admin/failover/logs` | See when failover happened |
| GET/POST | `/admin/virtual-keys` | Consumer-facing keys |
| GET/POST | `/admin/guardrails` | Guardrail rules |
| GET/POST | `/admin/aliases` | Model aliases |
| GET | `/admin/logs` | Request logs |
| GET | `/admin/stats` | Dashboard stats |
| POST | `/admin/race` | Provider race |
| POST | `/admin/rumble` | AI Rumble (SSE) |
| POST | `/admin/rap-battle` | Rap Battle (SSE) |
| POST | `/admin/roulette` | Model Roulette |
| POST | `/admin/roast-logs` | Roast my logs |
| POST | `/admin/therapist` | AI Therapist |
| POST | `/admin/vibe-check` | Vibe Check |
| POST | `/admin/poll` | AI Poll |
| GET/POST | `/admin/webhooks` | Webhook management |
| POST | `/admin/webhooks/test` | Test webhooks |
| GET | `/admin/logs/search` | Search logs with filters |
| GET/POST | `/admin/templates` | Prompt templates |
| GET/POST | `/admin/rate-tiers` | Rate limit tiers |
| GET/POST | `/admin/budget-alerts` | Budget alerts |
| GET | `/admin/budget-alerts/check` | Check triggered alerts |
| GET | `/admin/vk-usage` | Virtual key usage |
| POST | `/admin/auto-rotate` | Pick healthiest key |
| GET | `/admin/auto-rotate/status` | Provider health status |
| POST | `/admin/playground` | Chat playground |
| GET | `/admin/recap` | Weekly recap data |
| POST | `/admin/recap/generate` | AI-generated recap |
| GET | `/admin/analytics` | Cost analytics |

---

## Config Stuff

### CLI Flags

| Flag | Default | What it does |
|------|---------|-------------|
| `-port` | `8080` | Server port |
| `-admin-pass` | (required) | Admin dashboard password |
| `-db` | `gateway.db` | SQLite database path |
| `-cache-ttl` | `60` | Cache TTL in minutes |
| `-cors-origin` | (empty) | Allowed CORS origin |

### Environment Variables

| Variable | What it does |
|----------|-------------|
| `ADMIN_PASSWORD` | Admin password (alternative to `-admin-pass`) |

---

## Security

We take this seriously — not "trust me bro" seriously, actually seriously. Here's what's real:

### What we do

| Protection | Implementation |
|-----------|----------------|
| 🔐 **Encryption at Rest** | AES-256-GCM via `golang.org/x/crypto/chacha20poly1305`. All API keys encrypted in SQLite. Decrypted only in memory during request handling. |
| 🔑 **Key File** | `.gateway.key` stored with `0600` permissions. Key is a 256-bit random value generated on first run. Lost key = encrypted data is gone forever. |
| 🛡️ **Guardrails** | PII detection (SSN, email, phone), prompt injection blocking via regex patterns, custom rules configurable per-deployment. |
| 🚫 **SSRF Protection** | Blocks requests to localhost, private IPs, link-local, and cloud metadata endpoints (169.254.169.254). |
| 🔒 **Admin Auth** | Bearer token auth on all `/admin/*` endpoints. Password passed via CLI flag or env var. |
| 🕐 **Rate Limiting** | Per-virtual-key rate limits with configurable windows. |
| 📊 **Request Logging** | Full audit trail — who requested what, when, which provider, cost, latency. |

### Threat Model

**This gateway is designed for:**
- Solo devs or small teams (< 5 people)
- Self-hosted on a single machine
- Trusted network environment (home, personal VPS)
- Non-critical workloads (prototyping, personal projects, hackathons)

**This gateway is NOT designed for:**
- Public-facing production with untrusted users
- Enterprises needing SOC2/HIPAA compliance
- Multi-tenant SaaS with strict isolation requirements
- Environments where the host machine is untrusted

**Known limitations:**
- SQLite is not designed for high-concurrency writes. If you hit 100+ concurrent requests, you'll see contention.
- No TLS termination — use a reverse proxy (nginx, Caddy, Cloudflare Tunnel) for HTTPS.
- No authentication on the `/v1/chat/completions` endpoint unless you create virtual keys. Anyone with the unified key can use it.
- The admin password is passed as a CLI argument or env var. It's visible in `ps` output. Use a reverse proxy with its own auth if this matters.
- No audit log tamper protection. If someone gets root, they can modify the SQLite database.

### Audit Status

- **No formal security audit has been performed.** This is a solo developer project.
- **Crypto primitives are standard** — AES-256-GCM and ChaCha20-Poly1305 are well-studied and used by the Go standard library.
- **No custom crypto** — we use `golang.org/x/crypto`, not hand-rolled anything.
- **If you find a vulnerability**, open a GitHub issue or email the maintainer. Please don't tweet about it first.

---

## Operational Requirements

"But it's just a binary, right?" Yeah. And so is a nuclear warhead. Here's what you need to actually run this thing:

### Minimum Requirements

- **OS:** Linux, macOS, Windows (any)
- **RAM:** 13MB idle, ~33MB under load
- **Disk:** ~50MB for binary + database grows with usage (~1KB per request logged)
- **Go:** 1.24+ (only for building, not running)

### What You Need to Manage

| Task | Frequency | Difficulty |
|------|-----------|-----------|
| **Key rotation** | When a provider revokes/expires a key | Manual (dashboard or API) |
| **Database backups** | Weekly if you care about logs | `cp gateway.db gateway.db.bak` |
| **Monitoring** | Ongoing | Check `/admin/stats` or set up webhooks |
| **Log rotation** | Monthly | The request log grows. Delete old entries via API or SQLite. |
| **Updates** | When we ship new features | `git pull && go build` |
| **Provider management** | When free tiers change | Update keys, adjust failover config |

### Production Tips

1. **Put it behind a reverse proxy.** Nginx, Caddy, or Cloudflare Tunnel. Don't expose port 8080 directly.
2. **Use HTTPS.** The gateway doesn't do TLS. Your reverse proxy does.
3. **Set up webhooks** for error alerts so you know when providers die.
4. **Back up `.gateway.key` somewhere safe.** Lose it = lose all encrypted keys. There is no recovery.
5. **Don't run as root** if you can avoid it. The gateway doesn't need elevated privileges.
6. **Use virtual keys** instead of sharing your unified key. You can revoke them individually.

### What "Production" Means Here

This is a **self-hosted tool for individuals and small teams.** If you're deploying this for >10 users or handling sensitive data, you should:

- Conduct your own security review
- Set up proper monitoring and alerting
- Use a managed database if SQLite doesn't fit your concurrency needs
- Add TLS termination via reverse proxy
- Consider whether you need compliance certifications

**We're honest about what this is: a great tool for personal use and small teams.** If you need enterprise-grade everything, there are commercial alternatives. We're not trying to be them.

---

## Tech Stuff

- **Language:** Go 1.24+ (single binary, no deps, 13MB idle RAM. it's smol.)
- **Database:** SQLite (it just works. no config needed.)
- **Encryption:** AES-256-GCM (military grade or whatever)
- **Frontend:** Vanilla HTML/CSS/JS (no React, no Vue, no suffering. just vibes.)
- **Tunnel:** Cloudflare Quick Tunnel support (for when you're too lazy to set up nginx)

---

## Performance (we actually tested it)

We stress tested it — **16,000 requests, zero provider calls, zero credits burned.** Requests fail at the gateway level when no keys are configured.

### Binary

- **On disk:** 16MB
- **Threads:** 5

### RAM Usage

| Stage | RSS (actual RAM) | Virtual Size |
|-------|-----------------|--------------|
| Fresh start (idle) | **13MB** | 1.9GB |
| After 1,000 requests | **30MB** | 2.4GB |
| After 6,000 requests | **33MB** | 2.8GB |
| After 16,000 requests | **33MB** | 3.2GB |

> RSS = what your OS actually allocates. Virtual size includes Go runtime mappings (not real memory). Memory stabilizes at ~33MB — no leaks.

### Throughput

| Concurrent | Requests | RPS |
|-----------|----------|-----|
| 50 | 1,000 | **218** |
| 100 | 5,000 | **234** |
| 100 | 10,000 | **222** |

Consistent ~220 RPS with fast-failing requests. With real providers, RPS depends on provider latency — the gateway adds negligible overhead.

### Database

- Request logging writes ~100 bytes per entry
- DB stays at 88KB when requests fail before logging (no provider calls = no logs)
- With real traffic, expect ~100KB per 1,000 logged requests

### TL;DR

- **Idle:** 13MB RAM — lighter than most Electron apps
- **Under load:** ~33MB RAM, stabilizes — no memory leaks
- **Throughput:** ~220 RPS (gateway overhead only)
- Runs on a potato 🥔

---

## Who this is for

- Solo devs building AI agents on free tiers
- Small teams sharing API credits
- Hackathon projects that need multi-provider resilience
- Students learning AI/LLM integration
- Anyone tired of managing 5 different AI API dashboards
- People who think infrastructure can be fun
- The broke dev with $4 across three providers

## Who this is NOT for

- Enterprises needing horizontal scaling (it's a single binary)
- Teams needing RBAC, SSO, or audit logs
- Anyone processing millions of requests (SQLite has limits)
- People who want a plugin system (no Lua/Go extensions yet)
- gRPC-only setups (HTTP only)

---

## FAQs

**Q: What is this?**
A: An AI gateway for broke devs. One endpoint, ten providers, zero budget.

**Q: Why?**
A: Because my OpenAI quota died at 2am and I was too angry to sleep

**Q: Does it work?**
A: If you're reading this README then yeah probably.

**Q: How much does it cost?**
A: If you give me $5 I'll call it enterprise pricing

**Q: Is my data safe?**
A: AES-256-GCM encryption at rest. No custom crypto. Standard libraries. No formal audit yet. Read the [Security section](#security) for the full threat model. We're honest about what this is.

**Q: Can I use it with my existing apps?**
A: If your apps talk to OpenAI then yeah. It's a drop-in replacement.

**Q: What happens when a provider dies?**
A: It fails over to another one. Your chat doesn't die. You're welcome.

**Q: Can I race providers against each other?**
A: Yes. It's basically an AI thunderdome.

**Q: Can two AIs roast each other?**
A: Yes. AI Rumble and Rap Battle are built in. It's exactly as chaotic as it sounds.

**Q: Does it support streaming?**
A: Yes. The stream goes brrr just like it does with the original provider.

**Q: Can I set up rate limits?**
A: Yes. Virtual keys support rate limits and tiers. You can limit your friends. Or your enemies.

**Q: Can I track how much I'm spending?**
A: Yes. Cost analytics, per-provider breakdowns, per-virtual-key usage, budget alerts. Your wallet will thank you.

**Q: Can I use this with Claude Code / Cursor?**
A: Yes. Just point them at the gateway. You're welcome.

**Q: Can I run this on a Raspberry Pi?**
A: Yes. It'll run on a potato if it has Go installed. Don't test me.

**Q: What's the catch?**
A: There is no catch. It's free. Open source. MIT license. We just vibes.

**Q: What if I find a bug?**
A: Open an issue. I'll fix it. Probably. Eventually. No promises.

---

## Roadmap

- [x] Make gateway
- [x] Add failover with compaction
- [x] Add encryption
- [x] Make dashboard
- [x] Add race mode
- [x] Add guardrails
- [x] Add virtual keys
- [x] Add model aliases
- [x] Add request logging
- [x] Add cost tracking
- [x] Add AI Rumble (roast battle)
- [x] Add Rap Battle (SSE streaming)
- [x] Add Model Roulette, Vibe Check, Therapist, Roast Logs
- [x] Add Webhooks
- [x] Add Prompt Templates
- [x] Add AI Poll
- [x] Add Rate Limit Tiers
- [x] Add Budget Alerts
- [x] Add VK Usage Dashboard
- [x] Add Auto-Rotate Keys
- [x] Add Chat Playground
- [x] Add Weekly Recaps
- [x] Add Cost Analytics
- [ ] Add semantic caching (when I'm bored again)
- [ ] Add MCP support (because everyone wants MCP now)
- [ ] Add A/B testing (for the data nerds)
- [ ] Add streaming support for Rumble/Rap Battle (double SSE)
- [ ] Take over the AI gateway market
- [ ] Buy a real domain
- [ ] Hire someone (it's just me and the void)
- [ ] Retire at 25
- [ ] Actually fix the bugs instead of adding features

---

<div align="center">

**If you star this repo I'll add a feature**

**If you don't star this repo I'll still add features because I have no self control**

**[KeyParty](https://github.com/MastaChief117/keyparty)** ← Click here to join the chaos

*Made by a dude who should've been sleeping*
*Last updated: Whenever I remember*
*P.S. If you read this far you're legally obligated to star*

</div>
