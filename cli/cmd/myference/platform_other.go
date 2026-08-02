//go:build !windows

package main

import (
	"errors"
	"io"
	"os/exec"
	"runtime"
)

func runPlatformCommand(string, []string, io.Writer) error {
	return errors.New("Windows lifecycle commands require the Windows build")
}

func openBrowser(uri string) error {
	command := "xdg-open"
	if runtime.GOOS == "darwin" {
		command = "open"
	}
	return exec.Command(command, uri).Start()
}
