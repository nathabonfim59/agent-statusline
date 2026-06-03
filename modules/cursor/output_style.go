package cursor

import (
	"github.com/nathabonfim59/agent-statusline/harness"
)

func renderOutputStyle(in *CursorInput, t harness.Theme) string {
	if in.OutputStyle.Name == "" {
		return ""
	}
	return harness.Dim + in.OutputStyle.Name + harness.Reset
}
