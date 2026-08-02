package chain

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"math/big"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/kunalshah017/myference/server/internal/chain/bindings"
)

var ErrTransactionReverted = errors.New("EVM transaction reverted")

type Receipt = bindings.MyferenceMarketReceipt

type Client struct {
	eth             *ethclient.Client
	chainID         *big.Int
	key             *ecdsa.PrivateKey
	address         common.Address
	contract        *bindings.MyferenceMarket
	contractAddress common.Address
	mu              sync.Mutex
}

func Dial(ctx context.Context, rpcURL, privateKey string) (*Client, error) {
	eth, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return nil, err
	}
	chainID, err := eth.ChainID(ctx)
	if err != nil {
		eth.Close()
		return nil, err
	}
	key, err := crypto.HexToECDSA(strings.TrimPrefix(privateKey, "0x"))
	if err != nil {
		eth.Close()
		return nil, err
	}
	address := crypto.PubkeyToAddress(key.PublicKey)
	return &Client{eth: eth, chainID: chainID, key: key, address: address}, nil
}

func (c *Client) Close() { c.eth.Close() }

func (c *Client) Address() common.Address { return c.address }

func (c *Client) ContractAddress() common.Address { return c.contractAddress }

func (c *Client) Bind(address common.Address) error {
	contract, err := bindings.NewMyferenceMarket(address, c.eth)
	if err != nil {
		return err
	}
	c.contract, c.contractAddress = contract, address
	return nil
}

func (c *Client) Deploy(ctx context.Context, owner, feeRecipient, settlementSigner common.Address, minimumBond *big.Int, bondExitDelay, feeDelay uint64) (common.Address, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	opts, err := c.options(ctx, nil)
	if err != nil {
		return common.Address{}, err
	}
	address, transaction, contract, err := bindings.DeployMyferenceMarket(opts, c.eth, owner, feeRecipient, settlementSigner, minimumBond, bondExitDelay, feeDelay)
	if err != nil {
		return common.Address{}, err
	}
	if err := c.wait(ctx, transaction); err != nil {
		return common.Address{}, err
	}
	c.contract, c.contractAddress = contract, address
	return address, nil
}

func (c *Client) Deposit(ctx context.Context, value *big.Int) error {
	return c.transact(ctx, value, func(opts *bind.TransactOpts) (*types.Transaction, error) { return c.contract.Deposit(opts) })
}

func (c *Client) DepositBond(ctx context.Context, value *big.Int) error {
	return c.transact(ctx, value, func(opts *bind.TransactOpts) (*types.Transaction, error) { return c.contract.DepositBond(opts) })
}

func (c *Client) RequestWithdrawal(ctx context.Context, amount *big.Int) error {
	return c.transact(ctx, nil, func(opts *bind.TransactOpts) (*types.Transaction, error) {
		return c.contract.RequestWithdrawal(opts, amount)
	})
}

func (c *Client) Claim(ctx context.Context) error {
	return c.transact(ctx, nil, func(opts *bind.TransactOpts) (*types.Transaction, error) { return c.contract.Claim(opts) })
}

func (c *Client) RequestBondExit(ctx context.Context) error {
	return c.transact(ctx, nil, func(opts *bind.TransactOpts) (*types.Transaction, error) { return c.contract.RequestBondExit(opts) })
}

func (c *Client) FinalizeBondExit(ctx context.Context) error {
	return c.transact(ctx, nil, func(opts *bind.TransactOpts) (*types.Transaction, error) { return c.contract.FinalizeBondExit(opts) })
}

func (c *Client) RequestSessionClose(ctx context.Context, sessionID [32]byte) error {
	return c.transact(ctx, nil, func(opts *bind.TransactOpts) (*types.Transaction, error) {
		return c.contract.RequestSessionClose(opts, sessionID)
	})
}

func (c *Client) FinalizeSessionClose(ctx context.Context, sessionID [32]byte) error {
	return c.transact(ctx, nil, func(opts *bind.TransactOpts) (*types.Transaction, error) {
		return c.contract.FinalizeSessionClose(opts, sessionID)
	})
}

