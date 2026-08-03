package main

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
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
	"runtime"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/kunalshah017/myference/cli/internal/account"
	"github.com/kunalshah017/myference/cli/internal/backend"
	commandbackend "github.com/kunalshah017/myference/cli/internal/backend/command"
	"github.com/kunalshah017/myference/cli/internal/backend/ollama"
	openaiBackend "github.com/kunalshah017/myference/cli/internal/backend/openai"
	"github.com/kunalshah017/myference/cli/internal/config"
	"github.com/kunalshah017/myference/cli/internal/credential"
	"github.com/kunalshah017/myference/cli/internal/provider"
	v1 "github.com/kunalshah017/myference/protocol/v1"
)

const (
	machineCredentialService = "myference.machine"
	signerCredentialService  = "myference.signer"
	backendCredentialService = "myference.backend"
	defaultServerURL         = "https://api.myference.xyz"
	defaultWebURL            = "https://myference.xyz"
)

var version = "dev"
var commit = "unknown"

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "myference:", err)
		os.Exit(1)
	}
}

func run(args []string, output io.Writer) error {
	if len(args) == 0 {
		return errors.New("usage: myference login | host | backend <add|list|start|stop|version> | capacity | status | serve | service <install|start|stop|status|uninstall> | windows <doctor|status|models|test|dashboard|focus|headless|restore>")
	}
	switch args[0] {
	case "login":
		return runLogin(context.Background(), args[1:], output, defaultLoginDependencies())
	case "backend":
		return runBackend(args[1:], output)
	case "host":
		return runHost(context.Background(), args[1:], output)
	case "capacity", "publish":
		path, err := parseConfigFlag(args[0], args[1:])
		if err != nil {
			return err
		}
		return printCapacity(context.Background(), path, output)
	case "status":
		flags := flag.NewFlagSet("status", flag.ContinueOnError)
		flags.SetOutput(io.Discard)
		path := flags.String("config", defaultConfigPath(), "configuration path")
		asJSON := flags.Bool("json", false, "print machine-readable status")
		if err := flags.Parse(args[1:]); err != nil {
			return err
		}
		cfg, err := config.Load(*path)
		if err != nil {
			return err
		}
		if *asJSON {
			offers, _, err := discoverBackends(context.Background(), cfg, credential.Load)
			if err != nil {
				return fmt.Errorf("attest live capacity: %w", err)
			}
			capacity := v1.Capacity{Available: uint32(len(offers)), Offers: offers}
			return writeStatusJSON(cfg, capacity, output, credential.Load, time.Now)
		}
		_, err = fmt.Fprintf(output, "machine %s account %s backends %d\n", cfg.MachineID, cfg.AccountID, len(cfg.Backends))
		return err
	case "serve":
		path, err := parseConfigFlag("serve", args[1:])
		if err != nil {
			return err
		}
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()
		err = runServe(ctx, path, output)
		if errors.Is(err, context.Canceled) {
			return nil
		}
		return err
	case "service":
		return runPlatformCommand("service", args[1:], output)
	case "windows":
		return runPlatformCommand("windows", args[1:], output)
	case "stop", "legacy-start", "legacy-stop", "legacy-status":
		return runPlatformCommand(args[0], args[1:], output)
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runHost(ctx context.Context, args []string, output io.Writer) error {
	flags := flag.NewFlagSet("host", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	path := flags.String("config", defaultConfigPath(), "configuration path")
	serverURL := flags.String("server", defaultServerURL, "Myference server URL used when login is required")
	endpoint := flags.String("ollama-url", "http://127.0.0.1:11434", "local Ollama URL")
	modelName := flags.String("model", "", "installed model to serve; defaults to the first model")
	webURL := flags.String("web", defaultWebURL, "Myference web URL")
	setupOnly := flags.Bool("setup-only", false, "configure the backend without starting the foreground server")
	noBrowser := flags.Bool("no-browser", false, "do not open the provider workspace")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if _, err := config.Load(*path); errors.Is(err, os.ErrNotExist) {
		if err := runLogin(ctx, hostLoginArgs(*serverURL, *path, *noBrowser), output, defaultLoginDependencies()); err != nil {
			return fmt.Errorf("connect this machine: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("load host configuration: %w", err)
	}
	model, err := configureLocalHost(ctx, *path, *endpoint, *modelName, nil)
	if err != nil {
		return err
	}
	activationURL := strings.TrimRight(*webURL, "/") + "/host"
	if _, err := fmt.Fprintf(output, "Ready to host %s (%s). Activate pricing and collateral at %s\n", model.Name, model.Digest, activationURL); err != nil {
		return err
	}
	if !*noBrowser {
		if err := openBrowser(activationURL); err != nil {
			_, _ = fmt.Fprintln(output, "Browser did not open; use the URL above.")
		}
	}
	if *setupOnly {
		return nil
	}
	serveContext, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()
	return runServe(serveContext, *path, output)
}

func hostLoginArgs(serverURL, path string, noBrowser bool) []string {
	args := []string{"--server", serverURL, "--config", path}
	if noBrowser {
		args = append(args, "--no-browser")
	}
	return args
}

func configureLocalHost(ctx context.Context, path, endpoint, requestedModel string, client *http.Client) (backend.Model, error) {
	configured, err := ollama.New(endpoint, client)
	if err != nil {
		return backend.Model{}, err
	}
	queryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	models, err := configured.Models(queryCtx)
	if err != nil {
		return backend.Model{}, fmt.Errorf("connect to Ollama: %w", err)
	}
	if len(models) == 0 {
		return backend.Model{}, errors.New("Ollama has no installed models; install one with `ollama pull <model>`")
	}
	slices.SortFunc(models, func(a, b backend.Model) int { return strings.Compare(a.Name, b.Name) })
	selected := models[0]
	if requestedModel != "" {
		index := slices.IndexFunc(models, func(model backend.Model) bool { return model.Name == requestedModel })
		if index < 0 {
			return backend.Model{}, fmt.Errorf("model %q is not installed in Ollama", requestedModel)
		}
		selected = models[index]
	}
	cfg, err := config.Load(path)
	if err != nil {
		return backend.Model{}, fmt.Errorf("login before hosting: %w", err)
	}
	name := "local-" + safeBackendName(selected.Name)
	index := slices.IndexFunc(cfg.Backends, func(item config.Backend) bool { return item.Kind == "ollama" && item.Model == selected.Name })
	item := config.Backend{Name: name, Kind: "ollama", URL: endpoint, Model: selected.Name, PriceVersion: 1, Enabled: true}
	if index >= 0 {
		item.Name = cfg.Backends[index].Name
		item.PriceVersion = cfg.Backends[index].PriceVersion
		cfg.Backends[index] = item
	} else {
		if slices.ContainsFunc(cfg.Backends, func(existing config.Backend) bool { return existing.Name == name }) {
			return backend.Model{}, fmt.Errorf("backend name %q already exists", name)
		}
		cfg.Backends = append(cfg.Backends, item)
	}
	if err := config.Save(path, cfg); err != nil {
		return backend.Model{}, err
	}
	return selected, nil
}

func safeBackendName(model string) string {
	var result strings.Builder
	for _, character := range strings.ToLower(model) {
		if character >= 'a' && character <= 'z' || character >= '0' && character <= '9' {
			result.WriteRune(character)
		} else if result.Len() > 0 && !strings.HasSuffix(result.String(), "-") {
			result.WriteByte('-')
		}
	}
	return strings.Trim(result.String(), "-")
}

func writeStatusJSON(cfg config.Config, capacity v1.Capacity, output io.Writer, loadCredential func(string, string) (string, error), now func() time.Time) error {
	if err := capacity.Validate(); err != nil || capacity.Available != uint32(len(capacity.Offers)) {
		return errors.New("invalid live capacity")
	}
	capacityJSON, err := json.Marshal(capacity)
	if err != nil {
		return err
	}
	capacityHash := sha256.Sum256(capacityJSON)
	capacitySHA256 := fmt.Sprintf("%x", capacityHash)
	signerSecret, err := loadCredential(signerCredentialService, cfg.MachineID)
	if err != nil {
		return fmt.Errorf("load machine signer: %w", err)
	}
	signerKey, err := crypto.HexToECDSA(strings.TrimPrefix(signerSecret, "0x"))
	if err != nil {
		return errors.New("invalid machine signer")
	}
	signerAddress := crypto.PubkeyToAddress(signerKey.PublicKey).Hex()
	if common.IsHexAddress(cfg.SignerAddress) && !strings.EqualFold(cfg.SignerAddress, signerAddress) {
		return errors.New("configured machine signer does not match the credential vault")
	}
	generatedAt := now().UTC().Format(time.RFC3339)
	message := fmt.Sprintf("myference-status:v2:%s:%s:%s:%s:%s:%s:%s", cfg.MachineID, runtime.GOOS, runtime.GOARCH, version, commit, generatedAt, capacitySHA256)
	signature, err := crypto.Sign(accounts.TextHash([]byte(message)), signerKey)
	if err != nil {
		return err
	}
	signature[64] += 27
	return json.NewEncoder(output).Encode(map[string]any{"machine_id": cfg.MachineID, "account_id": cfg.AccountID, "signer_address": signerAddress, "goos": runtime.GOOS, "goarch": runtime.GOARCH, "version": version, "commit": commit, "backends": len(cfg.Backends), "capacity": capacity, "capacity_payload": base64.StdEncoding.EncodeToString(capacityJSON), "capacity_sha256": capacitySHA256, "generated_at": generatedAt, "attestation_message": message, "attestation_signature": "0x" + fmt.Sprintf("%x", signature)})
}

type loginDependencies struct {
	HTTPClient       *http.Client
	OpenBrowser      func(string) error
	SaveCredential   func(string, string, string) error
	DeleteCredential func(string, string) error
	Wait             func(context.Context, time.Duration) error
}

func defaultLoginDependencies() loginDependencies {
	return loginDependencies{
		HTTPClient:       &http.Client{Timeout: 15 * time.Second},
		OpenBrowser:      openBrowser,
		SaveCredential:   credential.Save,
		DeleteCredential: credential.Delete,
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
	serverURL := flags.String("server", defaultServerURL, "Myference server URL")
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
	signerKey, err := crypto.GenerateKey()
	if err != nil {
		return fmt.Errorf("generate machine signer: %w", err)
	}
	signerAddress := crypto.PubkeyToAddress(signerKey.PublicKey).Hex()
	signerSecret := fmt.Sprintf("%x", crypto.FromECDSA(signerKey))
	authorization, err := client.CreateDeviceAuthorization(ctx, *machineName, signerAddress)
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
	if exchanged.Machine.SignerAddress != "" && !strings.EqualFold(exchanged.Machine.SignerAddress, signerAddress) {
		return errors.New("server returned a different machine signer")
	}
	if err := dependencies.SaveCredential(signerCredentialService, exchanged.Machine.ID, signerSecret); err != nil {
		return fmt.Errorf("store signer credential: %w", err)
	}
	if err := dependencies.SaveCredential(machineCredentialService, exchanged.Machine.ID, exchanged.Token); err != nil {
		if dependencies.DeleteCredential != nil {
			_ = dependencies.DeleteCredential(signerCredentialService, exchanged.Machine.ID)
		}
		return fmt.Errorf("store machine credential: %w", err)
	}
	if authorization.ChainID == 0 || !common.IsHexAddress(authorization.ContractAddress) || common.HexToAddress(authorization.ContractAddress) == (common.Address{}) {
		return errors.New("server did not provide a valid settlement contract")
	}
	cfg := config.Config{ServerURL: *serverURL, AccountID: exchanged.Machine.AccountID, MachineID: exchanged.Machine.ID, ChainID: authorization.ChainID, ContractAddress: common.HexToAddress(authorization.ContractAddress).Hex(), SignerAddress: signerAddress}
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
	stopPath := path + ".stop"
	_ = os.Remove(stopPath)
	serveContext, stopServing := context.WithCancel(ctx)
	defer stopServing()
	go watchStopRequest(serveContext, stopPath, stopServing)
	cfg, err := config.Load(path)
	if err != nil {
		return err
	}
	token, err := credential.Load(machineCredentialService, cfg.MachineID)
	if err != nil {
		return fmt.Errorf("load machine credential: %w", err)
	}
	signerSecret, err := credential.Load(signerCredentialService, cfg.MachineID)
	if err != nil {
		return fmt.Errorf("load signer credential: %w", err)
	}
	signerKey, err := crypto.HexToECDSA(strings.TrimPrefix(signerSecret, "0x"))
	if err != nil || cfg.ChainID == 0 || !common.IsHexAddress(cfg.ContractAddress) {
		return errors.New("invalid machine signer or settlement domain")
	}
	var contract v1.Address
	copy(contract[:], common.HexToAddress(cfg.ContractAddress).Bytes())
	relay, err := relayURL(cfg.ServerURL)
	if err != nil {
		return err
	}
	if err := preparePlatformBackends(serveContext, cfg); err != nil {
		return fmt.Errorf("prepare provider backends: %w", err)
	}
	offers, backends, err := discoverBackends(serveContext, cfg, credential.Load)
	if err != nil {
		return err
	}
	if len(offers) == 0 {
		return errors.New("no enabled backends")
	}
	if _, err := fmt.Fprintf(output, "serving %d offer(s) from machine %s\n", len(offers), cfg.MachineID); err != nil {
		return err
	}
	daemon := provider.NewDaemon(provider.Config{RelayURL: relay, Token: token, MachineID: cfg.MachineID, Offers: offers, SignerKey: signerKey, ChainID: cfg.ChainID, Contract: contract}, backends)
	watchContext, stopWatching := context.WithCancel(serveContext)
	defer stopWatching()
	go watchBackendConfig(watchContext, path, daemon, output)
	return serveWithReconnect(serveContext, output, time.Second, 30*time.Second, daemon.Serve)
}

func serveWithReconnect(ctx context.Context, output io.Writer, initialDelay, maximumDelay time.Duration, serve func(context.Context) error) error {
	if initialDelay <= 0 {
		initialDelay = time.Second
	}
	if maximumDelay < initialDelay {
		maximumDelay = initialDelay
	}
	delay := initialDelay
	for {
		err := serve(ctx)
		if ctx.Err() != nil {
			return nil
		}
		if _, writeErr := fmt.Fprintf(output, "relay disconnected: %v; retrying in %s\n", err, delay); writeErr != nil {
			return writeErr
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return nil
		case <-timer.C:
		}
		if delay < maximumDelay {
			delay *= 2
			if delay > maximumDelay {
				delay = maximumDelay
			}
		}
	}
}

func watchStopRequest(ctx context.Context, stopPath string, stop context.CancelFunc) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := os.Stat(stopPath); err == nil {
				_ = os.Remove(stopPath)
				stop()
				return
			}
		}
	}
}

func discoverBackends(ctx context.Context, cfg config.Config, loadCredential func(string, string) (string, error)) ([]v1.OfferCapacity, map[string]backend.Backend, error) {
	offers := make([]v1.OfferCapacity, 0, len(cfg.Backends))
	backends := make(map[string]backend.Backend)
	for _, item := range cfg.Backends {
		if !item.Enabled {
			continue
		}
		client, err := configuredBackend(item, cfg.MachineID, loadCredential)
		if err != nil {
			return nil, nil, err
		}
		queryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		models, err := client.Models(queryCtx)
		cancel()
		if err != nil {
			return nil, nil, fmt.Errorf("discover backend %q: %w", item.Name, err)
		}
		modelIndex := slices.IndexFunc(models, func(model backend.Model) bool { return model.Name == item.Model })
		if modelIndex < 0 {
			return nil, nil, fmt.Errorf("configured model %q is not available", item.Model)
		}
		offers = append(offers, offerCapacity(item, models[modelIndex]))
		backends[item.Name] = client
	}
	return offers, backends, nil
}

func watchBackendConfig(ctx context.Context, path string, daemon *provider.Daemon, output io.Writer) {
	info, _ := os.Stat(path)
	last := time.Time{}
	if info != nil {
		last = info.ModTime()
	}
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			info, err := os.Stat(path)
			if err != nil || !info.ModTime().After(last) {
				continue
			}
			last = info.ModTime()
			cfg, err := config.Load(path)
			if err == nil {
				err = reloadBackends(ctx, cfg, daemon)
			}
			if err != nil {
				_, _ = fmt.Fprintf(output, "backend reload failed: %v\n", err)
			} else {
				_, _ = fmt.Fprintf(output, "backend capacity updated\n")
			}
		}
	}
}

func reloadBackends(ctx context.Context, cfg config.Config, daemon *provider.Daemon) error {
	if daemon == nil {
		return errors.New("provider daemon is required")
	}
	if err := preparePlatformBackends(ctx, cfg); err != nil {
		return fmt.Errorf("prepare changed backends: %w", err)
	}
	offers, backends, err := discoverBackends(ctx, cfg, credential.Load)
	if err != nil {
		return err
	}
	return daemon.UpdateBackends(offers, backends)
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
	return runBackendWithCredentials(args, output, credential.Save)
}

func runBackendWithCredentials(args []string, output io.Writer, saveCredential func(string, string, string) error) error {
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
	kind := flags.String("kind", "ollama", "backend kind: ollama, openai, codex, claude, or kimi")
	image := flags.String("image", "", "pinned Docker image containing the command agent")
	secret := flags.String("secret", "", "backend credential stored in the OS credential vault")
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
		supported := map[string]bool{"ollama": true, "openai": true, "codex": true, "claude": true, "kimi": true}
		if !supported[*kind] {
			return errors.New("unsupported backend kind")
		}
		switch *kind {
		case "ollama":
			if _, err := ollama.New(*endpoint, nil); err != nil {
				return err
			}
		case "openai":
			if _, err := openaiBackend.New(*endpoint, *secret, nil); err != nil {
				return err
			}
		default:
			if *secret == "" || *image == "" {
				return errors.New("--secret and a pinned --image are required for isolated command agents")
			}
			if _, err := commandbackend.New(*image, commandArguments(*kind, *model), *kind, *model, *secret, 2*time.Minute); err != nil {
				return err
			}
		}
		if slices.ContainsFunc(cfg.Backends, func(item config.Backend) bool { return item.Name == *name }) {
			return errors.New("backend name already exists")
		}
		if *priceVersion == 0 {
			return errors.New("--price-version must be positive")
		}
		item := config.Backend{Name: *name, Kind: *kind, Model: *model, PriceVersion: *priceVersion, Enabled: true, Image: *image}
		if *kind == "ollama" || *kind == "openai" {
			item.URL = *endpoint
		}
		cfg.Backends = append(cfg.Backends, item)
		if *kind != "ollama" {
			if saveCredential == nil {
				return errors.New("credential store unavailable")
			}
			if err := saveCredential(backendCredentialService, cfg.MachineID+"/"+*name, *secret); err != nil {
				return fmt.Errorf("store backend credential: %w", err)
			}
		}
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
	case "version":
		if *priceVersion == 0 {
			return errors.New("--price-version must be positive")
		}
		found := false
		for i := range cfg.Backends {
			if cfg.Backends[i].Name == *name {
				cfg.Backends[i].PriceVersion = *priceVersion
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

func commandArguments(kind, model string) []string {
	switch kind {
	case "codex":
		return []string{"exec", "--skip-git-repo-check", "--model", model, "-"}
	case "claude":
		return []string{"-p", "--model", model}
	case "kimi":
		return []string{"--print", "--model", model}
	default:
		return nil
	}
}

func configuredBackend(item config.Backend, machineID string, loadCredential func(string, string) (string, error)) (backend.Backend, error) {
	switch item.Kind {
	case "ollama":
		return ollama.New(item.URL, nil)
	case "openai":
		secret, err := loadCredential(backendCredentialService, machineID+"/"+item.Name)
		if err != nil {
			return nil, fmt.Errorf("load backend credential: %w", err)
		}
		return openaiBackend.New(item.URL, secret, nil)
	case "codex", "claude", "kimi":
		secret, err := loadCredential(backendCredentialService, machineID+"/"+item.Name)
		if err != nil {
			return nil, fmt.Errorf("load backend credential: %w", err)
		}
		return commandbackend.New(item.Image, commandArguments(item.Kind, item.Model), item.Kind, item.Model, secret, 2*time.Minute)
	default:
		return nil, fmt.Errorf("unsupported backend kind %q", item.Kind)
	}
}

func printCapacity(ctx context.Context, path string, output io.Writer) error {
	cfg, err := config.Load(path)
	if err != nil {
		return err
	}
	capacity := v1.Capacity{}
	for _, item := range cfg.Backends {
		if !item.Enabled {
			continue
		}
		client, err := configuredBackend(item, cfg.MachineID, credential.Load)
		if err != nil {
			return err
		}
		queryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		models, err := client.Models(queryCtx)
		cancel()
		if err != nil {
			return err
		}
		var discovered backend.Model
		found := false
		for _, model := range models {
			if model.Name == item.Model {
				found = true
				discovered = model
			}
		}
		if !found {
			return fmt.Errorf("configured model %q is not installed", item.Model)
		}
		capacity.Available++
		capacity.Offers = append(capacity.Offers, offerCapacity(item, discovered))
	}
	return json.NewEncoder(output).Encode(capacity)
}

func offerCapacity(item config.Backend, discovered backend.Model) v1.OfferCapacity {
	version := item.PriceVersion
	if version == 0 {
		version = 1
	}
	capabilities := []string{"stream", "text"}
	if item.Kind == "codex" || item.Kind == "claude" || item.Kind == "kimi" {
		capabilities = append(capabilities, "workspace")
	}
	evidenceKind, evidenceDigest, meteringMode := "upstream_model", discovered.Name, "tokens_and_compute"
	if item.Kind == "ollama" {
		evidenceKind, evidenceDigest = "ollama_digest", discovered.Digest
	}
	if item.Kind == "codex" || item.Kind == "claude" || item.Kind == "kimi" {
		evidenceKind, evidenceDigest, meteringMode = "runtime_image", strings.TrimPrefix(item.Image[strings.LastIndex(item.Image, "@")+1:], "@"), "compute_only"
	}
	return v1.OfferCapacity{OfferID: item.Name, Model: item.Model, PriceVersion: version, BackendKind: item.Kind, OfferHash: crypto.Keccak256Hash([]byte(item.Name)).Hex(), ModelHash: crypto.Keccak256Hash([]byte(item.Model)).Hex(), CapabilityHash: crypto.Keccak256Hash([]byte(strings.Join(capabilities, ","))).Hex(), Capabilities: capabilities, EvidenceKind: evidenceKind, EvidenceDigest: evidenceDigest, MeteringMode: meteringMode}
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
