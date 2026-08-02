// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package bindings

import (
	"errors"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
	_ = abi.ConvertType
)

// MyferenceMarketReceipt is an auto generated low-level Go binding around an user-defined struct.
type MyferenceMarketReceipt struct {
	RequestId           [32]byte
	SessionId           [32]byte
	Customer            common.Address
	Provider            common.Address
	SettlementSigner    common.Address
	OfferId             [32]byte
	PriceVersion        uint64
	ModelHash           [32]byte
	CapabilityHash      [32]byte
	InputTokens         uint64
	OutputTokens        uint64
	ComputeMilliseconds uint64
	MaximumCharge       uint64
	TotalCharge         uint64
	FeeBasisPoints      uint16
	FeeVersion          uint64
	Status              uint8
	CompletedAt         uint64
	InputHash           [32]byte
	OutputHash          [32]byte
	Nonce               uint64
}

// MyferenceMarketMetaData contains all meta data concerning the MyferenceMarket contract.
var MyferenceMarketMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"constructor\",\"inputs\":[{\"name\":\"initialOwner\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"feeRecipient_\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"settlementSigner_\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"minimumBond_\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"bondExitDelay_\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"feeDelay_\",\"type\":\"uint64\",\"internalType\":\"uint64\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"MAXIMUM_FEE_BASIS_POINTS\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint16\",\"internalType\":\"uint16\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"RECEIPT_TYPEHASH\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"SESSION_CLOSE_DELAY\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint64\",\"internalType\":\"uint64\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"acceptOwnership\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"bondExitAvailableAt\",\"inputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint64\",\"internalType\":\"uint64\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"bondExitDelay\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint64\",\"internalType\":\"uint64\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"claim\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"claimable\",\"inputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"customerBalances\",\"inputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"deposit\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"depositBond\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"eip712Domain\",\"inputs\":[],\"outputs\":[{\"name\":\"fields\",\"type\":\"bytes1\",\"internalType\":\"bytes1\"},{\"name\":\"name\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"version\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"chainId\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"verifyingContract\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"salt\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"extensions\",\"type\":\"uint256[]\",\"internalType\":\"uint256[]\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"executeFeeChange\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"feeBasisPoints\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint16\",\"internalType\":\"uint16\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"feeBpsByVersion\",\"inputs\":[{\"name\":\"\",\"type\":\"uint64\",\"internalType\":\"uint64\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint16\",\"internalType\":\"uint16\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"feeDelay\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint64\",\"internalType\":\"uint64\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"feeRecipient\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"feeVersion\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint64\",\"internalType\":\"uint64\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"finalizeBondExit\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"finalizeSessionClose\",\"inputs\":[{\"name\":\"sessionId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"hashReceipt\",\"inputs\":[{\"name\":\"receipt\",\"type\":\"tuple\",\"internalType\":\"structMyferenceMarket.Receipt\",\"components\":[{\"name\":\"requestId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"sessionId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"customer\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"provider\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"settlementSigner\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"offerId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"priceVersion\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"modelHash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"capabilityHash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"inputTokens\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"outputTokens\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"computeMilliseconds\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"maximumCharge\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"totalCharge\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"feeBasisPoints\",\"type\":\"uint16\",\"internalType\":\"uint16\"},{\"name\":\"feeVersion\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"status\",\"type\":\"uint8\",\"internalType\":\"uint8\"},{\"name\":\"completedAt\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"inputHash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"outputHash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"nonce\",\"type\":\"uint64\",\"internalType\":\"uint64\"}]}],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"latestOfferVersion\",\"inputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint64\",\"internalType\":\"uint64\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"minimumBond\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"offerVersions\",\"inputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"\",\"type\":\"uint64\",\"internalType\":\"uint64\"}],\"outputs\":[{\"name\":\"active\",\"type\":\"bool\",\"internalType\":\"bool\"},{\"name\":\"modelHash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"capabilityHash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"version\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"inputPerMillion\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"outputPerMillion\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"computePerSecond\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"openSession\",\"inputs\":[{\"name\":\"sessionId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"allowance\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"expiresAt\",\"type\":\"uint64\",\"internalType\":\"uint64\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"owner\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"pause\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"paused\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"pendingFeeAvailableAt\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint64\",\"internalType\":\"uint64\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"pendingFeeBasisPoints\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint16\",\"internalType\":\"uint16\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"pendingFeeChange\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"pendingOwner\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"proposeFee\",\"inputs\":[{\"name\":\"newFeeBasisPoints\",\"type\":\"uint16\",\"internalType\":\"uint16\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"providerBonds\",\"inputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"publishOffer\",\"inputs\":[{\"name\":\"offerId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"modelHash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"capabilityHash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"inputPerMillion\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"outputPerMillion\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"computePerSecond\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"renounceOwnership\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"requestBondExit\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"requestSessionClose\",\"inputs\":[{\"name\":\"sessionId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"requestWithdrawal\",\"inputs\":[{\"name\":\"amount\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"sessions\",\"inputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"customer\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"allowance\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"spent\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"expiresAt\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"closeAvailableAt\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"finalized\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"settleReceipt\",\"inputs\":[{\"name\":\"receipt\",\"type\":\"tuple\",\"internalType\":\"structMyferenceMarket.Receipt\",\"components\":[{\"name\":\"requestId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"sessionId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"customer\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"provider\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"settlementSigner\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"offerId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"priceVersion\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"modelHash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"capabilityHash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"inputTokens\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"outputTokens\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"computeMilliseconds\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"maximumCharge\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"totalCharge\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"feeBasisPoints\",\"type\":\"uint16\",\"internalType\":\"uint16\"},{\"name\":\"feeVersion\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"status\",\"type\":\"uint8\",\"internalType\":\"uint8\"},{\"name\":\"completedAt\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"inputHash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"outputHash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"nonce\",\"type\":\"uint64\",\"internalType\":\"uint64\"}]},{\"name\":\"providerSignature\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"settlementSignature\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"settleReceipts\",\"inputs\":[{\"name\":\"receipts\",\"type\":\"tuple[]\",\"internalType\":\"structMyferenceMarket.Receipt[]\",\"components\":[{\"name\":\"requestId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"sessionId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"customer\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"provider\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"settlementSigner\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"offerId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"priceVersion\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"modelHash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"capabilityHash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"inputTokens\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"outputTokens\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"computeMilliseconds\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"maximumCharge\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"totalCharge\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"feeBasisPoints\",\"type\":\"uint16\",\"internalType\":\"uint16\"},{\"name\":\"feeVersion\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"status\",\"type\":\"uint8\",\"internalType\":\"uint8\"},{\"name\":\"completedAt\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"inputHash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"outputHash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"nonce\",\"type\":\"uint64\",\"internalType\":\"uint64\"}]},{\"name\":\"providerSignatures\",\"type\":\"bytes[]\",\"internalType\":\"bytes[]\"},{\"name\":\"settlementSignatures\",\"type\":\"bytes[]\",\"internalType\":\"bytes[]\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"settledRequests\",\"inputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"settlementSigner\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"slashDoubleSign\",\"inputs\":[{\"name\":\"first\",\"type\":\"tuple\",\"internalType\":\"structMyferenceMarket.Receipt\",\"components\":[{\"name\":\"requestId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"sessionId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"customer\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"provider\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"settlementSigner\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"offerId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"priceVersion\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"modelHash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"capabilityHash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"inputTokens\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"outputTokens\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"computeMilliseconds\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"maximumCharge\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"totalCharge\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"feeBasisPoints\",\"type\":\"uint16\",\"internalType\":\"uint16\"},{\"name\":\"feeVersion\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"status\",\"type\":\"uint8\",\"internalType\":\"uint8\"},{\"name\":\"completedAt\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"inputHash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"outputHash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"nonce\",\"type\":\"uint64\",\"internalType\":\"uint64\"}]},{\"name\":\"firstSignature\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"second\",\"type\":\"tuple\",\"internalType\":\"structMyferenceMarket.Receipt\",\"components\":[{\"name\":\"requestId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"sessionId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"customer\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"provider\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"settlementSigner\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"offerId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"priceVersion\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"modelHash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"capabilityHash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"inputTokens\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"outputTokens\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"computeMilliseconds\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"maximumCharge\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"totalCharge\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"feeBasisPoints\",\"type\":\"uint16\",\"internalType\":\"uint16\"},{\"name\":\"feeVersion\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"status\",\"type\":\"uint8\",\"internalType\":\"uint8\"},{\"name\":\"completedAt\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"inputHash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"outputHash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"nonce\",\"type\":\"uint64\",\"internalType\":\"uint64\"}]},{\"name\":\"secondSignature\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"slashedRequests\",\"inputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"totalClaimable\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"totalCustomerBalances\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"totalLockedSessions\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"totalProviderBonds\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"transferOwnership\",\"inputs\":[{\"name\":\"newOwner\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"unpause\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"usedNonces\",\"inputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"\",\"type\":\"uint64\",\"internalType\":\"uint64\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"event\",\"name\":\"BondDeposited\",\"inputs\":[{\"name\":\"provider\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"amount\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"totalBond\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"BondExitFinalized\",\"inputs\":[{\"name\":\"provider\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"amount\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"BondExitRequested\",\"inputs\":[{\"name\":\"provider\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"availableAt\",\"type\":\"uint64\",\"indexed\":false,\"internalType\":\"uint64\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"Claimed\",\"inputs\":[{\"name\":\"account\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"amount\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"Deposited\",\"inputs\":[{\"name\":\"customer\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"amount\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"EIP712DomainChanged\",\"inputs\":[],\"anonymous\":false},{\"type\":\"event\",\"name\":\"FeeChanged\",\"inputs\":[{\"name\":\"feeBasisPoints\",\"type\":\"uint16\",\"indexed\":false,\"internalType\":\"uint16\"},{\"name\":\"version\",\"type\":\"uint64\",\"indexed\":false,\"internalType\":\"uint64\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"FeeProposed\",\"inputs\":[{\"name\":\"feeBasisPoints\",\"type\":\"uint16\",\"indexed\":false,\"internalType\":\"uint16\"},{\"name\":\"availableAt\",\"type\":\"uint64\",\"indexed\":false,\"internalType\":\"uint64\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"OfferPublished\",\"inputs\":[{\"name\":\"provider\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"offerId\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"version\",\"type\":\"uint64\",\"indexed\":true,\"internalType\":\"uint64\"},{\"name\":\"modelHash\",\"type\":\"bytes32\",\"indexed\":false,\"internalType\":\"bytes32\"},{\"name\":\"capabilityHash\",\"type\":\"bytes32\",\"indexed\":false,\"internalType\":\"bytes32\"},{\"name\":\"inputPerMillion\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"outputPerMillion\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"computePerSecond\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"OwnershipTransferStarted\",\"inputs\":[{\"name\":\"previousOwner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"newOwner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"OwnershipTransferred\",\"inputs\":[{\"name\":\"previousOwner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"newOwner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"Paused\",\"inputs\":[{\"name\":\"account\",\"type\":\"address\",\"indexed\":false,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"ProviderSlashed\",\"inputs\":[{\"name\":\"provider\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"requestId\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"amount\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"ReceiptSettled\",\"inputs\":[{\"name\":\"requestId\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"sessionId\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"provider\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"providerAmount\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"feeAmount\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"SessionCloseRequested\",\"inputs\":[{\"name\":\"sessionId\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"availableAt\",\"type\":\"uint64\",\"indexed\":false,\"internalType\":\"uint64\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"SessionClosed\",\"inputs\":[{\"name\":\"sessionId\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"returnedAmount\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"SessionOpened\",\"inputs\":[{\"name\":\"sessionId\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"customer\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"allowance\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"expiresAt\",\"type\":\"uint64\",\"indexed\":false,\"internalType\":\"uint64\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"Unpaused\",\"inputs\":[{\"name\":\"account\",\"type\":\"address\",\"indexed\":false,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"WithdrawalRequested\",\"inputs\":[{\"name\":\"account\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"amount\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"error\",\"name\":\"BondExitActive\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"ECDSAInvalidSignature\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"ECDSAInvalidSignatureLength\",\"inputs\":[{\"name\":\"length\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"type\":\"error\",\"name\":\"ECDSAInvalidSignatureS\",\"inputs\":[{\"name\":\"s\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}]},{\"type\":\"error\",\"name\":\"EnforcedPause\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"EvidenceAlreadyUsed\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"ExitDelayActive\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"ExpectedPause\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"FeeDelayActive\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"FeeTooHigh\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"InsufficientBalance\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"InsufficientBond\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"InvalidBatch\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"InvalidEvidence\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"InvalidReceipt\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"InvalidReceiptSignature\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"InvalidSession\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"InvalidShortString\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"MaximumChargeExceeded\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"NoPendingFee\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"NonceAlreadyUsed\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"NothingToClaim\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"OwnableInvalidOwner\",\"inputs\":[{\"name\":\"owner\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"type\":\"error\",\"name\":\"OwnableUnauthorizedAccount\",\"inputs\":[{\"name\":\"account\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"type\":\"error\",\"name\":\"ReentrancyGuardReentrantCall\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"RequestAlreadySettled\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"SessionAllowanceExceeded\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"SessionAlreadyExists\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"SessionCloseDelayActive\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"SessionExpired\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"StaleOffer\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"StringTooLong\",\"inputs\":[{\"name\":\"str\",\"type\":\"string\",\"internalType\":\"string\"}]},{\"type\":\"error\",\"name\":\"TransferFailed\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"ZeroAmount\",\"inputs\":[]}]",
	Bin: "0x610200604052600f80546001600160501b031916620101f4179055348015610025575f5ffd5b50604051613cdf380380613cdf833981016040819052610044916102d0565b604080518082018252600f81526e135e59995c995b98d953585c9ad95d608a1b602080830191909152825180840190935260018352603160f81b9083015290876001600160a01b0381166100b257604051631e4fbdf760e01b81525f60048201526024015b60405180910390fd5b6100bb816101f7565b5060017f9b779b17422d0df92223018b32b4d1fa46e071723d6817e2486d003becc55f00556100e982610213565b610120526100f681610213565b61014052815160208084019190912060e052815190820120610100524660a05261018260e05161010051604080517f8b73c3c69bb8fe3d512ecc4cf759cc79239f7b179b0ffacaa9a75d522b39400f60208201529081019290925260608201524660808201523060a08201525f9060c00160405160208183030381529060405280519060200120905090565b60805250503060c0526001600160a01b039485166101605292909316610180526101a0526001600160401b039182166101c052166101e0525060015f52600e6020527fa7c5ba7114a813b50159add3a36832908dc83db71d0b9a24c2ad0f83be958207805461ffff19166101f4179055610394565b600180546001600160a01b031916905561021081610250565b50565b5f5f829050601f8151111561023d578260405163305a27a960e01b81526004016100a99190610339565b80516102488261036e565b179392505050565b5f80546001600160a01b038381166001600160a01b0319831681178455604051919092169283917f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e09190a35050565b80516001600160a01b03811681146102b5575f5ffd5b919050565b80516001600160401b03811681146102b5575f5ffd5b5f5f5f5f5f5f60c087890312156102e5575f5ffd5b6102ee8761029f565b95506102fc6020880161029f565b945061030a6040880161029f565b93506060870151925061031f608088016102ba565b915061032d60a088016102ba565b90509295509295509295565b602081525f82518060208401528060208501604085015e5f604082850101526040601f19601f83011684010191505092915050565b8051602080830151919081101561038e575f198160200360031b1b821691505b50919050565b60805160a05160c05160e05161010051610120516101405161016051610180516101a0516101c0516101e05161389061044f5f395f818161077c01526119ce01525f818161034301526110d001525f81816109e3015281816117b20152611be701525f8181610a4f015281816122e0015261271701525f81816104ba015281816118fc01526128be01525f6121df01525f6121af01525f612a6701525f612a3f01525f61299a01525f6129c401525f6129ee01526138905ff3fe60806040526004361061030d575f3560e01c8063741b3c39116101a3578063a7abbc3c116100f2578063dc4638db11610092578063ebd9ce281161006d578063ebd9ce2814610b08578063ee754f9114610b27578063f2fde38b14610b3c578063f79c6ea414610b5b575f5ffd5b8063dc4638db14610aa7578063e30c397814610acc578063e6dd079114610ae9575f5ffd5b8063bc651f6c116100cd578063bc651f6c14610a1f578063c46914d814610a3e578063c691271b14610a71578063d0e30db014610a9f575f5ffd5b8063a7abbc3c146109b3578063aa7517e1146109d2578063b8606eef14610a05575f5ffd5b80638456cb591161015d57806395850f591161013857806395850f59146109235780639d9fabc8146109565780639ee679e814610975578063a146e24b14610994575f5ffd5b80638456cb59146108cc57806384b0196e146108e05780638da5cb5b14610907575f5ffd5b8063741b3c3914610763578063777ebe3d1461076b578063791d66a11461079e57806379ba5097146107c95780637dbd2832146107dd5780637eff8a4e1461088d575f5ffd5b8063558bc5e41161025f5780636527a8061161021957806369115cd7116101f457806369115cd7146107105780636a0755f9146107245780636a0b30781461073a578063715018a61461074f575f5ffd5b80636527a8061461060c5780636554018a146106c05780636767d653146106e1575f5ffd5b8063558bc5e414610551578063576e748d1461057f57806357950440146105945780635a73384b146105ba5780635c975abb146105ce5780635dc6aec6146105ec575f5ffd5b80633f4ba83a116102ca57806346904840116102a557806346904840146104a95780634838ed19146104f45780634e71d92d1461050957806355512c1f1461051d575f5ffd5b80633f4ba83a1461044b5780633fb3adba1461045f578063402914f51461047e575f5ffd5b8063131620fc1461031157806316ddad3314610332578063191708b7146103825780631fb545d3146103af57806333a87bb1146103d75780633a31201e14610420575b5f5ffd5b34801561031c575f5ffd5b5061033061032b366004612ef7565b610b6f565b005b34801561033d575f5ffd5b506103657f000000000000000000000000000000000000000000000000000000000000000081565b6040516001600160401b0390911681526020015b60405180910390f35b34801561038d575f5ffd5b506103a161039c366004612f95565b610cc8565b604051908152602001610379565b3480156103ba575f5ffd5b506103c46105dc81565b60405161ffff9091168152602001610379565b3480156103e2575f5ffd5b506104106103f13660046130fc565b600c60209081525f928352604080842090915290825290205460ff1681565b6040519015158152602001610379565b34801561042b575f5ffd5b506103a161043a36600461312d565b60066020525f908152604090205481565b348015610456575f5ffd5b50610330610d20565b34801561046a575f5ffd5b50610330610479366004613146565b610d32565b348015610489575f5ffd5b506103a161049836600461312d565b60056020525f908152604090205481565b3480156104b4575f5ffd5b506104dc7f000000000000000000000000000000000000000000000000000000000000000081565b6040516001600160a01b039091168152602001610379565b3480156104ff575f5ffd5b506103a160135481565b348015610514575f5ffd5b50610330610f40565b348015610528575f5ffd5b5061036561053736600461312d565b60076020525f90815260409020546001600160401b031681565b34801561055c575f5ffd5b5061041061056b366004612ef7565b600d6020525f908152604090205460ff1681565b34801561058a575f5ffd5b506103a160125481565b34801561059f575f5ffd5b50600f5461036590600160601b90046001600160401b031681565b3480156105c5575f5ffd5b50610330611066565b3480156105d9575f5ffd5b50600154600160a01b900460ff16610410565b3480156105f7575f5ffd5b50600f5461041090600160a01b900460ff1681565b348015610617575f5ffd5b50610681610626366004613178565b600960209081525f9384526040808520825292845282842090528252902080546001820154600283015460038401546004850154600586015460069096015460ff90951695939492936001600160401b039092169290919087565b6040805197151588526020880196909652948601939093526001600160401b039091166060850152608084015260a083015260c082015260e001610379565b3480156106cb575f5ffd5b50600f546103c490600160501b900461ffff1681565b3480156106ec575f5ffd5b506103c46106fb3660046131a8565b600e6020525f908152604090205461ffff1681565b34801561071b575f5ffd5b5061033061115d565b34801561072f575f5ffd5b506103656201518081565b348015610745575f5ffd5b506103a160115481565b34801561075a575f5ffd5b50610330611231565b610330611242565b348015610776575f5ffd5b506103657f000000000000000000000000000000000000000000000000000000000000000081565b3480156107a9575f5ffd5b506103a16107b836600461312d565b60046020525f908152604090205481565b3480156107d4575f5ffd5b50610330611325565b3480156107e8575f5ffd5b506108486107f7366004612ef7565b600a6020525f908152604090208054600182015460028301546003909301546001600160a01b039092169290916001600160401b0380821691600160401b810490911690600160801b900460ff1686565b604080516001600160a01b0390971687526020870195909552938501929092526001600160401b039081166060850152166080830152151560a082015260c001610379565b348015610898575f5ffd5b506103656108a73660046131c1565b600860209081525f92835260408084209091529082529020546001600160401b031681565b3480156108d7575f5ffd5b5061033061136e565b3480156108eb575f5ffd5b506108f461137e565b6040516103799796959493929190613217565b348015610912575f5ffd5b505f546001600160a01b03166104dc565b34801561092e575f5ffd5b506103a17f4fbe959b1ecaf3e2d7f4625bda452a4d6f1cb88a669414bee1719cda7b618f0581565b348015610961575f5ffd5b50610330610970366004612ef7565b6113c0565b348015610980575f5ffd5b5061033061098f366004612ef7565b6114b7565b34801561099f575f5ffd5b506103306109ae366004613308565b61157e565b3480156109be575f5ffd5b506103306109cd36600461338a565b61159a565b3480156109dd575f5ffd5b506103a17f000000000000000000000000000000000000000000000000000000000000000081565b348015610a10575f5ffd5b50600f546103c49061ffff1681565b348015610a2a575f5ffd5b50610330610a3936600461341f565b611980565b348015610a49575f5ffd5b506104dc7f000000000000000000000000000000000000000000000000000000000000000081565b348015610a7c575f5ffd5b50610410610a8b366004612ef7565b600b6020525f908152604090205460ff1681565b610330611a77565b348015610ab2575f5ffd5b50600f54610365906201000090046001600160401b031681565b348015610ad7575f5ffd5b506001546001600160a01b03166104dc565b348015610af4575f5ffd5b50610330610b03366004613478565b611b0c565b348015610b13575f5ffd5b50610330610b22366004613531565b611bce565b348015610b32575f5ffd5b506103a160105481565b348015610b47575f5ffd5b50610330610b5636600461312d565b611e57565b348015610b66575f5ffd5b50610330611ec7565b5f818152600a6020526040902080546001600160a01b031633141580610ba057506003810154600160801b900460ff165b15610bbe576040516316f78d3b60e11b815260040160405180910390fd5b6003810154600160401b90046001600160401b03161580610bf257506003810154600160401b90046001600160401b031642105b15610c10576040516336aa2eed60e21b815260040160405180910390fd5b60038101805460ff60801b1916600160801b179055600281015460018201545f91610c3a91613584565b90508060125f828254610c4d9190613584565b9091555050335f9081526004602052604081208054839290610c70908490613597565b925050819055508060105f828254610c889190613597565b909155505060405181815283907f1d8ada2ed73dcd9599105b1206ea1d3aa9e687b4e8f3b49940a4b03b6360585e906020015b60405180910390a2505050565b5f610d1a7f4fbe959b1ecaf3e2d7f4625bda452a4d6f1cb88a669414bee1719cda7b618f0583604051602001610cff9291906135aa565b60405160208183030381529060405280519060200120612003565b92915050565b610d2861202f565b610d3061205b565b565b610d3a6120ab565b821580610d45575081155b80610d59575042816001600160401b031611155b15610d77576040516316f78d3b60e11b815260040160405180910390fd5b5f838152600a60205260409020546001600160a01b031615610dac576040516323792a1160e11b815260040160405180910390fd5b335f90815260046020526040902054821115610ddb57604051631e9acf1760e31b815260040160405180910390fd5b335f9081526004602052604081208054849290610df9908490613584565b925050819055508160105f828254610e119190613584565b925050819055508160125f828254610e299190613597565b90915550506040805160c0810182523380825260208083018681525f8486018181526001600160401b03808916606088019081526080880184815260a089018581528d8652600a90975293899020975188546001600160a01b03919091166001600160a01b03199091161788559351600188015590516002870155915160039095018054915193511515600160801b0260ff60801b19948416600160401b026fffffffffffffffffffffffffffffffff199093169690931695909517179190911617909155905184907f88472a4d077942c1db7795cd75d79255367dc41ba9337b21598c772438befc8e90610f3390869086909182526001600160401b0316602082015260400190565b60405180910390a3505050565b610f486120d6565b335f9081526005602052604081205490819003610f78576040516312d37ee560e31b815260040160405180910390fd5b335f90815260056020526040812081905560138054839290610f9b908490613584565b90915550506040515f90339083908381818185875af1925050503d805f8114610fdf576040519150601f19603f3d011682016040523d82523d5f602084013e610fe4565b606091505b5050905080611006576040516312171d8360e31b815260040160405180910390fd5b60405182815233907fd8138f8a3f377c5259ca548e70e4c2de94f129f5a11036a15b69513cba2b426a9060200160405180910390a25050610d3060017f9b779b17422d0df92223018b32b4d1fa46e071723d6817e2486d003becc55f0055565b335f9081526006602052604081205490036110945760405163e92c469f60e01b815260040160405180910390fd5b335f908152600760205260409020546001600160401b0316156110ca5760405163f2352eff60e01b815260040160405180910390fd5b5f6110f57f00000000000000000000000000000000000000000000000000000000000000004261374d565b335f81815260076020908152604091829020805467ffffffffffffffff19166001600160401b038616908117909155915191825292935090917f2d558b0c7f59f19033dbb6e56acc49e254d2b1f528168d83a35a1f5255f6024791015b60405180910390a250565b335f908152600760205260409020546001600160401b031680158061118a5750806001600160401b031642105b156111a85760405163bd7a427b60e01b815260040160405180910390fd5b335f90815260066020908152604080832080549084905560079092528220805467ffffffffffffffff1916905560118054919283926111e8908490613584565b909155506111f890503382612104565b60405181815233907f23c59c6738d789e373cb3f3359278f32ec1227766afbd94405fd92217aaa74dd9060200160405180910390a25050565b61123961202f565b610d305f61214c565b61124a6120ab565b345f0361126a57604051631f2a200560e01b815260040160405180910390fd5b335f908152600760205260409020546001600160401b0316156112a05760405163f2352eff60e01b815260040160405180910390fd5b335f90815260066020526040812080543492906112be908490613597565b925050819055503460115f8282546112d69190613597565b9091555050335f81815260066020908152604091829020548251348152918201527f53709ba3cdfc6888ee4e1f05692566dfc3446e917fd4402db50e328307e6b39391015b60405180910390a2565b60015433906001600160a01b031681146113625760405163118cdaa760e01b81526001600160a01b03821660048201526024015b60405180910390fd5b61136b8161214c565b50565b61137661202f565b610d30612165565b5f6060805f5f5f606061138f6121a8565b6113976121d8565b604080515f80825260208201909252600f60f81b9b939a50919850469750309650945092509050565b5f818152600a6020526040902080546001600160a01b0316331415806113f157506003810154600160801b900460ff165b1561140f576040516316f78d3b60e11b815260040160405180910390fd5b6003810154600160401b90046001600160401b031615611442576040516336aa2eed60e21b815260040160405180910390fd5b5f611450620151804261374d565b6003830180546fffffffffffffffff00000000000000001916600160401b6001600160401b0384169081029190911790915560405190815290915083907f72b43cf8a65c18d2c3cb2c3ebce801281fcc67f6ada51dc7709c95947b443a8190602001610cbb565b805f036114d757604051631f2a200560e01b815260040160405180910390fd5b335f9081526004602052604090205481111561150657604051631e9acf1760e31b815260040160405180910390fd5b335f9081526004602052604081208054839290611524908490613584565b925050819055508060105f82825461153c9190613584565b9091555061154c90503382612104565b60405181815233907fe670e4e82118d22a1f9ee18920455ebc958bae26a90a05d31d3378788b1b0e4490602001611152565b6115866120ab565b6115938585858585612203565b5050505050565b5f6115ad61039c36899003890189612f95565b90505f6115c261039c36879003870187612f95565b9050873515806115d457508735853514155b806115f657505f6115eb60808a0160608b0161312d565b6001600160a01b0316145b80611631575061160c608086016060870161312d565b6001600160a01b031661162560808a0160608b0161312d565b6001600160a01b031614155b8061163b57508082145b156116595760405163325de78f60e21b815260040160405180910390fd5b87355f908152600d602052604090205460ff161561168a5760405163c84c217560e01b815260040160405180910390fd5b61169a6080890160608a0161312d565b6001600160a01b03166116e488888080601f0160208091040260200160405190810160405280939291908181526020018383808284375f9201919091525087939250506129549050565b6001600160a01b031614158061175b57506117056080890160608a0161312d565b6001600160a01b031661174f85858080601f0160208091040260200160405190810160405280939291908181526020018383808284375f9201919091525086939250506129549050565b6001600160a01b031614155b1561177957604051635b37cf6960e11b815260040160405180910390fd5b5f6117d660068261179060808d0160608e0161312d565b6001600160a01b03166001600160a01b031681526020019081526020015f20547f000000000000000000000000000000000000000000000000000000000000000061297c565b9050805f036117f85760405163e92c469f60e01b815260040160405180910390fd5b88355f908152600d60205260408120805460ff19166001179055819060069061182760808d0160608e0161312d565b6001600160a01b03166001600160a01b031681526020019081526020015f205f8282546118549190613584565b925050819055508060115f82825461186c9190613584565b90915550600690505f61188560808c0160608d0161312d565b6001600160a01b03166001600160a01b031681526020019081526020015f20545f036118f7575f6007816118bf60808d0160608e0161312d565b6001600160a01b0316815260208101919091526040015f20805467ffffffffffffffff19166001600160401b03929092169190911790555b6119217f000000000000000000000000000000000000000000000000000000000000000082612104565b883561193360808b0160608c0161312d565b6001600160a01b03167fc0fb837f95a6626fe0d8363a3eddb02605b2d7bb0e3e640a88859a5e9afebc0e8360405161196d91815260200190565b60405180910390a3505050505050505050565b61198861202f565b6105dc61ffff821611156119af5760405163cd4e616760e01b815260040160405180910390fd5b600f805461ffff60501b1916600160501b61ffff8416021790556119f37f00000000000000000000000000000000000000000000000000000000000000004261374d565b600f8054600160a01b68ffffffffffffffffff60601b19909116600160601b6001600160401b03948516810260ff60a01b19169190911791909117918290556040805161ffff861681529190920490921660208301527f99b204b1e2687a443690dc3caa13da2a50fb2668ad5b999f91257f23c75e4310910160405180910390a150565b611a7f6120ab565b345f03611a9f57604051631f2a200560e01b815260040160405180910390fd5b335f9081526004602052604081208054349290611abd908490613597565b925050819055503460105f828254611ad59190613597565b909155505060405134815233907f2da466a7b24304f47e87fa2e1e5a81b9831ce54fec19055ce277ca2f39ba42c49060200161131b565b611b146120ab565b841580611b215750848314155b80611b2c5750848114155b15611b4a576040516333b094a160e01b815260040160405180910390fd5b5f5b85811015611bc557611bbd878783818110611b6957611b6961376c565b90506102a00201868684818110611b8257611b8261376c565b9050602002810190611b949190613780565b868686818110611ba657611ba661376c565b9050602002810190611bb89190613780565b612203565b600101611b4c565b50505050505050565b611bd66120ab565b335f908152600660205260409020547f00000000000000000000000000000000000000000000000000000000000000001180611c285750335f908152600760205260409020546001600160401b031615155b15611c465760405163e92c469f60e01b815260040160405180910390fd5b335f908152600860209081526040808320898452909152812054611c74906001600160401b0316600161374d565b90508060085f336001600160a01b03166001600160a01b031681526020019081526020015f205f8981526020019081526020015f205f6101000a8154816001600160401b0302191690836001600160401b031602179055506040518060e00160405280600115158152602001878152602001868152602001826001600160401b031681526020018581526020018481526020018381525060095f336001600160a01b03166001600160a01b031681526020019081526020015f205f8981526020019081526020015f205f836001600160401b03166001600160401b031681526020019081526020015f205f820151815f015f6101000a81548160ff02191690831515021790555060208201518160010155604082015181600201556060820151816003015f6101000a8154816001600160401b0302191690836001600160401b031602179055506080820151816004015560a0820151816005015560c08201518160060155905050806001600160401b031687336001600160a01b03167f5ae57c6c471f08a53556a92bfe0656a3256d5712e6bb87c07294902c46ee99e38989898989604051611e46959493929190948552602085019390935260408401919091526060830152608082015260a00190565b60405180910390a450505050505050565b611e5f61202f565b600180546001600160a01b0383166001600160a01b03199091168117909155611e8f5f546001600160a01b031690565b6001600160a01b03167f38d16b8cac22d99fc7c124b9cd0de2d3fa1faef420bfe791d8c362d765e2270060405160405180910390a350565b600f54600160a01b900460ff16611ef157604051631e7ecc3960e31b815260040160405180910390fd5b600f54600160601b90046001600160401b0316421015611f245760405163102af6a760e31b815260040160405180910390fd5b600f8054600160501b810461ffff1661ffff1990911617808255600290611f59906201000090046001600160401b03166137c2565b82546101009290920a6001600160401b03818102199093169183160217909155600f80546201000080820484165f908152600e6020908152604091829020805461ffff95861661ffff1990911617905584546affffffffffffffffffffff60501b1981169586905582519416845291909304909316928101929092527f45e84326792e05741cc87e8bafc910ce7a953625da7c055b3b49cbfb1eb5a83091015b60405180910390a1565b5f610d1a61200f61298e565b8360405161190160f01b8152600281019290925260228201526042902090565b5f546001600160a01b03163314610d305760405163118cdaa760e01b8152336004820152602401611359565b612063612ab7565b6001805460ff60a01b191690557f5db9ee0a495bf2e6ff9c91a7834c1ba4fdd244a5e8aa4e537bd38aeae4b073aa335b6040516001600160a01b039091168152602001611ff9565b600154600160a01b900460ff1615610d305760405163d93c066560e01b815260040160405180910390fd5b6120de612ae1565b60027f9b779b17422d0df92223018b32b4d1fa46e071723d6817e2486d003becc55f0055565b6001600160a01b0382165f908152600560205260408120805483929061212b908490613597565b925050819055508060135f8282546121439190613597565b90915550505050565b600180546001600160a01b031916905561136b81612b23565b61216d6120ab565b6001805460ff60a01b1916600160a01b1790557f62e78cea01bee320cd4e420270b5ea74000d11b0c9f74754ebdbfc544b05a2586120933390565b60606121d37f0000000000000000000000000000000000000000000000000000000000000000612b72565b905090565b60606121d37f0000000000000000000000000000000000000000000000000000000000000000612b72565b84355f908152600b602052604090205460ff1615612234576040516366fad46560e01b815260040160405180910390fd5b600c5f612247608088016060890161312d565b6001600160a01b0316815260208101919091526040015f908120906122746102a0880161028089016131a8565b6001600160401b0316815260208101919091526040015f205460ff16156122ad57604051623f613760e71b815260040160405180910390fd5b843515806122d057506122c8610220860161020087016137ec565b60ff16600114155b8061231c57506001600160a01b037f00000000000000000000000000000000000000000000000000000000000000001661231060a087016080880161312d565b6001600160a01b031614155b8061233f575061233461020086016101e087016131a8565b6001600160401b0316155b8061239057506123576101e086016101c0870161341f565b61ffff16600e5f61237061020089016101e08a016131a8565b6001600160401b0316815260208101919091526040015f205461ffff1614155b806123b45750426123a9610240870161022088016131a8565b6001600160401b0316115b156123d25760405163300262ab60e21b815260040160405180910390fd5b6020808601355f908152600a909152604090819020906123f8906060880190880161312d565b81546001600160a01b03908116911614158061241c575080546001600160a01b0316155b8061243257506003810154600160801b900460ff165b15612450576040516316f78d3b60e11b815260040160405180910390fd5b60038101546001600160401b0316612470610240880161022089016131a8565b6001600160401b0316111561249857604051630fe82d2560e11b815260040160405180910390fd5b5f6009816124ac60808a0160608b0161312d565b6001600160a01b0316815260208082019290925260409081015f90812060a08b013582529092528120906124e660e08a0160c08b016131a8565b6001600160401b0316815260208101919091526040015f20805490915060ff16158061251a57508660e00135816001015414155b8061252e5750866101000135816002015414155b1561254c5760405163ef8d77e360e01b815260040160405180910390fd5b5f61257a6125626101808a016101608b016131a8565b6001600160401b031683600601546103e86001612baf565b6125a861258f6101608b016101408c016131a8565b6001600160401b03168460050154620f42406001612baf565b6125d66125bd6101408c016101208d016131a8565b6001600160401b03168560040154620f42406001612baf565b6125e09190613597565b6125ea9190613597565b90506125fe6101c089016101a08a016131a8565b6001600160401b031681146126265760405163300262ab60e21b815260040160405180910390fd5b6126386101a089016101808a016131a8565b6001600160401b03168111156126615760405163191baa5360e01b815260040160405180910390fd5b82600101548184600201546126769190613597565b11156126955760405163962b14e360e01b815260040160405180910390fd5b5f6126a861039c368b90038b018b612f95565b90506126ba60808a0160608b0161312d565b6001600160a01b031661270489898080601f0160208091040260200160405190810160405280939291908181526020018383808284375f9201919091525086939250506129549050565b6001600160a01b031614158061278c57507f00000000000000000000000000000000000000000000000000000000000000006001600160a01b031661278087878080601f0160208091040260200160405190810160405280939291908181526020018383808284375f9201919091525086939250506129549050565b6001600160a01b031614155b156127aa57604051635b37cf6960e11b815260040160405180910390fd5b88355f908152600b60205260408120805460ff1916600190811790915590600c906127db60808d0160608e0161312d565b6001600160a01b0316815260208101919091526040015f908120906128086102a08d016102808e016131a8565b6001600160401b03166001600160401b031681526020019081526020015f205f6101000a81548160ff02191690831515021790555081846002015f8282546128509190613597565b925050819055508160125f8282546128689190613584565b909155505f9050612891836128856101e08d016101c08e0161341f565b61ffff16612710612bfa565b90505f61289e8285613584565b90506128b96128b360808d0160608e0161312d565b82612104565b6128e37f000000000000000000000000000000000000000000000000000000000000000083612104565b6128f360808c0160608d0161312d565b6001600160a01b03168b602001358c5f01357fc429078b36d592f1bf76c1f12f1ca4c553be70723c2ad50d40a444e32b7ec06d848660405161293f929190918252602082015260400190565b60405180910390a45050505050505050505050565b5f5f5f5f6129628686612caa565b9250925092506129728282612cf3565b5090949350505050565b5f8282188284100282185b9392505050565b5f306001600160a01b037f0000000000000000000000000000000000000000000000000000000000000000161480156129e657507f000000000000000000000000000000000000000000000000000000000000000046145b15612a1057507f000000000000000000000000000000000000000000000000000000000000000090565b6121d3604080517f8b73c3c69bb8fe3d512ecc4cf759cc79239f7b179b0ffacaa9a75d522b39400f60208201527f0000000000000000000000000000000000000000000000000000000000000000918101919091527f000000000000000000000000000000000000000000000000000000000000000060608201524660808201523060a08201525f9060c00160405160208183030381529060405280519060200120905090565b600154600160a01b900460ff16610d3057604051638dfc202b60e01b815260040160405180910390fd5b7f9b779b17422d0df92223018b32b4d1fa46e071723d6817e2486d003becc55f0054600203610d3057604051633ee5aeb560e01b815260040160405180910390fd5b5f80546001600160a01b038381166001600160a01b0319831681178455604051919092169283917f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e09190a35050565b60605f612b7e83612daf565b6040805160208082528183019092529192505f91906020820181803683375050509182525060208101929092525090565b5f612bdc612bbc83612dd6565b8015612bd757505f8480612bd257612bd2613805565b868809115b151590565b612be7868686612bfa565b612bf19190613597565b95945050505050565b5f5f5f612c078686612e02565b91509150815f03612c2b57838181612c2157612c21613805565b0492505050612987565b818411612c4257612c426003851502601118612e1e565b5f848688095f868103871696879004966002600389028118808a02820302808a02820302808a02820302808a02820302808a02820302808a02909103029181900381900460010185841190960395909502919093039390930492909217029150509392505050565b5f5f5f8351604103612ce1576020840151604085015160608601515f1a612cd388828585612e2f565b955095509550505050612cec565b505081515f91506002905b9250925092565b5f826003811115612d0657612d06613819565b03612d0f575050565b6001826003811115612d2357612d23613819565b03612d415760405163f645eedf60e01b815260040160405180910390fd5b6002826003811115612d5557612d55613819565b03612d765760405163fce698f760e01b815260048101829052602401611359565b6003826003811115612d8a57612d8a613819565b03612dab576040516335e2f38360e21b815260048101829052602401611359565b5050565b5f60ff8216601f811115610d1a57604051632cd44ac360e21b815260040160405180910390fd5b5f6002826003811115612deb57612deb613819565b612df5919061382d565b60ff166001149050919050565b5f805f1983850993909202808410938190039390930393915050565b634e487b715f52806020526024601cfd5b5f80807f7fffffffffffffffffffffffffffffff5d576e7357a4501ddfe92f46681b20a0841115612e6857505f91506003905082612eed565b604080515f808252602082018084528a905260ff891692820192909252606081018790526080810186905260019060a0016020604051602081039080840390855afa158015612eb9573d5f5f3e3d5ffd5b5050604051601f1901519150506001600160a01b038116612ee457505f925060019150829050612eed565b92505f91508190505b9450945094915050565b5f60208284031215612f07575f5ffd5b5035919050565b6040516102a081016001600160401b0381118282101715612f3d57634e487b7160e01b5f52604160045260245ffd5b60405290565b80356001600160a01b0381168114612f59575f5ffd5b919050565b80356001600160401b0381168114612f59575f5ffd5b803561ffff81168114612f59575f5ffd5b803560ff81168114612f59575f5ffd5b5f6102a0828403128015612fa7575f5ffd5b50612fb0612f0e565b8235815260208084013590820152612fca60408401612f43565b6040820152612fdb60608401612f43565b6060820152612fec60808401612f43565b608082015260a0838101359082015261300760c08401612f5e565b60c082015260e08381013590820152610100808401359082015261302e6101208401612f5e565b6101208201526130416101408401612f5e565b6101408201526130546101608401612f5e565b6101608201526130676101808401612f5e565b61018082015261307a6101a08401612f5e565b6101a082015261308d6101c08401612f74565b6101c08201526130a06101e08401612f5e565b6101e08201526130b36102008401612f85565b6102008201526130c66102208401612f5e565b610220820152610240838101359082015261026080840135908201526130ef6102808401612f5e565b6102808201529392505050565b5f5f6040838503121561310d575f5ffd5b61311683612f43565b915061312460208401612f5e565b90509250929050565b5f6020828403121561313d575f5ffd5b61298782612f43565b5f5f5f60608486031215613158575f5ffd5b833592506020840135915061316f60408501612f5e565b90509250925092565b5f5f5f6060848603121561318a575f5ffd5b61319384612f43565b92506020840135915061316f60408501612f5e565b5f602082840312156131b8575f5ffd5b61298782612f5e565b5f5f604083850312156131d2575f5ffd5b6131db83612f43565b946020939093013593505050565b5f81518084528060208401602086015e5f602082860101526020601f19601f83011685010191505092915050565b60ff60f81b8816815260e060208201525f61323560e08301896131e9565b828103604084015261324781896131e9565b606084018890526001600160a01b038716608085015260a0840186905283810360c0850152845180825260208087019350909101905f5b8181101561329c57835183526020938401939092019160010161327e565b50909b9a5050505050505050505050565b5f6102a082840312156132be575f5ffd5b50919050565b5f5f83601f8401126132d4575f5ffd5b5081356001600160401b038111156132ea575f5ffd5b602083019150836020828501011115613301575f5ffd5b9250929050565b5f5f5f5f5f6102e0868803121561331d575f5ffd5b61332787876132ad565b94506102a08601356001600160401b03811115613342575f5ffd5b61334e888289016132c4565b9095509350506102c08601356001600160401b0381111561336d575f5ffd5b613379888289016132c4565b969995985093965092949392505050565b5f5f5f5f5f5f61058087890312156133a0575f5ffd5b6133aa88886132ad565b95506102a08701356001600160401b038111156133c5575f5ffd5b6133d189828a016132c4565b90965094506133e69050886102c089016132ad565b92506105608701356001600160401b03811115613401575f5ffd5b61340d89828a016132c4565b979a9699509497509295939492505050565b5f6020828403121561342f575f5ffd5b61298782612f74565b5f5f83601f840112613448575f5ffd5b5081356001600160401b0381111561345e575f5ffd5b6020830191508360208260051b8501011115613301575f5ffd5b5f5f5f5f5f5f6060878903121561348d575f5ffd5b86356001600160401b038111156134a2575f5ffd5b8701601f810189136134b2575f5ffd5b80356001600160401b038111156134c7575f5ffd5b8960206102a0830284010111156134dc575f5ffd5b6020918201975095508701356001600160401b038111156134fb575f5ffd5b61350789828a01613438565b90955093505060408701356001600160401b03811115613525575f5ffd5b61340d89828a01613438565b5f5f5f5f5f5f60c08789031215613546575f5ffd5b505084359660208601359650604086013595606081013595506080810135945060a0013592509050565b634e487b7160e01b5f52601160045260245ffd5b81810381811115610d1a57610d1a613570565b80820180821115610d1a57610d1a613570565b5f6102c082019050838252825160208301526020830151604083015260408301516135e060608401826001600160a01b03169052565b5060608301516001600160a01b03811660808401525060808301516001600160a01b03811660a08401525060a083015160c083015260c083015161362f60e08401826001600160401b03169052565b5060e08301516101008301526101008301516101208301526101208301516136636101408401826001600160401b03169052565b506101408301516001600160401b038116610160840152506101608301516001600160401b038116610180840152506101808301516001600160401b0381166101a0840152506101a08301516001600160401b0381166101c0840152506101c083015161ffff81166101e0840152506101e08301516001600160401b0381166102008401525061020083015160ff8116610220840152506102208301516001600160401b038116610240840152506102408301516102608301526102608301516102808301526102808301516137456102a08401826001600160401b03169052565b509392505050565b6001600160401b038181168382160190811115610d1a57610d1a613570565b634e487b7160e01b5f52603260045260245ffd5b5f5f8335601e19843603018112613795575f5ffd5b8301803591506001600160401b038211156137ae575f5ffd5b602001915036819003821315613301575f5ffd5b5f6001600160401b0382166001600160401b0381036137e3576137e3613570565b60010192915050565b5f602082840312156137fc575f5ffd5b61298782612f85565b634e487b7160e01b5f52601260045260245ffd5b634e487b7160e01b5f52602160045260245ffd5b5f60ff83168061384b57634e487b7160e01b5f52601260045260245ffd5b8060ff8416069150509291505056fea2646970667358221220d1e6bfe9a32545973c27bfc95c186e64f0ece0140d2876bfae7791df264db42d64736f6c634300081e0033",
}

