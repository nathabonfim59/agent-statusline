package shared

import (
	"github.com/nathabonfim59/agent-statusline/harness"
)

func Percent(pct float64, t harness.Theme, warn, danger float64) string {
	_, pctStr := harness.ProgressBar(pct, t, warn, danger)
	return pctStr
}
