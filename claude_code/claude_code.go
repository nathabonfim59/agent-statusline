package claude_code

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/nathabonfim59/agent-statusline/harness"
	"github.com/nathabonfim59/agent-statusline/shared"
)

type Input struct {
	Model struct {
		ID          string `json:"id"`
		DisplayName string `json:"display_name"`
	} `json:"model"`
	SessionID string `json:"session_id"`
	CWD       string `json:"cwd"`
	Version   string `json:"version"`
	Workspace struct {
		CurrentDir string `json:"current_dir"`
		ProjectDir string `json:"project_dir"`
	} `json:"workspace"`
	ContextWindow struct {
		TotalInputTokens  int     `json:"total_input_tokens"`
		TotalOutputTokens int     `json:"total_output_tokens"`
		ContextWindowSize int     `json:"context_window_size"`
		UsedPercentage    float64 `json:"used_percentage"`
		CurrentUsage      struct {
			InputTokens              int `json:"input_tokens"`
			OutputTokens             int `json:"output_tokens"`
			CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
			CacheReadInputTokens     int `json:"cache_read_input_tokens"`
		} `json:"current_usage"`
	} `json:"context_window"`
	Cost struct {
		TotalCostUSD       float64 `json:"total_cost_usd"`
		TotalDurationMS    int64   `json:"total_duration_ms"`
		TotalAPIDurationMS int64   `json:"total_api_duration_ms"`
		TotalLinesAdded    int     `json:"total_lines_added"`
		TotalLinesRemoved  int     `json:"total_lines_removed"`
	} `json:"cost"`
	RateLimits struct {
		FiveHour struct {
			UsedPercentage float64 `json:"used_percentage"`
		} `json:"five_hour"`
		SevenDay struct {
			UsedPercentage float64 `json:"used_percentage"`
		} `json:"seven_day"`
	} `json:"rate_limits"`
}

type Harness struct {
	in Input
}

func New() *Harness {
	return &Harness{}
}

func (h *Harness) Name() string           { return "claude_code" }
func (h *Harness) Parse(raw []byte) error { return json.Unmarshal(raw, &h.in) }
func (h *Harness) ModelID() string        { return h.in.Model.ID }
func (h *Harness) CWD() string {
	if h.in.CWD != "" {
		return h.in.CWD
	}
	if h.in.Workspace.CurrentDir != "" {
		return h.in.Workspace.CurrentDir
	}
	wd, _ := os.Getwd()
	return wd
}
func (h *Harness) TerminalWidth() int                { return harness.TerminalWidth() }
func (h *Harness) ContextPct() float64               { return h.in.ContextWindow.UsedPercentage }
func (h *Harness) ProxyConfig() *harness.ProxyConfig { return nil }

func (h *Harness) RenderBlock(name string, t harness.Theme, pct, warn, danger float64) string {
	cwd := h.CWD()

	projectDir := h.in.Workspace.ProjectDir
	if projectDir == "" {
		projectDir = cwd
	}

	switch name {
	case "model":
		return renderModel(&h.in, t)
	case "git":
		return shared.Git(cwd, t)
	case "project":
		return shared.Project(projectDir, t)
	case "version":
		return shared.Version(h.in.Version, cwd, projectDir, t)
	case "bar":
		return shared.Bar(pct, t, warn, danger)
	case "percent":
		return shared.Percent(pct, t, warn, danger)
	case "tokens":
		return renderTokens(&h.in, t)
	case "cost":
		return renderCost(&h.in, t)
	case "time":
		return renderTime(&h.in, t)
	case "rates":
		return renderRates(&h.in, t)
	case "diff":
		return renderDiff(&h.in, t)
	case "hash":
		return shared.Hash(cwd, t)
	default:
		return ""
	}
}

func init() {
	harness.RegisterNamed("claude_code", func() harness.Harness { return New() })
	harness.Register(func(raw []byte) (harness.Harness, error) {
		var disc struct {
			OutputStyle struct {
				Name string `json:"name"`
			} `json:"output_style"`
		}
		json.Unmarshal(raw, &disc)
		if disc.OutputStyle.Name == "default" || disc.OutputStyle.Name == "compact" {
			return nil, fmt.Errorf("cursor input, skip")
		}

		h := New()
		if err := h.Parse(raw); err != nil {
			return nil, err
		}
		return h, nil
	})
}
