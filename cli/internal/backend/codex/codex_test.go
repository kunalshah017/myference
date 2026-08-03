package codex

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/kunalshah017/myference/cli/internal/backend"
)

func TestParseOutputBuffersTextAndReturnsUsage(t *testing.T) {
	stream := strings.Join([]string{
		`{"type":"thread.started","thread_id":"thread"}`,
		`{"type":"turn.started"}`,
		`{"type":"item.completed","item":{"id":"message","type":"agent_message","text":"o"}}`,
		`{"type":"item.completed","item":{"id":"message","type":"agent_message","text":"k"}}`,
		`{"type":"turn.completed","usage":{"input_tokens":11,"cached_input_tokens":3,"output_tokens":4,"reasoning_output_tokens":1}}`,
	}, "\n")

	result, err := parseOutput(strings.NewReader(stream), 8)
	if err != nil {
		t.Fatal(err)
	}
	if result.text != "ok" || result.usage.InputTokens != 11 || result.usage.OutputTokens != 4 {
		t.Fatalf("result=%+v", result)
	}
}

func TestParseOutputRejectsToolEvents(t *testing.T) {
	for _, itemType := range []string{"command_execution", "file_change", "mcp_tool_call", "web_search", "tool_call"} {
		t.Run(itemType, func(t *testing.T) {
			stream := `{"type":"item.started","item":{"type":"` + itemType + `"}}`
			_, err := parseOutput(strings.NewReader(stream), 8)
			if !errors.Is(err, errToolAttempt) {
				t.Fatalf("error=%v, want errToolAttempt", err)
			}
		})
	}
}

