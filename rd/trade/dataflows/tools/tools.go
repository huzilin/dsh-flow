package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/huzilin/trade/dataflows/config"
	"github.com/huzilin/trade/dataflows/router"
)

type Tool struct {
	Name        string
	Description string
	Function    func(ctx context.Context, args ...any) (string, error)
}

var toolRegistry = make(map[string]Tool)

func RegisterTool(name, description string, fn func(ctx context.Context, args ...any) (string, error)) {
	toolRegistry[name] = Tool{
		Name:        name,
		Description: description,
		Function:    fn,
	}
}

func GetTool(name string) (Tool, bool) {
	t, ok := toolRegistry[name]
	return t, ok
}

func ListTools() []Tool {
	tools := make([]Tool, 0, len(toolRegistry))
	for _, t := range toolRegistry {
		tools = append(tools, t)
	}
	return tools
}

func ToolDescriptions() string {
	var lines []string
	lines = append(lines, "Available tools:")
	for name, tool := range toolRegistry {
		lines = append(lines, fmt.Sprintf("- %s: %s", name, tool.Description))
	}
	return joinLines(lines)
}

func joinLines(lines []string) string {
	result := ""
	for i, line := range lines {
		if i > 0 {
			result += "\n"
		}
		result += line
	}
	return result
}

func GetStockData(ctx context.Context, symbol, startDate, endDate string) (string, error) {
	r := router.NewDefaultRouter()
	result, err := r.GetStockData(ctx, symbol, startDate, endDate)
	if err != nil {
		return "", err
	}
	return marshalResult(result)
}

func GetIndicators(ctx context.Context, symbol, indicator, currDate string, lookBackDays int) (string, error) {
	r := router.NewDefaultRouter()
	result, err := r.GetIndicators(ctx, symbol, indicator, currDate, lookBackDays)
	if err != nil {
		return "", err
	}
	return marshalResult(result)
}

func GetFundamentals(ctx context.Context, ticker, currDate string) (string, error) {
	r := router.NewDefaultRouter()
	date, _ := time.Parse("2006-01-02", currDate)
	result, err := r.GetFundamentals(ctx, ticker, date)
	if err != nil {
		return "", err
	}
	return marshalResult(result)
}

func GetBalanceSheet(ctx context.Context, ticker, freq, currDate string) (string, error) {
	r := router.NewDefaultRouter()
	date, _ := time.Parse("2006-01-02", currDate)
	result, err := r.GetBalanceSheet(ctx, ticker, freq, date)
	if err != nil {
		return "", err
	}
	return marshalResult(result)
}

func GetCashflow(ctx context.Context, ticker, freq, currDate string) (string, error) {
	r := router.NewDefaultRouter()
	date, _ := time.Parse("2006-01-02", currDate)
	result, err := r.GetCashflow(ctx, ticker, freq, date)
	if err != nil {
		return "", err
	}
	return marshalResult(result)
}

func GetIncomeStatement(ctx context.Context, ticker, freq, currDate string) (string, error) {
	r := router.NewDefaultRouter()
	date, _ := time.Parse("2006-01-02", currDate)
	result, err := r.GetIncomeStatement(ctx, ticker, freq, date)
	if err != nil {
		return "", err
	}
	return marshalResult(result)
}

func GetNews(ctx context.Context, ticker, startDate, endDate string) (string, error) {
	r := router.NewDefaultRouter()
	start, _ := time.Parse("2006-01-02", startDate)
	end, _ := time.Parse("2006-01-02", endDate)
	result, err := r.GetNews(ctx, ticker, start, end, 20)
	if err != nil {
		return "", err
	}
	return marshalResult(result)
}

func GetGlobalNews(ctx context.Context, currDate string, lookBackDays, limit int) (string, error) {
	r := router.NewDefaultRouter()
	date, _ := time.Parse("2006-01-02", currDate)
	result, err := r.GetNews(ctx, "", date.AddDate(0, 0, -lookBackDays), date, limit)
	if err != nil {
		return "", err
	}
	return marshalResult(result)
}

func GetCompanyAnnouncements(ctx context.Context, ticker, startDate, endDate, category string) (string, error) {
	r := router.NewDefaultRouter()
	start, _ := time.Parse("2006-01-02", startDate)
	end, _ := time.Parse("2006-01-02", endDate)
	result, err := r.GetCompanyAnnouncements(ctx, ticker, start, end)
	if err != nil {
		return "", err
	}
	return marshalResult(result)
}

