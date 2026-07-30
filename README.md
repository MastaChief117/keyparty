<div align="center">

# 🔀 KeyParty

**One endpoint. Ten providers. Zero budget. Zero sleep.**

*For the broke dev juggling free tiers at 3AM, screaming into the void because OpenAI just rate-limited their demo for the 47th time.*

**[What is this](#what-is-this) • [Quick Start](#quick-start-in-60-seconds) • [The Stuff](#the-stuff) • [FAQ (it's funny)](#faqs)**

*Features: 31 • Sleep Deprived: Astronomically • Free Tier Credits: Scattered Across 5 Providers Like Confetti at a Sad Birthday Party*

</div>

---

## What is this

You're building an AI agent. You sign up for free tiers on Groq, Nvidia, DeepSeek, maybe grab an OpenAI trial like some kind of API key collector. Now you've got:

- 5 different API endpoints
- 5 different auth methods
- 5 different rate limits
- 5 different ways to get absolutely bodied when one goes down at 2AM during your demo

And you have $4 across all of them. Combined. Your entire AI budget is less than a gas station coffee.

**This gateway solves all of that.** You point ONE endpoint at it, and it:

- Routes to whichever provider is currently breathing (they take turns dying, it's like a rotating door of failure)
- Rotates across your keys automatically (like a DJ but for API keys, and the music is screaming)
- Fails over to another provider when one dies (your chat history survives because we're not complete monsters)
- **Compacts your chat history** to fit within cheaper provider limits when quota dies (basically AI summarization therapy)
- Encrypts everything at rest so nobody can steal your keys (we take security more seriously than you take your sleep schedule)
- Blocks prompt injection and PII from leaking (your SSN stays home, unlike your code at 3AM)
- Tracks every penny you spend across all providers (painful but necessary, like a financial advisor but for API calls)
- Lets you race providers, roast each other, rap battle, and more (because why not make infrastructure fun?)

It's like a load balancer but for AI. And it has a dashboard. Because everything needs a dashboard. Even your toaster probably needs one at this point.

Most API gateways are built for enterprises with Kubernetes clusters and DevOps teams who actually sleep. **This one is built for the dev who has $4, a dream, and questionable life choices.**

---

## Quick Start

```bash
# Clone the thing (this is the easy part, buckle up)
git clone https://github.com/MastaChief117/keyparty.git
cd keyparty

# Run the setup script (auto-installs Go + cloudflared if missing)
# It does more work than you've done all week
chmod +x keyparty.sh
./keyparty.sh
```

Dashboard: **http://localhost:8080**

That's it. No Docker. No Kubernetes. No npm install. No sacrifice to the JavaScript gods. The script handles Go, builds the binary, and optionally sets up a Cloudflare tunnel. You're welcome.

### Manual build (if you already have Go and enjoy suffering)

```bash
go build -o keyparty .
./keyparty -port 8080 -admin-pass your-password-here
# Congratulations, you just built an enterprise-grade AI gateway
# in less time than it takes to npm install node_modules
```

---

## The Stuff

### Core Features

| Feature | What it does | Why you care |
|---------|-------------|-------------|
| 🔀 **Multi-Provider Routing** | OpenAI, Anthropic, Gemini, Groq, NVIDIA, Together, DeepSeek, OpenRouter, Fireworks, Mistral — one endpoint | Stop rewriting your code every time a provider decides to take a nap |
| 🔄 **Round-Robin Rotation** | Spreads requests across multiple keys per provider | Your 3 Groq free tier keys finally work together instead of fighting like divorced parents |
| 💥 **Compaction Failover** | When quota dies (402), summarizes your chat history and sends it to a backup provider | Your $4 just became $40 worth of demo magic |
| ⚡ **Provider Race Mode** | Sends your message to multiple providers simultaneously, returns the fastest | Find out which free tier actually delivers (hint: it's not the one you think) |
| 🛡️ **Guardrails** | PII detection, prompt injection blocking, custom regex rules | Your SSN stays home. Your credit card too. |
| 🔑 **Virtual Keys** | Give users their own API keys with budgets, rate limits, and model allowlists | Share without sharing your real keys, like a responsible adult for once |
| 🏷️ **Model Aliases** | Map `smart` → `claude-sonnet-4-5`, `fast` → `gpt-4o-mini` | Sound smart without knowing model names. Fake it till you make it. |
| 🔒 **Encryption at Rest** | AES-256-GCM encryption for all API keys | Your keys are safe. We promise. More or less. |
| 📊 **Request Logging** | Full audit trail with token usage and cost tracking | Know where your money goes. Spoiler: it goes to GPU manufacturers. |
| 💰 **Cost Tracking** | Per-key and per-provider cost estimation | Your wallet will thank you. Or cry. Probably cry. |
| 📦 **Response Caching** | SHA-256 deduplication. Same request? Cached. | Your rate limits thank you. Your dignity doesn't. |
| 🎛️ **Admin Dashboard** | Web UI with tabs for everything. Dark mode. | Because light mode is a war crime and we all know it |

### Fun Features

| Feature | What it does | Vibe |
|---------|-------------|------|
| 🥊 **AI Rumble** | Two providers roast each other in real-time via SSE streaming | Boxing match but for LLMs. Popcorn not included. |
| 🎤 **Rap Battle** | Two AIs drop bars with context memory across rounds | 8 Mile but make it AI. Eminem is shook. |
| 🎰 **Model Roulette** | Random provider picks your message | Surprise me mode. Embrace the chaos. |
| 🌊 **Vibe Check** | Send a message, get a random vibe rating | Important metrics for important people |
| 🧠 **AI Therapist** | Sarcastic therapy for when your code won't compile | We've all been there. Some of us never left. |
| 🔥 **Roast My Logs** | AI roasts your API usage patterns | Painful but funny. Like looking in a mirror. |
| 🗳️ **AI Poll** | Same prompt to multiple providers, compare answers | Democratic AI. The future is here and it's petty. |

### Pro Tools

| Feature | What it does | Power level |
|---------|-------------|-------------|
| 📊 **VK Usage Dashboard** | Per virtual key usage breakdown with cost, latency, errors | Know who's spending what. Surveillance capitalism but wholesome. |
| 🔄 **Auto-Rotate Keys** | Pick the healthiest key per provider automatically | Never hit a dead key again. Dead keys are so last season. |
| 🧪 **Chat Playground** | Full chat UI with SSE streaming, provider/model picker | Test prompts without leaving the dashboard. Efficiency peak. |
| 📋 **Weekly Recap** | Stats summary + AI-generated snarky reports | Your usage, but make it funny. And slightly depressing. |
| 📈 **Cost Analytics** | Cost by provider/day/model with CSS bar charts | See where your credits go. Into the void, mostly. |
| 🪝 **Webhooks** | Notify URLs on events (request, error, failover) | Get alerted when things break. Because they will. |
| 📝 **Prompt Templates** | Save and manage system prompts | Reuse without copy-pasting like a caveman |
| ⏱️ **Rate Limit Tiers** | Tier-based rate limiting management (free/premium) | Give friends different access levels. Favoritism, but documented. |
| 💰 **Budget Alerts** | Threshold alerts per virtual key | Don't let anyone burn the group's credits. Again. |
| 🔍 **Search Logs** | Filter by provider/model/status/virtual key | Find that one request that broke everything. It's always the last one. |

---

## The Failover Thing (it's cool ok, let me explain)

Here's how compaction failover works, in case you skipped the other section like you skip your bedtime:

1. You send a request to OpenAI
2. OpenAI says "lol you're out of credits" (HTTP 402) — classic OpenAI behavior
3. Gateway goes "no worries bro" and summarizes your entire chat history like a loyal friend
4. Sends that summary to Groq (or whatever backup you configured — we don't judge)
5. You get a response like nothing happened. Magic? No. Engineering.
6. Response has `X-Gateway-Failover: true` header so you know it happened (we're transparent like that)
7. You feel like a genius for setting this up. You are. Briefly.

**Your conversation survives.** Even if the provider dies. Even if the provider has a existential crisis. Even if the provider decides to become a poet instead of an AI. That's the whole point.

Configure it in the dashboard or via API (if you enjoy typing curl commands like some kind of CLI wizard):

```bash
# Enable failover (because you're not a quitter)
curl -X POST http://localhost:8080/admin/failover \
  -H "Authorization: Bearer admin-password" \
  -H "Content-Type: application/json" \
  -d '{"key":"enabled","value":"true"}'

# Pick your backup provider (choose wisely, or don't, it's fine)
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

### Chat Completions (OpenAI-compatible, obviously, because we're not savages)

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer YOUR_UNIFIED_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o",
    "messages": [{"role": "user", "content": "say hi"}]
  }'
# Yes, it's that simple. No, there's no catch.
# Yes, your coworkers will think you're a wizard.
```

### Provider Race Mode (who's fastest? let them fight to the death)

```bash
curl http://localhost:8080/admin/race \
  -H "Authorization: Bearer YOUR_ADMIN_PASSWORD" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Hello, which model are you?",
    "providers": ["groq", "nvidia"]
  }'
# May the fastest provider win. The others get participation trophies.
```

### AI Rumble (roast battle, aka AI Thunderdome)

```bash
curl http://localhost:8080/admin/rumble \
  -H "Authorization: Bearer YOUR_ADMIN_PASSWORD" \
  -H "Content-Type: application/json" \
  -d '{"rounds": 5}' | stream
# Two providers enter. One provider leaves.
# The other one writes a sad blog post about it.
```

### Model Roulette (surprise me, I live life on the edge)

```bash
curl http://localhost:8080/admin/roulette \
  -H "Authorization: Bearer YOUR_ADMIN_PASSWORD" \
  -H "Content-Type: application/json" \
  -d '{"message": "Say something chaotic"}'
# The provider picks you. Like the Hunger Games but with APIs.
```

### All Admin Endpoints (there's a lot, buckle up)

| Method | Endpoint | What it does |
|--------|----------|-------------|
| GET | `/admin/unified-key` | Get your unified API key (the one key to rule them all) |
| POST | `/admin/unified-key` | Regenerate it (for when you inevitably lose it) |
| GET/POST | `/admin/keys` | List / Add provider API keys (your key collection grows) |
| DELETE | `/admin/keys/:id` | Delete a key (bye bye, you served well) |
| POST | `/admin/keys/:id/toggle` | Enable/disable a key (the on/off switch of destiny) |
| GET/POST | `/admin/failover` | Failover config (because hope is not a strategy) |
| GET | `/admin/failover/logs` | See when failover happened (the receipts) |
| GET/POST | `/admin/virtual-keys` | Consumer-facing keys (for your friends who keep asking) |
| GET/POST | `/admin/guardrails` | Guardrail rules (because someone has to be the adult) |
| GET/POST | `/admin/aliases` | Model aliases (sound smart without trying) |
| GET | `/admin/logs` | Request logs (the history books) |
| GET | `/admin/stats` | Dashboard stats (the scoreboard) |
| POST | `/admin/race` | Provider race (may the fastest win) |
| POST | `/admin/rumble` | AI Rumble (SSE) (popcorn required) |
| POST | `/admin/rap-battle` | Rap Battle (SSE) (Eminem is writing notes) |
| POST | `/admin/roulette` | Model Roulette (embrace the chaos) |
| POST | `/admin/roast-logs` | Roast my logs (pain, but funny pain) |
| POST | `/admin/therapist` | AI Therapist (for when your code won't compile) |
| POST | `/admin/vibe-check` | Vibe Check (important metrics) |
| POST | `/admin/poll` | AI Poll (democratic AI, the future is petty) |
| GET/POST | `/admin/webhooks` | Webhook management (notifications go brrr) |
| POST | `/admin/webhooks/test` | Test webhooks (because you don't trust yourself) |
| GET | `/admin/logs/search` | Search logs with filters (find that one request) |
| GET/POST | `/admin/templates` | Prompt templates (stop copy-pasting, you animal) |
| GET/POST | `/admin/rate-tiers` | Rate limit tiers (favoritism, documented) |
| GET/POST | `/admin/budget-alerts` | Budget alerts (don't let anyone burn the credits) |
| GET | `/admin/budget-alerts/check` | Check triggered alerts (the damage report) |
| GET | `/admin/vk-usage` | Virtual key usage (who's been naughty) |
| POST | `/admin/auto-rotate` | Pick healthiest key (survival of the fittest) |
| GET | `/admin/auto-rotate/status` | Provider health status (the vital signs) |
| POST | `/admin/playground` | Chat playground (toys for big kids) |
| GET | `/admin/recap` | Weekly recap data (the highlight reel) |
| POST | `/admin/recap/generate` | AI-generated recap (because you can't remember) |
| GET | `/admin/analytics` | Cost analytics (the financial report you didn't ask for) |

---

## Config Stuff

### CLI Flags

| Flag | Default | What it does |
|------|---------|-------------|
| `-port` | `8080` | Server port (change if 8080 is taken, you rebel) |
| `-admin-pass` | (required) | Admin dashboard password (don't use "password123", we believe in you) |
| `-db` | `gateway.db` | SQLite database path (it's a file, not a mystery) |
| `-cache-ttl` | `60` | Cache TTL in minutes (how long we remember things) |
| `-cors-origin` | (empty) | Allowed CORS origin (empty = allow all, because we're chill) |

### Environment Variables

| Variable | What it does |
|----------|-------------|
| `ADMIN_PASSWORD` | Admin password (alternative to `-admin-pass`, for the env var enthusiasts) |

---

## Security

We take this seriously — not "trust me bro" seriously, actually seriously. Like, we-wrote-tests-and-did-a-full-audit seriously. Here's what's real:

### What we do

| Protection | Implementation |
|-----------|----------------|
| 🔐 **Encryption at Rest** | AES-256-GCM via `golang.org/x/crypto`. All API keys encrypted in SQLite. Decrypted only in memory during request handling. Errors propagated, never silently ignored. Unlike your ex's promises. |
| 🔑 **Key File** | `.gateway.key` stored with `0600` permissions. Key is a 256-bit random value generated on first run. Lost key = encrypted data is gone forever. There's no "forgot password" link. |
| 🛡️ **Guardrails** | PII detection (SSN, email, phone), prompt injection blocking via regex patterns, custom rules configurable per-deployment. Your secrets stay secret. |
| 🚫 **SSRF Protection** | Uses `net.ParseIP()` with `IsLoopback()`, `IsPrivate()`, `IsLinkLocalUnicast()`, `IsUnspecified()` — not string matching like some kind of amateur. Blocks IPv6-mapped (`[::ffff:127.0.0.1]`), decimal IPs (`2130706433`), hex IPs, and cloud metadata endpoints. We caught the bypasses before they caught us. |
| 🔒 **Admin Auth** | Bearer token auth on all `/admin/*` endpoints. **Refuses to start without a password** (returns 503). It's like a bouncer that actually does its job. |
| 🕐 **Rate Limiting** | Per-IP brute-force protection on admin auth (5 attempts/min lockout). Try to brute-force us. We dare you. |
| 📊 **Request Logging** | Full audit trail — who requested what, when, which provider, cost, latency. We remember everything. Like elephants, but with better data. |
| 🧠 **Panic Recovery** | All handlers wrapped in recovery middleware. One bad request won't crash the server. Unlike your last deployment. |
| 🔒 **Cache Isolation** | Response cache includes virtual key ID in hash. Different users can't see each other's cached responses. Privacy matters, even in caching. |
| 💰 **Budget Enforcement** | Atomic budget deduction via SQL `WHERE` guard. No race conditions on concurrent requests. Your credits are safe. Probably. |
| 🛡️ **CORS** | Defaults to deny when no origin configured. `Vary: Origin` header set correctly. We're not careless. |
| 📦 **Connection Pooling** | Custom HTTP transport with 100 max idle connections, 20 per host. No connection exhaustion. We manage resources better than you manage your sleep schedule. |

### Threat Model

**This gateway is designed for:**
- Solo devs or small teams (< 5 people who actually talk to each other)
- Self-hosted on a single machine (not a Kubernetes cluster, you overachiever)
- Trusted network environment (home, personal VPS, that coffee shop WiFi you shouldn't trust)
- Non-critical workloads (prototyping, personal projects, hackathons where you pretend to sleep)

**This gateway is NOT designed for:**
- Public-facing production with untrusted users (we're not that brave)
- Enterprises needing SOC2/HIPAA compliance (go talk to the big boys)
- Multi-tenant SaaS with strict isolation requirements (that's a different beast)
- Environments where the host machine is untrusted (if you don't trust your own machine, you have bigger problems)

**Known limitations:**
- SQLite is not designed for high-concurrency writes. If you hit 100+ concurrent requests, you'll see contention. It's a feature, not a bug. (It's a bug.)
- No TLS termination — use a reverse proxy (nginx, Caddy, Cloudflare Tunnel) for HTTPS. We're not savages, but we're also not your HTTPS provider.
- No authentication on the `/v1/chat/completions` endpoint unless you create virtual keys. Anyone with the unified key can use it. Guard it with your life.
- The admin password is passed as a CLI argument or env var. It's visible in `ps` output. Use a reverse proxy with its own auth if this matters. We're honest about our flaws.
- No audit log tamper protection. If someone gets root, they can modify the SQLite database. If someone gets root, you have bigger problems than us.

### Audit Status

- **31 vulnerabilities found and fixed** in security audits (7 critical, 6 high, 8 medium, 6 low). We're basically bug bounty hunters but for our own code.
- **SSRF blocklist upgraded** from naive string prefix matching to proper `net.ParseIP()` validation. IPv6-mapped, decimal, and hex IP bypasses now blocked. We learned from our mistakes.
- **Brute-force protection** added to admin auth. 5 failed attempts = 5 minute lockout. Try harder next time.
- **Crypto primitives are standard** — AES-256-GCM, well-studied and used by the Go standard library. We didn't invent math.
- **No custom crypto** — we use `golang.org/x/crypto`, not hand-rolled anything. Because hand-rolling crypto is how you end up on Hacker News for the wrong reasons.
- **Known remaining limitations:** SQLite contention at high concurrency, no TLS (use reverse proxy), no audit log tamper protection. We're working on it. Probably. Eventually.
- **If you find a vulnerability**, open a GitHub issue. Please don't tweet about it first. We have feelings.

---

## Operational Requirements

"But it's just a binary, right?" Yeah. And so is a nuclear warhead. Here's what you need to actually run this thing without everything catching fire:

### Minimum Requirements

- **OS:** Linux, macOS, Windows (any OS that can run Go, so basically all of them)
- **RAM:** 13MB idle, ~33MB under load (lighter than your average Electron app, which is a low bar but still)
- **Disk:** ~50MB for binary + database grows with usage (~1KB per request logged, your disk will survive)
- **Go:** 1.24+ (only for building, not running — you don't need Go installed to use it, you just need it to build it, like a responsible adult)

### What You Need to Manage

| Task | Frequency | Difficulty |
|------|-----------|-----------|
| **Key rotation** | When a provider revokes/expires a key (they do this to mess with you) | Manual (dashboard or API) |
| **Database backups** | Weekly if you care about logs | `cp gateway.db gateway.db.bak` (it's really that simple) |
| **Monitoring** | Ongoing (like your anxiety) | Check `/admin/stats` or set up webhooks |
| **Log rotation** | Monthly (the request log grows, like your todo list) | Delete old entries via API or SQLite |
| **Updates** | When we ship new features (which is often because we have no self-control) | `git pull && go build` |
| **Provider management** | When free tiers change (which is constantly, because chaos) | Update keys, adjust failover config |

### Production Tips (if you're feeling brave)

1. **Put it behind a reverse proxy.** Nginx, Caddy, or Cloudflare Tunnel. Don't expose port 8080 directly. That's how you end up on "Today in Bad Security Decisions."
2. **Use HTTPS.** The gateway doesn't do TLS. Your reverse proxy does. We're a gateway, not a certificate authority.
3. **Set up webhooks** for error alerts so you know when providers die. Because they will. At the worst possible time.
4. **Back up `.gateway.key` somewhere safe.** Lose it = lose all encrypted keys. There is no recovery. None. Zero. Zilch.
5. **Don't run as root** if you can avoid it. The gateway doesn't need elevated privileges. Neither do you, probably.
6. **Use virtual keys** instead of sharing your unified key. You can revoke them individually. It's like having separate keys for different doors, except the doors are API endpoints.

### What "Production" Means Here

This is a **self-hosted tool for individuals and small teams who enjoy living on the edge.** If you're deploying this for >10 users or handling sensitive data, you should:

- Conduct your own security review (we did ours, now do yours)
- Set up proper monitoring and alerting (because you can't fix what you can't see)
- Use a managed database if SQLite doesn't fit your concurrency needs (SQLite is great, but it has limits, like all of us)
- Add TLS termination via reverse proxy (we're a gateway, not a security suite)
- Consider whether you need compliance certifications (if you're asking, you probably do)

**We're honest about what this is: a great tool for personal use and small teams who appreciate good engineering with a side of chaos.** If you need enterprise-grade everything, there are commercial alternatives. We're not trying to be them. We're trying to be better. For less money. With more jokes.

---

## Tech Stuff

- **Language:** Go 1.24+ (single binary, no deps, 13MB idle RAM. it's smol. like, adorably smol.)
- **Database:** SQLite (it just works. no config needed. the Honda Civic of databases.)
- **Encryption:** AES-256-GCM (military grade or whatever. your keys are safer than your browser history.)
- **Frontend:** Vanilla HTML/CSS/JS (no React, no Vue, no suffering. just vibes. pure, unfiltered vibes.)
- **Tunnel:** Cloudflare Quick Tunnel support (for when you're too lazy to set up nginx, which is always)

---

## Performance (we actually tested it, unlike some people)

We stress tested it — **16,000 requests, zero provider calls, zero credits burned.** Requests fail at the gateway level when no keys are configured. It's like a simulator for disappointment.

### Binary

- **On disk:** 16MB (smaller than your average selfie)
- **Threads:** 5 (more than your average Twitter argument)

### RAM Usage

| Stage | RSS (actual RAM) | Virtual Size |
|-------|-----------------|--------------|
| Fresh start (idle) | **13MB** | 1.9GB |
| After 1,000 requests | **30MB** | 2.4GB |
| After 6,000 requests | **33MB** | 2.8GB |
| After 16,000 requests | **33MB** | 3.2GB |

> RSS = what your OS actually allocates. Virtual size includes Go runtime mappings (not real memory, like your excuses). Memory stabilizes at ~33MB — no leaks. Unlike your code. Sorry.

### Throughput

| Concurrent | Requests | RPS |
|-----------|----------|-----|
| 50 | 1,000 | **218** |
| 100 | 5,000 | **234** |
| 100 | 10,000 | **222** |

Consistent ~220 RPS with fast-failing requests. With real providers, RPS depends on provider latency — the gateway adds negligible overhead. We're fast. Faster than your last deployment, at least.

### Database

- Request logging writes ~100 bytes per entry (smaller than your average tweet)
- DB stays at 88KB when requests fail before logging (no provider calls = no logs, like a tree falling in an empty forest)
- With real traffic, expect ~100KB per 1,000 logged requests (your disk will survive, we promise)

### TL;DR

- **Idle:** 13MB RAM — lighter than most Electron apps, which is a low bar but we'll take it
- **Under load:** ~33MB RAM, stabilizes — no memory leaks (unlike your last project)
- **Throughput:** ~220 RPS (gateway overhead only, the real bottleneck is your internet)
- Runs on a potato 🥔 (seriously, try it)

---

## Who this is for

- Solo devs building AI agents on free tiers (the brave ones)
- Small teams sharing API credits (the generous ones)
- Hackathon projects that need multi-provider resilience (the sleep-deprived ones)
- Students learning AI/LLM integration (the curious ones)
- Anyone tired of managing 5 different AI API dashboards (the frustrated ones)
- People who think infrastructure can be fun (the optimistic ones)
- The broke dev with $4 across three providers (the relatable ones)

## Who this is NOT for

- Enterprises needing horizontal scaling (it's a single binary, not a distributed system, you overachiever)
- Teams needing RBAC, SSO, or audit logs (go talk to the big boys with their fancy compliance requirements)
- Anyone processing millions of requests (SQLite has limits, like all of us)
- People who want a plugin system (no Lua/Go extensions yet, but we're working on it... probably)
- gRPC-only setups (HTTP only, we're not that fancy)

---

## FAQs

**Q: What is this?**
A: An AI gateway for broke devs. One endpoint, ten providers, zero budget. Like a Swiss Army knife but for APIs, and the knife is made of hopes and dreams.

**Q: Why?**
A: Because my OpenAI quota died at 2am and I was too angry to sleep. This project is basically revenge against rate limits.

**Q: Does it work?**
A: If you're reading this README then yeah probably. We tested it. We think. Memory is fuzzy at 3AM.

**Q: How much does it cost?**
A: If you give me $5 I'll call it enterprise pricing. For real though, it's free. MIT license. We're not greedy, just tired.

**Q: Is my data safe?**
A: AES-256-GCM encryption at rest. No custom crypto. Standard libraries. No formal audit yet (we're working on it). Read the [Security section](#security) for the full threat model. We're honest about what this is, unlike some dating profiles.

**Q: Can I use it with my existing apps?**
A: If your apps talk to OpenAI then yeah. It's a drop-in replacement. Like swapping regular coffee for decaf, except the coffee still works.

**Q: What happens when a provider dies?**
A: It fails over to another one. Your chat doesn't die. You're welcome. We're basically life support for your conversations.

**Q: Can I race providers against each other?**
A: Yes. It's basically an AI thunderdome. Two providers enter, one provider leaves. The other one writes a sad blog post about it.

**Q: Can two AIs roast each other?**
A: Yes. AI Rumble and Rap Battle are built in. It's exactly as chaotic as it sounds. We're not sorry.

**Q: Does it support streaming?**
A: Yes. The stream goes brrr just like it does with the original provider. We didn't break anything. Probably.

**Q: Can I set up rate limits?**
A: Yes. Virtual keys support rate limits and tiers. You can limit your friends. Or your enemies. Or both, if your friends are also your enemies.

**Q: Can I track how much I'm spending?**
A: Yes. Cost analytics, per-provider breakdowns, per-virtual-key usage, budget alerts. Your wallet will thank you. Or cry. Probably cry. But at least you'll know why.

**Q: Can I use this with Claude Code / Cursor?**
A: Yes. Just point them at the gateway. You're welcome. We're basically the middleman you never knew you needed.

**Q: Can I run this on a Raspberry Pi?**
A: Yes. It'll run on a potato if it has Go installed. Don't test me. Actually, do test me. I want to see a potato run an AI gateway.

**Q: What's the catch?**
A: There is no catch. It's free. Open source. MIT license. We just vibes. No hidden fees, no "enterprise tier," no "contact sales for pricing." Just vibes.

**Q: What if I find a bug?**
A: Open an issue. I'll fix it. Probably. Eventually. No promises. But seriously, open an issue. We actually read them. Unlike your ex's texts.

**Q: Can I contribute?**
A: Yes! Open a PR. We accept contributions. We also accept emotional support. And coffee. Mostly coffee.

**Q: Why is it called KeyParty?**
A: Because it's a party for your API keys. They finally get to hang out together instead of being locked in separate provider vaults crying alone. It's wholesome, actually.

**Q: Is this production ready?**
A: If by "production" you mean "running on my personal VPS at 3AM while I pray to the demo gods," then yes. If you mean "serving millions of requests for a Fortune 500 company," then no. Know your limits. We know ours.

---

## Roadmap

- [x] Make gateway (the hard part, apparently)
- [x] Add failover with compaction (because providers die like my motivation)
- [x] Add encryption (AES-256-GCM, military grade or whatever)
- [x] Make dashboard (dark mode, obviously)
- [x] Add race mode (may the fastest provider win)
- [x] Add guardrails (because someone has to be the adult)
- [x] Add virtual keys (sharing is caring)
- [x] Add model aliases (fake it till you make it)
- [x] Add request logging (the history books)
- [x] Add cost tracking (painful but necessary)
- [x] Add AI Rumble (roast battle, popcorn required)
- [x] Add Rap Battle (SSE streaming, Eminem is shook)
- [x] Add Model Roulette, Vibe Check, Therapist, Roast Logs (chaos features)
- [x] Add Webhooks (notifications go brrr)
- [x] Add Prompt Templates (stop copy-pasting)
- [x] Add AI Poll (democratic AI, the future is petty)
- [x] Add Rate Limit Tiers (favoritism, documented)
- [x] Add Budget Alerts (don't let anyone burn the credits)
- [x] Add VK Usage Dashboard (surveillance capitalism but wholesome)
- [x] Add Auto-Rotate Keys (survival of the fittest)
- [x] Add Chat Playground (toys for big kids)
- [x] Add Weekly Recaps (the highlight reel)
- [x] Add Cost Analytics (the financial report you didn't ask for)
- [x] Full security audit (31 vulnerabilities fixed, we're basically bug bounty hunters now)
- [x] Add keyparty.sh setup script (because manual installs are for cavemen)
- [x] Fix SSRF bypass (IPv6-mapped, decimal, hex IP encoding — we caught the bypasses before they caught us)
- [x] Add brute-force protection on admin auth (5 attempts/min lockout, try harder next time)
- [x] Fix GetKeyByID decryption for test-key endpoint (it was returning encrypted blobs, we fixed it, you're welcome)
- [ ] Add semantic caching (when I'm bored again, which is always)
- [ ] Add MCP support (because everyone wants MCP now, including your mom)
- [ ] Add A/B testing (for the data nerds who like graphs)
- [ ] Add streaming support for Rumble/Rap Battle (double SSE, maximum chaos)
- [ ] Take over the AI gateway market (one $4 coffee at a time)
- [ ] Buy a real domain (currently accepting donations in the form of stars)
- [ ] Hire someone (it's just me and the void, and the void doesn't write code)
- [ ] Retire at 25 (a man can dream)
- [ ] Actually fix the bugs instead of adding features (never gonna happen, let's be real)

---

<div align="center">

**If you star this repo I'll add a feature**

**If you don't star this repo I'll still add features because I have no self control and an unhealthy relationship with my keyboard**

**[KeyParty](https://github.com/MastaChief117/keyparty)** ← Click here to join the chaos. We have cookies. And rate limits.

*Made by a dude who should've been sleeping. Again. For the 47th night in a row.*
*Last updated: Whenever I remember, which is basically never*
*P.S. If you read this far you're legally obligated to star. It's in the fine print. Somewhere. Probably.*

*P.P.S. If this README made you smile, star it. If it made you cry, star it harder.*

</div>
