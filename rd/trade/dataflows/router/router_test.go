package router

import (
	"testing"

	"github.com/huzilin/trade/dataflows/normalizer"
	_ "github.com/huzilin/trade/dataflows/vendors"
	"github.com/stretchr/testify/assert"
)

func TestUnit_NewDefaultRouter(t *testing.T) {
	r := NewDefaultRouter()
	assert.NotNil(t, r)
	assert.NotNil(t, r.normalizer)
	assert.NotNil(t, r.config)
}

func TestUnit_Router_DetectMarketVendor(t *testing.T) {
	r := NewDefaultRouter()

	tests := []struct {
		name       string
		symbol     string
		wantMarket normalizer.MarketType
		wantVendor string
	}{
		{"A-share", "000001", normalizer.MarketAshare, "akshare"},
		{"HK-share", "00700", normalizer.MarketHKShare, "yfinance"},
		{"US-share", "AAPL", normalizer.MarketUSShare, "yfinance"},
		{"unknown", "INVALID", normalizer.MarketUnknown, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			market, vendor := r.detectMarketVendor(tt.symbol)
			assert.Equal(t, tt.wantMarket, market)
			assert.Equal(t, tt.wantVendor, vendor)
		})
	}
}

func TestUnit_Router_GetCapabilityMatrix(t *testing.T) {
	r := NewDefaultRouter()
	matrix := r.GetCapabilityMatrix()
	assert.NotEmpty(t, matrix)
}

func TestUnit_Router_GetSupportedMethods(t *testing.T) {
	r := NewDefaultRouter()
	methods := r.GetSupportedMethods()
	assert.NotEmpty(t, methods)
}
