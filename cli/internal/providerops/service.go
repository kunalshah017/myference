package providerops

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/kunalshah017/myference/cli/internal/account"
	"github.com/kunalshah017/myference/cli/internal/config"
)

var weiPerMON = new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
var maxUint256 = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))

type ProviderAPI interface {
	ProviderAccount(context.Context, string) (account.ProviderAccount, error)
	CreateProviderAction(context.Context, string, account.ProviderActionInput) (account.ProviderAction, error)
	ProviderAction(context.Context, string, string) (account.ProviderAction, error)
	MachineOfferVersions(context.Context, string, string) (map[string]uint64, error)
}

type Rates struct {
	InputPerMillionMON  string
	OutputPerMillionMON string
	ComputePerSecondMON string
}

type Service struct {
	API        ProviderAPI
	Token      string
	MachineID  string
	WebURL     string
	LoadConfig func() (config.Config, error)
	SaveConfig func(config.Config) error
	OpenURL    func(string) error
	Wait       func(context.Context, time.Duration) error
}

type ApprovalRequiredError struct{ URL string }

func (e *ApprovalRequiredError) Error() string {
	return "open this URL to approve the wallet transaction: " + e.URL
}

func ParseMON(value string) (string, error) {
	if value == "" || strings.TrimSpace(value) != value || strings.HasPrefix(value, "-") || strings.ContainsAny(value, "eE+") {
		return "", errors.New("MON amount must be a non-negative decimal with at most 18 fractional digits")
	}
	parts := strings.Split(value, ".")
	if len(parts) > 2 || parts[0] == "" || !decimalDigits(parts[0]) {
		return "", errors.New("MON amount must be a non-negative decimal with at most 18 fractional digits")
	}
	fraction := ""
	if len(parts) == 2 {
		fraction = parts[1]
		if fraction == "" || len(fraction) > 18 || !decimalDigits(fraction) {
			return "", errors.New("MON amount must be a non-negative decimal with at most 18 fractional digits")
		}
	}
	whole, ok := new(big.Int).SetString(parts[0], 10)
	if !ok {
		return "", errors.New("invalid MON amount")
	}
	wei := new(big.Int).Mul(whole, weiPerMON)
	if fraction != "" {
		padded := fraction + strings.Repeat("0", 18-len(fraction))
		fractionWei, _ := new(big.Int).SetString(padded, 10)
		wei.Add(wei, fractionWei)
	}
	if wei.Cmp(maxUint256) > 0 {
		return "", errors.New("MON amount exceeds uint256")
	}
	return wei.String(), nil
}

