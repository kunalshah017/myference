//go:build windows

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"slices"
	"time"

	"github.com/kunalshah017/myference/cli/internal/config"
	platform "github.com/kunalshah017/myference/cli/internal/platform/windows"
)

func runPlatformCommand(command string, args []string, _ io.Writer) error {
	if command != "service" || len(args) == 0 {
		return errors.New("usage: myference service <install|start|stop|status|uninstall> [--config path]")
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

var prepareWindowsDocker = func(ctx context.Context, images []string, timeout time.Duration) error {
	return platform.DiscoverDockerRuntime().Prepare(ctx, images, timeout)
}

func startPlatformProviderSession(ctx context.Context, _ config.Config, _ io.Writer) (func() error, error) {
	store, err := platform.DefaultJournalStore()
	if err != nil {
		return nil, err
	}
	runner := platform.NewNativeRunner()
	allowBattery, _ := ctx.Value(platformAllowBatteryKey{}).(bool)
	return platform.StartProviderSession(ctx, platform.DefaultConfig(), platform.TuningOptions{AllowBattery: allowBattery}, store, runner)
}

func preparePlatformBackends(ctx context.Context, cfg config.Config) error {
	return prepareWindowsBackends(ctx, cfg, platform.DefaultConfig(), nil)
}

func prepareWindowsBackends(ctx context.Context, cfg config.Config, hostConfig platform.Config, httpClient *http.Client) error {
	images := commandAgentImages(cfg)
	if len(images) > 0 {
		if err := prepareWindowsDocker(ctx, images, 2*time.Minute); err != nil {
			return fmt.Errorf("prepare Docker command agents: %w", err)
		}
	}
	for _, backend := range cfg.Backends {
		if !backend.Enabled || backend.Kind != "ollama" {
			continue
		}
		client, err := platform.NewOllamaHostClient(backend.URL, httpClient)
		if err != nil {
			return fmt.Errorf("configure Ollama backend %q: %w", backend.Name, err)
		}
		models, err := client.InstalledModels(ctx)
		if err != nil {
			return fmt.Errorf("discover Ollama backend %q before preload: %w", backend.Name, err)
		}
		model, err := platform.SelectInstalledModel(models, backend.Model)
		if err != nil {
			return fmt.Errorf("select Ollama backend %q model: %w", backend.Name, err)
		}
		if err := client.Preload(ctx, model, hostConfig); err != nil {
			return fmt.Errorf("preload Ollama backend %q: %w", backend.Name, err)
		}
	}
	return nil
}

func commandAgentImages(cfg config.Config) []string {
	images := make([]string, 0, len(cfg.Backends))
	seen := map[string]bool{}
	for _, backend := range cfg.Backends {
		if backend.Enabled && slices.Contains([]string{"codex", "claude", "kimi"}, backend.Kind) && !seen[backend.Image] {
			images = append(images, backend.Image)
			seen[backend.Image] = true
		}
	}
	return images
}

func openBrowser(uri string) error {
	return exec.Command("rundll32", "url.dll,FileProtocolHandler", uri).Start()
}
