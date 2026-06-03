package shared

import (
	"github.com/nathabonfim59/agent-statusline/harness"
)

func Version(version string, cwd, projectDir string, t harness.Theme) string {
	if version != "" {
		return harness.Dim + "v" + version + harness.Reset
	}
	v := harness.ProjectVersion(cwd, projectDir)
	if v != "" {
		return harness.Dim + v + harness.Reset
	}
	return ""
}
