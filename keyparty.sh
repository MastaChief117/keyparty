#!/bin/bash
set -e

BINARY="keyparty"
CONFIG_FILE="gateway.env"
PORT=8080
TUNNEL_LOG="/tmp/keyparty-tunnel.log"
GW_PID=""
CF_PID=""

cleanup() {
    echo ""
    echo "Shutting down..."
    [ -n "$GW_PID" ] && kill "$GW_PID" 2>/dev/null && echo "Killed gateway (PID: $GW_PID)"
    [ -n "$CF_PID" ] && kill "$CF_PID" 2>/dev/null && echo "Killed tunnel (PID: $CF_PID)"
    rm -f "$TUNNEL_LOG"
    exit 0
}
trap cleanup SIGINT SIGTERM

echo "=== KeyParty Setup ==="
echo ""

# ── Check Go ──
if ! command -v go &>/dev/null; then
    # Check common local install paths
    for p in \
        "$(dirname "$0")/../go-sdk/bin/go" \
        "$HOME/go/bin/go" \
        "$(dirname "$0")/../go/bin/go" \
        /usr/local/go/bin/go; do
        if [ -x "$p" ]; then
            export PATH="$(dirname "$p"):$PATH"
            break
        fi
    done
fi
if ! command -v go &>/dev/null; then
    echo "Error: Go is not installed."
    echo "Install from: https://go.dev/dl/"
    echo "Or put it in ../go-sdk/ next to the keyparty folder."
    exit 1
fi
GO_VER=$(go version | grep -oP 'go\K[0-9.]+')
echo "Go version: $GO_VER"

# ── Build ──
echo "Building..."
go build -o "$BINARY" .
echo "Build complete: ./$BINARY"

# ── Admin password ──
if [ -f "$CONFIG_FILE" ]; then
    source "$CONFIG_FILE"
fi
if [ -z "$ADMIN_PASSWORD" ]; then
    echo ""
    read -s -p "Set admin password (leave empty to disable): " ADMIN_PASSWORD
    echo ""
fi
if [ -n "$ADMIN_PASSWORD" ]; then
    echo "ADMIN_PASSWORD=$ADMIN_PASSWORD" > "$CONFIG_FILE"
else
    echo "# No admin password set" > "$CONFIG_FILE"
    echo "Warning: Admin auth is disabled."
fi

# ── Check / install cloudflared ──
echo ""
CF_BIN=""
for p in cloudflared /usr/local/bin/cloudflared "$HOME/.cloudflared/cloudflared"; do
    if command -v "$p" &>/dev/null || [ -x "$p" ]; then
        CF_BIN="$p"
        break
    fi
done

if [ -n "$CF_BIN" ]; then
    echo "cloudflared: installed ($CF_BIN)"
else
    echo "cloudflared not found. Installing..."
    ARCH=$(uname -m)
    case "$ARCH" in
        x86_64)  CF_ARCH="amd64" ;;
        aarch64) CF_ARCH="arm64" ;;
        armv7l)  CF_ARCH="arm" ;;
        *)       echo "Unsupported architecture: $ARCH"; CF_BIN=""; ;;
    esac

    if [ -n "$CF_ARCH" ]; then
        OS=$(uname -s | tr '[:upper:]' '[:lower:]')
        CF_URL="https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-${OS}-${CF_ARCH}"
        echo "Downloading: $CF_URL"
        if curl -fsSL "$CF_URL" -o /tmp/cloudflared; then
            chmod +x /tmp/cloudflared
            if [ -w /usr/local/bin/ ] 2>/dev/null; then
                mv /tmp/cloudflared /usr/local/bin/cloudflared
            elif sudo -n mv /tmp/cloudflared /usr/local/bin/cloudflared 2>/dev/null; then
                true
            else
                mkdir -p "$HOME/.local/bin"
                mv /tmp/cloudflared "$HOME/.local/bin/cloudflared"
                export PATH="$HOME/.local/bin:$PATH"
            fi
            CF_BIN=$(command -v cloudflared || echo "/usr/local/bin/cloudflared")
            echo "cloudflared installed: $(cloudflared --version 2>&1 | head -1)"
        else
            echo "Error: Failed to download cloudflared."
            CF_BIN=""
        fi
    fi
