package claude

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/kunalshah017/myference/cli/internal/backend"
)

func TestArgumentsDisableToolsAndPersistence(t *testing.T) {
	want := []string{"-p", "--model", "sonnet", "--output-format", "json", "--no-session-persistence", "--safe-mode", "--strict-mcp-config", "--tools", "", "--disallowedTools", "mcp__*", "--max-turns", "1"}
	if got := arguments("sonnet"); !reflect.DeepEqual(got, want) {
		t.Fatalf("arguments=%v want=%v", got, want)
	}
}

func TestParseResultReturnsTextAndUsage(t *testing.T) {
	text, usage, err := parseResult([]byte(`{"type":"result","subtype":"success","is_error":false,"result":"hello","usage":{"input_tokens":11,"cache_creation_input_tokens":2,"cache_read_input_tokens":3,"output_tokens":4}}`), 10)
	if err != nil || text != "hello" || usage.InputTokens != 16 || usage.OutputTokens != 4 {
		t.Fatalf("text=%q usage=%+v err=%v", text, usage, err)
	}
}

func TestParseResultRejectsErrorsMissingUsageAndOverLimit(t *testing.T) {
	for name, input := range map[string]string{
		"error":   `{"type":"result","is_error":true,"result":"failed","usage":{"input_tokens":1,"output_tokens":1}}`,
		"usage":   `{"type":"result","is_error":false,"result":"hello"}`,
		"limit":   `{"type":"result","is_error":false,"result":"hello","usage":{"input_tokens":1,"output_tokens":11}}`,
		"content": `{"type":"result","is_error":false,"result":"","usage":{"input_tokens":1,"output_tokens":1}}`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, _, err := parseResult([]byte(input), 10); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestRunnerGenerateUsesEmptyDirectoryAndMinimalEnvironment(t *testing.T) {
	var gotArgs, gotDir []string
	runner, err := newRunner("claude", "sonnet", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	runner.commandContext = func(ctx context.Context, executable string, args ...string) *exec.Cmd {
		gotArgs = append([]string{executable}, args...)
		command := exec.CommandContext(ctx, os.Args[0], "-test.run=TestClaudeHelperProcess")
		command.Env = append(command.Env, "GO_WANT_CLAUDE_HELPER=1")
		return command
	}
	runner.commandStarted = func(command *exec.Cmd) { gotDir = append(gotDir, command.Dir, strings.Join(command.Env, "\n")) }
	var output string
	usage, err := runner.Generate(context.Background(), backend.Request{Model: "sonnet", Prompt: "hello", MaximumOutputTokens: 10}, func(value string) error {
		output += value
		return nil
	})
	if err != nil || output != "ok" || usage.InputTokens != 3 || usage.OutputTokens != 2 {
		t.Fatalf("output=%q usage=%+v err=%v", output, usage, err)
	}
	if len(gotArgs) == 0 || !reflect.DeepEqual(gotArgs[1:], arguments("sonnet")) {
		t.Fatalf("args=%v", gotArgs)
	}
	if len(gotDir) != 2 || gotDir[0] == "" || strings.Contains(gotDir[1], "ANTHROPIC_API_KEY") {
		t.Fatalf("dir/env=%v", gotDir)
	}
	if _, err := os.Stat(gotDir[0]); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("job directory remains: %v", err)
	}
}

func TestRunnerRejectsWorkspace(t *testing.T) {
	runner, _ := newRunner("claude", "sonnet", time.Minute)
	_, err := runner.Generate(context.Background(), backend.Request{Model: "sonnet", Prompt: "hello", Workspace: []backend.WorkspaceFile{{Path: "secret"}}}, func(string) error { return nil })
	if err == nil || !strings.Contains(err.Error(), "workspace") {
		t.Fatalf("error=%v", err)
	}
}

func TestClaudeHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_CLAUDE_HELPER") != "1" {
		return
	}
	_, _ = os.Stdout.WriteString(`{"type":"result","subtype":"success","is_error":false,"result":"ok","usage":{"input_tokens":3,"output_tokens":2}}`)
	os.Exit(0)
}
