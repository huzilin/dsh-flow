package akshare

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-gota/gota/dataframe"
	"github.com/go-gota/gota/series"
	"github.com/huzilin/trade/dataflows/errs"
	"github.com/huzilin/trade/dataflows/indicators"
	"github.com/huzilin/trade/dataflows/interfaces"
	"github.com/huzilin/trade/dataflows/market_symbols"
	"github.com/huzilin/trade/dataflows/models"
	"github.com/huzilin/trade/dataflows/normalizer"
	"github.com/huzilin/trade/dataflows/vendors/tool_registry"
	"github.com/huzilin/trade/dataflows/vendors/utils"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
)

const (
	AkShareTimeout    = 30
	AkShareRetryDelay = 1.0
	AkShareMaxRetries = 3
)

var eastmoneyKlineFields = map[string]string{
	"f51": "Date",
	"f52": "Open",
	"f53": "Close",
	"f54": "High",
	"f55": "Low",
	"f56": "Volume",
	"f57": "Amount",
	"f58": "Amplitude",
	"f59": "PctChange",
	"f60": "ChangeAmount",
	"f61": "TurnoverPct",
}

func getEastmoneyFields2() string {
	fields := make([]string, 0, len(eastmoneyKlineFields))
	for f := range eastmoneyKlineFields {
		fields = append(fields, f)
	}
	return strings.Join(fields, ",")
}

func formatDateForAPI(dateStr string) (string, error) {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return "", err
	}
	return t.Format("20060102"), nil
}

func getPreviousTradeDate(dateStr string) (string, error) {
	target, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return "", err
	}

	prev := target.AddDate(0, 0, -1)
	for prev.After(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)) {
		if isWeekday(prev) && !isHoliday(prev) {
			return prev.Format("2006-01-02"), nil
		}
		prev = prev.AddDate(0, 0, -1)
	}

	return "", errors.Errorf("no trading date found on or before %s", dateStr)
}

func isWeekday(t time.Time) bool {
	day := t.Weekday()
	return day != time.Saturday && day != time.Sunday
}

func isHoliday(t time.Time) bool {
	holidays := map[string]bool{
		"2024-01-01": true, "2024-02-10": true, "2024-02-11": true, "2024-02-12": true,
		"2024-02-13": true, "2024-02-14": true, "2024-04-04": true, "2024-04-05": true,
		"2024-05-01": true, "2024-05-02": true, "2024-05-03": true, "2024-06-10": true,
		"2024-09-15": true, "2024-09-16": true, "2024-09-17": true, "2024-10-01": true,
		"2024-10-02": true, "2024-10-03": true, "2024-10-04": true, "2024-10-05": true,
		"2024-10-06": true, "2024-10-07": true,
		"2025-01-01": true, "2025-01-29": true, "2025-01-30": true, "2025-01-31": true,
		"2025-02-01": true, "2025-02-02": true, "2025-04-04": true, "2025-05-01": true,
		"2025-05-02": true, "2025-05-03": true, "2025-06-09": true, "2025-09-07": true,
		"2025-09-08": true, "2025-09-09": true, "2025-10-01": true, "2025-10-02": true,
		"2025-10-03": true, "2025-10-04": true, "2025-10-05": true, "2025-10-06": true,
		"2025-10-07": true,
	}
	return holidays[t.Format("2006-01-02")]
}

func callAkShareAPI(urlStr string, params map[string]string) ([]byte, error) {
	var fullURL string
	if params == nil {
		fullURL = urlStr
	} else {
		query := url.Values{}
		for k, v := range params {
			query.Set(k, v)
		}
		fullURL = urlStr + "?" + query.Encode()
	}

	cmd := exec.Command("curl", "-s", "--ipv4", "-A", "Mozilla/5.0 (Windows NT 6.3; WOW64; Trident/7.0; Touch; rv:11.0) like Gecko", "-H", "Accept: */*", fullURL)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, errors.Wrapf(err, "curl request failed for URL %s", fullURL)
	}

	return output, nil
}

func loadStockHistWithFallback(symbol, startDate, endDate string) (*dataframe.DataFrame, error) {
	apiStart, err := formatDateForAPI(startDate)
	if err != nil {
		return nil, errors.Wrap(err, "failed to format start date")
	}
	apiEnd, err := formatDateForAPI(endDate)
	if err != nil {
		return nil, errors.Wrap(err, "failed to format end date")
	}

	emUrl := "https://push2his.eastmoney.com/api/qt/stock/kline/get"
	params := map[string]string{
		"secid":   getSecID(symbol),
		"ut":      "fa5fd1943c7b386f172d6893dbfba10b",
		"fields1": "f1,f2,f3,f4,f5,f6,f7,f8,f9,f10,f11,f12,f13",
		"fields2": getEastmoneyFields2(),
		"klt":     "101",
		"fqt":     "1",
		"beg":     apiStart,
		"end":     apiEnd,
		"rtntype": "6",
	}

	body, err := callAkShareAPI(emUrl, params)
	if err != nil {
		return nil, errors.Wrap(err, "eastmoney request failed")
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, errors.Wrap(err, "failed to parse eastmoney response")
	}

	data, ok := result["data"].(map[string]any)
	if !ok {
		return nil, errors.New("no data in response")
	}

	name := ""
	if n, ok := data["name"].(string); ok {
		name = n
	}

	var klineStrings []string
	klineList, ok := data["klines"].([]any)
	if ok && len(klineList) > 0 {
		klineStrings = make([]string, len(klineList))
		for i, v := range klineList {
			klineStrings[i] = fmt.Sprintf("%v", v)
		}
	} else {
		klineStrings, ok = data["klines"].([]string)
		if !ok || len(klineStrings) == 0 {
			return nil, errs.NewNoMarketDataError(symbol, symbol, "no kline data")
		}
	}

	secID := getSecID(symbol)
	code := secID
	if idx := strings.LastIndex(secID, "."); idx >= 0 {
		code = secID[idx+1:]
	}

	return parseEastmoneyKlines(klineStrings, name, code), nil
}

