package yfinance

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/huzilin/trade/dataflows/errs"
	"github.com/huzilin/trade/dataflows/indicators"
	"github.com/huzilin/trade/dataflows/interfaces"
	"github.com/huzilin/trade/dataflows/market_symbols"
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
	YahooFinanceTimeout = 30
)

type YahooChartResponse struct {
	Chart struct {
		Result []YahooChartResult `json:"result"`
		Error  *YahooChartError   `json:"error"`
	} `json:"chart"`
}

type YahooChartResult struct {
	Meta       YahooMeta       `json:"meta"`
	Timestamp  []int64         `json:"timestamp"`
	Indicators YahooIndicators `json:"indicators"`
}

type YahooMeta struct {
	Symbol               string  `json:"symbol"`
	RegularMarketPrice   float64 `json:"regularMarketPrice"`
	ChartPreviousClose   float64 `json:"chartPreviousClose"`
	Currency             string  `json:"currency"`
	ExchangeName         string  `json:"exchangeName"`
	InstrumentType       string  `json:"instrumentType"`
	FirstTradeDate       int64   `json:"firstTradeDate"`
	RegularMarketTime    int64   `json:"regularMarketTime"`
	HasPrePostMarketData bool    `json:"hasPrePostMarketData"`
}

type YahooIndicators struct {
	Quote []struct {
		Open   []*float64 `json:"open"`
		High   []*float64 `json:"high"`
		Low    []*float64 `json:"low"`
		Close  []*float64 `json:"close"`
		Volume []*int64   `json:"volume"`
	} `json:"quote"`
	AdjClose []struct {
		AdjClose []*float64 `json:"adjclose"`
	} `json:"adjclose"`
}

type YahooChartError struct {
	Code        string `json:"code"`
	Description string `json:"description"`
}

func YahooFinanceRequest(symbol string, period1, period2 int64, interval string) (*YahooChartResponse, error) {
	query := url.Values{}
	query.Set("period1", strconv.FormatInt(period1, 10))
	query.Set("period2", strconv.FormatInt(period2, 10))
	query.Set("interval", interval)
	query.Set("events", "history")
	query.Set("includeAdjustedClose", "true")

	chartURL := fmt.Sprintf("https://query1.finance.yahoo.com/v8/finance/chart/%s?%s",
		url.QueryEscape(symbol), query.Encode())

	logger.L().Info("Yahoo Finance API request", zap.String("url", chartURL))

	client := &http.Client{Timeout: time.Duration(YahooFinanceTimeout) * time.Second}
	resp, err := client.Get(chartURL)
	if err != nil {
		return nil, errors.Wrap(err, "yahoo finance request failed")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response body")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("yahoo finance API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result YahooChartResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, errors.Wrap(err, "failed to parse JSON response")
	}

	if result.Chart.Error != nil {
		return nil, errors.Errorf("yahoo finance API error: %s - %s",
			result.Chart.Error.Code, result.Chart.Error.Description)
	}

	if len(result.Chart.Result) == 0 {
		return nil, errs.NewNoMarketDataError(symbol, symbol, "empty result from Yahoo Finance")
	}

	return &result, nil
}

func parseYahooChartResult(result *YahooChartResponse) (indicators.Series, error) {
	if len(result.Chart.Result) == 0 {
		return nil, errors.New("no data in response")
	}

	r := result.Chart.Result[0]
	quotes := r.Indicators.Quote
	if len(quotes) == 0 {
		return nil, errors.New("no quote data in response")
	}

	q := quotes[0]
	timestamps := r.Timestamp

	var adjCloses []*float64
	if len(r.Indicators.AdjClose) > 0 {
		adjCloses = r.Indicators.AdjClose[0].AdjClose
	}

	series := make(indicators.Series, 0, len(timestamps))

	for i := 0; i < len(timestamps); i++ {
		if q.Close[i] == nil {
			continue
		}

		date := time.Unix(timestamps[i], 0).UTC().Format("2006-01-02")

		ohlcv := indicators.OHLCV{
			Date:  date,
			Close: *q.Close[i],
		}

		if q.Open[i] != nil {
			ohlcv.Open = *q.Open[i]
		}
		if q.High[i] != nil {
			ohlcv.High = *q.High[i]
		}
		if q.Low[i] != nil {
			ohlcv.Low = *q.Low[i]
		}
		if q.Volume[i] != nil {
			ohlcv.Volume = float64(*q.Volume[i])
		}

		if i < len(adjCloses) && adjCloses[i] != nil {
			ohlcv.Close = *adjCloses[i]
		}

		series = append(series, ohlcv)
	}

	return series, nil
}

// GetStockData 获取股票K线数据（YahooFinance供应商实现，支持美股和港股）
func GetStockData(ctx context.Context, symbol, startDate, endDate string) ([]*models.Candle, error) {
	canonical := utils.NormalizeSymbol(symbol)

	if market_symbols.IsHKSymbol(symbol) {
		plainSymbol, err := market_symbols.ToPlainSymbol(symbol)
		if err == nil {
			canonical = plainSymbol + ".HK"
		}
	}

	startDt, err := utils.ParseDate(startDate)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid start_date format: %s", startDate)
	}
	endDt, err := utils.ParseDate(endDate)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid end_date format: %s", endDate)
	}

	period1 := startDt.Unix()
	period2 := endDt.AddDate(0, 0, 1).Unix()

	result, err := YahooFinanceRequest(canonical, period1, period2, "1d")
	if err != nil {
		if _, ok := err.(*errs.NoMarketDataError); ok {
			return nil, err
		}
		return nil, errors.Wrap(err, "yahoo finance request failed")
	}

	series, err := parseYahooChartResult(result)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse yahoo chart result")
	}

	if len(series) == 0 {
		return nil, errs.NewNoMarketDataError(symbol, canonical, "no data points returned")
	}

	name := ""
	if len(result.Chart.Result) > 0 {
		name = result.Chart.Result[0].Meta.Symbol
	}

	candles := make([]*models.Candle, 0, len(series))
	for _, s := range series {
		candle := &models.Candle{
			Name:   name,
			Code:   canonical,
			Volume: int64(s.Volume),
		}
		candle.Date, _ = time.Parse("2006-01-02", s.Date)
		candle.Open = decimal.NewFromFloat(s.Open)
		candle.High = decimal.NewFromFloat(s.High)
		candle.Low = decimal.NewFromFloat(s.Low)
		candle.Close = decimal.NewFromFloat(s.Close)
		candles = append(candles, candle)
	}

	return candles, nil
}

