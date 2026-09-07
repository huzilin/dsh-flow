package eastmoney_mx

import (
	"context"
	"testing"

	"github.com/huzilin/trade/logger"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	logger.InitZap("", logger.ModeDev)
	m.Run()
}

func TestUnit_MXAPIKey(t *testing.T) {
	key, err := MXAPIKey()
	assert.NoError(t, err)
	assert.Equal(t, MXAPIKeyValue, key)
}

func TestUnit_EastmoneyMXGetFundamentalsParams(t *testing.T) {
	_, err := EastmoneyMXGetFundamentals(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requires at least 1 argument")
}

func TestUnit_EastmoneyMXGetFundamentalsInvalidSymbol(t *testing.T) {
	_, err := EastmoneyMXGetFundamentals(context.Background(), "INVALID")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid A-share symbol")
}

func TestUnit_EastmoneyMXGetNewsParams(t *testing.T) {
	_, err := EastmoneyMXGetNews(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requires 3 arguments")

	_, err = EastmoneyMXGetNews(context.Background(), "000001", "2024-01-01")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requires 3 arguments")
}

func TestUnit_EastmoneyMXGetNewsInvalidSymbol(t *testing.T) {
	_, err := EastmoneyMXGetNews(context.Background(), "INVALID", "2024-01-01", "2024-01-10")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid A-share symbol")
}

func TestUnit_EastmoneyMXGetCompanyAnnouncementsParams(t *testing.T) {
	_, err := EastmoneyMXGetCompanyAnnouncements(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requires 3 arguments")

	_, err = EastmoneyMXGetCompanyAnnouncements(context.Background(), "000001", "2024-01-01")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requires 3 arguments")
}

func TestUnit_EastmoneyMXGetCompanyAnnouncementsInvalidSymbol(t *testing.T) {
	_, err := EastmoneyMXGetCompanyAnnouncements(context.Background(), "INVALID", "2024-01-01", "2024-01-10")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid A-share symbol")
}
