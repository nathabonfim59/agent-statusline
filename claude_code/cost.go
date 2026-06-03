package claude_code

import (
	"fmt"

	"github.com/nathabonfim59/agent-statusline/harness"
)

func renderCost(in *Input, t harness.Theme) string {
	if in.Cost.TotalCostUSD <= 0 {
		return harness.Dim + "$?" + harness.Reset
	}
	c := in.Cost.TotalCostUSD
	if c < 0.01 {
		return fmt.Sprintf("%s$%.4f%s", t.Text, c, harness.Reset)
	}
	return fmt.Sprintf("%s$%.2f%s", t.Text, c, harness.Reset)
}
