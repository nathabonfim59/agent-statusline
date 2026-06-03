package shared

import (
	"github.com/nathabonfim59/agent-statusline/harness"
)

func Hash(cwd string, t harness.Theme) string {
	_, _, _, _, hash := harness.GitInfo(cwd)
	if hash == "" {
		return ""
	}
	return harness.Dim + hash + harness.Reset
}
