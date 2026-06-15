package proxy

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/nathabonfim59/agent-statusline/harness"
)

type InstanceInfo struct {
	Harness   string    `json:"harness"`
	Label     string    `json:"label"`
	ProxyPort int       `json:"proxy_port"`
	DataPort  int       `json:"data_port"`
	StartedAt time.Time `json:"started_at"`
}

type Daemon struct {
	mu        sync.RWMutex
	instances map[string]*ProxyServer
	info      map[string]*InstanceInfo
	caCert    *x509.Certificate
	caKey     *rsa.PrivateKey
	listener  net.Listener
	done      chan struct{}
}

func NewDaemon() (*Daemon, error) {
	certPEM, keyPEM, err := LoadOrGenerateCA()
	if err != nil {
		return nil, fmt.Errorf("load CA: %w", err)
	}

	certBlock, _ := pem.Decode(certPEM)
	caCert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse CA cert: %w", err)
	}

	keyBlock, _ := pem.Decode(keyPEM)
	caKey, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse CA key: %w", err)
	}

	return &Daemon{
		instances: make(map[string]*ProxyServer),
		info:      make(map[string]*InstanceInfo),
		caCert:    caCert,
		caKey:     caKey,
		done:      make(chan struct{}),
	}, nil
}

var socketPathOverride string

func SocketPath() string {
	if socketPathOverride != "" {
		return socketPathOverride
	}
	return socketPathDefault()
}

func socketPathDefault() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(os.TempDir(), "agent-statusline-daemon.sock")
	}
	if xdg := os.Getenv("XDG_RUNTIME_DIR"); xdg != "" {
		return filepath.Join(xdg, "agent-statusline", "daemon.sock")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "agent-statusline", "daemon.sock")
}

func (d *Daemon) Start() error {
	socketPath := SocketPath()
	if err := os.MkdirAll(filepath.Dir(socketPath), 0o700); err != nil {
		return fmt.Errorf("create socket directory: %w", err)
	}
	os.Remove(socketPath)

	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", socketPath, err)
	}
	d.listener = ln

	srv := &http.Server{Handler: d.mux()}
	go func() {
		<-d.done
		srv.Close()
	}()
	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "daemon serve error: %v\n", err)
		}
	}()

	return nil
}

func (d *Daemon) Stop() {
	close(d.done)
	if d.listener != nil {
		d.listener.Close()
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, s := range d.instances {
		s.Stop()
	}
	d.instances = nil
	d.info = nil
	if runtime.GOOS != "windows" {
		os.Remove(SocketPath())
	}
}

func (d *Daemon) StartInstance(harnessName, label string, debug bool) (*InstanceInfo, error) {
	if label == "" {
		label = randomName()
	}
	key := harnessName + "/" + label

	h := harness.NewHarness(harnessName)
	if h == nil {
		return nil, fmt.Errorf("unknown harness: %s", harnessName)
	}

	cfg := h.ProxyConfig()
	if cfg == nil {
		return nil, fmt.Errorf("harness %s does not support proxy", harnessName)
	}
	cfg.Debug = debug

	srv, err := NewServer(*cfg, d.caCert, d.caKey)
	if err != nil {
		return nil, fmt.Errorf("start proxy: %w", err)
	}

	d.mu.Lock()
	defer d.mu.Unlock()
	if _, ok := d.instances[key]; ok {
		srv.Stop()
		return nil, fmt.Errorf("instance %s already exists", key)
	}

	info := &InstanceInfo{
		Harness:   harnessName,
		Label:     label,
		ProxyPort: srv.Port(),
		DataPort:  srv.DataPort(),
		StartedAt: time.Now(),
	}
	d.instances[key] = srv
	d.info[key] = info
	return info, nil
}

func (d *Daemon) StopInstance(label string) []string {
	d.mu.Lock()
	defer d.mu.Unlock()

	var stopped []string
	if label == "" {
		for key, s := range d.instances {
			s.Stop()
			delete(d.instances, key)
			delete(d.info, key)
			stopped = append(stopped, key)
		}
		return stopped
	}

	for key, s := range d.instances {
		_, lbl := keyParts(key)
		if lbl == label {
			s.Stop()
			delete(d.instances, key)
			delete(d.info, key)
			stopped = append(stopped, key)
		}
	}
	return stopped
}

func (d *Daemon) InstanceInfo(label string) *InstanceInfo {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.info[label]
}

func (d *Daemon) InstanceData(harnessName, label string) interface{} {
	key := harnessName + "/" + label
	d.mu.RLock()
	srv, ok := d.instances[key]
	d.mu.RUnlock()
	if !ok || srv.config.Collector == nil {
		return nil
	}
	return srv.config.Collector.GetData()
}

func (d *Daemon) Status() []*InstanceInfo {
	d.mu.RLock()
	defer d.mu.RUnlock()
	var infos []*InstanceInfo
	for _, info := range d.info {
		infos = append(infos, info)
	}
	return infos
}

func (d *Daemon) FindRecent(harnessName string) *InstanceInfo {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var recent *InstanceInfo
	for key, info := range d.info {
		hs, _ := keyParts(key)
		if hs != harnessName {
			continue
		}
		if recent == nil || info.StartedAt.After(recent.StartedAt) {
			recent = info
		}
	}
	return recent
}

func keyParts(key string) (string, string) {
	if i := len(key) - 1; i >= 0 {
		for j := len(key) - 1; j >= 0; j-- {
			if key[j] == '/' {
				return key[:j], key[j+1:]
			}
		}
	}
	return "", key
}

var adjectives = []string{"swift", "calm", "bright", "keen", "bold", "warm", "cool", "sharp", "quiet", "lively", "fresh", "eager", "gentle", "proud", "wild"}
var nouns = []string{"proxy", "relay", "bridge", "tunnel", "gate", "link", "node", "port", "route", "pulse", "spark", "flux", "beam", "wave", "stream"}

func randomName() string {
	return adjectives[time.Now().UnixNano()%int64(len(adjectives))] + "-" +
		nouns[time.Now().UnixNano()/int64(len(adjectives))%int64(len(nouns))]
}