func getSecID(symbol string) string {
	normalized, err := market_symbols.NormalizeAshareSymbol(symbol)
	if err != nil {
		return ""
	}
	parts := strings.SplitN(normalized, ".", 2)
	if len(parts) != 2 {
		return ""
	}
	exchange := parts[1]
	code := parts[0]
	switch exchange {
	case "SH":
		return "1." + code
	case "SZ":
		return "0." + code
	case "BJ":
		return "8." + code
	default:
		return ""
	}
}

func parseEastmoneyKlines(klineStrings []string, name, code string) *dataframe.DataFrame {
	columnNames := []string{"Name", "Code", "Date", "Open", "Close", "High", "Low", "Volume", "Amount", "Amplitude", "PctChange", "ChangeAmount", "TurnoverPct"}
	numCols := len(columnNames)

	var colData [][]string
	for i := 0; i < numCols; i++ {
		colData = append(colData, []string{})
	}

	for _, kline := range klineStrings {
		fields := strings.Split(kline, ",")
		if len(fields) >= 11 {
			colData[0] = append(colData[0], name)
			colData[1] = append(colData[1], code)
			colData[2] = append(colData[2], fields[0])
			colData[3] = append(colData[3], fields[1])
			colData[4] = append(colData[4], fields[2])
			colData[5] = append(colData[5], fields[3])
			colData[6] = append(colData[6], fields[4])
			colData[7] = append(colData[7], fields[5])
			colData[8] = append(colData[8], fields[6])
			colData[9] = append(colData[9], fields[7])
			colData[10] = append(colData[10], fields[8])
			colData[11] = append(colData[11], fields[9])
			colData[12] = append(colData[12], fields[10])
		}
	}

	var seriesList []series.Series
	for i, colName := range columnNames {
		seriesList = append(seriesList, series.New(colData[i], series.String, colName))
	}

	df := dataframe.New(seriesList...)
	return &df
}

// AkShareGetStockData 获取股票K线数据（AKShare供应商实现）
func AkShareGetStockData(ctx context.Context, symbol, startDate, endDate string) ([]*models.Candle, error) {
	normalizedSymbol, err := market_symbols.NormalizeAshareSymbol(symbol)
	if err != nil {
		return nil, errors.Wrap(err, "invalid A-share symbol")
	}

	df, err := loadStockHistWithFallback(symbol, startDate, endDate)
	if err != nil {
		if _, ok := err.(*errs.NoMarketDataError); ok {
			return nil, err
		}
		return nil, errors.Wrap(err, "failed to load stock data")
	}

	if df.Nrow() == 0 {
		return nil, errs.NewNoMarketDataError(symbol, normalizedSymbol, "no data points returned")
	}

	selectedDF := df.Select([]string{"Name", "Code", "Date", "Open", "High", "Low", "Close", "Volume", "Amount", "PctChange", "TurnoverPct"})

	names := selectedDF.Names()
	colIdx := make(map[string]int)
	for i, name := range names {
		colIdx[name] = i
	}

	candles := make([]*models.Candle, 0, selectedDF.Nrow())

	for i := 0; i < selectedDF.Nrow(); i++ {
		candle := &models.Candle{}

		candle.Name = selectedDF.Elem(i, colIdx["Name"]).String()
		candle.Code = selectedDF.Elem(i, colIdx["Code"]).String()
		candle.Date, _ = time.Parse("2006-01-02", selectedDF.Elem(i, colIdx["Date"]).String())

		if v := selectedDF.Elem(i, colIdx["Open"]).String(); v != "" {
			candle.Open, _ = decimal.NewFromString(v)
		}
		if v := selectedDF.Elem(i, colIdx["High"]).String(); v != "" {
			candle.High, _ = decimal.NewFromString(v)
		}
		if v := selectedDF.Elem(i, colIdx["Low"]).String(); v != "" {
			candle.Low, _ = decimal.NewFromString(v)
		}
		if v := selectedDF.Elem(i, colIdx["Close"]).String(); v != "" {
			candle.Close, _ = decimal.NewFromString(v)
		}

		if v := selectedDF.Elem(i, colIdx["Volume"]).String(); v != "" {
			if vol, err := strconv.ParseInt(v, 10, 64); err == nil {
				candle.Volume = vol
			}
		}

		if v := selectedDF.Elem(i, colIdx["Amount"]).String(); v != "" {
			if amt, err := decimal.NewFromString(v); err == nil {
				candle.Amount = &amt
			}
		}

		if v := selectedDF.Elem(i, colIdx["PctChange"]).String(); v != "" {
			if pct, err := decimal.NewFromString(v); err == nil {
				candle.PctChange = &pct
			}
		}

		if v := selectedDF.Elem(i, colIdx["TurnoverPct"]).String(); v != "" {
			if turnover, err := decimal.NewFromString(v); err == nil {
				candle.TurnoverPct = &turnover
			}
		}

		candles = append(candles, candle)
	}

	return candles, nil
}

