//go:build !windows && !darwin

package credential

import (
	"errors"
	"strings"
)

var (
	ErrInvalidCredentialKey = errors.New("credential service, account, and secret are required")
	ErrUnsupportedPlatform  = errors.New("secure credential storage is supported on Windows and macOS only")
)

func Save(service, account, secret string) error {
	if invalid(service, account) || secret == "" {
		return ErrInvalidCredentialKey
	}
	return ErrUnsupportedPlatform
}

func Load(service, account string) (string, error) {
	if invalid(service, account) {
		return "", ErrInvalidCredentialKey
	}
	return "", ErrUnsupportedPlatform
}

func Delete(service, account string) error {
	if invalid(service, account) {
		return ErrInvalidCredentialKey
	}
	return ErrUnsupportedPlatform
}

func invalid(service, account string) bool {
	return service == "" || account == "" || strings.ContainsRune(service, 0) || strings.ContainsRune(account, 0)
}
