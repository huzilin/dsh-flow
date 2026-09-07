package models

import (
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

type ColumnDefinition struct {
	EnglishName string
	ChineseName string
}

var StockDataColumns = []ColumnDefinition{
	{"Name", "名称"},
	{"Code", "代码"},
	{"Date", "日期"},
	{"Open", "开盘价"},
	{"High", "最高价"},
	{"Low", "最低价"},
	{"Close", "收盘价"},
	{"Volume", "成交量"},
	{"Amount", "成交额"},
	{"PctChange", "涨跌幅"},
	{"TurnoverPct", "换手率"},
}

func GetStockDataHeader(includeChinese bool) string {
	var headers []string
	for _, col := range StockDataColumns {
		if includeChinese {
			headers = append(headers, col.EnglishName+"("+col.ChineseName+")")
		} else {
			headers = append(headers, col.EnglishName)
		}
	}
	return strings.Join(headers, ",")
}

type Candle struct {
	Name        string
	Code        string
	Date        time.Time
	Open        decimal.Decimal
	High        decimal.Decimal
	Low         decimal.Decimal
	Close       decimal.Decimal
	Volume      int64
	Amount      *decimal.Decimal // 成交额
	PctChange   *decimal.Decimal // 涨跌幅
	TurnoverPct *decimal.Decimal // 换手率
	AdjClose    *decimal.Decimal // 调整后收盘价
}

func FormatCandleTable(candles []*Candle, title string) string {
	if len(candles) == 0 {
		return title + "\n\n无数据"
	}

	result := title + "\n\n"
	result += "| Name | Code | Date | Open | High | Low | Close | Volume | Amount | PctChange | TurnoverPct |\n"
	result += "| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |\n"

	for _, c := range candles {
		if c != nil {
			result += formatCandleRow(c)
		}
	}
	return result
}

func formatCandleRow(c *Candle) string {
	amount := "-"
	if c.Amount != nil {
		amount = formatDecimal(*c.Amount)
	}
	pctChange := "-"
	if c.PctChange != nil {
		pctChange = formatDecimal(*c.PctChange)
	}
	turnoverPct := "-"
	if c.TurnoverPct != nil {
		turnoverPct = formatDecimal(*c.TurnoverPct)
	}

	return c.Name + " | " +
		c.Code + " | " +
		c.Date.Format("2006-01-02") + " | " +
		formatDecimal(c.Open) + " | " +
		formatDecimal(c.High) + " | " +
		formatDecimal(c.Low) + " | " +
		formatDecimal(c.Close) + " | " +
		formatInt(c.Volume) + " | " +
		amount + " | " +
		pctChange + " | " +
		turnoverPct + "\n"
}
