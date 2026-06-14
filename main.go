package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/nathabonfim59/agent-statusline/claude_code"
	"github.com/nathabonfim59/agent-statusline/harness"
	"github.com/nathabonfim59/agent-statusline/modules/cursor"
	"github.com/nathabonfim59/agent-statusline/modules/devin"
	"github.com/nathabonfim59/agent-statusline/proxy"

	// Register harness detectors via init()
	_ "github.com/nathabonfim59/agent-statusline/claude_code"
	_ "github.com/nathabonfim59/agent-statusline/modules/cursor"
	_ "github.com/nathabonfim59/agent-statusline/modules/devin"
)

var runningProxy *proxy.ProxyServer

func main() {
	root := &cobra.Command{
		Use:   "claude-statusline",
		Short: "Render status bars for AI coding agents",
		Long: `claude-statusline reads session data from stdin and renders a colored
two-line ANSI status bar showing model info, token usage, cost, and more.

Supported agents: Claude Code, Cursor, Devin CLI`,
		Run: runFilter,
	}

	root.AddCommand(initCmd())
	root.AddCommand(proxyCmd())

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func runFilter(cmd *cobra.Command, args []string) {
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

func initCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize config directory with defaults",
		Run: func(cmd *cobra.Command, args []string) {
			runInit()
		},
	}
}

func proxyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "proxy",
		Short: "Manage HTTPS proxy for intercepting agent traffic",
	}

	available := availableHarnesses()

	cmd.AddCommand(&cobra.Command{
		Use:   fmt.Sprintf("start <%s>", strings.Join(available, "|")),
		Short: "Start proxy for a harness",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			proxyStart(args[0])
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "stop",
		Short: "Stop the running proxy",
		Run: func(cmd *cobra.Command, args []string) {
			proxyStop()
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "install-ca",
		Short: "Show instructions to install the CA certificate",
		Run: func(cmd *cobra.Command, args []string) {
			if err := proxy.InstallCA(); err != nil {
				fmt.Fprintf(os.Stderr, "install-ca: %v\n", err)
				os.Exit(1)
			}
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "Show proxy status and ports",
		Run: func(cmd *cobra.Command, args []string) {
			proxyStatus()
		},
	})

	return cmd
}

func availableHarnesses() []string {
	var available []string
	for _, h := range []harness.Harness{devin.New(), cursor.New(), claude_code.New()} {
		if h.ProxyConfig() != nil {
			available = append(available, h.Name())
		}
	}
	return available
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
		fmt.Fprintf(os.Stderr, "%sunknown harness: %s%s\n", harness.Red, harnessName, harness.Reset)
		os.Exit(1)
	}

	cfg := h.ProxyConfig()
	if cfg == nil {
		fmt.Fprintf(os.Stderr, "%sharness %s does not support proxy%s\n", harness.Red, harnessName, harness.Reset)
		os.Exit(1)
	}

	srv, err := proxy.Start(*cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%sfailed to start proxy: %v%s\n", harness.Red, err, harness.Reset)
		os.Exit(1)
	}
	runningProxy = srv

	os.WriteFile("/tmp/claude-statusline-devin-data.port", []byte(fmt.Sprintf("%d", srv.DataPort())), 0o644)

	fmt.Printf("%sProxy started for %s%s%s\n", harness.Green, harness.Bold, harnessName, harness.Reset)
	fmt.Printf("  Proxy port: %s%d%s\n", harness.Cyan, srv.Port(), harness.Reset)
	fmt.Printf("  Data port:  %s%d%s\n", harness.Cyan, srv.DataPort(), harness.Reset)
	fmt.Printf("  Set: %sexport HTTP_PROXY=http://127.0.0.1:%d%s\n", harness.Green, srv.Port(), harness.Reset)
	fmt.Printf("  Set: %sexport HTTPS_PROXY=http://127.0.0.1:%d%s\n", harness.Green, srv.Port(), harness.Reset)
	fmt.Printf("  Data URL: %shttp://127.0.0.1:%d/data%s\n", harness.Dim, srv.DataPort(), harness.Reset)

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
		fmt.Printf("%sProxy stopped.%s\n", harness.Green, harness.Reset)
	} else {
		fmt.Printf("%sNo proxy running.%s\n", harness.Dim, harness.Reset)
	}
}

func proxyStatus() {
	if runningProxy != nil {
		fmt.Printf("%sproxy running — data port %s%d%s\n", harness.Green, harness.Cyan, runningProxy.DataPort(), harness.Reset)
	} else {
		fmt.Printf("%sproxy not running%s\n", harness.Dim, harness.Reset)
	}
}