func GetMarketNews(ctx context.Context, currDate string, lookBackDays, limit int) (string, error) {
	r := router.NewDefaultRouter()
	date, _ := time.Parse("2006-01-02", currDate)
	result, err := r.GetNews(ctx, "", date.AddDate(0, 0, -lookBackDays), date, limit)
	if err != nil {
		return "", err
	}
	return marshalResult(result)
}

func GetInsiderTransactions(ctx context.Context, ticker string) (string, error) {
	r := router.NewDefaultRouter()
	return r.GetInsiderTransactions(ctx, ticker)
}

func GetMacroIndicators(ctx context.Context, indicator, currDate string, lookBackDays int) (string, error) {
	r := router.NewDefaultRouter()
	date, _ := time.Parse("2006-01-02", currDate)
	result, err := r.GetMacroIndicators(ctx, indicator, date, lookBackDays, nil)
	if err != nil {
		return "", err
	}
	return marshalResult(result)
}

func GetPredictionMarkets(ctx context.Context, topic string, limit int) (string, error) {
	r := router.NewDefaultRouter()
	result, err := r.GetPredictionMarkets(ctx, topic, limit)
	if err != nil {
		return "", err
	}
	return marshalResult(result)
}

func GetVerifiedMarketSnapshot(ctx context.Context, symbol, currDate string, lookBackDays int) (string, error) {
	r := router.NewDefaultRouter()
	date, _ := time.Parse("2006-01-02", currDate)
	return r.GetVerifiedMarketSnapshot(ctx, symbol, date, lookBackDays)
}

func marshalResult(result any) (string, error) {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", result), nil
	}
	return string(data), nil
}

