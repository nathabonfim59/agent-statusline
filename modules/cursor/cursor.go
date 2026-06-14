package cursor

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/nathabonfim59/agent-statusline/harness"
	"github.com/nathabonfim59/agent-statusline/shared"
)

type CursorInput struct {
	SessionID        string  `json:"session_id"`
	SessionName      *string `json:"session_name"`
	TranscriptPath   string  `json:"transcript_path"`
	RenderWidthChars int     `json:"render_width_chars"`
	CWD              string  `json:"cwd"`
	Autorun          bool    `json:"autorun"`
	Model            struct {
		ID           string  `json:"id"`
		DisplayName  string  `json:"display_name"`
		ParamSummary *string `json:"param_summary"`
		MaxMode      *bool   `json:"max_mode"`
	} `json:"model"`
	Workspace struct {
		CurrentDir string   `json:"current_dir"`
		ProjectDir string   `json:"project_dir"`
		AddedDirs  []string `json:"added_dirs"`
	} `json:"workspace"`
	Version     string `json:"version"`
	OutputStyle struct {
		Name string `json:"name"`
	} `json:"output_style"`
	ContextWindow struct {
		TotalInputTokens    int      `json:"total_input_tokens"`
		TotalOutputTokens   *int     `json:"total_output_tokens"`
		ContextWindowSize   int      `json:"context_window_size"`
		UsedPercentage      *float64 `json:"used_percentage"`
		RemainingPercentage *float64 `json:"remaining_percentage"`
		CurrentUsage        *struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"current_usage"`
	} `json:"context_window"`
	Vim *struct {
		Mode string `json:"mode"`
	} `json:"vim"`
	Worktree *struct {
		Name string `json:"name"`
		Path string `json:"path"`
	} `json:"worktree"`
}

type Harness struct {
	in CursorInput
}

func New() *Harness {
	return &Harness{}
}

func (h *Harness) Name() string   { return "cursor" }
func (h *Harness) Parse(raw []byte) error { return json.Unmarshal(raw, &h.in) }
func (h *Harness) ModelID() string { return h.in.Model.ID }

func (h *Harness) ContextPct() float64 {
	if u := h.in.ContextWindow.UsedPercentage; u != nil {
		return *u
	}
	if r := h.in.ContextWindow.RemainingPercentage; r != nil {
		return 100 - *r
	}
	return 0
}

func (h *Harness) TerminalWidth() int {
	if h.in.RenderWidthChars > 0 {
		return h.in.RenderWidthChars
	}
	return harness.TerminalWidth()
}
func (h *Harness) ProxyConfig() *harness.ProxyConfig { return nil }

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
	case "hash":
		return shared.Hash(cwd, t)
	case "vim":
		return renderVim(&h.in, t)
	case "worktree":
		return renderWorktree(&h.in, t)
	case "session":
		return renderSession(&h.in, t)
	case "autorun":
		return renderAutorun(&h.in, t)
	case "output_style":
		return renderOutputStyle(&h.in, t)
	default:
		return ""
	}
}

func init() {
	harness.Register(func(raw []byte) (harness.Harness, error) {
		var disc struct {
			OutputStyle struct {
				Name string `json:"name"`
			} `json:"output_style"`
		}
		if err := json.Unmarshal(raw, &disc); err != nil {
			return nil, err
		}
		if disc.OutputStyle.Name == "" {
			return nil, fmt.Errorf("not cursor input")
		}

		h := New()
		if err := h.Parse(raw); err != nil {
			return nil, err
		}
		return h, nil
	})
}
