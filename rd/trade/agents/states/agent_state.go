package states

// AgentState 代理主状态
type AgentState struct {
	// 基本信息
	CompanyOfInterest string `json:"company_of_interest" description:"关注的公司"`
	AssetType         string `json:"asset_type" description:"资产类型"`
	InstrumentContext string `json:"instrument_context" description:"确定性标的标识"`
	TradeDate         string `json:"trade_date" description:"交易日期"`

	Sender string `json:"sender" description:"发送消息的代理"`

	// 分析报告
	MarketReport      string `json:"market_report" description:"市场分析师报告"`
	SentimentReport   string `json:"sentiment_report" description:"情绪分析师报告"`
	NewsReport        string `json:"news_report" description:"新闻研究员报告"`
	FundamentalsReport string `json:"fundamentals_report" description:"基本面研究员报告"`

	// 研究团队讨论
	InvestmentDebateState *InvestDebateState `json:"investment_debate_state" description:"是否投资辩论的当前状态"`
	InvestmentPlan        string              `json:"investment_plan" description:"分析师生成的计划"`

	TraderInvestmentPlan string `json:"trader_investment_plan" description:"交易员生成的计划"`

	// 风险管理团队讨论
	RiskDebateState   *RiskDebateState `json:"risk_debate_state" description:"风险评估辩论的当前状态"`
	FinalTradeDecision string           `json:"final_trade_decision" description:"风险分析师的最终决策"`
	PastContext        string           `json:"past_context" description:"运行开始时注入的记忆日志上下文"`
}
