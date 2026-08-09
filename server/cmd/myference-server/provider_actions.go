package main

import (
	"context"
	"errors"
	"math/big"
	"slices"
	"sort"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/kunalshah017/myference/server/internal/api"
	"github.com/kunalshah017/myference/server/internal/store"
)

type providerActionRepository interface {
	MachineBelongsToAccount(context.Context, string, string) (bool, error)
	ProviderActionState(context.Context, string, store.ProviderAccountConfig, []store.ProviderOfferQuery) (store.ProviderActionState, error)
	ProviderAccount(context.Context, string, store.ProviderAccountConfig) (store.ProviderAccount, error)
}

func providerOfferQueries(offers []api.ProviderActionOffer) []store.ProviderOfferQuery {
	result := make([]store.ProviderOfferQuery, 0, len(offers))
	for _, offer := range offers {
		capabilities := append([]string(nil), offer.Capabilities...)
		sort.Strings(capabilities)
		result = append(result, store.ProviderOfferQuery{
			OfferID: offer.OfferID, OfferHash: crypto.Keccak256Hash([]byte(offer.OfferID)).Hex(),
			ModelHash: crypto.Keccak256Hash([]byte(offer.Model)).Hex(), CapabilityHash: crypto.Keccak256Hash([]byte(strings.Join(capabilities, ","))).Hex(),
			InputPerMillionWei: offer.InputPerMillionWei, OutputPerMillionWei: offer.OutputPerMillionWei, ComputePerSecondWei: offer.ComputePerSecondWei,
		})
	}
	return result
}

func prepareProviderAction(ctx context.Context, repository providerActionRepository, config store.ProviderAccountConfig, source, machineID, accountID string, input api.ProviderActionInput) (string, api.ProviderActionBaseline, error) {
	if source == api.ActionSourceMachine {
		owned, err := repository.MachineBelongsToAccount(ctx, machineID, accountID)
		if err != nil || !owned {
			return "", api.ProviderActionBaseline{}, errors.New("machine does not belong to this account")
		}
	}
	queries := providerOfferQueries(input.Offers)
	state, err := repository.ProviderActionState(ctx, accountID, config, queries)
	if err != nil {
		return "", api.ProviderActionBaseline{}, err
	}
	if source == api.ActionSourceBrowser && input.Kind == api.ActionPublishOffer {
		account, err := repository.ProviderAccount(ctx, accountID, config)
		if err != nil {
			return "", api.ProviderActionBaseline{}, err
		}
		for _, requested := range input.Offers {
			index := slices.IndexFunc(account.Offers, func(existing store.EditableOffer) bool { return existing.OfferID == requested.OfferID })
			if index < 0 || !sameOfferIdentity(account.Offers[index], requested) {
				return "", api.ProviderActionBaseline{}, errors.New("web pricing may only update an existing offer identity")
			}
		}
	}
	if err := validateProviderActionState(input, state, config.MinimumBondWei); err != nil {
		return "", api.ProviderActionBaseline{}, err
	}
	return state.WalletAddress, api.ProviderActionBaseline{BondWei: state.BondWei, ClaimableWei: state.ClaimableWei, ExitAvailableAt: state.ExitAvailableAt, Versions: state.LatestVersions}, nil
}

func verifyProviderAction(ctx context.Context, repository providerActionRepository, config store.ProviderAccountConfig, action api.ProviderAction) (map[string]uint64, bool, error) {
	state, err := repository.ProviderActionState(ctx, action.AccountID(), config, providerOfferQueries(action.Offers))
	if err != nil {
		return nil, false, err
	}
	versions, confirmed := providerActionConfirmed(action, state)
	return versions, confirmed, nil
}

func providerActionConfirmed(action api.ProviderAction, state store.ProviderActionState) (map[string]uint64, bool) {
	baseline := action.Baseline()
	switch action.Kind {
	case api.ActionPublishOffer:
		versions := make(map[string]uint64, len(action.Offers))
		for _, offer := range action.Offers {
			version := state.MatchingVersions[offer.OfferID]
			if version == 0 || version <= baseline.Versions[offer.OfferID] {
				return nil, false
			}
			versions[offer.OfferID] = version
		}
		return versions, true
	case api.ActionDepositCollateral:
		before, amount, current := decimalBig(baseline.BondWei), decimalBig(action.AmountWei), decimalBig(state.BondWei)
		return nil, before != nil && amount != nil && current != nil && current.Cmp(new(big.Int).Add(before, amount)) >= 0
	case api.ActionRequestCollateralExit:
		return nil, baseline.ExitAvailableAt == 0 && state.ExitAvailableAt > 0
	case api.ActionFinalizeCollateralExit:
		return nil, baseline.ExitAvailableAt > 0 && state.ExitAvailableAt == 0 && state.BondWei == "0"
	default:
		return nil, false
	}
}

func sameOfferIdentity(existing store.EditableOffer, requested api.ProviderActionOffer) bool {
	left, right := append([]string(nil), existing.Capabilities...), append([]string(nil), requested.Capabilities...)
	sort.Strings(left)
	sort.Strings(right)
	return existing.Model == requested.Model && existing.BackendKind == requested.Kind && existing.MeteringMode == requested.MeteringMode && slices.Equal(left, right)
}

func validateProviderActionState(input api.ProviderActionInput, state store.ProviderActionState, minimumBond string) error {
	switch input.Kind {
	case api.ActionPublishOffer:
		bond, minimum := decimalBig(state.BondWei), decimalBig(minimumBond)
		if bond == nil || minimum == nil || bond.Cmp(minimum) < 0 || state.ExitAvailableAt != 0 {
			return errors.New("provider collateral is insufficient or exiting")
		}
	case api.ActionDepositCollateral:
		if state.ExitAvailableAt != 0 {
			return errors.New("collateral cannot be deposited during an exit")
		}
	case api.ActionRequestCollateralExit:
		if state.BondWei == "0" || state.ExitAvailableAt != 0 {
			return errors.New("collateral exit is unavailable")
		}
	case api.ActionFinalizeCollateralExit:
		if state.ExitAvailableAt == 0 {
			return errors.New("collateral exit is not ready")
		}
	}
	return nil
}

func decimalBig(value string) *big.Int {
	result, ok := new(big.Int).SetString(value, 10)
	if !ok || result.Sign() < 0 {
		return nil
	}
	return result
}
