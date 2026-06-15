package devin

import (
	"fmt"

	"github.com/nathabonfim59/agent-statusline/harness"
)

func renderQuota(h *Harness, t harness.Theme) string {
	q := h.live.Quota
	if q == nil || q.Plan == "" {
		return ""
	}
	if q.DailyLimit > 0 && q.DailyLimit < 1_000_000_000_000 {
		remaining := 100 - q.DailyUsed*100/q.DailyLimit
		return fmt.Sprintf("%s%s%s %s%d%%%s",
			t.Text, q.Plan, harness.Reset,
			harness.Dim, remaining, harness.Reset)
	}
	return fmt.Sprintf("%s%s%s %sunlimited%s",
		t.Text, q.Plan, harness.Reset,
		harness.Dim, harness.Reset)
}
