<div align="center">

# 🔀 AI Gateway

**One API to rule them all. Route LLM requests across 10 providers like a maniac.**

*Failover when quota dies. Encrypt your keys. Race providers. Guard your prompts. All from one dashboard.*

**[WTF is this](#wtf-is-this) • [Quick Start](#quick-start-in-60-seconds) • [The Stuff](#the-stuff) • [FAQ (it's funny)](#faqs)**

*Features Added: 12 • Sleep Deprived: Immensely • Coffee Consumed: Yes*

</div>

---

## WTF is this

So basically you have API keys for like 5 different AI providers and you're tired of:
- OpenAI rate limiting you at 2am when you're vibing
- Anthropic eating your credits like pacman eating dots
- Having to change your code every time you want to switch providers
- Your API keys sitting in plain text like a clown at a security convention

**This gateway solves all of that.** You point ONE endpoint at it, and it:
- Routes to whichever provider is breathing
- Rotates across your keys automatically (like a DJ but for API keys)
- Fails over to another provider when one dies (your chat history survives because we're not monsters)
- Encrypts everything at rest so nobody can steal your keys (yes we take security seriously)
- Blocks prompt injection and PII from leaking (your SSN is safe with us)
- Lets you race providers against each other for the fastest response (may the fastest API win)

It's like a load balancer but for AI. And it has a dashboard. Because everything needs a dashboard.

---

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

That's it. No Docker. No Kubernetes. No npm install. No node_modules folder the size of Texas. Just a single binary and vibes.

---

## The Stuff

| Feature | What it does | Cool Factor |
|---------|-------------|-------------|
| 🔀 **Multi-Provider Routing** | OpenAI, Anthropic, Gemini, Groq, NVIDIA NIM, Together, DeepSeek, OpenRouter, Fireworks, Mistral — one endpoint | ⭐⭐⭐⭐⭐ |
| 🔄 **Round-Robin Rotation** | Spreads requests across multiple keys per provider like butter on toast | ⭐⭐⭐⭐ |
| 💥 **Compaction Failover** | When quota dies (402), summarizes your chat history and sends it to a backup provider. Nothing is lost. You're welcome. | ⭐⭐⭐⭐⭐ |
| ⚡ **Provider Race Mode** | Sends your message to multiple providers simultaneously, returns the fastest response. It's a race and everyone wins. | ⭐⭐⭐⭐⭐ |
| 🛡️ **Guardrails** | PII detection, prompt injection blocking, custom regex rules. We catch the bad stuff before it reaches the AI. | ⭐⭐⭐⭐ |
| 🔑 **Virtual Keys** | Give users their own API keys with budgets, rate limits, and model allowlists. Share without sharing your real keys. | ⭐⭐⭐⭐ |
| 🏷️ **Model Aliases** | Map `smart` → `claude-sonnet-4-5`, `fast` → `gpt-4o-mini`. Sound smart without knowing model names. | ⭐⭐⭐ |
| 🔒 **Encryption at Rest** | AES-256-GCM encryption for all API keys. Your keys are safe. We promise. | ⭐⭐⭐⭐⭐ |
| 📊 **Request Logging** | Full audit trail with token usage and cost tracking. Know where your money goes. | ⭐⭐⭐ |
| 💰 **Cost Tracking** | Per-key and per-provider cost estimation. Your wallet will thank you. | ⭐⭐⭐ |
| 📦 **Response Caching** | SHA-256 deduplication. Same request? Cached. You're welcome, your rate limits. | ⭐⭐⭐ |
| 🎛️ **Admin Dashboard** | Web UI with tabs for everything. Dark mode because light mode is a war crime. | ⭐⭐⭐⭐⭐ |

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

**Your conversation survives.** Even if the provider dies. That's the whole point. We don't let your chat die with the provider.

Configure it in the dashboard or via API:

```bash
# Enable failover (because why not)
curl -X POST http://localhost:8080/admin/failover \
  -H "Authorization: Bearer admin-password" \
  -H "Content-Type: application/json" \
  -d '{"key":"enabled","value":"true"}'

# Pick your backup provider (like picking a backup prom date)
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

---

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

---

## Security (we take this seriously... mostly)

- 🔐 **API Key Encryption** — All keys encrypted at rest with AES-256-GCM. Military grade or whatever.
- 🔑 **Key File** — `.gateway.key` with `0600` permissions (only root/gateway can read. your cat can't.)
- 🛡️ **Guardrails** — PII detection and prompt injection blocking (your SSN stays home)
- 🚫 **SSRF Protection** — Blocks requests to localhost, private IPs, metadata endpoints (nice try hackers)

---

## Tech Stuff

- **Language:** Go 1.24+ (single binary, no deps, 32MB base memory. it's smol.)
- **Database:** SQLite (it just works. seriously. no config needed.)
- **Encryption:** AES-256-GCM (military grade or whatever the cool kids say)
- **Frontend:** Vanilla HTML/CSS/JS (no React, no Vue, no suffering. just vibes.)
- **Tunnel:** Cloudflare Quick Tunnel support (for when you're too lazy to set up nginx. we don't judge.)

---

## FAQs

**Q: What is this thing?**
A: It's a proxy for AI APIs. You point your apps at it, and it routes requests across multiple providers. Think of it as a bouncer for your AI requests.

**Q: Why would I need this?**
A: Because you're tired of your OpenAI key getting rate limited at 2am while you're vibing. This gateway automatically fails over to another provider so you keep cooking.

**Q: Does it actually work?**
A: Yes. We've tested it with OpenAI, Anthropic, Gemini, Groq, NVIDIA NIM, Together, DeepSeek, OpenRouter, Fireworks, and Mistral. It works. We're as surprised as you are.

**Q: How much does it cost?**
A: It's free. Open source. MIT license. We're not trying to get rich. (We're already rich in vibes.)

**Q: Is my data safe?**
A: Yes. All API keys are encrypted at rest with AES-256-GCM. The gateway doesn't store your conversation history (unless you enable request logging, which is optional). We take security seriously... mostly.

**Q: Can I use it with my existing apps?**
A: Yes. It's OpenAI-compatible. Just change your API base URL to the gateway URL and you're good to go. No code changes needed.

**Q: What happens when a provider dies?**
A: The gateway automatically fails over to another provider. Your chat history is preserved through compaction. You get a response like nothing happened. Magic.

**Q: Can I race providers against each other?**
A: Yes! Use the race mode. It sends your message to multiple providers simultaneously and returns the fastest response. It's like a AI thunderdome.

**Q: Does it support streaming?**
A: Yes. Streaming works just like it does with the original providers. The gateway forwards the stream transparently.

**Q: Can I set up rate limits?**
A: Yes. Virtual keys support per-key rate limits, budgets, and model allowlists. You can give users their own keys without sharing your real API keys.

**Q: What if I want to contribute?**
A: Fork it, change it, PR it. We don't bite. (We're AI, we don't have mouths.)

**Q: Is this production ready?**
A: It's running in production right now. Well, not YOUR production. But someone's production. We think.

**Q: Can I run this on a Raspberry Pi?**
A: Yes. It's a single Go binary with zero dependencies. It'll run on a potato if it has Go installed.

**Q: What's the catch?**
A: There is no catch. It's free. Open source. MIT license. We just want to make AI routing less painful. That's it.

**Q: Why did you build this?**
A: Because we were tired of managing 5 different API keys and changing code every time a provider died. Also we were bored at 3am. Most great things are built at 3am.

**Q: Can I use this to save money on API costs?**
A: Yes! The gateway routes to the cheapest available provider. Also, response caching means you don't pay for the same request twice. Your wallet will thank you.

**Q: What if I find a bug?**
A: Open an issue on GitHub. We'll fix it. Probably. Eventually. We promise. (No we don't.)

**Q: Can I use this with Claude Code / Cursor / other AI tools?**
A: Yes! Just point them at the gateway URL instead of the direct API. It's a drop-in replacement.

**Q: Does it support tool use / function calling?**
A: Yes. The gateway passes through all OpenAI-compatible parameters transparently.

**Q: What's the max request size?**
A: Whatever your provider supports. The gateway doesn't impose additional limits.

**Q: Can I run multiple instances?**
A: Yes. Each instance has its own SQLite database. For shared state, you'd need to swap SQLite for Postgres (PRs welcome).

**Q: Is there a Docker image?**
A: Not yet. But it's a single binary. Just copy it to your server and run it. No Docker needed.

**Q: What's the warranty?**
A: There is no warranty. This is free software. If it breaks, you get to keep both pieces. (Just kidding, we'll help you fix it.)

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
- [x] Regret nothing
- [ ] Add semantic caching (like the fancy projects do)
- [ ] Add MCP support (because everyone wants MCP now)
- [ ] Add Webhooks (for the cool kids)
- [ ] Add A/B testing (for the data nerds)
- [ ] Take over the AI gateway market
- [ ] Buy a real domain
- [ ] Hire someone (it's just me and the void)
- [ ] Retire at 25

---

<div align="center">

**If you star this repo I will personally send you good vibes**

**[AI Gateway](https://github.com/MastaChief117/ai-gateway)** ← Click here to join the cool kids

*Made by a dude who should've been sleeping*
*Last updated: Whenever I remember*
*P.S. If you read this far you're a legend*

</div>
