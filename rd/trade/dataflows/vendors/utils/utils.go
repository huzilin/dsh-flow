package utils

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-gota/gota/dataframe"
	"github.com/huzilin/trade/dataflows/indicators"
)

func NormalizeSymbol(symbol string) string {
	symbol = strings.TrimSpace(strings.ToUpper(symbol))

	forexPatterns := map[string]string{
		"XAUUSD":  "GC=F",
		"XAUUSD+": "GC=F",
		"GOLD":    "GC=F",
		"XAGUSD":  "SI=F",
		"SILVER":  "SI=F",
	}

	if normalized, ok := forexPatterns[symbol]; ok {
		return normalized
	}

	if strings.HasPrefix(symbol, "BTC") {
		return "BTC-USD"
	}
	if strings.HasPrefix(symbol, "ETH") {
		return "ETH-USD"
	}

	return symbol
}

func FormatDate(t time.Time) string {
	return t.Format("2006-01-02")
}

func FormatDateTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

func ParseDate(dateStr string) (time.Time, error) {
	return time.Parse("2006-01-02", dateStr)
}

func ValidateSymbol(symbol string) bool {
	if symbol == "" {
		return false
	}
	matched, _ := regexp.MatchString(`^[A-Z]{1,5}$`, strings.ToUpper(symbol))
	return matched
}

func FormatLargeNumber(num float64) string {
	if num >= 1e12 {
		return fmt.Sprintf("%.2fT", num/1e12)
	}
	if num >= 1e9 {
		return fmt.Sprintf("%.2fB", num/1e9)
	}
	if num >= 1e6 {
		return fmt.Sprintf("%.2fM", num/1e6)
	}
	if num >= 1e3 {
		return fmt.Sprintf("%.2fK", num/1e3)
	}
	return fmt.Sprintf("%.2f", num)
}

func AddDays(date time.Time, days int) time.Time {
	return date.AddDate(0, 0, days)
}

var IndicatorDescriptions = map[string]string{
	"close_50_sma":  "50日简单移动平均线，用于识别中期趋势和动态支撑阻力。",
	"close_200_sma": "200日简单移动平均线，用于识别长期趋势和牛熊切换。",
	"close_10_ema":  "10日指数移动平均线，用于捕捉更快的短期趋势变化。",
	"macd":          "MACD 指标，用于识别趋势变化与动量。",
	"macds":         "MACD 信号线，用于配合 MACD 判断金叉死叉。",
	"macdh":         "MACD 柱状图，用于衡量动量强弱变化。",
	"rsi":           "RSI 指标，用于识别超买超卖与背离。",
	"boll":          "布林带中轨，衡量价格相对中枢。",
	"boll_ub":       "布林带上轨，衡量价格上沿压力。",
	"boll_lb":       "布林带下轨，衡量价格下沿支撑。",
	"atr":           "ATR 波动率指标，用于仓位和止损参考。",
	"vwma":          "成交量加权均线，用于结合量价确认趋势。",
	"mfi":           "资金流量指标，用于衡量量价驱动的超买超卖。",
}

func SafeTruncate(text string, limit int) string {
	clean := strings.Join(strings.Split(strings.TrimSpace(text), "\n"), " ")
	if len(clean) <= limit {
		return clean
	}
	return clean[:limit-3] + "..."
}

func FormatDataError(title string, exc error) string {
	return fmt.Sprintf("%s\n\n数据源访问失败：%v", title, SafeTruncate(exc.Error(), 220))
}

func FormatTable(data []map[string]any, title string, rows int) string {
	if len(data) == 0 {
		return fmt.Sprintf("%s\n\n暂无数据。", title)
	}

	if rows > len(data) {
		rows = len(data)
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("%s\n\n", title))

	if len(data) > 0 {
		keys := make([]string, 0, len(data[0]))
		for k := range data[0] {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		output.WriteString(strings.Join(keys, ",") + "\n")

		for i := 0; i < rows; i++ {
			var values []string
			for _, k := range keys {
				values = append(values, fmt.Sprintf("%v", data[i][k]))
			}
			output.WriteString(strings.Join(values, ",") + "\n")
		}
	}

	return output.String()
}

func RoundNumericValue(val string) string {
	f, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return val
	}
	return fmt.Sprintf("%.4f", f)
}

