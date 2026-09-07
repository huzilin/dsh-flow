package polymarket

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

func TestIntegration_GetPredictionMarkets_Normal(t *testing.T) {
	res, err := GetPredictionMarkets(context.Background(), "politics", 10)
	if err != nil {
		t.Skipf("Skip due to API error: %v", err)
	}
	assert.NotNil(t, res)
	assert.IsType(t, &models.PredictionResult{}, res)
	assert.Equal(t, "polymarket", res.Source)
	assert.LessOrEqual(t, len(res.Items), 10)
}

func TestUnit_GetPredictionMarkets_EmptyTopic(t *testing.T) {
	_, err := GetPredictionMarkets(context.Background(), "", 10)
	assert.Error(t, err)
}

func TestUnit_GetPredictionMarkets_InvalidLimit(t *testing.T) {
	_, err := GetPredictionMarkets(context.Background(), "politics", -1)
	assert.Error(t, err)
}

func TestIntegration_GetPredictionMarkets_SmallLimit(t *testing.T) {
	res, err := GetPredictionMarkets(context.Background(), "politics", 1)
	if err != nil {
		t.Skipf("Skip due to API error: %v", err)
	}
	assert.NotNil(t, res)
	assert.LessOrEqual(t, len(res.Items), 1)
}
