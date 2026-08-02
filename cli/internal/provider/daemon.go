package provider

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/kunalshah017/myference/cli/internal/backend"
	v1 "github.com/kunalshah017/myference/protocol/v1"
)

var (
	ErrLeaseAlreadyAccepted = errors.New("lease already accepted")
	ErrRequestTerminal      = errors.New("request is terminal")
	ErrBackpressure         = errors.New("outbound queue full")
)

type Cursor struct {
	Sequence uint64
	Done     bool
}

type request struct {
	accepted   bool
	sequence   uint64
	terminal   bool
	outputSeen bool
}

type RequestState struct {
	mu       sync.Mutex
	requests map[string]*request
}

func NewRequestState() *RequestState {
	return &RequestState{requests: make(map[string]*request)}
}

func (s *RequestState) Accept(requestID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing := s.requests[requestID]; existing != nil && existing.accepted {
		return ErrLeaseAlreadyAccepted
	}
	s.requests[requestID] = &request{accepted: true}
	return nil
}

func (s *RequestState) RecordChunk(chunk v1.OutputChunk) error {
	if err := chunk.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	current := s.requests[chunk.RequestID]
	if current == nil || !current.accepted || current.terminal {
		return ErrRequestTerminal
	}
	if chunk.Sequence != current.sequence+1 {
		return v1.ErrChunkSequence
	}
	current.sequence = chunk.Sequence
	current.outputSeen = true
	current.terminal = chunk.Done
	return nil
}

func (s *RequestState) Cancel(requestID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	current := s.requests[requestID]
	if current == nil || current.terminal {
		return ErrRequestTerminal
	}
	current.terminal = true
	return nil
}

func (s *RequestState) CanRetry(requestID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	current := s.requests[requestID]
	return current == nil || !current.outputSeen
}

func (s *RequestState) Cursor(requestID string) Cursor {
	s.mu.Lock()
	defer s.mu.Unlock()
	current := s.requests[requestID]
	if current == nil {
		return Cursor{}
	}
	return Cursor{Sequence: current.sequence, Done: current.terminal}
}

func (s *RequestState) Restore(requestID string, cursor Cursor) {
	s.mu.Lock()
	s.requests[requestID] = &request{accepted: true, sequence: cursor.Sequence, terminal: cursor.Done, outputSeen: cursor.Sequence > 0}
	s.mu.Unlock()
}

type OutboundQueue struct{ messages chan v1.Envelope }

func NewOutboundQueue(size int) *OutboundQueue {
	return &OutboundQueue{messages: make(chan v1.Envelope, size)}
}

func (q *OutboundQueue) TryPush(message v1.Envelope) error {
	select {
	case q.messages <- message:
		return nil
	default:
		return ErrBackpressure
	}
}

func (q *OutboundQueue) Messages() <-chan v1.Envelope { return q.messages }

type Config struct {
	RelayURL          string
	Token             string
	MachineID         string
	Offers            []v1.OfferCapacity
	HTTPClient        *http.Client
	HeartbeatInterval time.Duration
	DrainTimeout      time.Duration
	SignerKey         *ecdsa.PrivateKey
	ChainID           uint64
	Contract          v1.Address
}

type Daemon struct {
	config     Config
	backends   map[string]backend.Backend
	backendsMu sync.RWMutex
	state      *RequestState
	writeMu    sync.Mutex
	jobsMu     sync.Mutex
	jobs       map[string]context.CancelFunc
	wg         sync.WaitGroup
}

func NewDaemon(config Config, backends map[string]backend.Backend) *Daemon {
	if config.HeartbeatInterval <= 0 {
		config.HeartbeatInterval = 10 * time.Second
	}
	if config.DrainTimeout <= 0 {
		config.DrainTimeout = 30 * time.Second
	}
	return &Daemon{config: config, backends: backends, state: NewRequestState(), jobs: make(map[string]context.CancelFunc)}
}

