<div align="center">

# keyparty

**One endpoint. Ten providers. Zero budget.**

[![npm](https://img.shields.io/npm/v/@steelquill/keyparty?style=flat-square)](https://www.npmjs.com/package/@steelquill/keyparty)
[![license](https://img.shields.io/npm/l/@steelquill/keyparty?style=flat-square)](LICENSE)

*For the broke dev juggling free tiers at 3AM.*

</div>

---

## Install

```bash
# npmjs (recommended)
npm install -g @steelquill/keyparty

# GitHub Packages (needs registry flag)
npm install --registry=https://npm.pkg.github.com @MastaChief117/keyparty
```

Then run:
```bash
keyparty
```

No Go. No Docker. No build step. Just works.

## What happens when you run it

```
  ██╗  ██╗██╗████████╗██████╗  ██████╗ ███████╗
  ██║ ██╔╝██║╚══██╔══╝██╔══██╗██╔═══██╗██╔════╝
  █████╔╝ ██║   ██║   ██████╔╝██║   ██║███████╗
  ██╔═██╗ ██║   ██║   ██╔══██╗██║   ██║╚══════██║
  ╚═╝  ██╗██║   ██║   ██████╔╝╚██████╔╝███████║
    One endpoint. Ten providers. Zero budget.

╔══════════════════════════════════════════════╗
║               Setup Wizard                   ║
╠══════════════════════════════════════════════╣
║  → Step 1: Set admin password                ║
║  → Step 2: Install cloudflared (optional)    ║
║  → Step 3: Choose port                       ║
║  → Step 4: Start gateway                     ║
║  → Step 5: Start tunnel (optional)           ║
╚══════════════════════════════════════════════╝
```

## CLI Options

```bash
keyparty                    # Interactive setup wizard
keyparty --port 8080        # Custom port
keyparty --password xxx     # Set admin password
keyparty --tunnel           # Enable cloudflare tunnel
keyparty --version          # Show version
keyparty --help             # Show help
```

## What you get

- **Dashboard:** http://localhost:8080
- **Proxy:** http://localhost:8080/v1/chat/completions
- **Tunnel:** https://xxx.trycloudflare.com (if enabled)

## How it works

1. Downloads pre-built binary for your platform (linux/mac/windows, x64/arm64)
2. Optionally installs cloudflared for tunneling
3. Interactive wizard sets up password + port
4. Starts gateway + optional tunnel
5. Done. Use it like any OpenAI-compatible API.

## Platform Support

| Platform | Architecture |
|----------|-------------|
| Linux | x64, arm64 |
| macOS | x64, arm64 (Apple Silicon) |
| Windows | x64 |

## Use in your apps

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer gw-your-unified-key" \
  -d '{"model":"groq/llama-3.3-70b-versatile","messages":[{"role":"user","content":"Hello"}]}'
```

Works with OpenAI SDK, Anthropic SDK, Cursor, Claude Code, or any HTTP client.

## Docker

```bash
# Docker Hub
docker pull steelquill69/keyparty:latest

# GitHub Container Registry
docker pull ghcr.io/mastachief117/keyparty:latest
```

## Docs

- [Getting Started](https://mastachief117.github.io/keyparty/getting-started.html)
- [API Reference](https://mastachief117.github.io/keyparty/api.html)
- [GitHub](https://github.com/MastaChief117/keyparty)

## License

Apache-2.0
