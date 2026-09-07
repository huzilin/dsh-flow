package models

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestUnit_FormatFloat(t *testing.T) {
	tests := []struct {
		name  string
		input decimal.Decimal
		want  string
	}{
		{"zero", decimal.Zero, "0"},
		{"small positive", decimal.NewFromFloat(1.5), "1.5000"},
		{"large billion", decimal.NewFromFloat(1500000000), "1.50B"},
		{"large million", decimal.NewFromFloat(1500000), "1.50M"},
		{"large thousand", decimal.NewFromFloat(1500), "1.50K"},
		{"negative thousand", decimal.NewFromFloat(-1000), "-1.00K"},
		{"negative billion", decimal.NewFromFloat(-1500000000), "-1.50B"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatDecimal(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestUnit_FormatCandleTable(t *testing.T) {
	candles := []*Candle{
		{
			Date:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			Open:   decimal.NewFromFloat(100.0),
			High:   decimal.NewFromFloat(105.0),
			Low:    decimal.NewFromFloat(95.0),
			Close:  decimal.NewFromFloat(102.0),
			Volume: 1000000,
		},
	}

	result := FormatCandleTable(candles, "# Stock Data")
	assert.Contains(t, result, "| Date | Open | High | Low | Close | Volume |")
	assert.Contains(t, result, "2024-01-01")
	assert.Contains(t, result, "100.0000")
}

func TestUnit_FormatCandleTable_Empty(t *testing.T) {
	result := FormatCandleTable([]*Candle{}, "# Stock Data")
	assert.Contains(t, result, "无数据")
}

func TestUnit_FormatIndicatorTable(t *testing.T) {
	result := &IndicatorResult{
		Symbol:    "AAPL",
		Indicator: "RSI",
		Values: []*IndicatorValue{
			{Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Value: decimal.NewFromFloat(55.5)},
		},
	}

	output := FormatIndicatorTable(result, "# RSI")
	assert.Contains(t, output, "| Date | Value |")
	assert.Contains(t, output, "2024-01-01")
	assert.Contains(t, output, "55.5000")
}

func TestUnit_FormatIndicatorTable_Empty(t *testing.T) {
	result := &IndicatorResult{Symbol: "AAPL", Indicator: "RSI"}
	output := FormatIndicatorTable(result, "# RSI")
	assert.Contains(t, output, "无数据")
}

func TestUnit_FormatFundamentals(t *testing.T) {
	f := &Fundamentals{
		Symbol: "AAPL",
		CompanyInfo: &CompanyInfo{
			Name:        "Apple Inc.",
			Industry:    "Technology",
			MarketCap:   decimal.NewFromFloat(2000000000000),
			Description: "A technology company",
		},
		BusinessIntro: &BusinessIntro{
			MainBusiness: "Consumer electronics",
		},
		Composition: []*BusinessComposition{
			{Category: "iPhone", Revenue: decimal.NewFromFloat(150000000000), Proportion: decimal.NewFromFloat(0.5)},
		},
		Timestamp: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	output := FormatFundamentals(f)
	assert.Contains(t, output, "Apple Inc.")
	assert.Contains(t, output, "Technology")
	assert.Contains(t, output, "Consumer electronics")
	assert.Contains(t, output, "iPhone")
}

func TestUnit_FormatBalanceSheet(t *testing.T) {
	b := &BalanceSheet{
		Symbol:    "AAPL",
		Frequency: "quarterly",
		Items: []*BalanceSheetItem{
			{
				Date:             time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				TotalAssets:      decimal.NewFromFloat(350000000000),
				TotalLiabilities: decimal.NewFromFloat(200000000000),
				TotalEquity:      decimal.NewFromFloat(150000000000),
			},
		},
	}

	output := FormatBalanceSheet(b)
	assert.Contains(t, output, "AAPL")
	assert.Contains(t, output, "quarterly")
	assert.Contains(t, output, "350.00B")
}

func TestUnit_FormatCashflow(t *testing.T) {
	c := &Cashflow{
		Symbol:    "AAPL",
		Frequency: "annual",
		Items: []*CashflowItem{
			{
				Date:              time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				OperatingCashflow: decimal.NewFromFloat(100000000000),
				FreeCashflow:      decimal.NewFromFloat(80000000000),
			},
		},
	}

	output := FormatCashflow(c)
	assert.Contains(t, output, "AAPL")
	assert.Contains(t, output, "annual")
	assert.Contains(t, output, "100.00B")
}

func TestUnit_FormatIncomeStatement(t *testing.T) {
	i := &IncomeStatement{
		Symbol:    "AAPL",
		Frequency: "quarterly",
		Items: []*IncomeStatementItem{
			{
				Date:      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				Revenue:   decimal.NewFromFloat(90000000000),
				NetIncome: decimal.NewFromFloat(25000000000),
				EPS:       decimal.NewFromFloat(1.50),
			},
		},
	}

	output := FormatIncomeStatement(i)
	assert.Contains(t, output, "AAPL")
	assert.Contains(t, output, "90.00B")
	assert.Contains(t, output, "1.5000")
}

func TestUnit_FormatNewsTable(t *testing.T) {
	news := []*NewsItem{
		{
			Title:       "Apple releases new iPhone",
			Source:      "Reuters",
			PublishDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	output := FormatNewsTable(news, "# News", 10)
	assert.Contains(t, output, "Apple releases new iPhone")
	assert.Contains(t, output, "Reuters")
}

func TestUnit_FormatNewsTable_Empty(t *testing.T) {
	output := FormatNewsTable([]*NewsItem{}, "# News", 10)
	assert.Contains(t, output, "无新闻数据")
}

func TestUnit_FormatMacroTable(t *testing.T) {
	result := &MacroResult{
		Indicator: "GDP",
		Values: []*MacroIndicator{
			{Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Value: decimal.NewFromFloat(2500000000000)},
		},
		Unit: "USD",
	}

	output := FormatMacroTable(result, "# GDP")
	assert.Contains(t, output, "GDP")
	assert.Contains(t, output, "2500.00B")
	assert.Contains(t, output, "Unit: USD")
}

func TestUnit_FormatPredictionTable(t *testing.T) {
	result := &PredictionResult{
		Topic: "Election",
		Items: []*PredictionItem{
			{
				Question: "Will Biden win?",
				YesPrice: decimal.NewFromFloat(0.6),
				NoPrice:  decimal.NewFromFloat(0.4),
				EndDate:  time.Date(2024, 11, 5, 0, 0, 0, 0, time.UTC),
			},
		},
	}

	output := FormatPredictionTable(result, "# Election")
	assert.Contains(t, output, "Will Biden win?")
	assert.Contains(t, output, "60.0000%")
	assert.Contains(t, output, "40.0000%")
}

func TestUnit_FormatTable(t *testing.T) {
	rows := []map[string]any{
		{"Name": "Apple", "Price": decimal.NewFromFloat(100.0), "Volume": int64(1000)},
	}

	output := FormatTable(rows, "# Table", 10)
	assert.Contains(t, output, "| Name | Price | Volume |")
	assert.Contains(t, output, "Apple")
	assert.Contains(t, output, "100.0000")
}

func TestUnit_FormatTable_Empty(t *testing.T) {
	output := FormatTable([]map[string]any{}, "# Table", 10)
	assert.Contains(t, output, "无数据")
}
