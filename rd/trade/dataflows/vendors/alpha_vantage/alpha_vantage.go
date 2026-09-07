package alpha_vantage

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/huzilin/trade/dataflows/config"
	"github.com/huzilin/trade/dataflows/errs"
	"github.com/huzilin/trade/dataflows/interfaces"
	"github.com/huzilin/trade/dataflows/models"
	"github.com/huzilin/trade/dataflows/normalizer"
	"github.com/huzilin/trade/dataflows/vendors/tool_registry"
	"github.com/huzilin/trade/dataflows/vendors/utils"
	"github.com/huzilin/trade/logger"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

const (
	AlphaVantageBase    = "https://www.alphavantage.co/query"
	AlphaVantageTimeout = 30
)

func AlphaVantageAPIKey() (string, error) {
	cfg := config.Get()
	if cfg.AlphaVantageAPIKey == "" {
		return "", errs.NewVendorNotConfiguredError("alpha_vantage",
			"ALPHA_VANTAGE_API_KEY environment variable is not set. Get a free key at https://www.alphavantage.co/support/#api-key.")
	}
	return cfg.AlphaVantageAPIKey, nil
}

func AlphaVantageRequest(params map[string]string) (map[string]any, error) {
	apiKey, err := AlphaVantageAPIKey()
	if err != nil {
		return nil, err
	}

	params["apikey"] = apiKey

	query := url.Values{}
	for k, v := range params {
		query.Set(k, v)
	}

	requestURL := fmt.Sprintf("%s?%s", AlphaVantageBase, query.Encode())

	logger.L().Info("Alpha Vantage API request", zap.String("function", params["function"]))

	client := &http.Client{Timeout: time.Duration(AlphaVantageTimeout) * time.Second}
	resp, err := client.Get(requestURL)
	if err != nil {
		return nil, errors.Wrap(err, "alpha vantage request failed")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response body")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("alpha vantage API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, errors.Wrap(err, "failed to parse JSON response")
	}

	if note, ok := result["Note"].(string); ok && strings.Contains(note, "API call frequency") {
		return nil, errs.NewVendorRateLimitError("alpha_vantage", params["symbol"])
	}

	if info, ok := result["Information"].(string); ok {
		return nil, errors.Errorf("alpha vantage API error: %s", info)
	}

	return result, nil
}

// AlphaVantageStockData 获取股票K线数据（AlphaVantage供应商实现）
func AlphaVantageStockData(ctx context.Context, symbol, startDate, endDate string) ([]*models.Candle, error) {
	canonical := utils.NormalizeSymbol(symbol)

	result, err := AlphaVantageRequest(map[string]string{
		"function":   "TIME_SERIES_DAILY",
		"symbol":     canonical,
		"outputsize": "full",
	})
	if err != nil {
		return nil, err
	}

	timeSeries, ok := result["Time Series (Daily)"].(map[string]any)
	if !ok {
		return nil, errs.NewNoMarketDataError(symbol, canonical, "no daily time series data")
	}

	candles := make([]*models.Candle, 0, len(timeSeries))
	for date, data := range timeSeries {
		if date < startDate || date > endDate {
			continue
		}

		if dailyData, ok := data.(map[string]any); ok {
			candle := &models.Candle{
				Name: canonical,
				Code: canonical,
			}

			candle.Date, _ = time.Parse("2006-01-02", date)

			if v := getStringValue(dailyData, "1. open"); v != "" {
				candle.Open, _ = decimal.NewFromString(v)
			}
			if v := getStringValue(dailyData, "2. high"); v != "" {
				candle.High, _ = decimal.NewFromString(v)
			}
			if v := getStringValue(dailyData, "3. low"); v != "" {
				candle.Low, _ = decimal.NewFromString(v)
			}
			if v := getStringValue(dailyData, "4. close"); v != "" {
				candle.Close, _ = decimal.NewFromString(v)
			}
			if v := getStringValue(dailyData, "5. volume"); v != "" {
				if vol, err := strconv.ParseInt(v, 10, 64); err == nil {
					candle.Volume = vol
				}
			}

			candles = append(candles, candle)
		}
	}

	if len(candles) == 0 {
		return nil, errs.NewNoMarketDataError(symbol, canonical, "no data points returned")
	}

	return candles, nil
}

