# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build, Test & Run

```bash
make build
make test
```

Manual run:

```bash
go build -o agent-statusline .
echo '{"session_id":"test","model":{"id":"claude-sonnet-4-6","display_name":"Claude Sonnet 4.6"}}' | ./agent-statusline
```

## Release

Releases are built and published by GoReleaser + nFPM. Pushing a `v*` tag triggers `.github/workflows/release.yaml`.

Test the release pipeline locally (no publish):

```bash
make release-snapshot
```

Cut a release:

```bash
git tag -a v0.1.0 -m "release v0.1.0"
git push origin v0.1.0
```

## Architecture

A stdin-to-stdout CLI filter that renders a two-line ANSI status bar for Claude Code and Cursor sessions. Reads a JSON blob from stdin and prints a colored status bar to stdout.

**`main.go`** — Entry point. Reads stdin, delegates detection to `harness.Detect()`, resolves per-harness block config, builds block map, assembles lines, prints. Also contains the `proxy` and `devin` CLI commands.

**`harness/`** — Shared interface and utilities. Defines the `Harness` interface (`Name`, `Parse`, `RenderBlock`, `ModelID`, `ContextPct`, `TerminalWidth`, `CWD`, `ProxyConfig`), detection registry (`Register`/`Detect`), named constructor registry (`RegisterNamed`/`NewHarness`), `Theme` struct, ANSI color constants, `ProgressBar`, `HumanTokens`, `HumanDuration`, `GitInfo`, `ProjectVersion`, terminal width detection.

**`shared/`** — Blocks common to all harnesses: `git`, `project`, `version`, `bar`, `percent`, `hash`. Each is a single-function file.

**`claude_code/`** — Claude Code harness. `Input` struct matching Claude Code's JSON schema. Blocks: `model`, `tokens` (with cache%), `cost`, `time`, `rates`, `diff`. Detection rejects Cursor input by checking for `output_style.name`.

**`modules/cursor/`** — Cursor harness. `CursorInput` struct handling nullable fields (`*float64` for `used_percentage`, `*struct` for `current_usage`, etc.). Blocks: `model` (with `param_summary` + `MAX` inline), `tokens` (no cache), `vim`, `worktree`, `session`, `autorun`, `output_style`. Detection checks `output_style.name`. `ContextPct()` falls back from `used_percentage` → `100 - remaining_percentage` → 0. `TerminalWidth()` uses `render_width_chars` when available.

**`modules/devin/`** — Devin harness. Includes a `ProxyCollector` that extracts live model/token/quota data from intercepted Devin API responses.

**`proxy/`** — HTTPS MITM proxy and daemon.
- `proxy.go` — `ProxyServer`: per-instance TCP listeners, TLS cert generation, request interception.
- `daemon.go` — `Daemon`: single long-running process that owns the shared CA and a registry of `ProxyServer` instances keyed by `harness/label`.
- `control.go` — HTTP control API over a Unix socket (`/start`, `/stop`, `/status`, `/data/{harness}/{label}`).
- `client.go` — Client that talks to the daemon socket; auto-starts the daemon if it is not running.
- `systemd.go` — Generates and installs user or system systemd unit files for the daemon.
- `cert.go` — CA generation, loading, and trust-store install instructions.

**`theme.go`** — Theme engine + config loading. Supports `HarnessConfig` with `extends: true` to inherit global `blocks:` then override. Theme resolution: `~/.config/agent-statusline/themes/<name>.yaml` → embedded built-in → hardcoded default.

**`themes/`** — YAML theme files.

**`testdata/`** — JSON fixtures for Claude Code and Cursor (full + minimal with null fields).

Config lives at `~/.config/agent-statusline/config.yaml`.

## Key Details

- Pure Go, no CGO required. External dependencies: `gopkg.in/yaml.v3` and `modernc.org/sqlite` (used only by the Devin local-DB harness).
- No CLI flags — driven entirely by stdin JSON and config file
- Git info is gathered by shelling out to `git -C <dir>` (not a Go git library)
- Detection order: Cursor (checks `output_style.name`) → Claude Code (fallback)
- Unsupported blocks return `""` and are silently skipped during line assembly
