package settlement

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	v1 "github.com/kunalshah017/myference/protocol/v1"
	"github.com/kunalshah017/myference/server/internal/chain"
	"github.com/kunalshah017/myference/server/internal/relay"
	"github.com/kunalshah017/myference/server/internal/store"
)

type Config struct{ SignatureTimeout time.Duration }

type Coordinator struct {
	config     Config
	repository *store.Store
	hub        *relay.Hub
	queue      *chain.SettlementQueue
	broker     *chain.Client
}

func NewCoordinator(config Config, repository *store.Store, hub *relay.Hub, queue *chain.SettlementQueue, broker *chain.Client) *Coordinator {
	if config.SignatureTimeout <= 0 {
		config.SignatureTimeout = 10 * time.Second
	}
	return &Coordinator{config: config, repository: repository, hub: hub, queue: queue, broker: broker}
}

func (c *Coordinator) Complete(ctx context.Context, completed store.ReceiptProposal) error {
	if c.repository == nil || c.hub == nil || c.queue == nil || c.broker == nil {
		return errors.New("settlement coordinator is not configured")
	}
	if err := c.repository.CompleteInference(ctx, completed); err != nil {
		return err
	}
	terms, err := c.broker.ReceiptTerms(ctx)
	if err != nil {
		return err
	}
	receipt, machineID, expectedSigner, err := c.repository.PrepareReceipt(ctx, completed.RequestID, store.ReceiptDomain{ChainID: terms.ChainID, ContractAddress: terms.Contract.Hex(), SettlementSigner: terms.SettlementSigner.Hex(), FeeBasisPoints: terms.FeeBasisPoints, FeeVersion: terms.FeeVersion})
	if err != nil {
		return err
	}
	var contract v1.Address
	copy(contract[:], terms.Contract.Bytes())
	events, cancel := c.hub.Subscribe(completed.RequestID)
	defer cancel()
	proposal := v1.ReceiptProposal{RequestID: completed.RequestID, ChainID: terms.ChainID, Contract: contract, Receipt: receipt}
	envelope, err := v1.NewEnvelope("receipt-"+completed.RequestID, v1.MessageReceiptProposal, &proposal)
	if err != nil {
		return err
	}
	if err := c.hub.Send(machineID, envelope); err != nil {
		return err
	}
	timer := time.NewTimer(c.config.SignatureTimeout)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			return errors.New("provider receipt signature timed out")
		case event := <-events:
			if event.Type != v1.MessageReceiptSignature || event.MachineID != machineID {
				continue
			}
			var signed v1.ReceiptSignature
			if err := event.Envelope.DecodeBody(&signed); err != nil {
				return err
			}
			recovered, err := v1.RecoverReceiptSigner(receipt, terms.ChainID, contract, signed.Signature)
			if err != nil || recovered != signed.Signer || !strings.EqualFold(common.BytesToAddress(recovered[:]).Hex(), expectedSigner) {
				return errors.New("provider receipt signature is invalid")
			}
			allowed, err := c.repository.ProviderSignerAllowed(ctx, terms.ChainID, terms.Contract.Hex(), receipt.Provider, signed.Signer)
			if err != nil || !allowed {
				return errors.New("provider receipt signer is not authorized")
			}
			settlementSignature, err := c.broker.SignReceipt(ctx, toChainReceipt(receipt))
			if err != nil {
				return err
			}
			return c.queue.Enqueue(ctx, chain.SignedReceipt{Receipt: toChainReceipt(receipt), ProviderSignature: signed.Signature, SettlementSignature: settlementSignature})
		}
	}
}

func toChainReceipt(value v1.Receipt) chain.Receipt {
	return chain.Receipt{RequestId: [32]byte(value.RequestID), SessionId: [32]byte(value.SessionID), Customer: common.BytesToAddress(value.Customer[:]), Provider: common.BytesToAddress(value.Provider[:]), SettlementSigner: common.BytesToAddress(value.SettlementSigner[:]), OfferId: [32]byte(value.OfferID), PriceVersion: value.PriceVersion, ModelHash: [32]byte(value.ModelHash), CapabilityHash: [32]byte(value.CapabilityHash), InputTokens: value.InputTokens, OutputTokens: value.OutputTokens, ComputeMilliseconds: value.ComputeMilliseconds, MaximumCharge: value.MaximumCharge, TotalCharge: value.TotalCharge, FeeBasisPoints: value.FeeBasisPoints, FeeVersion: value.FeeVersion, Status: uint8(value.Status), CompletedAt: value.CompletedAt, InputHash: [32]byte(value.InputHash), OutputHash: [32]byte(value.OutputHash), Nonce: value.Nonce}
}
