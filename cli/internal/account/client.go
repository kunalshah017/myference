package account

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	ErrPending  = errors.New("device authorization pending")
	ErrExpired  = errors.New("device authorization expired")
	ErrConsumed = errors.New("device authorization consumed")
)

type Client struct {
	baseURL string
	http    *http.Client
}

type DeviceAuthorization struct {
	DeviceCode      string    `json:"device_code"`
	UserCode        string    `json:"user_code"`
	VerificationURI string    `json:"verification_uri"`
	ExpiresAt       time.Time `json:"expires_at"`
	ChainID         uint64    `json:"chain_id"`
	ContractAddress string    `json:"contract_address"`
}

type Machine struct {
	ID            string `json:"id"`
	AccountID     string `json:"account_id"`
	Name          string `json:"name"`
	SignerAddress string `json:"signer_address"`
}

type DeviceToken struct {
	Machine Machine `json:"machine"`
	Token   string  `json:"token"`
}

func NewClient(baseURL string, client *http.Client) (*Client, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "https" && !isLoopbackHTTP(parsed)) {
		return nil, errors.New("server URL must use HTTPS except on loopback")
	}
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return &Client{baseURL: strings.TrimSuffix(baseURL, "/"), http: client}, nil
}

func (c *Client) CreateDeviceAuthorization(ctx context.Context, machineName, signerAddress string) (DeviceAuthorization, error) {
	var result DeviceAuthorization
	err := c.post(ctx, "/auth/device", map[string]string{"machine_name": machineName, "signer_address": signerAddress}, &result)
	return result, err
}

func (c *Client) ExchangeDeviceAuthorization(ctx context.Context, deviceCode string) (DeviceToken, error) {
	var result DeviceToken
	err := c.post(ctx, "/auth/device/token", map[string]string{"device_code": deviceCode}, &result)
	return result, err
}

func (c *Client) post(ctx context.Context, path string, input, output any) error {
	return c.request(ctx, http.MethodPost, path, "", input, output)
}

func (c *Client) authorizedRequest(ctx context.Context, method, path, token string, input, output any) error {
	return c.request(ctx, method, path, strings.TrimSpace(token), input, output)
}

func (c *Client) request(ctx context.Context, method, path, token string, input, output any) error {
	var body bytes.Buffer
	if input != nil {
		if err := json.NewEncoder(&body).Encode(input); err != nil {
			return err
		}
	}
	var reader io.Reader
	if input != nil {
		reader = &body
	}
	request, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return err
	}
	if input != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response, err := c.http.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	switch response.StatusCode {
	case http.StatusOK, http.StatusCreated:
		return json.NewDecoder(io.LimitReader(response.Body, 64<<10)).Decode(output)
	case http.StatusTooEarly:
		return ErrPending
	case http.StatusGone:
		return ErrExpired
	case http.StatusConflict:
		return ErrConsumed
	default:
		message, _ := io.ReadAll(io.LimitReader(response.Body, 4<<10))
		return fmt.Errorf("server returned %s: %s", response.Status, strings.TrimSpace(string(message)))
	}
}

func isLoopbackHTTP(parsed *url.URL) bool {
	host := parsed.Hostname()
	return parsed.Scheme == "http" && (host == "127.0.0.1" || host == "localhost" || host == "::1")
}