func AlphaVantageIndicators(ctx context.Context, args ...any) (string, error) {
	if len(args) < 3 {
		return "", errors.New("get_indicators requires 3 arguments: symbol, indicator, curr_date")
	}

	symbol := fmt.Sprintf("%v", args[0])
	indicatorName := fmt.Sprintf("%v", args[1])

	canonical := utils.NormalizeSymbol(symbol)

	indicatorFunctions := map[string]string{
		"rsi":    "RSI",
		"macd":   "MACD",
		"sma":    "SMA",
		"ema":    "EMA",
		"wma":    "WMA",
		"bbands": "BBANDS",
		"atr":    "ATR",
	}

	function, ok := indicatorFunctions[strings.ToLower(indicatorName)]
	if !ok {
		return "", errors.Errorf("unsupported indicator for Alpha Vantage: %s", indicatorName)
	}

	params := map[string]string{
		"function":    function,
		"symbol":      canonical,
		"interval":    "daily",
		"series_type": "close",
	}

	switch strings.ToLower(indicatorName) {
	case "rsi":
		params["time_period"] = "14"
	case "macd":
		params["fastlimit"] = "0.02"
		params["slowlimit"] = "0.05"
	case "sma", "ema", "wma":
		params["time_period"] = "20"
	case "bbands":
		params["time_period"] = "20"
		params["nbdevup"] = "2"
		params["nbdevdn"] = "2"
	case "atr":
		params["time_period"] = "14"
	}

	result, err := AlphaVantageRequest(params)
	if err != nil {
		return "", errors.Wrap(err, "alpha vantage indicators request failed")
	}

	var indicatorKey string
	for key := range result {
		if strings.Contains(key, "Technical Analysis") {
			indicatorKey = key
			break
		}
	}

	if indicatorKey == "" {
		return "", errs.NewNoMarketDataError(symbol, canonical, "no technical indicator data")
	}

	indicatorData, ok := result[indicatorKey].(map[string]any)
	if !ok {
		return "", errors.New("invalid indicator data format")
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("## %s for %s\n\n", function, canonical))
	output.WriteString(fmt.Sprintf("| Date | Value |\n| --- | --- |\n"))

	count := 0
	for date, data := range indicatorData {
		if count >= 30 {
			break
		}
		if dailyData, ok := data.(map[string]any); ok {
			for _, v := range dailyData {
				output.WriteString(fmt.Sprintf("| %s | %v |\n", date, v))
				break
			}
		}
		count++
	}

	return output.String(), nil
}

func AlphaVantageFundamentals(ctx context.Context, args ...any) (string, error) {
	if len(args) < 1 {
		return "", errors.New("get_fundamentals requires at least 1 argument: ticker")
	}

	ticker := fmt.Sprintf("%v", args[0])
	canonical := utils.NormalizeSymbol(ticker)

	result, err := AlphaVantageRequest(map[string]string{
		"function": "OVERVIEW",
		"symbol":   canonical,
	})
	if err != nil {
		return "", errors.Wrap(err, "alpha vantage fundamentals request failed")
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("# Company Overview for %s\n\n", canonical))

	fields := []string{
		"Name", "Description", "Exchange", "Currency", "Country",
		"Sector", "Industry", "MarketCapitalization", "EBITDA",
		"PERatio", "PEGRatio", "BookValue", "DividendPerShare",
		"DividendYield", "EPS", "RevenuePerShareTTM",
		"ProfitMargin", "OperatingMarginTTM", "ReturnOnAssetsTTM",
		"ReturnOnEquityTTM", "RevenueTTM", "GrossProfitTTM",
		"DilutedEPSTTM", "QuarterlyEarningsGrowthYOY",
		"QuarterlyRevenueGrowthYOY", "AnalystTargetPrice",
		"TrailingPE", "ForwardPE", "PriceToSalesRatioTTM",
		"PriceToBookRatio", "EVToRevenue", "EVToEBITDA",
		"Beta", "52WeekHigh", "52WeekLow", "50DayMovingAverage",
		"200DayMovingAverage", "SharesOutstanding",
	}

	for _, field := range fields {
		if val, ok := result[field]; ok && val != nil && val != "" {
			output.WriteString(fmt.Sprintf("%s: %v\n", field, val))
		}
	}

	return output.String(), nil
}

