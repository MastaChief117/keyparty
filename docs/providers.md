# Supported Providers

KeyParty supports 10 LLM providers out of the box.

## Provider List

| Provider | Models | Free Tier | Notes |
|----------|--------|-----------|-------|
| **OpenAI** | GPT-4o, GPT-4o-mini, GPT-4.1, o3, o4-mini | Trial credits | The OG |
| **Anthropic** | Claude Sonnet 4.5, Claude 3.5 Haiku | No free tier | Best at coding |
| **Google Gemini** | Gemini 2.5 Pro, Gemini 2.5 Flash | Generous free tier | Google being generous for once |
| **Groq** | Llama 3.3 70B, Mixtral 8x7B | Very generous free tier | Fastest inference |
| **NVIDIA** | Llama 3.1 405B, Nemotron 70B | Free tier available | Big models, big vibes |
| **Together AI** | Llama 3.3 70B, Qwen 2.5 72B | Free trial | Good variety |
| **DeepSeek** | DeepSeek V3, DeepSeek R1 | Cheap | Chinese efficiency |
| **OpenRouter** | Everything (aggregator) | Varies | One key, all models |
| **Fireworks** | Llama 3.3 70B, Mixtral | Free trial | Fast inference |
| **Mistral** | Mistral Large, Codestral | Free trial | European excellence |

## Adding a Provider

1. Get an API key from the provider
2. Go to Dashboard → Keys → Add Key
3. Select the provider, paste your key
4. Enable the key
5. Start sending requests

## Provider Health

The dashboard shows provider health status:
- **Online:** responding normally
- **Degraded:** slow responses or intermittent errors
- **Offline:** not responding

Auto-rotate picks the healthiest key per provider automatically.

## Supported Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/v1/chat/completions` | POST | Chat completions (OpenAI-compatible) |
| `/admin/race` | POST | Provider race mode |
| `/admin/rumble` | POST | AI Rumble (SSE) |
| `/admin/rap-battle` | POST | Rap Battle (SSE) |
| `/admin/roulette` | POST | Model Roulette |
| `/admin/poll` | POST | AI Poll |

## Model Aliases

Map friendly names to any model:

| Alias | Default Model |
|-------|---------------|
| `smart` | `claude-sonnet-4-5` |
| `fast` | `gpt-4o-mini` |
| `creative` | `gpt-4o` |
| `code` | `claude-sonnet-4-5` |

## Virtual Keys

Create consumer-facing keys with:
- **Budget limits:** Per-key spending caps
- **Rate limits:** Requests per minute/hour/day
- **Model allowlists:** Restrict which models can be used
- **Provider restrictions:** Limit to specific providers

```bash
curl -X POST http://localhost:8080/admin/virtual-keys \
  -H "Authorization: Bearer admin-password" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-friend",
    "budget": 5.00,
    "rate_limit": 10,
    "models": ["gpt-4o", "claude-sonnet-4-5"]
  }'
```
