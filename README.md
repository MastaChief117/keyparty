<div align="center">

# KeyParty

**One endpoint. Ten providers. Zero budget.**

[![Go Version](https://img.shields.io/badge/go-1.24+-00ADD8?style=flat-square&logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/license-MIT-blue.svg?style=flat-square)](LICENSE)
[![GitHub Stars](https://img.shields.io/github/stars/MastaChief117/keyparty?style=flat-square&color=yellow)](https://github.com/MastaChief117/keyparty)
[![Docker](https://img.shields.io/badge/docker-ready-2496ED?style=flat-square&logo=docker)](https://hub.docker.com/r/steelquill69/keyparty)
[![npm](https://img.shields.io/badge/npm-@steelquill%2Fkeyparty-CB3837?style=flat-square&logo=npm)](https://www.npmjs.com/package/@steelquill/keyparty)

*For the broke dev juggling free tiers at 3AM.*

**[What is this](#what-is-this) | [Install](#install) | [Quick Start](#quick-start) | [Features](#features) | [API](#api-endpoints) | [Docker](#docker) | [FAQ](#faqs)**

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

## Install

### npm (easiest)

```bash
npm install -g @steelquill/keyparty
keyparty
```

The npm package handles everything:
- Downloads the correct binary for your platform (Linux/macOS/Windows, amd64/arm64)
- Interactive setup wizard on first run
- Auto-installs cloudflared for tunnel support
- Updates and manages the binary for you

### Prebuilt Binaries

Download the latest release for your platform from [GitHub Releases](https://github.com/MastaChief117/keyparty/releases):

| Platform | Architecture | File |
|----------|-------------|------|
| Linux | amd64 | `keyparty-linux-amd64` |
| Linux | arm64 | `keyparty-linux-arm64` |
| macOS | amd64 (Intel) | `keyparty-darwin-amd64` |
| macOS | arm64 (Apple Silicon) | `keyparty-darwin-arm64` |
| Windows | amd64 | `keyparty-windows-amd64.exe` |

```bash
# Linux/macOS
chmod +x keyparty-linux-amd64
./keyparty-linux-amd64 -port 8080

# Windows
keyparty-windows-amd64.exe -port 8080
```

### Docker Hub

```bash
# Pull and run
docker run -d \
  --name keyparty \
  -p 8080:8080 \
  -e ADMIN_PASSWORD=your-secret \
  -v keyparty-data:/app/data \
  steelquill69/keyparty:latest

# Or with a specific version
docker run -d -p 8080:8080 steelquill69/keyparty:1.1.0
```

### Docker (build from source)

```bash
git clone https://github.com/MastaChief117/keyparty.git
cd keyparty
docker build -t keyparty .
docker run -d -p 8080:8080 keyparty
```

### Docker Compose

```yaml
services:
  keyparty:
    image: steelquill69/keyparty:latest
    ports:
      - "8080:8080"
    volumes:
      - keyparty-data:/app/data
    environment:
      - ADMIN_PASSWORD=${ADMIN_PASSWORD:-changeme}
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 10s

volumes:
  keyparty-data:
```

```bash
echo "ADMIN_PASSWORD=your-secret" > .env
docker-compose up -d
```

### Shell Script

```bash
git clone https://github.com/MastaChief117/keyparty.git
cd keyparty
chmod +x keyparty.sh
./keyparty.sh
```

The script:
- Installs Go if not present
- Builds the binary
- Runs the setup wizard
- Optionally sets up a Cloudflare tunnel

### Manual Build (requires Go 1.24+)

```bash
git clone https://github.com/MastaChief117/keyparty.git
cd keyparty
go build -o keyparty .
./keyparty -port 8080
```

---

## Quick Start

### 1. Start the gateway

```bash
# npm
keyparty

# Or with flags
keyparty --port 8080 --password mysecret --tunnel
```

### 2. Open the dashboard

Go to **http://localhost:8080** in your browser.

### 3. Add an API key

The setup wizard will guide you, or:
1. Click **+ Add API Key**
2. Pick a provider (Groq has a free tier!)
3. Paste your API key
4. Click **Test Key** then **Save**

### 4. Get your unified key

Your unified API key is displayed at the top of the dashboard. Copy it.

### 5. Use it in your apps

```bash
# Curl
curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer gw-your-unified-key" \
  -d '{"model":"llama-3.3-70b-versatile","messages":[{"role":"user","content":"Hello"}]}'
```

Works with any OpenAI-compatible client:

```python
from openai import OpenAI

client = OpenAI(
    base_url="http://localhost:8080/v1",
    api_key="gw-your-unified-key"
)

response = client.chat.completions.create(
    model="llama-3.3-70b-versatile",
    messages=[{"role": "user", "content": "Hello"}]
)
```

### Provider/Model Override

```bash
# Force a specific provider
curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer YOUR_KEY" \
  -H "X-Gateway-Provider: gemini" \
  -d '{"model":"llama-3.3-70b-versatile","messages":[{"role":"user","content":"Hello"}]}'

# Override both provider and model
curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer YOUR_KEY" \
  -H "X-Gateway-Provider: nvidia" \
  -H "X-Gateway-Model: nvidia/nemotron-3-nano-30b-a3b" \
  -d '{"model":"llama-3.3-70b-versatile","messages":[{"role":"user","content":"Hello"}]}'
```

---

## Features

### Core Gateway

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
| Admin Dashboard | Web UI with dark mode, animations, responsive design |
| Setup Wizard | First-run wizard to add your first API key |

### Fun Zone

| Feature | What it does |
|---------|-------------|
| AI Rumble | Two providers roast each other in real-time via SSE streaming |
| Rap Battle | Two AIs drop bars with context memory across rounds |
| Model Roulette | Random provider picks your message |
| AI Therapist | Sarcastic therapy for when your code won't compile |
| Roast My Logs | AI roasts your API usage patterns |
| AI Fortune Teller | Dramatic predictions about your code's fate |
| Provider Roast | Two providers trash-talk each other's weaknesses |
| Debug Oracle | Paste an error, get dramatic debugging advice |
| Commit Message Generator | Dramatic, meme, poetic, or professional commit messages |
| Vibe Check | Get a random vibe rating for your message |
| AI Poll | Same prompt to multiple providers, compare answers |
| Magic 8-Ball | Ask the AI a question, get a dramatic answer |
| Provider Shipper | See how compatible two providers are |
| Fun Facts | Random facts about APIs and development |
| Provider Leaderboard | See which providers you use the most |
| Savings Tracker | Track how much you've saved via caching and failover |

### Pro Tools

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
| Cost Analytics | Cost by provider/day/model with bar charts |
| Webhooks | Notify URLs on events (request, error, failover) |
| Prompt Templates | Save and manage system prompts |
| Rate Limit Tiers | Tier-based rate limiting (free/premium) |
| Budget Alerts | Threshold alerts per virtual key |
| Search Logs | Filter by provider/model/status/virtual key |
| Provider Budgets | Set monthly token/cost budgets per provider |

---

## CLI Options

```bash
keyparty                           # Interactive setup wizard
keyparty --port 8080               # Start on custom port
keyparty --password mysecret       # Set admin password
keyparty --tunnel                  # Enable cloudflare tunnel
keyparty --port 8080 --tunnel     # Port + tunnel
keyparty --version                 # Show version
keyparty --help                    # Show help
```

---

## API Endpoints

### Proxy

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/v1/chat/completions` | POST | OpenAI-compatible chat completions |
| `/v1/models` | GET | List available models |
| `/health` | GET | Health check |

### Admin Dashboard

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/admin/stats` | GET | Gateway statistics |
| `/admin/keys` | GET/POST | List or add API keys |
| `/admin/keys/{id}` | DELETE | Delete a key |
| `/admin/keys/{id}/toggle` | POST | Enable/disable key |
| `/admin/test-key` | POST | Test an API key |
| `/admin/virtual-keys` | GET/POST | Manage virtual keys |
| `/admin/virtual-keys/{id}` | DELETE | Delete virtual key |
| `/admin/virtual-keys/{id}/toggle` | POST | Enable/disable virtual key |
| `/admin/unified-key` | GET/POST | Get or regenerate unified key |
| `/admin/providers` | GET | List available providers |
| `/admin/logs` | GET | Request logs |
| `/admin/logs/search` | GET | Search logs with filters |
| `/admin/export` | GET | Export config as JSON |
| `/admin/import` | POST | Import config from JSON |

### Fun Zone

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/admin/roast` | POST | Roast a username |
| `/admin/8ball` | POST | Ask the magic 8-ball |
| `/admin/ship` | POST | Provider compatibility check |
| `/admin/fun-facts` | GET | Random fun fact |
| `/admin/leaderboard` | GET | Provider usage leaderboard |
| `/admin/savings` | GET | Cost savings tracker |
| `/admin/rumble` | POST | AI rumble (SSE streaming) |
| `/admin/rap-battle` | POST | Rap battle (SSE streaming) |
| `/admin/roulette` | POST | Model roulette |
| `/admin/therapist` | POST | AI therapist |
| `/admin/vibe-check` | POST | Vibe check |
| `/admin/roast-logs` | POST | Roast your logs |
| `/admin/fortune` | POST | AI fortune teller |
| `/admin/provider-roast` | POST | Provider roast battle |
| `/admin/debug-oracle` | POST | Debug oracle |
| `/admin/commit-msg` | POST | Commit message generator |
| `/admin/poll` | POST | Multi-provider poll |

### Pro Tools

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/admin/smart-route` | POST | Smart router recommendation |
| `/admin/cost-limiter` | GET/POST/DELETE | Cost limits |
| `/admin/uptime` | GET | Provider uptime stats |
| `/admin/replay-queue` | GET | Failed request queue |
| `/admin/custom-models` | GET/POST/DELETE | Custom models |
| `/admin/health-check` | GET | Ping all providers |
| `/admin/tournament` | POST | AI tournament (SSE) |
| `/admin/tournament/models` | GET | Available tournament models |
| `/admin/recap` | GET | Weekly recap |
| `/admin/recap/generate` | POST | Generate new recap |
| `/admin/analytics` | GET | Cost analytics |
| `/admin/token-calculator` | POST | Token cost calculator |
| `/admin/compare` | POST | Compare providers |
| `/admin/playground` | POST | Chat playground |

### Security & Management

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/admin/guardrails` | GET/POST/DELETE | Guardrail rules |
| `/admin/blocked` | GET | Blocked requests log |
| `/admin/aliases` | GET/POST/DELETE | Model aliases |
| `/admin/failover` | GET/POST | Failover config |
| `/admin/failover/logs` | GET | Failover logs |
| `/admin/firewall` | GET/POST | Firewall config |
| `/admin/webhooks` | GET/POST/DELETE | Webhook management |
| `/admin/webhooks/test` | POST | Test webhook |
| `/admin/templates` | GET/POST/DELETE | Prompt templates |
| `/admin/rate-tiers` | GET/POST/DELETE | Rate limit tiers |
| `/admin/budget-alerts` | GET/POST/DELETE | Budget alerts |
| `/admin/budget-alerts/check` | GET | Check budget alerts |
| `/admin/ab-test` | GET/POST | A/B testing |
| `/admin/ab-test/vote` | POST | Vote on A/B test |
| `/admin/queue` | GET | Request queue |
| `/admin/queue/stats` | GET | Queue statistics |
| `/admin/provider-budgets` | GET/POST/DELETE | Provider budgets |
| `/admin/vk-usage` | GET | Virtual key usage stats |
| `/admin/auto-rotate` | POST | Trigger auto-rotation |
| `/admin/auto-rotate/status` | GET | Auto-rotation status |

---

## Supported Providers

| Provider | Free Tier | Models |
|----------|-----------|--------|
| Groq | Yes | llama-3.3-70b-versatile, llama-3.1-8b-instant |
| NVIDIA | Yes | nemotron-3-nano-30b-a3b, llama-3.1-nemotron-70b-instruct |
| Gemini | Yes | gemini-2.5-flash, gemini-2.5-pro |
| OpenAI | No | gpt-4o, gpt-4o-mini, gpt-4.1, gpt-4.1-mini, o3, o4-mini |
| Anthropic | No | claude-sonnet-4-5, claude-haiku-4-5, claude-opus-4-5 |
| DeepSeek | Yes | deepseek-chat, deepseek-r1 |
| Mistral | No | mistral-large-latest, codestral-latest |
| Together | No | meta-llama/Llama-3.3-70B-Instruct-Turbo |
| OpenRouter | Varies | auto |
| Fireworks | No | llama-v3p3-70b-instruct |

---

## Docker

### Pull from Docker Hub

```bash
docker pull steelquill69/keyparty:latest
```

### Build from Source

```bash
git clone https://github.com/MastaChief117/keyparty.git
cd keyparty
docker build -t keyparty .
```

Multi-stage build. Final image is ~20MB on Alpine.

### Run

```bash
docker run -d \
  --name keyparty \
  -p 8080:8080 \
  -e ADMIN_PASSWORD=your-secret-password \
  -v keyparty-data:/app/data \
  steelquill69/keyparty:latest
```

### Docker Compose

```yaml
services:
  keyparty:
    build: .
    # Or use the prebuilt image:
    # image: steelquill69/keyparty:latest
    ports:
      - "8080:8080"
    volumes:
      - keyparty-data:/app/data
    environment:
      - ADMIN_PASSWORD=${ADMIN_PASSWORD:-changeme}
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 10s

volumes:
  keyparty-data:
```

### Docker Commands

```bash
docker-compose up -d          # Start
docker-compose down           # Stop
docker-compose logs -f        # View logs
docker-compose up -d --build  # Rebuild after updates
docker-compose down -v        # Reset everything (deletes data)
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

## Comparison

| Feature | KeyParty | LiteLLM | OpenRouter |
|---------|----------|---------|------------|
| Self-hosted | Yes | Yes | No |
| Free & open source | Yes | Yes | No |
| Single binary (no deps) | Yes | No | N/A |
| npm install | Yes | No | N/A |
| Docker support | Yes | Yes | N/A |
| Setup wizard | Yes | No | N/A |
| Virtual Keys | Yes | No | No |
| AI Rumble / Rap Battle | Yes | No | No |
| Provider Roast | Yes | No | No |
| Model Roulette | Yes | No | No |
| Fortune Teller | Yes | No | No |
| Debug Oracle | Yes | No | No |
| Commit Message Gen | Yes | No | No |
| Smart Router | Yes | No | No |
| Cost Limiter | Yes | No | No |
| Provider Uptime | Yes | No | No |
| Replay Queue | Yes | No | No |
| Export Logs | Yes | No | No |
| Prompt Builder | Yes | No | No |
| Provider Race Mode | Yes | No | No |
| Compaction Failover | Yes | Yes | Yes |
| Cost Tracking | Yes | Yes | Yes |
| Dashboard | Yes | Yes | Yes |

---

## Security

- AES-256-GCM encryption at rest for API keys
- SSRF protection on all proxy requests
- Brute-force lockout on admin auth (5 attempts per minute)
- PII detection and blocking in guardrails
- Prompt injection blocking
- Rate limiting per virtual key

**Not designed for:** public-facing production, SOC2/HIPAA compliance, untrusted hosts.

---

## Performance

- **Idle:** 13MB RAM
- **Under load:** ~33MB RAM (stabilizes, no leaks)
- **Throughput:** ~220 RPS (gateway overhead only)
- **Binary:** 16MB on disk
- **Docker image:** ~20MB
- **Runs on:** a potato

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

**Q: How much does it cost?**
A: Free. MIT license. No "enterprise tier," no "contact sales." Just vibes.

**Q: Is my data safe?**
A: AES-256-GCM encryption at rest. Standard libraries. No hand-rolled crypto.

**Q: What happens when a provider dies?**
A: It fails over to another one. Your chat survives. We're basically life support for your conversations.

**Q: Can two AIs roast each other?**
A: Yes. Provider Roast and Rap Battle are built in. It's exactly as chaotic as it sounds.

**Q: Can I use this with Claude Code / Cursor?**
A: Yes. Just point them at the gateway URL.

**Q: Can I run this on a Raspberry Pi?**
A: Yes. It'll run on a potato if it has Go installed.

**Q: What's the catch?**
A: There is no catch. It's free, open source, MIT license. We just vibes.

**Q: Why is it called KeyParty?**
A: Because it's a party for your API keys. They finally get to hang out together instead of being locked in separate provider vaults.

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
- [x] Dashboard with dark mode + animations
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
- [x] Security audit (43 vulnerabilities fixed)
- [x] Docker support (multi-stage build, docker-compose)
- [x] Setup wizard for first-run experience
- [x] Fortune Teller, Provider Roast, Debug Oracle, Commit Message Generator
- [x] Smart Router, Cost Limiter, Provider Uptime
- [x] Replay Queue, Custom Models, Health Check
- [x] Export Logs (CSV/JSON), Prompt Builder
- [x] npm package (`@steelquill/keyparty`)
- [x] Docker Hub image (`steelquill69/keyparty`)
- [x] Provider fallback with automatic retry
- [x] Vibe Check, AI Therapist, Magic 8-Ball
- [ ] MCP support
- [ ] Provider speed tests
- [ ] Take over the AI gateway market (one $4 coffee at a time)

---

## Versioning

We don't do boring version numbers. Every release gets a codename.

| Version | Codename |
|---------|----------|
| v0.0.1 | Hello World But Broken |
| v0.1.0 | The Prototype of Chaos |
| v1.0.0 | It Compiles, Send It |
| v1.1.0 | Nemotron Nuke |
| v2.0.0 | Production Panic |
| v2.1.0 | Wombo Combo |
| v2.2.0 | Assignment: Gateway |
| v2.3.0 | Skynet Controller |
| v2.4.0 | It Works On My Machine |
| v3.0.0 | One Endpoint to Rule Them All |
| v4.0.0 | The Cake Is A Latency |
| v5.0.0 | API Gone Wild |
| v6.6.6 | Budget From Hell |

---

<div align="center">

**[Star this repo](https://github.com/MastaChief117/keyparty)** or don't. I'll add features anyway because I have no self-control.

*Made by a dude who should've been sleeping. Again.*

</div>
