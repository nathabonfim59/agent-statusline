package cursor

import (
	"fmt"
	"strings"

	"github.com/nathabonfim59/agent-statusline/harness"
)

func renderModel(in *CursorInput, t harness.Theme) string {
	ctxHuman := harness.HumanTokens(in.ContextWindow.ContextWindowSize)

	var curInUse int
	if in.ContextWindow.CurrentUsage != nil {
		curInUse = in.ContextWindow.CurrentUsage.InputTokens
	}

	var parts []string

	modelPart := fmt.Sprintf("%s%s%s%s %s(%s context)%s",
		t.Primary, harness.Bold, in.Model.DisplayName, harness.Reset,
		harness.Dim, ctxHuman, harness.Reset)
	parts = append(parts, modelPart)

	if curInUse > 0 {
		parts = append(parts, fmt.Sprintf("%s[%s]%s", t.Text, harness.HumanTokens(curInUse), harness.Reset))
	}

	if in.Model.ParamSummary != nil && *in.Model.ParamSummary != "" {
		parts = append(parts, fmt.Sprintf("%s(%s)%s", harness.Dim, *in.Model.ParamSummary, harness.Reset))
	}

	if in.Model.MaxMode != nil && *in.Model.MaxMode {
		parts = append(parts, fmt.Sprintf("%s%sMAX%s", t.Warning, harness.Bold, harness.Reset))
	}

	return strings.Join(parts, " ")
}
