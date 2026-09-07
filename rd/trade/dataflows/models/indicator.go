package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type IndicatorValue struct {
	Date  time.Time
	Value decimal.Decimal
}

type IndicatorResult struct {
	Symbol    string
	Indicator string
	Values    []*IndicatorValue
	Params    map[string]decimal.Decimal
	Source    string
}

func FormatIndicatorTable(result *IndicatorResult, title string) string {
	if len(result.Values) == 0 {
		return title + "\n\n无数据"
	}

	output := title + "\n\n"
	output += "| Date | Value |\n"
	output += "| --- | --- |\n"

	for _, v := range result.Values {
		if v != nil {
			output += v.Date.Format("2006-01-02") + " | " + formatDecimal(v.Value) + "\n"
		}
	}

	return output
}