func AlphaVantageBalanceSheet(ctx context.Context, args ...any) (string, error) {
	if len(args) < 1 {
		return "", errors.New("get_balance_sheet requires at least 1 argument: ticker")
	}

	ticker := fmt.Sprintf("%v", args[0])
	freq := "quarterly"
	if len(args) > 1 {
		freq = fmt.Sprintf("%v", args[1])
	}

	canonical := utils.NormalizeSymbol(ticker)

	result, err := AlphaVantageRequest(map[string]string{
		"function": "BALANCE_SHEET",
		"symbol":   canonical,
	})
	if err != nil {
		return "", errors.Wrap(err, "alpha vantage balance sheet request failed")
	}

	key := "quarterlyReports"
	if freq == "annual" {
		key = "annualReports"
	}

	reports, ok := result[key].([]any)
	if !ok || len(reports) == 0 {
		return "", errs.NewNoMarketDataError(ticker, canonical, "no balance sheet data")
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("# Balance Sheet for %s (%s)\n\n", canonical, freq))

	if report, ok := reports[0].(map[string]any); ok {
		if fiscalDate, ok := report["fiscalDateEnding"].(string); ok {
			output.WriteString(fmt.Sprintf("Fiscal Date Ending: %s\n\n", fiscalDate))
		}

		fields := []string{
			"totalAssets", "totalCurrentAssets", "cashAndCashEquivalentsAtCarryingValue",
			"cashAndShortTermInvestments", "inventory", "currentNetReceivables",
			"totalNonCurrentAssets", "propertyPlantEquipment", "accumulatedDepreciationAmortizationPPE",
			"intangibleAssets", "intangibleAssetsExcludingGoodwill", "goodwill",
			"investments", "longTermInvestments", "shortTermInvestments",
			"otherCurrentAssets", "otherNonCurrentAssets",
			"totalLiabilities", "totalCurrentLiabilities", "currentAccountsPayable",
			"deferredRevenue", "currentDebt", "shortTermDebt",
			"totalNonCurrentLiabilities", "capitalLeaseObligations",
			"longTermDebt", "currentLongTermDebt", "longTermDebtNoncurrent",
			"shortLongTermDebtTotal", "otherCurrentLiabilities", "otherNonCurrentLiabilities",
			"totalShareholderEquity", "treasuryStock", "retainedEarnings",
			"commonStock", "commonStockSharesOutstanding",
		}

		for _, field := range fields {
			if val, ok := report[field].(string); ok && val != "None" && val != "" {
				output.WriteString(fmt.Sprintf("%s: %s\n", field, val))
			}
		}
	}

	return output.String(), nil
}

func AlphaVantageCashflow(ctx context.Context, args ...any) (string, error) {
	if len(args) < 1 {
		return "", errors.New("get_cashflow requires at least 1 argument: ticker")
	}

	ticker := fmt.Sprintf("%v", args[0])
	freq := "quarterly"
	if len(args) > 1 {
		freq = fmt.Sprintf("%v", args[1])
	}

	canonical := utils.NormalizeSymbol(ticker)

	result, err := AlphaVantageRequest(map[string]string{
		"function": "CASH_FLOW",
		"symbol":   canonical,
	})
	if err != nil {
		return "", errors.Wrap(err, "alpha vantage cashflow request failed")
	}

	key := "quarterlyReports"
	if freq == "annual" {
		key = "annualReports"
	}

	reports, ok := result[key].([]any)
	if !ok || len(reports) == 0 {
		return "", errs.NewNoMarketDataError(ticker, canonical, "no cash flow data")
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("# Cash Flow for %s (%s)\n\n", canonical, freq))

	if report, ok := reports[0].(map[string]any); ok {
		if fiscalDate, ok := report["fiscalDateEnding"].(string); ok {
			output.WriteString(fmt.Sprintf("Fiscal Date Ending: %s\n\n", fiscalDate))
		}

		fields := []string{
			"operatingCashflow", "paymentsForOperatingActivities",
			"proceedsFromOperatingActivities", "changeInOperatingLiabilities",
			"changeInOperatingAssets", "depreciationDepletionAndAmortization",
			"capitalExpenditures", "changeInReceivables", "changeInInventory",
			"profitLoss", "cashflowFromInvestment", "cashflowFromFinancing",
			"proceedsFromRepaymentsOfShortTermDebt", "paymentsForRepurchaseOfCommonStock",
			"paymentsForRepurchaseOfEquity", "paymentsForRepurchaseOfPreferredStock",
			"dividendPayout", "dividendPayoutCommonStock", "dividendPayoutPreferredStock",
			"proceedsFromIssuanceOfCommonStock", "proceedsFromIssuanceOfLongTermDebtAndCapitalSecuritiesNet",
			"proceedsFromIssuanceOfPreferredStock", "proceedsFromRepurchaseOfEquity",
			"proceedsFromSaleOfTreasuryStock", "changeInCashAndCashEquivalents",
			"netIncome",
		}

		for _, field := range fields {
			if val, ok := report[field].(string); ok && val != "None" && val != "" {
				output.WriteString(fmt.Sprintf("%s: %s\n", field, val))
			}
		}
	}

	return output.String(), nil
}

