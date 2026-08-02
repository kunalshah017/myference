package v1

import (
	"crypto/ecdsa"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	bytes32Type     = mustABIType("bytes32")
	addressType     = mustABIType("address")
	uint256Type     = mustABIType("uint256")
	uint64Type      = mustABIType("uint64")
	uint16Type      = mustABIType("uint16")
	uint8Type       = mustABIType("uint8")
	receiptTypeHash = crypto.Keccak256Hash([]byte("Receipt(bytes32 requestId,bytes32 sessionId,address customer,address provider,address settlementSigner,bytes32 offerId,uint64 priceVersion,bytes32 modelHash,bytes32 capabilityHash,uint64 inputTokens,uint64 outputTokens,uint64 computeMilliseconds,uint64 maximumCharge,uint64 totalCharge,uint16 feeBasisPoints,uint64 feeVersion,uint8 status,uint64 completedAt,bytes32 inputHash,bytes32 outputHash,uint64 nonce)"))
	domainTypeHash  = crypto.Keccak256Hash([]byte("EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)"))
)

func ReceiptDigest(receipt Receipt, chainID uint64, contract Address) (Hash, error) {
	if err := receipt.Validate(); err != nil || chainID == 0 || contract.IsZero() {
		return Hash{}, ErrInvalidReceipt
	}
	domainEncoded, err := (abi.Arguments{{Type: bytes32Type}, {Type: bytes32Type}, {Type: bytes32Type}, {Type: uint256Type}, {Type: addressType}}).Pack(
		domainTypeHash, crypto.Keccak256Hash([]byte("MyferenceMarket")), crypto.Keccak256Hash([]byte("1")), new(big.Int).SetUint64(chainID), common.BytesToAddress(contract[:]),
	)
	if err != nil {
		return Hash{}, err
	}
	domain := crypto.Keccak256Hash(domainEncoded)
	encoded, err := (abi.Arguments{
		{Type: bytes32Type}, {Type: bytes32Type}, {Type: bytes32Type}, {Type: addressType}, {Type: addressType}, {Type: addressType}, {Type: bytes32Type}, {Type: uint64Type}, {Type: bytes32Type}, {Type: bytes32Type},
		{Type: uint64Type}, {Type: uint64Type}, {Type: uint64Type}, {Type: uint64Type}, {Type: uint64Type}, {Type: uint16Type}, {Type: uint64Type}, {Type: uint8Type}, {Type: uint64Type}, {Type: bytes32Type}, {Type: bytes32Type}, {Type: uint64Type},
	}).Pack(
		receiptTypeHash, receipt.RequestID, receipt.SessionID, common.BytesToAddress(receipt.Customer[:]), common.BytesToAddress(receipt.Provider[:]), common.BytesToAddress(receipt.SettlementSigner[:]), receipt.OfferID, receipt.PriceVersion, receipt.ModelHash, receipt.CapabilityHash,
		receipt.InputTokens, receipt.OutputTokens, receipt.ComputeMilliseconds, receipt.MaximumCharge, receipt.TotalCharge, receipt.FeeBasisPoints, receipt.FeeVersion, uint8(receipt.Status), receipt.CompletedAt, receipt.InputHash, receipt.OutputHash, receipt.Nonce,
	)
	if err != nil {
		return Hash{}, err
	}
	structHash := crypto.Keccak256Hash(encoded)
	return Hash(crypto.Keccak256Hash([]byte{0x19, 0x01}, domain[:], structHash[:])), nil
}

func SignReceipt(receipt Receipt, chainID uint64, contract Address, key *ecdsa.PrivateKey) ([]byte, error) {
	if key == nil {
		return nil, errors.New("receipt signer key is required")
	}
	digest, err := ReceiptDigest(receipt, chainID, contract)
	if err != nil {
		return nil, err
	}
	signature, err := crypto.Sign(digest[:], key)
	if err == nil {
		signature[64] += 27
	}
	return signature, err
}

func RecoverReceiptSigner(receipt Receipt, chainID uint64, contract Address, signature []byte) (Address, error) {
	if len(signature) != crypto.SignatureLength {
		return Address{}, ErrInvalidReceipt
	}
	digest, err := ReceiptDigest(receipt, chainID, contract)
	if err != nil {
		return Address{}, err
	}
	normalized := append([]byte(nil), signature...)
	if normalized[64] >= 27 {
		normalized[64] -= 27
	}
	publicKey, err := crypto.SigToPub(digest[:], normalized)
	if err != nil {
		return Address{}, err
	}
	var signer Address
	copy(signer[:], crypto.PubkeyToAddress(*publicKey).Bytes())
	return signer, nil
}

func mustABIType(name string) abi.Type {
	value, err := abi.NewType(name, "", nil)
	if err != nil {
		panic(err)
	}
	return value
}
