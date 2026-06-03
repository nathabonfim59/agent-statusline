package cursor

import (
	"github.com/nathabonfim59/claude-statusline/harness"
)

func renderAutorun(in *CursorInput, t harness.Theme) string {
	if !in.Autorun {
		return ""
	}
	return t.Warning + harness.Bold + "AUTO" + harness.Reset
}