func AlphaVantageIncomeStatement(ctx context.Context, args ...any) (string, error) {
	if len(args) < 1 {
		return "", errors.New("get_income_statement requires at least 1 argument: ticker")
	}

	ticker := fmt.Sprintf("%v", args[0])
	freq := "quarterly"
	if len(args) > 1 {
		freq = fmt.Sprintf("%v", args[1])
	}

	canonical := utils.NormalizeSymbol(ticker)

	result, err := AlphaVantageRequest(map[string]string{
		"function": "INCOME_STATEMENT",
		"symbol":   canonical,
	})
	if err != nil {
		return "", errors.Wrap(err, "alpha vantage income statement request failed")
	}

	key := "quarterlyReports"
	if freq == "annual" {
		key = "annualReports"
	}

	reports, ok := result[key].([]any)
	if !ok || len(reports) == 0 {
		return "", errs.NewNoMarketDataError(ticker, canonical, "no income statement data")
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("# Income Statement for %s (%s)\n\n", canonical, freq))

	if report, ok := reports[0].(map[string]any); ok {
		if fiscalDate, ok := report["fiscalDateEnding"].(string); ok {
			output.WriteString(fmt.Sprintf("Fiscal Date Ending: %s\n\n", fiscalDate))
		}

		fields := []string{
			"grossProfit", "totalRevenue", "costOfRevenue", "costofGoodsAndServicesSold",
			"operatingIncome", "sellingGeneralAndAdministrative", "researchAndDevelopment",
			"operatingExpenses", "investmentIncomeNet", "netInterestIncome",
			"interestIncome", "interestExpense", "nonInterestIncome",
			"otherNonOperatingIncome", "depreciation", "depreciationAndAmortization",
			"incomeBeforeTax", "incomeTaxExpense", "interestAndDebtExpense",
			"netIncomeFromContinuingOperations", "comprehensiveIncomeNetOfTax",
			"ebit", "ebitda", "netIncome",
		}

		for _, field := range fields {
			if val, ok := report[field].(string); ok && val != "None" && val != "" {
				output.WriteString(fmt.Sprintf("%s: %s\n", field, val))
			}
		}
	}

	return output.String(), nil
}

func AlphaVantageNews(ctx context.Context, args ...any) (string, error) {
	if len(args) < 3 {
		return "", errors.New("get_news requires 3 arguments: ticker, start_date, end_date")
	}

	ticker := fmt.Sprintf("%v", args[0])
	startDate := fmt.Sprintf("%v", args[1])
	endDate := fmt.Sprintf("%v", args[2])

	canonical := utils.NormalizeSymbol(ticker)

	result, err := AlphaVantageRequest(map[string]string{
		"function":  "NEWS_SENTIMENT",
		"tickers":   canonical,
		"time_from": startDate + "T0000",
		"time_to":   endDate + "T2359",
		"limit":     "50",
		"sort":      "LATEST",
	})
	if err != nil {
		return "", errors.Wrap(err, "alpha vantage news request failed")
	}

	feed, ok := result["feed"].([]any)
	if !ok || len(feed) == 0 {
		return "", errs.NewNoMarketDataError(ticker, canonical, "no news data")
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("# News for %s from %s to %s\n\n", canonical, startDate, endDate))

	for i, item := range feed {
		if i >= 10 {
			break
		}
		if news, ok := item.(map[string]any); ok {
			title := getStringValue(news, "title")
			summary := getStringValue(news, "summary")
			url := getStringValue(news, "url")
			timePublished := getStringValue(news, "time_published")
			sentiment := ""
			if sentimentData, ok := news["overall_sentiment_score"].(float64); ok {
				sentiment = fmt.Sprintf(" (Sentiment: %.2f)", sentimentData)
			}

			output.WriteString(fmt.Sprintf("- **%s**: %s%s\n", timePublished, title, sentiment))
			if summary != "" {
				output.WriteString(fmt.Sprintf("  %s\n", summary))
			}
			if url != "" {
				output.WriteString(fmt.Sprintf("  URL: %s\n", url))
			}
			output.WriteString("\n")
		}
	}

	return output.String(), nil
}

