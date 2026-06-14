package devin

import (
	"fmt"

	"github.com/nathabonfim59/agent-statusline/harness"
)

func renderTokens(h *Harness, t harness.Theme) string {
	if h.live.InputTokens <= 0 && h.live.OutputTokens <= 0 {
		return ""
	}
	return fmt.Sprintf("%s%s in%s  %s%s out%s",
		harness.Dim, humanTokensCeil(h.live.InputTokens), harness.Reset,
		harness.Dim, humanTokensCeil(h.live.OutputTokens), harness.Reset)
}
