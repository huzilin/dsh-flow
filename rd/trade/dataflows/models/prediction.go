package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type PredictionItem struct {
	Question    string
	YesPrice    decimal.Decimal // Yes 选项价格 (0-1)
	NoPrice     decimal.Decimal // No 选项价格 (0-1)
	Volume      decimal.Decimal
	EndDate     time.Time
	Description string
}

type PredictionResult struct {
	Topic  string
	Items  []*PredictionItem
	Source string
}

func FormatPredictionTable(result *PredictionResult, title string) string {
	output := title + "\n\n"
	output += "| Question | Yes | No | Volume | End Date |\n"
	output += "| --- | --- | --- | --- | --- |\n"

	for _, item := range result.Items {
		if item != nil {
			output += "| " + item.Question + " | " +
				formatDecimal(item.YesPrice.Mul(decimal.NewFromInt(100))) + "% | " +
				formatDecimal(item.NoPrice.Mul(decimal.NewFromInt(100))) + "% | " +
				formatDecimal(item.Volume) + " | " +
				item.EndDate.Format("2006-01-02") + " |\n"
		}
	}

	return output
}
