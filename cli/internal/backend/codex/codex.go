package codex

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/kunalshah017/myference/cli/internal/backend"
)

var errToolAttempt = errors.New("Codex attempted to use a blocked tool")

type Runner struct {
	executable, model, sourceAuth, personalSkills, providerHome, hookExecutable string
	timeout                                                                     time.Duration
	commandContext                                                              func(context.Context, string, ...string) *exec.Cmd
}

func New(model string, timeout time.Duration) (*Runner, error) {
	executable, err := exec.LookPath("codex")
	if err != nil {
		return nil, errors.New("Codex CLI is not installed or not available on PATH")
	}
	userHome, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	stateRoot, err := os.UserCacheDir()
	if err != nil {
		return nil, err
	}
	hookExecutable, err := os.Executable()
	if err != nil {
		return nil, err
	}
	return newRunner(executable, model, filepath.Join(userHome, ".codex", "auth.json"), filepath.Join(userHome, ".agents", "skills"), filepath.Join(stateRoot, "Myference", "codex-provider"), hookExecutable, timeout)
}

func newRunner(executable, model, sourceAuth, personalSkills, providerHome, hookExecutable string, timeout time.Duration) (*Runner, error) {
	if strings.TrimSpace(executable) == "" || strings.TrimSpace(model) == "" || strings.TrimSpace(sourceAuth) == "" || strings.TrimSpace(personalSkills) == "" || strings.TrimSpace(providerHome) == "" || strings.TrimSpace(hookExecutable) == "" || timeout <= 0 {
		return nil, errors.New("Codex executable, model, authentication, provider home, hook executable, and positive timeout are required")
	}
	return &Runner{executable: executable, model: model, sourceAuth: sourceAuth, personalSkills: personalSkills, providerHome: providerHome, hookExecutable: hookExecutable, timeout: timeout, commandContext: exec.CommandContext}, nil
}

func (r *Runner) Models(context.Context) ([]backend.Model, error) {
	if err := r.prepareProviderHome(); err != nil {
		return nil, err
	}
	return []backend.Model{{Name: r.model}}, nil
}

func (r *Runner) Generate(parent context.Context, input backend.Request, emit func(string) error) (backend.Usage, error) {
	if input.Model != r.model || strings.TrimSpace(input.Prompt) == "" || emit == nil {
		return backend.Usage{}, errors.New("configured model, prompt, and callback are required")
	}
	if len(input.Workspace) != 0 {
		return backend.Usage{}, errors.New("native Codex is a model-only provider and does not accept workspace files")
	}
	maximumOutputTokens := input.MaximumOutputTokens
	if maximumOutputTokens == 0 {
		maximumOutputTokens = 4096
	}
	if err := r.prepareProviderHome(); err != nil {
		return backend.Usage{}, err
	}
	jobs := filepath.Join(r.providerHome, "jobs")
	if err := os.MkdirAll(jobs, 0o700); err != nil {
		return backend.Usage{}, err
	}
	job, err := os.MkdirTemp(jobs, "job-")
	if err != nil {
		return backend.Usage{}, err
	}
	defer os.RemoveAll(job)
	instructions := fmt.Sprintf("You are serving a model-only text request. Never call tools, run commands, inspect files, browse, edit, or delegate. Answer only from the user prompt in at most %d output tokens.\n", maximumOutputTokens)
	if err := os.WriteFile(filepath.Join(job, "AGENTS.md"), []byte(instructions), 0o600); err != nil {
		return backend.Usage{}, err
	}
	marker := filepath.Join(job, "tool-attempted")
	ctx, cancel := context.WithTimeout(parent, r.timeout)
	defer cancel()
	arguments := []string{"exec", "--ephemeral", "--json", "--ignore-rules", "--dangerously-bypass-hook-trust", "--sandbox", "read-only", "--skip-git-repo-check", "--model", r.model, "-"}
	command := r.commandContext(ctx, r.executable, arguments...)
	command.Dir = job
	command.Env = r.environment(marker)
	command.Stdin = strings.NewReader(input.Prompt)
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	started := time.Now()
	runErr := command.Run()
	elapsed := time.Since(started).Milliseconds()
	if elapsed < 1 {
		elapsed = 1
	}
	if _, markerErr := os.Stat(marker); markerErr == nil {
		return backend.Usage{}, errToolAttempt
	} else if !errors.Is(markerErr, os.ErrNotExist) {
		return backend.Usage{}, markerErr
	}
	if runErr != nil {
		if ctx.Err() != nil {
			return backend.Usage{}, fmt.Errorf("Codex execution stopped: %w", ctx.Err())
		}
		detail := strings.TrimSpace(stderr.String())
		if len(detail) > 2048 {
			detail = detail[len(detail)-2048:]
		}
		if detail != "" {
			return backend.Usage{}, fmt.Errorf("Codex execution failed: %w: %s", runErr, detail)
		}
		return backend.Usage{}, fmt.Errorf("Codex execution failed: %w", runErr)
	}
	result, err := parseOutput(&stdout, maximumOutputTokens)
	if err != nil {
		return backend.Usage{}, err
	}
	if err := emit(result.text); err != nil {
		return backend.Usage{}, err
	}
	result.usage.ComputeMilliseconds = uint64(elapsed)
	return result.usage, nil
}

