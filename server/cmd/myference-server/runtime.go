package main

import (
	"context"
	"errors"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/kunalshah017/myference/server/internal/chain"
	"github.com/kunalshah017/myference/server/internal/relay"
	"github.com/kunalshah017/myference/server/internal/settlement"
	"github.com/kunalshah017/myference/server/internal/store"
)

type chainConfig struct {
	RPCURL, ContractAddress, SettlementPrivateKey string
	StartBlock                                    uint64
}

func loadChainConfig(getenv func(string) string) (chainConfig, error) {
	config := chainConfig{RPCURL: strings.TrimSpace(getenv("MYFERENCE_RPC_URL")), ContractAddress: strings.TrimSpace(getenv("MYFERENCE_CONTRACT_ADDRESS")), SettlementPrivateKey: strings.TrimSpace(getenv("MYFERENCE_SETTLEMENT_PRIVATE_KEY"))}
	start, err := strconv.ParseUint(strings.TrimSpace(getenv("MYFERENCE_CHAIN_START_BLOCK")), 10, 64)
	if config.RPCURL == "" || !common.IsHexAddress(config.ContractAddress) || common.HexToAddress(config.ContractAddress) == (common.Address{}) || config.SettlementPrivateKey == "" || err != nil {
		return chainConfig{}, errors.New("MYFERENCE_RPC_URL, MYFERENCE_CONTRACT_ADDRESS, MYFERENCE_SETTLEMENT_PRIVATE_KEY, and MYFERENCE_CHAIN_START_BLOCK are required")
	}
	config.StartBlock = start
	return config, nil
}

type chainRuntime struct {
	client      *chain.Client
	terms       chain.ReceiptTerms
	queue       *chain.SettlementQueue
	indexer     *chain.Indexer
	coordinator *settlement.Coordinator
}

func openChainRuntime(ctx context.Context, config chainConfig, databaseURL string, repository *store.Store, hub *relay.Hub) (*chainRuntime, error) {
	client, err := chain.Dial(ctx, config.RPCURL, config.SettlementPrivateKey)
	if err != nil {
		return nil, err
	}
	closeClient := true
	defer func() {
		if closeClient {
			client.Close()
		}
	}()
	if err := client.Bind(common.HexToAddress(config.ContractAddress)); err != nil {
		return nil, err
	}
	terms, err := client.ReceiptTerms(ctx)
	if err != nil {
		return nil, err
	}
	if terms.ChainID != 10143 && strings.ToLower(getenvOrEmpty("MYFERENCE_ALLOW_LOCAL_CHAIN")) != "true" {
		return nil, errors.New("settlement RPC is not Monad testnet chain 10143")
	}
	if terms.SettlementSigner != client.Address() {
		return nil, errors.New("settlement private key does not match the contract settlement signer")
	}
	queue, err := chain.OpenSettlementQueue(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	indexer, err := chain.OpenIndexer(ctx, chain.IndexerConfig{RPCURL: config.RPCURL, DatabaseURL: databaseURL, Contract: common.HexToAddress(config.ContractAddress), StartBlock: config.StartBlock, Confirmations: 2})
	if err != nil {
		queue.Close()
		return nil, err
	}
	closeClient = false
	return &chainRuntime{client: client, terms: terms, queue: queue, indexer: indexer, coordinator: settlement.NewCoordinator(settlement.Config{SignatureTimeout: 10 * time.Second}, repository, hub, queue, client)}, nil
}

func (r *chainRuntime) Close() { r.indexer.Close(); r.queue.Close(); r.client.Close() }

func (r *chainRuntime) Run(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := r.indexer.Sync(ctx); err != nil && ctx.Err() == nil {
					log.Printf("chain index retry: %v", err)
				}
			}
		}
	}()
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Isolate receipts so one invalid provider signature/session cannot revert
				// unrelated providers' payouts in the same EVM transaction.
				_, err := r.queue.SettleBatch(ctx, r.client, 1)
				if err != nil && !errors.Is(err, chain.ErrNoSignedReceipts) && ctx.Err() == nil {
					log.Printf("settlement batch retry: %v", err)
				}
			}
		}
	}()
}

func getenvOrEmpty(name string) string { return strings.TrimSpace(getenv(name)) }

var getenv = os.Getenv
