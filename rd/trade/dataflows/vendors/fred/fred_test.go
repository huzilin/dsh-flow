package fred

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

func TestIntegration_GetMacroData_Normal(t *testing.T) {
	date := time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC)
	res, err := GetMacroData(context.Background(), "GDP", date, 365, nil)
	if err != nil {
		t.Skipf("Skip due to API error: %v", err)
	}
	assert.NotNil(t, res)
	assert.IsType(t, &models.MacroResult{}, res)
	assert.Equal(t, "fred", res.Source)
}

func TestUnit_GetMacroData_EmptyIndicator(t *testing.T) {
	date := time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC)
	_, err := GetMacroData(context.Background(), "", date, 365, nil)
	assert.Error(t, err)
}

func TestUnit_GetMacroData_InvalidDate(t *testing.T) {
	_, err := GetMacroData(context.Background(), "GDP", time.Time{}, 365, nil)
	assert.Error(t, err)
}

func TestIntegration_GetMacroData_WithParams(t *testing.T) {
	date := time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC)
	res, err := GetMacroData(context.Background(), "CPIAUCSL", date, 30, map[string]string{"frequency": "m"})
	if err != nil {
		t.Skipf("Skip due to API error: %v", err)
	}
	assert.NotNil(t, res)
	assert.Equal(t, "fred", res.Source)
}