// AkShareGetIndicators 获取技术指标数据（AKShare供应商实现）
func AkShareGetIndicators(ctx context.Context, symbol, indicator, currDate string, lookBackDays int) (*models.IndicatorResult, error) {
	indicatorName := indicator

	if _, ok := utils.IndicatorDescriptions[indicatorName]; !ok {
		supported := make([]string, 0, len(utils.IndicatorDescriptions))
		for k := range utils.IndicatorDescriptions {
			supported = append(supported, k)
		}
		sort.Strings(supported)
		return nil, errors.Errorf("Indicator %s is not supported for A-share analysis. Choose from: %s", indicatorName, strings.Join(supported, ", "))
	}

	normalizedSymbol, err := market_symbols.NormalizeAshareSymbol(symbol)
	if err != nil {
		return nil, errors.Wrap(err, "invalid A-share symbol")
	}

	alignedTradeDate, err := getPreviousTradeDate(currDate)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get previous trade date")
	}

	endDt, _ := time.Parse("2006-01-02", alignedTradeDate)
	startDt := endDt.AddDate(0, 0, -lookBackDays-100)
	startDate := startDt.Format("2006-01-02")

	df, err := loadStockHistWithFallback(symbol, startDate, alignedTradeDate)
	if err != nil {
		if _, ok := err.(*errs.NoMarketDataError); ok {
			return nil, err
		}
		return nil, err
	}

	series := utils.DataFrameToSeries(df)
	if len(series) == 0 {
		return nil, errs.NewNoMarketDataError(symbol, normalizedSymbol, "no data for indicator calculation")
	}

	closes := indicators.ExtractClose(series)
	var floatValues []float64
	var params map[string]decimal.Decimal

	switch indicatorName {
	case "rsi":
		period := 14
		floatValues, err = indicators.RSI(closes, period)
		if err != nil {
			return nil, errors.Wrap(err, "RSI calculation failed")
		}
		params = map[string]decimal.Decimal{"period": decimal.NewFromInt(int64(period))}

	case "macd":
		macdResult, err := indicators.MACD(closes, 12, 26, 9)
		if err != nil {
			return nil, errors.Wrap(err, "MACD calculation failed")
		}
		floatValues = macdResult.MACD
		params = map[string]decimal.Decimal{"fast": decimal.NewFromInt(12), "slow": decimal.NewFromInt(26), "signal": decimal.NewFromInt(9)}

	case "boll", "boll_ub", "boll_lb":
		bbResult, err := indicators.BollingerBands(closes, 20, 2.0)
		if err != nil {
			return nil, errors.Wrap(err, "Bollinger Bands calculation failed")
		}
		floatValues = bbResult.Middle
		if indicatorName == "boll_ub" {
			floatValues = bbResult.Upper
		} else if indicatorName == "boll_lb" {
			floatValues = bbResult.Lower
		}
		params = map[string]decimal.Decimal{"period": decimal.NewFromInt(20), "std_dev": decimal.NewFromFloat(2.0)}

	case "atr":
		period := 14
		floatValues, err = indicators.ATR(series, period)
		if err != nil {
			return nil, errors.Wrap(err, "ATR calculation failed")
		}
		params = map[string]decimal.Decimal{"period": decimal.NewFromInt(int64(period))}

	case "close_50_sma":
		period := 50
		floatValues, err = indicators.SMA(closes, period)
		if err != nil {
			return nil, errors.Wrap(err, "SMA50 calculation failed")
		}
		params = map[string]decimal.Decimal{"period": decimal.NewFromInt(int64(period))}

	case "close_200_sma":
		period := 200
		floatValues, err = indicators.SMA(closes, period)
		if err != nil {
			return nil, errors.Wrap(err, "SMA200 calculation failed")
		}
		params = map[string]decimal.Decimal{"period": decimal.NewFromInt(int64(period))}

	case "close_10_ema":
		period := 10
		floatValues, err = indicators.EMA(closes, period)
		if err != nil {
			return nil, errors.Wrap(err, "EMA10 calculation failed")
		}
		params = map[string]decimal.Decimal{"period": decimal.NewFromInt(int64(period))}

	case "vwma":
		period := 20
		floatValues, err = indicators.VWMA(series, period)
		if err != nil {
			return nil, errors.Wrap(err, "VWMA calculation failed")
		}
		params = map[string]decimal.Decimal{"period": decimal.NewFromInt(int64(period))}

	case "mfi":
		period := 14
		floatValues, err = indicators.MFI(series, period)
		if err != nil {
			return nil, errors.Wrap(err, "MFI calculation failed")
		}
		params = map[string]decimal.Decimal{"period": decimal.NewFromInt(int64(period))}

	default:
		return nil, errors.Errorf("unsupported indicator: %s", indicatorName)
	}

	indicatorValues := make([]*models.IndicatorValue, 0, len(floatValues))
	startIdx := len(series) - len(floatValues)
	for i, v := range floatValues {
		if i >= lookBackDays {
			break
		}
		idx := startIdx + len(floatValues) - lookBackDays + i
		if idx >= 0 && idx < len(series) && !math.IsNaN(v) {
			date, _ := time.Parse("2006-01-02", series[idx].Date)
			indicatorValues = append(indicatorValues, &models.IndicatorValue{
				Date:  date,
				Value: decimal.NewFromFloat(v),
			})
		}
	}

	return &models.IndicatorResult{
		Symbol:    normalizedSymbol,
		Indicator: indicatorName,
		Values:    indicatorValues,
		Params:    params,
		Source:    "akshare",
	}, nil
}

