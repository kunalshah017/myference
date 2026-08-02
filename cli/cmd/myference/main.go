package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/kunalshah017/myference/cli/internal/account"
	"github.com/kunalshah017/myference/cli/internal/backend"
	"github.com/kunalshah017/myference/cli/internal/backend/ollama"
	"github.com/kunalshah017/myference/cli/internal/config"
	"github.com/kunalshah017/myference/cli/internal/credential"
	"github.com/kunalshah017/myference/cli/internal/provider"
	v1 "github.com/kunalshah017/myference/protocol/v1"
)

const machineCredentialService = "myference.machine"

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "myference:", err)
		os.Exit(1)
	}
}

func run(args []string, output io.Writer) error {
	if len(args) == 0 {
		return errors.New("usage: myference login | backend <add|list|start|stop> | capacity | status | serve | stop")
	}
	switch args[0] {
	case "login":
		return runLogin(context.Background(), args[1:], output, defaultLoginDependencies())
	case "backend":
		return runBackend(args[1:], output)
	case "capacity", "publish":
		path, err := parseConfigFlag(args[0], args[1:])
		if err != nil {
			return err
		}
		return printCapacity(context.Background(), path, output)
	case "status":
		path, err := parseConfigFlag("status", args[1:])
		if err != nil {
			return err
		}
		cfg, err := config.Load(path)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(output, "machine %s account %s backends %d\n", cfg.MachineID, cfg.AccountID, len(cfg.Backends))
		return err
	case "serve":
		path, err := parseConfigFlag("serve", args[1:])
		if err != nil {
			return err
		}
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
		defer cancel()
		return runServe(ctx, path, output)
	case "stop", "legacy-start", "legacy-stop", "legacy-status":
		return runPlatformCommand(args[0], args[1:], output)
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

type loginDependencies struct {
	HTTPClient     *http.Client
	OpenBrowser    func(string) error
	SaveCredential func(string, string, string) error
	Wait           func(context.Context, time.Duration) error
}

func defaultLoginDependencies() loginDependencies {
	return loginDependencies{
		HTTPClient:     &http.Client{Timeout: 15 * time.Second},
		OpenBrowser:    openBrowser,
		SaveCredential: credential.Save,
		Wait: func(ctx context.Context, duration time.Duration) error {
			timer := time.NewTimer(duration)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
				return nil
			}
		},
	}
}

func runLogin(ctx context.Context, args []string, output io.Writer, dependencies loginDependencies) error {
	flags := flag.NewFlagSet("login", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	serverURL := flags.String("server", "https://api.myference.network", "Myference server URL")
	machineName := flags.String("name", "", "machine name")
	path := flags.String("config", defaultConfigPath(), "configuration path")
	noBrowser := flags.Bool("no-browser", false, "print the verification URL without opening it")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*machineName) == "" {
		hostname, err := os.Hostname()
		if err != nil || strings.TrimSpace(hostname) == "" {
			return errors.New("--name is required when the hostname is unavailable")
		}
		*machineName = hostname
	}
	client, err := account.NewClient(*serverURL, dependencies.HTTPClient)
	if err != nil {
		return err
	}
	authorization, err := client.CreateDeviceAuthorization(ctx, *machineName)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(output, "Approve machine %q at %s\nCode: %s\n", *machineName, authorization.VerificationURI, authorization.UserCode); err != nil {
		return err
	}
	if !*noBrowser {
		if err := dependencies.OpenBrowser(authorization.VerificationURI); err != nil {
			_, _ = fmt.Fprintf(output, "Browser did not open; use the URL above.\n")
		}
	}
	var exchanged account.DeviceToken
	for {
		exchanged, err = client.ExchangeDeviceAuthorization(ctx, authorization.DeviceCode)
		if !errors.Is(err, account.ErrPending) {
			break
		}
		if err := dependencies.Wait(ctx, 2*time.Second); err != nil {
			return err
		}
	}
	if err != nil {
		return err
	}
	if err := dependencies.SaveCredential(machineCredentialService, exchanged.Machine.ID, exchanged.Token); err != nil {
		return fmt.Errorf("store machine credential: %w", err)
	}
	cfg := config.Config{ServerURL: *serverURL, AccountID: exchanged.Machine.AccountID, MachineID: exchanged.Machine.ID}
	if existing, loadErr := config.Load(*path); loadErr == nil {
		cfg.Backends = existing.Backends
	}
	if err := config.Save(*path, cfg); err != nil {
		return err
	}
	_, err = fmt.Fprintf(output, "Machine %s connected to account %s.\n", exchanged.Machine.ID, exchanged.Machine.AccountID)
	return err
}

func runServe(ctx context.Context, path string, output io.Writer) error {
	cfg, err := config.Load(path)
	if err != nil {
		return err
	}
	token, err := credential.Load(machineCredentialService, cfg.MachineID)
	if err != nil {
		return fmt.Errorf("load machine credential: %w", err)
	}
	relay, err := relayURL(cfg.ServerURL)
	if err != nil {
		return err
	}
	offers := make([]v1.OfferCapacity, 0, len(cfg.Backends))
	backends := make(map[string]backend.Backend)
	for _, item := range cfg.Backends {
		if !item.Enabled {
			continue
		}
		if item.Kind != "ollama" {
			return fmt.Errorf("unsupported backend kind %q", item.Kind)
		}
		client, err := ollama.New(item.URL, nil)
		if err != nil {
			return err
		}
		queryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		models, err := client.Models(queryCtx)
		cancel()
		if err != nil {
			return fmt.Errorf("discover backend %q: %w", item.Name, err)
		}
		if !slices.ContainsFunc(models, func(model backend.Model) bool { return model.Name == item.Model }) {
			return fmt.Errorf("configured model %q is not installed", item.Model)
		}
		offers = append(offers, offerCapacity(item))
		backends[item.Name] = client
	}
	if len(offers) == 0 {
		return errors.New("no enabled backends")
	}
	if _, err := fmt.Fprintf(output, "serving %d offer(s) from machine %s\n", len(offers), cfg.MachineID); err != nil {
		return err
	}
	daemon := provider.NewDaemon(provider.Config{RelayURL: relay, Token: token, MachineID: cfg.MachineID, Offers: offers}, backends)
	return daemon.Serve(ctx)
}

func relayURL(serverURL string) (string, error) {
	parsed, err := url.Parse(serverURL)
	if err != nil || parsed.Host == "" {
		return "", errors.New("invalid server URL")
	}
	switch parsed.Scheme {
	case "https":
		parsed.Scheme = "wss"
	case "http":
		parsed.Scheme = "ws"
	default:
		return "", errors.New("server URL must use http or https")
	}
	parsed.Path = strings.TrimSuffix(parsed.Path, "/") + "/relay"
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), nil
}

