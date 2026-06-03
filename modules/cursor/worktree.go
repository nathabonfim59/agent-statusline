package cursor

import (
	"github.com/nathabonfim59/claude-statusline/harness"
)

func renderWorktree(in *CursorInput, t harness.Theme) string {
	if in.Worktree == nil || in.Worktree.Name == "" {
		return ""
	}
	return t.Warning + in.Worktree.Name + harness.Reset
}
