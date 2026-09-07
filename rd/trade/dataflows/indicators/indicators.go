// Package indicators 提供金融技术指标计算
//
// 实现的核心指标包括：
//   - SMA/EMA: 简单/指数移动平均
//   - RSI: 相对强弱指标
//   - MACD: 移动平均收敛发散
//   - Bollinger Bands: 布林带
//   - ATR: 平均真实波幅
//   - VWMA: 成交量加权移动平均
//   - MFI: 资金流量指标
package indicators

import (
	"math"

	"github.com/pkg/errors"
)

// OHLCV 单条 K 线数据
type OHLCV struct {
	Date   string
	Open   float64
	High   float64
	Low    float64
	Close  float64
	Volume float64
}

// Series OHLCV 序列
type Series []OHLCV

// validatePeriod 验证周期参数
func validatePeriod(dataLen, period int, indicatorName string) error {
	if period <= 0 {
		return errors.Errorf("%s period must be positive, got %d", indicatorName, period)
	}
	if dataLen < period {
		return errors.Errorf("%s requires at least %d data points, got %d", indicatorName, period, dataLen)
	}
	return nil
}

// SMA 简单移动平均
func SMA(data []float64, period int) ([]float64, error) {
	if err := validatePeriod(len(data), period, "SMA"); err != nil {
		return nil, err
	}

	result := make([]float64, len(data))
	for i := 0; i < len(data); i++ {
		if i < period-1 {
			result[i] = math.NaN()
			continue
		}
		sum := 0.0
		for j := 0; j < period; j++ {
			sum += data[i-j]
		}
		result[i] = sum / float64(period)
	}
	return result, nil
}

// EMA 指数移动平均
func EMA(data []float64, period int) ([]float64, error) {
	if err := validatePeriod(len(data), period, "EMA"); err != nil {
		return nil, err
	}

	result := make([]float64, len(data))
	multiplier := 2.0 / (float64(period) + 1.0)

	// 第一个 EMA 使用 SMA
	sum := 0.0
	for i := 0; i < period; i++ {
		sum += data[i]
	}
	result[period-1] = sum / float64(period)

	for i := period; i < len(data); i++ {
		result[i] = (data[i]-result[i-1])*multiplier + result[i-1]
	}

	// 前 period-1 个设为 NaN
	for i := 0; i < period-1; i++ {
		result[i] = math.NaN()
	}

	return result, nil
}

// RSI 相对强弱指标
func RSI(data []float64, period int) ([]float64, error) {
	if err := validatePeriod(len(data), period, "RSI"); err != nil {
		return nil, err
	}

	result := make([]float64, len(data))
	gains := make([]float64, len(data))
	losses := make([]float64, len(data))

	// 计算价格变化
	for i := 1; i < len(data); i++ {
		change := data[i] - data[i-1]
		if change > 0 {
			gains[i] = change
			losses[i] = 0
		} else {
			gains[i] = 0
			losses[i] = -change
		}
	}

	// 计算平均收益和损失
	avgGain := make([]float64, len(data))
	avgLoss := make([]float64, len(data))

	// 第一个平均值使用简单平均
	gainSum, lossSum := 0.0, 0.0
	for i := 1; i <= period; i++ {
		gainSum += gains[i]
		lossSum += losses[i]
	}
	avgGain[period] = gainSum / float64(period)
	avgLoss[period] = lossSum / float64(period)

	// 后续使用平滑移动平均
	for i := period + 1; i < len(data); i++ {
		avgGain[i] = (avgGain[i-1]*(float64(period)-1) + gains[i]) / float64(period)
		avgLoss[i] = (avgLoss[i-1]*(float64(period)-1) + losses[i]) / float64(period)
	}

	// 计算 RSI
	for i := period; i < len(data); i++ {
		if avgLoss[i] == 0 {
			result[i] = 100
		} else {
			rs := avgGain[i] / avgLoss[i]
			result[i] = 100 - (100 / (1 + rs))
		}
	}

	// 前 period 个设为 NaN
	for i := 0; i < period; i++ {
		result[i] = math.NaN()
	}

	return result, nil
}

// MACDResult MACD 计算结果
type MACDResult struct {
	MACD   []float64
	Signal []float64
	Hist   []float64
}

// MACD 移动平均收敛发散
func MACD(data []float64, fastPeriod, slowPeriod, signalPeriod int) (*MACDResult, error) {
	if err := validatePeriod(len(data), slowPeriod, "MACD slow"); err != nil {
		return nil, err
	}

	fastEMA, err := EMA(data, fastPeriod)
	if err != nil {
		return nil, err
	}

	slowEMA, err := EMA(data, slowPeriod)
	if err != nil {
		return nil, err
	}

	macdLine := make([]float64, len(data))
	for i := 0; i < len(data); i++ {
		macdLine[i] = fastEMA[i] - slowEMA[i]
	}

	signalLine, err := EMA(macdLine, signalPeriod)
	if err != nil {
		return nil, err
	}

	hist := make([]float64, len(data))
	for i := 0; i < len(data); i++ {
		hist[i] = macdLine[i] - signalLine[i]
	}

	return &MACDResult{
		MACD:   macdLine,
		Signal: signalLine,
		Hist:   hist,
	}, nil
}

