#!/usr/bin/env bash
set -euo pipefail

BINARY=./claude-statusline

fail() { echo "FAIL: $1"; echo "$2"; exit 1; }
check() { echo "$1" | grep -q "$2" || fail "$3" "$1"; }
nocheck() { echo "$1" | grep -q "$2" && fail "$3" "$1" || true; }

# --- Claude Code ---
echo "--- Claude Code ---"
output=$(cat testdata/claude_code.json | $BINARY 2>&1)
lines=$(echo "$output" | wc -l)
[ "$lines" -eq 2 ] || fail "expected 2 lines, got $lines" "$output"
check "$output" "Claude Sonnet"   "missing model name"
check "$output" "200k context"    "missing context size"
check "$output" '28%'             "missing usage percentage"
check "$output" 'v2.5.0'          "missing version"
check "$output" '\$1.85'          "missing cost"
check "$output" 'cache:'          "missing cache percent"
check "$output" '5h:'             "missing rate limit"
check "$output" '+156'            "missing lines added"
check "$output" '\-42'            "missing lines removed"
echo "PASS"

# --- Cursor (full) ---
echo "--- Cursor (full) ---"
output=$(COLUMNS=200 cat testdata/cursor.json | $BINARY 2>&1)
lines=$(echo "$output" | wc -l)
[ "$lines" -eq 2 ] || fail "expected 2 lines, got $lines" "$output"
check "$output" "Claude 4 Opus"   "missing model name"
check "$output" "200k context"    "missing context size"
check "$output" '(Thinking)'      "missing param_summary"
check "$output" 'MAX'             "missing max_mode"
check "$output" '34%'             "missing usage percentage"
check "$output" '\[4k\]'          "missing current usage tokens"
echo "PASS"

# --- Cursor (minimal) ---
echo "--- Cursor (minimal) ---"
output=$(COLUMNS=200 cat testdata/cursor_minimal.json | $BINARY 2>&1)
lines=$(echo "$output" | wc -l)
[ "$lines" -eq 2 ] || fail "expected 2 lines, got $lines" "$output"
check   "$output" "Claude 4 Opus" "missing model name"
check   "$output" "2%"            "missing usage percentage (computed from remaining)"
nocheck "$output" "cache"         "cache should not appear when current_usage is null"
echo "PASS"

# --- Cursor blocks (vim, worktree, session, autorun, output_style) ---
echo "--- Cursor blocks (vim, worktree, session, autorun, output_style) ---"
tmpdir=$(mktemp -d)
trap "rm -rf $tmpdir" EXIT
mkdir -p "$tmpdir/claude-statusline"
cat > "$tmpdir/claude-statusline/config.yaml" <<'EOF'
blocks:
  line1: [model, git, project, version]
  line2: [bar, percent, cost, time, tokens, rates, diff, hash]
  compact: [model, bar, percent, cost, git, project, hash, time, tokens, rates, diff, version]
cursor:
  extends: true
  blocks:
    line1: [vim, worktree, session, autorun, output_style, model]
    line2: [bar, percent]
    compact: [model, bar, percent, vim, worktree, session, autorun, output_style]
EOF
output=$(cat testdata/cursor.json | COLUMNS=300 XDG_CONFIG_HOME="$tmpdir" $BINARY 2>&1)
check "$output" "NORMAL"           "missing vim block"
check "$output" "my-feature"       "missing worktree block"
check "$output" "my cursor session" "missing session block"
check "$output" "AUTO"             "missing autorun block"
check "$output" "default"          "missing output_style block"
echo "PASS"

# --- Bar brackets option ---
echo "--- Bar brackets option ---"
strip_ansi() { sed -E 's/\x1b\[[0-9;]*m//g'; }

tmpdir2=$(mktemp -d)
trap "rm -rf $tmpdir $tmpdir2" EXIT
mkdir -p "$tmpdir2/claude-statusline"
cat > "$tmpdir2/claude-statusline/config.yaml" <<'EOF'
bar:
  brackets: false
blocks:
  line1: [model]
  line2: [bar, percent]
  compact: [model, bar, percent]
EOF
line2_off=$(cat testdata/claude_code.json | XDG_CONFIG_HOME="$tmpdir2" $BINARY 2>&1 | sed -n '2p' | strip_ansi)
nocheck "$line2_off" '^\[' "bar should not be wrapped in [ when brackets: false (got: $line2_off)"
nocheck "$line2_off" '\] |' "bar should not be wrapped in ] when brackets: false (got: $line2_off)"

# Default (brackets unset) should still wrap the bar in [ ]
tmpdir3=$(mktemp -d)
trap "rm -rf $tmpdir $tmpdir2 $tmpdir3" EXIT
mkdir -p "$tmpdir3/claude-statusline"
cat > "$tmpdir3/claude-statusline/config.yaml" <<'EOF'
blocks:
  line1: [model]
  line2: [bar, percent]
  compact: [model, bar, percent]
EOF
line2_on=$(cat testdata/claude_code.json | XDG_CONFIG_HOME="$tmpdir3" $BINARY 2>&1 | sed -n '2p' | strip_ansi)
check   "$line2_on" '^\[' "bar should be wrapped in [ by default (got: $line2_on)"
check   "$line2_on" '\] |' "bar should be wrapped in ] by default (got: $line2_on)"
echo "PASS"

echo "=== all tests passed ==="
