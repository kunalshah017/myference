//go:build windows

package main

import (
	"context"
	"io"

	platform "github.com/kunalshah017/myference/cli/internal/platform/windows"
)

func runPlatformCommand(command string, _ []string, _ io.Writer) error {
	lifecycle, err := platform.Discover()
	if err != nil {
		return err
	}
	switch command {
	case "legacy-start":
		return lifecycle.Start(context.Background())
	case "legacy-status":
		return lifecycle.Status(context.Background())
	default:
		return lifecycle.Stop(context.Background())
	}
}
