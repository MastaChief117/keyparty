<div align="center">

# KeyParty

**One endpoint. Ten providers. Zero budget.**

[![Go Version](https://img.shields.io/badge/go-1.24+-00ADD8?style=flat-square&logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/license-MIT-blue.svg?style=flat-square)](LICENSE)
[![GitHub Stars](https://img.shields.io/github/stars/MastaChief117/keyparty?style=flat-square&color=yellow)](https://github.com/MastaChief117/keyparty)
[![Docker](https://img.shields.io/badge/docker-ready-2496ED?style=flat-square&logo=docker)](https://github.com/MastaChief117/keyparty/pkgs/container/keyparty)

*For the broke dev juggling free tiers at 3AM.*

**[What is this](#what-is-this) | [Quick Start](#quick-start) | [Docker](#docker) | [Features](#features) | [FAQ](#faqs)**

</div>

---

## What is this

You have 5 AI providers, 5 API keys, 5 rate limits, and $4 across all of them combined. Your entire AI budget is less than a gas station coffee.

**KeyParty solves that.** One endpoint, zero config, and it:

- Routes to whichever provider is currently breathing
- Fails over automatically when one dies (your chat survives)
- Rotates your keys like a DJ but for API keys
- Compacts chat history to fit cheaper provider limits
- Encrypts everything with AES-256-GCM
- Blocks prompt injection and PII leaks
- Tracks every penny across all providers
- Lets you race providers, roast each other, and rap battle (because why not)
- Has a setup wizard that doesn't suck

Most API gateways are built for enterprises with Kubernetes clusters. **This one is built for the dev with $4 and questionable life choices.**

---

## Quick Start

### Option 1: Shell Script (easiest)

```bash
git clone https://github.com/MastaChief117/keyparty.git
cd keyparty
chmod +x keyparty.sh
./keyparty.sh
```

The script will:
- Install Go if needed
- Build the binary
- Ask for admin password
- Optionally set up a Cloudflare tunnel
- Start everything

Dashboard: **http://localhost:8080**

### Option 2: Manual Build

```bash
git clone https://github.com/MastaChief117/keyparty.git
cd keyparty
go build -o keyparty .
./keyparty -port 8080 -admin-pass your-password
```

### Option 3: Docker (recommended)

```bash
git clone https://github.com/MastaChief117/keyparty.git
cd keyparty

# Build
docker build -t keyparty .

# Run
docker run -d \
  --name keyparty \
  -p 8080:8080 \
  -e ADMIN_PASSWORD=your-secret-password \
  -v keyparty-data:/app/data \
  keyparty

# Or use docker-compose
echo "ADMIN_PASSWORD=your-secret" > .env
docker-compose up -d
```

Dashboard: **http://localhost:8080**

### First Run Wizard

On first launch, a setup wizard appears:
1. Set admin password (or skip for no auth)
2. Add your first API key (Groq has a free tier!)
3. Test the connection
4. Get your unified API key

### Use in Your Apps

```bash
# Use the unified key as Bearer token
curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer gw-your-unified-key" \
  -d '{"model":"groq/llama-3.3-70b-versatile","messages":[{"role":"user","content":"Hello"}]}'
```

Works with OpenAI SDK, Anthropic SDK, any HTTP client. Just point them at the gateway.

### Provider/Model Override

```bash
# Force a specific provider
curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer YOUR_KEY" \
  -H "X-Gateway-Provider: gemini" \
  -d '{"model":"llama-3.3-70b-versatile","messages":[{"role":"user","content":"Hello"}]}'

# Override model too
curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer YOUR_KEY" \
  -H "X-Gateway-Provider: nvidia" \
  -H "X-Gateway-Model: nvidia/nemotron-3-nano-30b-a3b" \
  -d '{"model":"llama-3.3-70b-versatile","messages":[{"role":"user","content":"Hello"}]}'
```

---

## Docker

### Dockerfile

```bash
docker build -t keyparty .
```

Multi-stage build. Final image is ~20MB on Alpine.

### Docker Compose

```yaml
version: '3.8'
services:
  keyparty:
    build: .
    ports:
      - "8080:8080"
    volumes:
      - keyparty-data:/app/data
    environment:
      - ADMIN_PASSWORD=${ADMIN_PASSWORD:-changeme}
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3

volumes:
  keyparty-data:
```

### Commands

```bash
# Start
docker-compose up -d

# Stop
docker-compose down

# View logs
docker-compose logs -f

# Rebuild after updates
git pull
docker-compose up -d --build

# Reset everything
docker-compose down -v
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `ADMIN_PASSWORD` | (empty) | Admin password for dashboard. Empty = no auth. |

### Volumes

| Mount | Description |
|-------|-------------|
| `/app/data` | SQLite database (persists keys, logs, config) |

---

## Features

### Core (the serious stuff)

| Feature | What it does |
|---------|-------------|
| Multi-Provider Routing | OpenAI, Anthropic, Gemini, Groq, NVIDIA, Together, DeepSeek, OpenRouter, Fireworks, Mistral — one endpoint |
| Round-Robin Rotation | Spreads requests across multiple keys per provider |
| Provider/Model Override | Route to any saved provider via `X-Gateway-Provider` header |
| Compaction Failover | When quota dies (429/402), summarizes chat history and fails over |
| Model-Based Routing | Send `model: "llama-3.3-70b-versatile"` and it auto-matches to the right provider |
| Provider Race Mode | Sends to multiple providers simultaneously, returns the fastest |
| Guardrails | PII detection, prompt injection blocking, custom regex rules |
| Virtual Keys | Consumer-facing keys with budgets, rate limits, and model allowlists |
| Model Aliases | Map `smart` → `claude-sonnet-4-5`, `fast` → `gpt-4o-mini` |
| Encryption at Rest | AES-256-GCM for all API keys |
| Request Logging | Full audit trail with token usage and cost tracking |
| Response Caching | SHA-256 deduplication |
| Admin Dashboard | Web UI with dark mode |
| Setup Wizard | First-run wizard to add your first API key |

### Fun Stuff (the chaotic good)

| Feature | What it does |
|---------|-------------|
| AI Rumble | Two providers roast each other in real-time via SSE |
| Rap Battle | Two AIs drop bars with context memory across rounds |
| Model Roulette | Random provider picks your message |
| AI Therapist | Sarcastic therapy for when your code won't compile |
| Roast My Logs | AI roasts your API usage patterns |
| AI Fortune Teller | Dramatic predictions about your code's fate |
| Provider Roast | Two providers trash-talk each other's weaknesses |
| Debug Oracle | Paste an error, get dramatic debugging advice |
| Commit Message Generator | Dramatic, meme, or poetic commit messages |
| AI Poll | Same prompt to multiple providers, compare answers |

### Pro Tools (the big brain stuff)

| Feature | What it does |
|---------|-------------|
| Smart Router | AI recommends best provider based on cost/speed/quality priority |
| Cost Limiter | Set daily/weekly/monthly spending caps per provider |
| Provider Uptime | Track provider reliability over time with stats |
| Replay Queue | Retry failed requests with one click |
| Custom Models | Add models not in the default list |
| Health Check | Ping all providers and report status in real-time |
| Export Logs | Download logs as CSV or JSON |
| Prompt Builder | Build prompts with variables, optionally enhance with AI |
| VK Usage Dashboard | Per virtual key usage breakdown with cost, latency, errors |
| Auto-Rotate Keys | Pick the healthiest key per provider automatically |
| Chat Playground | Full chat UI with SSE streaming, provider/model picker |
| Weekly Recap | Stats summary + AI-generated snarky reports |
| Cost Analytics | Cost by provider/day/model with CSS bar charts |
| Webhooks | Notify URLs on events (request, error, failover) |
| Prompt Templates | Save and manage system prompts |
| Rate Limit Tiers | Tier-based rate limiting (free/premium) |
| Budget Alerts | Threshold alerts per virtual key |
| Search Logs | Filter by provider/model/status/virtual key |

---

## API Endpoints

### Proxy
| Endpoint | Method | Description |
|----------|--------|-------------|
| `/v1/chat/completions` | POST | OpenAI-compatible chat completions |
| `/v1/models` | GET | List available models |
| `/health` | GET | Health check |

### Admin (require auth)
| Endpoint | Method | Description |
|----------|--------|-------------|
| `/admin/keys` | GET/POST | List or add API keys |
| `/admin/keys/{id}` | DELETE/PUT | Delete or update key |
| `/admin/keys/{id}/toggle` | POST | Enable/disable key |
| `/admin/stats` | GET | Gateway statistics |
| `/admin/logs` | GET | Request logs |
| `/admin/virtual-keys` | GET/POST | Manage virtual keys |
| `/admin/aliases` | GET/POST | Model aliases |
| `/admin/guardrails` | GET/POST | Guardrail rules |
| `/admin/failover` | GET/POST | Failover config |
| `/admin/firewall` | GET/POST | Firewall config |
| `/admin/race` | POST | Race providers |
| `/admin/roast` | POST | Roast AI |
| `/admin/8ball` | POST | Ask the magic 8-ball |
| `/admin/rap-battle` | POST | Rap battle between providers |
| `/admin/roulette` | POST | Model roulette |
| `/admin/therapist` | POST | AI therapist |
| `/admin/vibe-check` | POST | Vibe check |
| `/admin/roast-logs` | POST | Roast your logs |
| `/admin/fortune` | POST | AI fortune teller |
| `/admin/provider-roast` | POST | Provider roast battle |
| `/admin/debug-oracle` | POST | Debug oracle |
| `/admin/commit-msg` | POST | Commit message generator |
| `/admin/smart-route` | POST | Smart router |
| `/admin/cost-limiter` | GET/POST/DELETE | Cost limits |
| `/admin/uptime` | GET | Provider uptime |
| `/admin/replay-queue` | GET/POST | Replay failed requests |
| `/admin/custom-models` | GET/POST/DELETE | Custom models |
| `/admin/health-check` | GET | Ping all providers |
| `/admin/export-logs` | GET | Export logs (CSV/JSON) |
| `/admin/prompt-builder` | POST | Build prompts |

---

## Comparison

| Feature | KeyParty | LiteLLM | OpenRouter |
|---------|----------|---------|------------|
| Self-hosted | ✅ | ✅ | ❌ |
| Free & open source | ✅ | ✅ | ❌ |
| Single binary (no deps) | ✅ | ❌ | N/A |
| Docker support | ✅ | ✅ | N/A |
| Setup wizard | ✅ | ❌ | N/A |
| Virtual Keys | ✅ | ❌ | ❌ |
| AI Rumble / Rap Battle | ✅ | ❌ | ❌ |
| Provider Roast | ✅ | ❌ | ❌ |
| Model Roulette | ✅ | ❌ | ❌ |
| Fortune Teller | ✅ | ❌ | ❌ |
| Debug Oracle | ✅ | ❌ | ❌ |
| Commit Message Gen | ✅ | ❌ | ❌ |
| Smart Router | ✅ | ❌ | ❌ |
| Cost Limiter | ✅ | ❌ | ❌ |
| Provider Uptime | ✅ | ❌ | ❌ |
| Replay Queue | ✅ | ❌ | ❌ |
| Export Logs | ✅ | ❌ | ❌ |
| Prompt Builder | ✅ | ❌ | ❌ |
| Provider Race Mode | ✅ | ❌ | ❌ |
| Compaction Failover | ✅ | ✅ | ✅ |
| Cost Tracking | ✅ | ✅ | ✅ |
| Dashboard | ✅ | ✅ | ✅ |

---

## Security

AES-256-GCM encryption at rest, SSRF protection, brute-force lockout on admin auth, PII detection, prompt injection blocking.

**Not designed for:** public-facing production, SOC2/HIPAA compliance, untrusted hosts.

---

## Performance

- **Idle:** 13MB RAM
- **Under load:** ~33MB RAM (stabilizes, no leaks)
- **Throughput:** ~220 RPS (gateway overhead only)
- **Binary:** 16MB on disk
- **Docker image:** ~20MB
- **Runs on:** a potato 🥔

---

## Who this is for

- Solo devs building AI agents on free tiers
- Small teams sharing API credits
- Hackathon projects needing multi-provider resilience
- Students learning AI/LLM integration
- Anyone tired of managing 5 different AI API dashboards
- People who want to watch two AIs roast each other

## Who this is NOT for

- Enterprises needing horizontal scaling (it's a single binary)
- Teams needing RBAC, SSO, or compliance certifications
- Anyone processing millions of requests (SQLite has limits)
- People who don't like fun

---

## FAQs

**Q: What is this?**
A: An AI gateway for broke devs. One endpoint, ten providers, zero budget.

**Q: Why?**
A: Because my OpenAI quota died at 2am and I was too angry to sleep. This project is revenge against rate limits.

**Q: Does it work?**
A: If you're reading this README then probably. We tested it. We think.

**Q: How much does it cost?**
A: Free. MIT license. No "enterprise tier," no "contact sales." Just vibes.

**Q: Is my data safe?**
A: AES-256-GCM encryption at rest. Standard libraries. No hand-rolled crypto.

**Q: What happens when a provider dies?**
A: It fails over to another one. Your chat survives. We're basically life support for your conversations.

**Q: Can two AIs roast each other?**
A: Yes. Provider Roast and Rap Battle are built in. It's exactly as chaotic as it sounds.

**Q: Can I use this with Claude Code / Cursor?**
A: Yes. Just point them at the gateway.

**Q: Can I run this on a Raspberry Pi?**
A: Yes. It'll run on a potato if it has Go installed.

**Q: How do I set it up?**
A: Run `./keyparty.sh` or use Docker. A wizard guides you through everything.

**Q: What's the catch?**
A: There is no catch. It's free, open source, MIT license. We just vibes.

**Q: Why is it called KeyParty?**
A: Because it's a party for your API keys. They finally get to hang out together instead of being locked in separate provider vaults. It's wholesome, actually.

**Q: Can the AI tell my fortune?**
A: Yes. And it will roast your code while doing it.

**Q: Can I set spending limits?**
A: Yes. Cost Limiter lets you set daily/weekly/monthly caps per provider.

**Q: Can I track provider reliability?**
A: Yes. Provider Uptime shows success rates, latency, and failure counts over time.

---

## Roadmap

- [x] Gateway with multi-provider routing
- [x] Failover with chat compaction
- [x] AES-256-GCM encryption
- [x] Dashboard with dark mode
- [x] Provider race mode
- [x] Guardrails (PII, injection blocking)
- [x] Virtual keys with budgets
- [x] Model aliases
- [x] Request logging & cost tracking
- [x] AI Rumble, Rap Battle, Model Roulette
- [x] Webhooks, Prompt Templates, AI Poll
- [x] Rate Limit Tiers & Budget Alerts
- [x] VK Usage Dashboard, Auto-Rotate Keys
- [x] Chat Playground, Weekly Recaps, Cost Analytics
- [x] Full security audit (31 vulnerabilities fixed)
- [x] Docker support (multi-stage build, docker-compose)
- [x] Setup wizard for first-run experience
- [x] Fortune Teller, Provider Roast, Debug Oracle, Commit Message Generator
- [x] Smart Router, Cost Limiter, Provider Uptime
- [x] Replay Queue, Custom Models, Health Check
- [x] Export Logs (CSV/JSON), Prompt Builder
- [ ] MCP support
- [ ] Provider speed tests
- [ ] Take over the AI gateway market (one $4 coffee at a time)

---

<div align="center">

**[Star this repo](https://github.com/MastaChief117/keyparty)** or don't. I'll add features anyway because I have no self-control.

*Made by a dude who should've been sleeping. Again.*

</div>
