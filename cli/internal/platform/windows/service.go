//go:build windows

package windows

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
)

type Service struct{ Executable, ConfigPath, Installer string }

func DiscoverService(configPath string) (Service, error) {
	executable, err := os.Executable()
	if err != nil {
		return Service{}, err
	}
	candidates := []string{filepath.Join(filepath.Dir(executable), "packaging", "windows", "install.ps1"), filepath.Join(filepath.Dir(executable), "install-windows.ps1"), filepath.Join(filepath.Dir(executable), "install.ps1")}
	for _, installer := range candidates {
		if _, statErr := os.Stat(installer); statErr == nil {
			return Service{Executable: executable, ConfigPath: configPath, Installer: installer}, nil
		}
	}
	return Service{}, errors.New("Windows service installer not found beside the CLI")
}
func (s Service) Install(ctx context.Context) error {
	return s.powerShell(ctx, "-File", s.Installer, "-Executable", s.Executable, "-Config", s.ConfigPath)
}
func (s Service) Uninstall(ctx context.Context) error {
	return s.powerShell(ctx, "-File", s.Installer, "-Executable", s.Executable, "-Config", s.ConfigPath, "-Remove")
}
func (s Service) Start(ctx context.Context) error {
	_ = os.Remove(s.ConfigPath + ".stop")
	return s.powerShell(ctx, "-Command", "Start-ScheduledTask -TaskName 'Myference Provider'")
}
func (s Service) Stop(ctx context.Context) error {
	if err := os.WriteFile(s.ConfigPath+".stop", []byte("stop\n"), 0o600); err != nil {
		return err
	}
	return s.powerShell(ctx, "-Command", "$deadline=(Get-Date).AddSeconds(35); while((Get-ScheduledTask -TaskName 'Myference Provider').State -eq 'Running' -and (Get-Date) -lt $deadline){Start-Sleep -Milliseconds 500}; if((Get-ScheduledTask -TaskName 'Myference Provider').State -eq 'Running'){Stop-ScheduledTask -TaskName 'Myference Provider'}")
}
func (s Service) Status(ctx context.Context) error {
	return s.powerShell(ctx, "-Command", "Get-ScheduledTask -TaskName 'Myference Provider' | Format-List TaskName,State")
}
func (Service) powerShell(ctx context.Context, args ...string) error {
	base := []string{"-NoProfile", "-ExecutionPolicy", "Bypass"}
	command := exec.CommandContext(ctx, "powershell.exe", append(base, args...)...)
	command.Stdout, command.Stderr, command.Stdin = os.Stdout, os.Stderr, os.Stdin
	return command.Run()
}
