//go:build !windows

package windows

import "os/exec"

func startHiddenProcess(path string) error {
	return exec.Command(path).Start()
}