func (c *Client) PublishOffer(ctx context.Context, offerID, modelHash, capabilityHash [32]byte, inputPerMillion, outputPerMillion, computePerSecond *big.Int) error {
	return c.transact(ctx, nil, func(opts *bind.TransactOpts) (*types.Transaction, error) {
		return c.contract.PublishOffer(opts, offerID, modelHash, capabilityHash, inputPerMillion, outputPerMillion, computePerSecond)
	})
}

func (c *Client) OpenSession(ctx context.Context, sessionID [32]byte, allowance *big.Int, expiresAt uint64) error {
	return c.transact(ctx, nil, func(opts *bind.TransactOpts) (*types.Transaction, error) {
		return c.contract.OpenSession(opts, sessionID, allowance, expiresAt)
	})
}

func (c *Client) SignReceipt(ctx context.Context, receipt Receipt) ([]byte, error) {
	if c.contract == nil {
		return nil, errors.New("contract is not bound")
	}
	digest, err := c.contract.HashReceipt(&bind.CallOpts{Context: ctx}, receipt)
	if err != nil {
		return nil, err
	}
	signature, err := crypto.Sign(digest[:], c.key)
	if err != nil {
		return nil, err
	}
	signature[64] += 27
	return signature, nil
}

func (c *Client) SettleReceipt(ctx context.Context, receipt Receipt, providerSignature, settlementSignature []byte) error {
	return c.transact(ctx, nil, func(opts *bind.TransactOpts) (*types.Transaction, error) {
		return c.contract.SettleReceipt(opts, receipt, providerSignature, settlementSignature)
	})
}

func (c *Client) PrepareSettlement(ctx context.Context, receipts []Receipt, providerSignatures, settlementSignatures [][]byte) (*types.Transaction, error) {
	if c.contract == nil {
		return nil, errors.New("contract is not bound")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	opts, err := c.options(ctx, nil)
	if err != nil {
		return nil, err
	}
	opts.NoSend = true
	return c.contract.SettleReceipts(opts, receipts, providerSignatures, settlementSignatures)
}

func (c *Client) Broadcast(ctx context.Context, transaction *types.Transaction) error {
	if err := c.eth.SendTransaction(ctx, transaction); err != nil {
		if _, receiptErr := c.eth.TransactionReceipt(ctx, transaction.Hash()); receiptErr != nil && !strings.Contains(strings.ToLower(err.Error()), "already known") {
			return err
		}
	}
	return c.wait(ctx, transaction)
}

func (c *Client) SettlementState(ctx context.Context, requestID [32]byte, provider common.Address) (bool, *big.Int, error) {
	if c.contract == nil {
		return false, nil, errors.New("contract is not bound")
	}
	settled, err := c.contract.SettledRequests(&bind.CallOpts{Context: ctx}, requestID)
	if err != nil {
		return false, nil, err
	}
	claimable, err := c.contract.Claimable(&bind.CallOpts{Context: ctx}, provider)
	return settled, claimable, err
}

func (c *Client) Header(ctx context.Context) (*types.Header, error) {
	return c.eth.HeaderByNumber(ctx, nil)
}

func (c *Client) transact(ctx context.Context, value *big.Int, send func(*bind.TransactOpts) (*types.Transaction, error)) error {
	if c.contract == nil {
		return errors.New("contract is not bound")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	opts, err := c.options(ctx, value)
	if err != nil {
		return err
	}
	transaction, err := send(opts)
	if err != nil {
		return err
	}
	return c.wait(ctx, transaction)
}

func (c *Client) options(ctx context.Context, value *big.Int) (*bind.TransactOpts, error) {
	opts, err := bind.NewKeyedTransactorWithChainID(c.key, c.chainID)
	if err != nil {
		return nil, err
	}
	opts.Context, opts.Value = ctx, value
	return opts, nil
}

func (c *Client) wait(ctx context.Context, transaction *types.Transaction) error {
	receipt, err := bind.WaitMined(ctx, c.eth, transaction)
	if err != nil {
		return err
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return ErrTransactionReverted
	}
	return nil
}
