package devin

import (
	"fmt"

	"github.com/nathabonfim59/agent-statusline/harness"
)

func renderModel(h *Harness, t harness.Theme) string {
	if h.live.Model == "" {
		return ""
	}
	ctx := h.models[h.live.Model]
	ctxHuman := harness.HumanTokens(ctx)
	total := h.live.InputTokens + h.live.OutputTokens
	result := fmt.Sprintf("%s%s%s%s %s(%s)%s",
		t.Primary, harness.Bold, h.live.Model, harness.Reset,
		harness.Dim, ctxHuman, harness.Reset)
	if total > 0 {
		result += fmt.Sprintf(" %s[%s]%s", t.Text, harness.HumanTokens(total), harness.Reset)
	}
	return result
}