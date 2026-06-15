package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Client struct {
	httpClient *http.Client
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Transport: &http.Transport{
				DialContext: dialSocket,
			},
			Timeout: 5 * time.Second,
		},
	}
}

func dialSocket(ctx context.Context, network, addr string) (net.Conn, error) {
	var d net.Dialer
	return d.DialContext(ctx, "unix", SocketPath())
}

func (c *Client) Start(harness, label string, debug bool) (*InstanceInfo, error) {
	if err := c.ensureDaemon(); err != nil {
		return nil, err
	}
	body, _ := json.Marshal(map[string]interface{}{
		"harness": harness,
		"label":   label,
		"debug":   debug,
	})
	resp, err := c.httpClient.Post("http://daemon/start", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("daemon request failed: %w", err)
	}
	defer resp.Body.Close()
	return decodeInstance(resp)
}

func (c *Client) Stop(label string) ([]string, error) {
	if err := c.ensureDaemon(); err != nil {
		return nil, err
	}
	body, _ := json.Marshal(map[string]interface{}{
		"label": label,
	})
	resp, err := c.httpClient.Post("http://daemon/stop", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("daemon request failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Stopped []string `json:"stopped"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode stop response: %w", err)
	}
	return result.Stopped, nil
}

func (c *Client) Status() ([]*InstanceInfo, error) {
	if err := c.ensureDaemon(); err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Get("http://daemon/status")
	if err != nil {
		return nil, fmt.Errorf("daemon request failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Instances []*InstanceInfo `json:"instances"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode status response: %w", err)
	}
	return result.Instances, nil
}

func (c *Client) Data(harness, label string) ([]byte, error) {
	if err := c.ensureDaemon(); err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Get(fmt.Sprintf("http://daemon/data/%s/%s", harness, label))
	if err != nil {
		return nil, fmt.Errorf("daemon request failed: %w", err)
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func (c *Client) ensureDaemon() error {
	if c.ping() == nil {
		return nil
	}
	return startDaemonFallback()
}

func (c *Client) ping() error {
	resp, err := c.httpClient.Get("http://daemon/status")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func startDaemonFallback() error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("find executable: %w", err)
	}

	cmd := exec.Command(exe, "proxy", "daemon")
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start daemon: %w", err)
	}

	// Wait briefly for the daemon socket to appear.
	for i := 0; i < 50; i++ {
		time.Sleep(50 * time.Millisecond)
		if _, err := os.Stat(SocketPath()); err == nil {
			return nil
		}
	}
	return fmt.Errorf("daemon did not start")
}

func decodeInstance(resp *http.Response) (*InstanceInfo, error) {
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("daemon error: %s", strings.TrimSpace(string(body)))
	}
	var info InstanceInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("decode instance: %w", err)
	}
	return &info, nil
}

func BinaryName() string {
	if exe, err := os.Executable(); err == nil {
		return filepath.Base(exe)
	}
	return "agent-statusline"
}
