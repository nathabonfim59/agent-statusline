package cursor

import (
	"github.com/nathabonfim59/claude-statusline/harness"
)

func renderVim(in *CursorInput, t harness.Theme) string {
	if in.Vim == nil || in.Vim.Mode == "" {
		return ""
	}
	return harness.Dim + "[" + in.Vim.Mode + "]" + harness.Reset
}
