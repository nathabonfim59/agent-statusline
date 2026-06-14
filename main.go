package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

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

var runningProxies = map[string]*proxy.ProxyServer{}

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
	root.AddCommand(devinCmd())

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

	fmt.Printf("%s\n%s\n", line1, line2)
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

	startCmd := &cobra.Command{
		Use:   fmt.Sprintf("start <%s>", strings.Join(available, "|")),
		Short: "Start proxy for a harness",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			label, _ := cmd.Flags().GetString("label")
			if label == "" {
				label = randomName()
			}
			proxyStart(args[0], label)
		},
	}
	startCmd.Flags().StringP("label", "l", "", "label for this proxy instance (random if not set)")

	stopCmd := &cobra.Command{
		Use:   "stop [label]",
		Short: "Stop a proxy by label, or all if no label given",
		Run: func(cmd *cobra.Command, args []string) {
			label := ""
			if len(args) > 0 {
				label = args[0]
			}
			proxyStop(label)
		},
	}

	statusCmd := &cobra.Command{
		Use:   "status [label]",
		Short: "Show proxy status and ports",
		Run: func(cmd *cobra.Command, args []string) {
			label := ""
			if len(args) > 0 {
				label = args[0]
			}
			proxyStatus(label)
		},
	}

	cmd.AddCommand(startCmd)
	cmd.AddCommand(stopCmd)
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
	cmd.AddCommand(statusCmd)

	return cmd
}

func devinCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "devin [label]",
		Short: "Render Devin CLI statusline from a running proxy session",
		Long: `Fetch live Devin session data from a proxy and render the statusline.
If a label is given, uses that specific proxy instance. Otherwise, uses the
most recently started Devin proxy.`,
		Run: func(cmd *cobra.Command, args []string) {
			label := ""
			if len(args) > 0 {
				label = args[0]
			}
			runDevinStatusline(label)
		},
	}
}

func runDevinStatusline(label string) {
	port := findDevinPort(label)
	if port == 0 {
		if label != "" {
			fmt.Fprintf(os.Stderr, "devin: no proxy with label '%s'\n", label)
		} else {
			fmt.Fprintf(os.Stderr, "devin: no live data (start proxy with: ./claude-statusline proxy start devin)\n")
		}
		os.Exit(1)
	}

	url := fmt.Sprintf("http://127.0.0.1:%d/data", port)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "devin: failed to fetch data: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "devin: failed to read response: %v\n", err)
		os.Exit(1)
	}

	h := harness.Detect(body)
	if h == nil {
		os.Exit(1)
	}

	cfg := loadConfig()
	t := loadTheme(cfg.Theme)
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

	fmt.Printf("%s\n%s\n", line1, line2)
}

func findDevinPort(label string) int {
	pattern := "/tmp/claude-statusline-devin-*.port"
	if label != "" {
		portFile := fmt.Sprintf("/tmp/claude-statusline-devin-%s.port", label)
		data, err := os.ReadFile(portFile)
		if err != nil {
			return 0
		}
		port := 0
		fmt.Sscanf(string(data), "%d", &port)
		return port
	}

	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) == 0 {
		return 0
	}

	sort.Slice(matches, func(i, j int) bool {
		iInfo, _ := os.Stat(matches[i])
		jInfo, _ := os.Stat(matches[j])
		return iInfo.ModTime().After(jInfo.ModTime())
	})

	data, err := os.ReadFile(matches[0])
	if err != nil {
		return 0
	}
	port := 0
	fmt.Sscanf(string(data), "%d", &port)
	return port
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

var adjectives = []string{"swift", "calm", "bright", "keen", "bold", "warm", "cool", "sharp", "quiet", "lively", "fresh", "eager", "gentle", "proud", "wild"}
var nouns = []string{"proxy", "relay", "bridge", "tunnel", "gate", "link", "node", "port", "route", "pulse", "spark", "flux", "beam", "wave", "stream"}

func randomName() string {
	return adjectives[time.Now().UnixNano()%int64(len(adjectives))] + "-" +
		nouns[time.Now().UnixNano()/int64(len(adjectives))%int64(len(nouns))]
}

func portFilePath(harness, label string) string {
	return fmt.Sprintf("/tmp/claude-statusline-%s-%s.port", harness, label)
}

func proxyStart(harnessName, label string) {
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

	key := harnessName + "/" + label
	runningProxies[key] = srv
	portFile := portFilePath(harnessName, label)
	os.WriteFile(portFile, []byte(fmt.Sprintf("%d", srv.DataPort())), 0o644)

	fmt.Printf("%sProxy started for %s%s%s (%s)%s\n", harness.Green, harness.Bold, harnessName, harness.Cyan, label, harness.Reset)
	fmt.Printf("  Proxy port: %s%d%s\n", harness.Cyan, srv.Port(), harness.Reset)
	fmt.Printf("  Data port:  %s%d%s\n", harness.Cyan, srv.DataPort(), harness.Reset)
	fmt.Printf("  Data URL:   %shttp://127.0.0.1:%d/data%s\n\n", harness.Dim, srv.DataPort(), harness.Reset)
	fmt.Printf("export HTTP_PROXY=http://127.0.0.1:%d\n", srv.Port())
	fmt.Printf("export HTTPS_PROXY=http://127.0.0.1:%d\n", srv.Port())
	fmt.Printf("\n%sRun ./claude-statusline devin %s to show statusline%s\n\n", harness.Dim, label, harness.Reset)
	fmt.Printf("./claude-statusline devin %s\n", label)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	fmt.Fprintf(os.Stderr, "\nShutting down %s...\n", label)
	srv.Stop()
	delete(runningProxies, key)
	os.Remove(portFile)
}

func proxyStop(label string) {
	if label == "" {
		for key, srv := range runningProxies {
			srv.Stop()
			delete(runningProxies, key)
		}
		fmt.Printf("%sAll proxies stopped.%s\n", harness.Green, harness.Reset)
		return
	}
	for key, srv := range runningProxies {
		if strings.HasSuffix(key, "/"+label) {
			srv.Stop()
			delete(runningProxies, key)
			fmt.Printf("%sProxy %s stopped.%s\n", harness.Green, label, harness.Reset)
			return
		}
	}
	fmt.Printf("%sNo proxy with label %s.%s\n", harness.Dim, label, harness.Reset)
}

func proxyStatus(label string) {
	if len(runningProxies) == 0 {
		fmt.Printf("%sno proxies running%s\n", harness.Dim, harness.Reset)
		return
	}
	for key, srv := range runningProxies {
		parts := strings.SplitN(key, "/", 2)
		if label != "" && parts[1] != label {
			continue
		}
		fmt.Printf("%s%s %s%s — proxy %s%d%s data %s%d%s\n",
			harness.Green, parts[0], harness.Cyan, parts[1],
			harness.Cyan, srv.Port(), harness.Reset,
			harness.Cyan, srv.DataPort(), harness.Reset)
	}
}