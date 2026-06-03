package shared

import (
	"fmt"
	"strings"

	"github.com/nathabonfim59/claude-statusline/harness"
)

func Git(cwd string, t harness.Theme) string {
	branch, added, modified, untracked, _ := harness.GitInfo(cwd)
	if branch == "" {
		return ""
	}

	counts := ""
	if added > 0 {
		counts += fmt.Sprintf("+%d ", added)
	}
	if modified > 0 {
		counts += fmt.Sprintf("~%d ", modified)
	}
	if untracked > 0 {
		counts += fmt.Sprintf("?%d", untracked)
	}
	counts = strings.TrimSpace(counts)
	if counts != "" {
		return fmt.Sprintf("%s%s%s %s%s%s", t.Success, branch, harness.Reset, harness.Dim, counts, harness.Reset)
	}
	return fmt.Sprintf("%s%s%s", t.Success, branch, harness.Reset)
}
