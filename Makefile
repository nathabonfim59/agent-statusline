.PHONY: build test test-claude-code test-cursor test-cursor-minimal test-all clean vet

BINARY := claude-statusline

build:
	go build -o $(BINARY) .

vet:
	go vet ./...

$(BINARY): build

test: build vet test-claude-code test-cursor test-cursor-minimal test-cursor-blocks
	@echo "=== all tests passed ==="

test-claude-code: $(BINARY)
	@echo "--- Claude Code ---"
	@output=$$(cat testdata/claude_code.json | ./$(BINARY) 2>&1); \
	lines=$$(echo "$$output" | wc -l); \
	if [ "$$lines" -ne 2 ]; then \
		echo "FAIL: expected 2 lines, got $$lines"; \
		echo "$$output"; \
		exit 1; \
	fi; \
	if ! echo "$$output" | grep -q "Claude Sonnet"; then \
		echo "FAIL: missing model name"; \
		echo "$$output"; \
		exit 1; \
	fi; \
	if ! echo "$$output" | grep -q "200k context"; then \
		echo "FAIL: missing context size"; \
		echo "$$output"; \
		exit 1; \
	fi; \
	if ! echo "$$output" | grep -q '28%'; then \
		echo "FAIL: missing usage percentage"; \
		echo "$$output"; \
		exit 1; \
	fi; \
	if ! echo "$$output" | grep -q 'v2.5.0'; then \
		echo "FAIL: missing version"; \
		echo "$$output"; \
		exit 1; \
	fi; \
	if ! echo "$$output" | grep -q '\$$1.85'; then \
		echo "FAIL: missing cost"; \
		echo "$$output"; \
		exit 1; \
	fi; \
	if ! echo "$$output" | grep -q 'cache:'; then \
		echo "FAIL: missing cache percent"; \
		echo "$$output"; \
		exit 1; \
	fi; \
	if ! echo "$$output" | grep -q '5h:'; then \
		echo "FAIL: missing rate limit"; \
		echo "$$output"; \
		exit 1; \
	fi; \
	if ! echo "$$output" | grep -q '+156'; then \
		echo "FAIL: missing lines added"; \
		echo "$$output"; \
		exit 1; \
	fi; \
	if ! echo "$$output" | grep -q '\-42'; then \
		echo "FAIL: missing lines removed"; \
		echo "$$output"; \
		exit 1; \
	fi; \
	echo "PASS"

test-cursor: $(BINARY)
	@echo "--- Cursor (full) ---"
	@output=$$(COLUMNS=200 cat testdata/cursor.json | ./$(BINARY) 2>&1); \
	lines=$$(echo "$$output" | wc -l); \
	if [ "$$lines" -ne 2 ]; then \
		echo "FAIL: expected 2 lines, got $$lines"; \
		echo "$$output"; \
		exit 1; \
	fi; \
	if ! echo "$$output" | grep -q "Claude 4 Opus"; then \
		echo "FAIL: missing model name"; \
		echo "$$output"; \
		exit 1; \
	fi; \
	if ! echo "$$output" | grep -q "200k context"; then \
		echo "FAIL: missing context size"; \
		echo "$$output"; \
		exit 1; \
	fi; \
	if ! echo "$$output" | grep -q '(Thinking)'; then \
		echo "FAIL: missing param_summary"; \
		echo "$$output"; \
		exit 1; \
	fi; \
	if ! echo "$$output" | grep -q 'MAX'; then \
		echo "FAIL: missing max_mode"; \
		echo "$$output"; \
		exit 1; \
	fi; \
	if ! echo "$$output" | grep -q '34%'; then \
		echo "FAIL: missing usage percentage"; \
		echo "$$output"; \
		exit 1; \
	fi; \
	if ! echo "$$output" | grep -q '\[4k\]'; then \
		echo "FAIL: missing current usage tokens"; \
		echo "$$output"; \
		exit 1; \
	fi; \
	echo "PASS"

test-cursor-minimal: $(BINARY)
	@echo "--- Cursor (minimal) ---"
	@output=$$(COLUMNS=200 cat testdata/cursor_minimal.json | ./$(BINARY) 2>&1); \
	lines=$$(echo "$$output" | wc -l); \
	if [ "$$lines" -ne 2 ]; then \
		echo "FAIL: expected 2 lines, got $$lines"; \
		echo "$$output"; \
		exit 1; \
	fi; \
	if ! echo "$$output" | grep -q "Claude 4 Opus"; then \
		echo "FAIL: missing model name"; \
		echo "$$output"; \
		exit 1; \
	fi; \
	if ! echo "$$output" | grep -q "2%"; then \
		echo "FAIL: missing usage percentage (computed from remaining)"; \
		echo "$$output"; \
		exit 1; \
	fi; \
	if echo "$$output" | grep -q 'cache'; then \
		echo "FAIL: cache should not appear when current_usage is null"; \
		echo "$$output"; \
		exit 1; \
	fi; \
	echo "PASS"

test-cursor-blocks: $(BINARY)
	@echo "--- Cursor blocks (vim, worktree, session, autorun, output_style) ---"
	@mkdir -p /tmp/claude-st-test/claude-statusline
	@printf 'blocks:\n  line1: [model, git, project, version]\n  line2: [bar, percent, cost, time, tokens, rates, diff, hash]\n  compact: [model, bar, percent, cost, git, project, hash, time, tokens, rates, diff, version]\ncursor:\n  extends: true\n  blocks:\n    line1: [vim, worktree, session, autorun, output_style, model]\n    line2: [bar, percent]\n    compact: [model, bar, percent, vim, worktree, session, autorun, output_style]\n' > /tmp/claude-st-test/claude-statusline/config.yaml
	@output=$$(cat testdata/cursor.json | COLUMNS=300 XDG_CONFIG_HOME=/tmp/claude-st-test ./$(BINARY) 2>&1); \
	if ! echo "$$output" | grep -q "NORMAL"; then \
		echo "FAIL: missing vim block"; \
		echo "$$output"; \
		exit 1; \
	fi; \
	if ! echo "$$output" | grep -q "my-feature"; then \
		echo "FAIL: missing worktree block"; \
		echo "$$output"; \
		exit 1; \
	fi; \
	if ! echo "$$output" | grep -q "my cursor session"; then \
		echo "FAIL: missing session block"; \
		echo "$$output"; \
		exit 1; \
	fi; \
	if ! echo "$$output" | grep -q "AUTO"; then \
		echo "FAIL: missing autorun block"; \
		echo "$$output"; \
		exit 1; \
	fi; \
	if ! echo "$$output" | grep -q "default"; then \
		echo "FAIL: missing output_style block"; \
		echo "$$output"; \
		exit 1; \
	fi; \
	rm -rf /tmp/claude-st-test; \
	echo "PASS"

test-all: test-claude-code test-cursor test-cursor-minimal test-cursor-blocks
	@echo "=== all tests passed ==="

clean:
	rm -f $(BINARY)
