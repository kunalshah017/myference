//go:build darwin

package main

import (
	"errors"
	"flag"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	platform "github.com/kunalshah017/myference/cli/internal/platform/darwin"
)

func runPlatformCommand(command string, args []string, _ io.Writer) error {
	if command != "service" || len(args) == 0 {
		return errors.New("usage: myference service <install|start|stop|status|uninstall> [--config path]")
	}
	action := args[0]
	flags := flag.NewFlagSet("service "+action, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	configPath := flags.String("config", defaultConfigPath(), "configuration path")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	logPath := filepath.Join(home, "Library", "Logs", "Myference", "provider.log")
	if err := os.MkdirAll(filepath.Dir(logPath), 0o700); err != nil {
		return err
	}
	lifecycle, err := platform.New(executable, *configPath, filepath.Join(home, "Library", "LaunchAgents", "com.myference.provider.plist"), logPath, nil)
	if err != nil {
		return err
	}
	switch action {
	case "install":
		return lifecycle.Install()
	case "start":
		return lifecycle.Start()
	case "stop":
		return lifecycle.Stop()
	case "status":
		return lifecycle.Status()
	case "uninstall":
		return lifecycle.Uninstall()
	default:
		return errors.New("unknown service action")
	}
}

func openBrowser(uri string) error { return exec.Command("open", uri).Start() }
