package store

import (
	"context"
	"strings"

	v1 "github.com/kunalshah017/myference/protocol/v1"
)

func (s *Store) ProviderSignerAllowed(ctx context.Context, chainID uint64, contract string, provider, signer v1.Address) (bool, error) {
	providerAddress, signerAddress := addressHex(provider), addressHex(signer)
	if strings.EqualFold(providerAddress, signerAddress) {
		return true, nil
	}
	var allowed bool
	err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM chain_provider_signers WHERE chain_id=$1 AND lower(contract_address)=lower($2) AND lower(provider)=lower($3) AND lower(signer)=lower($4) AND allowed)`, chainID, contract, providerAddress, signerAddress).Scan(&allowed)
	return allowed, err
}

func addressHex(value v1.Address) string {
	const digits = "0123456789abcdef"
	encoded := make([]byte, 42)
	encoded[0], encoded[1] = '0', 'x'
	for index, item := range value {
		encoded[2+index*2], encoded[3+index*2] = digits[item>>4], digits[item&15]
	}
	return string(encoded)
}
