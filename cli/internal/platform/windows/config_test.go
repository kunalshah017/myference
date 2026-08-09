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
