package devin

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/nathabonfim59/agent-statusline/harness"
	"github.com/nathabonfim59/agent-statusline/shared"
)

type LiveInput struct {
	Model        string     `json:"model"`
	InputTokens  int        `json:"input_tokens"`
	OutputTokens int        `json:"output_tokens"`
	Quota        *QuotaInfo `json:"quota"`
}

type QuotaInfo struct {
	Plan       string `json:"plan"`
	DailyLimit int    `json:"daily_limit"`
	DailyUsed  int    `json:"daily_used"`
}

type Harness struct {
	live   LiveInput
	local  *LocalData
	models ModelConfigs
}

func New() *Harness {
	return &Harness{}
}

func (h *Harness) Name() string       { return "devin" }
func (h *Harness) ModelID() string    { return h.live.Model }
func (h *Harness) TerminalWidth() int { return harness.TerminalWidth() }
func (h *Harness) ProxyConfig() *harness.ProxyConfig {
	cfg := DevinProxyConfig()
	return &cfg
}

func (h *Harness) Parse(raw []byte) error {
	if err := json.Unmarshal(raw, &h.live); err != nil {
		return err
	}
	h.local, h.models = loadLocalData()
	return nil
}

func (h *Harness) ContextPct() float64 {
	ctx := h.models[h.live.Model]
	if ctx > 0 {
		return float64(h.live.InputTokens) / float64(ctx) * 100
	}
	return 0
}

func (h *Harness) CWD() string {
	if h.local != nil && h.local.CWD != "" {
		return h.local.CWD
	}
	wd, _ := os.Getwd()
	return wd
}

func (h *Harness) RenderBlock(name string, t harness.Theme, pct, warn, danger float64) string {
	cwd := h.CWD()

	switch name {
	case "model":
		return renderModel(h, t)
	case "tokens":
		return renderTokens(h, t)
	case "quota":
		return renderQuota(h, t)
	case "git":
		return shared.Git(cwd, t)
	case "project":
		return shared.Project(cwd, t)
	case "bar":
		return shared.Bar(pct, t, warn, danger)
	case "percent":
		return shared.Percent(pct, t, warn, danger)
	case "hash":
		return shared.Hash(cwd, t)
	default:
		return ""
	}
}

func init() {
	harness.Register(func(raw []byte) (harness.Harness, error) {
		var disc struct {
			Quota *QuotaInfo `json:"quota"`
		}
		if err := json.Unmarshal(raw, &disc); err != nil {
			return nil, err
		}
		if disc.Quota == nil {
			return nil, fmt.Errorf("not devin input")
		}

		h := New()
		if err := h.Parse(raw); err != nil {
			return nil, err
		}
		return h, nil
	})
}