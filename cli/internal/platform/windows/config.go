package windows

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// Config defines the Windows host controls that can be applied by Myference.
// It deliberately excludes LAN listeners, firewall settings, and endpoint discovery.
type Config struct {
	KeepAlive            string `json:"keepAlive"`
	ContextLength        int    `json:"contextLength"`
	MaxLoadedModels      int    `json:"maxLoadedModels"`
	NumParallel          int    `json:"numParallel"`
	FlashAttention       bool   `json:"flashAttention"`
	KVCacheType          string `json:"kvCacheType"`
	PerformancePowerPlan bool   `json:"performancePowerPlan"`
	ProcessPriority      string `json:"processPriority"`
	RequireACPower       bool   `json:"requireACPower"`
}

// DefaultConfig keeps provider-session tuning explicit and reversible.
func DefaultConfig() Config {
	return Config{
		KeepAlive:            "-1",
		ContextLength:        4096,
		MaxLoadedModels:      1,
		NumParallel:          1,
		FlashAttention:       true,
		KVCacheType:          "q8_0",
		PerformancePowerPlan: true,
		ProcessPriority:      "High",
		RequireACPower:       true,
	}
}

// UnmarshalJSON rejects every field outside this safe, local host-control schema.
func (config *Config) UnmarshalJSON(data []byte) error {
	type configAlias Config
	var decoded configAlias
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&decoded); err != nil {
		return err
	}
	if decoder.More() {
		return fmt.Errorf("invalid trailing configuration data")
	}
	*config = Config(decoded)
	return config.Validate()
}

// Validate rejects invalid Ollama tuning.
func (config Config) Validate() error {
	if config.ContextLength < 512 {
		return fmt.Errorf("context length must be at least 512")
	}
	if config.MaxLoadedModels < 1 {
		return fmt.Errorf("max loaded models must be at least 1")
	}
	if config.NumParallel < 1 {
		return fmt.Errorf("num parallel must be at least 1")
	}
	switch config.KVCacheType {
	case "f16", "q8_0", "q4_0":
	default:
		return fmt.Errorf("KV cache type %q is not supported", config.KVCacheType)
	}
	switch config.ProcessPriority {
	case "Normal", "AboveNormal", "High":
	default:
		return fmt.Errorf("process priority %q is not supported", config.ProcessPriority)
	}
	return nil
}

// ValidatePower enforces the AC-power policy unless the caller explicitly opts in
// to battery operation with --allow-battery.
func (config Config) ValidatePower(onACPower, allowBattery bool) error {
	if config.RequireACPower && !onACPower && !allowBattery {
		return fmt.Errorf("AC power is required; connect AC power or use --allow-battery")
	}
	return nil
}