func (r *Runner) prepareProviderHome() error {
	if err := os.MkdirAll(r.providerHome, 0o700); err != nil {
		return fmt.Errorf("create private Codex provider home: %w", err)
	}
	authentication := filepath.Join(r.providerHome, "auth.json")
	if _, err := os.Stat(authentication); errors.Is(err, os.ErrNotExist) {
		raw, readErr := os.ReadFile(r.sourceAuth)
		if readErr != nil {
			return errors.New("Codex CLI login not found; run `codex login` before adding a native Codex backend")
		}
		if len(raw) == 0 || len(raw) > 4<<20 {
			return errors.New("Codex CLI login file is invalid")
		}
		if writeErr := os.WriteFile(authentication, raw, 0o600); writeErr != nil {
			return fmt.Errorf("seed private Codex login: %w", writeErr)
		}
	} else if err != nil {
		return err
	}
	hookCommand := fmt.Sprintf("%q internal codex-deny-tool", r.hookExecutable)
	configurationLines := []string{
		`approval_policy = "never"`,
		`sandbox_mode = "read-only"`,
		`web_search = "disabled"`,
		`agents.enabled = false`,
		``,
		`[features]`,
		`hooks = true`,
		``,
		`[[hooks.PreToolUse]]`,
		`matcher = "*"`,
		``,
		`[[hooks.PreToolUse.hooks]]`,
		`type = "command"`,
		`command = ` + strconv.Quote(hookCommand),
		`command_windows = ` + strconv.Quote(hookCommand),
		``,
	}
	disabledSkills, err := discoverSkills(r.personalSkills)
	if err != nil {
		return fmt.Errorf("inspect personal Codex skills: %w", err)
	}
	for _, path := range disabledSkills {
		configurationLines = append(configurationLines, `[[skills.config]]`, `path = `+strconv.Quote(path), `enabled = false`, ``)
	}
	configuration := strings.Join(configurationLines, "\n")
	if err := os.WriteFile(filepath.Join(r.providerHome, "config.toml"), []byte(configuration), 0o600); err != nil {
		return fmt.Errorf("write private Codex provider config: %w", err)
	}
	return nil
}

func discoverSkills(root string) ([]string, error) {
	if _, err := os.Stat(root); errors.Is(err, os.ErrNotExist) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	paths := make([]string, 0)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.IsDir() && strings.EqualFold(entry.Name(), "SKILL.md") {
			paths = append(paths, path)
		}
		return nil
	})
	sort.Strings(paths)
	return paths, err
}

func (r *Runner) environment(marker string) []string {
	keys := []string{"PATH", "PATHEXT", "SYSTEMROOT", "WINDIR", "COMSPEC", "TEMP", "TMP", "LOCALAPPDATA", "APPDATA", "ProgramFiles", "ProgramFiles(x86)"}
	environment := make([]string, 0, len(keys)+4)
	for _, key := range keys {
		if value := os.Getenv(key); value != "" {
			environment = append(environment, key+"="+value)
		}
	}
	environment = append(environment, "CODEX_HOME="+r.providerHome, "HOME="+r.providerHome, "USERPROFILE="+r.providerHome, "MYFERENCE_CODEX_TOOL_MARKER="+marker)
	if runtime.GOOS != "windows" {
		environment = append(environment, "SHELL=/bin/sh")
	}
	return environment
}

type outputResult struct {
	text  string
	usage backend.Usage
}

func parseOutput(reader io.Reader, maximumOutputTokens uint64) (outputResult, error) {
	decoder := json.NewDecoder(reader)
	var text strings.Builder
	var usage backend.Usage
	completed := false
	usageSeen := false
	for {
		var event struct {
			Type string `json:"type"`
			Item *struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"item"`
			Usage *struct {
				Input  uint64 `json:"input_tokens"`
				Output uint64 `json:"output_tokens"`
			} `json:"usage"`
			Error *struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := decoder.Decode(&event); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return outputResult{}, fmt.Errorf("decode Codex output: %w", err)
		}
		switch event.Type {
		case "turn.failed", "error":
			message := "Codex turn failed"
			if event.Error != nil && event.Error.Message != "" {
				message += ": " + event.Error.Message
			}
			return outputResult{}, errors.New(message)
		case "turn.completed":
			completed = true
			if event.Usage != nil {
				usageSeen = true
				usage.InputTokens = event.Usage.Input
				usage.OutputTokens = event.Usage.Output
			}
		case "item.started", "item.completed", "item.updated":
			if event.Item == nil {
				continue
			}
			switch event.Item.Type {
			case "agent_message":
				if event.Type == "item.completed" {
					text.WriteString(event.Item.Text)
				}
			case "reasoning", "error":
			default:
				return outputResult{}, fmt.Errorf("%w: %s", errToolAttempt, event.Item.Type)
			}
		}
	}
	if !completed {
		return outputResult{}, errors.New("Codex output ended before turn completion")
	}
	if !usageSeen {
		return outputResult{}, errors.New("Codex turn omitted usage")
	}
	if text.Len() == 0 {
		return outputResult{}, errors.New("Codex turn returned empty output")
	}
	if usage.OutputTokens == 0 {
		return outputResult{}, errors.New("Codex turn reported zero output usage")
	}
	if maximumOutputTokens == 0 || usage.OutputTokens > maximumOutputTokens {
		return outputResult{}, errors.New("Codex output exceeded the reserved token limit")
	}
	return outputResult{text: text.String(), usage: usage}, nil
}
