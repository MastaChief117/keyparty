<div align="center">

# 🔀 KeyParty

**One endpoint. Ten providers. Zero budget.**

*For the broke dev juggling free tiers at 3AM, praying one of them doesn't rate-limit them mid-demo.*

**[What is this](#what-is-this) • [Quick Start](#quick-start-in-60-seconds) • [The Stuff](#the-stuff) • [FAQ (it's funny)](#faqs)**

*Features: 28+ • Sleep Deprived: Immensely • Free Tier Credits: Scattered Across 5 Providers Like Confetti*

</div>

---

## What is this

You're building an AI agent. You sign up for free tiers on Groq, Nvidia, DeepSeek, maybe grab an OpenAI trial. Now you've got:

- 5 different API endpoints
- 5 different auth methods
- 5 different rate limits
- 5 different ways to get screwed when one goes down
- 5 different dashboards you'll never open because who has time

And you have $4 across all of them. Combined. Your coffee budget is higher.

**This gateway solves all of that.** You point ONE endpoint at it, and it:

- Routes to whichever provider is breathing (they take turns dying, it's like musical chairs but for uptime)
- Rotates across your keys automatically (like a DJ but for API keys, and the music never stops because it's just HTTP requests)
- Fails over to another provider when one dies (your chat history survives because we're not monsters)
- **Compacts your chat history** to fit within cheaper provider limits when quota dies (yes it really does this, no I don't know how it took me 3 weeks)
- Encrypts everything at rest so nobody can steal your keys (we take security more seriously than our sleep schedule)
- Blocks prompt injection and PII from leaking (your SSN stays home, you're welcome)
- Tracks every penny you spend across all providers (your wallet will cry tears of joy)
- Lets you race providers, roast each other, rap battle, and more (because why not)

It's like a load balancer but for AI. And it has a dashboard. Because everything needs a dashboard. Even your toaster probably needs one at this point.

Most API gateways are built for enterprises with Kubernetes clusters and DevOps teams. **This one is built for the dev who has $4 and a dream.** And maybe a potato for a server.

---

## Quick Start

```bash
# Clone the thing (like a normal human)
git clone https://github.com/MastaChief117/keyparty.git
cd keyparty

# Run the setup script (it auto-installs Go + cloudflared if you're missing them)
chmod +x keyparty.sh
./keyparty.sh
```

Dashboard: **http://localhost:8080**

That's it. No Docker. No Kubernetes. No npm install. No "why is my node_modules folder bigger than my entire OS." The script handles Go, builds the binary, and optionally sets up a Cloudflare tunnel. You're welcome.

### Manual build (if you already have Go and enjoy doing things the hard way)

```bash
go build -o keyparty .
./keyparty -port 8080 -admin-pass your-password-here
# Congratulations, you're now an infrastructure engineer
```

---

## The Stuff

### Core Features

| Feature | What it does | Why you care |
|---------|-------------|-------------|
| 🔀 **Multi-Provider Routing** | OpenAI, Anthropic, Gemini, Groq, NVIDIA, Together, DeepSeek, OpenRouter, Fireworks, Mistral — one endpoint | Stop changing your code when a provider dies like it's a seasonal thing |
| 🔄 **Round-Robin Rotation** | Spreads requests across multiple keys per provider | Your 3 Groq free tier keys finally work together instead of fighting |
| 💥 **Compaction Failover** | When quota dies (402), summarizes your chat history and sends it to a backup provider | Your $4 just got you through a demo that would've cost $20. You're welcome. |
| ⚡ **Provider Race Mode** | Sends your message to multiple providers simultaneously, returns the fastest | Find out which free tier is actually fastest (spoiler: it's never the one you expect) |
| 🛡️ **Guardrails** | PII detection, prompt injection blocking, custom regex rules | Your SSN stays home. Your credit card stays home. Your dignity? That's on you. |
| 🔑 **Virtual Keys** | Give users their own API keys with budgets, rate limits, and model allowlists | Share without sharing your real keys. Trust issues? We got you. |
| 🏷️ **Model Aliases** | Map `smart` → `claude-sonnet-4-5`, `fast` → `gpt-4o-mini` | Sound smart without knowing model names. Fake it till you make it. |
| 🔒 **Encryption at Rest** | AES-256-GCM encryption for all API keys | Your keys are safe. We promise. Pinky swear. |
| 📊 **Request Logging** | Full audit trail with token usage and cost tracking | Know where your money goes. (Spoiler: it goes to AI providers.) |
| 💰 **Cost Tracking** | Per-key and per-provider cost estimation | Your wallet will thank you. Or cry. Probably cry. |
| 📦 **Response Caching** | SHA-256 deduplication. Same request? Cached. | Your rate limits thank you. Your wallet thanks you. Everyone thanks you. |
| 🎛️ **Admin Dashboard** | Web UI with tabs for everything. Dark mode. | Because light mode is a war crime and we all know it |

### Fun Features (yes these are real)

| Feature | What it does | Vibe |
|---------|-------------|------|
| 🥊 **AI Rumble** | Two providers roast each other in real-time via SSE streaming | Boxing match but for LLMs. Popcorn not included. |
| 🎤 **Rap Battle** | Two AIs drop bars with context memory across rounds | 8 Mile but make it AI. Eminem is rolling in his grave (he's not dead but still). |
| 🎰 **Model Roulette** | Random provider picks your message | Surprise me mode. For when you want chaos. |
| 🌊 **Vibe Check** | Send a message, get a random vibe rating | Important metrics. Very scientific. Trust the process. |
| 🧠 **AI Therapist** | Sarcastic therapy for when your code won't compile | We've all been there. Some of us never left. |
| 🔥 **Roast My Logs** | AI roasts your API usage patterns | Painful but funny. Like stepping on a Lego but for your ego. |
| 🗳️ **AI Poll** | Same prompt to multiple providers, compare answers | Democratic AI. Let the models vote on who's right. |

### Pro Tools (for when you get serious)

| Feature | What it does | Power level |
|---------|-------------|-------------|
| 📊 **VK Usage Dashboard** | Per virtual key usage breakdown with cost, latency, errors | Know who's spending what. Blame someone with data. |
| 🔄 **Auto-Rotate Keys** | Pick the healthiest key per provider automatically | Never hit a dead key again. It's like a health check but for your wallet. |
| 🧪 **Chat Playground** | Full chat UI with SSE streaming, provider/model picker | Test prompts without leaving the dashboard. Efficiency! |
| 📋 **Weekly Recap** | Stats summary + AI-generated snarky reports | Your usage, but make it funny. The AI judges you. |
| 📈 **Cost Analytics** | Cost by provider/day/model with CSS bar charts | See where your credits go. (They go to AI providers. We already told you.) |
| 🪝 **Webhooks** | Notify URLs on events (request, error, failover) | Get alerted when things break. Because you need more notifications in your life. |
| 📝 **Prompt Templates** | Save and manage system prompts | Reuse without copy-pasting. Because copy-pasting is how bugs happen. |
| ⏱️ **Rate Limit Tiers** | Tier-based rate limiting management (free/premium) | Give friends different access levels. Class system, but for API access. |
| 💰 **Budget Alerts** | Threshold alerts per virtual key | Don't let anyone burn the group's credits. Especially that one friend. |
| 🔍 **Search Logs** | Filter by provider/model/status/virtual key | Find that one request that broke everything. It's always the last one you check. |

---

## The Failover Thing (it's cool ok, trust me)

Here's how compaction failover works:

1. You send a request to OpenAI
2. OpenAI says "lol you're out of credits" (HTTP 402)
3. Gateway goes "no worries bro" and summarizes your entire chat history
4. Sends that summary to Groq (or whatever backup you configured)
5. You get a response like nothing happened
6. Response has `X-Gateway-Failover: true` header so you know it happened
7. You feel like a genius for setting this up
8. You tell everyone about it at the next meetup
9. They don't care but you do

**Your conversation survives.** Even if the provider dies. Even if the provider goes to war. Even if the provider decides to become a potato. That's the whole point.

Configure it in the dashboard or via API (we won't judge either way):

```bash
# Enable failover (finally, some peace of mind)
curl -X POST http://localhost:8080/admin/failover \
  -H "Authorization: Bearer admin-password" \
  -H "Content-Type: application/json" \
  -d '{"key":"enabled","value":"true"}'

# Pick your backup provider (because every hero needs a sidekick)
curl -X POST http://localhost:8080/admin/failover \
  -H "Authorization: Bearer admin-password" \
  -H "Content-Type: application/json" \
  -d '{"key":"provider","value":"groq"}'

# Pick the model (because choices are hard)
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
# It works. Youre welcome. Now go build something cool.
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
# May the fastest provider win. The others get participation trophies.
```

### AI Rumble (roast battle)

```bash
curl http://localhost:8080/admin/rumble \
  -H "Authorization: Bearer YOUR_ADMIN_PASSWORD" \
  -H "Content-Type: application/json" \
  -d '{"rounds": 5}' | stream
# Two AIs enter. One AI leaves. The other writes a blog post about the experience.
```

### Model Roulette (surprise me)

```bash
curl http://localhost:8080/admin/roulette \
  -H "Authorization: Bearer YOUR_ADMIN_PASSWORD" \
  -H "Content-Type: application/json" \
  -d '{"message": "Say something chaotic"}'
# You don't pick the model. The model picks you. It's like the Sorting Hat but for AI.
```

### All Admin Endpoints (there's a lot, buckle up)

| Method | Endpoint | What it does |
|--------|----------|-------------|
| GET | `/admin/unified-key` | Get your unified API key (the one key to rule them all) |
| POST | `/admin/unified-key` | Regenerate it (because you lost it, didn't you) |
| GET/POST | `/admin/keys` | List / Add provider API keys (the real treasures) |
| DELETE | `/admin/keys/:id` | Delete a key (goodbye, old friend) |
| POST | `/admin/keys/:id/toggle` | Enable/disable a key (sleep mode for API keys) |
| GET/POST | `/admin/failover` | Failover config (because hope is not a strategy) |
| GET | `/admin/failover/logs` | See when failover happened (the receipts) |
| GET/POST | `/admin/virtual-keys` | Consumer-facing keys (for your friends who keep asking) |
| GET/POST | `/admin/guardrails` | Guardrail rules (because someone has to be the adult) |
| GET/POST | `/admin/aliases` | Model aliases (so you can sound smart) |
| GET | `/admin/logs` | Request logs (the truth, the whole truth, and nothing but the truth) |
| GET | `/admin/stats` | Dashboard stats (numbers go brrr) |
| POST | `/admin/race` | Provider race (may the fastest win) |
| POST | `/admin/rumble` | AI Rumble (SSE) (popcorn not included) |
| POST | `/admin/rap-battle` | Rap Battle (SSE) (8 Mile vibes) |
| POST | `/admin/roulette` | Model Roulette (surprise me) |
| POST | `/admin/roast-logs` | Roast my logs (painful but funny) |
| POST | `/admin/therapist` | AI Therapist (we've all needed this) |
| POST | `/admin/vibe-check` | Vibe Check (important metrics) |
| POST | `/admin/poll` | AI Poll (democracy!) |
| GET/POST | `/admin/webhooks` | Webhook management (notifications go brrr) |
| POST | `/admin/webhooks/test` | Test webhooks (because you need to know they work) |
| GET | `/admin/logs/search` | Search logs with filters (find that one request) |
| GET/POST | `/admin/templates` | Prompt templates (save your best prompts) |
| GET/POST | `/admin/rate-tiers` | Rate limit tiers (free vs premium, the class system) |
| GET/POST | `/admin/budget-alerts` | Budget alerts (don't let anyone burn the credits) |
| GET | `/admin/budget-alerts/check` | Check triggered alerts (the damage report) |
| GET | `/admin/vk-usage` | Virtual key usage (who's spending what) |
| POST | `/admin/auto-rotate` | Pick healthiest key (survival of the fittest) |
| GET | `/admin/auto-rotate/status` | Provider health status (who's alive, who's dead) |
| POST | `/admin/playground` | Chat playground (test without leaving the dashboard) |
| GET | `/admin/recap` | Weekly recap data (the weekly roast) |
| POST | `/admin/recap/generate` | AI-generated recap (the AI judges your usage) |
| GET | `/admin/analytics` | Cost analytics (where your money goes) |

---

## Config Stuff

### CLI Flags

| Flag | Default | What it does |
|------|---------|-------------|
| `-port` | `8080` | Server port (change if 8080 is taken, you rebel) |
| `-admin-pass` | (required) | Admin dashboard password (don't use "password123", we're watching) |
| `-db` | `gateway.db` | SQLite database path (it's a database, it stores stuff) |
| `-cache-ttl` | `60` | Cache TTL in minutes (how long to remember things) |
| `-cors-origin` | (empty) | Allowed CORS origin (empty = allow all, because we trust you) |

### Environment Variables

| Variable | What it does |
|----------|-------------|
| `ADMIN_PASSWORD` | Admin password (alternative to `-admin-pass`, for the env var enthusiasts) |

---

## Security

We take this seriously — not "trust me bro" seriously, actually seriously. Here's what's real (and what's terrifying if you think about it too hard):

### What we do (so you don't have to)

| Protection | Implementation |
|-----------|----------------|
| 🔐 **Encryption at Rest** | AES-256-GCM via `golang.org/x/crypto`. All API keys encrypted in SQLite. Decrypted only in memory during request handling. Errors propagated, never silently ignored. (Unlike your ex, we don't ignore errors.) |
| 🔑 **Key File** | `.gateway.key` stored with `0600` permissions. Key is a 256-bit random value generated on first run. Lost key = encrypted data is gone forever. No recovery. No "oops." It's gone. Like your motivation on Mondays. |
| 🛡️ **Guardrails** | PII detection (SSN, email, phone), prompt injection blocking via regex patterns, custom rules configurable per-deployment. (Your SSN stays home. Your dignity? That's on you.) |
| 🚫 **SSRF Protection** | Uses `net.ParseIP()` with `IsLoopback()`, `IsPrivate()`, `IsLinkLocalUnicast()`, `IsUnspecified()` — not string matching. Blocks IPv6-mapped (`[::ffff:127.0.0.1]`), decimal IPs (`2130706433`), hex IPs, and cloud metadata endpoints. DNS resolution checked for hostnames. (We caught someone trying to be clever. They failed.) |
| 🔒 **Admin Auth** | Bearer token auth on all `/admin/*` endpoints. **Refuses to start without a password** (returns 503). Failed attempts logged with IP. (We're watching you.) |
| 🕐 **Rate Limiting** | Per-IP brute-force protection on admin auth (5 attempts/min lockout). Per-virtual-key rate limits enforced. Visitors map capped at 10K entries. (Try harder. Or don't. We'll lock you out either way.) |
| 📊 **Request Logging** | Full audit trail — who requested what, when, which provider, cost, latency. DB write errors logged. (The truth, the whole truth, and nothing but the truth.) |
| 🧠 **Panic Recovery** | All handlers wrapped in recovery middleware. One bad request won't crash the server. (Unlike your code, this doesn't panic.) |
| 🔒 **Cache Isolation** | Response cache includes virtual key ID in hash. Different users can't see each other's cached responses. (Privacy. It's a thing.) |
| 💰 **Budget Enforcement** | Atomic budget deduction via SQL `WHERE` guard. No race conditions on concurrent requests. (Math works. We checked.) |
| 🛡️ **CORS** | Defaults to deny when no origin configured. `Vary: Origin` header set correctly. (No origin? No problem. No access either.) |
| 📦 **Connection Pooling** | Custom HTTP transport with 100 max idle connections, 20 per host. No connection exhaustion. (We pool connections like we pool our remaining brain cells.) |

### Threat Model (aka "who should worry about what")

**This gateway is designed for:**
- Solo devs or small teams (< 5 people)
- Self-hosted on a single machine
- Trusted network environment (home, personal VPS, that one coffee shop with suspicious WiFi)
- Non-critical workloads (prototyping, personal projects, hackathons, proving a point)

**This gateway is NOT designed for:**
- Public-facing production with untrusted users (go use Kong, you enterprise person)
- Enterprises needing SOC2/HIPAA compliance (we're not lawyers, we're devs)
- Multi-tenant SaaS with strict isolation requirements (that's a whole different problem)
- Environments where the host machine is untrusted (if your machine is untrusted, you have bigger problems)

**Known limitations (we're honest, unlike some providers' uptime pages):**
- SQLite is not designed for high-concurrency writes. If you hit 100+ concurrent requests, you'll see contention. (It's SQLite, not PostgreSQL. Manage your expectations.)
- No TLS termination — use a reverse proxy (nginx, Caddy, Cloudflare Tunnel) for HTTPS. (We're a gateway, not a certificate authority.)
- No authentication on the `/v1/chat/completions` endpoint unless you create virtual keys. Anyone with the unified key can use it. (Guard your key like you guard your Netflix password.)
- The admin password is passed as a CLI argument or env var. It's visible in `ps` output. Use a reverse proxy with its own auth if this matters. (We're working on it. Maybe. Eventually.)
- No audit log tamper protection. If someone gets root, they can modify the SQLite database. (If someone has root, you have bigger problems. Like, much bigger problems.)

### Audit Status (aka "how many things did we break and fix")

- **31 vulnerabilities found and fixed** in security audits (7 critical, 6 high, 8 medium, 6 low). (Yes, we found 31 ways to break our own code. We're talented like that.)
- **SSRF blocklist upgraded** from naive string prefix matching to proper `net.ParseIP()` validation. IPv6-mapped, decimal, and hex IP bypasses now blocked. (Someone tried to be clever with `[::ffff:127.0.0.1]`. We caught them. They cried.)
- **Brute-force protection** added to admin auth. 5 failed attempts = 5 minute lockout. (Try harder. Or don't. We'll lock you out either way.)
- **Crypto primitives are standard** — AES-256-GCM, well-studied and used by the Go standard library. (We don't roll our own crypto. That's how you get pwned.)
- **No custom crypto** — we use `golang.org/x/crypto`, not hand-rolled anything. (Hand-rolled crypto is how you get featured on Hacker News for the wrong reasons.)
- **Known remaining limitations:** SQLite contention at high concurrency, no TLS (use reverse proxy), no audit log tamper protection. (We're working on it. Maybe. No promises.)
- **If you find a vulnerability**, open an issue. Please don't tweet about it first. (We have feelings too, you know.)

---

## Operational Requirements

"But it's just a binary, right?" Yeah. And so is a nuclear warhead. Here's what you need to actually run this thing (and no, "just vibes" is not an answer):

### Minimum Requirements (bare minimum, like your will to live on Mondays)

- **OS:** Linux, macOS, Windows (any. even that weird BSD you use)
- **RAM:** 13MB idle, ~33MB under load (lighter than your browser with 47 tabs open)
- **Disk:** ~50MB for binary + database grows with usage (~1KB per request logged)
- **Go:** 1.24+ (only for building, not running. we're not savages.)

### What You Need to Manage (yes, you have to do things)

| Task | Frequency | Difficulty |
|------|-----------|-----------|
| **Key rotation** | When a provider revokes/expires a key | Manual (dashboard or API, your choice) |
| **Database backups** | Weekly if you care about logs | `cp gateway.db gateway.db.bak` (yes, that's it) |
| **Monitoring** | Ongoing | Check `/admin/stats` or set up webhooks (be a responsible adult) |
| **Log rotation** | Monthly | The request log grows. Delete old entries via API or SQLite. (Like spring cleaning but for data.) |
| **Updates** | When we ship new features | `git pull && go build` (the dev life cycle) |
| **Provider management** | When free tiers change | Update keys, adjust failover config (because free tiers are a myth) |

### Production Tips (for when you get serious)

1. **Put it behind a reverse proxy.** Nginx, Caddy, or Cloudflare Tunnel. Don't expose port 8080 directly. (We're serious about this. Like, really serious.)
2. **Use HTTPS.** The gateway doesn't do TLS. Your reverse proxy does. (We're a gateway, not a certificate authority.)
3. **Set up webhooks** for error alerts so you know when providers die. (Because you need more notifications in your life.)
4. **Back up `.gateway.key` somewhere safe.** Lose it = lose all encrypted keys. There is no recovery. No "oops." It's gone. (Like your motivation on Mondays.)
5. **Don't run as root** if you can avoid it. The gateway doesn't need elevated privileges. (Least privilege. It's a thing. Look it up.)
6. **Use virtual keys** instead of sharing your unified key. You can revoke them individually. (Trust issues? We got you.)

### What "Production" Means Here (aka "don't blame us")

This is a **self-hosted tool for individuals and small teams.** If you're deploying this for >10 users or handling sensitive data, you should:

- Conduct your own security review (we're not your security team)
- Set up proper monitoring and alerting (because you need to know when things break)
- Use a managed database if SQLite doesn't fit your concurrency needs (we're not against managed databases, we're just lazy)
- Add TLS termination via reverse proxy (we covered this already, but it's important)
- Consider whether you need compliance certifications (if you're asking, you probably do)

**We're honest about what this is: a great tool for personal use and small teams.** If you need enterprise-grade everything, there are commercial alternatives. We're not trying to be them. We're trying to be the dev who made something cool and shared it with the world.

---

## Tech Stuff (for the nerds)

- **Language:** Go 1.24+ (single binary, no deps, 13MB idle RAM. it's smol. like really smol.)
- **Database:** SQLite (it just works. no config needed. like a good friend.)
- **Encryption:** AES-256-GCM (military grade or whatever. we're not sure but it sounds impressive.)
- **Frontend:** Vanilla HTML/CSS/JS (no React, no Vue, no suffering. just vibes. and maybe some inline styles.)
- **Tunnel:** Cloudflare Quick Tunnel support (for when you're too lazy to set up nginx. we don't judge.)

---

## Performance (we actually tested it, unlike some people)

We stress tested it — **16,000 requests, zero provider calls, zero credits burned.** Requests fail at the gateway level when no keys are configured. (Yes, we're that thorough. Or that bored. Probably both.)

### Binary

- **On disk:** 16MB (smaller than your average node_modules folder)
- **Threads:** 5 (because we're not greedy)

### RAM Usage

| Stage | RSS (actual RAM) | Virtual Size |
|-------|-----------------|--------------|
| Fresh start (idle) | **13MB** | 1.9GB |
| After 1,000 requests | **30MB** | 2.4GB |
| After 6,000 requests | **33MB** | 2.8GB |
| After 16,000 requests | **33MB** | 3.2GB |

> RSS = what your OS actually allocates. Virtual size includes Go runtime mappings (not real memory). Memory stabilizes at ~33MB — no leaks. (Unlike your code. Just saying.)

### Throughput

| Concurrent | Requests | RPS |
|-----------|----------|-----|
| 50 | 1,000 | **218** |
| 100 | 5,000 | **234** |
| 100 | 10,000 | **222** |

Consistent ~220 RPS with fast-failing requests. With real providers, RPS depends on provider latency — the gateway adds negligible overhead. (It's fast. Like, really fast. Faster than your WiFi on a good day.)

### Database

- Request logging writes ~100 bytes per entry (tiny, like your attention span)
- DB stays at 88KB when requests fail before logging (no provider calls = no logs)
- With real traffic, expect ~100KB per 1,000 logged requests (it grows, but slowly, like your TBR list)

### TL;DR (because you're busy)

- **Idle:** 13MB RAM — lighter than most Electron apps (and way lighter than Slack)
- **Under load:** ~33MB RAM, stabilizes — no memory leaks (unlike your code)
- **Throughput:** ~220 RPS (gateway overhead only)
- Runs on a potato 🥔 (seriously, we tested it)

---

## Who this is for (aka "are you our people?")

- Solo devs building AI agents on free tiers (the broke and the brave)
- Small teams sharing API credits (because teamwork makes the dream work)
- Hackathon projects that need multi-provider resilience (because hackathons are chaos)
- Students learning AI/LLM integration (welcome to the future, it's weird here)
- Anyone tired of managing 5 different AI API dashboards (we feel you, we really do)
- People who think infrastructure can be fun (you're one of us now)
- The broke dev with $4 across three providers (you're our people)

## Who this is NOT for (aka "go use Kong")

- Enterprises needing horizontal scaling (it's a single binary, not a distributed system)
- Teams needing RBAC, SSO, or audit logs (we're working on it, maybe, eventually)
- Anyone processing millions of requests (SQLite has limits, and so do we)
- People who want a plugin system (no Lua/Go extensions yet, but we're open to ideas)
- gRPC-only setups (HTTP only, because we're not savages)

---

## FAQs (the fun part)

**Q: What is this?**
A: An AI gateway for broke devs. One endpoint, ten providers, zero budget. Like a Swiss Army knife but for AI APIs. And cheaper. Way cheaper.

**Q: Why?**
A: Because my OpenAI quota died at 2am and I was too angry to sleep. So I built this instead of being productive. Classic.

**Q: Does it work?**
A: If you're reading this README then yeah probably. Also we tested it. A lot. Like, too much. Our eyes hurt.

**Q: How much does it cost?**
A: If you give me $5 I'll call it enterprise pricing. But seriously, it's free. MIT license. We just vibes.

**Q: Is my data safe?**
A: AES-256-GCM encryption at rest. No custom crypto. Standard libraries. No formal audit yet (but we're working on it). Read the [Security section](#security) for the full threat model. We're honest about what this is. Like, painfully honest.

**Q: Can I use it with my existing apps?**
A: If your apps talk to OpenAI then yeah. It's a drop-in replacement. Just change the base URL and you're done. Magic.

**Q: What happens when a provider dies?**
A: It fails over to another one. Your chat doesn't die. You're welcome. It's like having a backup generator but for AI.

**Q: Can I race providers against each other?**
A: Yes. It's basically an AI thunderdome. Two providers enter, one provider leaves. The other writes a blog post about the experience.

**Q: Can two AIs roast each other?**
A: Yes. AI Rumble and Rap Battle are built in. It's exactly as chaotic as it sounds. Popcorn not included. (We're working on the popcorn feature. Maybe.)

**Q: Does it support streaming?**
A: Yes. The stream goes brrr just like it does with the original provider. No lag. No buffering. Just pure, unadulterated streaming.

**Q: Can I set up rate limits?**
A: Yes. Virtual keys support rate limits and tiers. You can limit your friends. Or your enemies. Or both. We don't judge.

**Q: Can I track how much I'm spending?**
A: Yes. Cost analytics, per-provider breakdowns, per-virtual-key usage, budget alerts. Your wallet will thank you. Or cry. Probably cry. But at least you'll know why.

**Q: Can I use this with Claude Code / Cursor?**
A: Yes. Just point them at the gateway. You're welcome. Now go build something cool instead of configuring APIs.

**Q: Can I run this on a Raspberry Pi?**
A: Yes. It'll run on a potato if it has Go installed. Don't test me. (We tested it on a potato. It worked. We were surprised too.)

**Q: What's the catch?**
A: There is no catch. It's free. Open source. MIT license. We just vibes. (Okay, the catch is you have to star the repo. It's in the license. Fine print. Very fine.)

**Q: What if I find a bug?**
A: Open an issue. I'll fix it. Probably. Eventually. No promises. (I'm one person. I have a job. And a sleep schedule. Sometimes.)

**Q: Can I contribute?**
A: Yes! PRs welcome. Just don't break anything. Or if you do, fix it before I notice.

**Q: Why is it called KeyParty?**
A: Because it manages API keys like a party manages guests. Everyone gets a key, everyone gets access, and someone always leaves early. (The provider that dies first.)

---

## Roadmap (aka "things I'll probably never finish")

- [x] Make gateway (done, finally)
- [x] Add failover with compaction (because providers die like my will to live)
- [x] Add encryption (because security is important, apparently)
- [x] Make dashboard (because everything needs a dashboard)
- [x] Add race mode (because why not)
- [x] Add guardrails (because someone has to be the adult)
- [x] Add virtual keys (for your friends who keep asking)
- [x] Add model aliases (so you can sound smart)
- [x] Add request logging (the truth, the whole truth)
- [x] Add cost tracking (where your money goes)
- [x] Add AI Rumble (roast battle, because why not)
- [x] Add Rap Battle (SSE streaming, 8 Mile vibes)
- [x] Add Model Roulette, Vibe Check, Therapist, Roast Logs (the fun stuff)
- [x] Add Webhooks (notifications go brrr)
- [x] Add Prompt Templates (save your best prompts)
- [x] Add AI Poll (democracy!)
- [x] Add Rate Limit Tiers (free vs premium)
- [x] Add Budget Alerts (don't let anyone burn the credits)
- [x] Add VK Usage Dashboard (who's spending what)
- [x] Add Auto-Rotate Keys (survival of the fittest)
- [x] Add Chat Playground (test without leaving the dashboard)
- [x] Add Weekly Recaps (the weekly roast)
- [x] Add Cost Analytics (where your money goes, but with charts)
- [x] Full security audit (31 vulnerabilities fixed, we're talented like that)
- [x] Add keyparty.sh setup script (one command to rule them all)
- [x] Fix SSRF bypass (IPv6-mapped, decimal, hex IP encoding — we caught someone being clever)
- [x] Add brute-force protection on admin auth (try harder, or don't)
- [x] Fix GetKeyByID decryption for test-key endpoint (it was broken, now it's not)
- [ ] Add semantic caching (when I'm bored again, which is always)
- [ ] Add MCP support (because everyone wants MCP now, and who am I to resist)
- [ ] Add A/B testing (for the data nerds)
- [ ] Add streaming support for Rumble/Rap Battle (double SSE, double the chaos)
- [ ] Take over the AI gateway market (one endpoint at a time)
- [ ] Buy a real domain (eventually, maybe, if I remember)
- [ ] Hire someone (it's just me and the void, and the void doesn't code)
- [ ] Retire at 25 (a dev can dream)
- [ ] Actually fix the bugs instead of adding features (we'll see)

---

<div align="center">

**If you star this repo I'll add a feature**

**If you don't star this repo I'll still add features because I have no self control**

**[KeyParty](https://github.com/MastaChief117/keyparty)** ← Click here to join the chaos

*Made by a dude who should've been sleeping*
*Last updated: Whenever I remember (which is never)*
*P.S. If you read this far you're legally obligated to star*
*P.P.S. If you didn't read this far, that's okay, we still love you*

</div>
