package cursor

import (
	"github.com/nathabonfim59/agent-statusline/harness"
)

func renderWorktree(in *CursorInput, t harness.Theme) string {
	if in.Worktree == nil || in.Worktree.Name == "" {
		return ""
	}
	return t.Warning + in.Worktree.Name + harness.Reset
}
