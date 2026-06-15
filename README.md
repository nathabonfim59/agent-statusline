# agent-statusline

A fast, customizable and themeable status line for [Claude Code](https://docs.anthropic.com/en/docs/claude-code). Single Go binary. Works in tmux and doesn't break when you resize your terminal.

<img width="851" height="123" alt="image" src="https://github.com/user-attachments/assets/162f91cd-35d4-45f7-ad47-bac65e8b2fe6" />


## Install

**Linux / macOS**

```bash
curl -fsSL https://raw.githubusercontent.com/nathabonfim59/agent-statusline/main/install.sh | sh
```

Installs to `~/.local/bin` by default. Override with `INSTALL_DIR`:

```bash
INSTALL_DIR=/usr/local/bin curl -fsSL https://raw.githubusercontent.com/nathabonfim59/agent-statusline/main/install.sh | sh
```

**Windows (PowerShell)**

```powershell
irm https://raw.githubusercontent.com/nathabonfim59/agent-statusline/main/install.ps1 | iex
```

Installs to `$env:USERPROFILE\.local\bin` by default. Override with `$env:INSTALL_DIR`:

```powershell
$env:INSTALL_DIR = "C:\tools"; irm https://raw.githubusercontent.com/nathabonfim59/agent-statusline/main/install.ps1 | iex
```

Both scripts will print instructions if the install directory is not in your `PATH`.

Environment variables supported by the install scripts:

- `VERSION` — install a specific release tag instead of the latest `v*` release.
- `INSTALL_DIR` — target directory for the binary.
- `NO_INSTALL_CONFIG` — skip copying bundled themes/example config.

**Install from a release archive with the GitHub CLI**

```bash
gh release download --repo nathabonfim59/agent-statusline --pattern '*linux_amd64.tar.gz'
tar -xzf agent-statusline_*.tar.gz
```

**Linux packages (deb / rpm / apk / arch)**

Go to the [latest release](https://github.com/nathabonfim59/agent-statusline/releases/latest) and download the package for your distro, or use `gh release download` with the appropriate pattern (`*.deb`, `*.rpm`, `*.apk`, `*.pkg.tar.zst`).

Or via `go install`:

```bash
go install github.com/nathabonfim59/agent-statusline@latest
```

Or build from source:

```bash
git clone https://github.com/nathabonfim59/agent-statusline.git
cd agent-statusline
go build -o agent-statusline .
```

Then add it to your Claude Code settings (`~/.claude/settings.json`):

```json
{
  "statusLine": {
    "type": "command",
    "command": "~/.local/bin/agent-statusline"
  }
}
```

## Quick Start

Generate a config file with sensible defaults:

```bash
agent-statusline init
```

This creates `~/.config/agent-statusline/config.yaml` (or `%AppData%\agent-statusline\config.yaml` on Windows).

## Configuration

Every field is optional. Missing values fall back to defaults.

```yaml
theme: default

thresholds:
  default:
    warning: 50
    danger: 75

blocks:
  line1: [model, git, project, version]
  line2: [bar, percent, cost, time, tokens, rates, diff, hash]
  compact: [model, bar, percent, cost, git, project, hash, time, tokens, rates, diff, version]
```

### Theme

Built-in themes:

| Name | Config value | Preview |
|---|---|---|
| Default | `default` | <img width="752" height="72" alt="image" src="https://github.com/user-attachments/assets/fd0fdf9e-6446-4b68-9c62-232f371723a0" /> |
| One Dark | `onedark` | <img width="752" height="72" alt="image" src="https://github.com/user-attachments/assets/67ced750-bc59-4542-a624-14a612912d95" /> |
| Monokai | `monokai` | <img width="752" height="72" alt="image" src="https://github.com/user-attachments/assets/6713c655-03cd-48eb-a44d-d1e5e71054e1" /> |
| Catppuccin Mocha | `catppuccin` | <img width="752" height="72" alt="image" src="https://github.com/user-attachments/assets/eef7246f-2c50-410b-851f-48783c38453b" /> |
| Dracula | `dracula` | <img width="752" height="72" alt="image" src="https://github.com/user-attachments/assets/9ae30ed3-23a7-4893-b433-0ee1e208f790" /> |
| JetBrains Dark | `jetbrains` | <img width="752" height="72" alt="image" src="https://github.com/user-attachments/assets/736f598a-7ed7-4664-9e0d-a641e17e3f43" /> |
| Oggu | `oggu` | <img width="752" height="72" alt="image" src="https://github.com/user-attachments/assets/6a390604-f7fc-4cf3-af06-7ed340fe7d86" /> |

Set to a custom name to load `~/.config/agent-statusline/themes/<name>.yaml`.

### Thresholds

As a session grows, the model's context window fills up and response quality degrades: the model starts repeating itself, forgetting earlier instructions, or making mistakes. The progress bar's threshold markers (`|`) and color changes help you spot this before it becomes a problem.

You can set different thresholds per model, since some models degrade sooner than others. Works with any model ID: Claude, GLM, whatever you're using.

```yaml
thresholds:
  default:            # all models
    warning: 50
    danger: 75
  claude-opus-4-7:    # Opus still works well past 50%
    warning: 60
    danger: 80
  GLM-5.1[1m]:        # GLM plan: set tighter if it degrades early
    warning: 40
    danger: 60
```

### Adaptive Layout

The statusline adapts to your terminal width in real time. Resize a tmux pane, shrink a terminal window: the next response automatically reflows to fit. Blocks that don't fit get dropped in the order you define, so you always see what matters most.

You control this through `compact`: it's a priority list. Blocks listed first are the last to be removed when space is tight. Combine it with `line1` and `line2` to decide what shows on each row and in what order:

```yaml
blocks:
  line1: [model, git, project, version]     # top row, full order
  line2: [bar, percent, cost, time, tokens, rates, diff, hash]  # bottom row
  compact: [model, bar, percent, cost, ...] # drop order when narrow
```

Only care about the essentials? Remove blocks entirely:

```yaml
blocks:
  line1: [model, project]
  line2: [bar, percent, cost]
  compact: [model, bar, percent]
```

#### Available blocks

| Block | Line | Description | Preview |
|---|---|---|---|
| `model` | 1 | Model name, context window size, tokens in use | <!-- screenshot: model block --> |
| `git` | 1 | Branch name with `+N` `~N` `?N` file counts | <!-- screenshot: git block --> |
| `project` | 1 | Project directory name | <!-- screenshot: project block --> |
| `version` | 1 | Claude Code or project version | <!-- screenshot: version block --> |
| `bar` | 2 | Context window progress bar | <!-- screenshot: bar block --> |
| `percent` | 2 | Context usage percentage | <!-- screenshot: percent block --> |
| `cost` | 2 | Session cost in USD | <!-- screenshot: cost block --> |
| `time` | 2 | Elapsed time and API time | <!-- screenshot: time block --> |
| `tokens` | 2 | Token count with cache hit % | <!-- screenshot: tokens block --> |
| `rates` | 2 | Rate limit usage (5h / 7d) | <!-- screenshot: rates block --> |
| `diff` | 2 | Lines added / removed | <!-- screenshot: diff block --> |
| `hash` | 2 | Short git commit hash | <!-- screenshot: hash block --> |

## Theming

Themes define 5 semantic color roles:

| Role | Where it's used |
|---|---|
| `primary` | Model name, section headers |
| `text` | General info (cost, project, tokens) |
| `success` | Git branch, progress bar (low), diff `+` |
| `warning` | Progress bar (medium) |
| `danger` | Progress bar (high), diff `-` |

### Color formats

Each role accepts three formats:

```yaml
primary: cyan             # named ANSI (cyan, green, yellow, red, white, dim, bold)
text: "#ABB2BF"           # hex true color (#RRGGBB)
danger: "\033[38;5;75m"   # raw ANSI escape
```

### Resolution order

1. Local override (`~/.config/agent-statusline/themes/<name>.yaml`)
2. Built-in (compiled-in `themes/*.yaml`)
3. Hard-coded fallback (cyan/white/green/yellow/red)

## Creating a Custom Theme

1. Create a YAML file in `~/.config/agent-statusline/themes/`:

   ```yaml
   name: My Theme
   colors:
     primary: "#FF79C6"
     text: "#F8F8F2"
     success: "#50FA7B"
     warning: "#F1FA8C"
     danger: "#FF5555"
   ```

2. Set `theme` in your config to the filename (without `.yaml`, lowercased):

   ```yaml
   theme: my-theme
   ```

3. Test it:

   ```bash
   echo '{"session_id":"test","model":{"id":"claude-sonnet-4-6","display_name":"Sonnet 4.6"},"cwd":"/tmp/myproject","context_window":{"context_window_size":200000,"used_percentage":42,"current_usage":{"input_tokens":80000,"cache_read_input_tokens":50000,"cache_creation_input_tokens":0,"output_tokens":3000}},"cost":{"total_cost_usd":1.23,"total_duration_ms":185000,"total_api_duration_ms":120000,"total_lines_added":45,"total_lines_removed":12},"rate_limits":{"five_hour":{"used_percentage":30},"seven_day":{"used_percentage":15}}}' | ./agent-statusline
   ```

You can mix color formats, and only `primary` is required. Missing roles fall back to defaults. Placing a file like `monokai.yaml` in your themes directory overrides the built-in of the same name.

## Proxy daemon (Devin)

The Devin harness can intercept live API traffic to show real-time token usage and quota. Instead of starting a new OS process per proxy, `agent-statusline` runs a single daemon that manages isolated proxy instances on separate ports.

Install the daemon as a systemd user service:

```bash
agent-statusline proxy install-systemd --user
```

Start a Devin proxy:

```bash
agent-statusline proxy start devin
```

The command returns immediately and prints the `HTTP_PROXY` / `HTTPS_PROXY` settings. Run it again to start another isolated instance — each gets its own listeners and collector, so data never mingles.

Show the live statusline:

```bash
agent-statusline devin
```

Or for a specific instance:

```bash
agent-statusline devin <label>
```

Stop instances:

```bash
agent-statusline proxy stop <label>   # stop one
agent-statusline proxy stop           # stop all
agent-statusline proxy status         # list running instances
```

For a shared system service (requires root and a dedicated `agent-statusline` user):

```bash
sudo agent-statusline proxy install-systemd --system
```

### Acknowledgment

| Theme |
|---|
| **One Dark** - From the [One Dark](https://github.com/joshdick/onedark.vim) color scheme. |
| **Monokai** - From the [Monokai](https://monokai.pro/) color palette. |
| **Catppuccin Mocha** - From the [Catppuccin](https://catppuccin.com/) theme collection. |
| **Dracula** - From the [Dracula](https://draculatheme.com/) theme. |
| **JetBrains Dark** - Based on the JetBrains IDE [dark theme](https://github.com/artemevsevev/zed-theme-jetbrains). |

## License

[MIT](LICENSE)
