# API Reference

KeyParty is fully OpenAI-compatible. Use it as a drop-in replacement.

## Chat Completions

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer YOUR_UNIFIED_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o",
    "messages": [{"role": "user", "content": "say hi"}]
  }'
```

## Provider Race Mode

```bash
curl http://localhost:8080/admin/race \
  -H "Authorization: Bearer YOUR_ADMIN_PASSWORD" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Hello, which model are you?",
    "providers": ["groq", "nvidia"]
  }'
```

## AI Rumble (SSE)

```bash
curl http://localhost:8080/admin/rumble \
  -H "Authorization: Bearer YOUR_ADMIN_PASSWORD" \
  -H "Content-Type: application/json" \
  -d '{"rounds": 5}' | stream
```

## Model Roulette

```bash
curl http://localhost:8080/admin/roulette \
  -H "Authorization: Bearer YOUR_ADMIN_PASSWORD" \
  -H "Content-Type: application/json" \
  -d '{"message": "Say something chaotic"}'
```

## Admin Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/admin/unified-key` | Get your unified API key |
| POST | `/admin/unified-key` | Regenerate it |
| GET/POST | `/admin/keys` | List / Add provider API keys |
| DELETE | `/admin/keys/:id` | Delete a key |
| POST | `/admin/keys/:id/toggle` | Enable/disable a key |
| GET/POST | `/admin/failover` | Failover config |
| GET | `/admin/failover/logs` | Failover history |
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

## Configuration

### CLI Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-port` | `8080` | Server port |
| `-admin-pass` | (required) | Admin dashboard password |
| `-db` | `gateway.db` | SQLite database path |
| `-cache-ttl` | `60` | Cache TTL in minutes |
| `-cors-origin` | (empty) | Allowed CORS origin (empty = allow all) |

### Environment Variables

| Variable | Description |
|----------|-------------|
| `ADMIN_PASSWORD` | Admin password (alternative to `-admin-pass`) |
