package orchestrator

import (
	"github.com/huzilin/trade/dataflows/interfaces"
	"github.com/huzilin/trade/dataflows/normalizer"
	"github.com/huzilin/trade/dataflows/registry"
)

type VendorStatus int

const (
	StatusNotInitialized VendorStatus = iota
	StatusReady
	StatusMissingConfig
	StatusError
)

type VendorInfo struct {
	Name           string
	Status         VendorStatus
	Capabilities   []interfaces.Capability
	Markets        []normalizer.MarketType
	RequiredConfig []string
}

type OrchestrationRule struct {
	Market     normalizer.MarketType
	Capability interfaces.Capability
	Vendors    []string
}

type OrchestrationConfig struct {
	Rules []OrchestrationRule
}

type Orchestrator interface {
	SelectVendor(capability interfaces.Capability, market normalizer.MarketType) (string, error)
	GetVendorChain(capability interfaces.Capability, market normalizer.MarketType) ([]string, error)
	GetVendorStatus(vendorName string) VendorStatus
	UpdateVendorStatus(vendorName string, status VendorStatus)
	SetConfig(config *OrchestrationConfig)
}

type DefaultOrchestrator struct {
	registry     *registry.ProviderRegistry
	config       *OrchestrationConfig
	vendorStatus map[string]VendorStatus
}

func NewDefaultOrchestrator(reg *registry.ProviderRegistry) *DefaultOrchestrator {
	return &DefaultOrchestrator{
		registry:     reg,
		config:       &OrchestrationConfig{Rules: defaultRules()},
		vendorStatus: make(map[string]VendorStatus),
	}
}

func (o *DefaultOrchestrator) SetConfig(config *OrchestrationConfig) {
	if config != nil {
		o.config = config
	}
}

func (o *DefaultOrchestrator) GetVendorStatus(vendorName string) VendorStatus {
	status, ok := o.vendorStatus[vendorName]
	if !ok {
		return StatusNotInitialized
	}
	return status
}

func (o *DefaultOrchestrator) UpdateVendorStatus(vendorName string, status VendorStatus) {
	o.vendorStatus[vendorName] = status
}

func (o *DefaultOrchestrator) SelectVendor(capability interfaces.Capability, market normalizer.MarketType) (string, error) {
	chain, err := o.GetVendorChain(capability, market)
	if err != nil {
		return "", err
	}
	if len(chain) == 0 {
		return "", nil
	}
	return chain[0], nil
}

func (o *DefaultOrchestrator) GetVendorChain(capability interfaces.Capability, market normalizer.MarketType) ([]string, error) {
	var chain []string
	seen := make(map[string]bool)

	rules := o.getRulesFor(capability, market)

	for _, rule := range rules {
		for _, vendorName := range rule.Vendors {
			if seen[vendorName] {
				continue
			}

			if !o.registry.HasCapability(vendorName, capability) {
				continue
			}

			if !o.supportsMarket(vendorName, market) {
				continue
			}

			status := o.GetVendorStatus(vendorName)
			if status == StatusError {
				continue
			}

			chain = append(chain, vendorName)
			seen[vendorName] = true
		}
	}

	fallbackChain := o.getFallbackChain(capability, market, seen)
	chain = append(chain, fallbackChain...)

	if len(chain) == 0 {
		return nil, nil
	}

	return chain, nil
}

func (o *DefaultOrchestrator) getRulesFor(capability interfaces.Capability, market normalizer.MarketType) []OrchestrationRule {
	var rules []OrchestrationRule

	for _, rule := range o.config.Rules {
		if rule.Capability == capability && rule.Market == market {
			rules = append(rules, rule)
		}
	}

	for _, rule := range o.config.Rules {
		if rule.Capability == capability && rule.Market == normalizer.MarketGlobal {
			rules = append(rules, rule)
		}
	}

	return rules
}

func (o *DefaultOrchestrator) getFallbackChain(capability interfaces.Capability, market normalizer.MarketType, exclude map[string]bool) []string {
	var chain []string
	providers := o.registry.GetProvidersForCapability(capability)

	for _, p := range providers {
		name := p.Name()
		if exclude[name] {
			continue
		}
		if !o.supportsMarket(name, market) {
			continue
		}
		status := o.GetVendorStatus(name)
		if status == StatusError {
			continue
		}
		chain = append(chain, name)
	}

	return chain
}

