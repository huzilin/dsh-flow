package agents

import (
	"github.com/huzilin/trade/agents/analysts/fundamentals_analyst"
	"github.com/huzilin/trade/agents/analysts/market_analyst"
	"github.com/huzilin/trade/agents/analysts/news_analyst"
	"github.com/huzilin/trade/agents/analysts/sentiment_analyst"
	"github.com/huzilin/trade/agents/analysts/social_media_analyst"
	"github.com/huzilin/trade/agents/managers/portfolio_manager"
	"github.com/huzilin/trade/agents/managers/research_manager"
	"github.com/huzilin/trade/agents/researchers/bear_researcher"
	"github.com/huzilin/trade/agents/researchers/bull_researcher"
	"github.com/huzilin/trade/agents/risk_mgmt/aggressive_debator"
	"github.com/huzilin/trade/agents/risk_mgmt/conservative_debator"
	"github.com/huzilin/trade/agents/risk_mgmt/neutral_debator"
	"github.com/huzilin/trade/agents/states"
	"github.com/huzilin/trade/agents/trader/trader"
	"github.com/huzilin/trade/llm"
)

type (
	// AgentState 代理状态
	AgentState = states.AgentState
	// InvestDebateState 投资辩论状态
	InvestDebateState = states.InvestDebateState
	// RiskDebateState 风险辩论状态
	RiskDebateState = states.RiskDebateState
)

// NewMarketAnalyst 创建市场分析师
func NewMarketAnalyst(llmClient *llm.OpenAIClientV2, language string) *market_analyst.MarketAnalyst {
	return market_analyst.NewMarketAnalyst(llmClient, language)
}

// NewSentimentAnalyst 创建情绪分析师
func NewSentimentAnalyst(llmClient *llm.OpenAIClientV2, language string) *sentiment_analyst.SentimentAnalyst {
	return sentiment_analyst.NewSentimentAnalyst(llmClient, language)
}

// NewSocialMediaAnalyst 创建社交媒体分析师（已废弃）
func NewSocialMediaAnalyst(llmClient *llm.OpenAIClientV2, language string) *social_media_analyst.SocialMediaAnalyst {
	return social_media_analyst.NewSocialMediaAnalyst(llmClient, language)
}

// NewNewsAnalyst 创建新闻分析师
func NewNewsAnalyst(llmClient *llm.OpenAIClientV2, language string) *news_analyst.NewsAnalyst {
	return news_analyst.NewNewsAnalyst(llmClient, language)
}

// NewFundamentalsAnalyst 创建基本面分析师
func NewFundamentalsAnalyst(llmClient *llm.OpenAIClientV2, language string) *fundamentals_analyst.FundamentalsAnalyst {
	return fundamentals_analyst.NewFundamentalsAnalyst(llmClient, language)
}

// NewBullResearcher 创建多头研究员
func NewBullResearcher(llmClient *llm.OpenAIClientV2, language string) *bull_researcher.BullResearcher {
	return bull_researcher.NewBullResearcher(llmClient, language)
}

// NewBearResearcher 创建空头研究员
func NewBearResearcher(llmClient *llm.OpenAIClientV2, language string) *bear_researcher.BearResearcher {
	return bear_researcher.NewBearResearcher(llmClient, language)
}

// NewResearchManager 创建研究经理
func NewResearchManager(llmClient *llm.OpenAIClientV2, language string) *research_manager.ResearchManager {
	return research_manager.NewResearchManager(llmClient, language)
}

// NewPortfolioManager 创建组合经理
func NewPortfolioManager(llmClient *llm.OpenAIClientV2, language string) *portfolio_manager.PortfolioManager {
	return portfolio_manager.NewPortfolioManager(llmClient, language)
}

// NewTrader 创建交易员
func NewTrader(llmClient *llm.OpenAIClientV2, language string) *trader.Trader {
	return trader.NewTrader(llmClient, language)
}

// NewAggressiveDebator 创建激进风险分析师
func NewAggressiveDebator(llmClient *llm.OpenAIClientV2, language string) *aggressive_debator.AggressiveDebator {
	return aggressive_debator.NewAggressiveDebator(llmClient, language)
}

// NewConservativeDebator 创建保守风险分析师
func NewConservativeDebator(llmClient *llm.OpenAIClientV2, language string) *conservative_debator.ConservativeDebator {
	return conservative_debator.NewConservativeDebator(llmClient, language)
}

// NewNeutralDebator 创建中立风险分析师
func NewNeutralDebator(llmClient *llm.OpenAIClientV2, language string) *neutral_debator.NeutralDebator {
	return neutral_debator.NewNeutralDebator(llmClient, language)
}
