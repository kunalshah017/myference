//go:build windows

package windows

import (
	"os/exec"
	"syscall"
)

func startHiddenProcess(path string) error {
	command := exec.Command(path)
	command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP}
	return command.Start()
}
