package claude_code

import (
	"fmt"
	"strings"

	"github.com/nathabonfim59/agent-statusline/harness"
)

func renderModel(in *Input, t harness.Theme) string {
	ctxHuman := harness.HumanTokens(in.ContextWindow.ContextWindowSize)

	ctxInUse := in.ContextWindow.CurrentUsage.CacheReadInputTokens +
		in.ContextWindow.CurrentUsage.CacheCreationInputTokens +
		in.ContextWindow.CurrentUsage.InputTokens

	result := fmt.Sprintf("%s%s%s%s", t.Primary, harness.Bold, in.Model.DisplayName, harness.Reset)
	if !strings.Contains(in.Model.DisplayName, "context") {
		result += fmt.Sprintf(" %s(%s context)%s", harness.Dim, ctxHuman, harness.Reset)
	}
	if ctxInUse > 0 {
		result += fmt.Sprintf(" %s[%s]%s", t.Text, harness.HumanTokens(ctxInUse), harness.Reset)
	}
	return result
}
