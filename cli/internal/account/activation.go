package account

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	ActivationPending   = "pending"
	ActivationConfirmed = "confirmed"
)

type ActivationOffer struct {
	OfferID      string   `json:"offer_id"`
	Model        string   `json:"model"`
	Kind         string   `json:"kind"`
	Capabilities []string `json:"capabilities,omitempty"`
	MeteringMode string   `json:"metering_mode,omitempty"`
}

type ProviderActivation struct {
	ID       string            `json:"id"`
	Status   string            `json:"status"`
	Offers   []ActivationOffer `json:"offers"`
	Versions map[string]int    `json:"versions,omitempty"`
	Expires  time.Time         `json:"expires_at"`
}

func (c *Client) CreateProviderActivation(ctx context.Context, token string, offers []ActivationOffer) (ProviderActivation, error) {
	var result ProviderActivation
	err := c.activationRequest(ctx, http.MethodPost, "/api/provider/activations", token, struct {
		Offers []ActivationOffer `json:"offers"`
	}{Offers: offers}, &result)
	return result, err
}

func (c *Client) ProviderActivation(ctx context.Context, token, id string) (ProviderActivation, error) {
	var result ProviderActivation
	err := c.activationRequest(ctx, http.MethodGet, "/api/provider/activations/"+url.PathEscape(id), token, nil, &result)
	return result, err
}

func (c *Client) activationRequest(ctx context.Context, method, path, token string, input, output any) error {
	var body io.Reader
	if input != nil {
		var encoded bytes.Buffer
		if err := json.NewEncoder(&encoded).Encode(input); err != nil {
			return err
		}
		body = &encoded
	}
	request, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+strings.TrimSpace(token))
	if input != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := c.http.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusCreated {
		message, _ := io.ReadAll(io.LimitReader(response.Body, 4<<10))
		return fmt.Errorf("server returned %s: %s", response.Status, strings.TrimSpace(string(message)))
	}
	return json.NewDecoder(io.LimitReader(response.Body, 64<<10)).Decode(output)
}
