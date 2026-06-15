package proxy

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

func (d *Daemon) mux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/start", d.handleStart)
	mux.HandleFunc("/stop", d.handleStop)
	mux.HandleFunc("/status", d.handleStatus)
	mux.HandleFunc("/data/", d.handleData)
	return mux
}

func (d *Daemon) handleStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Harness string `json:"harness"`
		Label   string `json:"label"`
		Debug   bool   `json:"debug"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("decode request: %v", err), http.StatusBadRequest)
		return
	}

	info, err := d.StartInstance(req.Harness, req.Label, req.Debug)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

func (d *Daemon) handleStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Label string `json:"label"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req.Label = ""
	}

	stopped := d.StopInstance(req.Label)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"stopped": stopped,
	})
}

func (d *Daemon) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"instances": d.Status(),
	})
}

func (d *Daemon) handleData(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	harnessName, label, ok := parseDataPath(r.URL.Path)
	if !ok {
		http.Error(w, "invalid path, expected /data/{harness}/{label}", http.StatusBadRequest)
		return
	}

	data := d.InstanceData(harnessName, label)
	if data == nil {
		http.Error(w, "instance not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func parseDataPath(path string) (harness, label string, ok bool) {
	const prefix = "/data/"
	if !strings.HasPrefix(path, prefix) {
		return "", "", false
	}
	rest := path[len(prefix):]
	if i := strings.IndexByte(rest, '/'); i >= 0 {
		return rest[:i], rest[i+1:], true
	}
	return "", "", false
}
