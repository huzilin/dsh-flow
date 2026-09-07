package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type MacroIndicator struct {
	Date  time.Time
	Value decimal.Decimal
}

type MacroResult struct {
	Indicator string
	Values    []*MacroIndicator
	Source    string
	Unit      string
}

func FormatMacroTable(result *MacroResult, title string) string {
	output := title + "\n\n"
	if result.Unit != "" {
		output += "Unit: " + result.Unit + "\n\n"
	}

	output += "| Date | Value |\n"
	output += "| --- | --- |\n"

	for _, v := range result.Values {
		if v != nil {
			output += "| " + v.Date.Format("2006-01-02") + " | " + formatDecimal(v.Value) + " |\n"
		}
	}

	return output
}
