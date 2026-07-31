# Routing & Failover

## How Routing Works

1. Client sends request to `http://localhost:8080/v1/chat/completions`
2. Gateway checks which provider the virtual key (or unified key) is configured for
3. Gateway selects an available key for that provider (round-robin rotation)
4. Request is forwarded to the provider
5. Response is returned to the client
6. Request is logged with token usage and cost

## Provider Rotation

When multiple keys exist for a provider, KeyParty rotates across them automatically:

- **Round-robin:** Keys are used in sequence
- **Health-aware:** Unhealthy keys are skipped
- **Auto-rotate:** Dashboard can pick the healthiest key per provider

## Failover

### How It Works

1. You send a request to OpenAI
2. OpenAI returns HTTP 402 (quota exceeded)
3. Gateway summarizes your chat history (compaction)
4. Sends the summarized conversation to your backup provider
5. You get a response like nothing happened
6. Response includes `X-Gateway-Failover: true` header

### Configuration

```bash
# Enable failover
curl -X POST http://localhost:8080/admin/failover \
  -H "Authorization: Bearer admin-password" \
  -H "Content-Type: application/json" \
  -d '{"key":"enabled","value":"true"}'

# Set backup provider
curl -X POST http://localhost:8080/admin/failover \
  -H "Authorization: Bearer admin-password" \
  -H "Content-Type: application/json" \
  -d '{"key":"provider","value":"groq"}'

# Set backup model
curl -X POST http://localhost:8080/admin/failover \
  -H "Authorization: Bearer admin-password" \
  -H "Content-Type: application/json" \
  -d '{"key":"model","value":"llama-3.3-70b-versatile"}'
```

### Failover Triggers

- HTTP 402: Payment Required (quota exceeded)
- HTTP 429: Too Many Requests (rate limited)
- HTTP 500+: Server errors
- Connection timeouts

## Provider Race Mode

Sends your message to multiple providers simultaneously and returns the fastest response:

```bash
curl http://localhost:8080/admin/race \
  -H "Authorization: Bearer admin-password" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Hello, which model are you?",
    "providers": ["groq", "nvidia", "openai"]
  }'
```

## Model Aliases

Map friendly names to model IDs:

```bash
curl -X POST http://localhost:8080/admin/aliases \
  -H "Authorization: Bearer admin-password" \
  -H "Content-Type: application/json" \
  -d '{"alias":"smart","model":"claude-sonnet-4-5"}'

# Now use "smart" in requests
curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer YOUR_KEY" \
  -H "Content-Type: application/json" \
  -d '{"model":"smart","messages":[{"role":"user","content":"hello"}]}'
```

## Response Caching

- SHA-256 hash of request (model + messages + temperature + max_tokens)
- Same request within TTL returns cached response
- Cache includes virtual key ID for isolation
- Configurable TTL via `-cache-ttl` flag (default: 60 minutes)

## Chat Compaction

When failover triggers and the conversation is too long for the backup provider:

1. Gateway summarizes the chat history using the primary provider (if available) or a built-in summarizer
2. Summary is sent as a system message to the backup provider
3. Conversation context is preserved
4. You lose some detail but keep the thread
