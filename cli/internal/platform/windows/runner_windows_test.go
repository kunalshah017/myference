//go:build windows

package windows

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestParseActivePowerScheme(t *testing.T) {
	got, err := parseActivePowerScheme("Power Scheme GUID: 381b4222-f694-41f0-9685-ff5bb260df2e  (Balanced)")
	if err != nil || got != "381b4222-f694-41f0-9685-ff5bb260df2e" {
		t.Fatalf("parseActivePowerScheme() = %q, %v", got, err)
	}
}

func TestParseProcessSnapshots(t *testing.T) {
	input := `[{"ProcessId":42,"Name":"Discord.exe","ExecutablePath":"C:\\Discord\\Discord.exe"},{"ProcessId":7,"Name":"ollama.exe","ExecutablePath":"C:\\Ollama\\ollama.exe"}]`
	got, err := parseProcessSnapshots(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	want := []ProcessSnapshot{{PID: 42, Name: "Discord.exe", Executable: `C:\Discord\Discord.exe`}, {PID: 7, Name: "ollama.exe", Executable: `C:\Ollama\ollama.exe`}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("processes=%+v want=%+v", got, want)
	}
}

func TestParseLidActionsAndShellPolicy(t *testing.T) {
	input := "Current AC Power Setting Index: 0x00000000\r\nCurrent DC Power Setting Index: 0x00000001"
	ac, dc, err := parseLidActions(input)
	if err != nil || ac != 0 || dc != 1 {
		t.Fatalf("lid=%d/%d err=%v", ac, dc, err)
	}
	had, value, err := parseShellPolicy("    Shell    REG_SZ    explorer.exe\r\n")
	if err != nil || !had || value != "explorer.exe" {
		t.Fatalf("shell=%v %q err=%v", had, value, err)
	}
	had, value, err = parseShellPolicy("")
	if err != nil || had || value != "" {
		t.Fatalf("absent shell=%v %q err=%v", had, value, err)
	}
}

func TestReadLidActionsIncludesHiddenPowerSettings(t *testing.T) {
	run := func(_ context.Context, program string, args ...string) ([]byte, error) {
		if program != "powercfg.exe" || !reflect.DeepEqual(args, []string{"/qh", "SCHEME_CURRENT", "SUB_BUTTONS", "LIDACTION"}) {
			return nil, fmt.Errorf("unexpected command: %s %v", program, args)
		}
		return []byte("Current AC Power Setting Index: 0x00000001\r\nCurrent DC Power Setting Index: 0x00000001"), nil
	}

	ac, dc, err := readLidActions(context.Background(), run)
	if err != nil || ac != 1 || dc != 1 {
		t.Fatalf("readLidActions() = %d/%d, %v", ac, dc, err)
	}
}
