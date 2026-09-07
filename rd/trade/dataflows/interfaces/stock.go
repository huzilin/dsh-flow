// Package interfaces 定义股票数据请求结构
package interfaces

import "time"

// StockDataRequest 股票数据请求
type StockDataRequest struct {
	Symbol    string    // 标准化后的股票代码
	StartDate time.Time // 开始日期
	EndDate   time.Time // 结束日期
	Interval  string    // 时间间隔 (1d, 1h, 15m)，默认 1d
}

// IndicatorRequest 技术指标请求
type IndicatorRequest struct {
	Symbol       string              // 标准化后的股票代码
	Indicator    string              // 指标类型: sma, ema, rsi, macd, boll, atr, vwma, mfi
	CurrentDate  time.Time           // 当前日期
	LookbackDays int                 // 回溯天数
	Params       map[string]float64  // 可选参数 (如 period=50)
}

// VerifiedSnapshotRequest 验证市场快照请求
type VerifiedSnapshotRequest struct {
	Symbol       string    // 标准化后的股票代码
	CurrentDate  time.Time // 当前日期
	LookbackDays int       // 回溯天数
}
