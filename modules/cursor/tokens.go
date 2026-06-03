package cursor

import (
	"fmt"

	"github.com/nathabonfim59/claude-statusline/harness"
)

func renderTokens(in *CursorInput, t harness.Theme) string {
	if in.ContextWindow.CurrentUsage == nil {
		return ""
	}
	totalCur := in.ContextWindow.CurrentUsage.InputTokens
	if totalCur <= 0 {
		return ""
	}
	return fmt.Sprintf("%s%s%s", harness.Dim, harness.HumanTokens(totalCur), harness.Reset)
}
