//go:build !windows && !darwin

package main

import (
	"context"
	"errors"
	"io"
	"os/exec"
	"runtime"

	"github.com/kunalshah017/myference/cli/internal/config"
)

func preparePlatformBackends(context.Context, config.Config) error { return nil }

func startPlatformProviderSession(context.Context, config.Config, io.Writer) (func() error, error) {
	return func() error { return nil }, nil
}

func runPlatformCommand(command string, args []string, _ io.Writer) error {
	if command != "service" || len(args) == 0 {
		return errors.New("usage: myference service <install|start|stop|status|uninstall> [--config path]")
	}
	return errors.New("service lifecycle commands require Windows or macOS")
}

func openBrowser(uri string) error {
	command := "xdg-open"
	if runtime.GOOS == "darwin" {
		command = "open"
	}
	return exec.Command(command, uri).Start()
}
