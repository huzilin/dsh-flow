package qa

import "github.com/huzilin/trade/skills"

// AnalysisContext 分析上下文，包含各 Agent 报告
type AnalysisContext struct {
	MarketReport       string
	SentimentReport    string
	NewsReport         string
	FundamentalsReport string
	InvestmentPlan     string
	FinalDecision      string
}

// QASessionState QQ 会话状态
type QASessionState struct {
	SessionID   string
	Ticker      string
	TradeDate   string
	History     []ChatMessage
	AnalysisCtx AnalysisContext
	ActiveSkill skills.Skill // 当前活跃的分析技能（可为 nil）
}

// ChatMessage 对话消息
type ChatMessage struct {
	Role    string // "user" 或 "assistant"
	Content string
}

// QARequest QQ 请求
type QARequest struct {
	Question     string
	SessionState *QASessionState
}

// QAResponse QQ 响应
type QAResponse struct {
	Answer     string   `json:"answer"`
	IntentType string   `json:"intent_type"`
	Confidence float64  `json:"confidence"`
	Citations  []string `json:"citations,omitempty"`
	Disclaimer string   `json:"disclaimer"`
}

// IntentType 意图类型
type IntentType string

const (
	IntentReportInterpretation IntentType = "report_interpretation"
	IntentDataQuery            IntentType = "data_query"
	IntentComparison           IntentType = "comparison"
	IntentRiskInquiry          IntentType = "risk_inquiry"
	IntentStrategyAdvice       IntentType = "strategy_advice"
	IntentMarketInsight        IntentType = "market_insight"
	IntentGeneral              IntentType = "general"
)

// IntentResult 意图分类结果
type IntentResult struct {
	Intent     IntentType `json:"intent"`
	Confidence float64    `json:"confidence"`
	Reasoning  string     `json:"reasoning"`
}

const defaultDisclaimer = "以上分析不构成投资建议，投资有风险，入市需谨慎。"

// NewQASession 创建新的 QA 会话
func NewQASession(ticker, tradeDate string) *QASessionState {
	return &QASessionState{
		SessionID: generateSessionID(ticker, tradeDate),
		Ticker:    ticker,
		TradeDate: tradeDate,
		History:   make([]ChatMessage, 0),
	}
}

// AddMessage 添加对话消息到历史
func (s *QASessionState) AddMessage(role, content string) {
	s.History = append(s.History, ChatMessage{Role: role, Content: content})
	// 只保留最近 10 轮对话
	if len(s.History) > 20 {
		s.History = s.History[len(s.History)-20:]
	}
}

// SetAnalysisContext 设置分析上下文
func (s *QASessionState) SetAnalysisContext(market, sentiment, news, fundamentals, plan, decision string) {
	s.AnalysisCtx = AnalysisContext{
		MarketReport:       market,
		SentimentReport:    sentiment,
		NewsReport:         news,
		FundamentalsReport: fundamentals,
		InvestmentPlan:     plan,
		FinalDecision:      decision,
	}
}

// BuildContextForLLM 构建发送给 LLM 的上下文文本
func (s *QASessionState) BuildContextForLLM() string {
	ctx := s.AnalysisCtx
	result := ""

	if ctx.MarketReport != "" {
		result += "## 市场分析报告\n" + ctx.MarketReport + "\n\n"
	}
	if ctx.SentimentReport != "" {
		result += "## 情绪分析报告\n" + ctx.SentimentReport + "\n\n"
	}
	if ctx.NewsReport != "" {
		result += "## 新闻分析报告\n" + ctx.NewsReport + "\n\n"
	}
	if ctx.FundamentalsReport != "" {
		result += "## 基本面分析报告\n" + ctx.FundamentalsReport + "\n\n"
	}
	if ctx.InvestmentPlan != "" {
		result += "## 投资计划\n" + ctx.InvestmentPlan + "\n\n"
	}
	if ctx.FinalDecision != "" {
		result += "## 最终决策\n" + ctx.FinalDecision + "\n\n"
	}

	return result
}

// BuildConversationHistory 构建对话历史文本
func (s *QASessionState) BuildConversationHistory() string {
	if len(s.History) == 0 {
		return ""
	}

	result := ""
	for _, msg := range s.History {
		result += msg.Role + ": " + msg.Content + "\n"
	}
	return result
}

func generateSessionID(ticker, tradeDate string) string {
	return ticker + "_" + tradeDate
}
