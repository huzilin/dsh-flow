package utils

import (
	"context"

	"github.com/huzilin/trade/dataflows/tools"
)

// GetVerifiedMarketSnapshot 获取经过验证的市场快照
// 返回最新 OHLCV 行、常见技术指标和近期收盘价，作为精确声明的"真相来源"
// 在分析师做出关于价格水平、布林带、RSI、MACD、移动平均线、支撑/阻力位
// 或历史比较的精确声明之前调用此函数。
func GetVerifiedMarketSnapshot(ctx context.Context, symbol, currDate string, lookBackDays int) (string, error) {
	return tools.GetVerifiedMarketSnapshot(ctx, symbol, currDate, lookBackDays)
}