func decimalDigits(value string) bool {
	for _, character := range value {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}

func (s Service) Publish(ctx context.Context, backend config.Backend, rates Rates) error {
	inputWei, err := ParseMON(rates.InputPerMillionMON)
	if err != nil {
		return fmt.Errorf("input price: %w", err)
	}
	outputWei, err := ParseMON(rates.OutputPerMillionMON)
	if err != nil {
		return fmt.Errorf("output price: %w", err)
	}
	computeWei, err := ParseMON(rates.ComputePerSecondMON)
	if err != nil {
		return fmt.Errorf("compute price: %w", err)
	}
	capabilities, metering := offerShape(backend)
	offerID := backend.EffectiveOfferID()
	offer := account.ProviderOffer{OfferID: offerID, Model: backend.Model, Kind: backend.Kind, Capabilities: capabilities, MeteringMode: metering, InputPerMillionWei: inputWei, OutputPerMillionWei: outputWei, ComputePerSecondWei: computeWei}
	action, err := s.execute(ctx, account.ProviderActionInput{Kind: account.ActionPublishOffer, Offers: []account.ProviderOffer{offer}})
	if err != nil {
		return err
	}
	if s.LoadConfig == nil || s.SaveConfig == nil {
		return errors.New("configuration persistence is unavailable")
	}
	cfg, err := s.LoadConfig()
	if err != nil {
		return err
	}
	for index := range cfg.Backends {
		if cfg.Backends[index].Name == backend.Name {
			version := action.Versions[offerID]
			if version == 0 {
				return fmt.Errorf("confirmed action did not publish offer %q", offerID)
			}
			cfg.Backends[index].PriceVersion = version
			return s.SaveConfig(cfg)
		}
	}
	return fmt.Errorf("backend %q is no longer configured", backend.Name)
}

func Compatible(backend config.Backend, offer account.EditableOffer) bool {
	capabilities, metering := offerShape(backend)
	want := append([]string(nil), capabilities...)
	got := append([]string(nil), offer.Capabilities...)
	slices.Sort(want)
	slices.Sort(got)
	return backend.Model == offer.Model && backend.Kind == offer.BackendKind && slices.Equal(want, got) && metering == offer.MeteringMode && offer.Version > 0
}

func (s Service) Attach(ctx context.Context, backendName, offerID string) error {
	if s.LoadConfig == nil || s.SaveConfig == nil {
		return errors.New("configuration persistence is unavailable")
	}
	cfg, err := s.LoadConfig()
	if err != nil {
		return err
	}
	backendIndex := slices.IndexFunc(cfg.Backends, func(item config.Backend) bool { return item.Name == backendName })
	if backendIndex < 0 {
		return fmt.Errorf("backend %q not found", backendName)
	}
	providerAccount, err := s.Account(ctx)
	if err != nil {
		return err
	}
	offerIndex := slices.IndexFunc(providerAccount.Offers, func(item account.EditableOffer) bool { return item.OfferID == offerID })
	if offerIndex < 0 {
		return fmt.Errorf("wallet offer %q not found", offerID)
	}
	offer := providerAccount.Offers[offerIndex]
	if !Compatible(cfg.Backends[backendIndex], offer) {
		return fmt.Errorf("wallet offer %q is incompatible with backend %q", offerID, backendName)
	}
	cfg.Backends[backendIndex].OfferID = offer.OfferID
	cfg.Backends[backendIndex].PriceVersion = offer.Version
	return s.SaveConfig(cfg)
}

func (s Service) Deposit(ctx context.Context, amountMON string) error {
	amount, err := ParseMON(amountMON)
	if err != nil {
		return err
	}
	if amount == "0" {
		return errors.New("deposit amount must be positive")
	}
	_, err = s.execute(ctx, account.ProviderActionInput{Kind: account.ActionDepositCollateral, AmountWei: amount})
	return err
}

func (s Service) RequestExit(ctx context.Context) error {
	_, err := s.execute(ctx, account.ProviderActionInput{Kind: account.ActionRequestCollateralExit})
	return err
}

func (s Service) FinalizeExit(ctx context.Context) error {
	_, err := s.execute(ctx, account.ProviderActionInput{Kind: account.ActionFinalizeCollateralExit})
	return err
}

func (s Service) Account(ctx context.Context) (account.ProviderAccount, error) {
	if s.API == nil {
		return account.ProviderAccount{}, errors.New("provider API is unavailable")
	}
	return s.API.ProviderAccount(ctx, s.Token)
}

func (s Service) SyncVersions(ctx context.Context) (bool, error) {
	if s.API == nil || s.LoadConfig == nil || s.SaveConfig == nil {
		return false, errors.New("version synchronization is unavailable")
	}
	versions, err := s.API.MachineOfferVersions(ctx, s.Token, s.MachineID)
	if err != nil {
		return false, err
	}
	cfg, err := s.LoadConfig()
	if err != nil {
		return false, err
	}
	changed := false
	for index := range cfg.Backends {
		if version := versions[cfg.Backends[index].EffectiveOfferID()]; version > 0 && version != cfg.Backends[index].PriceVersion {
			cfg.Backends[index].PriceVersion = version
			changed = true
		}
	}
	if changed {
		return true, s.SaveConfig(cfg)
	}
	return false, nil
}

func (s Service) execute(ctx context.Context, input account.ProviderActionInput) (account.ProviderAction, error) {
	if s.API == nil {
		return account.ProviderAction{}, errors.New("provider API is unavailable")
	}
	action, err := s.API.CreateProviderAction(ctx, s.Token, input)
	if err != nil {
		return account.ProviderAction{}, fmt.Errorf("create provider action: %w", err)
	}
	approvalURL, err := approvalURL(s.WebURL, action.ID)
	if err != nil {
		return account.ProviderAction{}, err
	}
	if s.OpenURL == nil || s.OpenURL(approvalURL) != nil {
		return account.ProviderAction{}, &ApprovalRequiredError{URL: approvalURL}
	}
	wait := s.Wait
	if wait == nil {
		wait = waitFor
	}
	for action.Status != account.ActionConfirmed {
		if !action.ExpiresAt.IsZero() && time.Now().After(action.ExpiresAt) {
			return account.ProviderAction{}, errors.New("provider action expired")
		}
		if err := wait(ctx, 2*time.Second); err != nil {
			return account.ProviderAction{}, fmt.Errorf("provider action was not completed: %w", err)
		}
		action, err = s.API.ProviderAction(ctx, s.Token, action.ID)
		if err != nil {
			return account.ProviderAction{}, fmt.Errorf("poll provider action: %w", err)
		}
	}
	return action, nil
}

func approvalURL(webURL, actionID string) (string, error) {
	parsed, err := url.Parse(webURL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "https" && !loopbackHTTP(parsed)) {
		return "", errors.New("web URL must use HTTPS except on loopback")
	}
	parsed.Path = strings.TrimSuffix(parsed.Path, "/") + "/provider/approve"
	query := parsed.Query()
	query.Set("action", actionID)
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func loopbackHTTP(parsed *url.URL) bool {
	host := parsed.Hostname()
	return parsed.Scheme == "http" && (host == "127.0.0.1" || host == "localhost" || host == "::1")
}

func waitFor(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func offerShape(item config.Backend) ([]string, string) {
	commandAgent := item.Kind == "kimi" || ((item.Kind == "codex" || item.Kind == "claude") && item.Image != "")
	if commandAgent {
		return []string{"stream", "text", "workspace"}, "compute_only"
	}
	return []string{"stream", "text"}, "tokens_and_compute"
}
