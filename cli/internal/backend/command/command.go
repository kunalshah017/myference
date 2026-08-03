package command

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kunalshah017/myference/cli/internal/backend"
)

type Runner struct {
	docker, proxyBinary, image, kind, model, secret string
	arguments                                       []string
	timeout                                         time.Duration
}

var pinnedImagePattern = regexp.MustCompile(`^[^[:space:]]+@sha256:[0-9a-f]{64}$`)

func New(image string, arguments []string, kind, model, secret string, timeout time.Duration) (*Runner, error) {
	docker, err := exec.LookPath("docker")
	if err != nil {
		return nil, errors.New("Docker Desktop is required for isolated command agents")
	}
	proxyBinary := os.Getenv("MYFERENCE_AGENT_PROXY")
	if proxyBinary == "" {
		executable, executableErr := os.Executable()
		if executableErr != nil {
			return nil, executableErr
		}
		proxyName := "myference-agent-proxy"
		if runtime.GOOS == "windows" {
			proxyName += ".exe"
		}
		proxyBinary = filepath.Join(filepath.Dir(executable), proxyName)
	}
	if info, statErr := os.Stat(proxyBinary); statErr != nil || info.IsDir() {
		return nil, errors.New("myference-agent-proxy is required beside the CLI")
	}
	return newRunner(docker, proxyBinary, image, arguments, kind, model, secret, timeout)
}

func newRunner(docker, proxyBinary, image string, arguments []string, kind, model, secret string, timeout time.Duration) (*Runner, error) {
	if docker == "" || proxyBinary == "" || !pinnedImagePattern.MatchString(image) || strings.TrimSpace(model) == "" || strings.TrimSpace(secret) == "" || timeout <= 0 {
		return nil, errors.New("docker, digest-pinned image, model, secret, and positive timeout are required")
	}
	if kind != "codex" && kind != "claude" && kind != "kimi" {
		return nil, errors.New("unsupported isolated agent kind")
	}
	return &Runner{docker: docker, proxyBinary: proxyBinary, image: image, arguments: append([]string(nil), arguments...), kind: kind, model: model, secret: secret, timeout: timeout}, nil
}

func (r *Runner) Models(ctx context.Context) ([]backend.Model, error) {
	if err := exec.CommandContext(ctx, r.docker, "image", "inspect", r.image).Run(); err != nil {
		return nil, errors.New("digest-pinned agent image is not present in Docker")
	}
	return []backend.Model{{Name: r.model}}, nil
}

func (r *Runner) Generate(parent context.Context, input backend.Request, emit func(string) error) (backend.Usage, error) {
	if input.Model != r.model || strings.TrimSpace(input.Prompt) == "" || emit == nil {
		return backend.Usage{}, errors.New("configured model, prompt, and callback are required")
	}
	workspace, err := materializeWorkspace(input.Workspace)
	if err != nil {
		return backend.Usage{}, err
	}
	defer os.RemoveAll(workspace)
	ctx, cancel := context.WithTimeout(parent, r.timeout)
	defer cancel()
	name, err := randomName()
	if err != nil {
		return backend.Usage{}, err
	}
	proxy, err := r.startProxySidecar(ctx, name, input.Model, input.MaximumOutputTokens)
	if err != nil {
		return backend.Usage{}, err
	}
	defer proxy.Close()
	arguments := r.dockerArguments(name, workspace, proxy.network, "http://"+proxy.name+":8080", proxy.token)
	process := exec.CommandContext(ctx, r.docker, arguments...)
	process.Env = minimalDockerEnvironment()
	process.Stdin = strings.NewReader(input.Prompt)
	stdout, err := process.StdoutPipe()
	if err != nil {
		return backend.Usage{}, err
	}
	process.Stderr = io.Discard
	started := time.Now()
	if err := process.Start(); err != nil {
		return backend.Usage{}, err
	}
	defer exec.Command(r.docker, "rm", "-f", name).Run()
	maximumOutputBytes := input.MaximumOutputTokens * 4
	if maximumOutputBytes == 0 {
		maximumOutputBytes = 16 << 10
	}
	var outputBytes uint64
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 64<<10), 1<<20)
	for scanner.Scan() {
		chunk := scanner.Text() + "\n"
		if uint64(len(chunk)) > maximumOutputBytes-outputBytes {
			_ = exec.Command(r.docker, "rm", "-f", name).Run()
			_ = process.Wait()
			return backend.Usage{}, errors.New("command agent exceeded output limit")
		}
		outputBytes += uint64(len(chunk))
		if err := emit(chunk); err != nil {
			_ = exec.Command(r.docker, "rm", "-f", name).Run()
			_ = process.Wait()
			return backend.Usage{}, err
		}
	}
	if err := scanner.Err(); err != nil {
		_ = exec.Command(r.docker, "rm", "-f", name).Run()
		_ = process.Wait()
		return backend.Usage{}, err
	}
	if err := process.Wait(); err != nil {
		if ctx.Err() != nil {
			return backend.Usage{}, fmt.Errorf("isolated agent stopped: %w", ctx.Err())
		}
		return backend.Usage{}, fmt.Errorf("isolated agent failed: %w", err)
	}
	return backend.Usage{ComputeMilliseconds: uint64(time.Since(started).Milliseconds())}, nil
}

