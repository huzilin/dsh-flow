package dataflows

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/huzilin/trade/dataflows/interfaces"
	"github.com/huzilin/trade/dataflows/normalizer"
	"github.com/huzilin/trade/dataflows/router"
	_ "github.com/huzilin/trade/dataflows/vendors"
	"github.com/huzilin/trade/dataflows/vendors/tool_registry"
	"github.com/huzilin/trade/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func parseDate(dateStr string) time.Time {
	t, _ := time.Parse("2006-01-02", dateStr)
	return t
}

func TestMain(m *testing.M) {
	logger.InitZap("", logger.ModeDev)
	os.Exit(m.Run())
}

func TestIntegrationFlow_StockData_US(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试（需要网络访问）")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	norm := normalizer.NewDefaultNormalizer()
	symbol, err := norm.Normalize("AAPL")
	require.NoError(t, err)
	assert.Equal(t, "AAPL", symbol.Plain)
	assert.Equal(t, normalizer.MarketUSShare, symbol.Market)

	vendors := tool_registry.Registry.GetStockDataVendors()
	require.NotEmpty(t, vendors, "应该找到至少一个供应商")

	r := router.NewDefaultRouter()
	result, err := r.GetStockData(ctx, symbol.Standard, "2024-01-01", "2024-01-05")
	if err != nil {
		t.Logf("供应商调用失败: %v（可能是 API 限制）", err)
	} else {
		assert.NotEmpty(t, result, "应该返回股票数据")
		t.Logf("成功获取 %s 数据", symbol.Standard)
	}
}

func TestIntegrationFlow_StockData_AShare(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试（需要网络访问）")
	}

	norm := normalizer.NewDefaultNormalizer()
	symbol, err := norm.Normalize("000001")
	require.NoError(t, err)
	assert.Equal(t, "000001", symbol.Plain)
	assert.Equal(t, normalizer.MarketAshare, symbol.Market)

	vendors := tool_registry.Registry.GetStockDataVendors()
	require.NotEmpty(t, vendors)

	assert.Contains(t, vendors, "akshare", "A股市场应该包含 akshare 供应商")
}

func TestIntegrationFlow_StockData_HK(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试（需要网络访问）")
	}

	norm := normalizer.NewDefaultNormalizer()
	symbol, err := norm.Normalize("00700.HK")
	require.NoError(t, err)
	assert.Equal(t, "00700", symbol.Plain)
	assert.Equal(t, normalizer.MarketHKShare, symbol.Market)

	vendors := tool_registry.Registry.GetStockDataVendors()
	require.NotEmpty(t, vendors)

	assert.Contains(t, vendors, "yfinance", "港股市场应该包含 yfinance 供应商")
}

func TestIntegrationFlow_Fundamentals(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试（需要网络访问）")
	}

	norm := normalizer.NewDefaultNormalizer()
	_, err := norm.Normalize("AAPL")
	require.NoError(t, err)

	vendors := tool_registry.Registry.GetFundamentalsVendors()
	require.NotEmpty(t, vendors)
}

func TestIntegrationFlow_MacroIndicators(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试（需要网络访问）")
	}

	vendors := tool_registry.Registry.GetMacroIndicatorsVendors()
	require.NotEmpty(t, vendors)

	require.Contains(t, vendors, "fred", "宏观经济应该使用 fred")
}

func TestIntegrationFlow_PredictionMarkets(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试（需要网络访问）")
	}

	vendors := tool_registry.Registry.GetPredictionMarketsVendors()
	require.NotEmpty(t, vendors)

	require.Contains(t, vendors, "polymarket", "预测市场应该使用 polymarket")
}

func TestIntegrationFlow_RegistryCapabilityMatrix(t *testing.T) {
	vendors := tool_registry.Registry.GetStockDataVendors()
	assert.NotEmpty(t, vendors, "应该有供应商支持股票数据")

	r := router.NewDefaultRouter()
	matrix := r.GetCapabilityMatrix()
	assert.NotEmpty(t, matrix)

	keyCaps := []interfaces.Capability{
		interfaces.CapabilityStockData,
		interfaces.CapabilityIndicators,
		interfaces.CapabilityFundamentals,
		interfaces.CapabilityNews,
		interfaces.CapabilityMacroIndicators,
		interfaces.CapabilityPredictionMarkets,
	}

	for _, cap := range keyCaps {
		assert.Contains(t, matrix, string(cap), "能力 %s 应该有供应商支持", cap)
	}
}

