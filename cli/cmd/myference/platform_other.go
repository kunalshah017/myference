//go:build !windows

package main

import (
	"errors"
	"io"
)

func runPlatformCommand(string, []string, io.Writer) error {
	return errors.New("Windows lifecycle commands require the Windows build")
}
