//go:build windows

package client

import (
	"os/exec"
	"syscall"
)

// detachProcess detaches the core process from tbox. Windows has no Setpgid /
// process-group concept, so it uses CREATE_NEW_PROCESS_GROUP instead so the core
// does not receive the Ctrl+C/Ctrl+Break sent to tbox and stays resident after
// tbox exits.
func detachProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP}
}
