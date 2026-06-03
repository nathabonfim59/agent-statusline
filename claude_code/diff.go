package claude_code

import (
	"fmt"

	"github.com/nathabonfim59/agent-statusline/harness"
)

func renderDiff(in *Input, t harness.Theme) string {
	if in.Cost.TotalLinesAdded <= 0 && in.Cost.TotalLinesRemoved <= 0 {
		return ""
	}
	return fmt.Sprintf("%s+%d%s %s-%d%s",
		t.Success, in.Cost.TotalLinesAdded, harness.Reset,
		t.Danger, in.Cost.TotalLinesRemoved, harness.Reset)
}
