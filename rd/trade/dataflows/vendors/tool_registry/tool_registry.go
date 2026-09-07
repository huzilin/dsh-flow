package tool_registry

import (
	"context"
	"time"

	"github.com/huzilin/trade/dataflows/models"
)

// GetStockDataFunc 获取股票K线数据函数类型
// symbol: 股票代码, startDate/endDate: 日期范围(格式:2025-01-01)
type GetStockDataFunc func(ctx context.Context, symbol, startDate, endDate string) ([]*models.Candle, error)

// GetIndicatorsFunc 获取技术指标数据函数类型
// symbol: 股票代码, indicator: 指标名称(rsi/macd/bollinger), currDate: 当前日期(格式:2025-01-01), lookBackDays: 回溯天数
type GetIndicatorsFunc func(ctx context.Context, symbol, indicator, currDate string, lookBackDays int) (*models.IndicatorResult, error)

// GetFundamentalsFunc 获取基本面概况函数类型
// symbol: 股票代码, currDate: 当前日期
type GetFundamentalsFunc func(ctx context.Context, symbol string, currDate time.Time) (*models.Fundamentals, error)

// GetBalanceSheetFunc 获取资产负债表函数类型
// symbol: 股票代码, freq: 频率(annual/quarter), currDate: 当前日期
type GetBalanceSheetFunc func(ctx context.Context, symbol, freq string, currDate time.Time) (*models.BalanceSheet, error)

// GetCashflowFunc 获取现金流量表函数类型
// symbol: 股票代码, freq: 频率(annual/quarter), currDate: 当前日期
type GetCashflowFunc func(ctx context.Context, symbol, freq string, currDate time.Time) (*models.Cashflow, error)

// GetIncomeStatementFunc 获取损益表函数类型
// symbol: 股票代码, freq: 频率(annual/quarter), currDate: 当前日期
type GetIncomeStatementFunc func(ctx context.Context, symbol, freq string, currDate time.Time) (*models.IncomeStatement, error)

// GetNewsFunc 获取新闻函数类型
// symbol: 股票代码(为空时获取市场/全球新闻), startDate/endDate: 日期范围, limit: 返回条数限制
type GetNewsFunc func(ctx context.Context, symbol string, startDate, endDate time.Time, limit int) (*models.NewsResult, error)

// GetCompanyAnnouncementsFunc 获取公司公告函数类型
// symbol: 股票代码, startDate/endDate: 日期范围
type GetCompanyAnnouncementsFunc func(ctx context.Context, symbol string, startDate, endDate time.Time) (*models.AnnouncementResult, error)

// GetInsiderTransactionsFunc 获取内部交易数据函数类型
// symbol: 股票代码
type GetInsiderTransactionsFunc func(ctx context.Context, symbol string) (string, error)

// GetMacroIndicatorsFunc 获取宏观经济指标函数类型
// indicator: 指标名称(gdp/cpi/interest_rate等), currDate: 当前日期, lookBackDays: 回溯天数, params: 额外参数
type GetMacroIndicatorsFunc func(ctx context.Context, indicator string, currDate time.Time, lookBackDays int, params map[string]string) (*models.MacroResult, error)

// GetPredictionMarketsFunc 获取预测市场数据函数类型
// topic: 主题(如us_presidential_election), limit: 返回条数限制
type GetPredictionMarketsFunc func(ctx context.Context, topic string, limit int) (*models.PredictionResult, error)

// GetVerifiedMarketSnapshotFunc 获取验证市场快照函数类型
// symbol: 股票代码, currDate: 当前日期, lookBackDays: 回溯天数
type GetVerifiedMarketSnapshotFunc func(ctx context.Context, symbol string, currDate time.Time, lookBackDays int) (string, error)