// MyferenceMarketABI is the input ABI used to generate the binding from.
// Deprecated: Use MyferenceMarketMetaData.ABI instead.
var MyferenceMarketABI = MyferenceMarketMetaData.ABI

// MyferenceMarketBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use MyferenceMarketMetaData.Bin instead.
var MyferenceMarketBin = MyferenceMarketMetaData.Bin

// DeployMyferenceMarket deploys a new Ethereum contract, binding an instance of MyferenceMarket to it.
func DeployMyferenceMarket(auth *bind.TransactOpts, backend bind.ContractBackend, initialOwner common.Address, feeRecipient_ common.Address, settlementSigner_ common.Address, minimumBond_ *big.Int, bondExitDelay_ uint64, feeDelay_ uint64) (common.Address, *types.Transaction, *MyferenceMarket, error) {
	parsed, err := MyferenceMarketMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(MyferenceMarketBin), backend, initialOwner, feeRecipient_, settlementSigner_, minimumBond_, bondExitDelay_, feeDelay_)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &MyferenceMarket{MyferenceMarketCaller: MyferenceMarketCaller{contract: contract}, MyferenceMarketTransactor: MyferenceMarketTransactor{contract: contract}, MyferenceMarketFilterer: MyferenceMarketFilterer{contract: contract}}, nil
}

