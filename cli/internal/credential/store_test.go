package credential

import (
	"errors"
	"testing"
)

func TestCredentialStoreRejectsEmptyIdentifiersWithoutTouchingKeychain(t *testing.T) {
	for _, tc := range []struct{ service, account, secret string }{
		{"", "machine", "token"},
		{"myference", "", "token"},
		{"myference", "machine", ""},
	} {
		if err := Save(tc.service, tc.account, tc.secret); !errors.Is(err, ErrInvalidCredentialKey) {
			t.Fatalf("Save(%q,%q): %v", tc.service, tc.account, err)
		}
	}
	if _, err := Load("", "machine"); !errors.Is(err, ErrInvalidCredentialKey) {
		t.Fatalf("Load: %v", err)
	}
	if err := Delete("myference", ""); !errors.Is(err, ErrInvalidCredentialKey) {
		t.Fatalf("Delete: %v", err)
	}
}