// ToolRegistry 工具注册表，管理所有供应商的类型化能力注册与调用
type ToolRegistry struct {
	GetStockDataTools              map[string]GetStockDataFunc              // 获取股票K线数据的供应商映射
	GetIndicatorsTools             map[string]GetIndicatorsFunc             // 获取技术指标数据的供应商映射
	GetFundamentalsTools           map[string]GetFundamentalsFunc           // 获取基本面概况的供应商映射
	GetBalanceSheetTools           map[string]GetBalanceSheetFunc           // 获取资产负债表的供应商映射
	GetCashflowTools               map[string]GetCashflowFunc               // 获取现金流量表的供应商映射
	GetIncomeStatementTools        map[string]GetIncomeStatementFunc        // 获取损益表的供应商映射
	GetNewsTools                   map[string]GetNewsFunc                   // 获取新闻的供应商映射
	GetCompanyAnnouncementsTools   map[string]GetCompanyAnnouncementsFunc   // 获取公司公告的供应商映射
	GetInsiderTransactionsTools    map[string]GetInsiderTransactionsFunc    // 获取内部交易数据的供应商映射
	GetMacroIndicatorsTools        map[string]GetMacroIndicatorsFunc        // 获取宏观经济指标的供应商映射
	GetPredictionMarketsTools      map[string]GetPredictionMarketsFunc      // 获取预测市场数据的供应商映射
	GetVerifiedMarketSnapshotTools map[string]GetVerifiedMarketSnapshotFunc // 获取验证市场快照的供应商映射
}

// NewRegistry 创建一个新的工具注册表实例
func NewRegistry() *ToolRegistry {
	return &ToolRegistry{
		GetStockDataTools:              make(map[string]GetStockDataFunc),
		GetIndicatorsTools:             make(map[string]GetIndicatorsFunc),
		GetFundamentalsTools:           make(map[string]GetFundamentalsFunc),
		GetBalanceSheetTools:           make(map[string]GetBalanceSheetFunc),
		GetCashflowTools:               make(map[string]GetCashflowFunc),
		GetIncomeStatementTools:        make(map[string]GetIncomeStatementFunc),
		GetNewsTools:                   make(map[string]GetNewsFunc),
		GetCompanyAnnouncementsTools:   make(map[string]GetCompanyAnnouncementsFunc),
		GetInsiderTransactionsTools:    make(map[string]GetInsiderTransactionsFunc),
		GetMacroIndicatorsTools:        make(map[string]GetMacroIndicatorsFunc),
		GetPredictionMarketsTools:      make(map[string]GetPredictionMarketsFunc),
		GetVerifiedMarketSnapshotTools: make(map[string]GetVerifiedMarketSnapshotFunc),
	}
}

// RegisterGetStockData 注册获取股票K线数据的供应商
func (r *ToolRegistry) RegisterGetStockData(vendor string, fn GetStockDataFunc) {
	r.GetStockDataTools[vendor] = fn
}

// RegisterGetIndicators 注册获取技术指标数据的供应商
func (r *ToolRegistry) RegisterGetIndicators(vendor string, fn GetIndicatorsFunc) {
	r.GetIndicatorsTools[vendor] = fn
}

// RegisterGetFundamentals 注册获取基本面概况的供应商
func (r *ToolRegistry) RegisterGetFundamentals(vendor string, fn GetFundamentalsFunc) {
	r.GetFundamentalsTools[vendor] = fn
}

// RegisterGetBalanceSheet 注册获取资产负债表的供应商
func (r *ToolRegistry) RegisterGetBalanceSheet(vendor string, fn GetBalanceSheetFunc) {
	r.GetBalanceSheetTools[vendor] = fn
}

// RegisterGetCashflow 注册获取现金流量表的供应商
func (r *ToolRegistry) RegisterGetCashflow(vendor string, fn GetCashflowFunc) {
	r.GetCashflowTools[vendor] = fn
}

// RegisterGetIncomeStatement 注册获取损益表的供应商
func (r *ToolRegistry) RegisterGetIncomeStatement(vendor string, fn GetIncomeStatementFunc) {
	r.GetIncomeStatementTools[vendor] = fn
}

// RegisterGetNews 注册获取新闻的供应商
func (r *ToolRegistry) RegisterGetNews(vendor string, fn GetNewsFunc) {
	r.GetNewsTools[vendor] = fn
}

// RegisterGetCompanyAnnouncements 注册获取公司公告的供应商
func (r *ToolRegistry) RegisterGetCompanyAnnouncements(vendor string, fn GetCompanyAnnouncementsFunc) {
	r.GetCompanyAnnouncementsTools[vendor] = fn
}