// AlphaVantageGlobalNews 获取新闻（AlphaVantage供应商实现，提供全球财经新闻）
func AlphaVantageGlobalNews(ctx context.Context, symbol string, startDate, endDate time.Time, limit int) (*models.NewsResult, error) {
	result, err := AlphaVantageRequest(map[string]string{
		"function":  "NEWS_SENTIMENT",
		"time_from": utils.FormatDate(startDate) + "T0000",
		"time_to":   utils.FormatDate(endDate) + "T2359",
		"limit":     fmt.Sprintf("%d", limit),
		"sort":      "LATEST",
	})
	if err != nil {
		return nil, errors.Wrap(err, "alpha vantage global news request failed")
	}

	feed, ok := result["feed"].([]any)
	if !ok || len(feed) == 0 {
		return nil, errors.New("no global news data")
	}

	newsItems := make([]*models.NewsItem, 0)

	for i, item := range feed {
		if i >= limit {
			break
		}
		if news, ok := item.(map[string]any); ok {
			title := getStringValue(news, "title")
			summary := getStringValue(news, "summary")
			timePublished := getStringValue(news, "time_published")
			url := getStringValue(news, "url")

			publishDate, _ := time.Parse("20060102T150405", timePublished)

			newsItems = append(newsItems, &models.NewsItem{
				Title:       title,
				Summary:     summary,
				Source:      "Alpha Vantage",
				URL:         url,
				PublishDate: publishDate,
			})
		}
	}

	return &models.NewsResult{
		Symbol: symbol,
		News:   newsItems,
		Source: "alpha_vantage",
	}, nil
}

// AlphaVantageInsiderTransactions 获取内部交易数据（AlphaVantage供应商实现）
func AlphaVantageInsiderTransactions(ctx context.Context, symbol string) (string, error) {
	canonical := utils.NormalizeSymbol(symbol)

	result, err := AlphaVantageRequest(map[string]string{
		"function": "INSIDER_TRANSACTIONS",
		"symbol":   canonical,
	})
	if err != nil {
		return "", errors.Wrap(err, "alpha vantage insider transactions request failed")
	}

	data, ok := result["data"].([]any)
	if !ok || len(data) == 0 {
		return "", errs.NewNoMarketDataError(symbol, canonical, "no insider transaction data")
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("# Insider Transactions for %s\n\n", canonical))
	output.WriteString("| Date | Insider | Transaction | Shares | Price |\n")
	output.WriteString("| --- | --- | --- | --- | --- |\n")

	for i, item := range data {
		if i >= 20 {
			break
		}
		if transaction, ok := item.(map[string]any); ok {
			date := getStringValue(transaction, "transaction_date")
			insider := getStringValue(transaction, "executive")
			txType := getStringValue(transaction, "transaction_type")
			shares := getStringValue(transaction, "shares")
			price := getStringValue(transaction, "price")

			output.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s |\n",
				date, insider, txType, shares, price))
		}
	}

	return output.String(), nil
}

func getStringValue(m map[string]any, key string) string {
	if val, ok := m[key]; ok {
		if s, ok := val.(string); ok {
			return s
		}
		return fmt.Sprintf("%v", val)
	}
	return ""
}

func init() {
	tool_registry.Registry.RegisterGetStockData("alpha_vantage", AlphaVantageStockData)
	tool_registry.Registry.RegisterGetNews("alpha_vantage", AlphaVantageGlobalNews)
	tool_registry.Registry.RegisterGetInsiderTransactions("alpha_vantage", AlphaVantageInsiderTransactions)
}

type AlphaVantageProvider struct{}

func (p *AlphaVantageProvider) Name() string {
	return "alpha_vantage"
}

func (p *AlphaVantageProvider) Capabilities() []interfaces.Capability {
	return []interfaces.Capability{
		interfaces.CapabilityStockData,
		interfaces.CapabilityIndicators,
		interfaces.CapabilityFundamentals,
		interfaces.CapabilityBalanceSheet,
		interfaces.CapabilityCashflow,
		interfaces.CapabilityIncomeStatement,
		interfaces.CapabilityNews,
		interfaces.CapabilityInsiderTransactions,
	}
}

func (p *AlphaVantageProvider) Markets() []normalizer.MarketType {
	return []normalizer.MarketType{normalizer.MarketUSShare, normalizer.MarketGlobal}
}

var _ interfaces.Provider = (*AlphaVantageProvider)(nil)
