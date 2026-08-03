package windows

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDefaultConfigUsesSafeHostControls(t *testing.T) {
	config := DefaultConfig()

	if config.KeepAlive != "-1" || config.ContextLength != 4096 || config.MaxLoadedModels != 1 || config.NumParallel != 1 || !config.FlashAttention || config.KVCacheType != "q8_0" || !config.PerformancePowerPlan || config.ProcessPriority != "High" || !config.RequireACPower {
		t.Fatalf("unexpected defaults: %+v", config)
	}
	if err := config.Validate(); err != nil {
		t.Fatalf("default config should be safe: %v", err)
	}
}

func TestConfigRejectsProtectedServices(t *testing.T) {
	config := DefaultConfig()
	config.StopServices = []string{"Spooler", "WinDefend"}

	err := config.Validate()
	if err == nil || !strings.Contains(err.Error(), "WinDefend") {
		t.Fatalf("Validate() error = %v, want protected-service rejection", err)
	}
}

func TestConfigProtectsWindowsUpdateServices(t *testing.T) {
	for _, service := range []string{"UsoSvc", "DoSvc", "wuauserv", "WaaSMedicSvc"} {
		t.Run(service, func(t *testing.T) {
			config := DefaultConfig()
			for _, configured := range config.StopServices {
				if strings.EqualFold(configured, service) {
					t.Fatalf("default config must not stop protected service %q", service)
				}
			}

			config.StopServices = []string{service}
			if err := config.Validate(); err == nil {
				t.Fatalf("Validate() accepted protected service %q", service)
			}
		})
	}
}

func TestConfigRequiresACPowerUnlessExplicitlyAllowed(t *testing.T) {
	config := DefaultConfig()

	if err := config.ValidatePower(false, false); err == nil {
		t.Fatal("ValidatePower() accepted battery operation without --allow-battery")
	}
	if err := config.ValidatePower(false, true); err != nil {
		t.Fatalf("ValidatePower() rejected explicit battery override: %v", err)
	}
	if err := config.ValidatePower(true, false); err != nil {
		t.Fatalf("ValidatePower() rejected AC power: %v", err)
	}
}

func TestConfigRejectsLANAndFirewallFields(t *testing.T) {
	fields := map[string]string{
		"preloadModel": `"qwen:latest"`, "backend": `{}`, "model": `"qwen:latest"`,
		"credential": `"secret"`, "priceVersion": `1`, "lan": `true`, "firewall": `true`,
		"port": `11434`, "allowedRemoteAddress": `"192.168.0.0/16"`,
	}
	for field, value := range fields {
		t.Run(field, func(t *testing.T) {
			var config Config
			err := json.Unmarshal([]byte(`{"`+field+`":`+value+`}`), &config)
			if err == nil || !strings.Contains(err.Error(), field) {
				t.Fatalf("Unmarshal(%q) error = %v, want unknown-field rejection", field, err)
			}
		})
	}
}

func TestConfigRejectsTrailingJSON(t *testing.T) {
	encoded, err := json.Marshal(DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}

	var config Config
	if err := json.Unmarshal(append(encoded, []byte(` {}`)...), &config); err == nil {
		t.Fatal("Unmarshal accepted trailing JSON")
	}
}

func TestParseCommandAcceptsWindowsActions(t *testing.T) {
	for _, action := range []string{"doctor", "status", "models", "test", "dashboard", "focus", "headless", "restore"} {
		t.Run(action, func(t *testing.T) {
			command, err := ParseCommand([]string{action})
			if err != nil {
				t.Fatalf("ParseCommand(%q): %v", action, err)
			}
			if command.Action != action {
				t.Fatalf("command.Action = %q, want %q", command.Action, action)
			}
		})
	}
}

func TestParseCommandRejectsUnknownWindowsAction(t *testing.T) {
	if _, err := ParseCommand([]string{"lan-check"}); err == nil {
		t.Fatal("ParseCommand accepted removed LAN action")
	}
}

func TestParseCommandUsesProviderOwnedFocusSurface(t *testing.T) {
	command, err := ParseCommand([]string{"focus", "status"})
	if err != nil || command.Action != "focus" {
		t.Fatalf("ParseCommand(focus) = %+v, %v", command, err)
	}
	for _, removed := range []string{"optimize", "exclusive"} {
		if _, err := ParseCommand([]string{removed}); err == nil {
			t.Fatalf("ParseCommand accepted removed action %q", removed)
		}
	}
}
