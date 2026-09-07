package registry

import (
	"sync"

	"github.com/huzilin/trade/dataflows/interfaces"
	"github.com/huzilin/trade/dataflows/normalizer"
	"github.com/pkg/errors"
)

type ProviderRegistry struct {
	mu            sync.RWMutex
	providers     map[string]interfaces.Provider                           // vendor name -> Provider
	capabilities  map[interfaces.Capability]map[string]interfaces.Provider // capability -> vendor -> Provider
	marketVendors map[normalizer.MarketType][]string                       // market -> vendors
}

func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		providers:     make(map[string]interfaces.Provider),
		capabilities:  make(map[interfaces.Capability]map[string]interfaces.Provider),
		marketVendors: make(map[normalizer.MarketType][]string),
	}
}

func (r *ProviderRegistry) RegisterProvider(p interfaces.Provider) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := p.Name()

	for _, cap := range p.Capabilities() {
		if !interfaces.IsValidCapability(cap) {
			return errors.Errorf("vendor %s declares invalid capability: %s", name, cap)
		}
	}

	r.providers[name] = p

	for _, cap := range p.Capabilities() {
		if r.capabilities[cap] == nil {
			r.capabilities[cap] = make(map[string]interfaces.Provider)
		}
		r.capabilities[cap][name] = p
	}

	for _, market := range p.Markets() {
		r.marketVendors[market] = append(r.marketVendors[market], name)
	}

	return nil
}

func (r *ProviderRegistry) GetProvider(name string) (interfaces.Provider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.providers[name]
	return p, ok
}

func (r *ProviderRegistry) GetProvidersForCapability(cap interfaces.Capability) []interfaces.Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	providers := make([]interfaces.Provider, 0, len(r.capabilities[cap]))
	for _, p := range r.capabilities[cap] {
		providers = append(providers, p)
	}
	return providers
}

func (r *ProviderRegistry) GetProviderNamesForCapability(cap interfaces.Capability) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.capabilities[cap]))
	for name := range r.capabilities[cap] {
		names = append(names, name)
	}
	return names
}

func (r *ProviderRegistry) GetProvidersForMarket(market normalizer.MarketType) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.marketVendors[market]
}

func (r *ProviderRegistry) HasCapability(vendor string, cap interfaces.Capability) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.capabilities[cap] == nil {
		return false
	}
	_, ok := r.capabilities[cap][vendor]
	return ok
}

func (r *ProviderRegistry) Capabilities() []interfaces.Capability {
	r.mu.RLock()
	defer r.mu.RUnlock()

	caps := make([]interfaces.Capability, 0, len(r.capabilities))
	for cap := range r.capabilities {
		caps = append(caps, cap)
	}
	return caps
}

func (r *ProviderRegistry) Providers() []interfaces.Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	providers := make([]interfaces.Provider, 0, len(r.providers))
	for _, p := range r.providers {
		providers = append(providers, p)
	}
	return providers
}

func (r *ProviderRegistry) ProviderNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	return names
}

func (r *ProviderRegistry) CapabilityMatrix() map[interfaces.Capability][]string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	matrix := make(map[interfaces.Capability][]string)
	for cap, vendors := range r.capabilities {
		names := make([]string, 0, len(vendors))
		for name := range vendors {
			names = append(names, name)
		}
		matrix[cap] = names
	}
	return matrix
}

func (r *ProviderRegistry) FindBestProvider(cap interfaces.Capability, market normalizer.MarketType, preferredOrder []string) (interfaces.Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if !interfaces.IsValidCapability(cap) {
		return nil, errors.Errorf("invalid capability: %s", cap)
	}

	methodProviders := r.capabilities[cap]
	if len(methodProviders) == 0 {
		return nil, errors.Errorf("no provider supports capability: %s", cap)
	}

	for _, preferred := range preferredOrder {
		if p, ok := methodProviders[preferred]; ok {
			for _, m := range p.Markets() {
				if m == market || m == normalizer.MarketGlobal {
					return p, nil
				}
			}
		}
	}

	var specializedProviders []interfaces.Provider
	var generalProviders []interfaces.Provider
	var globalProviders []interfaces.Provider

	for _, p := range methodProviders {
		markets := p.Markets()
		hasExact := false
		hasGlobal := false
		for _, m := range markets {
			if m == market {
				hasExact = true
			} else if m == normalizer.MarketGlobal {
				hasGlobal = true
			}
		}

		if hasExact && len(markets) == 1 {
			specializedProviders = append(specializedProviders, p)
		} else if hasExact {
			generalProviders = append(generalProviders, p)
		} else if hasGlobal {
			globalProviders = append(globalProviders, p)
		}
	}

	if len(specializedProviders) > 0 {
		return specializedProviders[0], nil
	}
	if len(generalProviders) > 0 {
		return generalProviders[0], nil
	}
	if len(globalProviders) > 0 {
		return globalProviders[0], nil
	}

	return nil, errors.Errorf("no provider found for capability %s and market %s", cap, market)
}

var GlobalRegistry = NewProviderRegistry()