func RenameColumns(data []map[string]any) []map[string]any {
	renameMap := map[string]string{
		"日期":     "Date",
		"date":   "Date",
		"开盘":     "Open",
		"open":   "Open",
		"收盘":     "Close",
		"close":  "Close",
		"最高":     "High",
		"high":   "High",
		"最低":     "Low",
		"low":    "Low",
		"成交量":    "Volume",
		"成交额":    "Amount",
		"amount": "Amount",
		"振幅":     "AmplitudePct",
		"涨跌幅":    "PctChange",
		"涨跌额":    "PriceChange",
		"换手率":    "TurnoverPct",
	}

	var result []map[string]any
	for _, row := range data {
		newRow := make(map[string]any)
		for k, v := range row {
			if newK, ok := renameMap[k]; ok {
				newRow[newK] = v
			} else {
				newRow[k] = v
			}
		}
		result = append(result, newRow)
	}
	return result
}

func SelectColumns(data []map[string]any, columns []string) []map[string]any {
	var result []map[string]any
	for _, row := range data {
		newRow := make(map[string]any)
		for _, col := range columns {
			if v, ok := row[col]; ok {
				newRow[col] = v
			}
		}
		result = append(result, newRow)
	}
	return result
}

func IsRetryableError(err error) bool {
	msg := err.Error()
	retryableMarkers := []string{
		"Remote end closed connection",
		"Connection aborted",
		"Read timed out",
		"ConnectTimeout",
		"Max retries exceeded",
		"temporarily unavailable",
	}
	for _, marker := range retryableMarkers {
		if strings.Contains(msg, marker) {
			return true
		}
	}
	return false
}

func FilterNewsByDate(data []map[string]any, startDate, endDate string) []map[string]any {
	start, _ := time.Parse("2006-01-02", startDate)
	end, _ := time.Parse("2006-01-02", endDate)
	end = end.AddDate(0, 0, 1).Add(-time.Second)

	var filtered []map[string]any
	for _, row := range data {
		publishTime := fmt.Sprintf("%v", row["发布时间"])
		var t time.Time
		var err error

		patterns := []string{"2006-01-02 15:04:05", "2006-01-02T15:04:05", "2006-01-02"}
		for _, pattern := range patterns {
			t, err = time.Parse(pattern, publishTime)
			if err == nil {
				break
			}
		}

		if err != nil {
			continue
		}

		if t.After(start.Add(-time.Second)) && t.Before(end.Add(time.Second)) {
			filtered = append(filtered, row)
		}
	}

	sort.Slice(filtered, func(i, j int) bool {
		return fmt.Sprintf("%v", filtered[i]["发布时间"]) > fmt.Sprintf("%v", filtered[j]["发布时间"])
	})

	return filtered
}

func ConvertToSeries(data []map[string]any) indicators.Series {
	var series indicators.Series
	for _, row := range data {
		ohlcv := indicators.OHLCV{
			Date: fmt.Sprintf("%v", row["Date"]),
		}
		if v, ok := row["Open"].(string); ok {
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				ohlcv.Open = f
			}
		}
		if v, ok := row["High"].(string); ok {
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				ohlcv.High = f
			}
		}
		if v, ok := row["Low"].(string); ok {
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				ohlcv.Low = f
			}
		}
		if v, ok := row["Close"].(string); ok {
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				ohlcv.Close = f
			}
		}
		if v, ok := row["Volume"].(string); ok {
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				ohlcv.Volume = f
			}
		}
		series = append(series, ohlcv)
	}
	return series
}