// GetStockIndicators 获取技术指标数据（YahooFinance供应商实现）
func GetStockIndicators(ctx context.Context, symbol, indicator, currDate string, lookBackDays int) (*models.IndicatorResult, error) {
	supportedIndicators := map[string]bool{
		"rsi":       true,
		"macd":      true,
		"boll":      true,
		"bollinger": true,
		"atr":       true,
		"sma50":     true,
		"sma200":    true,
		"vwma":      true,
		"mfi":       true,
	}
	if !supportedIndicators[strings.ToLower(indicator)] {
		return nil, errors.Errorf("unsupported indicator: %s (supported: rsi, macd, boll, atr, sma50, sma200, vwma, mfi)", indicator)
	}

	canonical := utils.NormalizeSymbol(symbol)

	endDt, _ := utils.ParseDate(currDate)
	startDt := utils.AddDays(endDt, -lookBackDays-60)

	period1 := startDt.Unix()
	period2 := endDt.AddDate(0, 0, 1).Unix()

	result, err := YahooFinanceRequest(canonical, period1, period2, "1d")
	if err != nil {
		return nil, errors.Wrap(err, "yahoo finance request failed")
	}

	series, err := parseYahooChartResult(result)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse yahoo chart result")
	}

	if len(series) == 0 {
		return nil, errs.NewNoMarketDataError(symbol, canonical, "no data points for indicator calculation")
	}

	closes := indicators.ExtractClose(series)
	indicatorValues := make([]*models.IndicatorValue, 0)

	switch strings.ToLower(indicator) {
	case "rsi":
		period := 14
		values, err := indicators.RSI(closes, period)
		if err != nil {
			return nil, errors.Wrap(err, "RSI calculation failed")
		}
		for i := len(series) - lookBackDays; i < len(series); i++ {
			if i >= 0 && !math.IsNaN(values[i]) {
				date, _ := time.Parse("2006-01-02", series[i].Date)
				indicatorValues = append(indicatorValues, &models.IndicatorValue{
					Date:  date,
					Value: decimal.NewFromFloat(values[i]),
				})
			}
		}

	case "macd":
		macdResult, err := indicators.MACD(closes, 12, 26, 9)
		if err != nil {
			return nil, errors.Wrap(err, "MACD calculation failed")
		}
		for i := len(series) - lookBackDays; i < len(series); i++ {
			if i >= 0 && !math.IsNaN(macdResult.MACD[i]) {
				date, _ := time.Parse("2006-01-02", series[i].Date)
				indicatorValues = append(indicatorValues, &models.IndicatorValue{
					Date:  date,
					Value: decimal.NewFromFloat(macdResult.MACD[i]),
				})
			}
		}

	case "boll", "bollinger":
		bbResult, err := indicators.BollingerBands(closes, 20, 2.0)
		if err != nil {
			return nil, errors.Wrap(err, "Bollinger Bands calculation failed")
		}
		for i := len(series) - lookBackDays; i < len(series); i++ {
			if i >= 0 && !math.IsNaN(bbResult.Middle[i]) {
				date, _ := time.Parse("2006-01-02", series[i].Date)
				indicatorValues = append(indicatorValues, &models.IndicatorValue{
					Date:  date,
					Value: decimal.NewFromFloat(bbResult.Middle[i]),
				})
			}
		}

	case "atr":
		period := 14
		values, err := indicators.ATR(series, period)
		if err != nil {
			return nil, errors.Wrap(err, "ATR calculation failed")
		}
		for i := len(series) - lookBackDays; i < len(series); i++ {
			if i >= 0 && !math.IsNaN(values[i]) {
				date, _ := time.Parse("2006-01-02", series[i].Date)
				indicatorValues = append(indicatorValues, &models.IndicatorValue{
					Date:  date,
					Value: decimal.NewFromFloat(values[i]),
				})
			}
		}

	case "sma50":
		period := 50
		values, err := indicators.SMA(closes, period)
		if err != nil {
			return nil, errors.Wrap(err, "SMA50 calculation failed")
		}
		for i := len(series) - lookBackDays; i < len(series); i++ {
			if i >= 0 && !math.IsNaN(values[i]) {
				date, _ := time.Parse("2006-01-02", series[i].Date)
				indicatorValues = append(indicatorValues, &models.IndicatorValue{
					Date:  date,
					Value: decimal.NewFromFloat(values[i]),
				})
			}
		}

	case "sma200":
		period := 200
		values, err := indicators.SMA(closes, period)
		if err != nil {
			return nil, errors.Wrap(err, "SMA200 calculation failed")
		}
		for i := len(series) - lookBackDays; i < len(series); i++ {
			if i >= 0 && !math.IsNaN(values[i]) {
				date, _ := time.Parse("2006-01-02", series[i].Date)
				indicatorValues = append(indicatorValues, &models.IndicatorValue{
					Date:  date,
					Value: decimal.NewFromFloat(values[i]),
				})
			}
		}

	case "vwma":
		period := 20
		values, err := indicators.VWMA(series, period)
		if err != nil {
			return nil, errors.Wrap(err, "VWMA calculation failed")
		}
		for i := len(series) - lookBackDays; i < len(series); i++ {
			if i >= 0 && !math.IsNaN(values[i]) {
				date, _ := time.Parse("2006-01-02", series[i].Date)
				indicatorValues = append(indicatorValues, &models.IndicatorValue{
					Date:  date,
					Value: decimal.NewFromFloat(values[i]),
				})
			}
		}

	case "mfi":
		period := 14
		values, err := indicators.MFI(series, period)
		if err != nil {
			return nil, errors.Wrap(err, "MFI calculation failed")
		}
		for i := len(series) - lookBackDays; i < len(series); i++ {
			if i >= 0 && !math.IsNaN(values[i]) {
				date, _ := time.Parse("2006-01-02", series[i].Date)
				indicatorValues = append(indicatorValues, &models.IndicatorValue{
					Date:  date,
					Value: decimal.NewFromFloat(values[i]),
				})
			}
		}

	default:
		return nil, errors.Errorf("unsupported indicator: %s (supported: rsi, macd, boll, atr, sma50, sma200, vwma, mfi)", indicator)
	}

	return &models.IndicatorResult{
		Symbol:    canonical,
		Indicator: indicator,
		Values:    indicatorValues,
		Source:    "yfinance",
	}, nil
}