// RegisterGetInsiderTransactions 注册获取内部交易数据的供应商
func (r *ToolRegistry) RegisterGetInsiderTransactions(vendor string, fn GetInsiderTransactionsFunc) {
	r.GetInsiderTransactionsTools[vendor] = fn
}

// RegisterGetMacroIndicators 注册获取宏观经济指标的供应商
func (r *ToolRegistry) RegisterGetMacroIndicators(vendor string, fn GetMacroIndicatorsFunc) {
	r.GetMacroIndicatorsTools[vendor] = fn
}

// RegisterGetPredictionMarkets 注册获取预测市场数据的供应商
func (r *ToolRegistry) RegisterGetPredictionMarkets(vendor string, fn GetPredictionMarketsFunc) {
	r.GetPredictionMarketsTools[vendor] = fn
}

// RegisterGetVerifiedMarketSnapshot 注册获取验证市场快照的供应商
func (r *ToolRegistry) RegisterGetVerifiedMarketSnapshot(vendor string, fn GetVerifiedMarketSnapshotFunc) {
	r.GetVerifiedMarketSnapshotTools[vendor] = fn
}

// GetGetStockData 获取获取股票K线数据的供应商函数
func (r *ToolRegistry) GetGetStockData(vendor string) (GetStockDataFunc, bool) {
	fn, ok := r.GetStockDataTools[vendor]
	return fn, ok
}

// GetGetIndicators 获取获取技术指标数据的供应商函数
func (r *ToolRegistry) GetGetIndicators(vendor string) (GetIndicatorsFunc, bool) {
	fn, ok := r.GetIndicatorsTools[vendor]
	return fn, ok
}

// GetGetFundamentals 获取获取基本面概况的供应商函数
func (r *ToolRegistry) GetGetFundamentals(vendor string) (GetFundamentalsFunc, bool) {
	fn, ok := r.GetFundamentalsTools[vendor]
	return fn, ok
}

// GetGetBalanceSheet 获取获取资产负债表的供应商函数
func (r *ToolRegistry) GetGetBalanceSheet(vendor string) (GetBalanceSheetFunc, bool) {
	fn, ok := r.GetBalanceSheetTools[vendor]
	return fn, ok
}

// GetGetCashflow 获取获取现金流量表的供应商函数
func (r *ToolRegistry) GetGetCashflow(vendor string) (GetCashflowFunc, bool) {
	fn, ok := r.GetCashflowTools[vendor]
	return fn, ok
}

// GetGetIncomeStatement 获取获取损益表的供应商函数
func (r *ToolRegistry) GetGetIncomeStatement(vendor string) (GetIncomeStatementFunc, bool) {
	fn, ok := r.GetIncomeStatementTools[vendor]
	return fn, ok
}

// GetGetNews 获取获取新闻的供应商函数
func (r *ToolRegistry) GetGetNews(vendor string) (GetNewsFunc, bool) {
	fn, ok := r.GetNewsTools[vendor]
	return fn, ok
}

// GetGetCompanyAnnouncements 获取获取公司公告的供应商函数
func (r *ToolRegistry) GetGetCompanyAnnouncements(vendor string) (GetCompanyAnnouncementsFunc, bool) {
	fn, ok := r.GetCompanyAnnouncementsTools[vendor]
	return fn, ok
}

// GetGetInsiderTransactions 获取获取内部交易数据的供应商函数
func (r *ToolRegistry) GetGetInsiderTransactions(vendor string) (GetInsiderTransactionsFunc, bool) {
	fn, ok := r.GetInsiderTransactionsTools[vendor]
	return fn, ok
}

// GetGetMacroIndicators 获取获取宏观经济指标的供应商函数
func (r *ToolRegistry) GetGetMacroIndicators(vendor string) (GetMacroIndicatorsFunc, bool) {
	fn, ok := r.GetMacroIndicatorsTools[vendor]
	return fn, ok
}

