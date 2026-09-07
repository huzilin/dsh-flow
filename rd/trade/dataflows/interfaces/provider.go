package interfaces

import (
	"github.com/huzilin/trade/dataflows/normalizer"
)

type Provider interface {
	Name() string
	Capabilities() []Capability
	Markets() []normalizer.MarketType
}

type CapabilityInfo struct {
	Method   Capability
	Markets  []normalizer.MarketType
	Params   []string
	Priority int
}

type ProviderInfo struct {
	Name         string
	Capabilities []CapabilityInfo
	Markets      []normalizer.MarketType
	Instance     Provider
}