func GetFundamentals(ctx context.Context, args ...any) (string, error) {
	if len(args) < 1 {
		return "", errors.New("get_fundamentals requires at least 1 argument: ticker")
	}

	ticker := fmt.Sprintf("%v", args[0])
	canonical := utils.NormalizeSymbol(ticker)

	fundamentals, err := fetchYahooFundamentals(canonical)
	if err != nil {
		logger.L().Warn("Failed to fetch detailed fundamentals", zap.String("symbol", canonical), zap.Error(err))
		return fallbackFundamentals(ticker, canonical), nil
	}

	return fundamentals, nil
}

func fetchYahooFundamentals(symbol string) (string, error) {
	url := fmt.Sprintf("https://query1.finance.yahoo.com/v10/finance/quoteSummary/%s?modules=assetProfile,summaryDetail,financialData,defaultKeyStatistics,earningsHistory,institutionOwnership",
		url.QueryEscape(symbol))

	client := &http.Client{Timeout: time.Duration(YahooFinanceTimeout) * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", errors.Wrap(err, "failed to create HTTP request")
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return "", errors.Wrap(err, "failed to fetch fundamentals")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errors.Errorf("Yahoo Finance API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Wrap(err, "failed to read response")
	}

	var result struct {
		QuoteSummary struct {
			Result []struct {
				AssetProfile struct {
					CompanyName         string `json:"companyOfficers"`
					Industry            string `json:"industry"`
					Sector              string `json:"sector"`
					Website             string `json:"website"`
					LongBusinessSummary string `json:"longBusinessSummary"`
				} `json:"assetProfile"`
				SummaryDetail struct {
					MarketCap            any `json:"marketCap"`
					TrailingPE           any `json:"trailingPE"`
					ForwardPE            any `json:"forwardPE"`
					PEGRatio             any `json:"pegRatio"`
					PriceToBook          any `json:"priceToBook"`
					DividendYield        any `json:"dividendYield"`
					Beta                 any `json:"beta"`
					FiftyTwoWeekHigh     any `json:"fiftyTwoWeekHigh"`
					FiftyTwoWeekLow      any `json:"fiftyTwoWeekLow"`
					FiftyDayAverage      any `json:"fiftyDayAverage"`
					TwoHundredDayAverage any `json:"twoHundredDayAverage"`
				} `json:"summaryDetail"`
				FinancialData struct {
					TotalRevenue      any `json:"totalRevenue"`
					GrossProfits      any `json:"grossProfits"`
					EBITDA            any `json:"ebitda"`
					NetIncomeToCommon any `json:"netIncomeToCommon"`
					ProfitMargins     any `json:"profitMargins"`
					OperatingMargins  any `json:"operatingMargins"`
					ReturnOnEquity    any `json:"returnOnEquity"`
					ReturnOnAssets    any `json:"returnOnAssets"`
					DebtToEquity      any `json:"debtToEquity"`
					CurrentRatio      any `json:"currentRatio"`
					BookValue         any `json:"bookValue"`
					FreeCashflow      any `json:"freeCashflow"`
					OperatingCashflow any `json:"operatingCashflow"`
					TotalCash         any `json:"totalCash"`
					TotalDebt         any `json:"totalDebt"`
				} `json:"financialData"`
				DefaultKeyStatistics struct {
					TrailingEPS       any `json:"trailingEps"`
					ForwardEPS        any `json:"forwardEps"`
					EnterpriseValue   any `json:"enterpriseValue"`
					SharesOutstanding any `json:"sharesOutstanding"`
					MarketCap         any `json:"marketCap"`
				} `json:"defaultKeyStatistics"`
			} `json:"result"`
			Error any `json:"error"`
		} `json:"quoteSummary"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", errors.Wrap(err, "failed to parse JSON")
	}

	if len(result.QuoteSummary.Result) == 0 {
		return "", errors.New("no fundamental data returned")
	}

	r := result.QuoteSummary.Result[0]

	var output strings.Builder
	output.WriteString(fmt.Sprintf("# Company Fundamentals for %s\n", symbol))
	output.WriteString(fmt.Sprintf("# Data retrieved on: %s\n\n", utils.FormatDateTime(time.Now())))

	if r.AssetProfile.LongBusinessSummary != "" {
		output.WriteString(fmt.Sprintf("# Company Summary:\n%s\n\n", r.AssetProfile.LongBusinessSummary))
	}
	if r.AssetProfile.Sector != "" {
		output.WriteString(fmt.Sprintf("Sector: %s\n", r.AssetProfile.Sector))
	}
	if r.AssetProfile.Industry != "" {
		output.WriteString(fmt.Sprintf("Industry: %s\n", r.AssetProfile.Industry))
	}
	if r.AssetProfile.Website != "" {
		output.WriteString(fmt.Sprintf("Website: %s\n", r.AssetProfile.Website))
	}
	output.WriteString("\n")

	output.WriteString("# Valuation Metrics\n")
	output.WriteString(fmt.Sprintf("Market Cap: %s\n", formatYahooValue(r.SummaryDetail.MarketCap)))
	output.WriteString(fmt.Sprintf("Trailing P/E: %s\n", formatYahooValue(r.SummaryDetail.TrailingPE)))
	output.WriteString(fmt.Sprintf("Forward P/E: %s\n", formatYahooValue(r.SummaryDetail.ForwardPE)))
	output.WriteString(fmt.Sprintf("PEG Ratio: %s\n", formatYahooValue(r.SummaryDetail.PEGRatio)))
	output.WriteString(fmt.Sprintf("Price to Book: %s\n", formatYahooValue(r.SummaryDetail.PriceToBook)))
	output.WriteString(fmt.Sprintf("Dividend Yield: %s\n", formatYahooValue(r.SummaryDetail.DividendYield)))
	output.WriteString("\n")

	output.WriteString("# Per Share Data\n")
	output.WriteString(fmt.Sprintf("Book Value: %s\n", formatYahooValue(r.FinancialData.BookValue)))
	output.WriteString(fmt.Sprintf("Trailing EPS: %s\n", formatYahooValue(r.DefaultKeyStatistics.TrailingEPS)))
	output.WriteString(fmt.Sprintf("Forward EPS: %s\n", formatYahooValue(r.DefaultKeyStatistics.ForwardEPS)))
	output.WriteString("\n")

	output.WriteString("# Financial Highlights\n")
	output.WriteString(fmt.Sprintf("Total Revenue: %s\n", formatYahooValue(r.FinancialData.TotalRevenue)))
	output.WriteString(fmt.Sprintf("Gross Profit: %s\n", formatYahooValue(r.FinancialData.GrossProfits)))
	output.WriteString(fmt.Sprintf("EBITDA: %s\n", formatYahooValue(r.FinancialData.EBITDA)))
	output.WriteString(fmt.Sprintf("Net Income: %s\n", formatYahooValue(r.FinancialData.NetIncomeToCommon)))
	output.WriteString("\n")

	output.WriteString("# Profitability\n")
	output.WriteString(fmt.Sprintf("Profit Margin: %s\n", formatYahooValue(r.FinancialData.ProfitMargins)))
	output.WriteString(fmt.Sprintf("Operating Margin: %s\n", formatYahooValue(r.FinancialData.OperatingMargins)))
	output.WriteString(fmt.Sprintf("Return on Equity: %s\n", formatYahooValue(r.FinancialData.ReturnOnEquity)))
	output.WriteString(fmt.Sprintf("Return on Assets: %s\n", formatYahooValue(r.FinancialData.ReturnOnAssets)))
	output.WriteString("\n")

	output.WriteString("# Liquidity & Solvency\n")
	output.WriteString(fmt.Sprintf("Debt to Equity: %s\n", formatYahooValue(r.FinancialData.DebtToEquity)))
	output.WriteString(fmt.Sprintf("Current Ratio: %s\n", formatYahooValue(r.FinancialData.CurrentRatio)))
	output.WriteString(fmt.Sprintf("Free Cash Flow: %s\n", formatYahooValue(r.FinancialData.FreeCashflow)))
	output.WriteString(fmt.Sprintf("Operating Cash Flow: %s\n", formatYahooValue(r.FinancialData.OperatingCashflow)))
	output.WriteString("\n")

	output.WriteString("# Growth & Momentum\n")
	output.WriteString(fmt.Sprintf("Beta: %s\n", formatYahooValue(r.SummaryDetail.Beta)))
	output.WriteString(fmt.Sprintf("52 Week High: %s\n", formatYahooValue(r.SummaryDetail.FiftyTwoWeekHigh)))
	output.WriteString(fmt.Sprintf("52 Week Low: %s\n", formatYahooValue(r.SummaryDetail.FiftyTwoWeekLow)))
	output.WriteString(fmt.Sprintf("50 Day Average: %s\n", formatYahooValue(r.SummaryDetail.FiftyDayAverage)))
	output.WriteString(fmt.Sprintf("200 Day Average: %s\n", formatYahooValue(r.SummaryDetail.TwoHundredDayAverage)))
	output.WriteString("\n")

	return output.String(), nil
}

func formatYahooValue(v any) string {
	if v == nil {
		return "N/A"
	}
	switch val := v.(type) {
	case float64:
		if val >= 1e12 {
			return fmt.Sprintf("%.2fT", val/1e12)
		} else if val >= 1e9 {
			return fmt.Sprintf("%.2fB", val/1e9)
		} else if val >= 1e6 {
			return fmt.Sprintf("%.2fM", val/1e6)
		} else if val >= 1e3 {
			return fmt.Sprintf("%.2fK", val/1e3)
		} else if val >= 1 {
			return fmt.Sprintf("%.2f", val)
		} else if val == 0 {
			return "0"
		}
		return fmt.Sprintf("%.4f", val)
	case int64:
		if val >= 1e12 {
			return fmt.Sprintf("%.2fT", float64(val)/1e12)
		} else if val >= 1e9 {
			return fmt.Sprintf("%.2fB", float64(val)/1e9)
		} else if val >= 1e6 {
			return fmt.Sprintf("%.2fM", float64(val)/1e6)
		} else if val >= 1e3 {
			return fmt.Sprintf("%.2fK", float64(val)/1e3)
		}
		return fmt.Sprintf("%d", val)
	default:
		return fmt.Sprintf("%v", val)
	}
}

func fallbackFundamentals(ticker, canonical string) string {
	endDt := time.Now()
	startDt := utils.AddDays(endDt, -5)

	period1 := startDt.Unix()
	period2 := endDt.Unix()

	result, err := YahooFinanceRequest(canonical, period1, period2, "1d")
	if err != nil {
		return fmt.Sprintf("# Company Fundamentals for %s\n# Data unavailable: %v\n", canonical, err)
	}

	if len(result.Chart.Result) == 0 {
		return fmt.Sprintf("# Company Fundamentals for %s\n# No data available\n", canonical)
	}

	meta := result.Chart.Result[0].Meta

	var output strings.Builder
	output.WriteString(fmt.Sprintf("# Company Fundamentals for %s\n", canonical))
	output.WriteString(fmt.Sprintf("# Data retrieved on: %s\n\n", utils.FormatDateTime(time.Now())))
	output.WriteString(fmt.Sprintf("Symbol: %s\n", meta.Symbol))
	output.WriteString(fmt.Sprintf("Exchange: %s\n", meta.ExchangeName))
	output.WriteString(fmt.Sprintf("Currency: %s\n", meta.Currency))
	output.WriteString(fmt.Sprintf("Current Price: %.2f\n", meta.RegularMarketPrice))
	output.WriteString("# Note: Detailed fundamental data requires Alpha Vantage API for full coverage.\n")

	return output.String()
}

func GetBalanceSheet(ctx context.Context, args ...any) (string, error) {
	if len(args) < 1 {
		return "", errors.New("get_balance_sheet requires at least 1 argument: ticker")
	}

	ticker := fmt.Sprintf("%v", args[0])
	freq := "quarterly"
	if len(args) > 1 {
		freq = fmt.Sprintf("%v", args[1])
	}

	canonical := utils.NormalizeSymbol(ticker)

	data, err := fetchYahooBalanceSheet(canonical, freq)
	if err != nil {
		logger.L().Warn("Failed to fetch balance sheet", zap.String("symbol", canonical), zap.Error(err))
		return fmt.Sprintf(
			"# Balance Sheet data for %s (%s)\n"+
				"# Data retrieved on: %s\n\n"+
				"# Note: Failed to retrieve balance sheet data: %v\n"+
				"# Symbol: %s\n# Frequency: %s\n",
			canonical, freq, utils.FormatDateTime(time.Now()), err, canonical, freq,
		), nil
	}

	return data, nil
}

func fetchYahooBalanceSheet(symbol, freq string) (string, error) {
	module := "balanceSheetHistoryQuarterly"
	if freq == "annual" {
		module = "balanceSheetHistory"
	}

	url := fmt.Sprintf("https://query1.finance.yahoo.com/v10/finance/quoteSummary/%s?modules=%s",
		url.QueryEscape(symbol), module)

	client := &http.Client{Timeout: time.Duration(YahooFinanceTimeout) * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", errors.Wrap(err, "failed to create HTTP request")
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return "", errors.Wrap(err, "failed to fetch balance sheet")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errors.Errorf("Yahoo Finance API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Wrap(err, "failed to read response")
	}

	var result struct {
		QuoteSummary struct {
			Result []struct {
				BalanceSheetHistory          []YahooBalanceSheet `json:"balanceSheetHistory"`
				BalanceSheetHistoryQuarterly []YahooBalanceSheet `json:"balanceSheetHistoryQuarterly"`
			} `json:"result"`
			Error any `json:"error"`
		} `json:"quoteSummary"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", errors.Wrap(err, "failed to parse JSON")
	}

	if len(result.QuoteSummary.Result) == 0 {
		return "", errors.New("no balance sheet data returned")
	}

	r := result.QuoteSummary.Result[0]
	var statements []YahooBalanceSheet
	if freq == "annual" {
		statements = r.BalanceSheetHistory
	} else {
		statements = r.BalanceSheetHistoryQuarterly
	}

	if len(statements) == 0 {
		return "", errors.New("no balance sheet statements available")
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("# Balance Sheet for %s (%s)\n", symbol, freq))
	output.WriteString(fmt.Sprintf("# Data retrieved on: %s\n\n", utils.FormatDateTime(time.Now())))

	for i, stmt := range statements {
		if i >= 5 {
			break
		}
		output.WriteString(fmt.Sprintf("## %s\n\n", stmt.EndDate))
		output.WriteString("| Item | Value |\n")
		output.WriteString("| --- | ---: |\n")
		output.WriteString(fmt.Sprintf("| Total Assets | %s |\n", formatYahooValue(stmt.TotalAssets)))
		output.WriteString(fmt.Sprintf("| Total Liabilities | %s |\n", formatYahooValue(stmt.TotalLiabilities)))
		output.WriteString(fmt.Sprintf("| Total Equity | %s |\n", formatYahooValue(stmt.TotalEquity)))
		output.WriteString(fmt.Sprintf("| Cash | %s |\n", formatYahooValue(stmt.Cash)))
		output.WriteString(fmt.Sprintf("| Short Term Investments | %s |\n", formatYahooValue(stmt.ShortTermInvestments)))
		output.WriteString(fmt.Sprintf("| Accounts Receivable | %s |\n", formatYahooValue(stmt.AccountsReceivable)))
		output.WriteString(fmt.Sprintf("| Inventory | %s |\n", formatYahooValue(stmt.Inventory)))
		output.WriteString(fmt.Sprintf("| Property Plant Equipment | %s |\n", formatYahooValue(stmt.PropertyPlantEquipment)))
		output.WriteString(fmt.Sprintf("| Goodwill | %s |\n", formatYahooValue(stmt.Goodwill)))
		output.WriteString(fmt.Sprintf("| Current Assets | %s |\n", formatYahooValue(stmt.CurrentAssets)))
		output.WriteString(fmt.Sprintf("| Current Liabilities | %s |\n", formatYahooValue(stmt.CurrentLiabilities)))
		output.WriteString("\n")
	}

	return output.String(), nil
}

type YahooBalanceSheet struct {
	EndDate                string `json:"endDate"`
	TotalAssets            any    `json:"totalAssets"`
	TotalLiabilities       any    `json:"totalLiabilities"`
	TotalEquity            any    `json:"totalEquity"`
	Cash                   any    `json:"cash"`
	ShortTermInvestments   any    `json:"shortTermInvestments"`
	AccountsReceivable     any    `json:"accountsReceivable"`
	Inventory              any    `json:"inventory"`
	PropertyPlantEquipment any    `json:"propertyPlantEquipment"`
	Goodwill               any    `json:"goodwill"`
	CurrentAssets          any    `json:"currentAssets"`
	CurrentLiabilities     any    `json:"currentLiabilities"`
}

func GetCashflow(ctx context.Context, args ...any) (string, error) {
	if len(args) < 1 {
		return "", errors.New("get_cashflow requires at least 1 argument: ticker")
	}

	ticker := fmt.Sprintf("%v", args[0])
	freq := "quarterly"
	if len(args) > 1 {
		freq = fmt.Sprintf("%v", args[1])
	}

	canonical := utils.NormalizeSymbol(ticker)

	data, err := fetchYahooCashflow(canonical, freq)
	if err != nil {
		logger.L().Warn("Failed to fetch cash flow", zap.String("symbol", canonical), zap.Error(err))
		return fmt.Sprintf(
			"# Cash Flow data for %s (%s)\n"+
				"# Data retrieved on: %s\n\n"+
				"# Note: Failed to retrieve cash flow data: %v\n"+
				"# Symbol: %s\n# Frequency: %s\n",
			canonical, freq, utils.FormatDateTime(time.Now()), err, canonical, freq,
		), nil
	}

	return data, nil
}

func fetchYahooCashflow(symbol, freq string) (string, error) {
	module := "cashflowStatementHistoryQuarterly"
	if freq == "annual" {
		module = "cashflowStatementHistory"
	}

	url := fmt.Sprintf("https://query1.finance.yahoo.com/v10/finance/quoteSummary/%s?modules=%s",
		url.QueryEscape(symbol), module)

	client := &http.Client{Timeout: time.Duration(YahooFinanceTimeout) * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", errors.Wrap(err, "failed to create HTTP request")
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return "", errors.Wrap(err, "failed to fetch cash flow")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errors.Errorf("Yahoo Finance API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Wrap(err, "failed to read response")
	}

	var result struct {
		QuoteSummary struct {
			Result []struct {
				CashflowStatementHistory          []YahooCashflow `json:"cashflowStatementHistory"`
				CashflowStatementHistoryQuarterly []YahooCashflow `json:"cashflowStatementHistoryQuarterly"`
			} `json:"result"`
			Error any `json:"error"`
		} `json:"quoteSummary"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", errors.Wrap(err, "failed to parse JSON")
	}

	if len(result.QuoteSummary.Result) == 0 {
		return "", errors.New("no cash flow data returned")
	}

	r := result.QuoteSummary.Result[0]
	var statements []YahooCashflow
	if freq == "annual" {
		statements = r.CashflowStatementHistory
	} else {
		statements = r.CashflowStatementHistoryQuarterly
	}

	if len(statements) == 0 {
		return "", errors.New("no cash flow statements available")
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("# Cash Flow for %s (%s)\n", symbol, freq))
	output.WriteString(fmt.Sprintf("# Data retrieved on: %s\n\n", utils.FormatDateTime(time.Now())))

	for i, stmt := range statements {
		if i >= 5 {
			break
		}
		output.WriteString(fmt.Sprintf("## %s\n\n", stmt.EndDate))
		output.WriteString("| Item | Value |\n")
		output.WriteString("| --- | ---: |\n")
		output.WriteString(fmt.Sprintf("| Operating Cash Flow | %s |\n", formatYahooValue(stmt.OperatingCashflow)))
		output.WriteString(fmt.Sprintf("| Investing Cash Flow | %s |\n", formatYahooValue(stmt.InvestingCashflow)))
		output.WriteString(fmt.Sprintf("| Financing Cash Flow | %s |\n", formatYahooValue(stmt.FinancingCashflow)))
		output.WriteString(fmt.Sprintf("| End Cash Position | %s |\n", formatYahooValue(stmt.EndCashPosition)))
		output.WriteString(fmt.Sprintf("| Free Cash Flow | %s |\n", formatYahooValue(stmt.FreeCashflow)))
		output.WriteString(fmt.Sprintf("| Depreciation | %s |\n", formatYahooValue(stmt.Depreciation)))
		output.WriteString(fmt.Sprintf("| Capital Expenditures | %s |\n", formatYahooValue(stmt.CapitalExpenditures)))
		output.WriteString(fmt.Sprintf("| Dividends Paid | %s |\n", formatYahooValue(stmt.DividendsPaid)))
		output.WriteString("\n")
	}

	return output.String(), nil
}

type YahooCashflow struct {
	EndDate             string `json:"endDate"`
	OperatingCashflow   any    `json:"operatingCashflow"`
	InvestingCashflow   any    `json:"investingCashflow"`
	FinancingCashflow   any    `json:"financingCashflow"`
	EndCashPosition     any    `json:"endCashPosition"`
	FreeCashflow        any    `json:"freeCashflow"`
	Depreciation        any    `json:"depreciation"`
	CapitalExpenditures any    `json:"capitalExpenditures"`
	DividendsPaid       any    `json:"dividendsPaid"`
}

func GetIncomeStatement(ctx context.Context, args ...any) (string, error) {
	if len(args) < 1 {
		return "", errors.New("get_income_statement requires at least 1 argument: ticker")
	}

	ticker := fmt.Sprintf("%v", args[0])
	freq := "quarterly"
	if len(args) > 1 {
		freq = fmt.Sprintf("%v", args[1])
	}

	canonical := utils.NormalizeSymbol(ticker)

	data, err := fetchYahooIncomeStatement(canonical, freq)
	if err != nil {
		logger.L().Warn("Failed to fetch income statement", zap.String("symbol", canonical), zap.Error(err))
		return fmt.Sprintf(
			"# Income Statement data for %s (%s)\n"+
				"# Data retrieved on: %s\n\n"+
				"# Note: Failed to retrieve income statement data: %v\n"+
				"# Symbol: %s\n# Frequency: %s\n",
			canonical, freq, utils.FormatDateTime(time.Now()), err, canonical, freq,
		), nil
	}

	return data, nil
}

func fetchYahooIncomeStatement(symbol, freq string) (string, error) {
	module := "incomeStatementHistoryQuarterly"
	if freq == "annual" {
		module = "incomeStatementHistory"
	}

	url := fmt.Sprintf("https://query1.finance.yahoo.com/v10/finance/quoteSummary/%s?modules=%s",
		url.QueryEscape(symbol), module)

	client := &http.Client{Timeout: time.Duration(YahooFinanceTimeout) * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", errors.Wrap(err, "failed to create HTTP request")
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return "", errors.Wrap(err, "failed to fetch income statement")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errors.Errorf("Yahoo Finance API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Wrap(err, "failed to read response")
	}

	var result struct {
		QuoteSummary struct {
			Result []struct {
				IncomeStatementHistory          []YahooIncomeStatement `json:"incomeStatementHistory"`
				IncomeStatementHistoryQuarterly []YahooIncomeStatement `json:"incomeStatementHistoryQuarterly"`
			} `json:"result"`
			Error any `json:"error"`
		} `json:"quoteSummary"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", errors.Wrap(err, "failed to parse JSON")
	}

	if len(result.QuoteSummary.Result) == 0 {
		return "", errors.New("no income statement data returned")
	}

	r := result.QuoteSummary.Result[0]
	var statements []YahooIncomeStatement
	if freq == "annual" {
		statements = r.IncomeStatementHistory
	} else {
		statements = r.IncomeStatementHistoryQuarterly
	}

	if len(statements) == 0 {
		return "", errors.New("no income statement statements available")
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("# Income Statement for %s (%s)\n", symbol, freq))
	output.WriteString(fmt.Sprintf("# Data retrieved on: %s\n\n", utils.FormatDateTime(time.Now())))

	for i, stmt := range statements {
		if i >= 5 {
			break
		}
		output.WriteString(fmt.Sprintf("## %s\n\n", stmt.EndDate))
		output.WriteString("| Item | Value |\n")
		output.WriteString("| --- | ---: |\n")
		output.WriteString(fmt.Sprintf("| Total Revenue | %s |\n", formatYahooValue(stmt.TotalRevenue)))
		output.WriteString(fmt.Sprintf("| Gross Profit | %s |\n", formatYahooValue(stmt.GrossProfit)))
		output.WriteString(fmt.Sprintf("| Operating Income | %s |\n", formatYahooValue(stmt.OperatingIncome)))
		output.WriteString(fmt.Sprintf("| Net Income | %s |\n", formatYahooValue(stmt.NetIncome)))
		output.WriteString(fmt.Sprintf("| EBITDA | %s |\n", formatYahooValue(stmt.EBITDA)))
		output.WriteString(fmt.Sprintf("| Operating Expenses | %s |\n", formatYahooValue(stmt.OperatingExpenses)))
		output.WriteString(fmt.Sprintf("| Cost of Revenue | %s |\n", formatYahooValue(stmt.CostOfRevenue)))
		output.WriteString(fmt.Sprintf("| Diluted EPS | %s |\n", formatYahooValue(stmt.DilutedEPS)))
		output.WriteString("\n")
	}

	return output.String(), nil
}

type YahooIncomeStatement struct {
	EndDate           string `json:"endDate"`
	TotalRevenue      any    `json:"totalRevenue"`
	GrossProfit       any    `json:"grossProfit"`
	OperatingIncome   any    `json:"operatingIncome"`
	NetIncome         any    `json:"netIncome"`
	EBITDA            any    `json:"ebitda"`
	OperatingExpenses any    `json:"operatingExpenses"`
	CostOfRevenue     any    `json:"costOfRevenue"`
	DilutedEPS        any    `json:"dilutedEPS"`
}

func GetNews(ctx context.Context, args ...any) (string, error) {
	if len(args) < 3 {
		return "", errors.New("get_news requires 3 arguments: ticker, start_date, end_date")
	}

	ticker := fmt.Sprintf("%v", args[0])
	startDate := fmt.Sprintf("%v", args[1])
	endDate := fmt.Sprintf("%v", args[2])

	canonical := utils.NormalizeSymbol(ticker)

	news, err := fetchYahooNews(canonical, startDate, endDate, 20)
	if err != nil {
		logger.L().Warn("Failed to fetch news", zap.String("symbol", canonical), zap.Error(err))
		return fmt.Sprintf("# News for %s from %s to %s\n\n# Note: News data temporarily unavailable.\n# Symbol: %s\n", canonical, startDate, endDate, canonical), nil
	}

	return news, nil
}

func fetchYahooNews(symbol, startDate, endDate string, limit int) (string, error) {
	startDt, err := utils.ParseDate(startDate)
	if err != nil {
		startDt = utils.AddDays(time.Now(), -30)
	}
	endDt, err := utils.ParseDate(endDate)
	if err != nil {
		endDt = time.Now()
	}

	url := fmt.Sprintf("https://query1.finance.yahoo.com/v7/finance/getDetails?symbols=%s&modules=secFilings",
		url.QueryEscape(symbol))

	client := &http.Client{Timeout: time.Duration(YahooFinanceTimeout) * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", errors.Wrap(err, "failed to create HTTP request")
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return "", errors.Wrap(err, "yahoo finance news request failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errors.Errorf("Yahoo Finance API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Wrap(err, "failed to read response body")
	}

	var result struct {
		Finance struct {
			Result []struct {
				Symbol     string `json:"symbol"`
				SecFilings struct {
					Filings []struct {
						Date  string `json:"date"`
						Title string `json:"title"`
						Type  string `json:"type"`
						Link  string `json:"link"`
					} `json:"filings"`
				} `json:"secFilings"`
			} `json:"result"`
		} `json:"finance"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", errors.Wrap(err, "failed to parse JSON response")
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("# News for %s from %s to %s\n\n", symbol, startDate, endDate))

	newsCount := 0
	for _, r := range result.Finance.Result {
		for _, filing := range r.SecFilings.Filings {
			filingDate, _ := utils.ParseDate(filing.Date[:10])
			if filingDate.Before(startDt) || filingDate.After(endDt) {
				continue
			}
			newsCount++
			if newsCount > limit {
				break
			}
			output.WriteString(fmt.Sprintf("## %s\n", filing.Title))
			output.WriteString(fmt.Sprintf("**Date:** %s\n", filing.Date[:10]))
			output.WriteString(fmt.Sprintf("**Type:** %s\n", filing.Type))
			output.WriteString(fmt.Sprintf("**Link:** %s\n\n", filing.Link))
		}
	}

	if newsCount == 0 {
		return fetchYahooNewsViaSearch(symbol, startDate, endDate, limit)
	}

	return output.String(), nil
}

func fetchYahooNewsViaSearch(symbol, startDate, endDate string, limit int) (string, error) {
	var output strings.Builder
	output.WriteString(fmt.Sprintf("# News for %s from %s to %s\n\n", symbol, startDate, endDate))
	output.WriteString(fmt.Sprintf("**Note:** No SEC filings found. Searching general news...\n\n"))
	output.WriteString("| Date | Title |\n| --- | --- |\n")
	return output.String(), nil
}

// GetVerifiedMarketSnapshot 获取验证市场快照（YahooFinance供应商实现）
func GetVerifiedMarketSnapshot(ctx context.Context, symbol string, currDate time.Time, lookBackDays int) (string, error) {
	canonical := utils.NormalizeSymbol(symbol)

	if market_symbols.IsHKSymbol(symbol) {
		plainSymbol, err := market_symbols.ToPlainSymbol(symbol)
		if err == nil {
			canonical = plainSymbol + ".HK"
		}
	}

	endDt := currDate
	startDt := utils.AddDays(endDt, -lookBackDays)

	period1 := startDt.Unix()
	period2 := endDt.AddDate(0, 0, 1).Unix()

	result, err := YahooFinanceRequest(canonical, period1, period2, "1d")
	if err != nil {
		return "", errors.Wrap(err, "yahoo finance request failed")
	}

	series, err := parseYahooChartResult(result)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse yahoo chart result")
	}

	if len(series) == 0 {
		return "", errs.NewNoMarketDataError(symbol, canonical, "no data for market snapshot")
	}

	latest := series[len(series)-1]
	var prev *indicators.OHLCV
	if len(series) > 1 {
		prev = &series[len(series)-2]
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("# Verified Market Snapshot for %s\n\n", canonical))
	output.WriteString(fmt.Sprintf("**As of:** %s\n\n", latest.Date))
	output.WriteString("| Metric | Value |\n")
	output.WriteString("| --- | --- |\n")
	output.WriteString(fmt.Sprintf("| Open | %.2f |\n", latest.Open))
	output.WriteString(fmt.Sprintf("| High | %.2f |\n", latest.High))
	output.WriteString(fmt.Sprintf("| Low | %.2f |\n", latest.Low))
	output.WriteString(fmt.Sprintf("| Close | %.2f |\n", latest.Close))
	output.WriteString(fmt.Sprintf("| Volume | %.0f |\n", latest.Volume))

	if prev != nil {
		change := latest.Close - prev.Close
		changePct := (change / prev.Close) * 100
		output.WriteString(fmt.Sprintf("| Change | %.2f (%.2f%%) |\n", change, changePct))
	}

	if len(series) >= 20 {
		closes := indicators.ExtractClose(series)
		rsi, _ := indicators.RSI(closes, 14)
		if len(rsi) > 0 && !math.IsNaN(rsi[len(rsi)-1]) {
			output.WriteString(fmt.Sprintf("| RSI(14) | %.2f |\n", rsi[len(rsi)-1]))
		}
	}

	return output.String(), nil
}

func init() {
	tool_registry.Registry.RegisterGetStockData("yfinance", GetStockData)
	tool_registry.Registry.RegisterGetIndicators("yfinance", GetStockIndicators)
	tool_registry.Registry.RegisterGetVerifiedMarketSnapshot("yfinance", GetVerifiedMarketSnapshot)
}

type YFinanceProvider struct{}

func (p *YFinanceProvider) Name() string {
	return "yfinance"
}

func (p *YFinanceProvider) Capabilities() []interfaces.Capability {
	return []interfaces.Capability{
		interfaces.CapabilityStockData,
		interfaces.CapabilityIndicators,
		interfaces.CapabilityFundamentals,
		interfaces.CapabilityBalanceSheet,
		interfaces.CapabilityCashflow,
		interfaces.CapabilityIncomeStatement,
		interfaces.CapabilityNews,
		interfaces.CapabilityVerifiedMarketSnapshot,
	}
}

func (p *YFinanceProvider) Markets() []normalizer.MarketType {
	return []normalizer.MarketType{normalizer.MarketUSShare, normalizer.MarketHKShare, normalizer.MarketGlobal}
}

var _ interfaces.Provider = (*YFinanceProvider)(nil)
