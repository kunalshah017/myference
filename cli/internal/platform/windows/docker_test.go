package windows

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestDockerRuntimePrepareUsesReadyLinuxEngineAndPresentImage(t *testing.T) {
	var commands []string
	starts := 0
	runtime := DockerRuntime{
		DockerPath:  "docker.exe",
		DesktopPath: "Docker Desktop.exe",
		run: func(_ context.Context, executable string, args ...string) ([]byte, error) {
			commands = append(commands, executable+" "+strings.Join(args, " "))
			if args[0] == "info" {
				return []byte("linux\n"), nil
			}
			return []byte("present"), nil
		},
		start: func(string) error { starts++; return nil },
	}
	image := "ghcr.io/example/codex@sha256:" + strings.Repeat("a", 64)
	if err := runtime.Prepare(context.Background(), []string{image, image}, time.Second); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"docker.exe info --format {{.OSType}}",
		"docker.exe image inspect " + image,
	}
	if starts != 0 || !reflect.DeepEqual(commands, want) {
		t.Fatalf("starts=%d commands=%v, want no start and %v", starts, commands, want)
	}
}

func TestDockerRuntimePrepareStartsWaitsAndPullsMissingPinnedImage(t *testing.T) {
	var commands []string
	infoCalls := 0
	inspectCalls := 0
	started := ""
	runtime := DockerRuntime{
		DockerPath:  "docker.exe",
		DesktopPath: "Docker Desktop.exe",
		run: func(_ context.Context, executable string, args ...string) ([]byte, error) {
			commands = append(commands, executable+" "+strings.Join(args, " "))
			switch args[0] {
			case "info":
				infoCalls++
				if infoCalls == 1 {
					return []byte("named pipe unavailable"), errors.New("exit 1")
				}
				return []byte("linux"), nil
			case "image":
				inspectCalls++
				if inspectCalls == 1 {
					return []byte("No such image"), errors.New("exit 1")
				}
				return []byte("present"), nil
			case "pull":
				return []byte("pulled"), nil
			default:
				t.Fatalf("unexpected Docker command %v", args)
				return nil, nil
			}
		},
		start: func(path string) error { started = path; return nil },
		wait:  func(context.Context, time.Duration) error { return nil },
	}
	image := "ghcr.io/example/codex@sha256:" + strings.Repeat("b", 64)
	if err := runtime.Prepare(context.Background(), []string{image}, time.Second); err != nil {
		t.Fatal(err)
	}
	if started != "Docker Desktop.exe" {
		t.Fatalf("started %q", started)
	}
	want := []string{
		"docker.exe info --format {{.OSType}}",
		"docker.exe info --format {{.OSType}}",
		"docker.exe image inspect " + image,
		"docker.exe pull " + image,
		"docker.exe image inspect " + image,
	}
	if !reflect.DeepEqual(commands, want) {
		t.Fatalf("commands=%v, want %v", commands, want)
	}
}

func TestDockerRuntimePrepareRejectsWindowsContainerEngine(t *testing.T) {
	runtime := DockerRuntime{
		DockerPath: "docker.exe",
		run: func(context.Context, string, ...string) ([]byte, error) {
			return []byte("windows"), nil
		},
	}
	err := runtime.Prepare(context.Background(), nil, time.Second)
	if err == nil || !strings.Contains(err.Error(), "Linux containers") {
		t.Fatalf("error=%v, want Linux containers action", err)
	}
}

func TestDockerRuntimePrepareTimesOutWithLastEngineError(t *testing.T) {
	runtime := DockerRuntime{
		DockerPath:  "docker.exe",
		DesktopPath: "Docker Desktop.exe",
		run: func(context.Context, string, ...string) ([]byte, error) {
			return []byte("engine is starting"), errors.New("exit 1")
		},
		start: func(string) error { return nil },
		wait: func(ctx context.Context, _ time.Duration) error {
			<-ctx.Done()
			return ctx.Err()
		},
	}
	err := runtime.Prepare(context.Background(), nil, time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "engine is starting") || !strings.Contains(err.Error(), "1ms") {
		t.Fatalf("error=%v, want bounded timeout and last engine error", err)
	}
}

func TestDockerRuntimeStatusIsReadOnlyAndReportsMissingImages(t *testing.T) {
	starts := 0
	pulls := 0
	image := "ghcr.io/example/codex@sha256:" + strings.Repeat("c", 64)
	runtime := DockerRuntime{
		DockerPath:  "docker.exe",
		DesktopPath: "Docker Desktop.exe",
		run: func(_ context.Context, _ string, args ...string) ([]byte, error) {
			if args[0] == "info" {
				return []byte("linux"), nil
			}
			if args[0] == "pull" {
				pulls++
			}
			return []byte("No such image"), errors.New("exit 1")
		},
		start: func(string) error { starts++; return nil },
	}
	status := runtime.Status(context.Background(), []string{image})
	if !status.EngineReady || status.EngineOS != "linux" || !reflect.DeepEqual(status.MissingImages, []string{image}) {
		t.Fatalf("status=%+v", status)
	}
	if starts != 0 || pulls != 0 {
		t.Fatalf("read-only status started=%d pulled=%d", starts, pulls)
	}
}
