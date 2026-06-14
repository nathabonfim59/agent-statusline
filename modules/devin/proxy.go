package devin

import "github.com/nathabonfim59/agent-statusline/harness"

func DevinProxyConfig() harness.ProxyConfig {
	return harness.ProxyConfig{
		Domains:   []string{"server.codeium.com", "api.devin.ai"},
		Collector: NewCollector(),
	}
}