// AkShareGetFundamentals 获取基本面概况（AKShare供应商实现）
func AkShareGetFundamentals(ctx context.Context, symbol string, currDate time.Time) (*models.Fundamentals, error) {
	normalizedSymbol, err := market_symbols.NormalizeAshareSymbol(symbol)
	if err != nil {
		return nil, errors.Wrap(err, "invalid A-share symbol")
	}

	plainSymbol, err := market_symbols.ToPlainSymbol(symbol)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get plain symbol")
	}

	result := &models.Fundamentals{
		Symbol:    normalizedSymbol,
		Source:    "akshare",
		Timestamp: time.Now(),
	}

	infoData, err := fetchCompanyProfile(plainSymbol)
	if err == nil {
		result.CompanyInfo = &models.CompanyInfo{}
		for _, row := range infoData {
			if name, ok := row["指标"].(string); ok {
				switch name {
				case "公司名称":
					result.CompanyInfo.Name = fmt.Sprintf("%v", row["数值"])
				case "所属行业":
					result.CompanyInfo.Industry = fmt.Sprintf("%v", row["数值"])
				case "所属板块":
					result.CompanyInfo.Sector = fmt.Sprintf("%v", row["数值"])
				case "注册资本":
					if val, ok := row["数值"].(string); ok {
						if dec, err := decimal.NewFromString(val); err == nil {
							result.CompanyInfo.MarketCap = dec
						}
					}
				case "公司简介":
					result.CompanyInfo.Description = fmt.Sprintf("%v", row["数值"])
				}
			}
		}
	}

	introData, err := fetchBusinessIntro(plainSymbol)
	if err == nil {
		result.BusinessIntro = &models.BusinessIntro{}
		for _, row := range introData {
			if main, ok := row["主营业务"].(string); ok {
				result.BusinessIntro.MainBusiness = main
			}
			if scope, ok := row["经营范围"].(string); ok {
				result.BusinessIntro.Scope = scope
			}
		}
	}

	businessData, err := fetchBusinessComposition(plainSymbol)
	if err == nil {
		for _, row := range businessData {
			category := fmt.Sprintf("%v", row["分类"])
			revenueStr := fmt.Sprintf("%v", row["营业收入"])
			proportionStr := fmt.Sprintf("%v", row["收入占比(%)"])

			revenue, _ := decimal.NewFromString(revenueStr)
			proportion, _ := decimal.NewFromString(proportionStr)

			result.Composition = append(result.Composition, &models.BusinessComposition{
				Category:   category,
				Revenue:    revenue,
				Proportion: proportion,
			})
		}
	}

	return result, nil
}

func fetchCompanyProfile(plainSymbol string) ([]map[string]any, error) {
	url := fmt.Sprintf("https://emweb.securities.eastmoney.com/PC_HSF10/CompanySurvey/CompanySurveyAjaxNew?type=web&code=%s", plainSymbol)

	body, err := callAkShareAPI(url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch company profile")
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, errors.Wrap(err, "failed to parse company profile")
	}

	data, ok := result["data"].(map[string]any)
	if !ok {
		return nil, errors.New("no data in company profile response")
	}

	var rows []map[string]any
	for k, v := range data {
		rows = append(rows, map[string]any{
			"指标": k,
			"数值": fmt.Sprintf("%v", v),
		})
	}
	return rows, nil
}

func fetchBusinessIntro(plainSymbol string) ([]map[string]any, error) {
	url := fmt.Sprintf("https://emweb.securities.eastmoney.com/PC_HSF10/BusinessAnalysis/BusinessAnalysisAjaxNew?type=web&code=%s", plainSymbol)

	body, err := callAkShareAPI(url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch business intro")
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, errors.Wrap(err, "failed to parse business intro")
	}

	data, ok := result["data"].(map[string]any)
	if !ok {
		return nil, errors.New("no data in business intro response")
	}

	var rows []map[string]any
	rows = append(rows, map[string]any{
		"主营业务": fmt.Sprintf("%v", data["businessScope"]),
		"经营范围": fmt.Sprintf("%v", data["businessDesc"]),
	})
	return rows, nil
}

func fetchBusinessComposition(plainSymbol string) ([]map[string]any, error) {
	url := fmt.Sprintf("https://emweb.securities.eastmoney.com/PC_HSF10/BusinessAnalysis/BusinessAnalysisAjaxNew?type=web&code=%s", plainSymbol)

	body, err := callAkShareAPI(url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch business composition")
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, errors.Wrap(err, "failed to parse business composition")
	}

	data, ok := result["data"].(map[string]any)
	if !ok {
		return nil, errors.New("no data in business composition response")
	}

	var rows []map[string]any

	if industryData, ok := data["industryData"].([]any); ok && len(industryData) > 0 {
		for _, item := range industryData[:utils.Min(len(industryData), 5)] {
			if row, ok := item.(map[string]any); ok {
				rows = append(rows, map[string]any{
					"分类":      fmt.Sprintf("%v", row["industryName"]),
					"营业收入":    fmt.Sprintf("%v", row["amount"]),
					"收入占比(%)": fmt.Sprintf("%v", row["proportion"]),
				})
			}
		}
	}

	if productData, ok := data["productData"].([]any); ok && len(productData) > 0 && len(rows) == 0 {
		for _, item := range productData[:utils.Min(len(productData), 5)] {
			if row, ok := item.(map[string]any); ok {
				rows = append(rows, map[string]any{
					"分类":      fmt.Sprintf("%v", row["productName"]),
					"营业收入":    fmt.Sprintf("%v", row["amount"]),
					"收入占比(%)": fmt.Sprintf("%v", row["proportion"]),
				})
			}
		}
	}

	if areaData, ok := data["areaData"].([]any); ok && len(areaData) > 0 && len(rows) == 0 {
		for _, item := range areaData[:utils.Min(len(areaData), 5)] {
			if row, ok := item.(map[string]any); ok {
				rows = append(rows, map[string]any{
					"分类":      fmt.Sprintf("%v", row["areaName"]),
					"营业收入":    fmt.Sprintf("%v", row["amount"]),
					"收入占比(%)": fmt.Sprintf("%v", row["proportion"]),
				})
			}
		}
	}

	if len(rows) == 0 {
		return nil, errors.New("no business composition data available")
	}

	return rows, nil
}

