package utils

import (
	"fmt"

	"github.com/huzilin/trade/agents/states"
	"github.com/huzilin/trade/dataflows/market_symbols"
)

// GetInstrumentContextFromState 从状态中获取标的上下文
// 使用方: 新闻分析师、基本面分析师、情绪分析师、市场分析师、
//
//	中立辩论者、保守辩论者、激进辩论者、
//	多头研究员、空头研究员、研究经理、投资组合经理、交易员
func GetInstrumentContextFromState(state *states.AgentState) string {
	if state.InstrumentContext != "" {
		return state.InstrumentContext
	}
	return BuildInstrumentContext(state.CompanyOfInterest, state.AssetType)
}

// BuildInstrumentContext 构建标的上下文
// 使用方: 交易代理入口(createInitialState)、GetInstrumentContextFromState
func BuildInstrumentContext(ticker string, assetType string) string {
	if assetType == "crypto" {
		return fmt.Sprintf("要分析的加密资产是`%s`。在每次工具调用、报告和建议中使用此精确代码。将其视为加密资产而非公司，不要假设公司基本面数据可用。", ticker)
	}

	normalizedTicker := normalizeMarketSymbol(ticker)
	if market_symbols.IsHKSymbol(normalizedTicker) {
		return fmt.Sprintf("要分析的标的是`%s`。在每次工具调用、报告和建议中使用此精确的港股代码，保留市场后缀`.HK`。", normalizedTicker)
	}

	marketLabel := market_symbols.GetMarketLabel(normalizedTicker)
	return fmt.Sprintf("要分析的标的是`%s`。在每次工具调用、报告和建议中使用此精确的%s代码，保留市场后缀`.SH`、`.SZ`或`.BJ`。", normalizedTicker, marketLabel)
}

// GetMarketDescriptor 返回市场描述词（A股/港股）
// 使用方: 中立辩论者、保守辩论者、激进辩论者、
//
//	多头研究员、空头研究员、研究经理、投资组合经理、交易员
func GetMarketDescriptor(ticker string) string {
	normalizedTicker := normalizeMarketSymbol(ticker)
	if market_symbols.IsHKSymbol(normalizedTicker) {
		return "港股"
	}
	return "A股"
}

// GetMarketIndefiniteDescriptor 返回带不定冠词的市场描述词（一只A股/一只港股）
// 使用方: 新闻分析师、基本面分析师、情绪分析师
func GetMarketIndefiniteDescriptor(ticker string) string {
	descriptor := GetMarketDescriptor(ticker)
	return fmt.Sprintf("一只%s", descriptor)
}

// GetMarketScope 返回市场范围描述（A股市场/港股市场）
// 使用方: 新闻分析师、情绪分析师、市场分析师
func GetMarketScope(ticker string) string {
	normalizedTicker := normalizeMarketSymbol(ticker)
	if market_symbols.IsHKSymbol(normalizedTicker) {
		return "港股市场"
	}
	return "A股市场"
}

// GetExecutionConstraintsPrompt 返回市场执行约束提示语
// 使用方: 市场分析师、中立辩论者、保守辩论者、激进辩论者、
//
//	研究经理、投资组合经理、交易员
func GetExecutionConstraintsPrompt(ticker string) string {
	if market_symbols.IsHKSymbol(normalizeMarketSymbol(ticker)) {
		return "手数大小、盘中流动性、隔夜中美科技情绪跳空风险、南向资金流向，以及多数港股主板股票不存在A股式的每日涨跌停限制或T+1转售约束"
	}
	return "T+1、涨跌停、成交额和换手率质量、以及情绪冲高回落风险"
}

// GetMarketNewsFocusPrompt 返回市场新闻关注点提示语
// 使用方: 新闻分析师
func GetMarketNewsFocusPrompt(ticker string) string {
	if market_symbols.IsHKSymbol(normalizeMarketSymbol(ticker)) {
		return "关注港交所式披露、公司新闻、中国政策溢出效应、美元/港币汇率、南向资金、板块轮动，以及任何可能影响港股次日或短线交易的事件。"
	}
	return "关注政策催化、监管变化、业绩披露窗口、流动性状况，以及任何可能影响A股次日或短线交易的事件。"
}

// GetFundamentalFocusPrompt 返回市场特定的基本面关注点
// 使用方: 基本面分析师
func GetFundamentalFocusPrompt(ticker string) string {
	if market_symbols.IsHKSymbol(normalizeMarketSymbol(ticker)) {
		return "特别关注港股特定信号，如收入质量、毛利结构、现金转换、杠杆率、ROE、回购、股东回报，以及相对港股同行的估值。"
	}
	return "特别关注A股特定基本面信号，如归母净利润、扣非净利润、经营现金流、存货、应收、商誉，以及主营业务的稳定性。"
}

// normalizeMarketSymbol 标准化市场符号（包装函数，处理错误）
func normalizeMarketSymbol(ticker string) string {
	normalized, err := market_symbols.NormalizeMarketSymbol(ticker)
	if err != nil {
		return ticker
	}
	return normalized
}

// GetInternalLanguageInstruction 返回内部语言提示语（中文）
// 使用方: 各分析师、研究员、辩论者的提示词构建
func GetInternalLanguageInstruction() string {
	return "将所有叙述性文本、章节标题、列表标签、表格标题和摘要使用中文编写。不要在行文中切换回英文。不要添加工作流叙述或元评论，如收集数据、编制报告或准备最终答案。除非需要显式的机器可读标记，否则翻译默认的英文标题。仅当是显式机器可读标记、股票代码、指标名称或下游评级关键字时，保留英文。"
}

// GetFinalLanguageInstruction 返回最终语言提示语（中文）
// 使用方: 报告整理器
func GetFinalLanguageInstruction() string {
	return "将所有叙述性文本、章节标题、列表标签、表格标题和摘要使用中文编写。不要在行文中切换回英文。不要添加工作流叙述或元评论。仅当是显式机器可读标记、股票代码、指标名称或下游评级关键字时，保留英文。"
}

// GetUserFacingReportInstruction 返回面向最终报告的统一输出约束（中文）
// 使用方: 投资组合经理、交易员
func GetUserFacingReportInstruction() string {
	return "仅返回完成的Markdown报告正文。不要包含流程叙述、工具调用摘要或真实内容之前的过渡句。"
}

// GetMarketPolicyReportLabel 返回市场政策报告标签
// 使用方: 中立辩论者、保守辩论者、激进辩论者、
//
//	多头研究员、空头研究员
func GetMarketPolicyReportLabel(ticker string) string {
	if market_symbols.IsHKSymbol(normalizeMarketSymbol(ticker)) {
		return "最新港股市场与政策报告"
	}
	return "最新A股市场与政策新闻"
}