// BollingerResult 布林带结果
type BollingerResult struct {
	Middle []float64
	Upper  []float64
	Lower  []float64
}

// BollingerBands 布林带
func BollingerBands(data []float64, period int, stdDev float64) (*BollingerResult, error) {
	if err := validatePeriod(len(data), period, "Bollinger Bands"); err != nil {
		return nil, err
	}

	middle, err := SMA(data, period)
	if err != nil {
		return nil, err
	}

	upper := make([]float64, len(data))
	lower := make([]float64, len(data))

	for i := period - 1; i < len(data); i++ {
		variance := 0.0
		for j := 0; j < period; j++ {
			diff := data[i-j] - middle[i]
			variance += diff * diff
		}
		std := math.Sqrt(variance / float64(period))
		upper[i] = middle[i] + stdDev*std
		lower[i] = middle[i] - stdDev*std
	}

	// 前 period-1 个设为 NaN
	for i := 0; i < period-1; i++ {
		upper[i] = math.NaN()
		lower[i] = math.NaN()
	}

	return &BollingerResult{
		Middle: middle,
		Upper:  upper,
		Lower:  lower,
	}, nil
}

// ATR 平均真实波幅
func ATR(series Series, period int) ([]float64, error) {
	if err := validatePeriod(len(series), period+1, "ATR"); err != nil {
		return nil, err
	}

	tr := make([]float64, len(series))
	tr[0] = series[0].High - series[0].Low

	for i := 1; i < len(series); i++ {
		tr1 := series[i].High - series[i].Low
		tr2 := math.Abs(series[i].High - series[i-1].Close)
		tr3 := math.Abs(series[i].Low - series[i-1].Close)
		tr[i] = math.Max(tr1, math.Max(tr2, tr3))
	}

	result := make([]float64, len(series))

	// 第一个 ATR 使用简单平均
	sum := 0.0
	for i := 1; i <= period; i++ {
		sum += tr[i]
	}
	result[period] = sum / float64(period)

	// 后续使用平滑移动平均
	for i := period + 1; i < len(series); i++ {
		result[i] = (result[i-1]*(float64(period)-1) + tr[i]) / float64(period)
	}

	// 前 period 个设为 NaN
	for i := 0; i <= period; i++ {
		result[i] = math.NaN()
	}

	return result, nil
}

// VWMA 成交量加权移动平均
func VWMA(series Series, period int) ([]float64, error) {
	if err := validatePeriod(len(series), period, "VWMA"); err != nil {
		return nil, err
	}

	result := make([]float64, len(series))

	for i := period - 1; i < len(series); i++ {
		pvSum := 0.0
		vSum := 0.0
		for j := 0; j < period; j++ {
			pvSum += series[i-j].Close * series[i-j].Volume
			vSum += series[i-j].Volume
		}
		if vSum > 0 {
			result[i] = pvSum / vSum
		} else {
			result[i] = math.NaN()
		}
	}

	// 前 period-1 个设为 NaN
	for i := 0; i < period-1; i++ {
		result[i] = math.NaN()
	}

	return result, nil
}

// MFI 资金流量指标
func MFI(series Series, period int) ([]float64, error) {
	if err := validatePeriod(len(series), period+1, "MFI"); err != nil {
		return nil, err
	}

	typicalPrice := make([]float64, len(series))
	rawMoneyFlow := make([]float64, len(series))

	for i := 0; i < len(series); i++ {
		typicalPrice[i] = (series[i].High + series[i].Low + series[i].Close) / 3.0
		rawMoneyFlow[i] = typicalPrice[i] * series[i].Volume
	}

	positiveFlow := make([]float64, len(series))
	negativeFlow := make([]float64, len(series))

	for i := 1; i < len(series); i++ {
		if typicalPrice[i] > typicalPrice[i-1] {
			positiveFlow[i] = rawMoneyFlow[i]
			negativeFlow[i] = 0
		} else {
			positiveFlow[i] = 0
			negativeFlow[i] = rawMoneyFlow[i]
		}
	}

	result := make([]float64, len(series))

	for i := period; i < len(series); i++ {
		posSum := 0.0
		negSum := 0.0
		for j := 0; j < period; j++ {
			posSum += positiveFlow[i-j]
			negSum += negativeFlow[i-j]
		}

		if negSum == 0 {
			result[i] = 100
		} else {
			moneyRatio := posSum / negSum
			result[i] = 100 - (100 / (1 + moneyRatio))
		}
	}

	// 前 period 个设为 NaN
	for i := 0; i < period; i++ {
		result[i] = math.NaN()
	}

	return result, nil
}

// ExtractClose 从 OHLCV 序列提取收盘价
func ExtractClose(series Series) []float64 {
	result := make([]float64, len(series))
	for i, s := range series {
		result[i] = s.Close
	}
	return result
}

// ExtractVolume 从 OHLCV 序列提取成交量
func ExtractVolume(series Series) []float64 {
	result := make([]float64, len(series))
	for i, s := range series {
		result[i] = s.Volume
	}
	return result
}
