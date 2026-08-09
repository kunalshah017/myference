package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

type Config struct {
	ServerURL       string    `json:"server_url"`
	AccountID       string    `json:"account_id"`
	MachineID       string    `json:"machine_id"`
	ChainID         uint64    `json:"chain_id,omitempty"`
	ContractAddress string    `json:"contract_address,omitempty"`
	SignerAddress   string    `json:"signer_address,omitempty"`
	Backends        []Backend `json:"backends,omitempty"`
}

type Backend struct {
	Name         string `json:"name"`
	OfferID      string `json:"offer_id,omitempty"`
	Kind         string `json:"kind"`
	URL          string `json:"url"`
	Model        string `json:"model"`
	PriceVersion uint64 `json:"price_version,omitempty"`
	Enabled      bool   `json:"enabled"`
	Image        string `json:"image,omitempty"`
}

func (b Backend) EffectiveOfferID() string {
	if b.OfferID != "" {
		return b.OfferID
	}
	return b.Name
}

func Load(path string) (Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return Config{}, err
	}
	defer file.Close()
	var cfg Config
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&cfg); err != nil {
		return Config{}, err
	}
	if cfg.ServerURL == "" || cfg.AccountID == "" || cfg.MachineID == "" {
		return Config{}, errors.New("server, account, and machine are required")
	}
	return cfg, nil
}

func Save(path string, cfg Config) error {
	if cfg.ServerURL == "" || cfg.AccountID == "" || cfg.MachineID == "" {
		return errors.New("server, account, and machine are required")
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	temp, err := os.CreateTemp(dir, ".myference-config-*")
	if err != nil {
		return err
	}
	tempName := temp.Name()
	defer os.Remove(tempName)
	if err := temp.Chmod(0o600); err != nil {
		temp.Close()
		return err
	}
	encoder := json.NewEncoder(temp)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(cfg); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(tempName, path)
}