func runBackend(args []string, output io.Writer) error {
	if len(args) == 0 {
		return errors.New("backend command is required")
	}
	command := args[0]
	flags := flag.NewFlagSet("backend "+command, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	path := flags.String("config", defaultConfigPath(), "configuration path")
	name := flags.String("name", "", "backend name")
	model := flags.String("model", "", "model name")
	endpoint := flags.String("url", "http://127.0.0.1:11434", "Ollama URL")
	priceVersion := flags.Uint64("price-version", 1, "published Monad offer version")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	cfg, err := config.Load(*path)
	if err != nil {
		return err
	}
	switch command {
	case "add":
		if *name == "" || *model == "" {
			return errors.New("--name and --model are required")
		}
		if _, err := ollama.New(*endpoint, nil); err != nil {
			return err
		}
		if slices.ContainsFunc(cfg.Backends, func(item config.Backend) bool { return item.Name == *name }) {
			return errors.New("backend name already exists")
		}
		if *priceVersion == 0 {
			return errors.New("--price-version must be positive")
		}
		cfg.Backends = append(cfg.Backends, config.Backend{Name: *name, Kind: "ollama", URL: *endpoint, Model: *model, PriceVersion: *priceVersion, Enabled: true})
	case "start", "stop":
		found := false
		for i := range cfg.Backends {
			if cfg.Backends[i].Name == *name {
				cfg.Backends[i].Enabled = command == "start"
				found = true
			}
		}
		if !found {
			return errors.New("backend not found")
		}
	case "list":
		for _, item := range cfg.Backends {
			state := "stopped"
			if item.Enabled {
				state = "enabled"
			}
			fmt.Fprintf(output, "%s\t%s\t%s\t%s\n", item.Name, item.Kind, item.Model, state)
		}
		return nil
	default:
		return fmt.Errorf("unknown backend command %q", command)
	}
	return config.Save(*path, cfg)
}

func printCapacity(ctx context.Context, path string, output io.Writer) error {
	cfg, err := config.Load(path)
	if err != nil {
		return err
	}
	capacity := v1.Capacity{}
	for _, item := range cfg.Backends {
		if !item.Enabled || item.Kind != "ollama" {
			continue
		}
		client, err := ollama.New(item.URL, nil)
		if err != nil {
			return err
		}
		queryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		models, err := client.Models(queryCtx)
		cancel()
		if err != nil {
			return err
		}
		found := false
		for _, model := range models {
			found = found || model.Name == item.Model
		}
		if !found {
			return fmt.Errorf("configured model %q is not installed", item.Model)
		}
		capacity.Available++
		capacity.Offers = append(capacity.Offers, offerCapacity(item))
	}
	return json.NewEncoder(output).Encode(capacity)
}

func offerCapacity(item config.Backend) v1.OfferCapacity {
	version := item.PriceVersion
	if version == 0 {
		version = 1
	}
	return v1.OfferCapacity{OfferID: item.Name, Model: item.Model, PriceVersion: version, BackendKind: item.Kind, OfferHash: crypto.Keccak256Hash([]byte(item.Name)).Hex(), ModelHash: crypto.Keccak256Hash([]byte(item.Model)).Hex(), CapabilityHash: crypto.Keccak256Hash([]byte("stream,text")).Hex()}
}

func parseConfigFlag(name string, args []string) (string, error) {
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	path := flags.String("config", defaultConfigPath(), "configuration path")
	if err := flags.Parse(args); err != nil {
		return "", err
	}
	return *path, nil
}

func defaultConfigPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "myference.json"
	}
	return filepath.Join(dir, "myference", "config.json")
}
