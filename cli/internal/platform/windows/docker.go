package windows

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

type DockerStatus struct {
	DockerPath    string
	DesktopPath   string
	EngineReady   bool
	EngineOS      string
	MissingImages []string
	Error         string
}

type DockerRuntime struct {
	DockerPath  string
	DesktopPath string
	run         func(context.Context, string, ...string) ([]byte, error)
	start       func(string) error
	wait        func(context.Context, time.Duration) error
}

var immutableDockerImage = regexp.MustCompile(`^[^[:space:]]+@sha256:[0-9a-f]{64}$`)

func DiscoverDockerRuntime() DockerRuntime {
	dockerPath, _ := exec.LookPath("docker.exe")
	if dockerPath == "" {
		dockerPath, _ = exec.LookPath("docker")
	}
	desktopPath := ""
	for _, candidate := range dockerDesktopCandidates() {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			desktopPath = candidate
			break
		}
	}
	return DockerRuntime{DockerPath: dockerPath, DesktopPath: desktopPath}
}

func dockerDesktopCandidates() []string {
	result := []string{}
	if programFiles := os.Getenv("ProgramFiles"); programFiles != "" {
		result = append(result, filepath.Join(programFiles, "Docker", "Docker", "Docker Desktop.exe"))
	}
	if local := os.Getenv("LOCALAPPDATA"); local != "" {
		result = append(result, filepath.Join(local, "Docker", "Docker Desktop.exe"))
	}
	return result
}

func (r DockerRuntime) Status(ctx context.Context, images []string) DockerStatus {
	status := DockerStatus{DockerPath: r.DockerPath, DesktopPath: r.DesktopPath}
	if strings.TrimSpace(r.DockerPath) == "" {
		status.Error = "Docker CLI was not found; install Docker Desktop"
		return status
	}
	osType, output, err := r.engineOS(ctx)
	if err != nil {
		status.Error = commandFailure("Docker engine is unavailable", output, err)
		return status
	}
	status.EngineReady = true
	status.EngineOS = osType
	if osType != "linux" {
		status.Error = "Docker Desktop is using Windows containers; switch to Linux containers"
		return status
	}
	for _, image := range uniqueImages(images) {
		if _, err := r.runCommand(ctx, r.DockerPath, "image", "inspect", image); err != nil {
			status.MissingImages = append(status.MissingImages, image)
		}
	}
	return status
}

func (r DockerRuntime) Prepare(ctx context.Context, images []string, timeout time.Duration) error {
	if strings.TrimSpace(r.DockerPath) == "" {
		return errors.New("Docker CLI was not found; install Docker Desktop")
	}
	if timeout <= 0 {
		return errors.New("Docker startup timeout must be positive")
	}
	osType, output, err := r.engineOS(ctx)
	if err != nil {
		if strings.TrimSpace(r.DesktopPath) == "" {
			return fmt.Errorf("Docker engine is unavailable and Docker Desktop could not be found: %s", cleanCommandError(output, err))
		}
		if err := r.startProcess(r.DesktopPath); err != nil {
			return fmt.Errorf("start Docker Desktop: %w", err)
		}
		deadline, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		lastOutput, lastErr := output, err
		for {
			osType, lastOutput, lastErr = r.engineOS(deadline)
			if lastErr == nil {
				break
			}
			if waitErr := r.waitFor(deadline, 250*time.Millisecond); waitErr != nil {
				return fmt.Errorf("Docker Desktop did not become ready within %s: %s", timeout, cleanCommandError(lastOutput, lastErr))
			}
		}
	}
	if osType != "linux" {
		return fmt.Errorf("Docker Desktop is using %q containers; switch to Linux containers", osType)
	}
	for _, image := range uniqueImages(images) {
		if !immutableDockerImage.MatchString(image) {
			return fmt.Errorf("command-agent image must be digest-pinned: %q", image)
		}
		if _, inspectErr := r.runCommand(ctx, r.DockerPath, "image", "inspect", image); inspectErr == nil {
			continue
		}
		pullOutput, pullErr := r.runCommand(ctx, r.DockerPath, "pull", image)
		if pullErr != nil {
			return fmt.Errorf("pull digest-pinned command-agent image %q: %s", image, cleanCommandError(pullOutput, pullErr))
		}
		inspectOutput, inspectErr := r.runCommand(ctx, r.DockerPath, "image", "inspect", image)
		if inspectErr != nil {
			return fmt.Errorf("verify pulled command-agent image %q: %s", image, cleanCommandError(inspectOutput, inspectErr))
		}
	}
	return nil
}

func (r DockerRuntime) engineOS(ctx context.Context) (string, []byte, error) {
	output, err := r.runCommand(ctx, r.DockerPath, "info", "--format", "{{.OSType}}")
	return strings.ToLower(strings.TrimSpace(string(output))), output, err
}

func (r DockerRuntime) runCommand(ctx context.Context, executable string, args ...string) ([]byte, error) {
	if r.run != nil {
		return r.run(ctx, executable, args...)
	}
	return exec.CommandContext(ctx, executable, args...).CombinedOutput()
}

func (r DockerRuntime) startProcess(path string) error {
	if r.start != nil {
		return r.start(path)
	}
	return startHiddenProcess(path)
}

func (r DockerRuntime) waitFor(ctx context.Context, duration time.Duration) error {
	if r.wait != nil {
		return r.wait(ctx, duration)
	}
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func uniqueImages(images []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(images))
	for _, image := range images {
		image = strings.TrimSpace(image)
		if image != "" && !seen[image] {
			seen[image] = true
			result = append(result, image)
		}
	}
	sort.Strings(result)
	return result
}

func cleanCommandError(output []byte, err error) string {
	message := strings.TrimSpace(string(output))
	if message != "" {
		return message
	}
	if err != nil {
		return err.Error()
	}
	return "unknown Docker error"
}

func commandFailure(prefix string, output []byte, err error) string {
	return prefix + ": " + cleanCommandError(output, err)
}
