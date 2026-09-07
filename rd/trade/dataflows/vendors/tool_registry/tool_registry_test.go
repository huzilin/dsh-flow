package tool_registry

import (
	"context"
	"testing"
	"time"

	"github.com/huzilin/trade/dataflows/models"
	"github.com/stretchr/testify/assert"
)

func TestUnit_NewRegistry(t *testing.T) {
	registry := NewRegistry()
	assert.NotNil(t, registry)
	assert.NotNil(t, registry.GetStockDataTools)
}

func TestUnit_RegisterGetStockData(t *testing.T) {
	registry := NewRegistry()

	testFunc := func(ctx context.Context, symbol, startDate, endDate string) ([]*models.Candle, error) {
		return nil, nil
	}

	registry.RegisterGetStockData("test_vendor", testFunc)

	vendors := registry.GetStockDataVendors()
	assert.Contains(t, vendors, "test_vendor")

	fn, ok := registry.GetGetStockData("test_vendor")
	assert.True(t, ok)
	assert.NotNil(t, fn)
}

func TestUnit_GetStockDataVendors(t *testing.T) {
	registry := NewRegistry()

	testFunc := func(ctx context.Context, symbol, startDate, endDate string) ([]*models.Candle, error) {
		return nil, nil
	}

	registry.RegisterGetStockData("vendor1", testFunc)
	registry.RegisterGetStockData("vendor2", testFunc)

	vendors := registry.GetStockDataVendors()
	assert.Len(t, vendors, 2)

	_, ok := registry.GetGetStockData("unknown_vendor")
	assert.False(t, ok)
}

func TestUnit_RegisterGetMacroIndicators(t *testing.T) {
	registry := NewRegistry()

	testFunc := func(ctx context.Context, indicator string, currDate time.Time, lookBackDays int, params map[string]string) (*models.MacroResult, error) {
		return nil, nil
	}

	registry.RegisterGetMacroIndicators("test_vendor", testFunc)

	vendors := registry.GetMacroIndicatorsVendors()
	assert.Contains(t, vendors, "test_vendor")

	fn, ok := registry.GetGetMacroIndicators("test_vendor")
	assert.True(t, ok)
	assert.NotNil(t, fn)
}

func TestUnit_RegisterGetPredictionMarkets(t *testing.T) {
	registry := NewRegistry()

	testFunc := func(ctx context.Context, topic string, limit int) (*models.PredictionResult, error) {
		return nil, nil
	}

	registry.RegisterGetPredictionMarkets("test_vendor", testFunc)

	vendors := registry.GetPredictionMarketsVendors()
	assert.Contains(t, vendors, "test_vendor")

	fn, ok := registry.GetGetPredictionMarkets("test_vendor")
	assert.True(t, ok)
	assert.NotNil(t, fn)
}

func TestUnit_GlobalRegistry(t *testing.T) {
	assert.NotNil(t, Registry)
	assert.NotNil(t, Registry.GetStockDataTools)
}