func fetchFinancialAbstract(plainSymbol string) ([]map[string]any, error) {
	url := fmt.Sprintf("https://datacenter-web.eastmoney.com/api/data/v1/get?reportName=RPT_DMSK_FN_CASHFLOW&columns=SECURITY_CODE,SECURITY_NAME,REPORT_DATE,TOTAL_OPERATE_INCOME,TOTAL_PROFIT,NETPROFIT,PARENT_NETPROFIT,BASIC_EPS&filter=(SECURITY_CODE=\"%s\")&pageSize=5&sortColumns=REPORT_DATE&sortTypes=-1", plainSymbol)

	body, err := callAkShareAPI(url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch financial abstract")
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, errors.Wrap(err, "failed to parse financial abstract")
	}

	resultData, ok := result["result"].(map[string]any)
	if !ok {
		return nil, errors.New("no result in financial abstract response")
	}

	data, ok := resultData["data"].([]any)
	if !ok || len(data) == 0 {
		return nil, errors.New("no data in financial abstract")
	}

	var rows []map[string]any
	for _, item := range data[:3] {
		if row, ok := item.(map[string]any); ok {
			rows = append(rows, map[string]any{
				"报告日期":   fmt.Sprintf("%v", row["REPORT_DATE"]),
				"营业总收入":  fmt.Sprintf("%v", row["TOTAL_OPERATE_INCOME"]),
				"利润总额":   fmt.Sprintf("%v", row["TOTAL_PROFIT"]),
				"净利润":    fmt.Sprintf("%v", row["NETPROFIT"]),
				"归母净利润":  fmt.Sprintf("%v", row["PARENT_NETPROFIT"]),
				"基本每股收益": fmt.Sprintf("%v", row["BASIC_EPS"]),
			})
		}
	}
	return rows, nil
}

// AkShareGetBalanceSheet 获取资产负债表（AKShare供应商实现）
func AkShareGetBalanceSheet(ctx context.Context, symbol, freq string, currDate time.Time) (*models.BalanceSheet, error) {
	normalizedSymbol, err := market_symbols.NormalizeAshareSymbol(symbol)
	if err != nil {
		return nil, errors.Wrap(err, "invalid A-share symbol")
	}

	plainSymbol, err := market_symbols.ToPlainSymbol(symbol)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get plain symbol")
	}

	data, err := fetchBalanceSheet(plainSymbol, freq)
	if err != nil {
		return nil, err
	}

	result := &models.BalanceSheet{
		Symbol:    normalizedSymbol,
		Frequency: freq,
		Source:    "akshare",
	}

	for _, row := range data {
		dateStr := fmt.Sprintf("%v", row["报告日期"])
		date, _ := time.Parse("2006-01-02", dateStr)

		totalAssets, _ := decimal.NewFromString(fmt.Sprintf("%v", row["总资产"]))
		totalLiabilities, _ := decimal.NewFromString(fmt.Sprintf("%v", row["总负债"]))
		totalEquity, _ := decimal.NewFromString(fmt.Sprintf("%v", row["归属于母公司股东权益"]))
		cash, _ := decimal.NewFromString(fmt.Sprintf("%v", row["货币资金"]))
		currentAssets, _ := decimal.NewFromString(fmt.Sprintf("%v", row["存货"]))
		currentLiabilities, _ := decimal.NewFromString(fmt.Sprintf("%v", row["应收账款"]))

		result.Items = append(result.Items, &models.BalanceSheetItem{
			Date:               date,
			TotalAssets:        totalAssets,
			TotalLiabilities:   totalLiabilities,
			TotalEquity:        totalEquity,
			Cash:               cash,
			CurrentAssets:      currentAssets,
			CurrentLiabilities: currentLiabilities,
		})
	}

	return result, nil
}

func fetchBalanceSheet(plainSymbol, freq string) ([]map[string]any, error) {
	url := fmt.Sprintf("https://datacenter-web.eastmoney.com/api/data/v1/get?reportName=RPT_DMSK_FN_BALANCE&columns=REPORT_DATE,TOTAL_ASSETS,TOTAL_LIABILITIES,TOTAL_PARENT_EQUITY,MONETARYFUNDS,INVENTORY,ACCOUNTS_RECE,GOODWILL&filter=(SECURITY_CODE=\"%s\")&pageSize=5&sortColumns=REPORT_DATE&sortTypes=-1", plainSymbol)

	body, err := callAkShareAPI(url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch balance sheet")
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, errors.Wrap(err, "failed to parse balance sheet")
	}

	resultData, ok := result["result"].(map[string]any)
	if !ok {
		return nil, errors.New("no result in balance sheet response")
	}

	data, ok := resultData["data"].([]any)
	if !ok || len(data) == 0 {
		return nil, errors.New("no data in balance sheet")
	}

	var rows []map[string]any
	for _, item := range data[:5] {
		if row, ok := item.(map[string]any); ok {
			rows = append(rows, map[string]any{
				"报告日期":       fmt.Sprintf("%v", row["REPORT_DATE"]),
				"总资产":        fmt.Sprintf("%v", row["TOTAL_ASSETS"]),
				"总负债":        fmt.Sprintf("%v", row["TOTAL_LIABILITIES"]),
				"归属于母公司股东权益": fmt.Sprintf("%v", row["TOTAL_PARENT_EQUITY"]),
				"货币资金":       fmt.Sprintf("%v", row["MONETARYFUNDS"]),
				"存货":         fmt.Sprintf("%v", row["INVENTORY"]),
				"应收账款":       fmt.Sprintf("%v", row["ACCOUNTS_RECE"]),
				"商誉":         fmt.Sprintf("%v", row["GOODWILL"]),
			})
		}
	}
	return rows, nil
}

