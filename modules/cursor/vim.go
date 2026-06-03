package cursor

import (
	"github.com/nathabonfim59/agent-statusline/harness"
)

func renderVim(in *CursorInput, t harness.Theme) string {
	if in.Vim == nil || in.Vim.Mode == "" {
		return ""
	}
	return harness.Dim + "[" + in.Vim.Mode + "]" + harness.Reset
}
