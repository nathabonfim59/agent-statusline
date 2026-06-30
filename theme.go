package main

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/nathabonfim59/agent-statusline/harness"
	"gopkg.in/yaml.v3"
)

//go:embed themes/*.yaml
var builtinThemesFS embed.FS

//go:embed config.example.yaml
var sampleConfig []byte

type ThemeColors struct {
	Primary string `yaml:"primary"`
	Text    string `yaml:"text"`
	Success string `yaml:"success"`
	Warning string `yaml:"warning"`
	Danger  string `yaml:"danger"`
}

type ThemeFile struct {
	Name   string      `yaml:"name"`
	Colors ThemeColors `yaml:"colors"`
}

type BlockConfig struct {
	Line1   []string `yaml:"line1"`
	Line2   []string `yaml:"line2"`
	Compact []string `yaml:"compact"`
}

type ThresholdConfig struct {
	Warning float64 `yaml:"warning"`
	Danger  float64 `yaml:"danger"`
}

type HarnessConfig struct {
	Extends bool        `yaml:"extends"`
	Blocks  BlockConfig `yaml:"blocks"`
}

// BarConfig configures the context progress bar block.
// Brackets is a pointer so an unset value can default to true.
type BarConfig struct {
	Brackets *bool `yaml:"brackets"`
}

type Config struct {
	Theme      string                     `yaml:"theme"`
	Thresholds map[string]ThresholdConfig `yaml:"thresholds"`
	Blocks     BlockConfig                `yaml:"blocks"`
	Bar        BarConfig                  `yaml:"bar"`
	ClaudeCode HarnessConfig              `yaml:"claude_code"`
	Cursor     HarnessConfig              `yaml:"cursor"`
}

var builtinDefault = ThemeFile{
	Name: "Default",
	Colors: ThemeColors{
		Primary: "cyan",
		Text:    "white",
		Success: "green",
		Warning: "yellow",
		Danger:  "red",
	},
}

func configDir() string {
	if runtime.GOOS == "windows" {
		if dir, err := os.UserConfigDir(); err == nil {
			return filepath.Join(dir, "claude-statusline")
		}
	}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "claude-statusline")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "claude-statusline")
}

func runInit() {
	dir := configDir()
	themesDir := filepath.Join(dir, "themes")
	if err := os.MkdirAll(themesDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "error creating config directory: %v\n", err)
		os.Exit(1)
	}

	cfgPath := filepath.Join(dir, "config.yaml")
	if _, err := os.Stat(cfgPath); err == nil {
		fmt.Printf("config already exists at %s\n", cfgPath)
		return
	}

	if err := os.WriteFile(cfgPath, sampleConfig, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "error writing config: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("created config at %s\n", cfgPath)
}

func loadConfig() Config {
	cfg := Config{
		Theme: "default",
		Blocks: BlockConfig{
			Line1:   []string{"model", "git", "project", "version"},
			Line2:   []string{"bar", "percent", "cost", "time", "tokens", "rates", "diff", "hash"},
			Compact: []string{"model", "bar", "percent", "cost", "git", "project", "hash", "time", "tokens", "rates", "diff", "version"},
		},
	}
	data, err := os.ReadFile(filepath.Join(configDir(), "config.yaml"))
	if err != nil {
		return cfg
	}
	_ = yaml.Unmarshal(data, &cfg)
	if len(cfg.Blocks.Line1) == 0 {
		cfg.Blocks.Line1 = []string{"model", "git", "project", "version"}
	}
	if len(cfg.Blocks.Line2) == 0 {
		cfg.Blocks.Line2 = []string{"bar", "percent", "cost", "time", "tokens", "rates", "diff", "hash"}
	}
	if len(cfg.Blocks.Compact) == 0 {
		cfg.Blocks.Compact = []string{"model", "bar", "percent", "cost", "git", "project", "hash", "time", "tokens", "rates", "diff", "version"}
	}
	return cfg
}