func (d *Daemon) Serve(ctx context.Context) error {
	if err := d.validate(); err != nil {
		return err
	}
	connection, _, err := websocket.Dial(ctx, d.config.RelayURL, &websocket.DialOptions{
		HTTPClient: d.config.HTTPClient,
		HTTPHeader: http.Header{"Authorization": []string{"Bearer " + d.config.Token}},
	})
	if err != nil {
		return fmt.Errorf("connect relay: %w", err)
	}
	heartbeatCtx, stopHeartbeat := context.WithCancel(ctx)
	heartbeatDone := make(chan struct{})
	heartbeatStarted := false
	defer func() {
		if heartbeatStarted {
			stopHeartbeat()
			<-heartbeatDone
		}
		if ctx.Err() == nil {
			d.cancelAll()
			d.wg.Wait()
		} else {
			drained := make(chan struct{})
			go func() { d.wg.Wait(); close(drained) }()
			timer := time.NewTimer(d.config.DrainTimeout)
			select {
			case <-drained:
				if !timer.Stop() {
					<-timer.C
				}
			case <-timer.C:
				d.cancelAll()
				<-drained
			}
		}
		connection.Close(websocket.StatusNormalClosure, "provider stopped")
	}()
	if err := d.send(ctx, connection, v1.MessageHello, &v1.Hello{MachineID: d.config.MachineID}); err != nil {
		return err
	}
	capacity := d.Capacity()
	if err := d.send(ctx, connection, v1.MessageCapacity, &capacity); err != nil {
		return err
	}
	go d.heartbeat(heartbeatCtx, connection, heartbeatDone)
	heartbeatStarted = true
	for {
		_, payload, err := connection.Read(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return fmt.Errorf("read relay: %w", err)
		}
		envelope, err := v1.DecodeEnvelope(bytes.NewReader(payload), 1<<20)
		if err != nil {
			return err
		}
		switch envelope.Type {
		case v1.MessageJobOffer:
			var offer v1.JobOffer
			if err := envelope.DecodeBody(&offer); err != nil {
				return err
			}
			if err := d.startJob(ctx, connection, offer); err != nil {
				return err
			}
		case v1.MessageCancel:
			var cancel v1.Cancel
			if err := envelope.DecodeBody(&cancel); err != nil {
				return err
			}
			d.cancel(cancel.RequestID)
		case v1.MessageReceiptProposal:
			var proposal v1.ReceiptProposal
			if err := envelope.DecodeBody(&proposal); err != nil {
				return err
			}
			signature, err := d.signProposal(proposal)
			if err != nil {
				return err
			}
			if err := d.send(ctx, connection, v1.MessageReceiptSignature, &signature); err != nil {
				return err
			}
		default:
			return v1.ErrInvalidMessage
		}
	}
}

func (d *Daemon) signProposal(proposal v1.ReceiptProposal) (v1.ReceiptSignature, error) {
	if d.config.SignerKey == nil || proposal.ChainID != d.config.ChainID || proposal.Contract != d.config.Contract {
		return v1.ReceiptSignature{}, errors.New("receipt proposal domain does not match this machine")
	}
	signerAddress := crypto.PubkeyToAddress(d.config.SignerKey.PublicKey)
	var signer v1.Address
	copy(signer[:], signerAddress.Bytes())
	signature, err := v1.SignReceipt(proposal.Receipt, proposal.ChainID, proposal.Contract, d.config.SignerKey)
	if err != nil {
		return v1.ReceiptSignature{}, err
	}
	return v1.ReceiptSignature{RequestID: proposal.RequestID, Signer: signer, Signature: signature}, nil
}

func (d *Daemon) heartbeat(ctx context.Context, connection *websocket.Conn, done chan<- struct{}) {
	defer close(done)
	ticker := time.NewTicker(d.config.HeartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			capacity := d.Capacity()
			if err := d.send(ctx, connection, v1.MessageCapacity, &capacity); err != nil {
				connection.Close(websocket.StatusInternalError, "heartbeat failed")
				return
			}
		}
	}
}

func (d *Daemon) validate() error {
	if strings.TrimSpace(d.config.RelayURL) == "" || strings.TrimSpace(d.config.Token) == "" || strings.TrimSpace(d.config.MachineID) == "" {
		return errors.New("relay URL, machine token, and machine ID are required")
	}
	if err := validateBackends(d.config.Offers, d.backends); err != nil {
		return err
	}
	return nil
}

func validateBackends(offers []v1.OfferCapacity, backends map[string]backend.Backend) error {
	if len(offers) == 0 {
		return errors.New("at least one enabled backend is required")
	}
	for _, offer := range offers {
		if err := offer.Validate(); err != nil || backends[offer.OfferID] == nil {
			return errors.New("every offer requires a valid backend")
		}
	}
	return nil
}

func (d *Daemon) UpdateBackends(offers []v1.OfferCapacity, backends map[string]backend.Backend) error {
	if len(offers) > 0 {
		if err := validateBackends(offers, backends); err != nil {
			return err
		}
	}
	d.backendsMu.Lock()
	d.config.Offers = append([]v1.OfferCapacity(nil), offers...)
	d.backends = backends
	d.backendsMu.Unlock()
	return nil
}

