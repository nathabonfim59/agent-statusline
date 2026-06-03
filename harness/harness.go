package harness

import (
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const (
	Reset = "\033[0m"
	Bold  = "\033[1m"
	Dim   = "\033[2m"

	Cyan   = "\033[36m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Red    = "\033[31m"
	White  = "\033[37m"
)

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

type Theme struct {
	Primary string
	Text    string
	Success string
	Warning string
	Danger  string
}

type Harness interface {
	Name() string
	Parse(raw []byte) error
	RenderBlock(name string, t Theme, pct, warn, danger float64) string
	ModelID() string
	ContextPct() float64
	TerminalWidth() int
	CWD() string
}

func VisibleWidth(s string) int {
	return len(ansiRe.ReplaceAllString(s, ""))
}

func HumanTokens(n int) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.0fM", float64(n)/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%.0fk", float64(n)/1_000)
	default:
		return strconv.Itoa(n)
	}
}

func HumanDuration(ms int64) string {
	s := ms / 1000
	m := s / 60
	s = s % 60
	return fmt.Sprintf("%dm%02ds", m, s)
}

func Repeat(s string, n int) string {
	if n <= 0 {
		return ""
	}
	return strings.Repeat(s, n)
}

func ProgressBar(pct float64, t Theme, warn, danger float64) (bar, pctPart string) {
	const barWidth = 20
	filled := int(math.Round(pct * barWidth / 100))
	if filled > barWidth {
		filled = barWidth
	}

	greenEnd := int(math.Round(warn * float64(barWidth) / 100))
	yellowEnd := int(math.Round(danger * float64(barWidth) / 100))

	g := min(filled, greenEnd)
	y := 0
	if filled > greenEnd {
		y = min(filled, yellowEnd) - greenEnd
	}
	r := 0
	if filled > yellowEnd {
		r = filled - yellowEnd
	}

	emptyBeforeThresh := 0
	if filled < greenEnd {
		emptyBeforeThresh = greenEnd - filled
	}
	emptyAfterThresh := barWidth - max(filled, greenEnd)

	var b strings.Builder
	if g > 0 {
		b.WriteString(t.Success + Repeat("█", g))
	}
	if emptyBeforeThresh > 0 {
		b.WriteString(Dim + Repeat("░", emptyBeforeThresh))
	}
	b.WriteString(t.Danger + "┃" + Reset)
	if y > 0 {
		b.WriteString(t.Warning + Repeat("█", y))
	}
	if r > 0 {
		b.WriteString(t.Danger + Repeat("█", r))
	}
	if emptyAfterThresh > 0 {
		b.WriteString(Dim + Repeat("░", emptyAfterThresh) + Reset)
	}
	bar = b.String()

	var col string
	switch {
	case pct >= danger:
		col = t.Danger
	case pct >= warn:
		col = t.Warning
	default:
		col = t.Success
	}
	pctPart = fmt.Sprintf("%s%s%.0f%%%s", col, Bold, pct, Reset)
	return
}

func GitInfo(dir string) (branch string, added, modified, untracked int, hash string) {
	run := func(args ...string) (string, error) {
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		out, err := cmd.Output()
		return strings.TrimSpace(string(out)), err
	}

	if _, err := run("rev-parse", "--git-dir"); err != nil {
		return
	}

	branch, _ = run("symbolic-ref", "--short", "HEAD")
	if branch == "" {
		branch, _ = run("rev-parse", "--short", "HEAD")
	}

	out, _ := run("status", "--porcelain")
	for _, line := range strings.Split(out, "\n") {
		if len(line) < 2 {
			continue
		}
		xy := line[:2]
		if xy == "??" {
			untracked++
			continue
		}
		x, y := rune(xy[0]), rune(xy[1])
		if x != ' ' && x != '.' {
			added++
		}
		if y != ' ' && y != '.' {
			modified++
		}
	}

	hash, _ = run("rev-parse", "--short=8", "HEAD")
	return
}

func ProjectVersion(dirs ...string) string {
	reVersion := regexp.MustCompile(`"version"\s*:\s*"([^"]+)"`)
	reToml := regexp.MustCompile(`(?m)^version\s*=\s*"([^"]+)"`)

	for _, dir := range dirs {
		if data, err := os.ReadFile(filepath.Join(dir, "package.json")); err == nil {
			if m := reVersion.FindSubmatch(data); m != nil {
				return "v" + string(m[1])
			}
		}
		for _, f := range []string{"Cargo.toml", "pyproject.toml"} {
			if data, err := os.ReadFile(filepath.Join(dir, f)); err == nil {
				if m := reToml.FindSubmatch(data); m != nil {
					return "v" + string(m[1])
				}
			}
		}
	}
	return ""
}