// MyferenceMarket is an auto generated Go binding around an Ethereum contract.
type MyferenceMarket struct {
	MyferenceMarketCaller     // Read-only binding to the contract
	MyferenceMarketTransactor // Write-only binding to the contract
	MyferenceMarketFilterer   // Log filterer for contract events
}

// MyferenceMarketCaller is an auto generated read-only Go binding around an Ethereum contract.
type MyferenceMarketCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MyferenceMarketTransactor is an auto generated write-only Go binding around an Ethereum contract.
type MyferenceMarketTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MyferenceMarketFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type MyferenceMarketFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MyferenceMarketSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type MyferenceMarketSession struct {
	Contract     *MyferenceMarket  // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// MyferenceMarketCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type MyferenceMarketCallerSession struct {
	Contract *MyferenceMarketCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts          // Call options to use throughout this session
}

// MyferenceMarketTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type MyferenceMarketTransactorSession struct {
	Contract     *MyferenceMarketTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts          // Transaction auth options to use throughout this session
}

// MyferenceMarketRaw is an auto generated low-level Go binding around an Ethereum contract.
type MyferenceMarketRaw struct {
	Contract *MyferenceMarket // Generic contract binding to access the raw methods on
}

// MyferenceMarketCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type MyferenceMarketCallerRaw struct {
	Contract *MyferenceMarketCaller // Generic read-only contract binding to access the raw methods on
}

// MyferenceMarketTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type MyferenceMarketTransactorRaw struct {
	Contract *MyferenceMarketTransactor // Generic write-only contract binding to access the raw methods on
}

