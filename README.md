<div align="center">

# KeyParty

**One endpoint. Ten providers. Zero budget.**

[![Go Version](https://img.shields.io/badge/go-1.24+-00ADD8?style=flat-square&logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/license-MIT-blue.svg?style=flat-square)](LICENSE)
[![GitHub Stars](https://img.shields.io/github/stars/MastaChief117/keyparty?style=flat-square&color=yellow)](https://github.com/MastaChief117/keyparty)
[![Release](https://img.shields.io/github/v/release/MastaChief117/keyparty?style=flat-square)](https://github.com/MastaChief117/keyparty/releases)
[![Security Audit](https://img.shields.io/badge/security-31%20vulnerabilities%20fixed-green?style=flat-square)](docs/security.md)

*For the broke dev juggling free tiers at 3AM.*

**[What is this](#what-is-this) | [Quick Start](#quick-start) | [Features](#features) | [Docs](#documentation) | [FAQ](#faqs)**

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

Most API gateways are built for enterprises with Kubernetes clusters. **This one is built for the dev with $4 and questionable life choices.**

---

## Quick Start

```bash
git clone https://github.com/MastaChief117/keyparty.git
cd keyparty
chmod +x keyparty.sh
./keyparty.sh
```

Dashboard: **http://localhost:8080**

No Docker. No npm. No sacrifice to the JavaScript gods. The script handles Go, builds the binary, and optionally sets up a Cloudflare tunnel.

### Provider/Model Override

Route requests to any saved provider using request headers, even if the model name would normally go elsewhere:

```bash
# Default routing → auto-matches model to provider
curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer YOUR_UNIFIED_KEY" \
  -d '{"model":"llama-3.3-70b-versatile","messages":[{"role":"user","content":"Hello"}]}'

# Force a specific provider
curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer YOUR_UNIFIED_KEY" \
  -H "X-Gateway-Provider: gemini" \
  -d '{"model":"llama-3.3-70b-versatile","messages":[{"role":"user","content":"Hello"}]}'

# Force provider + override model name
curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer YOUR_UNIFIED_KEY" \
  -H "X-Gateway-Provider: nvidia" \
  -H "X-Gateway-Model: nvidia/nemotron-3-nano-30b-a3b" \
  -d '{"model":"llama-3.3-70b-versatile","messages":[{"role":"user","content":"Hello"}]}'
```

### Manual build

```bash
go build -o keyparty .
./keyparty -port 8080 -admin-pass your-password-here
```

---

## Architecture

```
                    Client
                       │
                       ▼
                 ┌───────────┐
                 │  KeyParty  │
                 │  Gateway   │
                 └─────┬─────┘
                       │
        ┌──────────────┼──────────────┐
        │              │              │
        ▼              ▼              ▼
   ┌─────────┐   ┌─────────┐   ┌─────────┐
   │  OpenAI  │   │ Claude  │   │ Gemini  │
   └─────────┘   └─────────┘   └─────────┘
        │
        ▼
   ┌─────────┐
   │  Groq   │
   └─────────┘
        │
        ▼
   ┌─────────────┐
   │ OpenRouter  │
   └─────────────┘
```

One client → one KeyParty endpoint → any provider. That's it. [Full architecture docs →](docs/architecture.md)

---

## Features

### Core

| Feature | What it does |
|---------|-------------|
| Multi-Provider Routing | OpenAI, Anthropic, Gemini, Groq, NVIDIA, Together, DeepSeek, OpenRouter, Fireworks, Mistral — one endpoint |
| Round-Robin Rotation | Spreads requests across multiple keys per provider |
| Provider/Model Override | Route to any saved provider via `X-Gateway-Provider` header, override model via `X-Gateway-Model` |
| Compaction Failover | When quota dies (429/402), summarizes chat history and fails over to backup provider |
| Model-Based Routing | Send `model: "llama-3.3-70b-versatile"` and it auto-matches to the right provider |
| Provider Race Mode | Sends to multiple providers simultaneously, returns the fastest |
| Guardrails | PII detection, prompt injection blocking, custom regex rules |
| Virtual Keys | Consumer-facing keys with budgets, rate limits, and model allowlists |
| Model Aliases | Map `smart` → `claude-sonnet-4-5`, `fast` → `gpt-4o-mini` |
| Encryption at Rest | AES-256-GCM for all API keys |
| Request Logging | Full audit trail with token usage and cost tracking |
| Response Caching | SHA-256 deduplication |
| Admin Dashboard | Web UI with dark mode |

### Fun Stuff

| Feature | What it does |
|---------|-------------|
| AI Rumble | Two providers roast each other in real-time via SSE |
| Rap Battle | Two AIs drop bars with context memory across rounds |
| Model Roulette | Random provider picks your message |
| AI Therapist | Sarcastic therapy for when your code won't compile |
| Roast My Logs | AI roasts your API usage patterns |
| AI Poll | Same prompt to multiple providers, compare answers |

### Pro Tools

| Feature | What it does |
|---------|-------------|
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

[Full API reference →](docs/api.md) | [Routing & failover details →](docs/routing.md) | [Supported providers →](docs/providers.md)

---

## Comparison

| Feature | KeyParty | LiteLLM | OpenRouter |
|---------|----------|---------|------------|
| Self-hosted | ✅ | ✅ | ❌ |
| Free & open source | ✅ | ✅ | ❌ |
| Single binary (no deps) | ✅ | ❌ | N/A |
| Virtual Keys | ✅ | ❌ | ❌ |
| AI Rumble / Rap Battle | ✅ | ❌ | ❌ |
| Model Roulette | ✅ | ❌ | ❌ |
| Provider Race Mode | ✅ | ❌ | ❌ |
| Chat Compaction Failover | ✅ | ✅ | ✅ |
| Cost Tracking | ✅ | ✅ | ✅ |
| Dashboard | ✅ | ✅ | ✅ |
| Budget Alerts | ✅ | ❌ | ❌ |
| Auto-Rotate Keys | ✅ | ❌ | ❌ |
| Weekly Recaps | ✅ | ❌ | ❌ |
| Setup Script | ✅ | ❌ | N/A |

---

## Security

AES-256-GCM encryption at rest, SSRF protection via `net.ParseIP()`, brute-force lockout on admin auth, PII detection, prompt injection blocking. 31 vulnerabilities found and fixed in audits.

**Not designed for:** public-facing production, SOC2/HIPAA compliance, untrusted hosts.

[Full security docs →](docs/security.md) | [Threat model →](docs/security.md#threat-model)

---

## Performance

- **Idle:** 13MB RAM
- **Under load:** ~33MB RAM (stabilizes, no leaks)
- **Throughput:** ~220 RPS (gateway overhead only)
- **Binary:** 16MB on disk
- **Runs on:** a potato 🥔

---

## Who this is for

- Solo devs building AI agents on free tiers
- Small teams sharing API credits
- Hackathon projects needing multi-provider resilience
- Students learning AI/LLM integration
- Anyone tired of managing 5 different AI API dashboards

## Who this is NOT for

- Enterprises needing horizontal scaling (it's a single binary)
- Teams needing RBAC, SSO, or compliance certifications
- Anyone processing millions of requests (SQLite has limits)
- gRPC-only setups (HTTP only)

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
A: AES-256-GCM encryption at rest. Standard libraries. No hand-rolled crypto. [Full threat model →](docs/security.md)

**Q: What happens when a provider dies?**
A: It fails over to another one. Your chat survives. We're basically life support for your conversations.

**Q: Can two AIs roast each other?**
A: Yes. AI Rumble and Rap Battle are built in. It's exactly as chaotic as it sounds.

**Q: Can I use this with Claude Code / Cursor?**
A: Yes. Just point them at the gateway.

**Q: Can I run this on a Raspberry Pi?**
A: Yes. It'll run on a potato if it has Go installed.

**Q: What's the catch?**
A: There is no catch. It's free, open source, MIT license. We just vibes.

**Q: Why is it called KeyParty?**
A: Because it's a party for your API keys. They finally get to hang out together instead of being locked in separate provider vaults. It's wholesome, actually.

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
- [x] SSRF bypass fix (IPv6-mapped, decimal, hex IPs)
- [x] Brute-force protection on admin auth
- [x] Provider/Model override via request headers
- [x] Model-based auto-routing (send model name, gateway finds the right provider)
- [x] Compaction rollover verified working (429 → compact → failover)
- [ ] Semantic caching
- [ ] MCP support
- [ ] A/B testing
- [ ] Take over the AI gateway market (one $4 coffee at a time)

---

<div align="center">

**[Star this repo](https://github.com/MastaChief117/keyparty)** or don't. I'll add features anyway because I have no self-control.

*Made by a dude who should've been sleeping. Again.*

</div>
