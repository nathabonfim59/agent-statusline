package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/nathabonfim59/agent-statusline/claude_code"
	"github.com/nathabonfim59/agent-statusline/harness"
	"github.com/nathabonfim59/agent-statusline/modules/cursor"
	"github.com/nathabonfim59/agent-statusline/modules/devin"
	"github.com/nathabonfim59/agent-statusline/proxy"
)

var runningProxy *proxy.ProxyServer

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
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "init":
			runInit()
			return
		case "proxy":
			runProxy(os.Args[2:])
			return
		}
	}

	cfg := loadConfig()
	t := loadTheme(cfg.Theme)

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

func runProxy(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: claude-statusline proxy <start|stop|install-ca|status> [harness]")
		os.Exit(1)
	}

	switch args[0] {
	case "start":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: claude-statusline proxy start <harness>")
			os.Exit(1)
		}
		proxyStart(args[1])
	case "stop":
		proxyStop()
	case "install-ca":
		if err := proxy.InstallCA(); err != nil {
			fmt.Fprintf(os.Stderr, "install-ca: %v\n", err)
			os.Exit(1)
		}
	case "status":
		proxyStatus()
	default:
		fmt.Fprintf(os.Stderr, "unknown proxy command: %s\n", args[0])
		os.Exit(1)
	}
}

func proxyStart(harnessName string) {
	var h harness.Harness
	switch harnessName {
	case "devin":
		h = devin.New()
	case "cursor":
		h = cursor.New()
	case "claude_code":
		h = claude_code.New()
	default:
		fmt.Fprintf(os.Stderr, "unknown harness: %s\n", harnessName)
		os.Exit(1)
	}

	cfg := h.ProxyConfig()
	if cfg == nil {
		fmt.Fprintf(os.Stderr, "harness %s does not support proxy\n", harnessName)
		os.Exit(1)
	}

	srv, err := proxy.Start(*cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to start proxy: %v\n", err)
		os.Exit(1)
	}
	runningProxy = srv

	// Write port file for shell scripts
	os.WriteFile("/tmp/claude-statusline-devin-data.port", []byte(fmt.Sprintf("%d", srv.DataPort())), 0o644)

	fmt.Printf("Proxy started for %s\n", harnessName)
	fmt.Printf("  Proxy port: %d\n", srv.Port())
	fmt.Printf("  Data port:  %d\n", srv.DataPort())
	fmt.Printf("  Set: export HTTP_PROXY=http://127.0.0.1:%d\n", srv.Port())
	fmt.Printf("  Set: export HTTPS_PROXY=http://127.0.0.1:%d\n", srv.Port())
	fmt.Printf("  Data URL: http://127.0.0.1:%d/data\n", srv.DataPort())

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	fmt.Fprintf(os.Stderr, "\nShutting down...\n")
	srv.Stop()
	os.Remove("/tmp/claude-statusline-devin-data.port")
}

func proxyStop() {
	if runningProxy != nil {
		runningProxy.Stop()
		runningProxy = nil
		fmt.Println("Proxy stopped.")
	} else {
		fmt.Println("No proxy running.")
	}
}

func proxyStatus() {
	if runningProxy != nil {
		data, _ := json.MarshalIndent(map[string]interface{}{
			"status": "running",
			"port":   runningProxy.DataPort(),
		}, "", "  ")
		fmt.Println(string(data))
	} else {
		fmt.Println(`{"status": "not running"}`)
	}
}