func materializeWorkspace(files []backend.WorkspaceFile) (string, error) {
	workspace, err := os.MkdirTemp("", "myference-job-*")
	if err != nil {
		return "", err
	}
	if err := os.Chmod(workspace, 0o700); err != nil {
		os.RemoveAll(workspace)
		return "", err
	}
	for _, file := range files {
		relative := filepath.Clean(filepath.FromSlash(file.Path))
		if relative == "." || filepath.IsAbs(relative) || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || strings.ContainsRune(relative, 0) {
			os.RemoveAll(workspace)
			return "", errors.New("invalid workspace path")
		}
		target := filepath.Join(workspace, relative)
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			os.RemoveAll(workspace)
			return "", err
		}
		handle, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			os.RemoveAll(workspace)
			return "", err
		}
		_, writeErr := handle.Write(file.Content)
		closeErr := handle.Close()
		if writeErr != nil || closeErr != nil {
			os.RemoveAll(workspace)
			if writeErr != nil {
				return "", writeErr
			}
			return "", closeErr
		}
	}
	return workspace, nil
}

func (r *Runner) dockerArguments(name, workspace, network, proxyURL, token string) []string {
	environment := []string{"HOME=/workspace", "TMPDIR=/tmp", "NO_COLOR=1", "TERM=dumb"}
	switch r.kind {
	case "codex":
		environment = append(environment, "OPENAI_API_KEY="+token, "OPENAI_BASE_URL="+proxyURL)
	case "claude":
		environment = append(environment, "ANTHROPIC_API_KEY="+token, "ANTHROPIC_BASE_URL="+proxyURL)
	case "kimi":
		environment = append(environment, "MOONSHOT_API_KEY="+token, "OPENAI_BASE_URL="+proxyURL)
	}
	arguments := []string{"run", "--rm", "--name", name, "--network", network, "--read-only", "--cap-drop=ALL", "--security-opt=no-new-privileges", "--pids-limit=256", "--memory=2g", "--cpus=2", "--mount", "type=bind,source=" + workspace + ",target=/workspace", "--tmpfs", "/tmp:rw,noexec,nosuid,size=64m", "--workdir", "/workspace"}
	for _, item := range environment {
		arguments = append(arguments, "--env", item)
	}
	arguments = append(arguments, r.image)
	return append(arguments, r.arguments...)
}

func minimalDockerEnvironment() []string {
	result := []string{"PATH=/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin"}
	if value := os.Getenv("DOCKER_HOST"); value != "" {
		result = append(result, "DOCKER_HOST="+value)
	}
	return result
}

func randomName() (string, error) {
	value := make([]byte, 8)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return "myference-job-" + hex.EncodeToString(value), nil
}

type credentialProxy struct {
	server *http.Server
	url    string
	token  string
	done   chan struct{}
}

func newCredentialProxy(kind, secret, model string, maximumOutputTokens uint64) (*credentialProxy, error) {
	targets := map[string]string{"codex": "https://api.openai.com", "claude": "https://api.anthropic.com", "kimi": "https://api.moonshot.ai"}
	return newCredentialProxyForTarget(kind, secret, model, maximumOutputTokens, targets[kind])
}

func newCredentialProxyForTarget(kind, secret, model string, maximumOutputTokens uint64, targetURL string) (*credentialProxy, error) {
	token, err := randomToken()
	if err != nil {
		return nil, err
	}
	handler, err := credentialProxyHandler(kind, secret, model, maximumOutputTokens, targetURL, token)
	if err != nil {
		return nil, err
	}
	listener, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp4", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second}
	proxy := &credentialProxy{server: server, url: "http://" + listener.Addr().String(), token: token, done: make(chan struct{})}
	go func() { _ = server.Serve(listener); close(proxy.done) }()
	return proxy, nil
}