// AkShareGetCashflow 获取现金流量表（AKShare供应商实现）
func AkShareGetCashflow(ctx context.Context, symbol, freq string, currDate time.Time) (*models.Cashflow, error) {
	normalizedSymbol, err := market_symbols.NormalizeAshareSymbol(symbol)
	if err != nil {
		return nil, errors.Wrap(err, "invalid A-share symbol")
	}

	plainSymbol, err := market_symbols.ToPlainSymbol(symbol)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get plain symbol")
	}

	data, err := fetchCashflow(plainSymbol, freq)
	if err != nil {
		return nil, err
	}

	result := &models.Cashflow{
		Symbol:    normalizedSymbol,
		Frequency: freq,
		Source:    "akshare",
	}

	for _, row := range data {
		dateStr := fmt.Sprintf("%v", row["报告日期"])
		date, _ := time.Parse("2006-01-02", dateStr)

		operating, _ := decimal.NewFromString(fmt.Sprintf("%v", row["经营活动产生的现金流量净额"]))
		investing, _ := decimal.NewFromString(fmt.Sprintf("%v", row["投资活动产生的现金流量净额"]))
		financing, _ := decimal.NewFromString(fmt.Sprintf("%v", row["筹资活动产生的现金流量净额"]))
		freeCashflow, _ := decimal.NewFromString(fmt.Sprintf("%v", row["现金及现金等价物净增加额"]))

		result.Items = append(result.Items, &models.CashflowItem{
			Date:              date,
			OperatingCashflow: operating,
			InvestingCashflow: investing,
			FinancingCashflow: financing,
			FreeCashflow:      freeCashflow,
		})
	}

	return result, nil
}

func fetchCashflow(plainSymbol, freq string) ([]map[string]any, error) {
	url := fmt.Sprintf("https://datacenter-web.eastmoney.com/api/data/v1/get?reportName=RPT_DMSK_FN_CASHFLOW&columns=REPORT_DATE,NETCASH_OPERATE,NETCASH_INVEST,NETCASH_FINANCE,CCE_ADD,PAY_STAFF_CASH,PAY_ALL_TAX&filter=(SECURITY_CODE=\"%s\")&pageSize=5&sortColumns=REPORT_DATE&sortTypes=-1", plainSymbol)

	body, err := callAkShareAPI(url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch cashflow")
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, errors.Wrap(err, "failed to parse cashflow")
	}

	resultData, ok := result["result"].(map[string]any)
	if !ok {
		return nil, errors.New("no result in cashflow response")
	}

	data, ok := resultData["data"].([]any)
	if !ok || len(data) == 0 {
		return nil, errors.New("no data in cashflow")
	}

	var rows []map[string]any
	for _, item := range data[:5] {
		if row, ok := item.(map[string]any); ok {
			rows = append(rows, map[string]any{
				"报告日期": fmt.Sprintf("%v", row["REPORT_DATE"]),
				"经营活动产生的现金流量净额":   fmt.Sprintf("%v", row["NETCASH_OPERATE"]),
				"投资活动产生的现金流量净额":   fmt.Sprintf("%v", row["NETCASH_INVEST"]),
				"筹资活动产生的现金流量净额":   fmt.Sprintf("%v", row["NETCASH_FINANCE"]),
				"现金及现金等价物净增加额":    fmt.Sprintf("%v", row["CCE_ADD"]),
				"支付给职工以及为职工支付的现金": fmt.Sprintf("%v", row["PAY_STAFF_CASH"]),
				"支付的各项税费":         fmt.Sprintf("%v", row["PAY_ALL_TAX"]),
			})
		}
	}
	return rows, nil
}

// AkShareGetIncomeStatement 获取损益表（AKShare供应商实现）
func AkShareGetIncomeStatement(ctx context.Context, symbol, freq string, currDate time.Time) (*models.IncomeStatement, error) {
	normalizedSymbol, err := market_symbols.NormalizeAshareSymbol(symbol)
	if err != nil {
		return nil, errors.Wrap(err, "invalid A-share symbol")
	}

	plainSymbol, err := market_symbols.ToPlainSymbol(symbol)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get plain symbol")
	}

	data, err := fetchIncomeStatement(plainSymbol, freq)
	if err != nil {
		return nil, err
	}

	result := &models.IncomeStatement{
		Symbol:    normalizedSymbol,
		Frequency: freq,
		Source:    "akshare",
	}

	for _, row := range data {
		dateStr := fmt.Sprintf("%v", row["报告日期"])
		date, _ := time.Parse("2006-01-02", dateStr)

		revenue, _ := decimal.NewFromString(fmt.Sprintf("%v", row["营业总收入"]))
		grossProfit, _ := decimal.NewFromString(fmt.Sprintf("%v", row["营业利润"]))
		operatingIncome, _ := decimal.NewFromString(fmt.Sprintf("%v", row["营业利润"]))
		netIncome, _ := decimal.NewFromString(fmt.Sprintf("%v", row["净利润"]))
		eps, _ := decimal.NewFromString(fmt.Sprintf("%v", row["基本每股收益"]))

		result.Items = append(result.Items, &models.IncomeStatementItem{
			Date:            date,
			Revenue:         revenue,
			GrossProfit:     grossProfit,
			OperatingIncome: operatingIncome,
			NetIncome:       netIncome,
			EPS:             eps,
		})
	}

	return result, nil
}

