package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type CompanyInfo struct {
	Name        string
	Industry    string
	Sector      string
	MarketCap   decimal.Decimal
	Employees   int
	Description string
	Website     string
}

type BusinessIntro struct {
	MainBusiness string
	Scope        string
}

type BusinessComposition struct {
	Category   string // 行业/产品/地区
	Revenue    decimal.Decimal
	Proportion decimal.Decimal // 占比百分比
}

type FinancialSummary struct {
	Revenue       decimal.Decimal
	NetIncome     decimal.Decimal
	EPS           decimal.Decimal
	PERatio       decimal.Decimal
	DividendYield decimal.Decimal
}

type Fundamentals struct {
	Symbol        string
	CompanyInfo   *CompanyInfo
	BusinessIntro *BusinessIntro
	Composition   []*BusinessComposition
	Financials    *FinancialSummary
	Source        string
	Timestamp     time.Time
}

type BalanceSheetItem struct {
	Date               time.Time
	TotalAssets        decimal.Decimal
	TotalLiabilities   decimal.Decimal
	TotalEquity        decimal.Decimal
	Cash               decimal.Decimal
	CurrentAssets      decimal.Decimal
	CurrentLiabilities decimal.Decimal
}

type BalanceSheet struct {
	Symbol    string
	Frequency string // quarterly, annual
	Items     []*BalanceSheetItem
	Source    string
}

type CashflowItem struct {
	Date               time.Time
	OperatingCashflow  decimal.Decimal
	InvestingCashflow  decimal.Decimal
	FinancingCashflow  decimal.Decimal
	FreeCashflow       decimal.Decimal
	CapitalExpenditure decimal.Decimal
	DividendsPaid      decimal.Decimal
}

type Cashflow struct {
	Symbol    string
	Frequency string
	Items     []*CashflowItem
	Source    string
}

type IncomeStatementItem struct {
	Date            time.Time
	Revenue         decimal.Decimal
	GrossProfit     decimal.Decimal
	OperatingIncome decimal.Decimal
	NetIncome       decimal.Decimal
	EBITDA          decimal.Decimal
	EPS             decimal.Decimal
}

type IncomeStatement struct {
	Symbol    string
	Frequency string
	Items     []*IncomeStatementItem
	Source    string
}

// TODO: 移动到调用侧
func FormatFundamentals(f *Fundamentals) string {
	output := "# Fundamentals for " + f.Symbol + "\n\n"

	if f.CompanyInfo != nil {
		output += "## Company Info\n\n"
		output += "- Name: " + f.CompanyInfo.Name + "\n"
		output += "- Industry: " + f.CompanyInfo.Industry + "\n"
		output += "- Sector: " + f.CompanyInfo.Sector + "\n"
		if f.CompanyInfo.MarketCap.GreaterThan(decimal.Zero) {
			output += "- Market Cap: " + formatDecimal(f.CompanyInfo.MarketCap) + "\n"
		}
		if f.CompanyInfo.Description != "" {
			output += "- Description: " + f.CompanyInfo.Description + "\n"
		}
	}

	if f.BusinessIntro != nil && f.BusinessIntro.MainBusiness != "" {
		output += "\n## Main Business\n\n"
		output += f.BusinessIntro.MainBusiness + "\n"
	}

	if len(f.Composition) > 0 {
		output += "\n## Business Composition\n\n"
		output += "| Category | Revenue | Proportion |\n"
		output += "| --- | --- | --- |\n"
		for _, c := range f.Composition {
			if c != nil {
				output += "| " + c.Category + " | " + formatDecimal(c.Revenue) + " | " + formatDecimal(c.Proportion.Mul(decimal.NewFromInt(100))) + "% |\n"
			}
		}
	}

	output += "\n## Source\n\n"
	output += "- Vendor: " + f.Source + "\n"
	output += "- Timestamp: " + f.Timestamp.Format("2006-01-02 15:04:05") + "\n"

	return output
}

func FormatBalanceSheet(b *BalanceSheet) string {
	output := "# Balance Sheet for " + b.Symbol + " (" + b.Frequency + ")\n\n"

	output += "| Date | Total Assets | Total Liabilities | Total Equity | Cash | Current Assets | Current Liabilities |\n"
	output += "| --- | --- | --- | --- | --- | --- | --- |\n"

	for _, item := range b.Items {
		if item != nil {
			output += "| " + item.Date.Format("2006-01-02") + " | " +
				formatDecimal(item.TotalAssets) + " | " +
				formatDecimal(item.TotalLiabilities) + " | " +
				formatDecimal(item.TotalEquity) + " | " +
				formatDecimal(item.Cash) + " | " +
				formatDecimal(item.CurrentAssets) + " | " +
				formatDecimal(item.CurrentLiabilities) + " |\n"
		}
	}

	return output
}

func FormatCashflow(c *Cashflow) string {
	output := "# Cashflow for " + c.Symbol + " (" + c.Frequency + ")\n\n"

	output += "| Date | Operating | Investing | Financing | Free Cashflow | CapEx | Dividends |\n"
	output += "| --- | --- | --- | --- | --- | --- | --- |\n"

	for _, item := range c.Items {
		if item != nil {
			output += "| " + item.Date.Format("2006-01-02") + " | " +
				formatDecimal(item.OperatingCashflow) + " | " +
				formatDecimal(item.InvestingCashflow) + " | " +
				formatDecimal(item.FinancingCashflow) + " | " +
				formatDecimal(item.FreeCashflow) + " | " +
				formatDecimal(item.CapitalExpenditure) + " | " +
				formatDecimal(item.DividendsPaid) + " |\n"
		}
	}

	return output
}

func FormatIncomeStatement(i *IncomeStatement) string {
	output := "# Income Statement for " + i.Symbol + " (" + i.Frequency + ")\n\n"

	output += "| Date | Revenue | Gross Profit | Operating Income | Net Income | EBITDA | EPS |\n"
	output += "| --- | --- | --- | --- | --- | --- | --- |\n"

	for _, item := range i.Items {
		if item != nil {
			output += "| " + item.Date.Format("2006-01-02") + " | " +
				formatDecimal(item.Revenue) + " | " +
				formatDecimal(item.GrossProfit) + " | " +
				formatDecimal(item.OperatingIncome) + " | " +
				formatDecimal(item.NetIncome) + " | " +
				formatDecimal(item.EBITDA) + " | " +
				formatDecimal(item.EPS) + " |\n"
		}
	}

	return output
}
