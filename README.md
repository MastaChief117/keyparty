<div align="center">

# KeyParty

**One endpoint. Ten providers. Zero budget.**

[![Go Version](https://img.shields.io/badge/go-1.24+-00ADD8?style=flat-square&logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg?style=flat-square)](LICENSE)
[![GitHub Stars](https://img.shields.io/github/stars/MastaChief117/keyparty?style=flat-square&color=yellow)](https://github.com/MastaChief117/keyparty)
[![Docker](https://img.shields.io/badge/docker-ready-2496ED?style=flat-square&logo=docker)](https://hub.docker.com/r/steelquill69/keyparty)
[![npm](https://img.shields.io/badge/npm-@steelquill%2Fkeyparty-CB3837?style=flat-square&logo=npm)](https://www.npmjs.com/package/@steelquill/keyparty)
[![GitHub Packages](https://img.shields.io/badge/GitHub-Packages-24292e?style=flat-square&logo=github)](https://github.com/MastaChief117/keyparty/packages)

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

Also available on GitHub Packages:
```bash
npm install --registry=https://npm.pkg.github.com @MastaChief117/keyparty
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

### GitHub Container Registry

```bash
# Same image, different registry
docker run -d \
  --name keyparty \
  -p 8080:8080 \
  -e ADMIN_PASSWORD=your-secret \
  -v keyparty-data:/app/data \
  ghcr.io/mastachief117/keyparty:latest
```

### npm (GitHub Packages)

```bash
# Install from GitHub Packages
npm install --registry=https://npm.pkg.github.com @MastaChief117/keyparty
keyparty
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
- Basically does everything except pay your rent

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

Go to **http://localhost:8080** in your browser. It's prettier than your code.

### 3. Add an API key

The setup wizard will guide you, or:
1. Click **+ Add API Key**
2. Pick a provider (Groq has a free tier — you're welcome)
3. Paste your API key
4. Click **Test Key** then **Save**

### 4. Get your unified key

Your unified API key is displayed at the top of the dashboard. Copy it. This key unlocks everything.

### 5. Use it in your apps

```bash
# Curl (the classic)
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

### Core Gateway (the serious stuff)

| Feature | What it does |
|---------|-------------|
| Multi-Provider Routing | OpenAI, Anthropic, Gemini, Groq, NVIDIA, Together, DeepSeek, OpenRouter, Fireworks, Mistral — one endpoint to rule them all |
| Round-Robin Rotation | Spreads requests across multiple keys per provider like butter on toast |
| Provider/Model Override | Route to any saved provider via `X-Gateway-Provider` header |
| Compaction Failover | When quota dies (429/402), summarizes chat history and fails over. Your chat survives. You're welcome. |
| Model-Based Routing | Send `model: "llama-3.3-70b-versatile"` and it auto-matches to the right provider |
| Provider Race Mode | Sends to multiple providers simultaneously, returns the fastest. It's a race and you're the judge. |
| Guardrails | PII detection, prompt injection blocking, custom regex rules. Keeps the bad stuff out. |
| Virtual Keys | Consumer-facing keys with budgets, rate limits, and model allowlists |
| Model Aliases | Map `smart` → `claude-sonnet-4-5`, `fast` → `gpt-4o-mini` |
| Encryption at Rest | AES-256-GCM for all API keys. Your secrets are safe with us. |
| Request Logging | Full audit trail with token usage and cost tracking. Know where every penny went. |
| Response Caching | SHA-256 deduplication. Same prompt? Same response. No double-charging. |
| Admin Dashboard | Web UI with dark mode, animations, responsive design. Looks good on your phone too. |
| Setup Wizard | First-run wizard to add your first API key. We hold your hand. |

### Fun Zone (the chaotic good)

| Feature | What it does |
|---------|-------------|
| AI Rumble | Two providers roast each other in real-time via SSE streaming. Popcorn not included. |
| Rap Battle | Two AIs drop bars with context memory across rounds. Better than your SoundCloud career. |
| Model Roulette | Random provider picks your message. Spin the wheel of AI fate. |
| AI Therapist | Sarcastic therapy for when your code won't compile. "And how does that make you feel?" |
| Roast My Logs | AI roasts your API usage patterns. Your coding habits are about to get judged. |
| AI Fortune Teller | Dramatic predictions about your code's fate. Spoiler: it's not great. |
| Provider Roast | Two providers trash-talk each other's weaknesses. Drama you actually want. |
| Debug Oracle | Paste an error, get dramatic debugging advice. "Have you tried turning it off and on again?" |
| Commit Message Generator | Dramatic, meme, poetic, or professional commit messages. Never write "fix stuff" again. |
| Vibe Check | Get a random vibe rating for your message. Are you a 10/10 or a 3/10? |
| AI Poll | Same prompt to multiple providers, compare answers. Who said it better? |
| Magic 8-Ball | Ask the AI a question, get a dramatic answer. "Outlook: not so good." |
| Provider Shipper | See how compatible two providers are. Matchmaking for AI models. |
| Fun Facts | Random facts about APIs and development. Learn something while you wait for rate limits. |
| Provider Leaderboard | See which providers you use the most. No judgment. |
| Savings Tracker | Track how much you've saved via caching and failover. Your wallet will thank you. |

### Pro Tools (the big brain stuff)

| Feature | What it does |
|---------|-------------|
| Smart Router | AI recommends best provider based on cost/speed/quality priority. Let the AI decide for once. |
| Cost Limiter | Set daily/weekly/monthly spending caps per provider. Because self-control is hard. |
| Provider Uptime | Track provider reliability over time with stats. Know who's flaky. |
| Replay Queue | Retry failed requests with one click. Second chances for your broken requests. |
| Custom Models | Add models not in the default list. Bring your own model, we don't judge. |
| Health Check | Ping all providers and report status in real-time. The pulse of your AI empire. |
| Export Logs | Download logs as CSV or JSON. For when you need to prove something to someone. |
| Prompt Builder | Build prompts with variables, optionally enhance with AI. LEGO but for prompts. |
| VK Usage Dashboard | Per virtual key usage breakdown with cost, latency, errors. Know who's spending what. |
| Auto-Rotate Keys | Pick the healthiest key per provider automatically. The gateway does the thinking. |
| Chat Playground | Full chat UI with SSE streaming, provider/model picker. Test before you deploy. |
| Weekly Recap | Stats summary + AI-generated snarky reports. Your AI week in review. |
| Cost Analytics | Cost by provider/day/model with bar charts. Pretty graphs for your wallet. |
| Webhooks | Notify URLs on events (request, error, failover). Stay in the loop. |
| Prompt Templates | Save and manage system prompts. Build once, use forever. |
| Rate Limit Tiers | Tier-based rate limiting (free/premium). Because not everyone gets the VIP treatment. |
| Budget Alerts | Threshold alerts per virtual key. We'll tell you when you're about to go broke. |
| Search Logs | Filter by provider/model/status/virtual key. Find that one request from last Tuesday. |
| Provider Budgets | Set monthly token/cost budgets per provider. Budgets you'll actually follow. |

---

## CLI Options

```bash
keyparty                           # Interactive setup wizard (the friendly path)
keyparty --port 8080               # Start on custom port
keyparty --password mysecret       # Set admin password (don't use "password123")
keyparty --tunnel                  # Enable cloudflare tunnel (for the brave)
keyparty --port 8080 --tunnel     # Port + tunnel (the combo meal)
keyparty --version                 # Show version (and the codename of course)
keyparty --help                    # Show help (when all else fails)
```

---

## API Endpoints

### Proxy (the main event)

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/v1/chat/completions` | POST | OpenAI-compatible chat completions (the bread and butter) |
| `/v1/models` | GET | List available models |
| `/health` | GET | Health check ("are you alive?") |

### Admin Dashboard (the control center)

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/admin/stats` | GET | Gateway statistics (how broke are you?) |
| `/admin/keys` | GET/POST | List or add API keys |
| `/admin/keys/{id}` | DELETE | Delete a key (goodbye, old friend) |
| `/admin/keys/{id}/toggle` | POST | Enable/disable key (the snooze button) |
| `/admin/test-key` | POST | Test an API key ("does this thing work?") |
| `/admin/virtual-keys` | GET/POST | Manage virtual keys |
| `/admin/virtual-keys/{id}` | DELETE | Delete virtual key |
| `/admin/virtual-keys/{id}/toggle` | POST | Enable/disable virtual key |
| `/admin/unified-key` | GET/POST | Get or regenerate unified key |
| `/admin/providers` | GET | List available providers |
| `/admin/logs` | GET | Request logs (the receipts) |
| `/admin/logs/search` | GET | Search logs with filters (CSI: KeyParty) |
| `/admin/export` | GET | Export config as JSON |
| `/admin/import` | POST | Import config from JSON |

### Fun Zone (the party section)

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/admin/roast` | POST | Roast a username (no mercy) |
| `/admin/8ball` | POST | Ask the magic 8-ball (ask nicely) |
| `/admin/ship` | POST | Provider compatibility check (AI matchmaking) |
| `/admin/fun-facts` | GET | Random fun fact (for the curious) |
| `/admin/leaderboard` | GET | Provider usage leaderboard (who's carrying?) |
| `/admin/savings` | GET | Cost savings tracker (pat yourself on the back) |
| `/admin/rumble` | POST | AI rumble (SSE streaming) (let them fight) |
| `/admin/rap-battle` | POST | Rap battle (SSE streaming) (drop the mic) |
| `/admin/roulette` | POST | Model roulette (spin the wheel) |
| `/admin/therapist` | POST | AI therapist ("and how does that make you feel?") |
| `/admin/vibe-check` | POST | Vibe check (are you worthy?) |
| `/admin/roast-logs` | POST | Roast your logs (your code is about to get judged) |
| `/admin/fortune` | POST | AI fortune teller ("your code will compile... eventually") |
| `/admin/provider-roast` | POST | Provider roast battle (let them fight) |
| `/admin/debug-oracle` | POST | Debug oracle ("have you checked Stack Overflow?") |
| `/admin/commit-msg` | POST | Commit message generator (never write "fix stuff" again) |
| `/admin/poll` | POST | Multi-provider poll (who said it best?) |

### Pro Tools (the big brain section)

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/admin/smart-route` | POST | Smart router recommendation ("which provider should I use?") |
| `/admin/cost-limiter` | GET/POST/DELETE | Cost limits (budgets you'll actually follow) |
| `/admin/uptime` | GET | Provider uptime stats (who's flaky?) |
| `/admin/replay-queue` | GET | Failed request queue (second chances) |
| `/admin/custom-models` | GET/POST/DELETE | Custom models (bring your own) |
| `/admin/health-check` | GET | Ping all providers (the pulse check) |
| `/admin/tournament` | POST | AI tournament (SSE) (may the best model win) |
| `/admin/tournament/models` | GET | Available tournament models (the contestants) |
| `/admin/recap` | GET | Weekly recap (your AI week in review) |
| `/admin/recap/generate` | POST | Generate new recap (spit out the stats) |
| `/admin/analytics` | GET | Cost analytics (pretty graphs for your wallet) |
| `/admin/token-calculator` | POST | Token cost calculator (how much will this cost?) |
| `/admin/compare` | POST | Compare providers (who does it better?) |
| `/admin/playground` | POST | Chat playground (test before you deploy) |

### Security & Management (the boring but important stuff)

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/admin/guardrails` | GET/POST/DELETE | Guardrail rules (keep the bad stuff out) |
| `/admin/blocked` | GET | Blocked requests log (the wall of shame) |
| `/admin/aliases` | GET/POST/DELETE | Model aliases (call it what you want) |
| `/admin/failover` | GET/POST | Failover config (plan B) |
| `/admin/failover/logs` | GET | Failover logs (when plan B kicked in) |
| `/admin/firewall` | GET/POST | Firewall config (the bouncer) |
| `/admin/webhooks` | GET/POST/DELETE | Webhook management (stay in the loop) |
| `/admin/webhooks/test` | POST | Test webhook ("did it go through?") |
| `/admin/templates` | GET/POST/DELETE | Prompt templates (build once, use forever) |
| `/admin/rate-tiers` | GET/POST/DELETE | Rate limit tiers (VIP vs. pleb) |
| `/admin/budget-alerts` | GET/POST/DELETE | Budget alerts ("you're about to go broke") |
| `/admin/budget-alerts/check` | GET | Check budget alerts ("how broke am I?") |
| `/admin/ab-test` | GET/POST | A/B testing (which one's better?) |
| `/admin/ab-test/vote` | POST | Vote on A/B test (democracy in action) |
| `/admin/queue` | GET | Request queue (waiting room) |
| `/admin/queue/stats` | GET | Queue statistics (how long's the wait?) |
| `/admin/provider-budgets` | GET/POST/DELETE | Provider budgets (per-provider spending caps) |
| `/admin/vk-usage` | GET | Virtual key usage stats (who's spending what?) |
| `/admin/auto-rotate` | POST | Trigger auto-rotation (the gateway does the thinking) |
| `/admin/auto-rotate/status` | GET | Auto-rotation status (is it rotating?) |

---

## Supported Providers

| Provider | Free Tier | Models |
|----------|-----------|--------|
| Groq | Yes (and it's fast) | llama-3.3-70b-versatile, llama-3.1-8b-instant |
| NVIDIA | Yes (Nemotron gang) | nemotron-3-nano-30b-a3b, llama-3.1-nemotron-70b-instruct |
| Gemini | Yes (Google's entry) | gemini-2.5-flash, gemini-2.5-pro |
| OpenAI | No (but you knew that) | gpt-4o, gpt-4o-mini, gpt-4.1, gpt-4.1-mini, o3, o4-mini |
| Anthropic | No (Claude isn't cheap) | claude-sonnet-4-5, claude-haiku-4-5, claude-opus-4-5 |
| DeepSeek | Yes (the underdog) | deepseek-chat, deepseek-r1 |
| Mistral | No (French AI) | mistral-large-latest, codestral-latest |
| Together | No (but worth it) | meta-llama/Llama-3.3-70B-Instruct-Turbo |
| OpenRouter | Varies (the wildcard) | auto |
| Fireworks | No (fast and flashy) | llama-v3p3-70b-instruct |

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

Multi-stage build. Final image is ~20MB on Alpine. Smaller than most node_modules folders.

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
docker-compose up -d          # Start (the magic)
docker-compose down           # Stop (the sadness)
docker-compose logs -f        # View logs (the drama)
docker-compose up -d --build  # Rebuild after updates (fresh start)
docker-compose down -v        # Reset everything (the nuclear option)
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `ADMIN_PASSWORD` | (empty) | Admin password for dashboard. Empty = no auth. We trust you. |

### Volumes

| Mount | Description |
|-------|-------------|
| `/app/data` | SQLite database (persists keys, logs, config). Don't lose this. |

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
| Two AIs roasting each other | Yes | No | No |
| Setup that doesn't suck | Yes | No | N/A |

---

## Security

- AES-256-GCM encryption at rest for API keys (your secrets are safe)
- SSRF protection on all proxy requests (no sneaky stuff)
- Brute-force lockout on admin auth (5 attempts per minute — try harder)
- PII detection and blocking in guardrails (we see you, credit card numbers)
- Prompt injection blocking (nice try, hackers)
- Rate limiting per virtual key (no one hogs the bandwidth)

**Not designed for:** public-facing production, SOC2/HIPAA compliance, untrusted hosts, or people who don't like fun.

---

## Performance

- **Idle:** 13MB RAM (smaller than your Chrome tabs)
- **Under load:** ~33MB RAM (stabilizes, no leaks — we checked)
- **Throughput:** ~220 RPS (gateway overhead only)
- **Binary:** 16MB on disk (smaller than most node_modules)
- **Docker image:** ~20MB (tiny but mighty)
- **Runs on:** a potato (seriously, we tested it)

---

## Who this is for

- Solo devs building AI agents on free tiers (the broke and beautiful)
- Small teams sharing API credits (sharing is caring)
- Hackathon projects needing multi-provider resilience (ship fast, break nothing)
- Students learning AI/LLM integration (welcome to the chaos)
- Anyone tired of managing 5 different AI API dashboards (we feel you)
- People who want to watch two AIs roast each other (culture)

## Who this is NOT for

- Enterprises needing horizontal scaling (it's a single binary, not a fleet)
- Teams needing RBAC, SSO, or compliance certifications (we're not that fancy)
- Anyone processing millions of requests (SQLite has limits, and so do we)
- People who don't like fun (why are you even here?)

---

## FAQs

**Q: What is this?**
A: An AI gateway for broke devs. One endpoint, ten providers, zero budget. It's like a bouncer for your AI requests, but it actually lets everyone in.

**Q: Why?**
A: Because my OpenAI quota died at 2am and I was too angry to sleep. This project is revenge against rate limits. And it worked.

**Q: How much does it cost?**
A: Free. Apache 2.0 license. No "enterprise tier," no "contact sales." Just vibes. The only thing you pay for is your API keys.

**Q: Is my data safe?**
A: AES-256-GCM encryption at rest. Standard libraries. No hand-rolled crypto. Your secrets are safer here than in your browser history.

**Q: What happens when a provider dies?**
A: It fails over to another one. Your chat survives. We're basically life support for your conversations. You're welcome.

**Q: Can two AIs roast each other?**
A: Yes. Provider Roast and Rap Battle are built in. It's exactly as chaotic as it sounds. Grab popcorn.

**Q: Can I use this with Claude Code / Cursor?**
A: Yes. Just point them at the gateway URL. They'll never know the difference.

**Q: Can I run this on a Raspberry Pi?**
A: Yes. It'll run on a potato if it has Go installed. We've seen worse hardware run worse software.

**Q: What's the catch?**
A: There is no catch. It's free, open source, Apache 2.0 license. We just vibes. The only catch is you might get addicted to watching AIs roast each other.

**Q: Why is it called KeyParty?**
A: Because it's a party for your API keys. They finally get to hang out together instead of being locked in separate provider vaults. It's wholesome, actually.

**Q: Can the AI tell my fortune?**
A: Yes. And it will roast your code while doing it. "Your future holds... many merge conflicts."

**Q: Can I set spending limits?**
A: Yes. Cost Limiter lets you set daily/weekly/monthly caps per provider. Because self-control is hard, but budgets help.

**Q: Can I track provider reliability?**
A: Yes. Provider Uptime shows success rates, latency, and failure counts over time. Now you know who's flaky.

**Q: Can I watch two AIs rap battle?**
A: Yes. And it's exactly as ridiculous as you'd expect. "I'm the model with the most, my tokens are the dopest..."

---

## Roadmap

### Done

- [x] Gateway with multi-provider routing (the main quest)
- [x] Failover with chat compaction (plan B, but fancy)
- [x] AES-256-GCM encryption (your secrets are safe)
- [x] Dashboard with dark mode + animations (because dark mode is life)
- [x] Provider race mode (may the fastest model win)
- [x] Guardrails (PII, injection blocking) (keep the bad stuff out)
- [x] Virtual keys with budgets (sharing is caring)
- [x] Model aliases (call it what you want)
- [x] Request logging & cost tracking (know where every penny went)
- [x] AI Rumble, Rap Battle, Model Roulette (the fun stuff)
- [x] Webhooks, Prompt Templates, AI Poll (the useful stuff)
- [x] Rate Limit Tiers & Budget Alerts (the responsible stuff)
- [x] VK Usage Dashboard, Auto-Rotate Keys (the smart stuff)
- [x] Chat Playground, Weekly Recaps, Cost Analytics (the pretty stuff)
- [x] Security audit (43 vulnerabilities fixed) (the serious stuff)
- [x] Docker support (multi-stage build, docker-compose) (the container stuff)
- [x] Setup wizard for first-run experience (the hand-holding stuff)
- [x] Fortune Teller, Provider Roast, Debug Oracle, Commit Message Generator (the chaotic stuff)
- [x] Smart Router, Cost Limiter, Provider Uptime (the smart stuff)
- [x] Replay Queue, Custom Models, Health Check (the useful stuff)
- [x] Export Logs (CSV/JSON), Prompt Builder (the data stuff)
- [x] npm package (`@steelquill/keyparty`) (the install stuff)
- [x] Docker Hub image (`steelquill69/keyparty`) (the container stuff)
- [x] Provider fallback with automatic retry (the resilient stuff)
- [x] Vibe Check, AI Therapist, Magic 8-Ball (the fun stuff)

### Coming Soon

- [ ] Request replay with diff (before/after comparison)
- [ ] Provider response time leaderboard (who's the fastest?)
- [ ] Monthly cost report (auto-generated spending summary)
- [ ] Model cost calculator (estimate costs before you send)
- [ ] Custom guardrail regex (your rules, your way)
- [ ] MCP support (Claude Desktop, tool integrations)
- [ ] Prompt marketplace (save, share, rate community prompts)
- [ ] Multi-user auth (separate logins, per-user tracking)
- [ ] Request queue with priorities (VIP requests jump the line)
- [ ] Semantic request search ("find all requests about Python debugging")

### Big Brain Features

- [ ] Provider speed tests (automated benchmarking, auto-update smart router)
- [ ] Agent mode (tools: web search, code execution via function calling)
- [ ] Federated gateway (multiple instances share keys and sync state)
- [ ] Real-time cost dashboard (WebSocket-powered live spending feed)
- [ ] Plugin system (custom middleware in Go or WASM)

### Chaos Features

- [ ] AI debate club (3+ providers argue, judge picks winner)
- [ ] Provider tier list (auto-rankings based on your usage)
- [ ] Code review mode (send PR diff to multiple providers, compare)
- [ ] Roast my code (paste code, get roasted by the angriest provider)

### The Dream

- [ ] Take over the AI gateway market (one $4 coffee at a time)

---

## Versioning

We don't do boring version numbers. Every release gets a codename that reflects the emotional state of the developer at the time.

| Version | Codename | What was happening |
|---------|----------|-------------------|
| v0.0.1 | Hello World But Broken | Even hello world didn't work. Dark times. |
| v0.1.0 | The Prototype of Chaos | It technically worked. We were scared. |
| v1.0.0 | It Compiles, Send It | The moment of truth. It compiled. We cried. |
| v1.1.0 | Nemotron Nuke | NVIDIA said "here's a free model" and we went wild. |
| v1.2.0 | malloc(meltdown) | Memory issues. So many memory issues. |
| v1.2.1 | The Readme Strikes Back | The docs fight back. |
| v1.2.2 | Four Registries of the Apocalypse | Publishing to everything at once. |
| v1.2.3 | The Changelogening | Changelogs finally exist. |
| v1.3.0 | Segfault Surprise | The segfaults came from everywhere. |
| v1.4.0 | Null Pointer Nap | We took a nap. The null pointers didn't. |
| v1.5.0 | Race Condition Rave | Concurrency bugs. The party nobody wanted. |
| v1.6.0 | Cache Money | Caching finally worked. We felt rich. |
| v1.7.0 | sudo make me a sandwich | The sandwich was a lie. But the gateway worked. |
| v1.8.0 | rm -rf /sanity | We lost our sanity. The gateway gained features. |
| v1.9.0 | ctrl+alt+defeat | The bugs fought back. We fought harder. |
| v2.0.0 | Production Panic | We put it in production. Nobody died. |
| v2.1.0 | Wombo Combo | Multiple providers combined. It was beautiful chaos. |
| v2.2.0 | Assignment: Gateway | "Just build a gateway," they said. "It'll be fun," they said. |
| v2.3.0 | It's ALIVE!!! Call John!!!! | It started making decisions on its own. We're fine. |
| v2.4.0 | It Works On My Machine | The classic developer excuse. Now it's a feature. |
| v2.5.0 | The Gang Fixes Rate Limits | Rate limits thought they could stop us. They were wrong. |
| v3.0.0 | One Endpoint to Rule Them All | The One Ring of API gateways. |
| v3.1.0 | Hack the Planet | We hacked the AI gateway market. One commit at a time. |
| v3.2.0 | Zero Budget, Zero Chill | $4 budget. Infinite ambition. No regrets. |
| v3.3.0 | Have You Tried Turning It Off | Sometimes the best feature is the off switch. |
| v4.0.0 | The Cake Is A Latency | The cake was a lie. The latency was real. |
| v4.1.0 | 429 Too Many Roasts | We roasted too hard. The rate limits agreed. |
| v4.2.0 | Deploy On Friday I Dare You | We did it. Nothing broke. We were shocked. |
| v5.0.0 | API Gone Wild | The APIs started doing their own thing. We let them. |
| v5.1.0 | Catch Me If You Can (Failover) | Failover became a sport. We're winning. |
| v5.2.0 | Rage Against The Machine (Learning) | We raged against the machine learning models. They learned. |
| v6.0.0 | The Last API Key | The final key. The ultimate key. The key to end all keys. |
| v6.6.6 | Budget From Hell | $6.66 budget. Coincidence? We think not. |

---

<div align="center">

**[Star this repo](https://github.com/MastaChief117/keyparty)** or don't. I'll add features anyway because I have no self-control. Seriously, I can't stop. Help.

*Made by a dude who should've been sleeping. Again. And again. And again.*

*If you're reading this at 3AM, you're exactly who this is for.*

</div>
