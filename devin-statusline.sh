#!/usr/bin/env bash
# Devin CLI Statusline — fetches live data from the built-in proxy.
# The proxy must be running: ./claude-statusline proxy start devin
set -euo pipefail

DIR="$(cd "$(dirname "$0")" && pwd)"
STATUSLINE="${DIR}/claude-statusline"
DATA_PORT="${1:-${CLAUDE_STATUSLINE_DATA_PORT:-0}}"

if [[ "$DATA_PORT" -gt 0 ]]; then
    curl -s "http://127.0.0.1:${DATA_PORT}/data" | "$STATUSLINE"
elif [[ -f /tmp/claude-statusline-devin-data.port ]]; then
    PORT=$(cat /tmp/claude-statusline-devin-data.port)
    curl -s "http://127.0.0.1:${PORT}/data" | "$STATUSLINE"
elif [[ -f /tmp/devin_live.json ]]; then
    cat /tmp/devin_live.json | "$STATUSLINE"
else
    echo "devin: no live data (start proxy with: ./claude-statusline proxy start devin)"
fi
echo