package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strings"

"github.com/nathabonfim59/agent-statusline/harness"
	"github.com/nathabonfim59/agent-statusline/shared"

	_ "github.com/nathabonfim59/agent-statusline/claude_code"

	_ "github.com/nathabonfim59/agent-statusline/modules/cursor"
)

func buildLine(order []string, blocks map[string]string, compact []string, tw int) string {
	sep := harness.Dim + "|" + harness.Reset
	sepLen := harness.VisibleWidth(" " + sep + " ")

	var parts []string
	for _, name := range order {
		if s, ok := blocks[name]; ok {
			parts = append(parts, s)
		}
	}
	line := strings.Join(parts, " "+sep+" ")
	if harness.VisibleWidth(line) <= tw {
		return line
	}

	lineSet := make(map[string]bool)
	for _, name := range order {
		lineSet[name] = true
	}
	var compactParts []string
	for _, name := range compact {
		if lineSet[name] {
			if s, ok := blocks[name]; ok {
				compactParts = append(compactParts, s)
			}
		}
	}

	var fit []string
	w := 0
	for _, s := range compactParts {
		need := harness.VisibleWidth(s)
		if len(fit) > 0 {
			need += sepLen
		}
		if w+need > tw {
			break
		}
		fit = append(fit, s)
		w += need
	}
	if len(fit) == 0 && len(compactParts) > 0 {
		fit = compactParts[:1]
	}
	return strings.Join(fit, " "+sep+" ")
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "init" {
		runInit()
		return
	}

	cfg := loadConfig()
	t := loadTheme(cfg.Theme)
	shared.SetBarBrackets(barBracketsEnabled(cfg))

	var buf bytes.Buffer
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		buf.Write(scanner.Bytes())
	}

	h := harness.Detect(buf.Bytes())
	if h == nil {
		os.Exit(1)
	}

	warn, danger := resolveThresholds(cfg, h.ModelID())
	blockCfg := resolveBlocks(cfg, h.Name())
	pct := h.ContextPct()
	tw := h.TerminalWidth()

	rendered := make(map[string]string)
	for _, list := range [][]string{blockCfg.Line1, blockCfg.Line2} {
		for _, name := range list {
			if _, ok := rendered[name]; ok {
				continue
			}
			if s := h.RenderBlock(name, t, pct, warn, danger); s != "" {
				rendered[name] = s
			}
		}
	}

	line1 := buildLine(blockCfg.Line1, rendered, blockCfg.Compact, tw)
	line2 := buildLine(blockCfg.Line2, rendered, blockCfg.Compact, tw)

	fmt.Printf("%s\n%s", line1, line2)
}
