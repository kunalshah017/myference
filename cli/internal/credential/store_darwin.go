//go:build darwin

package credential

import (
	"errors"
	"os/exec"
	"strings"
)

var ErrInvalidCredentialKey = errors.New("credential service, account, and secret are required")

func Save(service, account, secret string) error {
	if invalid(service, account) || secret == "" {
		return ErrInvalidCredentialKey
	}
	return exec.Command("security", "add-generic-password", "-U", "-s", service, "-a", account, "-w", secret).Run()
}

func Load(service, account string) (string, error) {
	if invalid(service, account) {
		return "", ErrInvalidCredentialKey
	}
	output, err := exec.Command("security", "find-generic-password", "-s", service, "-a", account, "-w").Output()
	return strings.TrimSpace(string(output)), err
}

func Delete(service, account string) error {
	if invalid(service, account) {
		return ErrInvalidCredentialKey
	}
	return exec.Command("security", "delete-generic-password", "-s", service, "-a", account).Run()
}

func invalid(service, account string) bool {
	return service == "" || account == "" || strings.ContainsRune(service, 0) || strings.ContainsRune(account, 0)
}
