package devin

import (
	"fmt"
	"strings"

	"github.com/nathabonfim59/agent-statusline/harness"
)

func renderModel(h *Harness, t harness.Theme) string {
	if h.live.Model == "" {
		return ""
	}
	ctx := h.models[h.live.Model]
	if ctx == 0 {
		// Try case-insensitive fallback
		for k, v := range h.models {
			if strings.EqualFold(k, h.live.Model) {
				ctx = v
				break
			}
		}
	}
	ctxHuman := harness.HumanTokens(ctx)
	result := fmt.Sprintf("%s%s%s%s %s(%s)%s",
		t.Primary, harness.Bold, h.live.Model, harness.Reset,
		harness.Dim, ctxHuman, harness.Reset)
	if h.live.InputTokens > 0 {
		result += fmt.Sprintf(" %s[%s]%s", t.Text, harness.HumanTokens(h.live.InputTokens), harness.Reset)
	}
	return result
}