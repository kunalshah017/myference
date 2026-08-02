// Package provider exposes the production provider daemon without exposing the
// CLI's platform-specific implementation packages.
package provider

import (
	"context"
	"fmt"

	"github.com/kunalshah017/myference/cli/internal/backend"
	"github.com/kunalshah017/myference/cli/internal/backend/ollama"
	internal "github.com/kunalshah017/myference/cli/internal/provider"
)

type Config = internal.Config

type Daemon struct{ inner *internal.Daemon }

func NewOllama(config Config, endpoints map[string]string) (*Daemon, error) {
	backends := make(map[string]backend.Backend, len(config.Offers))
	for _, offer := range config.Offers {
		endpoint, ok := endpoints[offer.OfferID]
		if !ok {
			return nil, fmt.Errorf("missing Ollama endpoint for offer %q", offer.OfferID)
		}
		client, err := ollama.New(endpoint, nil)
		if err != nil {
			return nil, err
		}
		backends[offer.OfferID] = client
	}
	return &Daemon{inner: internal.NewDaemon(config, backends)}, nil
}

func (d *Daemon) Serve(ctx context.Context) error { return d.inner.Serve(ctx) }
