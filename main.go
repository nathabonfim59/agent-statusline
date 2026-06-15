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

func main() {
	root := &cobra.Command{
		Use:   "agent-statusline",
		Short: "Render status bars for AI coding agents",
		Long: `agent-statusline reads session data from stdin and renders a colored
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
			debug, _ := cmd.Flags().GetBool("debug")
			proxyStart(args[0], label, debug)
		},
	}
	startCmd.Flags().StringP("label", "l", "", "label for this proxy instance (random if not set)")
	startCmd.Flags().BoolP("debug", "d", false, "dump parsed message fields to stderr")

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

	daemonCmd := &cobra.Command{
		Use:    "daemon",
		Short:  "Run the proxy daemon (usually managed by systemd)",
		Hidden: true,
		Run: func(cmd *cobra.Command, args []string) {
			runProxyDaemon()
		},
	}

	installSystemdCmd := &cobra.Command{
		Use:   "install-systemd",
		Short: "Install the proxy daemon as a systemd service",
		Run: func(cmd *cobra.Command, args []string) {
			system, _ := cmd.Flags().GetBool("system")
			if err := proxy.InstallSystemdService(system); err != nil {
				fmt.Fprintf(os.Stderr, "%sinstall-systemd: %v%s\n", harness.Red, err, harness.Reset)
				os.Exit(1)
			}
		},
	}
	installSystemdCmd.Flags().Bool("user", true, "install as a user service (default)")
	installSystemdCmd.Flags().Bool("system", false, "install as a system service (requires root and agent-statusline user)")

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
	cmd.AddCommand(daemonCmd)
	cmd.AddCommand(installSystemdCmd)

	return cmd
}

func runProxyDaemon() {
	d, err := proxy.NewDaemon()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%sfailed to create daemon: %v%s\n", harness.Red, err, harness.Reset)
		os.Exit(1)
	}
	if err := d.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "%sfailed to start daemon: %v%s\n", harness.Red, err, harness.Reset)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "agent-statusline proxy daemon listening on %s\n", proxy.SocketPath())

	// Block until a signal arrives. The daemon goroutine owns the listeners;
	// this goroutine just keeps the process alive.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
	d.Stop()
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
	client := proxy.NewClient()

	if label == "" {
		infos, err := client.Status()
		if err != nil {
			fmt.Fprintf(os.Stderr, "devin: %v\n", err)
			os.Exit(1)
		}
		var recent *proxy.InstanceInfo
		for _, info := range infos {
			if info.Harness != "devin" {
				continue
			}
			if recent == nil || info.StartedAt.After(recent.StartedAt) {
				recent = info
			}
		}
		if recent == nil {
			fmt.Fprintf(os.Stderr, "devin: no live data (start proxy with: agent-statusline proxy start devin)\n")
			os.Exit(1)
		}
		label = recent.Label
	}

	body, err := client.Data("devin", label)
	if err != nil {
		fmt.Fprintf(os.Stderr, "devin: failed to fetch data: %v\n", err)
		os.Exit(1)
	}

	h := harness.Detect(body)
	if h == nil {
		fmt.Fprintf(os.Stderr, "devin: no usable data from proxy (is Devin sending traffic?)\n")
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

func availableHarnesses() []string {
	var available []string
	for _, h := range []harness.Harness{devin.New(), cursor.New(), claude_code.New()} {
		if h.ProxyConfig() != nil {
			available = append(available, h.Name())
		}
	}
	return available
}

func proxyStart(harnessName, label string, debug bool) {
	client := proxy.NewClient()
	info, err := client.Start(harnessName, label, debug)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%sfailed to start proxy: %v%s\n", harness.Red, err, harness.Reset)
		os.Exit(1)
	}

	fmt.Printf("%sProxy started for %s%s%s (%s)%s\n", harness.Green, harness.Bold, harnessName, harness.Cyan, info.Label, harness.Reset)
	fmt.Printf("  Proxy port: %s%d%s\n", harness.Cyan, info.ProxyPort, harness.Reset)
	fmt.Printf("  Data port:  %s%d%s\n", harness.Cyan, info.DataPort, harness.Reset)
	fmt.Printf("  Data URL:   %shttp://127.0.0.1:%d/data%s\n\n", harness.Dim, info.DataPort, harness.Reset)
	fmt.Printf("export HTTP_PROXY=http://127.0.0.1:%d\n", info.ProxyPort)
	fmt.Printf("export HTTPS_PROXY=http://127.0.0.1:%d\n", info.ProxyPort)
	fmt.Printf("\n%sRun the command below to show the statusline%s\n\n", harness.Dim, harness.Reset)
	fmt.Printf("agent-statusline devin %s\n", info.Label)
}

func proxyStop(label string) {
	client := proxy.NewClient()
	stopped, err := client.Stop(label)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%sfailed to stop proxy: %v%s\n", harness.Red, err, harness.Reset)
		os.Exit(1)
	}
	if len(stopped) == 0 {
		fmt.Printf("%sNo proxy with label %s.%s\n", harness.Dim, label, harness.Reset)
		return
	}
	for _, key := range stopped {
		parts := strings.SplitN(key, "/", 2)
		display := key
		if len(parts) == 2 {
			display = parts[1]
		}
		fmt.Printf("%sProxy %s stopped.%s\n", harness.Green, display, harness.Reset)
	}
}

func proxyStatus(label string) {
	client := proxy.NewClient()
	infos, err := client.Status()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%sfailed to get status: %v%s\n", harness.Red, err, harness.Reset)
		os.Exit(1)
	}
	if len(infos) == 0 {
		fmt.Printf("%sno proxies running%s\n", harness.Dim, harness.Reset)
		return
	}
	for _, info := range infos {
		if label != "" && info.Label != label {
			continue
		}
		fmt.Printf("%s%s %s%s — proxy %s%d%s data %s%d%s\n",
			harness.Green, info.Harness, harness.Cyan, info.Label,
			harness.Cyan, info.ProxyPort, harness.Reset,
			harness.Cyan, info.DataPort, harness.Reset)
	}
}
