package akshare

import (
	"context"
	"testing"
	"time"

	"github.com/huzilin/trade/dataflows/models"
	"github.com/huzilin/trade/logger"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	logger.InitZap("", logger.ModeDev)
	m.Run()
}

func TestUnit_AkShareGetStockDataInvalidSymbol(t *testing.T) {
	_, err := AkShareGetStockData(context.Background(), "INVALID", "2024-01-01", "2024-01-10")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid A-share symbol")
}

func TestIntegration_AkShareGetStockData_Normal(t *testing.T) {
	res, err := AkShareGetStockData(context.Background(), "000001.SZ", "2025-01-02", "2025-01-10")
	if err != nil {
		t.Skipf("Skip due to API error: %v", err)
	}
	assert.NotEmpty(t, res)
	assert.IsType(t, []*models.Candle{}, res)
}

func TestUnit_AkShareGetIndicatorsInvalidSymbol(t *testing.T) {
	_, err := AkShareGetIndicators(context.Background(), "INVALID", "rsi", "2025-01-10", 30)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid A-share symbol")
}

func TestUnit_AkShareGetIndicatorsInvalidIndicator(t *testing.T) {
	_, err := AkShareGetIndicators(context.Background(), "000001", "invalid_indicator", "2025-01-10", 30)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "is not supported")
}

func TestIntegration_AkShareGetIndicators_RSI(t *testing.T) {
	res, err := AkShareGetIndicators(context.Background(), "000001", "rsi", "2025-01-10", 30)
	if err != nil {
		t.Skipf("Skip due to API error: %v", err)
	}
	assert.NotNil(t, res)
	assert.Equal(t, "rsi", res.Indicator)
	assert.NotEmpty(t, res.Values)
	for _, v := range res.Values {
		assert.NotNil(t, v.Date)
		assert.False(t, v.Value.IsZero())
		logger.S().Info("res", v.Value)
	}
}

func TestIntegration_AkShareGetIndicators_MACD(t *testing.T) {
	res, err := AkShareGetIndicators(context.Background(), "000001", "macd", "2025-01-10", 30)
	if err != nil {
		t.Skipf("Skip due to API error: %v", err)
	}
	assert.NotNil(t, res)
	assert.Equal(t, "macd", res.Indicator)
	assert.NotEmpty(t, res.Values)
}

func TestUnit_AkShareGetFundamentalsInvalidSymbol(t *testing.T) {
	_, err := AkShareGetFundamentals(context.Background(), "INVALID", time.Now())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid A-share symbol")
}

func TestIntegration_AkShareGetFundamentals_Normal(t *testing.T) {
	res, err := AkShareGetFundamentals(context.Background(), "000001", time.Now())
	if err != nil {
		t.Skipf("Skip due to API error: %v", err)
	}
	assert.NotNil(t, res)
	assert.Equal(t, "akshare", res.Source)
	assert.NotEmpty(t, res.Symbol)
}

func TestUnit_AkShareGetBalanceSheetInvalidSymbol(t *testing.T) {
	_, err := AkShareGetBalanceSheet(context.Background(), "INVALID", "quarterly", time.Now())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid A-share symbol")
}

func TestIntegration_AkShareGetBalanceSheet_Normal(t *testing.T) {
	res, err := AkShareGetBalanceSheet(context.Background(), "000001", "quarterly", time.Now())
	if err != nil {
		t.Skipf("Skip due to API error: %v", err)
	}
	assert.NotNil(t, res)
	assert.Equal(t, "akshare", res.Source)
	assert.Equal(t, "quarterly", res.Frequency)
}

func TestUnit_AkShareGetCashflowInvalidSymbol(t *testing.T) {
	_, err := AkShareGetCashflow(context.Background(), "INVALID", "quarterly", time.Now())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid A-share symbol")
}

func TestIntegration_AkShareGetCashflow_Normal(t *testing.T) {
	res, err := AkShareGetCashflow(context.Background(), "000001", "quarterly", time.Now())
	if err != nil {
		t.Skipf("Skip due to API error: %v", err)
	}
	assert.NotNil(t, res)
	assert.Equal(t, "akshare", res.Source)
}

func TestUnit_AkShareGetIncomeStatementInvalidSymbol(t *testing.T) {
	_, err := AkShareGetIncomeStatement(context.Background(), "INVALID", "quarterly", time.Now())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid A-share symbol")
}

func TestIntegration_AkShareGetIncomeStatement_Normal(t *testing.T) {
	res, err := AkShareGetIncomeStatement(context.Background(), "000001", "quarterly", time.Now())
	if err != nil {
		t.Skipf("Skip due to API error: %v", err)
	}
	assert.NotNil(t, res)
	assert.Equal(t, "akshare", res.Source)
}

func TestUnit_AkShareGetNewsInvalidSymbol(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC)
	_, err := AkShareGetNews(context.Background(), "INVALID", start, end, 20)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid A-share symbol")
}

func TestIntegration_AkShareGetNews_Normal(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC)
	res, err := AkShareGetNews(context.Background(), "000001", start, end, 20)
	if err != nil {
		t.Skipf("Skip due to API error: %v", err)
	}
	assert.NotNil(t, res)
	assert.Equal(t, "akshare", res.Source)
}

func TestIntegration_AkShareGetMarketNews_Normal(t *testing.T) {
	currDate := time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC)
	res, err := AkShareGetMarketNews(context.Background(), currDate, 7, 10)
	if err != nil {
		t.Skipf("Skip due to API error: %v", err)
	}
	assert.NotNil(t, res)
	assert.Equal(t, "akshare", res.Source)
}

func TestUnit_AkShareGetCompanyAnnouncementsInvalidSymbol(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC)
	_, err := AkShareGetCompanyAnnouncements(context.Background(), "INVALID", start, end)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid A-share symbol")
}

func TestIntegration_AkShareGetCompanyAnnouncements_Normal(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC)
	res, err := AkShareGetCompanyAnnouncements(context.Background(), "000001", start, end)
	if err != nil {
		t.Skipf("Skip due to API error: %v", err)
	}
	assert.NotNil(t, res)
	assert.Equal(t, "akshare", res.Source)
}
