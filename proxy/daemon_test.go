package proxy

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/nathabonfim59/agent-statusline/harness"
)

type mockCollector struct {
	id string
}

func (c *mockCollector) HandleResponse(host, path, contentType string, body []byte) {}
func (c *mockCollector) GetData() interface{}                                       { return c.id }
func (c *mockCollector) SetDebug(on bool)                                           {}

type mockHarness struct {
	id string
}

func (h *mockHarness) Name() string           { return "mock" }
func (h *mockHarness) Parse(raw []byte) error { return nil }
func (h *mockHarness) RenderBlock(name string, t harness.Theme, pct, warn, danger float64) string {
	return ""
}
func (h *mockHarness) ModelID() string     { return "" }
func (h *mockHarness) ContextPct() float64 { return 0 }
func (h *mockHarness) TerminalWidth() int  { return 80 }
func (h *mockHarness) CWD() string         { return "" }
func (h *mockHarness) ProxyConfig() *harness.ProxyConfig {
	return &harness.ProxyConfig{
		Domains:   []string{"example.com"},
		Collector: &mockCollector{id: h.id},
	}
}

var mockCounter int

func newMockHarness() harness.Harness {
	mockCounter++
	return &mockHarness{id: fmt.Sprintf("mock-%d", mockCounter)}
}

func init() {
	harness.RegisterNamed("mock", newMockHarness)
}

func withTestSocket(t *testing.T) func() {
	socketPath := fmt.Sprintf("%s/agent-statusline-test-%d.sock", os.TempDir(), time.Now().UnixNano())
	old := socketPathOverride
	socketPathOverride = socketPath
	return func() {
		socketPathOverride = old
		os.Remove(socketPath)
	}
}

func TestDaemonInstanceIsolation(t *testing.T) {
	cleanup := withTestSocket(t)
	defer cleanup()

	d, err := NewDaemon()
	if err != nil {
		t.Fatalf("NewDaemon: %v", err)
	}
	if err := d.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer d.Stop()

	a, err := d.StartInstance("mock", "a", false)
	if err != nil {
		t.Fatalf("StartInstance a: %v", err)
	}
	b, err := d.StartInstance("mock", "b", false)
	if err != nil {
		t.Fatalf("StartInstance b: %v", err)
	}

	if a.ProxyPort == b.ProxyPort || a.DataPort == b.DataPort {
		t.Fatalf("instances must have different ports: a=%+v b=%+v", a, b)
	}

	if got := d.InstanceData("mock", "a"); got != "mock-1" {
		t.Fatalf("instance a data = %v, want mock-1", got)
	}
	if got := d.InstanceData("mock", "b"); got != "mock-2" {
		t.Fatalf("instance b data = %v, want mock-2", got)
	}

	stopped := d.StopInstance("a")
	if len(stopped) != 1 {
		t.Fatalf("expected 1 stopped, got %d", len(stopped))
	}
	if d.InstanceData("mock", "a") != nil {
		t.Fatalf("instance a should be gone")
	}
	if d.InstanceData("mock", "b") == nil {
		t.Fatalf("instance b should still exist")
	}
}

func TestDaemonDuplicateLabel(t *testing.T) {
	cleanup := withTestSocket(t)
	defer cleanup()

	d, err := NewDaemon()
	if err != nil {
		t.Fatalf("NewDaemon: %v", err)
	}
	if err := d.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer d.Stop()

	if _, err := d.StartInstance("mock", "same", false); err != nil {
		t.Fatalf("first StartInstance: %v", err)
	}
	if _, err := d.StartInstance("mock", "same", false); err == nil {
		t.Fatalf("expected error for duplicate label")
	}
}
