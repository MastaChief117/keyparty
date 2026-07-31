# Architecture

## Overview

KeyParty is a single-binary AI gateway written in Go. It routes requests across multiple LLM providers, handles failover, encrypts keys at rest, and provides a web dashboard.

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

## Components

### Proxy Layer (`proxy/`)
- Routes requests to the selected provider
- Handles SSRF protection via `net.ParseIP()` validation
- Manages provider health checks and failover
- Supports streaming (SSE) responses

### Auth Layer (`auth/`)
- Bearer token authentication for admin endpoints
- Per-IP brute-force protection (5 attempts/min lockout)
- Virtual key management with budgets and rate limits

### Store Layer (`store/`)
- SQLite database for all persistent data
- AES-256-GCM encryption for API keys at rest
- Request logging with token usage and cost tracking
- Response caching with SHA-256 deduplication

### Frontend
- Vanilla HTML/CSS/JS (no framework dependencies)
- Dark mode by default (light mode is a war crime)
- Real-time streaming via SSE
- Dashboard with tabs for all features

## Data Flow

```
Request → Auth Check → Route Selection → Provider Selection
    → Key Selection → Forward Request → Log Request
    → Cache Response → Return to Client

On Provider Failure:
    → Detect Error (402/429/500) → Select Backup Provider
    → Compact Chat History → Forward to Backup → Log Failover
```

## Encryption

- **At Rest:** AES-256-GCM via `golang.org/x/crypto`
- **Key Storage:** `.gateway.key` (32-byte random, `0600` perms)
- **First Run:** Key generated automatically, stored in `.gateway.key`
- **Lost Key:** All encrypted data is permanently unrecoverable

## Database

- **Engine:** SQLite (zero-config, single file)
- **Size:** ~100 bytes per logged request
- **Tables:** keys, virtual_keys, logs, config, guardrails, templates, rate_tiers, budget_alerts, webhooks

## Deployment

```
Single binary (~16MB)
    ↓
Runs on Linux, macOS, Windows
    ↓
13MB idle RAM, ~33MB under load
    ↓
No Docker, no npm, no dependencies
```

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Language | Go 1.24+ |
| Database | SQLite |
| Encryption | AES-256-GCM |
| Frontend | Vanilla HTML/CSS/JS |
| Tunnel | Cloudflare Quick Tunnel (optional) |