// NewMyferenceMarket creates a new instance of MyferenceMarket, bound to a specific deployed contract.
func NewMyferenceMarket(address common.Address, backend bind.ContractBackend) (*MyferenceMarket, error) {
	contract, err := bindMyferenceMarket(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &MyferenceMarket{MyferenceMarketCaller: MyferenceMarketCaller{contract: contract}, MyferenceMarketTransactor: MyferenceMarketTransactor{contract: contract}, MyferenceMarketFilterer: MyferenceMarketFilterer{contract: contract}}, nil
}

// NewMyferenceMarketCaller creates a new read-only instance of MyferenceMarket, bound to a specific deployed contract.
func NewMyferenceMarketCaller(address common.Address, caller bind.ContractCaller) (*MyferenceMarketCaller, error) {
	contract, err := bindMyferenceMarket(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &MyferenceMarketCaller{contract: contract}, nil
}

// NewMyferenceMarketTransactor creates a new write-only instance of MyferenceMarket, bound to a specific deployed contract.
func NewMyferenceMarketTransactor(address common.Address, transactor bind.ContractTransactor) (*MyferenceMarketTransactor, error) {
	contract, err := bindMyferenceMarket(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &MyferenceMarketTransactor{contract: contract}, nil
}

// NewMyferenceMarketFilterer creates a new log filterer instance of MyferenceMarket, bound to a specific deployed contract.
func NewMyferenceMarketFilterer(address common.Address, filterer bind.ContractFilterer) (*MyferenceMarketFilterer, error) {
	contract, err := bindMyferenceMarket(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &MyferenceMarketFilterer{contract: contract}, nil
}

// bindMyferenceMarket binds a generic wrapper to an already deployed contract.
func bindMyferenceMarket(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := MyferenceMarketMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_MyferenceMarket *MyferenceMarketRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _MyferenceMarket.Contract.MyferenceMarketCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_MyferenceMarket *MyferenceMarketRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MyferenceMarket.Contract.MyferenceMarketTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_MyferenceMarket *MyferenceMarketRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _MyferenceMarket.Contract.MyferenceMarketTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_MyferenceMarket *MyferenceMarketCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _MyferenceMarket.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_MyferenceMarket *MyferenceMarketTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MyferenceMarket.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_MyferenceMarket *MyferenceMarketTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _MyferenceMarket.Contract.contract.Transact(opts, method, params...)
}

// MAXIMUMFEEBASISPOINTS is a free data retrieval call binding the contract method 0x1fb545d3.
//
// Solidity: function MAXIMUM_FEE_BASIS_POINTS() view returns(uint16)
func (_MyferenceMarket *MyferenceMarketCaller) MAXIMUMFEEBASISPOINTS(opts *bind.CallOpts) (uint16, error) {
	var out []interface{}
	err := _MyferenceMarket.contract.Call(opts, &out, "MAXIMUM_FEE_BASIS_POINTS")

	if err != nil {
		return *new(uint16), err
	}

	out0 := *abi.ConvertType(out[0], new(uint16)).(*uint16)

	return out0, err

}

// MAXIMUMFEEBASISPOINTS is a free data retrieval call binding the contract method 0x1fb545d3.
//
// Solidity: function MAXIMUM_FEE_BASIS_POINTS() view returns(uint16)
func (_MyferenceMarket *MyferenceMarketSession) MAXIMUMFEEBASISPOINTS() (uint16, error) {
	return _MyferenceMarket.Contract.MAXIMUMFEEBASISPOINTS(&_MyferenceMarket.CallOpts)
}

// MAXIMUMFEEBASISPOINTS is a free data retrieval call binding the contract method 0x1fb545d3.
//
// Solidity: function MAXIMUM_FEE_BASIS_POINTS() view returns(uint16)
func (_MyferenceMarket *MyferenceMarketCallerSession) MAXIMUMFEEBASISPOINTS() (uint16, error) {
	return _MyferenceMarket.Contract.MAXIMUMFEEBASISPOINTS(&_MyferenceMarket.CallOpts)
}

// RECEIPTTYPEHASH is a free data retrieval call binding the contract method 0x95850f59.
//
// Solidity: function RECEIPT_TYPEHASH() view returns(bytes32)
func (_MyferenceMarket *MyferenceMarketCaller) RECEIPTTYPEHASH(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _MyferenceMarket.contract.Call(opts, &out, "RECEIPT_TYPEHASH")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// RECEIPTTYPEHASH is a free data retrieval call binding the contract method 0x95850f59.
//
// Solidity: function RECEIPT_TYPEHASH() view returns(bytes32)
func (_MyferenceMarket *MyferenceMarketSession) RECEIPTTYPEHASH() ([32]byte, error) {
	return _MyferenceMarket.Contract.RECEIPTTYPEHASH(&_MyferenceMarket.CallOpts)
}

// RECEIPTTYPEHASH is a free data retrieval call binding the contract method 0x95850f59.
//
// Solidity: function RECEIPT_TYPEHASH() view returns(bytes32)
func (_MyferenceMarket *MyferenceMarketCallerSession) RECEIPTTYPEHASH() ([32]byte, error) {
	return _MyferenceMarket.Contract.RECEIPTTYPEHASH(&_MyferenceMarket.CallOpts)
}

// SESSIONCLOSEDELAY is a free data retrieval call binding the contract method 0x6a0755f9.
//
// Solidity: function SESSION_CLOSE_DELAY() view returns(uint64)
func (_MyferenceMarket *MyferenceMarketCaller) SESSIONCLOSEDELAY(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _MyferenceMarket.contract.Call(opts, &out, "SESSION_CLOSE_DELAY")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// SESSIONCLOSEDELAY is a free data retrieval call binding the contract method 0x6a0755f9.
//
// Solidity: function SESSION_CLOSE_DELAY() view returns(uint64)
func (_MyferenceMarket *MyferenceMarketSession) SESSIONCLOSEDELAY() (uint64, error) {
	return _MyferenceMarket.Contract.SESSIONCLOSEDELAY(&_MyferenceMarket.CallOpts)
}

// SESSIONCLOSEDELAY is a free data retrieval call binding the contract method 0x6a0755f9.
//
// Solidity: function SESSION_CLOSE_DELAY() view returns(uint64)
func (_MyferenceMarket *MyferenceMarketCallerSession) SESSIONCLOSEDELAY() (uint64, error) {
	return _MyferenceMarket.Contract.SESSIONCLOSEDELAY(&_MyferenceMarket.CallOpts)
}

// BondExitAvailableAt is a free data retrieval call binding the contract method 0x55512c1f.
//
// Solidity: function bondExitAvailableAt(address ) view returns(uint64)
func (_MyferenceMarket *MyferenceMarketCaller) BondExitAvailableAt(opts *bind.CallOpts, arg0 common.Address) (uint64, error) {
	var out []interface{}
	err := _MyferenceMarket.contract.Call(opts, &out, "bondExitAvailableAt", arg0)

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// BondExitAvailableAt is a free data retrieval call binding the contract method 0x55512c1f.
//
// Solidity: function bondExitAvailableAt(address ) view returns(uint64)
func (_MyferenceMarket *MyferenceMarketSession) BondExitAvailableAt(arg0 common.Address) (uint64, error) {
	return _MyferenceMarket.Contract.BondExitAvailableAt(&_MyferenceMarket.CallOpts, arg0)
}

// BondExitAvailableAt is a free data retrieval call binding the contract method 0x55512c1f.
//
// Solidity: function bondExitAvailableAt(address ) view returns(uint64)
func (_MyferenceMarket *MyferenceMarketCallerSession) BondExitAvailableAt(arg0 common.Address) (uint64, error) {
	return _MyferenceMarket.Contract.BondExitAvailableAt(&_MyferenceMarket.CallOpts, arg0)
}

// BondExitDelay is a free data retrieval call binding the contract method 0x16ddad33.
//
// Solidity: function bondExitDelay() view returns(uint64)
func (_MyferenceMarket *MyferenceMarketCaller) BondExitDelay(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _MyferenceMarket.contract.Call(opts, &out, "bondExitDelay")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// BondExitDelay is a free data retrieval call binding the contract method 0x16ddad33.
//
// Solidity: function bondExitDelay() view returns(uint64)
func (_MyferenceMarket *MyferenceMarketSession) BondExitDelay() (uint64, error) {
	return _MyferenceMarket.Contract.BondExitDelay(&_MyferenceMarket.CallOpts)
}

// BondExitDelay is a free data retrieval call binding the contract method 0x16ddad33.
//
// Solidity: function bondExitDelay() view returns(uint64)
func (_MyferenceMarket *MyferenceMarketCallerSession) BondExitDelay() (uint64, error) {
	return _MyferenceMarket.Contract.BondExitDelay(&_MyferenceMarket.CallOpts)
}

// Claimable is a free data retrieval call binding the contract method 0x402914f5.
//
// Solidity: function claimable(address ) view returns(uint256)
func (_MyferenceMarket *MyferenceMarketCaller) Claimable(opts *bind.CallOpts, arg0 common.Address) (*big.Int, error) {
	var out []interface{}
	err := _MyferenceMarket.contract.Call(opts, &out, "claimable", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Claimable is a free data retrieval call binding the contract method 0x402914f5.
//
// Solidity: function claimable(address ) view returns(uint256)
func (_MyferenceMarket *MyferenceMarketSession) Claimable(arg0 common.Address) (*big.Int, error) {
	return _MyferenceMarket.Contract.Claimable(&_MyferenceMarket.CallOpts, arg0)
}

// Claimable is a free data retrieval call binding the contract method 0x402914f5.
//
// Solidity: function claimable(address ) view returns(uint256)
func (_MyferenceMarket *MyferenceMarketCallerSession) Claimable(arg0 common.Address) (*big.Int, error) {
	return _MyferenceMarket.Contract.Claimable(&_MyferenceMarket.CallOpts, arg0)
}

// CustomerBalances is a free data retrieval call binding the contract method 0x791d66a1.
//
// Solidity: function customerBalances(address ) view returns(uint256)
func (_MyferenceMarket *MyferenceMarketCaller) CustomerBalances(opts *bind.CallOpts, arg0 common.Address) (*big.Int, error) {
	var out []interface{}
	err := _MyferenceMarket.contract.Call(opts, &out, "customerBalances", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// CustomerBalances is a free data retrieval call binding the contract method 0x791d66a1.
//
// Solidity: function customerBalances(address ) view returns(uint256)
func (_MyferenceMarket *MyferenceMarketSession) CustomerBalances(arg0 common.Address) (*big.Int, error) {
	return _MyferenceMarket.Contract.CustomerBalances(&_MyferenceMarket.CallOpts, arg0)
}

// CustomerBalances is a free data retrieval call binding the contract method 0x791d66a1.
//
// Solidity: function customerBalances(address ) view returns(uint256)
func (_MyferenceMarket *MyferenceMarketCallerSession) CustomerBalances(arg0 common.Address) (*big.Int, error) {
	return _MyferenceMarket.Contract.CustomerBalances(&_MyferenceMarket.CallOpts, arg0)
}

// Eip712Domain is a free data retrieval call binding the contract method 0x84b0196e.
//
// Solidity: function eip712Domain() view returns(bytes1 fields, string name, string version, uint256 chainId, address verifyingContract, bytes32 salt, uint256[] extensions)
func (_MyferenceMarket *MyferenceMarketCaller) Eip712Domain(opts *bind.CallOpts) (struct {
	Fields            [1]byte
	Name              string
	Version           string
	ChainId           *big.Int
	VerifyingContract common.Address
	Salt              [32]byte
	Extensions        []*big.Int
}, error) {
	var out []interface{}
	err := _MyferenceMarket.contract.Call(opts, &out, "eip712Domain")

	outstruct := new(struct {
		Fields            [1]byte
		Name              string
		Version           string
		ChainId           *big.Int
		VerifyingContract common.Address
		Salt              [32]byte
		Extensions        []*big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Fields = *abi.ConvertType(out[0], new([1]byte)).(*[1]byte)
	outstruct.Name = *abi.ConvertType(out[1], new(string)).(*string)
	outstruct.Version = *abi.ConvertType(out[2], new(string)).(*string)
	outstruct.ChainId = *abi.ConvertType(out[3], new(*big.Int)).(**big.Int)
	outstruct.VerifyingContract = *abi.ConvertType(out[4], new(common.Address)).(*common.Address)
	outstruct.Salt = *abi.ConvertType(out[5], new([32]byte)).(*[32]byte)
	outstruct.Extensions = *abi.ConvertType(out[6], new([]*big.Int)).(*[]*big.Int)

	return *outstruct, err

}

// Eip712Domain is a free data retrieval call binding the contract method 0x84b0196e.
//
// Solidity: function eip712Domain() view returns(bytes1 fields, string name, string version, uint256 chainId, address verifyingContract, bytes32 salt, uint256[] extensions)
func (_MyferenceMarket *MyferenceMarketSession) Eip712Domain() (struct {
	Fields            [1]byte
	Name              string
	Version           string
	ChainId           *big.Int
	VerifyingContract common.Address
	Salt              [32]byte
	Extensions        []*big.Int
}, error) {
	return _MyferenceMarket.Contract.Eip712Domain(&_MyferenceMarket.CallOpts)
}

// Eip712Domain is a free data retrieval call binding the contract method 0x84b0196e.
//
// Solidity: function eip712Domain() view returns(bytes1 fields, string name, string version, uint256 chainId, address verifyingContract, bytes32 salt, uint256[] extensions)
func (_MyferenceMarket *MyferenceMarketCallerSession) Eip712Domain() (struct {
	Fields            [1]byte
	Name              string
	Version           string
	ChainId           *big.Int
	VerifyingContract common.Address
	Salt              [32]byte
	Extensions        []*big.Int
}, error) {
	return _MyferenceMarket.Contract.Eip712Domain(&_MyferenceMarket.CallOpts)
}

// FeeBasisPoints is a free data retrieval call binding the contract method 0xb8606eef.
//
// Solidity: function feeBasisPoints() view returns(uint16)
func (_MyferenceMarket *MyferenceMarketCaller) FeeBasisPoints(opts *bind.CallOpts) (uint16, error) {
	var out []interface{}
	err := _MyferenceMarket.contract.Call(opts, &out, "feeBasisPoints")

	if err != nil {
		return *new(uint16), err
	}

	out0 := *abi.ConvertType(out[0], new(uint16)).(*uint16)

	return out0, err

}

// FeeBasisPoints is a free data retrieval call binding the contract method 0xb8606eef.
//
// Solidity: function feeBasisPoints() view returns(uint16)
func (_MyferenceMarket *MyferenceMarketSession) FeeBasisPoints() (uint16, error) {
	return _MyferenceMarket.Contract.FeeBasisPoints(&_MyferenceMarket.CallOpts)
}

// FeeBasisPoints is a free data retrieval call binding the contract method 0xb8606eef.
//
// Solidity: function feeBasisPoints() view returns(uint16)
func (_MyferenceMarket *MyferenceMarketCallerSession) FeeBasisPoints() (uint16, error) {
	return _MyferenceMarket.Contract.FeeBasisPoints(&_MyferenceMarket.CallOpts)
}

// FeeBpsByVersion is a free data retrieval call binding the contract method 0x6767d653.
//
// Solidity: function feeBpsByVersion(uint64 ) view returns(uint16)
func (_MyferenceMarket *MyferenceMarketCaller) FeeBpsByVersion(opts *bind.CallOpts, arg0 uint64) (uint16, error) {
	var out []interface{}
	err := _MyferenceMarket.contract.Call(opts, &out, "feeBpsByVersion", arg0)

	if err != nil {
		return *new(uint16), err
	}

	out0 := *abi.ConvertType(out[0], new(uint16)).(*uint16)

	return out0, err

}

// FeeBpsByVersion is a free data retrieval call binding the contract method 0x6767d653.
//
// Solidity: function feeBpsByVersion(uint64 ) view returns(uint16)
func (_MyferenceMarket *MyferenceMarketSession) FeeBpsByVersion(arg0 uint64) (uint16, error) {
	return _MyferenceMarket.Contract.FeeBpsByVersion(&_MyferenceMarket.CallOpts, arg0)
}

// FeeBpsByVersion is a free data retrieval call binding the contract method 0x6767d653.
//
// Solidity: function feeBpsByVersion(uint64 ) view returns(uint16)
func (_MyferenceMarket *MyferenceMarketCallerSession) FeeBpsByVersion(arg0 uint64) (uint16, error) {
	return _MyferenceMarket.Contract.FeeBpsByVersion(&_MyferenceMarket.CallOpts, arg0)
}

// FeeDelay is a free data retrieval call binding the contract method 0x777ebe3d.
//
// Solidity: function feeDelay() view returns(uint64)
func (_MyferenceMarket *MyferenceMarketCaller) FeeDelay(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _MyferenceMarket.contract.Call(opts, &out, "feeDelay")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// FeeDelay is a free data retrieval call binding the contract method 0x777ebe3d.
//
// Solidity: function feeDelay() view returns(uint64)
func (_MyferenceMarket *MyferenceMarketSession) FeeDelay() (uint64, error) {
	return _MyferenceMarket.Contract.FeeDelay(&_MyferenceMarket.CallOpts)
}

// FeeDelay is a free data retrieval call binding the contract method 0x777ebe3d.
//
// Solidity: function feeDelay() view returns(uint64)
func (_MyferenceMarket *MyferenceMarketCallerSession) FeeDelay() (uint64, error) {
	return _MyferenceMarket.Contract.FeeDelay(&_MyferenceMarket.CallOpts)
}

// FeeRecipient is a free data retrieval call binding the contract method 0x46904840.
//
// Solidity: function feeRecipient() view returns(address)
func (_MyferenceMarket *MyferenceMarketCaller) FeeRecipient(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _MyferenceMarket.contract.Call(opts, &out, "feeRecipient")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// FeeRecipient is a free data retrieval call binding the contract method 0x46904840.
//
// Solidity: function feeRecipient() view returns(address)
func (_MyferenceMarket *MyferenceMarketSession) FeeRecipient() (common.Address, error) {
	return _MyferenceMarket.Contract.FeeRecipient(&_MyferenceMarket.CallOpts)
}

// FeeRecipient is a free data retrieval call binding the contract method 0x46904840.
//
// Solidity: function feeRecipient() view returns(address)
func (_MyferenceMarket *MyferenceMarketCallerSession) FeeRecipient() (common.Address, error) {
	return _MyferenceMarket.Contract.FeeRecipient(&_MyferenceMarket.CallOpts)
}

// FeeVersion is a free data retrieval call binding the contract method 0xdc4638db.
//
// Solidity: function feeVersion() view returns(uint64)
func (_MyferenceMarket *MyferenceMarketCaller) FeeVersion(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _MyferenceMarket.contract.Call(opts, &out, "feeVersion")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// FeeVersion is a free data retrieval call binding the contract method 0xdc4638db.
//
// Solidity: function feeVersion() view returns(uint64)
func (_MyferenceMarket *MyferenceMarketSession) FeeVersion() (uint64, error) {
	return _MyferenceMarket.Contract.FeeVersion(&_MyferenceMarket.CallOpts)
}

// FeeVersion is a free data retrieval call binding the contract method 0xdc4638db.
//
// Solidity: function feeVersion() view returns(uint64)
func (_MyferenceMarket *MyferenceMarketCallerSession) FeeVersion() (uint64, error) {
	return _MyferenceMarket.Contract.FeeVersion(&_MyferenceMarket.CallOpts)
}

// HashReceipt is a free data retrieval call binding the contract method 0x191708b7.
//
// Solidity: function hashReceipt((bytes32,bytes32,address,address,address,bytes32,uint64,bytes32,bytes32,uint64,uint64,uint64,uint64,uint64,uint16,uint64,uint8,uint64,bytes32,bytes32,uint64) receipt) view returns(bytes32)
func (_MyferenceMarket *MyferenceMarketCaller) HashReceipt(opts *bind.CallOpts, receipt MyferenceMarketReceipt) ([32]byte, error) {
	var out []interface{}
	err := _MyferenceMarket.contract.Call(opts, &out, "hashReceipt", receipt)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// HashReceipt is a free data retrieval call binding the contract method 0x191708b7.
//
// Solidity: function hashReceipt((bytes32,bytes32,address,address,address,bytes32,uint64,bytes32,bytes32,uint64,uint64,uint64,uint64,uint64,uint16,uint64,uint8,uint64,bytes32,bytes32,uint64) receipt) view returns(bytes32)
func (_MyferenceMarket *MyferenceMarketSession) HashReceipt(receipt MyferenceMarketReceipt) ([32]byte, error) {
	return _MyferenceMarket.Contract.HashReceipt(&_MyferenceMarket.CallOpts, receipt)
}

// HashReceipt is a free data retrieval call binding the contract method 0x191708b7.
//
// Solidity: function hashReceipt((bytes32,bytes32,address,address,address,bytes32,uint64,bytes32,bytes32,uint64,uint64,uint64,uint64,uint64,uint16,uint64,uint8,uint64,bytes32,bytes32,uint64) receipt) view returns(bytes32)
func (_MyferenceMarket *MyferenceMarketCallerSession) HashReceipt(receipt MyferenceMarketReceipt) ([32]byte, error) {
	return _MyferenceMarket.Contract.HashReceipt(&_MyferenceMarket.CallOpts, receipt)
}

// LatestOfferVersion is a free data retrieval call binding the contract method 0x7eff8a4e.
//
// Solidity: function latestOfferVersion(address , bytes32 ) view returns(uint64)
func (_MyferenceMarket *MyferenceMarketCaller) LatestOfferVersion(opts *bind.CallOpts, arg0 common.Address, arg1 [32]byte) (uint64, error) {
	var out []interface{}
	err := _MyferenceMarket.contract.Call(opts, &out, "latestOfferVersion", arg0, arg1)

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// LatestOfferVersion is a free data retrieval call binding the contract method 0x7eff8a4e.
//
// Solidity: function latestOfferVersion(address , bytes32 ) view returns(uint64)
func (_MyferenceMarket *MyferenceMarketSession) LatestOfferVersion(arg0 common.Address, arg1 [32]byte) (uint64, error) {
	return _MyferenceMarket.Contract.LatestOfferVersion(&_MyferenceMarket.CallOpts, arg0, arg1)
}

// LatestOfferVersion is a free data retrieval call binding the contract method 0x7eff8a4e.
//
// Solidity: function latestOfferVersion(address , bytes32 ) view returns(uint64)
func (_MyferenceMarket *MyferenceMarketCallerSession) LatestOfferVersion(arg0 common.Address, arg1 [32]byte) (uint64, error) {
	return _MyferenceMarket.Contract.LatestOfferVersion(&_MyferenceMarket.CallOpts, arg0, arg1)
}

// MinimumBond is a free data retrieval call binding the contract method 0xaa7517e1.
//
// Solidity: function minimumBond() view returns(uint256)
func (_MyferenceMarket *MyferenceMarketCaller) MinimumBond(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _MyferenceMarket.contract.Call(opts, &out, "minimumBond")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MinimumBond is a free data retrieval call binding the contract method 0xaa7517e1.
//
// Solidity: function minimumBond() view returns(uint256)
func (_MyferenceMarket *MyferenceMarketSession) MinimumBond() (*big.Int, error) {
	return _MyferenceMarket.Contract.MinimumBond(&_MyferenceMarket.CallOpts)
}

// MinimumBond is a free data retrieval call binding the contract method 0xaa7517e1.
//
// Solidity: function minimumBond() view returns(uint256)
func (_MyferenceMarket *MyferenceMarketCallerSession) MinimumBond() (*big.Int, error) {
	return _MyferenceMarket.Contract.MinimumBond(&_MyferenceMarket.CallOpts)
}

// OfferVersions is a free data retrieval call binding the contract method 0x6527a806.
//
// Solidity: function offerVersions(address , bytes32 , uint64 ) view returns(bool active, bytes32 modelHash, bytes32 capabilityHash, uint64 version, uint256 inputPerMillion, uint256 outputPerMillion, uint256 computePerSecond)
func (_MyferenceMarket *MyferenceMarketCaller) OfferVersions(opts *bind.CallOpts, arg0 common.Address, arg1 [32]byte, arg2 uint64) (struct {
	Active           bool
	ModelHash        [32]byte
	CapabilityHash   [32]byte
	Version          uint64
	InputPerMillion  *big.Int
	OutputPerMillion *big.Int
	ComputePerSecond *big.Int
}, error) {
	var out []interface{}
	err := _MyferenceMarket.contract.Call(opts, &out, "offerVersions", arg0, arg1, arg2)

	outstruct := new(struct {
		Active           bool
		ModelHash        [32]byte
		CapabilityHash   [32]byte
		Version          uint64
		InputPerMillion  *big.Int
		OutputPerMillion *big.Int
		ComputePerSecond *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Active = *abi.ConvertType(out[0], new(bool)).(*bool)
	outstruct.ModelHash = *abi.ConvertType(out[1], new([32]byte)).(*[32]byte)
	outstruct.CapabilityHash = *abi.ConvertType(out[2], new([32]byte)).(*[32]byte)
	outstruct.Version = *abi.ConvertType(out[3], new(uint64)).(*uint64)
	outstruct.InputPerMillion = *abi.ConvertType(out[4], new(*big.Int)).(**big.Int)
	outstruct.OutputPerMillion = *abi.ConvertType(out[5], new(*big.Int)).(**big.Int)
	outstruct.ComputePerSecond = *abi.ConvertType(out[6], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// OfferVersions is a free data retrieval call binding the contract method 0x6527a806.
//
// Solidity: function offerVersions(address , bytes32 , uint64 ) view returns(bool active, bytes32 modelHash, bytes32 capabilityHash, uint64 version, uint256 inputPerMillion, uint256 outputPerMillion, uint256 computePerSecond)
func (_MyferenceMarket *MyferenceMarketSession) OfferVersions(arg0 common.Address, arg1 [32]byte, arg2 uint64) (struct {
	Active           bool
	ModelHash        [32]byte
	CapabilityHash   [32]byte
	Version          uint64
	InputPerMillion  *big.Int
	OutputPerMillion *big.Int
	ComputePerSecond *big.Int
}, error) {
	return _MyferenceMarket.Contract.OfferVersions(&_MyferenceMarket.CallOpts, arg0, arg1, arg2)
}

// OfferVersions is a free data retrieval call binding the contract method 0x6527a806.
//
// Solidity: function offerVersions(address , bytes32 , uint64 ) view returns(bool active, bytes32 modelHash, bytes32 capabilityHash, uint64 version, uint256 inputPerMillion, uint256 outputPerMillion, uint256 computePerSecond)
func (_MyferenceMarket *MyferenceMarketCallerSession) OfferVersions(arg0 common.Address, arg1 [32]byte, arg2 uint64) (struct {
	Active           bool
	ModelHash        [32]byte
	CapabilityHash   [32]byte
	Version          uint64
	InputPerMillion  *big.Int
	OutputPerMillion *big.Int
	ComputePerSecond *big.Int
}, error) {
	return _MyferenceMarket.Contract.OfferVersions(&_MyferenceMarket.CallOpts, arg0, arg1, arg2)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_MyferenceMarket *MyferenceMarketCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _MyferenceMarket.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_MyferenceMarket *MyferenceMarketSession) Owner() (common.Address, error) {
	return _MyferenceMarket.Contract.Owner(&_MyferenceMarket.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_MyferenceMarket *MyferenceMarketCallerSession) Owner() (common.Address, error) {
	return _MyferenceMarket.Contract.Owner(&_MyferenceMarket.CallOpts)
}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_MyferenceMarket *MyferenceMarketCaller) Paused(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _MyferenceMarket.contract.Call(opts, &out, "paused")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_MyferenceMarket *MyferenceMarketSession) Paused() (bool, error) {
	return _MyferenceMarket.Contract.Paused(&_MyferenceMarket.CallOpts)
}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_MyferenceMarket *MyferenceMarketCallerSession) Paused() (bool, error) {
	return _MyferenceMarket.Contract.Paused(&_MyferenceMarket.CallOpts)
}

// PendingFeeAvailableAt is a free data retrieval call binding the contract method 0x57950440.
//
// Solidity: function pendingFeeAvailableAt() view returns(uint64)
func (_MyferenceMarket *MyferenceMarketCaller) PendingFeeAvailableAt(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _MyferenceMarket.contract.Call(opts, &out, "pendingFeeAvailableAt")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// PendingFeeAvailableAt is a free data retrieval call binding the contract method 0x57950440.
//
// Solidity: function pendingFeeAvailableAt() view returns(uint64)
func (_MyferenceMarket *MyferenceMarketSession) PendingFeeAvailableAt() (uint64, error) {
	return _MyferenceMarket.Contract.PendingFeeAvailableAt(&_MyferenceMarket.CallOpts)
}

// PendingFeeAvailableAt is a free data retrieval call binding the contract method 0x57950440.
//
// Solidity: function pendingFeeAvailableAt() view returns(uint64)
func (_MyferenceMarket *MyferenceMarketCallerSession) PendingFeeAvailableAt() (uint64, error) {
	return _MyferenceMarket.Contract.PendingFeeAvailableAt(&_MyferenceMarket.CallOpts)
}

// PendingFeeBasisPoints is a free data retrieval call binding the contract method 0x6554018a.
//
// Solidity: function pendingFeeBasisPoints() view returns(uint16)
func (_MyferenceMarket *MyferenceMarketCaller) PendingFeeBasisPoints(opts *bind.CallOpts) (uint16, error) {
	var out []interface{}
	err := _MyferenceMarket.contract.Call(opts, &out, "pendingFeeBasisPoints")

	if err != nil {
		return *new(uint16), err
	}

	out0 := *abi.ConvertType(out[0], new(uint16)).(*uint16)

	return out0, err

}

// PendingFeeBasisPoints is a free data retrieval call binding the contract method 0x6554018a.
//
// Solidity: function pendingFeeBasisPoints() view returns(uint16)
func (_MyferenceMarket *MyferenceMarketSession) PendingFeeBasisPoints() (uint16, error) {
	return _MyferenceMarket.Contract.PendingFeeBasisPoints(&_MyferenceMarket.CallOpts)
}

// PendingFeeBasisPoints is a free data retrieval call binding the contract method 0x6554018a.
//
// Solidity: function pendingFeeBasisPoints() view returns(uint16)
func (_MyferenceMarket *MyferenceMarketCallerSession) PendingFeeBasisPoints() (uint16, error) {
	return _MyferenceMarket.Contract.PendingFeeBasisPoints(&_MyferenceMarket.CallOpts)
}

// PendingFeeChange is a free data retrieval call binding the contract method 0x5dc6aec6.
//
// Solidity: function pendingFeeChange() view returns(bool)
func (_MyferenceMarket *MyferenceMarketCaller) PendingFeeChange(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _MyferenceMarket.contract.Call(opts, &out, "pendingFeeChange")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// PendingFeeChange is a free data retrieval call binding the contract method 0x5dc6aec6.
//
// Solidity: function pendingFeeChange() view returns(bool)
func (_MyferenceMarket *MyferenceMarketSession) PendingFeeChange() (bool, error) {
	return _MyferenceMarket.Contract.PendingFeeChange(&_MyferenceMarket.CallOpts)
}

// PendingFeeChange is a free data retrieval call binding the contract method 0x5dc6aec6.
//
// Solidity: function pendingFeeChange() view returns(bool)
func (_MyferenceMarket *MyferenceMarketCallerSession) PendingFeeChange() (bool, error) {
	return _MyferenceMarket.Contract.PendingFeeChange(&_MyferenceMarket.CallOpts)
}

// PendingOwner is a free data retrieval call binding the contract method 0xe30c3978.
//
// Solidity: function pendingOwner() view returns(address)
func (_MyferenceMarket *MyferenceMarketCaller) PendingOwner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _MyferenceMarket.contract.Call(opts, &out, "pendingOwner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// PendingOwner is a free data retrieval call binding the contract method 0xe30c3978.
//
// Solidity: function pendingOwner() view returns(address)
func (_MyferenceMarket *MyferenceMarketSession) PendingOwner() (common.Address, error) {
	return _MyferenceMarket.Contract.PendingOwner(&_MyferenceMarket.CallOpts)
}

// PendingOwner is a free data retrieval call binding the contract method 0xe30c3978.
//
// Solidity: function pendingOwner() view returns(address)
func (_MyferenceMarket *MyferenceMarketCallerSession) PendingOwner() (common.Address, error) {
	return _MyferenceMarket.Contract.PendingOwner(&_MyferenceMarket.CallOpts)
}

// ProviderBonds is a free data retrieval call binding the contract method 0x3a31201e.
//
// Solidity: function providerBonds(address ) view returns(uint256)
func (_MyferenceMarket *MyferenceMarketCaller) ProviderBonds(opts *bind.CallOpts, arg0 common.Address) (*big.Int, error) {
	var out []interface{}
	err := _MyferenceMarket.contract.Call(opts, &out, "providerBonds", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// ProviderBonds is a free data retrieval call binding the contract method 0x3a31201e.
//
// Solidity: function providerBonds(address ) view returns(uint256)
func (_MyferenceMarket *MyferenceMarketSession) ProviderBonds(arg0 common.Address) (*big.Int, error) {
	return _MyferenceMarket.Contract.ProviderBonds(&_MyferenceMarket.CallOpts, arg0)
}

// ProviderBonds is a free data retrieval call binding the contract method 0x3a31201e.
//
// Solidity: function providerBonds(address ) view returns(uint256)
func (_MyferenceMarket *MyferenceMarketCallerSession) ProviderBonds(arg0 common.Address) (*big.Int, error) {
	return _MyferenceMarket.Contract.ProviderBonds(&_MyferenceMarket.CallOpts, arg0)
}

// Sessions is a free data retrieval call binding the contract method 0x7dbd2832.
//
// Solidity: function sessions(bytes32 ) view returns(address customer, uint256 allowance, uint256 spent, uint64 expiresAt, uint64 closeAvailableAt, bool finalized)
func (_MyferenceMarket *MyferenceMarketCaller) Sessions(opts *bind.CallOpts, arg0 [32]byte) (struct {
	Customer         common.Address
	Allowance        *big.Int
	Spent            *big.Int
	ExpiresAt        uint64
	CloseAvailableAt uint64
	Finalized        bool
}, error) {
	var out []interface{}
	err := _MyferenceMarket.contract.Call(opts, &out, "sessions", arg0)

	outstruct := new(struct {
		Customer         common.Address
		Allowance        *big.Int
		Spent            *big.Int
		ExpiresAt        uint64
		CloseAvailableAt uint64
		Finalized        bool
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Customer = *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	outstruct.Allowance = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.Spent = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)
	outstruct.ExpiresAt = *abi.ConvertType(out[3], new(uint64)).(*uint64)
	outstruct.CloseAvailableAt = *abi.ConvertType(out[4], new(uint64)).(*uint64)
	outstruct.Finalized = *abi.ConvertType(out[5], new(bool)).(*bool)

	return *outstruct, err

}

// Sessions is a free data retrieval call binding the contract method 0x7dbd2832.
//
// Solidity: function sessions(bytes32 ) view returns(address customer, uint256 allowance, uint256 spent, uint64 expiresAt, uint64 closeAvailableAt, bool finalized)
func (_MyferenceMarket *MyferenceMarketSession) Sessions(arg0 [32]byte) (struct {
	Customer         common.Address
	Allowance        *big.Int
	Spent            *big.Int
	ExpiresAt        uint64
	CloseAvailableAt uint64
	Finalized        bool
}, error) {
	return _MyferenceMarket.Contract.Sessions(&_MyferenceMarket.CallOpts, arg0)
}

// Sessions is a free data retrieval call binding the contract method 0x7dbd2832.
//
// Solidity: function sessions(bytes32 ) view returns(address customer, uint256 allowance, uint256 spent, uint64 expiresAt, uint64 closeAvailableAt, bool finalized)
func (_MyferenceMarket *MyferenceMarketCallerSession) Sessions(arg0 [32]byte) (struct {
	Customer         common.Address
	Allowance        *big.Int
	Spent            *big.Int
	ExpiresAt        uint64
	CloseAvailableAt uint64
	Finalized        bool
}, error) {
	return _MyferenceMarket.Contract.Sessions(&_MyferenceMarket.CallOpts, arg0)
}

// SettledRequests is a free data retrieval call binding the contract method 0xc691271b.
//
// Solidity: function settledRequests(bytes32 ) view returns(bool)
func (_MyferenceMarket *MyferenceMarketCaller) SettledRequests(opts *bind.CallOpts, arg0 [32]byte) (bool, error) {
	var out []interface{}
	err := _MyferenceMarket.contract.Call(opts, &out, "settledRequests", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// SettledRequests is a free data retrieval call binding the contract method 0xc691271b.
//
// Solidity: function settledRequests(bytes32 ) view returns(bool)
func (_MyferenceMarket *MyferenceMarketSession) SettledRequests(arg0 [32]byte) (bool, error) {
	return _MyferenceMarket.Contract.SettledRequests(&_MyferenceMarket.CallOpts, arg0)
}

// SettledRequests is a free data retrieval call binding the contract method 0xc691271b.
//
// Solidity: function settledRequests(bytes32 ) view returns(bool)
func (_MyferenceMarket *MyferenceMarketCallerSession) SettledRequests(arg0 [32]byte) (bool, error) {
	return _MyferenceMarket.Contract.SettledRequests(&_MyferenceMarket.CallOpts, arg0)
}

// SettlementSigner is a free data retrieval call binding the contract method 0xc46914d8.
//
// Solidity: function settlementSigner() view returns(address)
func (_MyferenceMarket *MyferenceMarketCaller) SettlementSigner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _MyferenceMarket.contract.Call(opts, &out, "settlementSigner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// SettlementSigner is a free data retrieval call binding the contract method 0xc46914d8.
//
// Solidity: function settlementSigner() view returns(address)
func (_MyferenceMarket *MyferenceMarketSession) SettlementSigner() (common.Address, error) {
	return _MyferenceMarket.Contract.SettlementSigner(&_MyferenceMarket.CallOpts)
}

// SettlementSigner is a free data retrieval call binding the contract method 0xc46914d8.
//
// Solidity: function settlementSigner() view returns(address)
func (_MyferenceMarket *MyferenceMarketCallerSession) SettlementSigner() (common.Address, error) {
	return _MyferenceMarket.Contract.SettlementSigner(&_MyferenceMarket.CallOpts)
}

// SlashedRequests is a free data retrieval call binding the contract method 0x558bc5e4.
//
// Solidity: function slashedRequests(bytes32 ) view returns(bool)
func (_MyferenceMarket *MyferenceMarketCaller) SlashedRequests(opts *bind.CallOpts, arg0 [32]byte) (bool, error) {
	var out []interface{}
	err := _MyferenceMarket.contract.Call(opts, &out, "slashedRequests", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// SlashedRequests is a free data retrieval call binding the contract method 0x558bc5e4.
//
// Solidity: function slashedRequests(bytes32 ) view returns(bool)
func (_MyferenceMarket *MyferenceMarketSession) SlashedRequests(arg0 [32]byte) (bool, error) {
	return _MyferenceMarket.Contract.SlashedRequests(&_MyferenceMarket.CallOpts, arg0)
}

// SlashedRequests is a free data retrieval call binding the contract method 0x558bc5e4.
//
// Solidity: function slashedRequests(bytes32 ) view returns(bool)
func (_MyferenceMarket *MyferenceMarketCallerSession) SlashedRequests(arg0 [32]byte) (bool, error) {
	return _MyferenceMarket.Contract.SlashedRequests(&_MyferenceMarket.CallOpts, arg0)
}

// TotalClaimable is a free data retrieval call binding the contract method 0x4838ed19.
//
// Solidity: function totalClaimable() view returns(uint256)
func (_MyferenceMarket *MyferenceMarketCaller) TotalClaimable(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _MyferenceMarket.contract.Call(opts, &out, "totalClaimable")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalClaimable is a free data retrieval call binding the contract method 0x4838ed19.
//
// Solidity: function totalClaimable() view returns(uint256)
func (_MyferenceMarket *MyferenceMarketSession) TotalClaimable() (*big.Int, error) {
	return _MyferenceMarket.Contract.TotalClaimable(&_MyferenceMarket.CallOpts)
}

// TotalClaimable is a free data retrieval call binding the contract method 0x4838ed19.
//
// Solidity: function totalClaimable() view returns(uint256)
func (_MyferenceMarket *MyferenceMarketCallerSession) TotalClaimable() (*big.Int, error) {
	return _MyferenceMarket.Contract.TotalClaimable(&_MyferenceMarket.CallOpts)
}

// TotalCustomerBalances is a free data retrieval call binding the contract method 0xee754f91.
//
// Solidity: function totalCustomerBalances() view returns(uint256)
func (_MyferenceMarket *MyferenceMarketCaller) TotalCustomerBalances(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _MyferenceMarket.contract.Call(opts, &out, "totalCustomerBalances")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalCustomerBalances is a free data retrieval call binding the contract method 0xee754f91.
//
// Solidity: function totalCustomerBalances() view returns(uint256)
func (_MyferenceMarket *MyferenceMarketSession) TotalCustomerBalances() (*big.Int, error) {
	return _MyferenceMarket.Contract.TotalCustomerBalances(&_MyferenceMarket.CallOpts)
}

// TotalCustomerBalances is a free data retrieval call binding the contract method 0xee754f91.
//
// Solidity: function totalCustomerBalances() view returns(uint256)
func (_MyferenceMarket *MyferenceMarketCallerSession) TotalCustomerBalances() (*big.Int, error) {
	return _MyferenceMarket.Contract.TotalCustomerBalances(&_MyferenceMarket.CallOpts)
}

// TotalLockedSessions is a free data retrieval call binding the contract method 0x576e748d.
//
// Solidity: function totalLockedSessions() view returns(uint256)
func (_MyferenceMarket *MyferenceMarketCaller) TotalLockedSessions(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _MyferenceMarket.contract.Call(opts, &out, "totalLockedSessions")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalLockedSessions is a free data retrieval call binding the contract method 0x576e748d.
//
// Solidity: function totalLockedSessions() view returns(uint256)
func (_MyferenceMarket *MyferenceMarketSession) TotalLockedSessions() (*big.Int, error) {
	return _MyferenceMarket.Contract.TotalLockedSessions(&_MyferenceMarket.CallOpts)
}

// TotalLockedSessions is a free data retrieval call binding the contract method 0x576e748d.
//
// Solidity: function totalLockedSessions() view returns(uint256)
func (_MyferenceMarket *MyferenceMarketCallerSession) TotalLockedSessions() (*big.Int, error) {
	return _MyferenceMarket.Contract.TotalLockedSessions(&_MyferenceMarket.CallOpts)
}

// TotalProviderBonds is a free data retrieval call binding the contract method 0x6a0b3078.
//
// Solidity: function totalProviderBonds() view returns(uint256)
func (_MyferenceMarket *MyferenceMarketCaller) TotalProviderBonds(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _MyferenceMarket.contract.Call(opts, &out, "totalProviderBonds")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalProviderBonds is a free data retrieval call binding the contract method 0x6a0b3078.
//
// Solidity: function totalProviderBonds() view returns(uint256)
func (_MyferenceMarket *MyferenceMarketSession) TotalProviderBonds() (*big.Int, error) {
	return _MyferenceMarket.Contract.TotalProviderBonds(&_MyferenceMarket.CallOpts)
}

// TotalProviderBonds is a free data retrieval call binding the contract method 0x6a0b3078.
//
// Solidity: function totalProviderBonds() view returns(uint256)
func (_MyferenceMarket *MyferenceMarketCallerSession) TotalProviderBonds() (*big.Int, error) {
	return _MyferenceMarket.Contract.TotalProviderBonds(&_MyferenceMarket.CallOpts)
}

// UsedNonces is a free data retrieval call binding the contract method 0x33a87bb1.
//
// Solidity: function usedNonces(address , uint64 ) view returns(bool)
func (_MyferenceMarket *MyferenceMarketCaller) UsedNonces(opts *bind.CallOpts, arg0 common.Address, arg1 uint64) (bool, error) {
	var out []interface{}
	err := _MyferenceMarket.contract.Call(opts, &out, "usedNonces", arg0, arg1)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// UsedNonces is a free data retrieval call binding the contract method 0x33a87bb1.
//
// Solidity: function usedNonces(address , uint64 ) view returns(bool)
func (_MyferenceMarket *MyferenceMarketSession) UsedNonces(arg0 common.Address, arg1 uint64) (bool, error) {
	return _MyferenceMarket.Contract.UsedNonces(&_MyferenceMarket.CallOpts, arg0, arg1)
}

// UsedNonces is a free data retrieval call binding the contract method 0x33a87bb1.
//
// Solidity: function usedNonces(address , uint64 ) view returns(bool)
func (_MyferenceMarket *MyferenceMarketCallerSession) UsedNonces(arg0 common.Address, arg1 uint64) (bool, error) {
	return _MyferenceMarket.Contract.UsedNonces(&_MyferenceMarket.CallOpts, arg0, arg1)
}

// AcceptOwnership is a paid mutator transaction binding the contract method 0x79ba5097.
//
// Solidity: function acceptOwnership() returns()
func (_MyferenceMarket *MyferenceMarketTransactor) AcceptOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MyferenceMarket.contract.Transact(opts, "acceptOwnership")
}

// AcceptOwnership is a paid mutator transaction binding the contract method 0x79ba5097.
//
// Solidity: function acceptOwnership() returns()
func (_MyferenceMarket *MyferenceMarketSession) AcceptOwnership() (*types.Transaction, error) {
	return _MyferenceMarket.Contract.AcceptOwnership(&_MyferenceMarket.TransactOpts)
}

// AcceptOwnership is a paid mutator transaction binding the contract method 0x79ba5097.
//
// Solidity: function acceptOwnership() returns()
func (_MyferenceMarket *MyferenceMarketTransactorSession) AcceptOwnership() (*types.Transaction, error) {
	return _MyferenceMarket.Contract.AcceptOwnership(&_MyferenceMarket.TransactOpts)
}

// Claim is a paid mutator transaction binding the contract method 0x4e71d92d.
//
// Solidity: function claim() returns()
func (_MyferenceMarket *MyferenceMarketTransactor) Claim(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MyferenceMarket.contract.Transact(opts, "claim")
}

// Claim is a paid mutator transaction binding the contract method 0x4e71d92d.
//
// Solidity: function claim() returns()
func (_MyferenceMarket *MyferenceMarketSession) Claim() (*types.Transaction, error) {
	return _MyferenceMarket.Contract.Claim(&_MyferenceMarket.TransactOpts)
}

// Claim is a paid mutator transaction binding the contract method 0x4e71d92d.
//
// Solidity: function claim() returns()
func (_MyferenceMarket *MyferenceMarketTransactorSession) Claim() (*types.Transaction, error) {
	return _MyferenceMarket.Contract.Claim(&_MyferenceMarket.TransactOpts)
}

// Deposit is a paid mutator transaction binding the contract method 0xd0e30db0.
//
// Solidity: function deposit() payable returns()
func (_MyferenceMarket *MyferenceMarketTransactor) Deposit(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MyferenceMarket.contract.Transact(opts, "deposit")
}

// Deposit is a paid mutator transaction binding the contract method 0xd0e30db0.
//
// Solidity: function deposit() payable returns()
func (_MyferenceMarket *MyferenceMarketSession) Deposit() (*types.Transaction, error) {
	return _MyferenceMarket.Contract.Deposit(&_MyferenceMarket.TransactOpts)
}

// Deposit is a paid mutator transaction binding the contract method 0xd0e30db0.
//
// Solidity: function deposit() payable returns()
func (_MyferenceMarket *MyferenceMarketTransactorSession) Deposit() (*types.Transaction, error) {
	return _MyferenceMarket.Contract.Deposit(&_MyferenceMarket.TransactOpts)
}

// DepositBond is a paid mutator transaction binding the contract method 0x741b3c39.
//
// Solidity: function depositBond() payable returns()
func (_MyferenceMarket *MyferenceMarketTransactor) DepositBond(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MyferenceMarket.contract.Transact(opts, "depositBond")
}

// DepositBond is a paid mutator transaction binding the contract method 0x741b3c39.
//
// Solidity: function depositBond() payable returns()
func (_MyferenceMarket *MyferenceMarketSession) DepositBond() (*types.Transaction, error) {
	return _MyferenceMarket.Contract.DepositBond(&_MyferenceMarket.TransactOpts)
}

// DepositBond is a paid mutator transaction binding the contract method 0x741b3c39.
//
// Solidity: function depositBond() payable returns()
func (_MyferenceMarket *MyferenceMarketTransactorSession) DepositBond() (*types.Transaction, error) {
	return _MyferenceMarket.Contract.DepositBond(&_MyferenceMarket.TransactOpts)
}

// ExecuteFeeChange is a paid mutator transaction binding the contract method 0xf79c6ea4.
//
// Solidity: function executeFeeChange() returns()
func (_MyferenceMarket *MyferenceMarketTransactor) ExecuteFeeChange(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MyferenceMarket.contract.Transact(opts, "executeFeeChange")
}

// ExecuteFeeChange is a paid mutator transaction binding the contract method 0xf79c6ea4.
//
// Solidity: function executeFeeChange() returns()
func (_MyferenceMarket *MyferenceMarketSession) ExecuteFeeChange() (*types.Transaction, error) {
	return _MyferenceMarket.Contract.ExecuteFeeChange(&_MyferenceMarket.TransactOpts)
}

// ExecuteFeeChange is a paid mutator transaction binding the contract method 0xf79c6ea4.
//
// Solidity: function executeFeeChange() returns()
func (_MyferenceMarket *MyferenceMarketTransactorSession) ExecuteFeeChange() (*types.Transaction, error) {
	return _MyferenceMarket.Contract.ExecuteFeeChange(&_MyferenceMarket.TransactOpts)
}

// FinalizeBondExit is a paid mutator transaction binding the contract method 0x69115cd7.
//
// Solidity: function finalizeBondExit() returns()
func (_MyferenceMarket *MyferenceMarketTransactor) FinalizeBondExit(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MyferenceMarket.contract.Transact(opts, "finalizeBondExit")
}

// FinalizeBondExit is a paid mutator transaction binding the contract method 0x69115cd7.
//
// Solidity: function finalizeBondExit() returns()
func (_MyferenceMarket *MyferenceMarketSession) FinalizeBondExit() (*types.Transaction, error) {
	return _MyferenceMarket.Contract.FinalizeBondExit(&_MyferenceMarket.TransactOpts)
}

// FinalizeBondExit is a paid mutator transaction binding the contract method 0x69115cd7.
//
// Solidity: function finalizeBondExit() returns()
func (_MyferenceMarket *MyferenceMarketTransactorSession) FinalizeBondExit() (*types.Transaction, error) {
	return _MyferenceMarket.Contract.FinalizeBondExit(&_MyferenceMarket.TransactOpts)
}

// FinalizeSessionClose is a paid mutator transaction binding the contract method 0x131620fc.
//
// Solidity: function finalizeSessionClose(bytes32 sessionId) returns()
func (_MyferenceMarket *MyferenceMarketTransactor) FinalizeSessionClose(opts *bind.TransactOpts, sessionId [32]byte) (*types.Transaction, error) {
	return _MyferenceMarket.contract.Transact(opts, "finalizeSessionClose", sessionId)
}

// FinalizeSessionClose is a paid mutator transaction binding the contract method 0x131620fc.
//
// Solidity: function finalizeSessionClose(bytes32 sessionId) returns()
func (_MyferenceMarket *MyferenceMarketSession) FinalizeSessionClose(sessionId [32]byte) (*types.Transaction, error) {
	return _MyferenceMarket.Contract.FinalizeSessionClose(&_MyferenceMarket.TransactOpts, sessionId)
}

// FinalizeSessionClose is a paid mutator transaction binding the contract method 0x131620fc.
//
// Solidity: function finalizeSessionClose(bytes32 sessionId) returns()
func (_MyferenceMarket *MyferenceMarketTransactorSession) FinalizeSessionClose(sessionId [32]byte) (*types.Transaction, error) {
	return _MyferenceMarket.Contract.FinalizeSessionClose(&_MyferenceMarket.TransactOpts, sessionId)
}

// OpenSession is a paid mutator transaction binding the contract method 0x3fb3adba.
//
// Solidity: function openSession(bytes32 sessionId, uint256 allowance, uint64 expiresAt) returns()
func (_MyferenceMarket *MyferenceMarketTransactor) OpenSession(opts *bind.TransactOpts, sessionId [32]byte, allowance *big.Int, expiresAt uint64) (*types.Transaction, error) {
	return _MyferenceMarket.contract.Transact(opts, "openSession", sessionId, allowance, expiresAt)
}

// OpenSession is a paid mutator transaction binding the contract method 0x3fb3adba.
//
// Solidity: function openSession(bytes32 sessionId, uint256 allowance, uint64 expiresAt) returns()
func (_MyferenceMarket *MyferenceMarketSession) OpenSession(sessionId [32]byte, allowance *big.Int, expiresAt uint64) (*types.Transaction, error) {
	return _MyferenceMarket.Contract.OpenSession(&_MyferenceMarket.TransactOpts, sessionId, allowance, expiresAt)
}

// OpenSession is a paid mutator transaction binding the contract method 0x3fb3adba.
//
// Solidity: function openSession(bytes32 sessionId, uint256 allowance, uint64 expiresAt) returns()
func (_MyferenceMarket *MyferenceMarketTransactorSession) OpenSession(sessionId [32]byte, allowance *big.Int, expiresAt uint64) (*types.Transaction, error) {
	return _MyferenceMarket.Contract.OpenSession(&_MyferenceMarket.TransactOpts, sessionId, allowance, expiresAt)
}

// Pause is a paid mutator transaction binding the contract method 0x8456cb59.
//
// Solidity: function pause() returns()
func (_MyferenceMarket *MyferenceMarketTransactor) Pause(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MyferenceMarket.contract.Transact(opts, "pause")
}

// Pause is a paid mutator transaction binding the contract method 0x8456cb59.
//
// Solidity: function pause() returns()
func (_MyferenceMarket *MyferenceMarketSession) Pause() (*types.Transaction, error) {
	return _MyferenceMarket.Contract.Pause(&_MyferenceMarket.TransactOpts)
}

// Pause is a paid mutator transaction binding the contract method 0x8456cb59.
//
// Solidity: function pause() returns()
func (_MyferenceMarket *MyferenceMarketTransactorSession) Pause() (*types.Transaction, error) {
	return _MyferenceMarket.Contract.Pause(&_MyferenceMarket.TransactOpts)
}

// ProposeFee is a paid mutator transaction binding the contract method 0xbc651f6c.
//
// Solidity: function proposeFee(uint16 newFeeBasisPoints) returns()
func (_MyferenceMarket *MyferenceMarketTransactor) ProposeFee(opts *bind.TransactOpts, newFeeBasisPoints uint16) (*types.Transaction, error) {
	return _MyferenceMarket.contract.Transact(opts, "proposeFee", newFeeBasisPoints)
}

// ProposeFee is a paid mutator transaction binding the contract method 0xbc651f6c.
//
// Solidity: function proposeFee(uint16 newFeeBasisPoints) returns()
func (_MyferenceMarket *MyferenceMarketSession) ProposeFee(newFeeBasisPoints uint16) (*types.Transaction, error) {
	return _MyferenceMarket.Contract.ProposeFee(&_MyferenceMarket.TransactOpts, newFeeBasisPoints)
}

// ProposeFee is a paid mutator transaction binding the contract method 0xbc651f6c.
//
// Solidity: function proposeFee(uint16 newFeeBasisPoints) returns()
func (_MyferenceMarket *MyferenceMarketTransactorSession) ProposeFee(newFeeBasisPoints uint16) (*types.Transaction, error) {
	return _MyferenceMarket.Contract.ProposeFee(&_MyferenceMarket.TransactOpts, newFeeBasisPoints)
}

// PublishOffer is a paid mutator transaction binding the contract method 0xebd9ce28.
//
// Solidity: function publishOffer(bytes32 offerId, bytes32 modelHash, bytes32 capabilityHash, uint256 inputPerMillion, uint256 outputPerMillion, uint256 computePerSecond) returns()
func (_MyferenceMarket *MyferenceMarketTransactor) PublishOffer(opts *bind.TransactOpts, offerId [32]byte, modelHash [32]byte, capabilityHash [32]byte, inputPerMillion *big.Int, outputPerMillion *big.Int, computePerSecond *big.Int) (*types.Transaction, error) {
	return _MyferenceMarket.contract.Transact(opts, "publishOffer", offerId, modelHash, capabilityHash, inputPerMillion, outputPerMillion, computePerSecond)
}

// PublishOffer is a paid mutator transaction binding the contract method 0xebd9ce28.
//
// Solidity: function publishOffer(bytes32 offerId, bytes32 modelHash, bytes32 capabilityHash, uint256 inputPerMillion, uint256 outputPerMillion, uint256 computePerSecond) returns()
func (_MyferenceMarket *MyferenceMarketSession) PublishOffer(offerId [32]byte, modelHash [32]byte, capabilityHash [32]byte, inputPerMillion *big.Int, outputPerMillion *big.Int, computePerSecond *big.Int) (*types.Transaction, error) {
	return _MyferenceMarket.Contract.PublishOffer(&_MyferenceMarket.TransactOpts, offerId, modelHash, capabilityHash, inputPerMillion, outputPerMillion, computePerSecond)
}

// PublishOffer is a paid mutator transaction binding the contract method 0xebd9ce28.
//
// Solidity: function publishOffer(bytes32 offerId, bytes32 modelHash, bytes32 capabilityHash, uint256 inputPerMillion, uint256 outputPerMillion, uint256 computePerSecond) returns()
func (_MyferenceMarket *MyferenceMarketTransactorSession) PublishOffer(offerId [32]byte, modelHash [32]byte, capabilityHash [32]byte, inputPerMillion *big.Int, outputPerMillion *big.Int, computePerSecond *big.Int) (*types.Transaction, error) {
	return _MyferenceMarket.Contract.PublishOffer(&_MyferenceMarket.TransactOpts, offerId, modelHash, capabilityHash, inputPerMillion, outputPerMillion, computePerSecond)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_MyferenceMarket *MyferenceMarketTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MyferenceMarket.contract.Transact(opts, "renounceOwnership")
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_MyferenceMarket *MyferenceMarketSession) RenounceOwnership() (*types.Transaction, error) {
	return _MyferenceMarket.Contract.RenounceOwnership(&_MyferenceMarket.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_MyferenceMarket *MyferenceMarketTransactorSession) RenounceOwnership() (*types.Transaction, error) {
	return _MyferenceMarket.Contract.RenounceOwnership(&_MyferenceMarket.TransactOpts)
}

// RequestBondExit is a paid mutator transaction binding the contract method 0x5a73384b.
//
// Solidity: function requestBondExit() returns()
func (_MyferenceMarket *MyferenceMarketTransactor) RequestBondExit(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MyferenceMarket.contract.Transact(opts, "requestBondExit")
}

// RequestBondExit is a paid mutator transaction binding the contract method 0x5a73384b.
//
// Solidity: function requestBondExit() returns()
func (_MyferenceMarket *MyferenceMarketSession) RequestBondExit() (*types.Transaction, error) {
	return _MyferenceMarket.Contract.RequestBondExit(&_MyferenceMarket.TransactOpts)
}

// RequestBondExit is a paid mutator transaction binding the contract method 0x5a73384b.
//
// Solidity: function requestBondExit() returns()
func (_MyferenceMarket *MyferenceMarketTransactorSession) RequestBondExit() (*types.Transaction, error) {
	return _MyferenceMarket.Contract.RequestBondExit(&_MyferenceMarket.TransactOpts)
}

// RequestSessionClose is a paid mutator transaction binding the contract method 0x9d9fabc8.
//
// Solidity: function requestSessionClose(bytes32 sessionId) returns()
func (_MyferenceMarket *MyferenceMarketTransactor) RequestSessionClose(opts *bind.TransactOpts, sessionId [32]byte) (*types.Transaction, error) {
	return _MyferenceMarket.contract.Transact(opts, "requestSessionClose", sessionId)
}

// RequestSessionClose is a paid mutator transaction binding the contract method 0x9d9fabc8.
//
// Solidity: function requestSessionClose(bytes32 sessionId) returns()
func (_MyferenceMarket *MyferenceMarketSession) RequestSessionClose(sessionId [32]byte) (*types.Transaction, error) {
	return _MyferenceMarket.Contract.RequestSessionClose(&_MyferenceMarket.TransactOpts, sessionId)
}

// RequestSessionClose is a paid mutator transaction binding the contract method 0x9d9fabc8.
//
// Solidity: function requestSessionClose(bytes32 sessionId) returns()
func (_MyferenceMarket *MyferenceMarketTransactorSession) RequestSessionClose(sessionId [32]byte) (*types.Transaction, error) {
	return _MyferenceMarket.Contract.RequestSessionClose(&_MyferenceMarket.TransactOpts, sessionId)
}

// RequestWithdrawal is a paid mutator transaction binding the contract method 0x9ee679e8.
//
// Solidity: function requestWithdrawal(uint256 amount) returns()
func (_MyferenceMarket *MyferenceMarketTransactor) RequestWithdrawal(opts *bind.TransactOpts, amount *big.Int) (*types.Transaction, error) {
	return _MyferenceMarket.contract.Transact(opts, "requestWithdrawal", amount)
}

// RequestWithdrawal is a paid mutator transaction binding the contract method 0x9ee679e8.
//
// Solidity: function requestWithdrawal(uint256 amount) returns()
func (_MyferenceMarket *MyferenceMarketSession) RequestWithdrawal(amount *big.Int) (*types.Transaction, error) {
	return _MyferenceMarket.Contract.RequestWithdrawal(&_MyferenceMarket.TransactOpts, amount)
}

// RequestWithdrawal is a paid mutator transaction binding the contract method 0x9ee679e8.
//
// Solidity: function requestWithdrawal(uint256 amount) returns()
func (_MyferenceMarket *MyferenceMarketTransactorSession) RequestWithdrawal(amount *big.Int) (*types.Transaction, error) {
	return _MyferenceMarket.Contract.RequestWithdrawal(&_MyferenceMarket.TransactOpts, amount)
}

// SettleReceipt is a paid mutator transaction binding the contract method 0xa146e24b.
//
// Solidity: function settleReceipt((bytes32,bytes32,address,address,address,bytes32,uint64,bytes32,bytes32,uint64,uint64,uint64,uint64,uint64,uint16,uint64,uint8,uint64,bytes32,bytes32,uint64) receipt, bytes providerSignature, bytes settlementSignature) returns()
func (_MyferenceMarket *MyferenceMarketTransactor) SettleReceipt(opts *bind.TransactOpts, receipt MyferenceMarketReceipt, providerSignature []byte, settlementSignature []byte) (*types.Transaction, error) {
	return _MyferenceMarket.contract.Transact(opts, "settleReceipt", receipt, providerSignature, settlementSignature)
}

// SettleReceipt is a paid mutator transaction binding the contract method 0xa146e24b.
//
// Solidity: function settleReceipt((bytes32,bytes32,address,address,address,bytes32,uint64,bytes32,bytes32,uint64,uint64,uint64,uint64,uint64,uint16,uint64,uint8,uint64,bytes32,bytes32,uint64) receipt, bytes providerSignature, bytes settlementSignature) returns()
func (_MyferenceMarket *MyferenceMarketSession) SettleReceipt(receipt MyferenceMarketReceipt, providerSignature []byte, settlementSignature []byte) (*types.Transaction, error) {
	return _MyferenceMarket.Contract.SettleReceipt(&_MyferenceMarket.TransactOpts, receipt, providerSignature, settlementSignature)
}

// SettleReceipt is a paid mutator transaction binding the contract method 0xa146e24b.
//
// Solidity: function settleReceipt((bytes32,bytes32,address,address,address,bytes32,uint64,bytes32,bytes32,uint64,uint64,uint64,uint64,uint64,uint16,uint64,uint8,uint64,bytes32,bytes32,uint64) receipt, bytes providerSignature, bytes settlementSignature) returns()
func (_MyferenceMarket *MyferenceMarketTransactorSession) SettleReceipt(receipt MyferenceMarketReceipt, providerSignature []byte, settlementSignature []byte) (*types.Transaction, error) {
	return _MyferenceMarket.Contract.SettleReceipt(&_MyferenceMarket.TransactOpts, receipt, providerSignature, settlementSignature)
}

// SettleReceipts is a paid mutator transaction binding the contract method 0xe6dd0791.
//
// Solidity: function settleReceipts((bytes32,bytes32,address,address,address,bytes32,uint64,bytes32,bytes32,uint64,uint64,uint64,uint64,uint64,uint16,uint64,uint8,uint64,bytes32,bytes32,uint64)[] receipts, bytes[] providerSignatures, bytes[] settlementSignatures) returns()
func (_MyferenceMarket *MyferenceMarketTransactor) SettleReceipts(opts *bind.TransactOpts, receipts []MyferenceMarketReceipt, providerSignatures [][]byte, settlementSignatures [][]byte) (*types.Transaction, error) {
	return _MyferenceMarket.contract.Transact(opts, "settleReceipts", receipts, providerSignatures, settlementSignatures)
}

// SettleReceipts is a paid mutator transaction binding the contract method 0xe6dd0791.
//
// Solidity: function settleReceipts((bytes32,bytes32,address,address,address,bytes32,uint64,bytes32,bytes32,uint64,uint64,uint64,uint64,uint64,uint16,uint64,uint8,uint64,bytes32,bytes32,uint64)[] receipts, bytes[] providerSignatures, bytes[] settlementSignatures) returns()
func (_MyferenceMarket *MyferenceMarketSession) SettleReceipts(receipts []MyferenceMarketReceipt, providerSignatures [][]byte, settlementSignatures [][]byte) (*types.Transaction, error) {
	return _MyferenceMarket.Contract.SettleReceipts(&_MyferenceMarket.TransactOpts, receipts, providerSignatures, settlementSignatures)
}

// SettleReceipts is a paid mutator transaction binding the contract method 0xe6dd0791.
//
// Solidity: function settleReceipts((bytes32,bytes32,address,address,address,bytes32,uint64,bytes32,bytes32,uint64,uint64,uint64,uint64,uint64,uint16,uint64,uint8,uint64,bytes32,bytes32,uint64)[] receipts, bytes[] providerSignatures, bytes[] settlementSignatures) returns()
func (_MyferenceMarket *MyferenceMarketTransactorSession) SettleReceipts(receipts []MyferenceMarketReceipt, providerSignatures [][]byte, settlementSignatures [][]byte) (*types.Transaction, error) {
	return _MyferenceMarket.Contract.SettleReceipts(&_MyferenceMarket.TransactOpts, receipts, providerSignatures, settlementSignatures)
}

// SlashDoubleSign is a paid mutator transaction binding the contract method 0xa7abbc3c.
//
// Solidity: function slashDoubleSign((bytes32,bytes32,address,address,address,bytes32,uint64,bytes32,bytes32,uint64,uint64,uint64,uint64,uint64,uint16,uint64,uint8,uint64,bytes32,bytes32,uint64) first, bytes firstSignature, (bytes32,bytes32,address,address,address,bytes32,uint64,bytes32,bytes32,uint64,uint64,uint64,uint64,uint64,uint16,uint64,uint8,uint64,bytes32,bytes32,uint64) second, bytes secondSignature) returns()
func (_MyferenceMarket *MyferenceMarketTransactor) SlashDoubleSign(opts *bind.TransactOpts, first MyferenceMarketReceipt, firstSignature []byte, second MyferenceMarketReceipt, secondSignature []byte) (*types.Transaction, error) {
	return _MyferenceMarket.contract.Transact(opts, "slashDoubleSign", first, firstSignature, second, secondSignature)
}

// SlashDoubleSign is a paid mutator transaction binding the contract method 0xa7abbc3c.
//
// Solidity: function slashDoubleSign((bytes32,bytes32,address,address,address,bytes32,uint64,bytes32,bytes32,uint64,uint64,uint64,uint64,uint64,uint16,uint64,uint8,uint64,bytes32,bytes32,uint64) first, bytes firstSignature, (bytes32,bytes32,address,address,address,bytes32,uint64,bytes32,bytes32,uint64,uint64,uint64,uint64,uint64,uint16,uint64,uint8,uint64,bytes32,bytes32,uint64) second, bytes secondSignature) returns()
func (_MyferenceMarket *MyferenceMarketSession) SlashDoubleSign(first MyferenceMarketReceipt, firstSignature []byte, second MyferenceMarketReceipt, secondSignature []byte) (*types.Transaction, error) {
	return _MyferenceMarket.Contract.SlashDoubleSign(&_MyferenceMarket.TransactOpts, first, firstSignature, second, secondSignature)
}

// SlashDoubleSign is a paid mutator transaction binding the contract method 0xa7abbc3c.
//
// Solidity: function slashDoubleSign((bytes32,bytes32,address,address,address,bytes32,uint64,bytes32,bytes32,uint64,uint64,uint64,uint64,uint64,uint16,uint64,uint8,uint64,bytes32,bytes32,uint64) first, bytes firstSignature, (bytes32,bytes32,address,address,address,bytes32,uint64,bytes32,bytes32,uint64,uint64,uint64,uint64,uint64,uint16,uint64,uint8,uint64,bytes32,bytes32,uint64) second, bytes secondSignature) returns()
func (_MyferenceMarket *MyferenceMarketTransactorSession) SlashDoubleSign(first MyferenceMarketReceipt, firstSignature []byte, second MyferenceMarketReceipt, secondSignature []byte) (*types.Transaction, error) {
	return _MyferenceMarket.Contract.SlashDoubleSign(&_MyferenceMarket.TransactOpts, first, firstSignature, second, secondSignature)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_MyferenceMarket *MyferenceMarketTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _MyferenceMarket.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_MyferenceMarket *MyferenceMarketSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _MyferenceMarket.Contract.TransferOwnership(&_MyferenceMarket.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_MyferenceMarket *MyferenceMarketTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _MyferenceMarket.Contract.TransferOwnership(&_MyferenceMarket.TransactOpts, newOwner)
}

// Unpause is a paid mutator transaction binding the contract method 0x3f4ba83a.
//
// Solidity: function unpause() returns()
func (_MyferenceMarket *MyferenceMarketTransactor) Unpause(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MyferenceMarket.contract.Transact(opts, "unpause")
}

// Unpause is a paid mutator transaction binding the contract method 0x3f4ba83a.
//
// Solidity: function unpause() returns()
func (_MyferenceMarket *MyferenceMarketSession) Unpause() (*types.Transaction, error) {
	return _MyferenceMarket.Contract.Unpause(&_MyferenceMarket.TransactOpts)
}

// Unpause is a paid mutator transaction binding the contract method 0x3f4ba83a.
//
// Solidity: function unpause() returns()
func (_MyferenceMarket *MyferenceMarketTransactorSession) Unpause() (*types.Transaction, error) {
	return _MyferenceMarket.Contract.Unpause(&_MyferenceMarket.TransactOpts)
}

// MyferenceMarketBondDepositedIterator is returned from FilterBondDeposited and is used to iterate over the raw logs and unpacked data for BondDeposited events raised by the MyferenceMarket contract.
type MyferenceMarketBondDepositedIterator struct {
	Event *MyferenceMarketBondDeposited // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *MyferenceMarketBondDepositedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(MyferenceMarketBondDeposited)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(MyferenceMarketBondDeposited)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *MyferenceMarketBondDepositedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *MyferenceMarketBondDepositedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// MyferenceMarketBondDeposited represents a BondDeposited event raised by the MyferenceMarket contract.
type MyferenceMarketBondDeposited struct {
	Provider  common.Address
	Amount    *big.Int
	TotalBond *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterBondDeposited is a free log retrieval operation binding the contract event 0x53709ba3cdfc6888ee4e1f05692566dfc3446e917fd4402db50e328307e6b393.
//
// Solidity: event BondDeposited(address indexed provider, uint256 amount, uint256 totalBond)
func (_MyferenceMarket *MyferenceMarketFilterer) FilterBondDeposited(opts *bind.FilterOpts, provider []common.Address) (*MyferenceMarketBondDepositedIterator, error) {

	var providerRule []interface{}
	for _, providerItem := range provider {
		providerRule = append(providerRule, providerItem)
	}

	logs, sub, err := _MyferenceMarket.contract.FilterLogs(opts, "BondDeposited", providerRule)
	if err != nil {
		return nil, err
	}
	return &MyferenceMarketBondDepositedIterator{contract: _MyferenceMarket.contract, event: "BondDeposited", logs: logs, sub: sub}, nil
}

// WatchBondDeposited is a free log subscription operation binding the contract event 0x53709ba3cdfc6888ee4e1f05692566dfc3446e917fd4402db50e328307e6b393.
//
// Solidity: event BondDeposited(address indexed provider, uint256 amount, uint256 totalBond)
func (_MyferenceMarket *MyferenceMarketFilterer) WatchBondDeposited(opts *bind.WatchOpts, sink chan<- *MyferenceMarketBondDeposited, provider []common.Address) (event.Subscription, error) {

	var providerRule []interface{}
	for _, providerItem := range provider {
		providerRule = append(providerRule, providerItem)
	}

	logs, sub, err := _MyferenceMarket.contract.WatchLogs(opts, "BondDeposited", providerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(MyferenceMarketBondDeposited)
				if err := _MyferenceMarket.contract.UnpackLog(event, "BondDeposited", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseBondDeposited is a log parse operation binding the contract event 0x53709ba3cdfc6888ee4e1f05692566dfc3446e917fd4402db50e328307e6b393.
//
// Solidity: event BondDeposited(address indexed provider, uint256 amount, uint256 totalBond)
func (_MyferenceMarket *MyferenceMarketFilterer) ParseBondDeposited(log types.Log) (*MyferenceMarketBondDeposited, error) {
	event := new(MyferenceMarketBondDeposited)
	if err := _MyferenceMarket.contract.UnpackLog(event, "BondDeposited", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// MyferenceMarketBondExitFinalizedIterator is returned from FilterBondExitFinalized and is used to iterate over the raw logs and unpacked data for BondExitFinalized events raised by the MyferenceMarket contract.
type MyferenceMarketBondExitFinalizedIterator struct {
	Event *MyferenceMarketBondExitFinalized // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *MyferenceMarketBondExitFinalizedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(MyferenceMarketBondExitFinalized)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(MyferenceMarketBondExitFinalized)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *MyferenceMarketBondExitFinalizedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *MyferenceMarketBondExitFinalizedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// MyferenceMarketBondExitFinalized represents a BondExitFinalized event raised by the MyferenceMarket contract.
type MyferenceMarketBondExitFinalized struct {
	Provider common.Address
	Amount   *big.Int
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterBondExitFinalized is a free log retrieval operation binding the contract event 0x23c59c6738d789e373cb3f3359278f32ec1227766afbd94405fd92217aaa74dd.
//
// Solidity: event BondExitFinalized(address indexed provider, uint256 amount)
func (_MyferenceMarket *MyferenceMarketFilterer) FilterBondExitFinalized(opts *bind.FilterOpts, provider []common.Address) (*MyferenceMarketBondExitFinalizedIterator, error) {

	var providerRule []interface{}
	for _, providerItem := range provider {
		providerRule = append(providerRule, providerItem)
	}

	logs, sub, err := _MyferenceMarket.contract.FilterLogs(opts, "BondExitFinalized", providerRule)
	if err != nil {
		return nil, err
	}
	return &MyferenceMarketBondExitFinalizedIterator{contract: _MyferenceMarket.contract, event: "BondExitFinalized", logs: logs, sub: sub}, nil
}

// WatchBondExitFinalized is a free log subscription operation binding the contract event 0x23c59c6738d789e373cb3f3359278f32ec1227766afbd94405fd92217aaa74dd.
//
// Solidity: event BondExitFinalized(address indexed provider, uint256 amount)
func (_MyferenceMarket *MyferenceMarketFilterer) WatchBondExitFinalized(opts *bind.WatchOpts, sink chan<- *MyferenceMarketBondExitFinalized, provider []common.Address) (event.Subscription, error) {

	var providerRule []interface{}
	for _, providerItem := range provider {
		providerRule = append(providerRule, providerItem)
	}

	logs, sub, err := _MyferenceMarket.contract.WatchLogs(opts, "BondExitFinalized", providerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(MyferenceMarketBondExitFinalized)
				if err := _MyferenceMarket.contract.UnpackLog(event, "BondExitFinalized", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseBondExitFinalized is a log parse operation binding the contract event 0x23c59c6738d789e373cb3f3359278f32ec1227766afbd94405fd92217aaa74dd.
//
// Solidity: event BondExitFinalized(address indexed provider, uint256 amount)
func (_MyferenceMarket *MyferenceMarketFilterer) ParseBondExitFinalized(log types.Log) (*MyferenceMarketBondExitFinalized, error) {
	event := new(MyferenceMarketBondExitFinalized)
	if err := _MyferenceMarket.contract.UnpackLog(event, "BondExitFinalized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// MyferenceMarketBondExitRequestedIterator is returned from FilterBondExitRequested and is used to iterate over the raw logs and unpacked data for BondExitRequested events raised by the MyferenceMarket contract.
type MyferenceMarketBondExitRequestedIterator struct {
	Event *MyferenceMarketBondExitRequested // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *MyferenceMarketBondExitRequestedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(MyferenceMarketBondExitRequested)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(MyferenceMarketBondExitRequested)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *MyferenceMarketBondExitRequestedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *MyferenceMarketBondExitRequestedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// MyferenceMarketBondExitRequested represents a BondExitRequested event raised by the MyferenceMarket contract.
type MyferenceMarketBondExitRequested struct {
	Provider    common.Address
	AvailableAt uint64
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterBondExitRequested is a free log retrieval operation binding the contract event 0x2d558b0c7f59f19033dbb6e56acc49e254d2b1f528168d83a35a1f5255f60247.
//
// Solidity: event BondExitRequested(address indexed provider, uint64 availableAt)
func (_MyferenceMarket *MyferenceMarketFilterer) FilterBondExitRequested(opts *bind.FilterOpts, provider []common.Address) (*MyferenceMarketBondExitRequestedIterator, error) {

	var providerRule []interface{}
	for _, providerItem := range provider {
		providerRule = append(providerRule, providerItem)
	}

	logs, sub, err := _MyferenceMarket.contract.FilterLogs(opts, "BondExitRequested", providerRule)
	if err != nil {
		return nil, err
	}
	return &MyferenceMarketBondExitRequestedIterator{contract: _MyferenceMarket.contract, event: "BondExitRequested", logs: logs, sub: sub}, nil
}

// WatchBondExitRequested is a free log subscription operation binding the contract event 0x2d558b0c7f59f19033dbb6e56acc49e254d2b1f528168d83a35a1f5255f60247.
//
// Solidity: event BondExitRequested(address indexed provider, uint64 availableAt)
func (_MyferenceMarket *MyferenceMarketFilterer) WatchBondExitRequested(opts *bind.WatchOpts, sink chan<- *MyferenceMarketBondExitRequested, provider []common.Address) (event.Subscription, error) {

	var providerRule []interface{}
	for _, providerItem := range provider {
		providerRule = append(providerRule, providerItem)
	}

	logs, sub, err := _MyferenceMarket.contract.WatchLogs(opts, "BondExitRequested", providerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(MyferenceMarketBondExitRequested)
				if err := _MyferenceMarket.contract.UnpackLog(event, "BondExitRequested", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseBondExitRequested is a log parse operation binding the contract event 0x2d558b0c7f59f19033dbb6e56acc49e254d2b1f528168d83a35a1f5255f60247.
//
// Solidity: event BondExitRequested(address indexed provider, uint64 availableAt)
func (_MyferenceMarket *MyferenceMarketFilterer) ParseBondExitRequested(log types.Log) (*MyferenceMarketBondExitRequested, error) {
	event := new(MyferenceMarketBondExitRequested)
	if err := _MyferenceMarket.contract.UnpackLog(event, "BondExitRequested", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// MyferenceMarketClaimedIterator is returned from FilterClaimed and is used to iterate over the raw logs and unpacked data for Claimed events raised by the MyferenceMarket contract.
type MyferenceMarketClaimedIterator struct {
	Event *MyferenceMarketClaimed // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *MyferenceMarketClaimedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(MyferenceMarketClaimed)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(MyferenceMarketClaimed)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *MyferenceMarketClaimedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *MyferenceMarketClaimedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// MyferenceMarketClaimed represents a Claimed event raised by the MyferenceMarket contract.
type MyferenceMarketClaimed struct {
	Account common.Address
	Amount  *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterClaimed is a free log retrieval operation binding the contract event 0xd8138f8a3f377c5259ca548e70e4c2de94f129f5a11036a15b69513cba2b426a.
//
// Solidity: event Claimed(address indexed account, uint256 amount)
func (_MyferenceMarket *MyferenceMarketFilterer) FilterClaimed(opts *bind.FilterOpts, account []common.Address) (*MyferenceMarketClaimedIterator, error) {

	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}

	logs, sub, err := _MyferenceMarket.contract.FilterLogs(opts, "Claimed", accountRule)
	if err != nil {
		return nil, err
	}
	return &MyferenceMarketClaimedIterator{contract: _MyferenceMarket.contract, event: "Claimed", logs: logs, sub: sub}, nil
}

// WatchClaimed is a free log subscription operation binding the contract event 0xd8138f8a3f377c5259ca548e70e4c2de94f129f5a11036a15b69513cba2b426a.
//
// Solidity: event Claimed(address indexed account, uint256 amount)
func (_MyferenceMarket *MyferenceMarketFilterer) WatchClaimed(opts *bind.WatchOpts, sink chan<- *MyferenceMarketClaimed, account []common.Address) (event.Subscription, error) {

	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}

	logs, sub, err := _MyferenceMarket.contract.WatchLogs(opts, "Claimed", accountRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(MyferenceMarketClaimed)
				if err := _MyferenceMarket.contract.UnpackLog(event, "Claimed", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseClaimed is a log parse operation binding the contract event 0xd8138f8a3f377c5259ca548e70e4c2de94f129f5a11036a15b69513cba2b426a.
//
// Solidity: event Claimed(address indexed account, uint256 amount)
func (_MyferenceMarket *MyferenceMarketFilterer) ParseClaimed(log types.Log) (*MyferenceMarketClaimed, error) {
	event := new(MyferenceMarketClaimed)
	if err := _MyferenceMarket.contract.UnpackLog(event, "Claimed", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// MyferenceMarketDepositedIterator is returned from FilterDeposited and is used to iterate over the raw logs and unpacked data for Deposited events raised by the MyferenceMarket contract.
type MyferenceMarketDepositedIterator struct {
	Event *MyferenceMarketDeposited // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *MyferenceMarketDepositedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(MyferenceMarketDeposited)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(MyferenceMarketDeposited)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *MyferenceMarketDepositedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *MyferenceMarketDepositedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// MyferenceMarketDeposited represents a Deposited event raised by the MyferenceMarket contract.
type MyferenceMarketDeposited struct {
	Customer common.Address
	Amount   *big.Int
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterDeposited is a free log retrieval operation binding the contract event 0x2da466a7b24304f47e87fa2e1e5a81b9831ce54fec19055ce277ca2f39ba42c4.
//
// Solidity: event Deposited(address indexed customer, uint256 amount)
func (_MyferenceMarket *MyferenceMarketFilterer) FilterDeposited(opts *bind.FilterOpts, customer []common.Address) (*MyferenceMarketDepositedIterator, error) {

	var customerRule []interface{}
	for _, customerItem := range customer {
		customerRule = append(customerRule, customerItem)
	}

	logs, sub, err := _MyferenceMarket.contract.FilterLogs(opts, "Deposited", customerRule)
	if err != nil {
		return nil, err
	}
	return &MyferenceMarketDepositedIterator{contract: _MyferenceMarket.contract, event: "Deposited", logs: logs, sub: sub}, nil
}

// WatchDeposited is a free log subscription operation binding the contract event 0x2da466a7b24304f47e87fa2e1e5a81b9831ce54fec19055ce277ca2f39ba42c4.
//
// Solidity: event Deposited(address indexed customer, uint256 amount)
func (_MyferenceMarket *MyferenceMarketFilterer) WatchDeposited(opts *bind.WatchOpts, sink chan<- *MyferenceMarketDeposited, customer []common.Address) (event.Subscription, error) {

	var customerRule []interface{}
	for _, customerItem := range customer {
		customerRule = append(customerRule, customerItem)
	}

	logs, sub, err := _MyferenceMarket.contract.WatchLogs(opts, "Deposited", customerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(MyferenceMarketDeposited)
				if err := _MyferenceMarket.contract.UnpackLog(event, "Deposited", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseDeposited is a log parse operation binding the contract event 0x2da466a7b24304f47e87fa2e1e5a81b9831ce54fec19055ce277ca2f39ba42c4.
//
// Solidity: event Deposited(address indexed customer, uint256 amount)
func (_MyferenceMarket *MyferenceMarketFilterer) ParseDeposited(log types.Log) (*MyferenceMarketDeposited, error) {
	event := new(MyferenceMarketDeposited)
	if err := _MyferenceMarket.contract.UnpackLog(event, "Deposited", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// MyferenceMarketEIP712DomainChangedIterator is returned from FilterEIP712DomainChanged and is used to iterate over the raw logs and unpacked data for EIP712DomainChanged events raised by the MyferenceMarket contract.
type MyferenceMarketEIP712DomainChangedIterator struct {
	Event *MyferenceMarketEIP712DomainChanged // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *MyferenceMarketEIP712DomainChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(MyferenceMarketEIP712DomainChanged)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(MyferenceMarketEIP712DomainChanged)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *MyferenceMarketEIP712DomainChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *MyferenceMarketEIP712DomainChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// MyferenceMarketEIP712DomainChanged represents a EIP712DomainChanged event raised by the MyferenceMarket contract.
type MyferenceMarketEIP712DomainChanged struct {
	Raw types.Log // Blockchain specific contextual infos
}

// FilterEIP712DomainChanged is a free log retrieval operation binding the contract event 0x0a6387c9ea3628b88a633bb4f3b151770f70085117a15f9bf3787cda53f13d31.
//
// Solidity: event EIP712DomainChanged()
func (_MyferenceMarket *MyferenceMarketFilterer) FilterEIP712DomainChanged(opts *bind.FilterOpts) (*MyferenceMarketEIP712DomainChangedIterator, error) {

	logs, sub, err := _MyferenceMarket.contract.FilterLogs(opts, "EIP712DomainChanged")
	if err != nil {
		return nil, err
	}
	return &MyferenceMarketEIP712DomainChangedIterator{contract: _MyferenceMarket.contract, event: "EIP712DomainChanged", logs: logs, sub: sub}, nil
}

// WatchEIP712DomainChanged is a free log subscription operation binding the contract event 0x0a6387c9ea3628b88a633bb4f3b151770f70085117a15f9bf3787cda53f13d31.
//
// Solidity: event EIP712DomainChanged()
func (_MyferenceMarket *MyferenceMarketFilterer) WatchEIP712DomainChanged(opts *bind.WatchOpts, sink chan<- *MyferenceMarketEIP712DomainChanged) (event.Subscription, error) {

	logs, sub, err := _MyferenceMarket.contract.WatchLogs(opts, "EIP712DomainChanged")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(MyferenceMarketEIP712DomainChanged)
				if err := _MyferenceMarket.contract.UnpackLog(event, "EIP712DomainChanged", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseEIP712DomainChanged is a log parse operation binding the contract event 0x0a6387c9ea3628b88a633bb4f3b151770f70085117a15f9bf3787cda53f13d31.
//
// Solidity: event EIP712DomainChanged()
func (_MyferenceMarket *MyferenceMarketFilterer) ParseEIP712DomainChanged(log types.Log) (*MyferenceMarketEIP712DomainChanged, error) {
	event := new(MyferenceMarketEIP712DomainChanged)
	if err := _MyferenceMarket.contract.UnpackLog(event, "EIP712DomainChanged", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// MyferenceMarketFeeChangedIterator is returned from FilterFeeChanged and is used to iterate over the raw logs and unpacked data for FeeChanged events raised by the MyferenceMarket contract.
type MyferenceMarketFeeChangedIterator struct {
	Event *MyferenceMarketFeeChanged // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *MyferenceMarketFeeChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(MyferenceMarketFeeChanged)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(MyferenceMarketFeeChanged)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *MyferenceMarketFeeChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *MyferenceMarketFeeChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// MyferenceMarketFeeChanged represents a FeeChanged event raised by the MyferenceMarket contract.
type MyferenceMarketFeeChanged struct {
	FeeBasisPoints uint16
	Version        uint64
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterFeeChanged is a free log retrieval operation binding the contract event 0x45e84326792e05741cc87e8bafc910ce7a953625da7c055b3b49cbfb1eb5a830.
//
// Solidity: event FeeChanged(uint16 feeBasisPoints, uint64 version)
func (_MyferenceMarket *MyferenceMarketFilterer) FilterFeeChanged(opts *bind.FilterOpts) (*MyferenceMarketFeeChangedIterator, error) {

	logs, sub, err := _MyferenceMarket.contract.FilterLogs(opts, "FeeChanged")
	if err != nil {
		return nil, err
	}
	return &MyferenceMarketFeeChangedIterator{contract: _MyferenceMarket.contract, event: "FeeChanged", logs: logs, sub: sub}, nil
}

// WatchFeeChanged is a free log subscription operation binding the contract event 0x45e84326792e05741cc87e8bafc910ce7a953625da7c055b3b49cbfb1eb5a830.
//
// Solidity: event FeeChanged(uint16 feeBasisPoints, uint64 version)
func (_MyferenceMarket *MyferenceMarketFilterer) WatchFeeChanged(opts *bind.WatchOpts, sink chan<- *MyferenceMarketFeeChanged) (event.Subscription, error) {

	logs, sub, err := _MyferenceMarket.contract.WatchLogs(opts, "FeeChanged")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(MyferenceMarketFeeChanged)
				if err := _MyferenceMarket.contract.UnpackLog(event, "FeeChanged", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseFeeChanged is a log parse operation binding the contract event 0x45e84326792e05741cc87e8bafc910ce7a953625da7c055b3b49cbfb1eb5a830.
//
// Solidity: event FeeChanged(uint16 feeBasisPoints, uint64 version)
func (_MyferenceMarket *MyferenceMarketFilterer) ParseFeeChanged(log types.Log) (*MyferenceMarketFeeChanged, error) {
	event := new(MyferenceMarketFeeChanged)
	if err := _MyferenceMarket.contract.UnpackLog(event, "FeeChanged", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// MyferenceMarketFeeProposedIterator is returned from FilterFeeProposed and is used to iterate over the raw logs and unpacked data for FeeProposed events raised by the MyferenceMarket contract.
type MyferenceMarketFeeProposedIterator struct {
	Event *MyferenceMarketFeeProposed // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *MyferenceMarketFeeProposedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(MyferenceMarketFeeProposed)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(MyferenceMarketFeeProposed)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *MyferenceMarketFeeProposedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *MyferenceMarketFeeProposedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// MyferenceMarketFeeProposed represents a FeeProposed event raised by the MyferenceMarket contract.
type MyferenceMarketFeeProposed struct {
	FeeBasisPoints uint16
	AvailableAt    uint64
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterFeeProposed is a free log retrieval operation binding the contract event 0x99b204b1e2687a443690dc3caa13da2a50fb2668ad5b999f91257f23c75e4310.
//
// Solidity: event FeeProposed(uint16 feeBasisPoints, uint64 availableAt)
func (_MyferenceMarket *MyferenceMarketFilterer) FilterFeeProposed(opts *bind.FilterOpts) (*MyferenceMarketFeeProposedIterator, error) {

	logs, sub, err := _MyferenceMarket.contract.FilterLogs(opts, "FeeProposed")
	if err != nil {
		return nil, err
	}
	return &MyferenceMarketFeeProposedIterator{contract: _MyferenceMarket.contract, event: "FeeProposed", logs: logs, sub: sub}, nil
}

// WatchFeeProposed is a free log subscription operation binding the contract event 0x99b204b1e2687a443690dc3caa13da2a50fb2668ad5b999f91257f23c75e4310.
//
// Solidity: event FeeProposed(uint16 feeBasisPoints, uint64 availableAt)
func (_MyferenceMarket *MyferenceMarketFilterer) WatchFeeProposed(opts *bind.WatchOpts, sink chan<- *MyferenceMarketFeeProposed) (event.Subscription, error) {

	logs, sub, err := _MyferenceMarket.contract.WatchLogs(opts, "FeeProposed")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(MyferenceMarketFeeProposed)
				if err := _MyferenceMarket.contract.UnpackLog(event, "FeeProposed", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseFeeProposed is a log parse operation binding the contract event 0x99b204b1e2687a443690dc3caa13da2a50fb2668ad5b999f91257f23c75e4310.
//
// Solidity: event FeeProposed(uint16 feeBasisPoints, uint64 availableAt)
func (_MyferenceMarket *MyferenceMarketFilterer) ParseFeeProposed(log types.Log) (*MyferenceMarketFeeProposed, error) {
	event := new(MyferenceMarketFeeProposed)
	if err := _MyferenceMarket.contract.UnpackLog(event, "FeeProposed", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// MyferenceMarketOfferPublishedIterator is returned from FilterOfferPublished and is used to iterate over the raw logs and unpacked data for OfferPublished events raised by the MyferenceMarket contract.
type MyferenceMarketOfferPublishedIterator struct {
	Event *MyferenceMarketOfferPublished // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *MyferenceMarketOfferPublishedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(MyferenceMarketOfferPublished)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(MyferenceMarketOfferPublished)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *MyferenceMarketOfferPublishedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *MyferenceMarketOfferPublishedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// MyferenceMarketOfferPublished represents a OfferPublished event raised by the MyferenceMarket contract.
type MyferenceMarketOfferPublished struct {
	Provider         common.Address
	OfferId          [32]byte
	Version          uint64
	ModelHash        [32]byte
	CapabilityHash   [32]byte
	InputPerMillion  *big.Int
	OutputPerMillion *big.Int
	ComputePerSecond *big.Int
	Raw              types.Log // Blockchain specific contextual infos
}

// FilterOfferPublished is a free log retrieval operation binding the contract event 0x5ae57c6c471f08a53556a92bfe0656a3256d5712e6bb87c07294902c46ee99e3.
//
// Solidity: event OfferPublished(address indexed provider, bytes32 indexed offerId, uint64 indexed version, bytes32 modelHash, bytes32 capabilityHash, uint256 inputPerMillion, uint256 outputPerMillion, uint256 computePerSecond)
func (_MyferenceMarket *MyferenceMarketFilterer) FilterOfferPublished(opts *bind.FilterOpts, provider []common.Address, offerId [][32]byte, version []uint64) (*MyferenceMarketOfferPublishedIterator, error) {

	var providerRule []interface{}
	for _, providerItem := range provider {
		providerRule = append(providerRule, providerItem)
	}
	var offerIdRule []interface{}
	for _, offerIdItem := range offerId {
		offerIdRule = append(offerIdRule, offerIdItem)
	}
	var versionRule []interface{}
	for _, versionItem := range version {
		versionRule = append(versionRule, versionItem)
	}

	logs, sub, err := _MyferenceMarket.contract.FilterLogs(opts, "OfferPublished", providerRule, offerIdRule, versionRule)
	if err != nil {
		return nil, err
	}
	return &MyferenceMarketOfferPublishedIterator{contract: _MyferenceMarket.contract, event: "OfferPublished", logs: logs, sub: sub}, nil
}

// WatchOfferPublished is a free log subscription operation binding the contract event 0x5ae57c6c471f08a53556a92bfe0656a3256d5712e6bb87c07294902c46ee99e3.
//
// Solidity: event OfferPublished(address indexed provider, bytes32 indexed offerId, uint64 indexed version, bytes32 modelHash, bytes32 capabilityHash, uint256 inputPerMillion, uint256 outputPerMillion, uint256 computePerSecond)
func (_MyferenceMarket *MyferenceMarketFilterer) WatchOfferPublished(opts *bind.WatchOpts, sink chan<- *MyferenceMarketOfferPublished, provider []common.Address, offerId [][32]byte, version []uint64) (event.Subscription, error) {

	var providerRule []interface{}
	for _, providerItem := range provider {
		providerRule = append(providerRule, providerItem)
	}
	var offerIdRule []interface{}
	for _, offerIdItem := range offerId {
		offerIdRule = append(offerIdRule, offerIdItem)
	}
	var versionRule []interface{}
	for _, versionItem := range version {
		versionRule = append(versionRule, versionItem)
	}

	logs, sub, err := _MyferenceMarket.contract.WatchLogs(opts, "OfferPublished", providerRule, offerIdRule, versionRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(MyferenceMarketOfferPublished)
				if err := _MyferenceMarket.contract.UnpackLog(event, "OfferPublished", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseOfferPublished is a log parse operation binding the contract event 0x5ae57c6c471f08a53556a92bfe0656a3256d5712e6bb87c07294902c46ee99e3.
//
// Solidity: event OfferPublished(address indexed provider, bytes32 indexed offerId, uint64 indexed version, bytes32 modelHash, bytes32 capabilityHash, uint256 inputPerMillion, uint256 outputPerMillion, uint256 computePerSecond)
func (_MyferenceMarket *MyferenceMarketFilterer) ParseOfferPublished(log types.Log) (*MyferenceMarketOfferPublished, error) {
	event := new(MyferenceMarketOfferPublished)
	if err := _MyferenceMarket.contract.UnpackLog(event, "OfferPublished", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// MyferenceMarketOwnershipTransferStartedIterator is returned from FilterOwnershipTransferStarted and is used to iterate over the raw logs and unpacked data for OwnershipTransferStarted events raised by the MyferenceMarket contract.
type MyferenceMarketOwnershipTransferStartedIterator struct {
	Event *MyferenceMarketOwnershipTransferStarted // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *MyferenceMarketOwnershipTransferStartedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(MyferenceMarketOwnershipTransferStarted)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(MyferenceMarketOwnershipTransferStarted)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *MyferenceMarketOwnershipTransferStartedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *MyferenceMarketOwnershipTransferStartedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// MyferenceMarketOwnershipTransferStarted represents a OwnershipTransferStarted event raised by the MyferenceMarket contract.
type MyferenceMarketOwnershipTransferStarted struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferStarted is a free log retrieval operation binding the contract event 0x38d16b8cac22d99fc7c124b9cd0de2d3fa1faef420bfe791d8c362d765e22700.
//
// Solidity: event OwnershipTransferStarted(address indexed previousOwner, address indexed newOwner)
func (_MyferenceMarket *MyferenceMarketFilterer) FilterOwnershipTransferStarted(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*MyferenceMarketOwnershipTransferStartedIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _MyferenceMarket.contract.FilterLogs(opts, "OwnershipTransferStarted", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &MyferenceMarketOwnershipTransferStartedIterator{contract: _MyferenceMarket.contract, event: "OwnershipTransferStarted", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferStarted is a free log subscription operation binding the contract event 0x38d16b8cac22d99fc7c124b9cd0de2d3fa1faef420bfe791d8c362d765e22700.
//
// Solidity: event OwnershipTransferStarted(address indexed previousOwner, address indexed newOwner)
func (_MyferenceMarket *MyferenceMarketFilterer) WatchOwnershipTransferStarted(opts *bind.WatchOpts, sink chan<- *MyferenceMarketOwnershipTransferStarted, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _MyferenceMarket.contract.WatchLogs(opts, "OwnershipTransferStarted", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(MyferenceMarketOwnershipTransferStarted)
				if err := _MyferenceMarket.contract.UnpackLog(event, "OwnershipTransferStarted", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseOwnershipTransferStarted is a log parse operation binding the contract event 0x38d16b8cac22d99fc7c124b9cd0de2d3fa1faef420bfe791d8c362d765e22700.
//
// Solidity: event OwnershipTransferStarted(address indexed previousOwner, address indexed newOwner)
func (_MyferenceMarket *MyferenceMarketFilterer) ParseOwnershipTransferStarted(log types.Log) (*MyferenceMarketOwnershipTransferStarted, error) {
	event := new(MyferenceMarketOwnershipTransferStarted)
	if err := _MyferenceMarket.contract.UnpackLog(event, "OwnershipTransferStarted", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// MyferenceMarketOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the MyferenceMarket contract.
type MyferenceMarketOwnershipTransferredIterator struct {
	Event *MyferenceMarketOwnershipTransferred // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *MyferenceMarketOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(MyferenceMarketOwnershipTransferred)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(MyferenceMarketOwnershipTransferred)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *MyferenceMarketOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *MyferenceMarketOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// MyferenceMarketOwnershipTransferred represents a OwnershipTransferred event raised by the MyferenceMarket contract.
type MyferenceMarketOwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_MyferenceMarket *MyferenceMarketFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*MyferenceMarketOwnershipTransferredIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _MyferenceMarket.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &MyferenceMarketOwnershipTransferredIterator{contract: _MyferenceMarket.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_MyferenceMarket *MyferenceMarketFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *MyferenceMarketOwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _MyferenceMarket.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(MyferenceMarketOwnershipTransferred)
				if err := _MyferenceMarket.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseOwnershipTransferred is a log parse operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_MyferenceMarket *MyferenceMarketFilterer) ParseOwnershipTransferred(log types.Log) (*MyferenceMarketOwnershipTransferred, error) {
	event := new(MyferenceMarketOwnershipTransferred)
	if err := _MyferenceMarket.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// MyferenceMarketPausedIterator is returned from FilterPaused and is used to iterate over the raw logs and unpacked data for Paused events raised by the MyferenceMarket contract.
type MyferenceMarketPausedIterator struct {
	Event *MyferenceMarketPaused // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *MyferenceMarketPausedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(MyferenceMarketPaused)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(MyferenceMarketPaused)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *MyferenceMarketPausedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *MyferenceMarketPausedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// MyferenceMarketPaused represents a Paused event raised by the MyferenceMarket contract.
type MyferenceMarketPaused struct {
	Account common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterPaused is a free log retrieval operation binding the contract event 0x62e78cea01bee320cd4e420270b5ea74000d11b0c9f74754ebdbfc544b05a258.
//
// Solidity: event Paused(address account)
func (_MyferenceMarket *MyferenceMarketFilterer) FilterPaused(opts *bind.FilterOpts) (*MyferenceMarketPausedIterator, error) {

	logs, sub, err := _MyferenceMarket.contract.FilterLogs(opts, "Paused")
	if err != nil {
		return nil, err
	}
	return &MyferenceMarketPausedIterator{contract: _MyferenceMarket.contract, event: "Paused", logs: logs, sub: sub}, nil
}

// WatchPaused is a free log subscription operation binding the contract event 0x62e78cea01bee320cd4e420270b5ea74000d11b0c9f74754ebdbfc544b05a258.
//
// Solidity: event Paused(address account)
func (_MyferenceMarket *MyferenceMarketFilterer) WatchPaused(opts *bind.WatchOpts, sink chan<- *MyferenceMarketPaused) (event.Subscription, error) {

	logs, sub, err := _MyferenceMarket.contract.WatchLogs(opts, "Paused")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(MyferenceMarketPaused)
				if err := _MyferenceMarket.contract.UnpackLog(event, "Paused", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParsePaused is a log parse operation binding the contract event 0x62e78cea01bee320cd4e420270b5ea74000d11b0c9f74754ebdbfc544b05a258.
//
// Solidity: event Paused(address account)
func (_MyferenceMarket *MyferenceMarketFilterer) ParsePaused(log types.Log) (*MyferenceMarketPaused, error) {
	event := new(MyferenceMarketPaused)
	if err := _MyferenceMarket.contract.UnpackLog(event, "Paused", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// MyferenceMarketProviderSlashedIterator is returned from FilterProviderSlashed and is used to iterate over the raw logs and unpacked data for ProviderSlashed events raised by the MyferenceMarket contract.
type MyferenceMarketProviderSlashedIterator struct {
	Event *MyferenceMarketProviderSlashed // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *MyferenceMarketProviderSlashedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(MyferenceMarketProviderSlashed)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(MyferenceMarketProviderSlashed)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *MyferenceMarketProviderSlashedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *MyferenceMarketProviderSlashedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// MyferenceMarketProviderSlashed represents a ProviderSlashed event raised by the MyferenceMarket contract.
type MyferenceMarketProviderSlashed struct {
	Provider  common.Address
	RequestId [32]byte
	Amount    *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterProviderSlashed is a free log retrieval operation binding the contract event 0xc0fb837f95a6626fe0d8363a3eddb02605b2d7bb0e3e640a88859a5e9afebc0e.
//
// Solidity: event ProviderSlashed(address indexed provider, bytes32 indexed requestId, uint256 amount)
func (_MyferenceMarket *MyferenceMarketFilterer) FilterProviderSlashed(opts *bind.FilterOpts, provider []common.Address, requestId [][32]byte) (*MyferenceMarketProviderSlashedIterator, error) {

	var providerRule []interface{}
	for _, providerItem := range provider {
		providerRule = append(providerRule, providerItem)
	}
	var requestIdRule []interface{}
	for _, requestIdItem := range requestId {
		requestIdRule = append(requestIdRule, requestIdItem)
	}

	logs, sub, err := _MyferenceMarket.contract.FilterLogs(opts, "ProviderSlashed", providerRule, requestIdRule)
	if err != nil {
		return nil, err
	}
	return &MyferenceMarketProviderSlashedIterator{contract: _MyferenceMarket.contract, event: "ProviderSlashed", logs: logs, sub: sub}, nil
}

// WatchProviderSlashed is a free log subscription operation binding the contract event 0xc0fb837f95a6626fe0d8363a3eddb02605b2d7bb0e3e640a88859a5e9afebc0e.
//
// Solidity: event ProviderSlashed(address indexed provider, bytes32 indexed requestId, uint256 amount)
func (_MyferenceMarket *MyferenceMarketFilterer) WatchProviderSlashed(opts *bind.WatchOpts, sink chan<- *MyferenceMarketProviderSlashed, provider []common.Address, requestId [][32]byte) (event.Subscription, error) {

	var providerRule []interface{}
	for _, providerItem := range provider {
		providerRule = append(providerRule, providerItem)
	}
	var requestIdRule []interface{}
	for _, requestIdItem := range requestId {
		requestIdRule = append(requestIdRule, requestIdItem)
	}

	logs, sub, err := _MyferenceMarket.contract.WatchLogs(opts, "ProviderSlashed", providerRule, requestIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(MyferenceMarketProviderSlashed)
				if err := _MyferenceMarket.contract.UnpackLog(event, "ProviderSlashed", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseProviderSlashed is a log parse operation binding the contract event 0xc0fb837f95a6626fe0d8363a3eddb02605b2d7bb0e3e640a88859a5e9afebc0e.
//
// Solidity: event ProviderSlashed(address indexed provider, bytes32 indexed requestId, uint256 amount)
func (_MyferenceMarket *MyferenceMarketFilterer) ParseProviderSlashed(log types.Log) (*MyferenceMarketProviderSlashed, error) {
	event := new(MyferenceMarketProviderSlashed)
	if err := _MyferenceMarket.contract.UnpackLog(event, "ProviderSlashed", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// MyferenceMarketReceiptSettledIterator is returned from FilterReceiptSettled and is used to iterate over the raw logs and unpacked data for ReceiptSettled events raised by the MyferenceMarket contract.
type MyferenceMarketReceiptSettledIterator struct {
	Event *MyferenceMarketReceiptSettled // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *MyferenceMarketReceiptSettledIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(MyferenceMarketReceiptSettled)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(MyferenceMarketReceiptSettled)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *MyferenceMarketReceiptSettledIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *MyferenceMarketReceiptSettledIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// MyferenceMarketReceiptSettled represents a ReceiptSettled event raised by the MyferenceMarket contract.
type MyferenceMarketReceiptSettled struct {
	RequestId      [32]byte
	SessionId      [32]byte
	Provider       common.Address
	ProviderAmount *big.Int
	FeeAmount      *big.Int
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterReceiptSettled is a free log retrieval operation binding the contract event 0xc429078b36d592f1bf76c1f12f1ca4c553be70723c2ad50d40a444e32b7ec06d.
//
// Solidity: event ReceiptSettled(bytes32 indexed requestId, bytes32 indexed sessionId, address indexed provider, uint256 providerAmount, uint256 feeAmount)
func (_MyferenceMarket *MyferenceMarketFilterer) FilterReceiptSettled(opts *bind.FilterOpts, requestId [][32]byte, sessionId [][32]byte, provider []common.Address) (*MyferenceMarketReceiptSettledIterator, error) {

	var requestIdRule []interface{}
	for _, requestIdItem := range requestId {
		requestIdRule = append(requestIdRule, requestIdItem)
	}
	var sessionIdRule []interface{}
	for _, sessionIdItem := range sessionId {
		sessionIdRule = append(sessionIdRule, sessionIdItem)
	}
	var providerRule []interface{}
	for _, providerItem := range provider {
		providerRule = append(providerRule, providerItem)
	}

	logs, sub, err := _MyferenceMarket.contract.FilterLogs(opts, "ReceiptSettled", requestIdRule, sessionIdRule, providerRule)
	if err != nil {
		return nil, err
	}
	return &MyferenceMarketReceiptSettledIterator{contract: _MyferenceMarket.contract, event: "ReceiptSettled", logs: logs, sub: sub}, nil
}

// WatchReceiptSettled is a free log subscription operation binding the contract event 0xc429078b36d592f1bf76c1f12f1ca4c553be70723c2ad50d40a444e32b7ec06d.
//
// Solidity: event ReceiptSettled(bytes32 indexed requestId, bytes32 indexed sessionId, address indexed provider, uint256 providerAmount, uint256 feeAmount)
func (_MyferenceMarket *MyferenceMarketFilterer) WatchReceiptSettled(opts *bind.WatchOpts, sink chan<- *MyferenceMarketReceiptSettled, requestId [][32]byte, sessionId [][32]byte, provider []common.Address) (event.Subscription, error) {

	var requestIdRule []interface{}
	for _, requestIdItem := range requestId {
		requestIdRule = append(requestIdRule, requestIdItem)
	}
	var sessionIdRule []interface{}
	for _, sessionIdItem := range sessionId {
		sessionIdRule = append(sessionIdRule, sessionIdItem)
	}
	var providerRule []interface{}
	for _, providerItem := range provider {
		providerRule = append(providerRule, providerItem)
	}

	logs, sub, err := _MyferenceMarket.contract.WatchLogs(opts, "ReceiptSettled", requestIdRule, sessionIdRule, providerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(MyferenceMarketReceiptSettled)
				if err := _MyferenceMarket.contract.UnpackLog(event, "ReceiptSettled", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseReceiptSettled is a log parse operation binding the contract event 0xc429078b36d592f1bf76c1f12f1ca4c553be70723c2ad50d40a444e32b7ec06d.
//
// Solidity: event ReceiptSettled(bytes32 indexed requestId, bytes32 indexed sessionId, address indexed provider, uint256 providerAmount, uint256 feeAmount)
func (_MyferenceMarket *MyferenceMarketFilterer) ParseReceiptSettled(log types.Log) (*MyferenceMarketReceiptSettled, error) {
	event := new(MyferenceMarketReceiptSettled)
	if err := _MyferenceMarket.contract.UnpackLog(event, "ReceiptSettled", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// MyferenceMarketSessionCloseRequestedIterator is returned from FilterSessionCloseRequested and is used to iterate over the raw logs and unpacked data for SessionCloseRequested events raised by the MyferenceMarket contract.
type MyferenceMarketSessionCloseRequestedIterator struct {
	Event *MyferenceMarketSessionCloseRequested // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *MyferenceMarketSessionCloseRequestedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(MyferenceMarketSessionCloseRequested)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(MyferenceMarketSessionCloseRequested)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *MyferenceMarketSessionCloseRequestedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *MyferenceMarketSessionCloseRequestedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// MyferenceMarketSessionCloseRequested represents a SessionCloseRequested event raised by the MyferenceMarket contract.
type MyferenceMarketSessionCloseRequested struct {
	SessionId   [32]byte
	AvailableAt uint64
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterSessionCloseRequested is a free log retrieval operation binding the contract event 0x72b43cf8a65c18d2c3cb2c3ebce801281fcc67f6ada51dc7709c95947b443a81.
//
// Solidity: event SessionCloseRequested(bytes32 indexed sessionId, uint64 availableAt)
func (_MyferenceMarket *MyferenceMarketFilterer) FilterSessionCloseRequested(opts *bind.FilterOpts, sessionId [][32]byte) (*MyferenceMarketSessionCloseRequestedIterator, error) {

	var sessionIdRule []interface{}
	for _, sessionIdItem := range sessionId {
		sessionIdRule = append(sessionIdRule, sessionIdItem)
	}

	logs, sub, err := _MyferenceMarket.contract.FilterLogs(opts, "SessionCloseRequested", sessionIdRule)
	if err != nil {
		return nil, err
	}
	return &MyferenceMarketSessionCloseRequestedIterator{contract: _MyferenceMarket.contract, event: "SessionCloseRequested", logs: logs, sub: sub}, nil
}

// WatchSessionCloseRequested is a free log subscription operation binding the contract event 0x72b43cf8a65c18d2c3cb2c3ebce801281fcc67f6ada51dc7709c95947b443a81.
//
// Solidity: event SessionCloseRequested(bytes32 indexed sessionId, uint64 availableAt)
func (_MyferenceMarket *MyferenceMarketFilterer) WatchSessionCloseRequested(opts *bind.WatchOpts, sink chan<- *MyferenceMarketSessionCloseRequested, sessionId [][32]byte) (event.Subscription, error) {

	var sessionIdRule []interface{}
	for _, sessionIdItem := range sessionId {
		sessionIdRule = append(sessionIdRule, sessionIdItem)
	}

	logs, sub, err := _MyferenceMarket.contract.WatchLogs(opts, "SessionCloseRequested", sessionIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(MyferenceMarketSessionCloseRequested)
				if err := _MyferenceMarket.contract.UnpackLog(event, "SessionCloseRequested", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseSessionCloseRequested is a log parse operation binding the contract event 0x72b43cf8a65c18d2c3cb2c3ebce801281fcc67f6ada51dc7709c95947b443a81.
//
// Solidity: event SessionCloseRequested(bytes32 indexed sessionId, uint64 availableAt)
func (_MyferenceMarket *MyferenceMarketFilterer) ParseSessionCloseRequested(log types.Log) (*MyferenceMarketSessionCloseRequested, error) {
	event := new(MyferenceMarketSessionCloseRequested)
	if err := _MyferenceMarket.contract.UnpackLog(event, "SessionCloseRequested", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// MyferenceMarketSessionClosedIterator is returned from FilterSessionClosed and is used to iterate over the raw logs and unpacked data for SessionClosed events raised by the MyferenceMarket contract.
type MyferenceMarketSessionClosedIterator struct {
	Event *MyferenceMarketSessionClosed // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *MyferenceMarketSessionClosedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(MyferenceMarketSessionClosed)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(MyferenceMarketSessionClosed)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *MyferenceMarketSessionClosedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *MyferenceMarketSessionClosedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// MyferenceMarketSessionClosed represents a SessionClosed event raised by the MyferenceMarket contract.
type MyferenceMarketSessionClosed struct {
	SessionId      [32]byte
	ReturnedAmount *big.Int
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterSessionClosed is a free log retrieval operation binding the contract event 0x1d8ada2ed73dcd9599105b1206ea1d3aa9e687b4e8f3b49940a4b03b6360585e.
//
// Solidity: event SessionClosed(bytes32 indexed sessionId, uint256 returnedAmount)
func (_MyferenceMarket *MyferenceMarketFilterer) FilterSessionClosed(opts *bind.FilterOpts, sessionId [][32]byte) (*MyferenceMarketSessionClosedIterator, error) {

	var sessionIdRule []interface{}
	for _, sessionIdItem := range sessionId {
		sessionIdRule = append(sessionIdRule, sessionIdItem)
	}

	logs, sub, err := _MyferenceMarket.contract.FilterLogs(opts, "SessionClosed", sessionIdRule)
	if err != nil {
		return nil, err
	}
	return &MyferenceMarketSessionClosedIterator{contract: _MyferenceMarket.contract, event: "SessionClosed", logs: logs, sub: sub}, nil
}

// WatchSessionClosed is a free log subscription operation binding the contract event 0x1d8ada2ed73dcd9599105b1206ea1d3aa9e687b4e8f3b49940a4b03b6360585e.
//
// Solidity: event SessionClosed(bytes32 indexed sessionId, uint256 returnedAmount)
func (_MyferenceMarket *MyferenceMarketFilterer) WatchSessionClosed(opts *bind.WatchOpts, sink chan<- *MyferenceMarketSessionClosed, sessionId [][32]byte) (event.Subscription, error) {

	var sessionIdRule []interface{}
	for _, sessionIdItem := range sessionId {
		sessionIdRule = append(sessionIdRule, sessionIdItem)
	}

	logs, sub, err := _MyferenceMarket.contract.WatchLogs(opts, "SessionClosed", sessionIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(MyferenceMarketSessionClosed)
				if err := _MyferenceMarket.contract.UnpackLog(event, "SessionClosed", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseSessionClosed is a log parse operation binding the contract event 0x1d8ada2ed73dcd9599105b1206ea1d3aa9e687b4e8f3b49940a4b03b6360585e.
//
// Solidity: event SessionClosed(bytes32 indexed sessionId, uint256 returnedAmount)
func (_MyferenceMarket *MyferenceMarketFilterer) ParseSessionClosed(log types.Log) (*MyferenceMarketSessionClosed, error) {
	event := new(MyferenceMarketSessionClosed)
	if err := _MyferenceMarket.contract.UnpackLog(event, "SessionClosed", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// MyferenceMarketSessionOpenedIterator is returned from FilterSessionOpened and is used to iterate over the raw logs and unpacked data for SessionOpened events raised by the MyferenceMarket contract.
type MyferenceMarketSessionOpenedIterator struct {
	Event *MyferenceMarketSessionOpened // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *MyferenceMarketSessionOpenedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(MyferenceMarketSessionOpened)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(MyferenceMarketSessionOpened)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *MyferenceMarketSessionOpenedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *MyferenceMarketSessionOpenedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// MyferenceMarketSessionOpened represents a SessionOpened event raised by the MyferenceMarket contract.
type MyferenceMarketSessionOpened struct {
	SessionId [32]byte
	Customer  common.Address
	Allowance *big.Int
	ExpiresAt uint64
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterSessionOpened is a free log retrieval operation binding the contract event 0x88472a4d077942c1db7795cd75d79255367dc41ba9337b21598c772438befc8e.
//
// Solidity: event SessionOpened(bytes32 indexed sessionId, address indexed customer, uint256 allowance, uint64 expiresAt)
func (_MyferenceMarket *MyferenceMarketFilterer) FilterSessionOpened(opts *bind.FilterOpts, sessionId [][32]byte, customer []common.Address) (*MyferenceMarketSessionOpenedIterator, error) {

	var sessionIdRule []interface{}
	for _, sessionIdItem := range sessionId {
		sessionIdRule = append(sessionIdRule, sessionIdItem)
	}
	var customerRule []interface{}
	for _, customerItem := range customer {
		customerRule = append(customerRule, customerItem)
	}

	logs, sub, err := _MyferenceMarket.contract.FilterLogs(opts, "SessionOpened", sessionIdRule, customerRule)
	if err != nil {
		return nil, err
	}
	return &MyferenceMarketSessionOpenedIterator{contract: _MyferenceMarket.contract, event: "SessionOpened", logs: logs, sub: sub}, nil
}

// WatchSessionOpened is a free log subscription operation binding the contract event 0x88472a4d077942c1db7795cd75d79255367dc41ba9337b21598c772438befc8e.
//
// Solidity: event SessionOpened(bytes32 indexed sessionId, address indexed customer, uint256 allowance, uint64 expiresAt)
func (_MyferenceMarket *MyferenceMarketFilterer) WatchSessionOpened(opts *bind.WatchOpts, sink chan<- *MyferenceMarketSessionOpened, sessionId [][32]byte, customer []common.Address) (event.Subscription, error) {

	var sessionIdRule []interface{}
	for _, sessionIdItem := range sessionId {
		sessionIdRule = append(sessionIdRule, sessionIdItem)
	}
	var customerRule []interface{}
	for _, customerItem := range customer {
		customerRule = append(customerRule, customerItem)
	}

	logs, sub, err := _MyferenceMarket.contract.WatchLogs(opts, "SessionOpened", sessionIdRule, customerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(MyferenceMarketSessionOpened)
				if err := _MyferenceMarket.contract.UnpackLog(event, "SessionOpened", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseSessionOpened is a log parse operation binding the contract event 0x88472a4d077942c1db7795cd75d79255367dc41ba9337b21598c772438befc8e.
//
// Solidity: event SessionOpened(bytes32 indexed sessionId, address indexed customer, uint256 allowance, uint64 expiresAt)
func (_MyferenceMarket *MyferenceMarketFilterer) ParseSessionOpened(log types.Log) (*MyferenceMarketSessionOpened, error) {
	event := new(MyferenceMarketSessionOpened)
	if err := _MyferenceMarket.contract.UnpackLog(event, "SessionOpened", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// MyferenceMarketUnpausedIterator is returned from FilterUnpaused and is used to iterate over the raw logs and unpacked data for Unpaused events raised by the MyferenceMarket contract.
type MyferenceMarketUnpausedIterator struct {
	Event *MyferenceMarketUnpaused // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *MyferenceMarketUnpausedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(MyferenceMarketUnpaused)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(MyferenceMarketUnpaused)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *MyferenceMarketUnpausedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *MyferenceMarketUnpausedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// MyferenceMarketUnpaused represents a Unpaused event raised by the MyferenceMarket contract.
type MyferenceMarketUnpaused struct {
	Account common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterUnpaused is a free log retrieval operation binding the contract event 0x5db9ee0a495bf2e6ff9c91a7834c1ba4fdd244a5e8aa4e537bd38aeae4b073aa.
//
// Solidity: event Unpaused(address account)
func (_MyferenceMarket *MyferenceMarketFilterer) FilterUnpaused(opts *bind.FilterOpts) (*MyferenceMarketUnpausedIterator, error) {

	logs, sub, err := _MyferenceMarket.contract.FilterLogs(opts, "Unpaused")
	if err != nil {
		return nil, err
	}
	return &MyferenceMarketUnpausedIterator{contract: _MyferenceMarket.contract, event: "Unpaused", logs: logs, sub: sub}, nil
}

// WatchUnpaused is a free log subscription operation binding the contract event 0x5db9ee0a495bf2e6ff9c91a7834c1ba4fdd244a5e8aa4e537bd38aeae4b073aa.
//
// Solidity: event Unpaused(address account)
func (_MyferenceMarket *MyferenceMarketFilterer) WatchUnpaused(opts *bind.WatchOpts, sink chan<- *MyferenceMarketUnpaused) (event.Subscription, error) {

	logs, sub, err := _MyferenceMarket.contract.WatchLogs(opts, "Unpaused")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(MyferenceMarketUnpaused)
				if err := _MyferenceMarket.contract.UnpackLog(event, "Unpaused", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseUnpaused is a log parse operation binding the contract event 0x5db9ee0a495bf2e6ff9c91a7834c1ba4fdd244a5e8aa4e537bd38aeae4b073aa.
//
// Solidity: event Unpaused(address account)
func (_MyferenceMarket *MyferenceMarketFilterer) ParseUnpaused(log types.Log) (*MyferenceMarketUnpaused, error) {
	event := new(MyferenceMarketUnpaused)
	if err := _MyferenceMarket.contract.UnpackLog(event, "Unpaused", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// MyferenceMarketWithdrawalRequestedIterator is returned from FilterWithdrawalRequested and is used to iterate over the raw logs and unpacked data for WithdrawalRequested events raised by the MyferenceMarket contract.
type MyferenceMarketWithdrawalRequestedIterator struct {
	Event *MyferenceMarketWithdrawalRequested // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *MyferenceMarketWithdrawalRequestedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(MyferenceMarketWithdrawalRequested)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(MyferenceMarketWithdrawalRequested)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *MyferenceMarketWithdrawalRequestedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *MyferenceMarketWithdrawalRequestedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// MyferenceMarketWithdrawalRequested represents a WithdrawalRequested event raised by the MyferenceMarket contract.
type MyferenceMarketWithdrawalRequested struct {
	Account common.Address
	Amount  *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterWithdrawalRequested is a free log retrieval operation binding the contract event 0xe670e4e82118d22a1f9ee18920455ebc958bae26a90a05d31d3378788b1b0e44.
//
// Solidity: event WithdrawalRequested(address indexed account, uint256 amount)
func (_MyferenceMarket *MyferenceMarketFilterer) FilterWithdrawalRequested(opts *bind.FilterOpts, account []common.Address) (*MyferenceMarketWithdrawalRequestedIterator, error) {

	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}

	logs, sub, err := _MyferenceMarket.contract.FilterLogs(opts, "WithdrawalRequested", accountRule)
	if err != nil {
		return nil, err
	}
	return &MyferenceMarketWithdrawalRequestedIterator{contract: _MyferenceMarket.contract, event: "WithdrawalRequested", logs: logs, sub: sub}, nil
}

// WatchWithdrawalRequested is a free log subscription operation binding the contract event 0xe670e4e82118d22a1f9ee18920455ebc958bae26a90a05d31d3378788b1b0e44.
//
// Solidity: event WithdrawalRequested(address indexed account, uint256 amount)
func (_MyferenceMarket *MyferenceMarketFilterer) WatchWithdrawalRequested(opts *bind.WatchOpts, sink chan<- *MyferenceMarketWithdrawalRequested, account []common.Address) (event.Subscription, error) {

	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}

	logs, sub, err := _MyferenceMarket.contract.WatchLogs(opts, "WithdrawalRequested", accountRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(MyferenceMarketWithdrawalRequested)
				if err := _MyferenceMarket.contract.UnpackLog(event, "WithdrawalRequested", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseWithdrawalRequested is a log parse operation binding the contract event 0xe670e4e82118d22a1f9ee18920455ebc958bae26a90a05d31d3378788b1b0e44.
//
// Solidity: event WithdrawalRequested(address indexed account, uint256 amount)
func (_MyferenceMarket *MyferenceMarketFilterer) ParseWithdrawalRequested(log types.Log) (*MyferenceMarketWithdrawalRequested, error) {
	event := new(MyferenceMarketWithdrawalRequested)
	if err := _MyferenceMarket.contract.UnpackLog(event, "WithdrawalRequested", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
