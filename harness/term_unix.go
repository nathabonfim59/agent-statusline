//go:build !windows

package harness

import (
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

type winsize struct {
	Rows uint16
	Cols uint16
	X    uint16
	Y    uint16
}

func TerminalWidth() int {
	if col := os.Getenv("COLUMNS"); col != "" {
		if w, err := strconv.Atoi(col); err == nil && w > 0 {
			return w
		}
	}
	if os.Getenv("TMUX") != "" {
		out, err := exec.Command("tmux", "display-message", "-p", "#{pane_width}").Output()
		if err == nil {
			if w, err := strconv.Atoi(strings.TrimSpace(string(out))); err == nil && w > 0 {
				return w
			}
		}
	}
	if tty, err := os.Open("/dev/tty"); err == nil {
		defer tty.Close()
		ws := winsize{}
		_, _, _ = syscall.Syscall(syscall.SYS_IOCTL, tty.Fd(), uintptr(syscall.TIOCGWINSZ), uintptr(unsafe.Pointer(&ws)))
		if ws.Cols > 0 {
			return int(ws.Cols)
		}
	}
	return 80
}
