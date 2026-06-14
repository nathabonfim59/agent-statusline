#!/usr/bin/env bash
# Devin CLI Statusline — fetches live data from the built-in proxy.
# The proxy must be running: claude-statusline proxy start devin
# Accepts data port as argument or reads from pid file.
DATA_PORT="${1:-${CLAUDE_STATUSLINE_DATA_PORT:-0}}"

if [[ "$DATA_PORT" -gt 0 ]]; then
    exec curl -s "http://127.0.0.1:${DATA_PORT}/data" | claude-statusline
elif [[ -f /tmp/claude-statusline-devin-data.port ]]; then
    PORT=$(cat /tmp/claude-statusline-devin-data.port)
    exec curl -s "http://127.0.0.1:${PORT}/data" | claude-statusline
elif [[ -f /tmp/devin_live.json ]]; then
    # Fallback: old mitmproxy format
    exec cat /tmp/devin_live.json | claude-statusline
else
    echo "devin: no live data (start proxy with: claude-statusline proxy start devin)"
fi