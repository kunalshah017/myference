package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSaveAndLoadNonSecretConfiguration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	want := Config{ServerURL: "https://api.myference.network", AccountID: "acct-1", MachineID: "mach-1"}
	if err := Save(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v want %+v", got, want)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("configuration is accessible to other users: %o", info.Mode().Perm())
	}
}
