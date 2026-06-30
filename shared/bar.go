package shared

import (
	"github.com/nathabonfim59/agent-statusline/harness"
)

// barBrackets controls whether the Bar block wraps the progress bar in
// "[" and "]" brackets. It defaults to false and can be toggled via
// SetBarBrackets from the resolved config.
var barBrackets = false

// SetBarBrackets enables or disables the "[" / "]" brackets around the
// context progress bar. Call once during startup from the resolved config.
func SetBarBrackets(enabled bool) { barBrackets = enabled }

func Bar(pct float64, t harness.Theme, warn, danger float64) string {
	bar, _ := harness.ProgressBar(pct, t, warn, danger)
	if !barBrackets {
		return bar
	}
	return "[" + bar + "]"
}
