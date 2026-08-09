package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/kunalshah017/myference/cli/internal/backend"
)

const maximumResultBytes = 4 << 20

type Runner struct {
	executable, model string
	timeout           time.Duration
	commandContext    func(context.Context, string, ...string) *exec.Cmd
	commandStarted    func(*exec.Cmd)
}

func New(model string, timeout time.Duration) (*Runner, error) {
	executable, err := exec.LookPath("claude")
	if err != nil {
		return nil, errors.New("Claude CLI is not installed or not available on PATH")
	}
	return newRunner(executable, model, timeout)
}

func newRunner(executable, model string, timeout time.Duration) (*Runner, error) {
	if strings.TrimSpace(executable) == "" || strings.TrimSpace(model) == "" || timeout <= 0 {
		return nil, errors.New("Claude executable, model, and positive timeout are required")
	}
	return &Runner{executable: executable, model: model, timeout: timeout, commandContext: exec.CommandContext}, nil
}

func (runner *Runner) Models(context.Context) ([]backend.Model, error) {
	return []backend.Model{{Name: runner.model}}, nil
}

func (runner *Runner) Generate(parent context.Context, input backend.Request, emit func(string) error) (backend.Usage, error) {
	if input.Model != runner.model || strings.TrimSpace(input.Prompt) == "" || emit == nil {
		return backend.Usage{}, errors.New("configured model, prompt, and callback are required")
	}
	if len(input.Workspace) != 0 {
		return backend.Usage{}, errors.New("native Claude is a model-only provider and does not accept workspace files")
	}
	maximumOutputTokens := input.MaximumOutputTokens
	if maximumOutputTokens == 0 {
		maximumOutputTokens = 4096
	}
	jobDirectory, err := os.MkdirTemp("", "myference-claude-")
	if err != nil {
		return backend.Usage{}, err
	}
	defer os.RemoveAll(jobDirectory)

	ctx, cancel := context.WithTimeout(parent, runner.timeout)
	defer cancel()
	command := runner.commandContext(ctx, runner.executable, arguments(runner.model)...)
	command.Dir = jobDirectory
	command.Env = minimalEnvironment(command.Env)
	command.Stdin = strings.NewReader(input.Prompt)
	var output boundedBuffer
	var diagnostic boundedBuffer
	command.Stdout = &output
	command.Stderr = &diagnostic
	if runner.commandStarted != nil {
		runner.commandStarted(command)
	}
	started := time.Now()
	if err := command.Run(); err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return backend.Usage{}, errors.New("Claude generation timed out")
		}
		return backend.Usage{}, fmt.Errorf("Claude generation failed: %w", err)
	}
	if output.overflow || diagnostic.overflow {
		return backend.Usage{}, errors.New("Claude output exceeded the safe size limit")
	}
	text, usage, err := parseResult(output.Bytes(), maximumOutputTokens)
	if err != nil {
		return backend.Usage{}, err
	}
	usage.ComputeMilliseconds = uint64(time.Since(started).Milliseconds())
	if err := emit(text); err != nil {
		return backend.Usage{}, err
	}
	return usage, nil
}

func arguments(model string) []string {
	return []string{"-p", "--model", model, "--output-format", "json", "--no-session-persistence", "--safe-mode", "--strict-mcp-config", "--tools", "", "--disallowedTools", "mcp__*", "--max-turns", "1"}
}

func parseResult(raw []byte, maximumOutputTokens uint64) (string, backend.Usage, error) {
	var result struct {
		Type    string `json:"type"`
		Subtype string `json:"subtype"`
		IsError bool   `json:"is_error"`
		Result  string `json:"result"`
		Usage   *struct {
			Input         uint64 `json:"input_tokens"`
			CacheCreation uint64 `json:"cache_creation_input_tokens"`
			CacheRead     uint64 `json:"cache_read_input_tokens"`
			Output        uint64 `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", backend.Usage{}, errors.New("Claude returned invalid JSON")
	}
	if result.Type != "result" || result.IsError || result.Subtype == "error" {
		return "", backend.Usage{}, errors.New("Claude generation failed")
	}
	if strings.TrimSpace(result.Result) == "" {
		return "", backend.Usage{}, errors.New("Claude returned empty output")
	}
	if result.Usage == nil || result.Usage.Output == 0 {
		return "", backend.Usage{}, errors.New("Claude omitted token usage")
	}
	if result.Usage.Output > maximumOutputTokens {
		return "", backend.Usage{}, errors.New("Claude reported usage above the output token limit")
	}
	usage := backend.Usage{InputTokens: result.Usage.Input + result.Usage.CacheCreation + result.Usage.CacheRead, OutputTokens: result.Usage.Output}
	return result.Result, usage, nil
}

func minimalEnvironment(existing []string) []string {
	allowed := []string{"HOME", "USERPROFILE", "APPDATA", "LOCALAPPDATA", "PATH", "SystemRoot", "TMP", "TEMP"}
	result := make([]string, 0, len(allowed)+len(existing))
	for _, value := range existing {
		if strings.HasPrefix(value, "GO_WANT_CLAUDE_HELPER=") {
			result = append(result, value)
		}
	}
	for _, name := range allowed {
		if value, ok := os.LookupEnv(name); ok {
			result = append(result, name+"="+value)
		}
	}
	return result
}

type boundedBuffer struct {
	bytes.Buffer
	overflow bool
}

func (buffer *boundedBuffer) Write(value []byte) (int, error) {
	written := len(value)
	remaining := maximumResultBytes - buffer.Len()
	if remaining <= 0 {
		buffer.overflow = true
		return written, nil
	}
	if len(value) > remaining {
		value = value[:remaining]
		buffer.overflow = true
	}
	_, _ = buffer.Buffer.Write(value)
	return written, nil
}
