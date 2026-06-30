package claude_code

import (
	"fmt"

	"github.com/nathabonfim59/agent-statusline/harness"
)

func renderTokens(in *Input, t harness.Theme) string {
	cu := in.ContextWindow.CurrentUsage
	curTurnTotal := cu.InputTokens + cu.CacheReadInputTokens + cu.CacheCreationInputTokens
	total := in.ContextWindow.TotalInputTokens
	if total == 0 {
		total = curTurnTotal
	}
	if total <= 0 {
		return ""
	}
	cachePct := float64(0)
	if curTurnTotal > 0 {
		cachePct = float64(cu.CacheReadInputTokens) / float64(curTurnTotal) * 100
	}
	return fmt.Sprintf("%s%s cache:%.0f%%%s", harness.Dim, harness.HumanTokens(total), cachePct, harness.Reset)
}
