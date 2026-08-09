package provider

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

type OfferStatus struct {
	OfferID string `json:"offerId"`
	Model   string `json:"model"`
	Healthy bool   `json:"healthy"`
	Error   string `json:"error,omitempty"`
}

type RequestStatus struct {
	RequestID           string    `json:"requestId"`
	OfferID             string    `json:"offerId,omitempty"`
	Model               string    `json:"model,omitempty"`
	State               string    `json:"state"`
	InputTokens         uint64    `json:"inputTokens,omitempty"`
	OutputTokens        uint64    `json:"outputTokens,omitempty"`
	ComputeMilliseconds uint64    `json:"computeMilliseconds,omitempty"`
	EarningsWei         string    `json:"earningsWei,omitempty"`
	Error               string    `json:"error,omitempty"`
	StartedAt           time.Time `json:"startedAt,omitempty"`
	CompletedAt         time.Time `json:"completedAt,omitempty"`
}

type StatusSnapshot struct {
	Connected           bool            `json:"connected"`
	StartedAt           time.Time       `json:"startedAt"`
	UpdatedAt           time.Time       `json:"updatedAt"`
	Requests            uint64          `json:"requests"`
	InputTokens         uint64          `json:"inputTokens"`
	OutputTokens        uint64          `json:"outputTokens"`
	ComputeMilliseconds uint64          `json:"computeMilliseconds"`
	RunEarningsWei      string          `json:"runEarningsWei,omitempty"`
	RecentRequests      []RequestStatus `json:"recentRequests,omitempty"`
	Offers              []OfferStatus   `json:"offers"`
}

func WriteStatusFile(path string, snapshot StatusSnapshot) error {
	if path == "" {
		return errors.New("provider status path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create provider status directory: %w", err)
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".myference-status-*.tmp")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return err
	}
	encoder := json.NewEncoder(temporary)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(snapshot); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := replaceStatusFile(temporaryPath, path); err != nil {
		return fmt.Errorf("replace provider status: %w", err)
	}
	return nil
}

func LoadStatusFile(path string) (StatusSnapshot, error) {
	file, err := os.Open(path)
	if err != nil {
		return StatusSnapshot{}, err
	}
	defer file.Close()
	var snapshot StatusSnapshot
	decoder := json.NewDecoder(io.LimitReader(file, 4<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&snapshot); err != nil {
		return StatusSnapshot{}, err
	}
	if snapshot.StartedAt.IsZero() {
		return StatusSnapshot{}, errors.New("provider status start time is missing")
	}
	return snapshot, nil
}

func RemoveStatusFile(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
