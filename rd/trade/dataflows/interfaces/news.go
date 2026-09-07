// Package interfaces 定义新闻数据请求结构
package interfaces

import "time"

// NewsRequest 新闻数据请求
type NewsRequest struct {
	Symbol    string    // 标准化后的股票代码
	StartDate time.Time // 开始日期
	EndDate   time.Time // 结束日期
}

// GlobalNewsRequest 全球新闻请求
type GlobalNewsRequest struct {
	CurrentDate  time.Time // 当前日期
	LookbackDays int       // 回溯天数
	Limit        int       // 返回条数限制
}

// AnnouncementRequest 公司公告请求
type AnnouncementRequest struct {
	Symbol    string    // 标准化后的股票代码
	StartDate time.Time // 开始日期
	EndDate   time.Time // 结束日期
}

// InsiderTransactionRequest 内部交易请求
type InsiderTransactionRequest struct {
	Symbol string // 标准化后的股票代码
}