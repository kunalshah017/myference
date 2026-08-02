//go:build linux

package credential

import (
	"errors"
	"testing"
)

func TestLinuxCredentialStoreIsExplicitlyUnsupported(t *testing.T) {
	if err := Save("myference", "machine", "token"); !errors.Is(err, ErrUnsupportedPlatform) {
		t.Fatalf("Save: %v", err)
	}
	if _, err := Load("myference", "machine"); !errors.Is(err, ErrUnsupportedPlatform) {
		t.Fatalf("Load: %v", err)
	}
	if err := Delete("myference", "machine"); !errors.Is(err, ErrUnsupportedPlatform) {
		t.Fatalf("Delete: %v", err)
	}
}