fi

# ── Tunnel setup ──
ENABLE_TUNNEL="n"
if [ -n "$CF_BIN" ]; then
    echo ""
    echo "Start a Cloudflare tunnel to expose the gateway publicly?"
    echo "This hides your real IP behind a *.trycloudflare.com URL."
    echo ""
    read -p "Enable tunnel? (y/N): " ENABLE_TUNNEL
    ENABLE_TUNNEL=$(echo "$ENABLE_TUNNEL" | tr '[:upper:]' '[:lower:]')
else
    echo ""
    echo "Skipping tunnel (cloudflared not available)."
fi

# ── Start gateway ──
echo ""
echo "Starting KeyParty on port $PORT..."
export ADMIN_PASSWORD
./$BINARY -port $PORT &
GW_PID=$!
sleep 1

# Check gateway started
if ! kill -0 $GW_PID 2>/dev/null; then
    echo "Error: Gateway failed to start."
    exit 1
fi
echo "Gateway running (PID: $GW_PID)"

# ── Start tunnel ──
if [ "$ENABLE_TUNNEL" = "y" ] || [ "$ENABLE_TUNNEL" = "yes" ]; then
    echo ""
    echo "Starting cloudflared tunnel..."
    rm -f "$TUNNEL_LOG"
    cloudflared tunnel --url http://localhost:$PORT > "$TUNNEL_LOG" 2>&1 &
    CF_PID=$!

    # Wait for tunnel URL
    echo "Waiting for tunnel URL (up to 30s)..."
    TUNNEL_URL=""
    for i in $(seq 1 30); do
        sleep 1
        if [ -f "$TUNNEL_LOG" ]; then
            TUNNEL_URL=$(grep -oP 'https://[a-z0-9-]+\.trycloudflare\.com' "$TUNNEL_LOG" 2>/dev/null | head -1 || true)
        fi
        if [ -n "$TUNNEL_URL" ]; then
            break
        fi
        # Check if cloudflared died
        if ! kill -0 $CF_PID 2>/dev/null; then
            echo "Error: cloudflared process died."
            echo "Log:"
            cat "$TUNNEL_LOG" 2>/dev/null
            CF_PID=""
            break
        fi
    done

    if [ -n "$TUNNEL_URL" ]; then
        echo ""
        echo "============================================"
        echo "  TUNNEL ACTIVE"
        echo "============================================"
        echo ""
        echo "  Public URL:  $TUNNEL_URL"
        echo "  Proxy:       $TUNNEL_URL/v1/chat/completions"
        echo "  Dashboard:   $TUNNEL_URL"
        echo ""
        echo "  Use the unified API key as Bearer token."
        echo "============================================"
    else
        echo "Warning: Could not get tunnel URL."
        echo "Check logs: cat $TUNNEL_LOG"
    fi
fi

echo ""
echo "============================================"
echo "  LOCAL ACCESS"
echo "============================================"
echo ""
echo "  Dashboard:   http://localhost:$PORT"
echo "  Proxy:       http://localhost:$PORT/v1/chat/completions"
echo "  Health:      http://localhost:$PORT/health"
echo ""
echo "  Gateway PID: $GW_PID"
[ "$ENABLE_TUNNEL" = "y" ] || [ "$ENABLE_TUNNEL" = "yes" ] && echo "  Tunnel PID:  $CF_PID"
echo ""
echo "  Press Ctrl+C to stop everything."
echo "============================================"

# Wait for gateway (keeps script alive, trap handles cleanup)
wait $GW_PID 2>/dev/null
cleanup
