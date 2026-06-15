#!/usr/bin/env bash
# Devin CLI Statusline — fetches live data from the built-in proxy.
# Usage: ./devin-statusline.sh [label]
set -euo pipefail

DIR="$(cd "$(dirname "$0")" && pwd)"
STATUSLINE="${DIR}/agent-statusline"
LABEL="${1:-}"

if [[ -n "$LABEL" ]]; then
    PORT_FILE="/tmp/agent-statusline-devin-${LABEL}.port"
    if [[ -f "$PORT_FILE" ]]; then
        PORT=$(cat "$PORT_FILE")
        curl -s "http://127.0.0.1:${PORT}/data" | "$STATUSLINE"
    else
        echo "devin: no proxy with label '${LABEL}'"
    fi
else
    # Find most recent port file
    PORT_FILE=$(ls -t /tmp/agent-statusline-devin-*.port 2>/dev/null | head -1)
    if [[ -n "$PORT_FILE" ]]; then
        PORT=$(cat "$PORT_FILE")
        curl -s "http://127.0.0.1:${PORT}/data" | "$STATUSLINE"
    else
        echo "devin: no live data (start proxy with: ./agent-statusline proxy start devin)"
    fi
fi
echo