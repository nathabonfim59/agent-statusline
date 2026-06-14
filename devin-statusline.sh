#!/usr/bin/env bash
# Devin CLI Statusline — pipes /tmp/devin_live.json into claude-statusline.
# The Go harness reads model configs + session info from disk directly.
TOKEN_FILE="/tmp/devin_live.json"

if [[ -f "$TOKEN_FILE" ]]; then
    exec cat "$TOKEN_FILE" | claude-statusline
else
    echo "devin: no live data (start mitmproxy with devin_token_addon.py)"
fi