// GetGetPredictionMarkets 获取获取预测市场数据的供应商函数
func (r *ToolRegistry) GetGetPredictionMarkets(vendor string) (GetPredictionMarketsFunc, bool) {
	fn, ok := r.GetPredictionMarketsTools[vendor]
	return fn, ok
}

// GetGetVerifiedMarketSnapshot 获取获取验证市场快照的供应商函数
func (r *ToolRegistry) GetGetVerifiedMarketSnapshot(vendor string) (GetVerifiedMarketSnapshotFunc, bool) {
	fn, ok := r.GetVerifiedMarketSnapshotTools[vendor]
	return fn, ok
}

// GetStockDataVendors 获取注册了股票K线数据能力的供应商列表
func (r *ToolRegistry) GetStockDataVendors() []string {
	return getStringKeys(r.GetStockDataTools)
}

// GetIndicatorsVendors 获取注册了技术指标数据能力的供应商列表
func (r *ToolRegistry) GetIndicatorsVendors() []string {
	return getStringKeys(r.GetIndicatorsTools)
}

// GetFundamentalsVendors 获取注册了基本面概况能力的供应商列表
func (r *ToolRegistry) GetFundamentalsVendors() []string {
	return getStringKeys(r.GetFundamentalsTools)
}

// GetBalanceSheetVendors 获取注册了资产负债表能力的供应商列表
func (r *ToolRegistry) GetBalanceSheetVendors() []string {
	return getStringKeys(r.GetBalanceSheetTools)
}

// GetCashflowVendors 获取注册了现金流量表能力的供应商列表
func (r *ToolRegistry) GetCashflowVendors() []string {
	return getStringKeys(r.GetCashflowTools)
}

// GetIncomeStatementVendors 获取注册了损益表能力的供应商列表
func (r *ToolRegistry) GetIncomeStatementVendors() []string {
	return getStringKeys(r.GetIncomeStatementTools)
}

// GetNewsVendors 获取注册了新闻能力的供应商列表
func (r *ToolRegistry) GetNewsVendors() []string {
	return getStringKeys(r.GetNewsTools)
}

// GetCompanyAnnouncementsVendors 获取注册了公司公告能力的供应商列表
func (r *ToolRegistry) GetCompanyAnnouncementsVendors() []string {
	return getStringKeys(r.GetCompanyAnnouncementsTools)
}

// GetInsiderTransactionsVendors 获取注册了内部交易数据能力的供应商列表
func (r *ToolRegistry) GetInsiderTransactionsVendors() []string {
	return getStringKeys(r.GetInsiderTransactionsTools)
}

// GetMacroIndicatorsVendors 获取注册了宏观经济指标能力的供应商列表
func (r *ToolRegistry) GetMacroIndicatorsVendors() []string {
	return getStringKeys(r.GetMacroIndicatorsTools)
}

// GetPredictionMarketsVendors 获取注册了预测市场数据能力的供应商列表
func (r *ToolRegistry) GetPredictionMarketsVendors() []string {
	return getStringKeys(r.GetPredictionMarketsTools)
}

// GetVerifiedMarketSnapshotVendors 获取注册了验证市场快照能力的供应商列表
func (r *ToolRegistry) GetVerifiedMarketSnapshotVendors() []string {
	return getStringKeys(r.GetVerifiedMarketSnapshotTools)
}

func getStringKeys(m any) []string {
	switch v := m.(type) {
	case map[string]GetStockDataFunc:
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		return keys
	case map[string]GetIndicatorsFunc:
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		return keys
	case map[string]GetFundamentalsFunc:
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		return keys
	case map[string]GetBalanceSheetFunc:
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		return keys
	case map[string]GetCashflowFunc:
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		return keys
	case map[string]GetIncomeStatementFunc:
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		return keys
	case map[string]GetNewsFunc:
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		return keys
	case map[string]GetCompanyAnnouncementsFunc:
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		return keys
	case map[string]GetInsiderTransactionsFunc:
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		return keys
	case map[string]GetMacroIndicatorsFunc:
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		return keys
	case map[string]GetPredictionMarketsFunc:
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		return keys
	case map[string]GetVerifiedMarketSnapshotFunc:
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		return keys
	default:
		return nil
	}
}

var Registry = NewRegistry()
