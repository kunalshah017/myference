package provider

import (
	"errors"
	"sync"

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