func (o *DefaultOrchestrator) supportsMarket(vendorName string, market normalizer.MarketType) bool {
	providers := o.registry.Providers()
	for _, p := range providers {
		if p.Name() == vendorName {
			for _, m := range p.Markets() {
				if m == market || m == normalizer.MarketGlobal {
					return true
				}
			}
			break
		}
	}
	return false
}

func defaultRules() []OrchestrationRule {
	return []OrchestrationRule{
		{Market: normalizer.MarketAshare, Capability: interfaces.CapabilityStockData, Vendors: []string{"akshare"}},
		{Market: normalizer.MarketAshare, Capability: interfaces.CapabilityIndicators, Vendors: []string{"akshare"}},
		{Market: normalizer.MarketAshare, Capability: interfaces.CapabilityFundamentals, Vendors: []string{"akshare", "eastmoney_mx"}},
		{Market: normalizer.MarketAshare, Capability: interfaces.CapabilityBalanceSheet, Vendors: []string{"akshare", "eastmoney_mx"}},
		{Market: normalizer.MarketAshare, Capability: interfaces.CapabilityCashflow, Vendors: []string{"akshare", "eastmoney_mx"}},
		{Market: normalizer.MarketAshare, Capability: interfaces.CapabilityIncomeStatement, Vendors: []string{"akshare", "eastmoney_mx"}},
		{Market: normalizer.MarketAshare, Capability: interfaces.CapabilityNews, Vendors: []string{"eastmoney_mx", "akshare"}},
		{Market: normalizer.MarketAshare, Capability: interfaces.CapabilityCompanyAnnouncements, Vendors: []string{"eastmoney_mx"}},

		{Market: normalizer.MarketHKShare, Capability: interfaces.CapabilityStockData, Vendors: []string{"yfinance"}},
		{Market: normalizer.MarketHKShare, Capability: interfaces.CapabilityIndicators, Vendors: []string{"yfinance"}},
		{Market: normalizer.MarketHKShare, Capability: interfaces.CapabilityFundamentals, Vendors: []string{"yfinance"}},
		{Market: normalizer.MarketHKShare, Capability: interfaces.CapabilityBalanceSheet, Vendors: []string{"yfinance"}},
		{Market: normalizer.MarketHKShare, Capability: interfaces.CapabilityCashflow, Vendors: []string{"yfinance"}},
		{Market: normalizer.MarketHKShare, Capability: interfaces.CapabilityIncomeStatement, Vendors: []string{"yfinance"}},
		{Market: normalizer.MarketHKShare, Capability: interfaces.CapabilityNews, Vendors: []string{"yfinance"}},
		{Market: normalizer.MarketHKShare, Capability: interfaces.CapabilityCompanyAnnouncements, Vendors: []string{"yfinance"}},

		{Market: normalizer.MarketUSShare, Capability: interfaces.CapabilityStockData, Vendors: []string{"yfinance", "alpha_vantage"}},
		{Market: normalizer.MarketUSShare, Capability: interfaces.CapabilityIndicators, Vendors: []string{"yfinance", "alpha_vantage"}},
		{Market: normalizer.MarketUSShare, Capability: interfaces.CapabilityFundamentals, Vendors: []string{"alpha_vantage", "yfinance"}},
		{Market: normalizer.MarketUSShare, Capability: interfaces.CapabilityBalanceSheet, Vendors: []string{"alpha_vantage", "yfinance"}},
		{Market: normalizer.MarketUSShare, Capability: interfaces.CapabilityCashflow, Vendors: []string{"alpha_vantage", "yfinance"}},
		{Market: normalizer.MarketUSShare, Capability: interfaces.CapabilityIncomeStatement, Vendors: []string{"alpha_vantage", "yfinance"}},
		{Market: normalizer.MarketUSShare, Capability: interfaces.CapabilityNews, Vendors: []string{"alpha_vantage", "yfinance"}},
		{Market: normalizer.MarketUSShare, Capability: interfaces.CapabilityInsiderTransactions, Vendors: []string{"alpha_vantage"}},

		{Market: normalizer.MarketGlobal, Capability: interfaces.CapabilityMacroIndicators, Vendors: []string{"fred"}},
		{Market: normalizer.MarketGlobal, Capability: interfaces.CapabilityPredictionMarkets, Vendors: []string{"polymarket"}},
		{Market: normalizer.MarketGlobal, Capability: interfaces.CapabilityVerifiedMarketSnapshot, Vendors: []string{"yfinance"}},
	}
}
