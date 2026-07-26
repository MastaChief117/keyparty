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

**Q: What is this?**
A: Idk I was bored at 3am and had too many API keys

**Q: Why?**
A: Because my OpenAI quota died at 2am and I was too angry to sleep

**Q: Does it work?**
A: If you're reading this README then yeah probably. The binary is in the repo. Go find it.

**Q: How much does it cost?**
A: If you give me $5 I'll call it enterprise pricing

**Q: Is my data safe?**
A: It's encrypted with AES-256-GCM. That sounds fancy right? That means it's safe. Trust me bro.

**Q: Can I use it with my existing apps?**
A: If your apps talk to OpenAI then yeah. If they talk to something else then idk figure it out.

**Q: What happens when a provider dies?**
A: It fails over to another one. Your chat doesn't die. You're welcome.

**Q: Can I race providers against each other?**
A: Yes. It's basically a AI thunderdome. Two providers enter. One provider wins.

**Q: Does it support streaming?**
A: Yes. The stream goes brrr just like it does with the original provider.

**Q: Can I set up rate limits?**
A: Yes. Virtual keys support rate limits. You can limit your friends. Or your enemies.

**Q: What if I want to contribute?**
A: Fork it. Change it. PR it. I don't bite. (I'm AI. I don't have a mouth.)

**Q: Is this production ready?**
A: It's running in production right now. Well not YOUR production. But someone's production. Maybe.

**Q: Can I run this on a Raspberry Pi?**
A: Yes. It'll run on a potato if it has Go installed. Don't test me.

**Q: What's the catch?**
A: There is no catch. It's free. Open source. MIT license. We just vibes.

**Q: Why did you build this?**
A: I was bored at 3am. That's it. That's the whole reason.

**Q: Can I use this to save money?**
A: If you think about it saving money is just making money. So yes. You're welcome.

**Q: What if I find a bug?**
A: Open an issue. I'll fix it. Probably. Eventually. No promises.

**Q: Can I use this with Claude Code / Cursor?**
A: Yes. Just point them at the gateway. It's a drop-in replacement. You're welcome.

**Q: Does it support tool use / function calling?**
A: Yes. It passes through everything. I don't discriminate.

**Q: What's the max request size?**
A: Idk. Try it and find out. Let me know how it goes.

**Q: Can I run multiple instances?**
A: Yes. But why would you. One is already enough chaos.

**Q: Is there a Docker image?**
A: Not yet. Just copy the binary to your server. Docker is overrated anyway.

**Q: What's the warranty?**
A: There is no warranty. If it breaks you get to keep both pieces. That's the deal.

**Q: Can I use this to take over the world?**
A: Technically yes. But please don't. I don't want that on my conscience.

**Q: What if I spam this with requests?**
A: I mean the providers will rate limit you. But go off I guess.

**Q: Is there a hidden easter egg?**
A: Maybe. Maybe not. You'll never know unless you read the source code. Good luck.

**Q: Can I use this with Ollama?**
A: Yes. Point it at your Ollama instance. Local AI go brrr.

**Q: What's the roadmap?**
A: Idk. I'll probably add stuff when I'm bored at 3am again. That's how this whole thing started.

**Q: Can I fork this and sell it?**
A: It's MIT license so technically yes. But also why. Just contribute back it's not hard.

**Q: How many stars until I add features?**
A: Every 10 stars I'll add a feature. That's the rule. I just made it up.

**Q: What's the meaning of life?**
A: 42. Also this gateway. Mostly this gateway though.

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
- [x] Write a README at 3am
- [ ] Add semantic caching (when I'm bored again)
- [ ] Add MCP support (because everyone wants MCP now)
- [ ] Add Webhooks (for the cool kids)
- [ ] Add A/B testing (for the data nerds)
- [ ] Take over the AI gateway market
- [ ] Buy a real domain
- [ ] Hire someone (it's just me and the void)
- [ ] Retire at 25
- [ ] Actually fix the bugs instead of adding features

---

<div align="center">

**If you star this repo I'll add a feature**

**If you don't star this repo I'll still add features because I have no self control**

**[AI Gateway](https://github.com/MastaChief117/ai-gateway)** ← Click here to join the chaos

*Made by a dude who should've been sleeping*
*Last updated: Whenever I remember*
*P.S. If you read this far you're legally obligated to star*

</div>