func fetchIncomeStatement(plainSymbol, freq string) ([]map[string]any, error) {
	url := fmt.Sprintf("https://datacenter-web.eastmoney.com/api/data/v1/get?reportName=RPT_DMSK_FN_INCOME&columns=REPORT_DATE,TOTAL_OPERATE_INCOME,OPERATE_PROFIT,TOTAL_PROFIT,NETPROFIT,PARENT_NETPROFIT,DEDUCT_PARENT_NETPROFIT,BASIC_EPS&filter=(SECURITY_CODE=\"%s\")&pageSize=5&sortColumns=REPORT_DATE&sortTypes=-1", plainSymbol)

	body, err := callAkShareAPI(url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch income statement")
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, errors.Wrap(err, "failed to parse income statement")
	}

	resultData, ok := result["result"].(map[string]any)
	if !ok {
		return nil, errors.New("no result in income statement response")
	}

	data, ok := resultData["data"].([]any)
	if !ok || len(data) == 0 {
		return nil, errors.New("no data in income statement")
	}

	var rows []map[string]any
	for _, item := range data[:5] {
		if row, ok := item.(map[string]any); ok {
			rows = append(rows, map[string]any{
				"报告日期":  fmt.Sprintf("%v", row["REPORT_DATE"]),
				"营业总收入": fmt.Sprintf("%v", row["TOTAL_OPERATE_INCOME"]),
				"营业利润":  fmt.Sprintf("%v", row["OPERATE_PROFIT"]),
				"利润总额":  fmt.Sprintf("%v", row["TOTAL_PROFIT"]),
				"净利润":   fmt.Sprintf("%v", row["NETPROFIT"]),
				"归属于母公司所有者的净利润": fmt.Sprintf("%v", row["PARENT_NETPROFIT"]),
				"扣除非经常性损益后的净利润": fmt.Sprintf("%v", row["DEDUCT_PARENT_NETPROFIT"]),
				"基本每股收益":        fmt.Sprintf("%v", row["BASIC_EPS"]),
			})
		}
	}
	return rows, nil
}

// AkShareGetNews 获取新闻（AKShare供应商实现）
// symbol为空时获取市场新闻，不为空时获取个股新闻
func AkShareGetNews(ctx context.Context, symbol string, startDate, endDate time.Time, limit int) (*models.NewsResult, error) {
	if symbol == "" {
		return AkShareGetMarketNews(ctx, endDate, int(endDate.Sub(startDate).Hours()/24), limit)
	}

	normalizedSymbol, err := market_symbols.NormalizeAshareSymbol(symbol)
	if err != nil {
		return nil, errors.Wrap(err, "invalid A-share symbol")
	}

	plainSymbol, err := market_symbols.ToPlainSymbol(symbol)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get plain symbol")
	}

	data, err := fetchStockNews(plainSymbol)
	if err != nil {
		return nil, err
	}

	filtered := utils.FilterNewsByDate(data, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
	if len(filtered) > limit {
		filtered = filtered[:limit]
	}

	result := &models.NewsResult{
		Symbol: normalizedSymbol,
		Source: "akshare",
		News:   make([]*models.NewsItem, 0, len(filtered)),
	}

	for _, row := range filtered {
		publishTimeStr := fmt.Sprintf("%v", row["发布时间"])
		publishDate, _ := time.Parse("2006-01-02 15:04:05", publishTimeStr)

		result.News = append(result.News, &models.NewsItem{
			Title:       fmt.Sprintf("%v", row["新闻标题"]),
			Source:      fmt.Sprintf("%v", row["文章来源"]),
			Summary:     fmt.Sprintf("%v", row["新闻内容"]),
			URL:         fmt.Sprintf("%v", row["新闻链接"]),
			PublishDate: publishDate,
		})
	}

	return result, nil
}

func fetchStockNews(plainSymbol string) ([]map[string]any, error) {
	url := fmt.Sprintf("https://searchapi.eastmoney.com/bkds/api/search/v1/news?keyword=%s&pageIndex=1&pageSize=30&sortName=&sortType=0&searchFields=title,content", plainSymbol)

	body, err := callAkShareAPI(url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch stock news")
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, errors.Wrap(err, "failed to parse stock news")
	}

	data, ok := result["data"].([]any)
	if !ok {
		return nil, errors.New("no data in stock news response")
	}

	var rows []map[string]any
	for _, item := range data {
		if row, ok := item.(map[string]any); ok {
			rows = append(rows, map[string]any{
				"发布时间": fmt.Sprintf("%v", row["publishTime"]),
				"文章来源": fmt.Sprintf("%v", row["source"]),
				"新闻标题": fmt.Sprintf("%v", row["title"]),
				"新闻内容": utils.SafeTruncate(fmt.Sprintf("%v", row["content"]), 160),
				"新闻链接": fmt.Sprintf("%v", row["url"]),
			})
		}
	}
	return rows, nil
}

// AkShareGetMarketNews 获取市场新闻（AKShare供应商实现）
func AkShareGetMarketNews(ctx context.Context, currDate time.Time, lookBackDays, limit int) (*models.NewsResult, error) {
	data, err := fetchMarketNews()
	if err != nil {
		return nil, err
	}

	start := currDate.AddDate(0, 0, -lookBackDays)

	filtered := utils.FilterNewsByDate(data, start.Format("2006-01-02"), currDate.Format("2006-01-02"))
	if len(filtered) > limit {
		filtered = filtered[:limit]
	}

	result := &models.NewsResult{
		Source: "akshare",
		News:   make([]*models.NewsItem, 0, len(filtered)),
	}

	for _, row := range filtered {
		publishTimeStr := fmt.Sprintf("%v", row["发布时间"])
		publishDate, _ := time.Parse("2006-01-02 15:04:05", publishTimeStr)

		result.News = append(result.News, &models.NewsItem{
			Title:       fmt.Sprintf("%v", row["标题"]),
			Source:      "",
			Summary:     fmt.Sprintf("%v", row["摘要"]),
			URL:         fmt.Sprintf("%v", row["链接"]),
			PublishDate: publishDate,
		})
	}

	return result, nil
}

