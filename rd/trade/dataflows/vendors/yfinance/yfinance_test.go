package yfinance

import (
	"context"
	"testing"

	"github.com/huzilin/trade/dataflows/models"
	"github.com/huzilin/trade/logger"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	logger.InitZap("", logger.ModeDev)
	m.Run()
}

func TestUnit_GetStockDataInvalidSymbol(t *testing.T) {
	_, err := GetStockData(context.Background(), "INVALID_INVALID", "2024-01-01", "2024-01-10")
	assert.Error(t, err)
}

func TestUnit_GetStockDataInvalidDate(t *testing.T) {
	_, err := GetStockData(context.Background(), "AAPL", "invalid-date", "2024-01-10")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid start_date format")
}

func TestIntegration_GetStockData_US(t *testing.T) {
	res, err := GetStockData(context.Background(), "AAPL", "2025-01-02", "2025-01-10")
	if err != nil {
		t.Skipf("Skip due to API error: %v", err)
	}
	assert.NotEmpty(t, res)
	assert.IsType(t, []*models.Candle{}, res)
	for _, candle := range res {
		assert.NotNil(t, candle.Date)
	}
}

func TestIntegration_GetStockData_HK(t *testing.T) {
	res, err := GetStockData(context.Background(), "00700.HK", "2025-01-02", "2025-01-10")
	if err != nil {
		t.Skipf("Skip due to API error: %v", err)
	}
	assert.NotEmpty(t, res)
	assert.IsType(t, []*models.Candle{}, res)
	for _, candle := range res {
		assert.NotNil(t, candle.Date)
	}
}

func TestIntegration_GetStockData_HK_PlainSymbol(t *testing.T) {
	res, err := GetStockData(context.Background(), "00700", "2025-01-02", "2025-01-10")
	if err != nil {
		t.Skipf("Skip due to API error: %v", err)
	}
	assert.NotEmpty(t, res)
	assert.IsType(t, []*models.Candle{}, res)
}

func TestUnit_GetStockIndicatorsInvalidSymbol(t *testing.T) {
	_, err := GetStockIndicators(context.Background(), "INVALID_INVALID", "rsi", "2025-01-10", 30)
	assert.Error(t, err)
}

func TestUnit_GetStockIndicatorsInvalidIndicator(t *testing.T) {
	_, err := GetStockIndicators(context.Background(), "AAPL", "invalid_indicator", "2025-01-10", 30)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported indicator")
}

func TestIntegration_GetStockIndicators_RSI_US(t *testing.T) {
	res, err := GetStockIndicators(context.Background(), "AAPL", "rsi", "2025-01-10", 30)
	if err != nil {
		t.Skipf("Skip due to API error: %v", err)
	}
	assert.NotNil(t, res)
	assert.Equal(t, "rsi", res.Indicator)
	assert.NotEmpty(t, res.Values)
	for _, v := range res.Values {
		assert.NotNil(t, v.Date)
	}
}

func TestIntegration_GetStockIndicators_MACD_US(t *testing.T) {
	res, err := GetStockIndicators(context.Background(), "AAPL", "macd", "2025-01-10", 30)
	if err != nil {
		t.Skipf("Skip due to API error: %v", err)
	}
	assert.NotNil(t, res)
	assert.Equal(t, "macd", res.Indicator)
	assert.NotEmpty(t, res.Values)
}

func TestIntegration_GetStockIndicators_RSI_HK(t *testing.T) {
	res, err := GetStockIndicators(context.Background(), "00700.HK", "rsi", "2025-01-10", 30)
	if err != nil {
		t.Skipf("Skip due to API error: %v", err)
	}
	assert.NotNil(t, res)
	assert.Equal(t, "rsi", res.Indicator)
	assert.NotEmpty(t, res.Values)
}
