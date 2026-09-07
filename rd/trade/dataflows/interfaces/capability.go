package interfaces

import (
	"reflect"

	"github.com/huzilin/trade/dataflows/models"
)

type Capability string

const (
	CapabilityStockData              Capability = "get_stock_data"              // 获取股票K线数据
	CapabilityIndicators             Capability = "get_indicators"             // 获取技术指标数据（RSI、MACD、布林带等）
	CapabilityFundamentals           Capability = "get_fundamentals"           // 获取基本面概况（市盈率、市净率、营收等）
	CapabilityBalanceSheet           Capability = "get_balance_sheet"           // 获取资产负债表数据
	CapabilityCashflow               Capability = "get_cashflow"               // 获取现金流量表数据
	CapabilityIncomeStatement        Capability = "get_income_statement"        // 获取损益表数据
	CapabilityNews                   Capability = "get_news"                   // 获取新闻（个股/全球/市场，通过symbol区分）
	CapabilityCompanyAnnouncements   Capability = "get_company_announcements"   // 获取公司公告
	CapabilityInsiderTransactions    Capability = "get_insider_transactions"    // 获取内部交易数据
	CapabilityMacroIndicators        Capability = "get_macro_indicators"        // 获取宏观经济指标（GDP、CPI、利率等）
	CapabilityPredictionMarkets      Capability = "get_prediction_markets"      // 获取预测市场数据（Polymarket等）
	CapabilityVerifiedMarketSnapshot Capability = "get_verified_market_snapshot" // 获取验证市场快照
)

type CapabilityDefinition struct {
	Name         Capability
	Description  string
	RequestType  reflect.Type
	ResponseType reflect.Type
}

var CapabilityRegistry = map[Capability]CapabilityDefinition{
	CapabilityStockData: {
		Name:         CapabilityStockData,
		Description:  "获取股票K线数据",
		RequestType:  reflect.TypeOf(StockDataRequest{}),
		ResponseType: reflect.TypeOf(""),
	},
	CapabilityIndicators: {
		Name:         CapabilityIndicators,
		Description:  "获取技术指标数据",
		RequestType:  reflect.TypeOf(IndicatorRequest{}),
		ResponseType: reflect.TypeOf(&models.IndicatorResult{}),
	},
	CapabilityFundamentals: {
		Name:         CapabilityFundamentals,
		Description:  "获取基本面概况数据",
		RequestType:  reflect.TypeOf(FundamentalRequest{}),
		ResponseType: reflect.TypeOf(&models.Fundamentals{}),
	},
	CapabilityBalanceSheet: {
		Name:         CapabilityBalanceSheet,
		Description:  "获取资产负债表数据",
		RequestType:  reflect.TypeOf(StatementRequest{}),
		ResponseType: reflect.TypeOf(&models.BalanceSheet{}),
	},
	CapabilityCashflow: {
		Name:         CapabilityCashflow,
		Description:  "获取现金流量表数据",
		RequestType:  reflect.TypeOf(StatementRequest{}),
		ResponseType: reflect.TypeOf(&models.Cashflow{}),
	},
	CapabilityIncomeStatement: {
		Name:         CapabilityIncomeStatement,
		Description:  "获取损益表数据",
		RequestType:  reflect.TypeOf(StatementRequest{}),
		ResponseType: reflect.TypeOf(&models.IncomeStatement{}),
	},
	CapabilityNews: {
		Name:         CapabilityNews,
		Description:  "获取新闻（个股/全球/市场）",
		RequestType:  reflect.TypeOf(NewsRequest{}),
		ResponseType: reflect.TypeOf(&models.NewsResult{}),
	},
	CapabilityCompanyAnnouncements: {
		Name:         CapabilityCompanyAnnouncements,
		Description:  "获取公司公告",
		RequestType:  reflect.TypeOf(AnnouncementRequest{}),
		ResponseType: reflect.TypeOf(&models.AnnouncementResult{}),
	},
	CapabilityInsiderTransactions: {
		Name:         CapabilityInsiderTransactions,
		Description:  "获取内部交易数据",
		RequestType:  reflect.TypeOf(InsiderTransactionRequest{}),
		ResponseType: reflect.TypeOf(""),
	},
	CapabilityMacroIndicators: {
		Name:         CapabilityMacroIndicators,
		Description:  "获取宏观经济指标",
		RequestType:  reflect.TypeOf(MacroRequest{}),
		ResponseType: reflect.TypeOf(&models.MacroResult{}),
	},
	CapabilityPredictionMarkets: {
		Name:         CapabilityPredictionMarkets,
		Description:  "获取预测市场数据",
		RequestType:  reflect.TypeOf(PredictionRequest{}),
		ResponseType: reflect.TypeOf(&models.PredictionResult{}),
	},
	CapabilityVerifiedMarketSnapshot: {
		Name:         CapabilityVerifiedMarketSnapshot,
		Description:  "获取验证市场快照",
		RequestType:  reflect.TypeOf(VerifiedSnapshotRequest{}),
		ResponseType: reflect.TypeOf(""),
	},
}

func IsValidCapability(cap Capability) bool {
	_, ok := CapabilityRegistry[cap]
	return ok
}

func GetCapabilityDefinition(cap Capability) (CapabilityDefinition, bool) {
	def, ok := CapabilityRegistry[cap]
	return def, ok
}

func GetAllCapabilities() []Capability {
	caps := make([]Capability, 0, len(CapabilityRegistry))
	for cap := range CapabilityRegistry {
		caps = append(caps, cap)
	}
	return caps
}
