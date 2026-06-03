# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build, Test & Run

```bash
make build
make test
```

Manual run:

```bash
go build -o claude-statusline .
echo '{"session_id":"test","model":{"id":"claude-sonnet-4-6","display_name":"Claude Sonnet 4.6"}}' | ./claude-statusline
```

## Architecture

A stdin-to-stdout CLI filter that renders a two-line ANSI status bar for Claude Code and Cursor sessions. Reads a JSON blob from stdin and prints a colored status bar to stdout.

**`main.go`** — Entry point. Reads stdin, delegates detection to `harness.Detect()`, resolves per-harness block config, builds block map, assembles lines, prints.

**`harness/`** — Shared interface and utilities. Defines the `Harness` interface (`Name`, `Parse`, `RenderBlock`, `ModelID`, `ContextPct`, `TerminalWidth`, `CWD`), detection registry (`Register`/`Detect`), `Theme` struct, ANSI color constants, `ProgressBar`, `HumanTokens`, `HumanDuration`, `GitInfo`, `ProjectVersion`, terminal width detection.

**`shared/`** — Blocks common to all harnesses: `git`, `project`, `version`, `bar`, `percent`, `hash`. Each is a single-function file.

**`claude_code/`** — Claude Code harness. `Input` struct matching Claude Code's JSON schema. Blocks: `model`, `tokens` (with cache%), `cost`, `time`, `rates`, `diff`. Detection rejects Cursor input by checking for `output_style.name`.

**`modules/cursor/`** — Cursor harness. `CursorInput` struct handling nullable fields (`*float64` for `used_percentage`, `*struct` for `current_usage`, etc.). Blocks: `model` (with `param_summary` + `MAX` inline), `tokens` (no cache), `vim`, `worktree`, `session`, `autorun`, `output_style`. Detection checks `output_style.name`. `ContextPct()` falls back from `used_percentage` → `100 - remaining_percentage` → 0. `TerminalWidth()` uses `render_width_chars` when available.

**`theme.go`** — Theme engine + config loading. Supports `HarnessConfig` with `extends: true` to inherit global `blocks:` then override. Theme resolution: `~/.config/claude-statusline/themes/<name>.yaml` → embedded built-in → hardcoded default.

**`themes/`** — YAML theme files.

**`testdata/`** — JSON fixtures for Claude Code and Cursor (full + minimal with null fields).

Config lives at `~/.config/claude-statusline/config.yaml`.

## Key Details

- Pure Go, single external dependency: `gopkg.in/yaml.v3`
- No CLI flags — driven entirely by stdin JSON and config file
- Git info is gathered by shelling out to `git -C <dir>` (not a Go git library)
- Detection order: Cursor (checks `output_style.name`) → Claude Code (fallback)
- Unsupported blocks return `""` and are silently skipped during line assembly