func TestParseOutputRejectsInvalidCompletion(t *testing.T) {
	tests := map[string]string{
		"turn failed":      `{"type":"turn.failed","error":{"message":"failed"}}`,
		"missing turn":     `{"type":"item.completed","item":{"type":"agent_message","text":"ok"}}`,
		"missing usage":    `{"type":"item.completed","item":{"type":"agent_message","text":"ok"}}` + "\n" + `{"type":"turn.completed"}`,
		"empty output":     `{"type":"turn.completed","usage":{"input_tokens":1,"output_tokens":1}}`,
		"over reservation": `{"type":"item.completed","item":{"type":"agent_message","text":"ok"}}` + "\n" + `{"type":"turn.completed","usage":{"input_tokens":1,"output_tokens":9}}`,
	}
	for name, stream := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := parseOutput(strings.NewReader(stream), 8); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestRunnerUsesIsolatedCodexExecAndBuffersOutput(t *testing.T) {
	root := t.TempDir()
	sourceAuth := filepath.Join(root, "source-auth.json")
	if err := os.WriteFile(sourceAuth, []byte(`{"tokens":"test-only"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	providerHome := filepath.Join(root, "provider-home")
	skillsRoot := filepath.Join(root, "personal-skills")
	if err := os.MkdirAll(filepath.Join(skillsRoot, "demo"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillsRoot, "demo", "SKILL.md"), []byte("test skill"), 0o600); err != nil {
		t.Fatal(err)
	}
	runner, err := newRunner("codex.exe", "gpt-5.6-terra", sourceAuth, skillsRoot, providerHome, "myference.exe", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	var requestedArgs []string
	var command *exec.Cmd
	runner.commandContext = func(ctx context.Context, _ string, args ...string) *exec.Cmd {
		requestedArgs = append([]string(nil), args...)
		command = exec.CommandContext(ctx, os.Args[0], "-test.run=^TestCodexHelperProcess$", "--", "--codex-helper")
		return command
	}
	var output strings.Builder
	usage, err := runner.Generate(t.Context(), backend.Request{Model: "gpt-5.6-terra", Prompt: "hello", MaximumOutputTokens: 8}, func(chunk string) error {
		output.WriteString(chunk)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if output.String() != "hello" || usage.InputTokens != 11 || usage.OutputTokens != 4 || usage.ComputeMilliseconds == 0 {
		t.Fatalf("output=%q usage=%+v", output.String(), usage)
	}
	wantArgs := []string{"exec", "--ephemeral", "--json", "--ignore-rules", "--dangerously-bypass-hook-trust", "--sandbox", "read-only", "--skip-git-repo-check", "--model", "gpt-5.6-terra", "-"}
	if !slices.Equal(requestedArgs, wantArgs) {
		t.Fatalf("args=%q, want %q", requestedArgs, wantArgs)
	}
	environment := strings.Join(command.Env, "\n")
	for _, required := range []string{"CODEX_HOME=" + providerHome, "HOME=" + providerHome, "USERPROFILE=" + providerHome, "MYFERENCE_CODEX_TOOL_MARKER="} {
		if !strings.Contains(environment, required) {
			t.Fatalf("missing environment %q in %q", required, environment)
		}
	}
	for _, forbidden := range []string{"OPENAI_API_KEY=", "CODEX_API_KEY="} {
		if strings.Contains(environment, forbidden) {
			t.Fatalf("inherited secret environment %q", forbidden)
		}
	}
	if _, err := os.Stat(command.Dir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("job directory still exists: %v", err)
	}
	storedAuth, err := os.ReadFile(filepath.Join(providerHome, "auth.json"))
	if err != nil || string(storedAuth) != `{"tokens":"test-only"}` {
		t.Fatalf("stored auth=%q err=%v", storedAuth, err)
	}
	configuration, err := os.ReadFile(filepath.Join(providerHome, "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{`web_search = "disabled"`, `agents.enabled = false`, `matcher = "*"`, `internal codex-deny-tool`, strconv.Quote(filepath.Join(skillsRoot, "demo", "SKILL.md")), `enabled = false`} {
		if !strings.Contains(string(configuration), required) {
			t.Fatalf("provider config missing %q: %s", required, configuration)
		}
	}
}

func TestRunnerRejectsWorkspaceAndToolMarker(t *testing.T) {
	root := t.TempDir()
	sourceAuth := filepath.Join(root, "auth.json")
	if err := os.WriteFile(sourceAuth, []byte(`{"tokens":"test-only"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	runner, err := newRunner("codex.exe", "model", sourceAuth, filepath.Join(root, "skills"), filepath.Join(root, "home"), "myference.exe", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Generate(t.Context(), backend.Request{Model: "model", Prompt: "hello", Workspace: []backend.WorkspaceFile{{Path: "file", Content: []byte("secret")}}}, func(string) error { return nil }); err == nil {
		t.Fatal("workspace input accepted")
	}
	runner.commandContext = func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		return exec.CommandContext(ctx, os.Args[0], "-test.run=^TestCodexHelperProcess$", "--", "--codex-helper", "--write-marker")
	}
	emitted := false
	_, err = runner.Generate(t.Context(), backend.Request{Model: "model", Prompt: "hello", MaximumOutputTokens: 8}, func(string) error { emitted = true; return nil })
	if !errors.Is(err, errToolAttempt) || emitted {
		t.Fatalf("error=%v emitted=%v", err, emitted)
	}
}

func TestCodexHelperProcess(t *testing.T) {
	if !slices.Contains(os.Args, "--codex-helper") {
		return
	}
	if slices.Contains(os.Args, "--write-marker") {
		if err := os.WriteFile(os.Getenv("MYFERENCE_CODEX_TOOL_MARKER"), []byte("blocked"), 0o600); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
	}
	prompt, err := io.ReadAll(os.Stdin)
	if err != nil {
		os.Exit(2)
	}
	fmt.Printf("{\"type\":\"item.completed\",\"item\":{\"type\":\"agent_message\",\"text\":%q}}\n", string(prompt))
	fmt.Println(`{"type":"turn.completed","usage":{"input_tokens":11,"output_tokens":4}}`)
	os.Exit(0)
}
