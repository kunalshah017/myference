package backend

import "context"

type Model struct {
	Name   string
	Digest string
	Size   int64
}

type Request struct {
	Model  string
	Prompt string
}

type Usage struct {
	InputTokens         uint64
	OutputTokens        uint64
	ComputeMilliseconds uint64
}

type Backend interface {
	Models(context.Context) ([]Model, error)
	Generate(context.Context, Request, func(string) error) (Usage, error)
}
