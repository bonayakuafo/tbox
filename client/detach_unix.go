//go:build !windows

package client

import (
	"os/exec"
	"syscall"
)

// detachProcess detaches the core process from tbox's process group (Setpgid),
// so that when tbox exits (including the SIGINT the terminal sends to the
// foreground process group on Ctrl+C) the core is unaffected and stays resident.
func detachProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}