func (d *Daemon) Capacity() v1.Capacity {
	d.backendsMu.RLock()
	defer d.backendsMu.RUnlock()
	offers := append([]v1.OfferCapacity(nil), d.config.Offers...)
	return v1.Capacity{Available: uint32(len(offers)), Offers: offers}
}

func (d *Daemon) startJob(parent context.Context, connection *websocket.Conn, offer v1.JobOffer) error {
	d.backendsMu.RLock()
	selected := d.backends[offer.OfferID]
	configured := false
	for _, candidate := range d.config.Offers {
		if candidate.OfferID == offer.OfferID && candidate.Model == offer.Model && candidate.PriceVersion == offer.PriceVersion {
			configured = true
			break
		}
	}
	d.backendsMu.RUnlock()
	if selected == nil || !configured || time.Now().After(offer.LeaseExpiresAt) {
		return v1.ErrInvalidMessage
	}
	if err := d.state.Accept(offer.RequestID); err != nil {
		return err
	}
	jobCtx, cancel := context.WithCancel(context.WithoutCancel(parent))
	d.jobsMu.Lock()
	d.jobs[offer.RequestID] = cancel
	d.jobsMu.Unlock()
	if err := d.send(parent, connection, v1.MessageJobAccept, &v1.JobAccept{RequestID: offer.RequestID}); err != nil {
		cancel()
		d.removeJob(offer.RequestID)
		return err
	}
	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		defer d.removeJob(offer.RequestID)
		sequence := uint64(0)
		workspace := make([]backend.WorkspaceFile, 0, len(offer.Workspace))
		for _, file := range offer.Workspace {
			content, decodeErr := base64.StdEncoding.DecodeString(file.ContentBase64)
			if decodeErr != nil {
				return
			}
			workspace = append(workspace, backend.WorkspaceFile{Path: file.Path, Content: content})
		}
		usage, err := selected.Generate(jobCtx, backend.Request{Model: offer.Model, Prompt: offer.Prompt, Workspace: workspace, MaximumOutputTokens: offer.MaximumOutputTokens}, func(content string) error {
			sequence++
			return d.sendChunk(jobCtx, connection, v1.OutputChunk{RequestID: offer.RequestID, Sequence: sequence, Data: content})
		})
		sequence++
		if err != nil {
			failureContext, stopFailure := context.WithTimeout(context.WithoutCancel(jobCtx), 5*time.Second)
			defer stopFailure()
			_ = d.sendChunk(failureContext, connection, v1.OutputChunk{RequestID: offer.RequestID, Sequence: sequence, Done: true, ErrorCode: "backend_failed"})
			return
		}
		_ = d.sendChunk(jobCtx, connection, v1.OutputChunk{RequestID: offer.RequestID, Sequence: sequence, Done: true, InputTokens: usage.InputTokens, OutputTokens: usage.OutputTokens, ComputeMilliseconds: usage.ComputeMilliseconds})
	}()
	return nil
}

func (d *Daemon) sendChunk(ctx context.Context, connection *websocket.Conn, chunk v1.OutputChunk) error {
	if err := d.state.RecordChunk(chunk); err != nil {
		return err
	}
	return d.send(ctx, connection, v1.MessageOutputChunk, &chunk)
}

func (d *Daemon) send(ctx context.Context, connection *websocket.Conn, messageType string, message v1.Validatable) error {
	envelope, err := v1.NewEnvelope(fmt.Sprintf("%s-%d", messageType, time.Now().UnixNano()), messageType, message)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(envelope)
	if err != nil {
		return err
	}
	d.writeMu.Lock()
	defer d.writeMu.Unlock()
	if err := connection.Write(ctx, websocket.MessageText, payload); err != nil {
		return fmt.Errorf("write relay: %w", err)
	}
	return nil
}

func (d *Daemon) cancel(requestID string) {
	d.jobsMu.Lock()
	cancel := d.jobs[requestID]
	d.jobsMu.Unlock()
	if cancel != nil {
		cancel()
		_ = d.state.Cancel(requestID)
	}
}

func (d *Daemon) cancelAll() {
	d.jobsMu.Lock()
	defer d.jobsMu.Unlock()
	for _, cancel := range d.jobs {
		cancel()
	}
}

func (d *Daemon) removeJob(requestID string) {
	d.jobsMu.Lock()
	delete(d.jobs, requestID)
	d.jobsMu.Unlock()
}
