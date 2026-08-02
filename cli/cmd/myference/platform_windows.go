//go:build windows

package main

import (
	"context"
	"errors"
	"flag"
	"io"
	"os/exec"

	platform "github.com/kunalshah017/myference/cli/internal/platform/windows"
)

func runPlatformCommand(command string, args []string, _ io.Writer) error {
	if command == "service" {
		if len(args) == 0 {
			return errors.New("usage: myference service <install|start|stop|status|uninstall>")
		}
		flags := flag.NewFlagSet("service "+args[0], flag.ContinueOnError)
		flags.SetOutput(io.Discard)
		path := flags.String("config", defaultConfigPath(), "configuration path")
		if err := flags.Parse(args[1:]); err != nil {
			return err
		}
		service, err := platform.DiscoverService(*path)
		if err != nil {
			return err
		}
		ctx := context.Background()
		switch args[0] {
		case "install":
			return service.Install(ctx)
		case "start":
			return service.Start(ctx)
		case "stop":
			return service.Stop(ctx)
		case "status":
			return service.Status(ctx)
		case "uninstall":
			return service.Uninstall(ctx)
		default:
			return errors.New("unknown service action")
		}
	}
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

func openBrowser(uri string) error {
	return exec.Command("rundll32", "url.dll,FileProtocolHandler", uri).Start()
}
