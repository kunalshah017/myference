//go:build darwin

package darwin

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const serviceLabel = "com.myference.provider"

type CommandRunner func(string, ...string) error
type Lifecycle struct {
	Executable, ConfigPath, PlistPath, LogPath string
	run                                        CommandRunner
}

func New(executable, configPath, plistPath, logPath string, runner CommandRunner) (Lifecycle, error) {
	for _, path := range []string{executable, configPath, plistPath, logPath} {
		if !filepath.IsAbs(path) {
			return Lifecycle{}, errors.New("launchd lifecycle paths must be absolute")
		}
	}
	if runner == nil {
		runner = func(name string, args ...string) error {
			command := exec.Command(name, args...)
			command.Stdout, command.Stderr = os.Stdout, os.Stderr
			return command.Run()
		}
	}
	return Lifecycle{Executable: executable, ConfigPath: configPath, PlistPath: plistPath, LogPath: logPath, run: runner}, nil
}

func (l Lifecycle) Install() error {
	if err := os.MkdirAll(filepath.Dir(l.PlistPath), 0o700); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(l.PlistPath), ".myference-launchd-*")
	if err != nil {
		return err
	}
	name := temporary.Name()
	defer os.Remove(name)
	if _, err := temporary.WriteString(l.plist()); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := l.run("plutil", "-lint", name); err != nil {
		return err
	}
	_ = l.run("launchctl", "bootout", l.domain(), l.PlistPath)
	if err := os.Rename(name, l.PlistPath); err != nil {
		return err
	}
	return l.run("launchctl", "bootstrap", l.domain(), l.PlistPath)
}
func (l Lifecycle) Start() error {
	_ = l.run("launchctl", "bootstrap", l.domain(), l.PlistPath)
	return l.run("launchctl", "kickstart", "-k", l.service())
}
func (l Lifecycle) Stop() error   { return l.run("launchctl", "bootout", l.domain(), l.PlistPath) }
func (l Lifecycle) Status() error { return l.run("launchctl", "print", l.service()) }
func (l Lifecycle) Uninstall() error {
	_ = l.run("launchctl", "bootout", l.domain(), l.PlistPath)
	if err := os.Remove(l.PlistPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
func (l Lifecycle) domain() string  { return fmt.Sprintf("gui/%d", os.Getuid()) }
func (l Lifecycle) service() string { return l.domain() + "/" + serviceLabel }
func (l Lifecycle) plist() string {
	escape := func(value string) string {
		var output bytes.Buffer
		_ = xml.EscapeText(&output, []byte(value))
		return output.String()
	}
	return `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>Label</key><string>` + serviceLabel + `</string>
<key>ProgramArguments</key><array><string>` + escape(l.Executable) + `</string><string>serve</string><string>--config</string><string>` + escape(l.ConfigPath) + `</string></array>
<key>KeepAlive</key><true/><key>RunAtLoad</key><true/><key>ProcessType</key><string>Background</string>
<key>StandardOutPath</key><string>` + escape(l.LogPath) + `</string><key>StandardErrorPath</key><string>` + escape(l.LogPath) + `</string>
</dict></plist>
`
}
