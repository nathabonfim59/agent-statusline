package shared

import (
	"github.com/nathabonfim59/claude-statusline/harness"
)

func Bar(pct float64, t harness.Theme, warn, danger float64) string {
	bar, _ := harness.ProgressBar(pct, t, warn, danger)
	return "[" + bar + "]"
}
