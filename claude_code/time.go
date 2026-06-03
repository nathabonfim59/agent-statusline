package claude_code

import (
	"fmt"

	"github.com/nathabonfim59/claude-statusline/harness"
)

func renderTime(in *Input, t harness.Theme) string {
	if in.Cost.TotalDurationMS <= 0 {
		return ""
	}
	elapsed := harness.HumanDuration(in.Cost.TotalDurationMS)
	if in.Cost.TotalAPIDurationMS > 0 {
		apiS := in.Cost.TotalAPIDurationMS / 1000
		return fmt.Sprintf("%s%s (api:%ds)%s", harness.Dim, elapsed, apiS, harness.Reset)
	}
	return harness.Dim + elapsed + harness.Reset
}
