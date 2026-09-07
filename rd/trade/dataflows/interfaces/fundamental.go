// Package interfaces 定义基本面数据请求结构
package interfaces

import "time"

// FundamentalRequest 基本面数据请求
type FundamentalRequest struct {
	Symbol      string    // 标准化后的股票代码
	CurrentDate time.Time // 当前日期
}

// StatementRequest 财务报表请求
type StatementRequest struct {
	Symbol      string    // 标准化后的股票代码
	Frequency   string    // 报表频率: quarterly, annual
	CurrentDate time.Time // 当前日期
}