func credentialProxyHandler(kind, secret, model string, maximumOutputTokens uint64, targetURL, token string) (http.Handler, error) {
	target, err := url.Parse(targetURL)
	if err != nil || target.Host == "" || strings.TrimSpace(model) == "" || strings.TrimSpace(secret) == "" || strings.TrimSpace(token) == "" || maximumOutputTokens == 0 {
		return nil, errors.New("invalid credential proxy configuration")
	}
	reverse := httputil.NewSingleHostReverseProxy(target)
	reverse.ModifyResponse = func(response *http.Response) error {
		response.Body = struct {
			io.Reader
			io.Closer
		}{Reader: io.LimitReader(response.Body, 16<<20), Closer: response.Body}
		return nil
	}
	originalDirector := reverse.Director
	var budgetMu sync.Mutex
	remainingTokens := maximumOutputTokens
	requests := uint32(0)
	reverse.Director = func(request *http.Request) {
		originalDirector(request)
		request.Host = target.Host
		request.Header.Del("Authorization")
		request.Header.Del("x-api-key")
		if kind == "claude" {
			request.Header.Set("x-api-key", secret)
		} else {
			request.Header.Set("Authorization", "Bearer "+secret)
		}
	}
	handler := http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer "+token && request.Header.Get("x-api-key") != token {
			http.Error(response, "unauthorized", http.StatusUnauthorized)
			return
		}
		budgetMu.Lock()
		requests++
		tooMany := requests > 64
		budgetMu.Unlock()
		if tooMany {
			http.Error(response, "agent request limit exceeded", http.StatusTooManyRequests)
			return
		}
		if request.Method == http.MethodGet && request.URL.Path == "/v1/models" {
			reverse.ServeHTTP(response, request)
			return
		}
		if request.Method != http.MethodPost || !allowedInferencePath(request.URL.Path) {
			http.Error(response, "unsupported agent API request", http.StatusForbidden)
			return
		}
		raw, readErr := io.ReadAll(http.MaxBytesReader(response, request.Body, 8<<20))
		if readErr != nil {
			http.Error(response, "invalid agent API request", http.StatusBadRequest)
			return
		}
		budgetMu.Lock()
		rewritten, budgetErr := authorizeInferenceBody(raw, request.URL.Path, model, remainingTokens)
		if budgetErr == nil {
			remainingTokens -= rewritten.tokens
		}
		budgetMu.Unlock()
		if budgetErr != nil {
			status := http.StatusForbidden
			if errors.Is(budgetErr, errTokenBudget) {
				status = http.StatusPaymentRequired
			}
			http.Error(response, budgetErr.Error(), status)
			return
		}
		request.Body = io.NopCloser(bytes.NewReader(rewritten.body))
		request.ContentLength = int64(len(rewritten.body))
		reverse.ServeHTTP(response, request)
	})
	return handler, nil
}

var errTokenBudget = errors.New("agent output token budget exceeded")

type authorizedBody struct {
	body   []byte
	tokens uint64
}

func allowedInferencePath(path string) bool {
	return path == "/v1/chat/completions" || path == "/v1/responses" || path == "/v1/messages"
}

func authorizeInferenceBody(raw []byte, path, model string, remaining uint64) (authorizedBody, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var payload map[string]any
	if err := decoder.Decode(&payload); err != nil {
		return authorizedBody{}, errors.New("invalid agent API JSON")
	}
	if requestModel, ok := payload["model"].(string); !ok || requestModel != model {
		return authorizedBody{}, errors.New("agent model is not authorized")
	}
	key := "max_tokens"
	if path == "/v1/responses" {
		key = "max_output_tokens"
	} else if path == "/v1/chat/completions" {
		if _, ok := payload["max_completion_tokens"]; ok {
			key = "max_completion_tokens"
		}
	}
	tokens := remaining
	if value, ok := payload[key]; ok {
		number, ok := value.(json.Number)
		if !ok {
			return authorizedBody{}, errors.New("invalid agent output token limit")
		}
		parsed, err := strconv.ParseUint(number.String(), 10, 64)
		if err != nil || parsed == 0 {
			return authorizedBody{}, errors.New("invalid agent output token limit")
		}
		tokens = parsed
	}
	if tokens == 0 || tokens > remaining {
		return authorizedBody{}, errTokenBudget
	}
	payload[key] = tokens
	rewritten, err := json.Marshal(payload)
	return authorizedBody{body: rewritten, tokens: tokens}, err
}

func randomToken() (string, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return "myference-job-" + hex.EncodeToString(value), nil
}

func (p *credentialProxy) containerURL() string {
	return strings.Replace(p.url, "127.0.0.1", "host.docker.internal", 1)
}

func (p *credentialProxy) Close() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_ = p.server.Shutdown(ctx)
	<-p.done
}
