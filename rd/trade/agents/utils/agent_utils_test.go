package utils

import (
	"testing"

	"github.com/huzilin/trade/agents/states"
	"github.com/stretchr/testify/assert"
)

func TestUnit_GetMarketDescriptor(t *testing.T) {
	tests := []struct {
		name     string
		ticker   string
		expected string
	}{
		{"A-share SH", "600519", "A股"},
		{"A-share SZ", "000001", "A股"},
		{"A-share BJ", "830883", "A股"},
		{"HK stock", "00700.HK", "港股"},
		{"HK stock normalized", "00700.HK", "港股"},
		{"Unknown ticker", "UNKNOWN", "A股"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetMarketDescriptor(tt.ticker)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUnit_GetMarketIndefiniteDescriptor(t *testing.T) {
	tests := []struct {
		name     string
		ticker   string
		expected string
	}{
		{"A-share", "600519", "一只A股"},
		{"HK stock", "00700.HK", "一只港股"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetMarketIndefiniteDescriptor(tt.ticker)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUnit_GetMarketScope(t *testing.T) {
	tests := []struct {
		name     string
		ticker   string
		expected string
	}{
		{"A-share", "600519", "A股市场"},
		{"HK stock", "00700.HK", "港股市场"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetMarketScope(tt.ticker)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUnit_GetExecutionConstraintsPrompt(t *testing.T) {
	tests := []struct {
		name     string
		ticker   string
		contains string
	}{
		{"A-share constraints", "600519", "T+1"},
		{"HK stock constraints", "00700.HK", "南向资金"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetExecutionConstraintsPrompt(tt.ticker)
			assert.Contains(t, result, tt.contains)
		})
	}
}

func TestUnit_GetMarketNewsFocusPrompt(t *testing.T) {
	tests := []struct {
		name     string
		ticker   string
		contains string
	}{
		{"A-share news focus", "600519", "政策催化"},
		{"HK stock news focus", "00700.HK", "港交所"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetMarketNewsFocusPrompt(tt.ticker)
			assert.Contains(t, result, tt.contains)
		})
	}
}

func TestUnit_GetFundamentalFocusPrompt(t *testing.T) {
	tests := []struct {
		name     string
		ticker   string
		contains string
	}{
		{"A-share fundamentals", "600519", "归母净利润"},
		{"HK stock fundamentals", "00700.HK", "ROE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetFundamentalFocusPrompt(tt.ticker)
			assert.Contains(t, result, tt.contains)
		})
	}
}

func TestUnit_GetMarketPolicyReportLabel(t *testing.T) {
	tests := []struct {
		name     string
		ticker   string
		expected string
	}{
		{"A-share label", "600519", "最新A股市场与政策新闻"},
		{"HK stock label", "00700.HK", "最新港股市场与政策报告"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetMarketPolicyReportLabel(tt.ticker)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUnit_BuildInstrumentContext(t *testing.T) {
	tests := []struct {
		name      string
		ticker    string
		assetType string
		contains  string
	}{
		{"Crypto asset", "BTC", "crypto", "加密资产"},
		{"HK stock", "00700.HK", "stock", "港股代码"},
		{"A-share", "600519", "stock", "A-share代码"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildInstrumentContext(tt.ticker, tt.assetType)
			assert.Contains(t, result, tt.contains)
		})
	}
}

func TestUnit_GetInstrumentContextFromState(t *testing.T) {
	t.Run("Use existing context", func(t *testing.T) {
		state := &states.AgentState{
			InstrumentContext: "自定义上下文",
			CompanyOfInterest: "600519",
			AssetType:         "stock",
		}
		result := GetInstrumentContextFromState(state)
		assert.Equal(t, "自定义上下文", result)
	})

	t.Run("Build new context", func(t *testing.T) {
		state := &states.AgentState{
			InstrumentContext: "",
			CompanyOfInterest: "600519",
			AssetType:         "stock",
		}
		result := GetInstrumentContextFromState(state)
		assert.Contains(t, result, "600519")
	})
}

func TestUnit_LanguageInstructions(t *testing.T) {
	t.Run("GetInternalLanguageInstruction not empty", func(t *testing.T) {
		result := GetInternalLanguageInstruction()
		assert.NotEmpty(t, result)
		assert.Contains(t, result, "中文")
	})

	t.Run("GetFinalLanguageInstruction not empty", func(t *testing.T) {
		result := GetFinalLanguageInstruction()
		assert.NotEmpty(t, result)
		assert.Contains(t, result, "中文")
	})

	t.Run("GetUserFacingReportInstruction not empty", func(t *testing.T) {
		result := GetUserFacingReportInstruction()
		assert.NotEmpty(t, result)
		assert.Contains(t, result, "Markdown")
	})
}
