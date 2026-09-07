package registry

import (
	"testing"

	"github.com/huzilin/trade/dataflows/interfaces"
	"github.com/huzilin/trade/dataflows/normalizer"
	"github.com/stretchr/testify/assert"
)

type mockProvider struct {
	name         string
	capabilities []interfaces.Capability
	markets      []normalizer.MarketType
}

func (m *mockProvider) Name() string                          { return m.name }
func (m *mockProvider) Capabilities() []interfaces.Capability { return m.capabilities }
func (m *mockProvider) Markets() []normalizer.MarketType      { return m.markets }

var _ interfaces.Provider = (*mockProvider)(nil)

func TestUnit_NewProviderRegistry(t *testing.T) {
	reg := NewProviderRegistry()
	assert.NotNil(t, reg)
}

func TestUnit_RegisterProvider(t *testing.T) {
	reg := NewProviderRegistry()

	p := &mockProvider{
		name:         "test_provider",
		capabilities: []interfaces.Capability{interfaces.CapabilityStockData},
		markets:      []normalizer.MarketType{normalizer.MarketUSShare},
	}

	reg.RegisterProvider(p)

	got, ok := reg.GetProvider("test_provider")
	assert.True(t, ok)
	assert.Equal(t, p, got)

	methods := reg.GetProviderNamesForCapability(interfaces.CapabilityStockData)
	assert.Contains(t, methods, "test_provider")

	markets := reg.GetProvidersForMarket(normalizer.MarketUSShare)
	assert.Contains(t, markets, "test_provider")
}

func TestUnit_GetProvidersForCapability(t *testing.T) {
	reg := NewProviderRegistry()

	p1 := &mockProvider{name: "p1", capabilities: []interfaces.Capability{interfaces.CapabilityStockData}, markets: []normalizer.MarketType{normalizer.MarketUSShare}}
	p2 := &mockProvider{name: "p2", capabilities: []interfaces.Capability{interfaces.CapabilityStockData, interfaces.CapabilityIndicators}, markets: []normalizer.MarketType{normalizer.MarketAshare}}

	reg.RegisterProvider(p1)
	reg.RegisterProvider(p2)

	providers := reg.GetProvidersForCapability(interfaces.CapabilityStockData)
	assert.Len(t, providers, 2)

	providers = reg.GetProvidersForCapability(interfaces.CapabilityIndicators)
	assert.Len(t, providers, 1)
	assert.Equal(t, "p2", providers[0].Name())

	providers = reg.GetProvidersForCapability("nonexistent")
	assert.Len(t, providers, 0)
}

func TestUnit_GetProviderNamesForCapability(t *testing.T) {
	reg := NewProviderRegistry()

	p1 := &mockProvider{name: "p1", capabilities: []interfaces.Capability{interfaces.CapabilityStockData}, markets: []normalizer.MarketType{normalizer.MarketUSShare}}
	p2 := &mockProvider{name: "p2", capabilities: []interfaces.Capability{interfaces.CapabilityStockData}, markets: []normalizer.MarketType{normalizer.MarketAshare}}

	reg.RegisterProvider(p1)
	reg.RegisterProvider(p2)

	names := reg.GetProviderNamesForCapability(interfaces.CapabilityStockData)
	assert.Len(t, names, 2)
	assert.Contains(t, names, "p1")
	assert.Contains(t, names, "p2")
}

func TestUnit_HasCapability(t *testing.T) {
	reg := NewProviderRegistry()

	p := &mockProvider{name: "test", capabilities: []interfaces.Capability{interfaces.CapabilityStockData}, markets: []normalizer.MarketType{normalizer.MarketUSShare}}
	reg.RegisterProvider(p)

	assert.True(t, reg.HasCapability("test", interfaces.CapabilityStockData))
	assert.False(t, reg.HasCapability("test", interfaces.CapabilityIndicators))
	assert.False(t, reg.HasCapability("nonexistent", interfaces.CapabilityStockData))
}

