package proxy

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

const unitName = "agent-statusline-proxy.service"

func InstallSystemdService(system bool) error {
	if runtime.GOOS == "windows" {
		return fmt.Errorf("systemd is not available on Windows")
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("find executable: %w", err)
	}

	if system {
		return installSystemService(exe)
	}
	return installUserService(exe)
}

func installUserService(exe string) error {
	unitPath, err := userUnitPath()
	if err != nil {
		return err
	}

	unit := renderUnit(exe, "", "")
	if err := os.MkdirAll(filepath.Dir(unitPath), 0o755); err != nil {
		return fmt.Errorf("create systemd user directory: %w", err)
	}
	if err := os.WriteFile(unitPath, []byte(unit), 0o644); err != nil {
		return fmt.Errorf("write unit file: %w", err)
	}

	if err := runSystemctl("--user", "daemon-reload"); err != nil {
		return err
	}
	if err := runSystemctl("--user", "enable", unitName); err != nil {
		return err
	}
	if err := runSystemctl("--user", "start", unitName); err != nil {
		return err
	}

	fmt.Printf("Installed user systemd service: %s\n", unitPath)
	fmt.Printf("Manage with: systemctl --user %s\n", unitName)
	return nil
}

func installSystemService(exe string) error {
	if os.Getuid() != 0 {
		return fmt.Errorf("--system requires root")
	}

	const user = "agent-statusline"
	if !userExists(user) {
		fmt.Fprintf(os.Stderr, "System user %s does not exist. Create it with:\n\n", user)
		fmt.Fprintf(os.Stderr, "  sudo useradd --system --home-dir /var/lib/agent-statusline --shell /usr/sbin/nologin %s\n\n", user)
		return fmt.Errorf("missing system user %s", user)
	}

	unitPath := filepath.Join("/etc/systemd/system", unitName)
	unit := renderUnit(exe, user, user)
	if err := os.WriteFile(unitPath, []byte(unit), 0o644); err != nil {
		return fmt.Errorf("write unit file: %w", err)
	}

	if err := runSystemctl("daemon-reload"); err != nil {
		return err
	}
	if err := runSystemctl("enable", unitName); err != nil {
		return err
	}
	if err := runSystemctl("start", unitName); err != nil {
		return err
	}

	fmt.Printf("Installed system systemd service: %s\n", unitPath)
	fmt.Printf("Manage with: sudo systemctl %s\n", unitName)
	return nil
}

func renderUnit(exe, runAs, runGroup string) string {
	unit := fmt.Sprintf(`[Unit]
Description=agent-statusline proxy daemon
After=network.target

[Service]
Type=simple
ExecStart=%s proxy daemon
Restart=on-failure
`, exe)
	if runAs != "" {
		unit += fmt.Sprintf("User=%s\nGroup=%s\n", runAs, runGroup)
	}
	unit += "\n[Install]\nWantedBy=default.target\n"
	return unit
}

func userUnitPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("find home directory: %w", err)
	}
	return filepath.Join(home, ".config", "systemd", "user", unitName), nil
}

func runSystemctl(args ...string) error {
	cmd := exec.Command("systemctl", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("systemctl %s: %w\n%s", args, err, string(out))
	}
	return nil
}

func userExists(name string) bool {
	cmd := exec.Command("id", "-u", name)
	err := cmd.Run()
	return err == nil
}