func init() {
	config.Init()

	RegisterTool("get_stock_data", "Retrieve stock price data (OHLCV) for a given ticker symbol",
		func(ctx context.Context, args ...any) (string, error) {
			if len(args) < 3 {
				return "", errors.New("get_stock_data requires 3 arguments: symbol, start_date, end_date")
			}
			return GetStockData(ctx, fmt.Sprintf("%v", args[0]), fmt.Sprintf("%v", args[1]), fmt.Sprintf("%v", args[2]))
		})
	RegisterTool("get_indicators", "Retrieve a single technical indicator for a given ticker symbol",
		func(ctx context.Context, args ...any) (string, error) {
			if len(args) < 3 {
				return "", errors.New("get_indicators requires at least 3 arguments: symbol, indicator, curr_date")
			}
			lookBack := 30
			if len(args) > 3 {
				if lb, ok := args[3].(int); ok {
					lookBack = lb
				}
			}
			return GetIndicators(ctx, fmt.Sprintf("%v", args[0]), fmt.Sprintf("%v", args[1]), fmt.Sprintf("%v", args[2]), lookBack)
		})
	RegisterTool("get_fundamentals", "Retrieve comprehensive fundamental data for a given ticker symbol",
		func(ctx context.Context, args ...any) (string, error) {
			if len(args) < 1 {
				return "", errors.New("get_fundamentals requires 1 argument: ticker")
			}
			currDate := ""
			if len(args) > 1 {
				currDate = fmt.Sprintf("%v", args[1])
			}
			return GetFundamentals(ctx, fmt.Sprintf("%v", args[0]), currDate)
		})
	RegisterTool("get_balance_sheet", "Retrieve balance sheet data for a given ticker symbol",
		func(ctx context.Context, args ...any) (string, error) {
			if len(args) < 1 {
				return "", errors.New("get_balance_sheet requires 1 argument: ticker")
			}
			freq := "quarterly"
			currDate := ""
			if len(args) > 1 {
				freq = fmt.Sprintf("%v", args[1])
			}
			if len(args) > 2 {
				currDate = fmt.Sprintf("%v", args[2])
			}
			return GetBalanceSheet(ctx, fmt.Sprintf("%v", args[0]), freq, currDate)
		})
	RegisterTool("get_cashflow", "Retrieve cash flow statement data for a given ticker symbol",
		func(ctx context.Context, args ...any) (string, error) {
			if len(args) < 1 {
				return "", errors.New("get_cashflow requires 1 argument: ticker")
			}
			freq := "quarterly"
			currDate := ""
			if len(args) > 1 {
				freq = fmt.Sprintf("%v", args[1])
			}
			if len(args) > 2 {
				currDate = fmt.Sprintf("%v", args[2])
			}
			return GetCashflow(ctx, fmt.Sprintf("%v", args[0]), freq, currDate)
		})
	RegisterTool("get_income_statement", "Retrieve income statement data for a given ticker symbol",
		func(ctx context.Context, args ...any) (string, error) {
			if len(args) < 1 {
				return "", errors.New("get_income_statement requires 1 argument: ticker")
			}
			freq := "quarterly"
			currDate := ""
			if len(args) > 1 {
				freq = fmt.Sprintf("%v", args[1])
			}
			if len(args) > 2 {
				currDate = fmt.Sprintf("%v", args[2])
			}
			return GetIncomeStatement(ctx, fmt.Sprintf("%v", args[0]), freq, currDate)
		})
	RegisterTool("get_news", "Retrieve news data for a given ticker symbol",
		func(ctx context.Context, args ...any) (string, error) {
			if len(args) < 3 {
				return "", errors.New("get_news requires 3 arguments: ticker, start_date, end_date")
			}
			return GetNews(ctx, fmt.Sprintf("%v", args[0]), fmt.Sprintf("%v", args[1]), fmt.Sprintf("%v", args[2]))
		})
	RegisterTool("get_global_news", "Retrieve global news data",
		func(ctx context.Context, args ...any) (string, error) {
			if len(args) < 1 {
				return "", errors.New("get_global_news requires 1 argument: curr_date")
			}
			lookBack := 7
			limit := 10
			if len(args) > 1 {
				if lb, ok := args[1].(int); ok {
					lookBack = lb
				}
			}
			if len(args) > 2 {
				if l, ok := args[2].(int); ok {
					limit = l
				}
			}
			return GetGlobalNews(ctx, fmt.Sprintf("%v", args[0]), lookBack, limit)
		})
	RegisterTool("get_company_announcements", "Retrieve company announcements for a given ticker symbol",
		func(ctx context.Context, args ...any) (string, error) {
			if len(args) < 3 {
				return "", errors.New("get_company_announcements requires 3 arguments: ticker, start_date, end_date")
			}
			category := "全部"
			if len(args) > 3 {
				category = fmt.Sprintf("%v", args[3])
			}
			return GetCompanyAnnouncements(ctx, fmt.Sprintf("%v", args[0]), fmt.Sprintf("%v", args[1]), fmt.Sprintf("%v", args[2]), category)
		})
	RegisterTool("get_market_news", "Retrieve market-wide news and policy updates",
		func(ctx context.Context, args ...any) (string, error) {
			if len(args) < 1 {
				return "", errors.New("get_market_news requires 1 argument: curr_date")
			}
			lookBack := 7
			limit := 10
			if len(args) > 1 {
				if lb, ok := args[1].(int); ok {
					lookBack = lb
				}
			}
			if len(args) > 2 {
				if l, ok := args[2].(int); ok {
					limit = l
				}
			}
			return GetMarketNews(ctx, fmt.Sprintf("%v", args[0]), lookBack, limit)
		})
	RegisterTool("get_insider_transactions", "Retrieve insider transaction information about a company",
		func(ctx context.Context, args ...any) (string, error) {
			if len(args) < 1 {
				return "", errors.New("get_insider_transactions requires 1 argument: ticker")
			}
			return GetInsiderTransactions(ctx, fmt.Sprintf("%v", args[0]))
		})
	RegisterTool("get_macro_indicators", "Retrieve macroeconomic indicator time series from FRED",
		func(ctx context.Context, args ...any) (string, error) {
			if len(args) < 2 {
				return "", errors.New("get_macro_indicators requires 2 arguments: indicator, curr_date")
			}
			lookBack := 365
			if len(args) > 2 {
				if lb, ok := args[2].(int); ok {
					lookBack = lb
				}
			}
			return GetMacroIndicators(ctx, fmt.Sprintf("%v", args[0]), fmt.Sprintf("%v", args[1]), lookBack)
		})
	RegisterTool("get_prediction_markets", "Retrieve live prediction market probabilities",
		func(ctx context.Context, args ...any) (string, error) {
			if len(args) < 1 {
				return "", errors.New("get_prediction_markets requires 1 argument: topic")
			}
			limit := 6
			if len(args) > 1 {
				if l, ok := args[1].(int); ok {
					limit = l
				}
			}
			return GetPredictionMarkets(ctx, fmt.Sprintf("%v", args[0]), limit)
		})
	RegisterTool("get_verified_market_snapshot", "Retrieve a deterministic verification snapshot of latest OHLCV, technical indicators, and recent closes. Use as source of truth before making exact claims about price levels, indicators, or historical comparisons.",
		func(ctx context.Context, args ...any) (string, error) {
			if len(args) < 2 {
				return "", errors.New("get_verified_market_snapshot requires 2 arguments: symbol, curr_date")
			}
			lookBack := 30
			if len(args) > 2 {
				if lb, ok := args[2].(int); ok {
					lookBack = lb
				}
			}
			return GetVerifiedMarketSnapshot(ctx, fmt.Sprintf("%v", args[0]), fmt.Sprintf("%v", args[1]), lookBack)
		})
}
