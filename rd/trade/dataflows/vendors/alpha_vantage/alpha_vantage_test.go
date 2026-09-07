package alpha_vantage

import (
	"context"
	"os"
	"testing"

	"github.com/huzilin/trade/dataflows/models"
	"github.com/huzilin/trade/logger"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	logger.InitZap("", logger.ModeDev)
	m.Run()
}

func TestUnit_AlphaVantageStockDataInvalidSymbol(t *testing.T) {
	_, err := AlphaVantageStockData(context.Background(), "INVALID", "2025-01-01", "2025-01-10")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ALPHA_VANTAGE_API_KEY")
}

func TestIntegration_AlphaVantageStockData_Normal(t *testing.T) {
	if os.Getenv("ALPHA_VANTAGE_API_KEY") == "" {
		t.Skip("ALPHA_VANTAGE_API_KEY not set, skipping integration test")
	}

	res, err := AlphaVantageStockData(context.Background(), "AAPL", "2025-01-02", "2025-01-10")
	if err != nil {
		t.Skipf("Skip due to API error: %v", err)
	}
	assert.NotEmpty(t, res)
	assert.IsType(t, []*models.Candle{}, res)
	for _, candle := range res {
		assert.NotNil(t, candle.Date)
	}
}

func TestIntegration_AlphaVantageStockData_DailyAdjusted(t *testing.T) {
	if os.Getenv("ALPHA_VANTAGE_API_KEY") == "" {
		t.Skip("ALPHA_VANTAGE_API_KEY not set, skipping integration test")
	}

	res, err := AlphaVantageStockData(context.Background(), "MSFT", "2025-01-01", "2025-01-15")
	if err != nil {
		t.Skipf("Skip due to API error: %v", err)
	}
	assert.NotEmpty(t, res)
}
