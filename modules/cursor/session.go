package cursor

import (
	"github.com/nathabonfim59/agent-statusline/harness"
)

func renderSession(in *CursorInput, t harness.Theme) string {
	if in.SessionName == nil || *in.SessionName == "" {
		return ""
	}
	return t.Text + *in.SessionName + harness.Reset
}
