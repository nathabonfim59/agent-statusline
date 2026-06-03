package claude_code

import (
	"fmt"

	"github.com/nathabonfim59/claude-statusline/harness"
)

func renderTokens(in *Input, t harness.Theme) string {
	cu := in.ContextWindow.CurrentUsage
	totalCur := cu.InputTokens + cu.CacheReadInputTokens + cu.CacheCreationInputTokens
	if totalCur <= 0 {
		return ""
	}
	cachePct := float64(cu.CacheReadInputTokens) / float64(totalCur) * 100
	return fmt.Sprintf("%s%s cache:%.0f%%%s", harness.Dim, harness.HumanTokens(totalCur), cachePct, harness.Reset)
}
