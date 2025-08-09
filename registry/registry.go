package registry

import (
	"fmt"
	"strings"

	"github.com/recera/gai/core"
)

type Provider interface {
	core.ProviderClient
}

type Registry struct {
	providers map[string]Provider
}

func New() *Registry { return &Registry{providers: map[string]Provider{}} }

func (r *Registry) Register(name string, p Provider) { r.providers[strings.ToLower(name)] = p }

func (r *Registry) Resolve(key string) (Provider, string, error) {
	parts := strings.SplitN(key, ":", 2)
	if len(parts) != 2 {
		return nil, "", fmt.Errorf("invalid model key: %s", key)
	}
	p, ok := r.providers[strings.ToLower(parts[0])]
	if !ok {
		return nil, "", fmt.Errorf("provider not registered: %s", parts[0])
	}
	return p, parts[1], nil
}