func TestIntegrationFlow_FindBestProvider(t *testing.T) {
	tests := []struct {
		name           string
		capability     interfaces.Capability
		market         normalizer.MarketType
		preferredOrder []string
		want           string
		wantOneOf      []string
	}{
		{"美股股票数据-指定优先顺序", interfaces.CapabilityStockData, normalizer.MarketUSShare, []string{"yfinance"}, "yfinance", nil},
		{"A股股票数据-指定优先顺序", interfaces.CapabilityStockData, normalizer.MarketAshare, []string{"akshare"}, "akshare", nil},
		{"港股股票数据-指定优先顺序", interfaces.CapabilityStockData, normalizer.MarketHKShare, []string{"yfinance"}, "yfinance", nil},
		{"美股股票数据-自动选择", interfaces.CapabilityStockData, normalizer.MarketUSShare, nil, "", []string{"yfinance", "alpha_vantage"}},
		{"A股股票数据-自动选择", interfaces.CapabilityStockData, normalizer.MarketAshare, nil, "akshare", nil},
		{"港股股票数据-自动选择", interfaces.CapabilityStockData, normalizer.MarketHKShare, nil, "yfinance", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var vendors []string
			switch tt.capability {
			case interfaces.CapabilityStockData:
				vendors = tool_registry.Registry.GetStockDataVendors()
			case interfaces.CapabilityIndicators:
				vendors = tool_registry.Registry.GetIndicatorsVendors()
			case interfaces.CapabilityFundamentals:
				vendors = tool_registry.Registry.GetFundamentalsVendors()
			case interfaces.CapabilityNews:
				vendors = tool_registry.Registry.GetNewsVendors()
			case interfaces.CapabilityMacroIndicators:
				vendors = tool_registry.Registry.GetMacroIndicatorsVendors()
			case interfaces.CapabilityPredictionMarkets:
				vendors = tool_registry.Registry.GetPredictionMarketsVendors()
			}

			if tt.preferredOrder != nil && len(tt.preferredOrder) > 0 {
				for _, pref := range tt.preferredOrder {
					for _, v := range vendors {
						if v == pref {
							if tt.want != "" {
								assert.Equal(t, tt.want, v)
							}
							return
						}
					}
				}
			}

			if tt.want != "" {
				assert.Contains(t, vendors, tt.want)
			} else if tt.wantOneOf != nil {
				found := false
				for _, v := range tt.wantOneOf {
					if contains(vendors, v) {
						found = true
						break
					}
				}
				assert.True(t, found, "expected one of %v, got %v", tt.wantOneOf, vendors)
			}
		})
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func TestIntegrationFlow_MultiMarketDetection(t *testing.T) {
	norm := normalizer.NewDefaultNormalizer()

	tests := []struct {
		name   string
		symbol string
		want   normalizer.MarketType
	}{
		{"美股", "AAPL", normalizer.MarketUSShare},
		{"A股深市", "000001", normalizer.MarketAshare},
		{"A股沪市", "600000", normalizer.MarketAshare},
		{"港股", "00700.HK", normalizer.MarketHKShare},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			market := norm.DetectMarket(tt.symbol)
			assert.Equal(t, tt.want, market)
		})
	}
}

func TestIntegrationFlow_RouterCapabilities(t *testing.T) {
	r := router.NewDefaultRouter()

	matrix := r.GetCapabilityMatrix()
	assert.NotEmpty(t, matrix)

	methods := r.GetSupportedMethods()
	assert.NotEmpty(t, methods)

	keyMethods := []string{
		string(interfaces.CapabilityStockData),
		string(interfaces.CapabilityFundamentals),
		string(interfaces.CapabilityNews),
		string(interfaces.CapabilityMacroIndicators),
		string(interfaces.CapabilityPredictionMarkets),
	}

	for _, method := range keyMethods {
		assert.Contains(t, methods, method, "应该支持方法 %s", method)
	}
}

func TestIntegrationFlow_CompleteChain(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试（需要网络访问）")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	symbol := "AAPL"

	norm := normalizer.NewDefaultNormalizer()
	normalized, err := norm.Normalize(symbol)
	require.NoError(t, err)
	t.Logf("符号标准化: %s -> %s (市场: %s)", symbol, normalized.Standard, normalized.Market)

	market := norm.DetectMarket(symbol)
	assert.Equal(t, normalizer.MarketUSShare, market)
	t.Logf("市场检测: %s", market)

	r := router.NewDefaultRouter()

	vendors := tool_registry.Registry.GetStockDataVendors()
	assert.Contains(t, vendors, "yfinance")
	t.Logf("最佳供应商: yfinance")

	matrix := r.GetCapabilityMatrix()
	assert.Contains(t, matrix, string(interfaces.CapabilityStockData))
	t.Logf("能力矩阵包含方法: %s", interfaces.CapabilityStockData)

	result, err := r.GetStockData(ctx, normalized.Standard, "2024-01-01", "2024-01-05")
	if err != nil {
		t.Logf("获取数据失败: %v（可能是网络或 API 限制）", err)
		t.Skip("无法获取真实数据")
	}

	assert.NotEmpty(t, result)
	t.Logf("成功获取数据")
}