func fetchMarketNews() ([]map[string]any, error) {
	url := "https://www.eastmoney.com/api/guonei/news?pagesize=30&pageindex=1"

	body, err := callAkShareAPI(url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch market news")
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, errors.Wrap(err, "failed to parse market news")
	}

	data, ok := result["data"].([]any)
	if !ok {
		return nil, errors.New("no data in market news response")
	}

	var rows []map[string]any
	for _, item := range data {
		if row, ok := item.(map[string]any); ok {
			rows = append(rows, map[string]any{
				"发布时间": fmt.Sprintf("%v", row["publishTime"]),
				"标题":   fmt.Sprintf("%v", row["title"]),
				"摘要":   utils.SafeTruncate(fmt.Sprintf("%v", row["summary"]), 180),
				"链接":   fmt.Sprintf("%v", row["url"]),
			})
		}
	}
	return rows, nil
}

// AkShareGetCompanyAnnouncements 获取公司公告（AKShare供应商实现）
func AkShareGetCompanyAnnouncements(ctx context.Context, symbol string, startDate, endDate time.Time) (*models.AnnouncementResult, error) {
	normalizedSymbol, err := market_symbols.NormalizeAshareSymbol(symbol)
	if err != nil {
		return nil, errors.Wrap(err, "invalid A-share symbol")
	}

	plainSymbol, err := market_symbols.ToPlainSymbol(symbol)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get plain symbol")
	}

	data, err := fetchCompanyAnnouncements(plainSymbol, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
	if err != nil {
		return nil, err
	}

	if len(data) > 20 {
		data = data[:20]
	}

	result := &models.AnnouncementResult{
		Symbol:        normalizedSymbol,
		Source:        "akshare",
		Announcements: make([]*models.AnnouncementItem, 0, len(data)),
	}

	for _, row := range data {
		publishTimeStr := fmt.Sprintf("%v", row["发布时间"])
		publishDate, _ := time.Parse("2006-01-02 15:04:05", publishTimeStr)

		result.Announcements = append(result.Announcements, &models.AnnouncementItem{
			Date:  publishDate,
			Title: fmt.Sprintf("%v", row["公告标题"]),
			Type:  "",
			URL:   fmt.Sprintf("%v", row["公告链接"]),
		})
	}

	return result, nil
}

func fetchCompanyAnnouncements(plainSymbol, startDate, endDate string) ([]map[string]any, error) {
	apiStart, err := formatDateForAPI(startDate)
	if err != nil {
		return nil, errors.Wrap(err, "failed to format start date")
	}
	apiEnd, err := formatDateForAPI(endDate)
	if err != nil {
		return nil, errors.Wrap(err, "failed to format end date")
	}

	url := fmt.Sprintf("https://api.eastmoney.com/bkds/api/search/v1/announcement?keyword=%s&pageIndex=1&pageSize=30&startTime=%s&endTime=%s",
		plainSymbol, apiStart, apiEnd)

	body, err := callAkShareAPI(url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch company announcements")
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, errors.Wrap(err, "failed to parse company announcements")
	}

	data, ok := result["data"].([]any)
	if !ok {
		return nil, errors.New("no data in announcements response")
	}

	var rows []map[string]any
	for _, item := range data {
		if row, ok := item.(map[string]any); ok {
			rows = append(rows, map[string]any{
				"公告日期": fmt.Sprintf("%v", row["publishTime"]),
				"公告类型": fmt.Sprintf("%v", row["type"]),
				"公告标题": fmt.Sprintf("%v", row["title"]),
				"网址":   fmt.Sprintf("%v", row["url"]),
			})
		}
	}

	sort.Slice(rows, func(i, j int) bool {
		return fmt.Sprintf("%v", rows[i]["公告日期"]) > fmt.Sprintf("%v", rows[j]["公告日期"])
	})

	return rows, nil
}

func init() {
	tool_registry.Registry.RegisterGetStockData("akshare", AkShareGetStockData)
	tool_registry.Registry.RegisterGetIndicators("akshare", AkShareGetIndicators)
	tool_registry.Registry.RegisterGetFundamentals("akshare", AkShareGetFundamentals)
	tool_registry.Registry.RegisterGetBalanceSheet("akshare", AkShareGetBalanceSheet)
	tool_registry.Registry.RegisterGetCashflow("akshare", AkShareGetCashflow)
	tool_registry.Registry.RegisterGetIncomeStatement("akshare", AkShareGetIncomeStatement)
	tool_registry.Registry.RegisterGetNews("akshare", AkShareGetNews)
	tool_registry.Registry.RegisterGetCompanyAnnouncements("akshare", AkShareGetCompanyAnnouncements)
}

type AkShareProvider struct{}

func (p *AkShareProvider) Name() string {
	return "akshare"
}

func (p *AkShareProvider) Capabilities() []interfaces.Capability {
	return []interfaces.Capability{
		interfaces.CapabilityStockData,
		interfaces.CapabilityIndicators,
		interfaces.CapabilityFundamentals,
		interfaces.CapabilityBalanceSheet,
		interfaces.CapabilityCashflow,
		interfaces.CapabilityIncomeStatement,
		interfaces.CapabilityNews,
		interfaces.CapabilityCompanyAnnouncements,
	}
}

func (p *AkShareProvider) Markets() []normalizer.MarketType {
	return []normalizer.MarketType{normalizer.MarketAshare}
}