func TestUnit_Capabilities(t *testing.T) {
	reg := NewProviderRegistry()

	p1 := &mockProvider{name: "p1", capabilities: []interfaces.Capability{interfaces.CapabilityStockData}, markets: []normalizer.MarketType{normalizer.MarketUSShare}}
	p2 := &mockProvider{name: "p2", capabilities: []interfaces.Capability{interfaces.CapabilityStockData, interfaces.CapabilityIndicators}, markets: []normalizer.MarketType{normalizer.MarketAshare}}

	reg.RegisterProvider(p1)
	reg.RegisterProvider(p2)

	caps := reg.Capabilities()
	assert.Len(t, caps, 2)
	assert.Contains(t, caps, interfaces.CapabilityStockData)
	assert.Contains(t, caps, interfaces.CapabilityIndicators)
}

func TestUnit_ProviderNames(t *testing.T) {
	reg := NewProviderRegistry()

	p1 := &mockProvider{name: "p1", capabilities: []interfaces.Capability{interfaces.CapabilityStockData}, markets: []normalizer.MarketType{normalizer.MarketUSShare}}
	p2 := &mockProvider{name: "p2", capabilities: []interfaces.Capability{interfaces.CapabilityIndicators}, markets: []normalizer.MarketType{normalizer.MarketAshare}}

	reg.RegisterProvider(p1)
	reg.RegisterProvider(p2)

	names := reg.ProviderNames()
	assert.Len(t, names, 2)
	assert.Contains(t, names, "p1")
	assert.Contains(t, names, "p2")
}

func TestUnit_CapabilityMatrix(t *testing.T) {
	reg := NewProviderRegistry()

	p1 := &mockProvider{name: "p1", capabilities: []interfaces.Capability{interfaces.CapabilityStockData}, markets: []normalizer.MarketType{normalizer.MarketUSShare}}
	p2 := &mockProvider{name: "p2", capabilities: []interfaces.Capability{interfaces.CapabilityStockData, interfaces.CapabilityIndicators}, markets: []normalizer.MarketType{normalizer.MarketAshare}}

	reg.RegisterProvider(p1)
	reg.RegisterProvider(p2)

	matrix := reg.CapabilityMatrix()
	assert.Len(t, matrix, 2)
	assert.Len(t, matrix[interfaces.CapabilityStockData], 2)
	assert.Len(t, matrix[interfaces.CapabilityIndicators], 1)
}

func TestUnit_FindBestProvider(t *testing.T) {
	reg := NewProviderRegistry()

	p1 := &mockProvider{name: "p1", capabilities: []interfaces.Capability{interfaces.CapabilityStockData}, markets: []normalizer.MarketType{normalizer.MarketUSShare}}
	p2 := &mockProvider{name: "p2", capabilities: []interfaces.Capability{interfaces.CapabilityStockData}, markets: []normalizer.MarketType{normalizer.MarketAshare}}
	p3 := &mockProvider{name: "p3", capabilities: []interfaces.Capability{interfaces.CapabilityStockData}, markets: []normalizer.MarketType{normalizer.MarketGlobal}}

	reg.RegisterProvider(p1)
	reg.RegisterProvider(p2)
	reg.RegisterProvider(p3)

	p, err := reg.FindBestProvider(interfaces.CapabilityStockData, normalizer.MarketUSShare, []string{"p1", "p2"})
	assert.NoError(t, err)
	assert.Equal(t, "p1", p.Name())

	p, err = reg.FindBestProvider(interfaces.CapabilityStockData, normalizer.MarketHKShare, []string{"p3"})
	assert.NoError(t, err)
	assert.Equal(t, "p3", p.Name())

	p, err = reg.FindBestProvider(interfaces.CapabilityStockData, normalizer.MarketAshare, []string{})
	assert.NoError(t, err)
	assert.NotNil(t, p)

	_, err = reg.FindBestProvider("nonexistent", normalizer.MarketUSShare, []string{"p1"})
	assert.Error(t, err)
}