func DataFrameToSeries(df *dataframe.DataFrame) indicators.Series {
	var series indicators.Series
	dateColIdx := -1
	openColIdx := -1
	highColIdx := -1
	lowColIdx := -1
	closeColIdx := -1
	volumeColIdx := -1

	names := df.Names()
	for i, name := range names {
		switch name {
		case "Date":
			dateColIdx = i
		case "Open":
			openColIdx = i
		case "High":
			highColIdx = i
		case "Low":
			lowColIdx = i
		case "Close":
			closeColIdx = i
		case "Volume":
			volumeColIdx = i
		}
	}

	for i := 0; i < df.Nrow(); i++ {
		ohlcv := indicators.OHLCV{}
		if dateColIdx >= 0 {
			ohlcv.Date = df.Elem(i, dateColIdx).String()
		}
		if openColIdx >= 0 {
			if v := df.Elem(i, openColIdx).String(); v != "" {
				if f, err := strconv.ParseFloat(v, 64); err == nil {
					ohlcv.Open = f
				}
			}
		}
		if highColIdx >= 0 {
			if v := df.Elem(i, highColIdx).String(); v != "" {
				if f, err := strconv.ParseFloat(v, 64); err == nil {
					ohlcv.High = f
				}
			}
		}
		if lowColIdx >= 0 {
			if v := df.Elem(i, lowColIdx).String(); v != "" {
				if f, err := strconv.ParseFloat(v, 64); err == nil {
					ohlcv.Low = f
				}
			}
		}
		if closeColIdx >= 0 {
			if v := df.Elem(i, closeColIdx).String(); v != "" {
				if f, err := strconv.ParseFloat(v, 64); err == nil {
					ohlcv.Close = f
				}
			}
		}
		if volumeColIdx >= 0 {
			if v := df.Elem(i, volumeColIdx).String(); v != "" {
				if f, err := strconv.ParseFloat(v, 64); err == nil {
					ohlcv.Volume = f
				}
			}
		}
		series = append(series, ohlcv)
	}
	return series
}

func FormatIndicatorValues(series indicators.Series, values []float64, lookBackDays int) string {
	var output strings.Builder
	startIdx := len(series) - lookBackDays
	if startIdx < 0 {
		startIdx = 0
	}

	for i := startIdx; i < len(series); i++ {
		if i >= len(values) || math.IsNaN(values[i]) {
			output.WriteString(fmt.Sprintf("%s: N/A\n", series[i].Date))
		} else {
			output.WriteString(fmt.Sprintf("%s: %.4f\n", series[i].Date, values[i]))
		}
	}
	return output.String()
}

func FormatMACDValues(series indicators.Series, result indicators.MACDResult, lookBackDays int) string {
	var output strings.Builder
	startIdx := len(series) - lookBackDays
	if startIdx < 0 {
		startIdx = 0
	}

	output.WriteString("Date: MACD | Signal | Histogram\n")
	for i := startIdx; i < len(series); i++ {
		if i >= len(result.MACD) || math.IsNaN(result.MACD[i]) {
			output.WriteString(fmt.Sprintf("%s: N/A\n", series[i].Date))
		} else {
			output.WriteString(fmt.Sprintf("%s: %.4f | %.4f | %.4f\n",
				series[i].Date, result.MACD[i], result.Signal[i], result.Hist[i]))
		}
	}
	return output.String()
}

func FormatBollingerValues(series indicators.Series, result indicators.BollingerResult, lookBackDays int) string {
	var output strings.Builder
	startIdx := len(series) - lookBackDays
	if startIdx < 0 {
		startIdx = 0
	}

	output.WriteString("Date: Middle | Upper | Lower\n")
	for i := startIdx; i < len(series); i++ {
		if i >= len(result.Middle) || math.IsNaN(result.Middle[i]) {
			output.WriteString(fmt.Sprintf("%s: N/A\n", series[i].Date))
		} else {
			output.WriteString(fmt.Sprintf("%s: %.4f | %.4f | %.4f\n",
				series[i].Date, result.Middle[i], result.Upper[i], result.Lower[i]))
		}
	}
	return output.String()
}

func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
