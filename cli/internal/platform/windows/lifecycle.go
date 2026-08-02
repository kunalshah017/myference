//go:build windows

package windows

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
)

type Lifecycle struct{ Script string }

func Discover() (Lifecycle, error) {
	if configured := os.Getenv("MYFERENCE_LEGACY_WINDOWS_SCRIPT"); configured != "" {
		return Lifecycle{Script: configured}, nil
	}
	executable, err := os.Executable()
	if err != nil {
		return Lifecycle{}, err
	}
	candidates := []string{
		filepath.Join(filepath.Dir(executable), "legacy", "myference.ps1"),
		filepath.Join(filepath.Dir(executable), "platform", "windows", "legacy", "myference.ps1"),
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return Lifecycle{Script: candidate}, nil
		}
	}
	return Lifecycle{}, errors.New("preserved Windows lifecycle script not found")
}

func (l Lifecycle) Start(ctx context.Context) error {
	return l.run(ctx, "start", "-NoDashboard")
}

func (l Lifecycle) Stop(ctx context.Context) error { return l.run(ctx, "stop") }

func (l Lifecycle) Status(ctx context.Context) error { return l.run(ctx, "status") }

func (l Lifecycle) run(ctx context.Context, args ...string) error {
	commandArgs := []string{"-NoProfile", "-ExecutionPolicy", "Bypass", "-File", l.Script}
	commandArgs = append(commandArgs, args...)
	command := exec.CommandContext(ctx, "powershell.exe", commandArgs...)
	command.Stdout, command.Stderr, command.Stdin = os.Stdout, os.Stderr, os.Stdin
	return command.Run()
}
