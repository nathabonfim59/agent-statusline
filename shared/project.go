package shared

import (
	"github.com/nathabonfim59/claude-statusline/harness"
)

func Project(projectDir string, t harness.Theme) string {
	name := projectDir
	if name == "" {
		return ""
	}
	for i := len(name) - 1; i >= 0; i-- {
		if name[i] == '/' || name[i] == '\\' {
			name = name[i+1:]
			break
		}
	}
	return t.Text + name + harness.Reset
}
