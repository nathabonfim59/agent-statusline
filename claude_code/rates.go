package claude_code

import (
	"fmt"
	"strings"

	"github.com/nathabonfim59/agent-statusline/harness"
)

func renderRates(in *Input, t harness.Theme) string {
	fh := in.RateLimits.FiveHour.UsedPercentage
	sd := in.RateLimits.SevenDay.UsedPercentage
	if fh <= 0 && sd <= 0 {
		return ""
	}

	var parts []string
	if fh > 0 {
		parts = append(parts, fmt.Sprintf("5h:%.0f%%", fh))
	}
	if sd > 0 {
		parts = append(parts, fmt.Sprintf("7d:%.0f%%", sd))
	}
	return harness.Dim + strings.Join(parts, " ") + harness.Reset
}
