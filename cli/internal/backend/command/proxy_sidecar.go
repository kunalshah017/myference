package command

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type ProxyConfiguration struct {
	Kind                string `json:"kind"`
	Secret              string `json:"secret"`
	Model               string `json:"model"`
	Token               string `json:"token"`
	MaximumOutputTokens uint64 `json:"maximum_output_tokens"`
}

func ServeProxy(ctx context.Context, configurationPath string) error {
	handle, err := os.Open(configurationPath)
	if err != nil {
		return err
	}
	defer handle.Close()
	var configuration ProxyConfiguration
	decoder := json.NewDecoder(io.LimitReader(handle, 64<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&configuration); err != nil {
		return err
	}
	targets := map[string]string{"codex": "https://api.openai.com", "claude": "https://api.anthropic.com", "kimi": "https://api.moonshot.ai"}
	target := targets[configuration.Kind]
	handler, err := credentialProxyHandler(configuration.Kind, configuration.Secret, configuration.Model, configuration.MaximumOutputTokens, target, configuration.Token)
	if err != nil {
		return err
	}
	mux := http.NewServeMux()
	mux.Handle("/", handler)
	mux.HandleFunc("GET /healthz", func(response http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(response, "ok") })
	server := &http.Server{Addr: ":8080", Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()
	err = server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func ProxyHealth(ctx context.Context) error {
	request, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://127.0.0.1:8080/healthz", nil)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return errors.New("credential proxy is not healthy")
	}
	return nil
}

type dockerProxy struct {
	docker, name, network, egress, directory, token string
}

func (r *Runner) startProxySidecar(ctx context.Context, jobName, model string, maximumOutputTokens uint64) (*dockerProxy, error) {
	token, err := randomToken()
	if err != nil {
		return nil, err
	}
	directory, err := os.MkdirTemp("", "myference-proxy-*")
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(directory, 0o700); err != nil {
		os.RemoveAll(directory)
		return nil, err
	}
	configuration, err := json.Marshal(ProxyConfiguration{Kind: r.kind, Secret: r.secret, Model: model, Token: token, MaximumOutputTokens: maximumOutputTokens})
	if err != nil {
		os.RemoveAll(directory)
		return nil, err
	}
	configurationPath := filepath.Join(directory, "proxy.json")
	if err := os.WriteFile(configurationPath, configuration, 0o600); err != nil {
		os.RemoveAll(directory)
		return nil, err
	}
	proxy := &dockerProxy{docker: r.docker, name: jobName + "-proxy", network: jobName + "-internal", egress: jobName + "-egress", directory: directory, token: token}
	fail := func(cause error) (*dockerProxy, error) {
		proxy.Close()
		return nil, cause
	}
	if output, commandErr := exec.CommandContext(ctx, r.docker, "network", "create", "--internal", proxy.network).CombinedOutput(); commandErr != nil {
		return fail(fmt.Errorf("create isolated agent network: %w: %s", commandErr, output))
	}
	if output, commandErr := exec.CommandContext(ctx, r.docker, "network", "create", proxy.egress).CombinedOutput(); commandErr != nil {
		return fail(fmt.Errorf("create proxy egress network: %w: %s", commandErr, output))
	}
	arguments := []string{"run", "-d", "--rm", "--name", proxy.name, "--network", proxy.network, "--read-only", "--cap-drop=ALL", "--security-opt=no-new-privileges", "--pids-limit=64", "--memory=128m", "--cpus=0.25", "--mount", "type=bind,source=" + r.proxyBinary + ",target=/myference-proxy,readonly", "--mount", "type=bind,source=" + configurationPath + ",target=/proxy.json,readonly", "--entrypoint", "/myference-proxy", r.image, "--config", "/proxy.json"}
	if output, commandErr := exec.CommandContext(ctx, r.docker, arguments...).CombinedOutput(); commandErr != nil {
		return fail(fmt.Errorf("start isolated credential proxy: %w: %s", commandErr, output))
	}
	if output, commandErr := exec.CommandContext(ctx, r.docker, "network", "connect", proxy.egress, proxy.name).CombinedOutput(); commandErr != nil {
		return fail(fmt.Errorf("connect credential proxy egress: %w: %s", commandErr, output))
	}
	for attempt := 0; attempt < 50; attempt++ {
		healthCtx, cancel := context.WithTimeout(ctx, time.Second)
		healthErr := exec.CommandContext(healthCtx, r.docker, "exec", proxy.name, "/myference-proxy", "--health").Run()
		cancel()
		if healthErr == nil {
			return proxy, nil
		}
		time.Sleep(20 * time.Millisecond)
	}
	return fail(errors.New("isolated credential proxy did not become healthy"))
}

func (p *dockerProxy) Close() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = exec.CommandContext(ctx, p.docker, "rm", "-f", p.name).Run()
	_ = exec.CommandContext(ctx, p.docker, "network", "rm", p.network).Run()
	_ = exec.CommandContext(ctx, p.docker, "network", "rm", p.egress).Run()
	_ = os.RemoveAll(p.directory)
}