func resolveBlocks(cfg Config, harnessName string) BlockConfig {
	var hc HarnessConfig
	switch harnessName {
	case "claude_code":
		hc = cfg.ClaudeCode
	case "cursor":
		hc = cfg.Cursor
	default:
		return cfg.Blocks
	}

	if len(hc.Blocks.Line1) == 0 && len(hc.Blocks.Line2) == 0 && len(hc.Blocks.Compact) == 0 {
		return cfg.Blocks
	}
	if !hc.Extends {
		return hc.Blocks
	}
	result := cfg.Blocks
	if len(hc.Blocks.Line1) > 0 {
		result.Line1 = hc.Blocks.Line1
	}
	if len(hc.Blocks.Line2) > 0 {
		result.Line2 = hc.Blocks.Line2
	}
	if len(hc.Blocks.Compact) > 0 {
		result.Compact = hc.Blocks.Compact
	}
	return result
}

func resolveColor(val string) string {
	switch strings.ToLower(strings.TrimSpace(val)) {
	case "cyan":
		return harness.Cyan
	case "green":
		return harness.Green
	case "yellow":
		return harness.Yellow
	case "red":
		return harness.Red
	case "white":
		return harness.White
	case "dim":
		return harness.Dim
	case "bold":
		return harness.Bold
	case "reset", "default", "":
		return harness.Reset
	}
	if strings.HasPrefix(val, "#") && len(val) == 7 {
		r := hexNibble(val[1:3])
		g := hexNibble(val[3:5])
		b := hexNibble(val[5:7])
		return fmt.Sprintf("\033[38;2;%d;%d;%dm", r, g, b)
	}
	return val
}

func hexNibble(s string) int {
	v := 0
	for _, c := range strings.ToLower(s) {
		v <<= 4
		switch {
		case c >= '0' && c <= '9':
			v |= int(c - '0')
		case c >= 'a' && c <= 'f':
			v |= int(c-'a') + 10
		}
	}
	return v
}

func resolveTheme(tf ThemeFile) harness.Theme {
	return harness.Theme{
		Primary: resolveColor(tf.Colors.Primary),
		Text:    resolveColor(tf.Colors.Text),
		Success: resolveColor(tf.Colors.Success),
		Warning: resolveColor(tf.Colors.Warning),
		Danger:  resolveColor(tf.Colors.Danger),
	}
}

func loadTheme(name string) harness.Theme {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		name = "default"
	}

	localPath := filepath.Join(configDir(), "themes", name+".yaml")
	if data, err := os.ReadFile(localPath); err == nil {
		var tf ThemeFile
		if yaml.Unmarshal(data, &tf) == nil && tf.Colors.Primary != "" {
			return resolveTheme(tf)
		}
	}

	if data, err := builtinThemesFS.ReadFile("themes/" + name + ".yaml"); err == nil {
		var tf ThemeFile
		if yaml.Unmarshal(data, &tf) == nil {
			return resolveTheme(tf)
		}
	}

	return resolveTheme(builtinDefault)
}

func resolveThresholds(cfg Config, modelID string) (warn, danger float64) {
	warn, danger = 50, 75
	if cfg.Thresholds == nil {
		return
	}
	if d, ok := cfg.Thresholds["default"]; ok {
		if d.Warning > 0 {
			warn = d.Warning
		}
		if d.Danger > 0 {
			danger = d.Danger
		}
	}
	if t, ok := cfg.Thresholds[modelID]; ok {
		if t.Warning > 0 {
			warn = t.Warning
		}
		if t.Danger > 0 {
			danger = t.Danger
		}
	}
	return
}

func getHarnessConfig(cfg Config, name string) *HarnessConfig {
	switch name {
	case "claude_code":
		return &cfg.ClaudeCode
	case "cursor":
		return &cfg.Cursor
	}
	return nil
}

// barBracketsEnabled reports whether the context progress bar should be
// wrapped in "[" / "]" brackets. Defaults to true when unset.
func barBracketsEnabled(cfg Config) bool {
	if cfg.Bar.Brackets == nil {
		return true
	}
	return *cfg.Bar.Brackets
}
