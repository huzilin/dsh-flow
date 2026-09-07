package agents

import (
	"testing"

	"github.com/huzilin/trade/agents/states"
	"github.com/huzilin/trade/logger"
	"github.com/huzilin/trade/types"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	logger.InitZap("", logger.ModeDev)
	m.Run()
}

func TestIntegration_TradingAgentsAgentCreation(t *testing.T) {
	t.Run("Default configuration", func(t *testing.T) {
		opts := &TradingAgentsAgentOptions{}
		agent := NewTradingAgentsAgent(opts)

		assert.NotNil(t, agent)
		assert.NotNil(t, agent.config)
		assert.Equal(t, int64(1), agent.config.MaxDebateRounds)
		assert.Equal(t, int64(1), agent.config.MaxRiskDiscussRounds)
		assert.Equal(t, int64(30), agent.config.ResearchDepth)
		assert.Equal(t, []string{"market", "social", "news", "fundamentals"}, agent.selectedAnalysts)
	})

	t.Run("Custom configuration", func(t *testing.T) {
		cfg := NewTradingAgentsConfig()
		cfg.MaxDebateRounds = 3
		cfg.MaxRiskDiscussRounds = 2
		cfg.ResearchDepth = 60

		opts := &TradingAgentsAgentOptions{
			Config:           cfg,
			SelectedAnalysts: []string{"market", "news"},
			Debug:            true,
		}
		agent := NewTradingAgentsAgent(opts)

		assert.NotNil(t, agent)
		assert.Equal(t, int64(3), agent.config.MaxDebateRounds)
		assert.Equal(t, int64(2), agent.config.MaxRiskDiscussRounds)
		assert.Equal(t, int64(60), agent.config.ResearchDepth)
		assert.Equal(t, []string{"market", "news"}, agent.selectedAnalysts)
		assert.True(t, agent.debug)
	})

	t.Run("Nil options", func(t *testing.T) {
		agent := NewTradingAgentsAgent(nil)

		assert.NotNil(t, agent)
		assert.NotNil(t, agent.config)
		assert.Equal(t, int64(1), agent.config.MaxDebateRounds)
	})
}

func TestIntegration_CreateInitialState(t *testing.T) {
	agent := NewTradingAgentsAgent(nil)

	t.Run("A-share stock", func(t *testing.T) {
		state := agent.createInitialState("600519", "2024-01-01")

		assert.Equal(t, "600519", state.CompanyOfInterest)
		assert.Equal(t, "stock", state.AssetType)
		assert.Equal(t, "2024-01-01", state.TradeDate)
		assert.Contains(t, state.InstrumentContext, "600519")
		assert.NotNil(t, state.InvestmentDebateState)
		assert.NotNil(t, state.RiskDebateState)
		assert.Equal(t, int64(0), state.InvestmentDebateState.Count)
		assert.Equal(t, int64(0), state.RiskDebateState.Count)
	})

	t.Run("HK stock", func(t *testing.T) {
		state := agent.createInitialState("00700.HK", "2024-01-01")

		assert.Equal(t, "00700.HK", state.CompanyOfInterest)
		assert.Equal(t, "stock", state.AssetType)
		assert.Contains(t, state.InstrumentContext, "港股")
	})

	t.Run("Crypto asset", func(t *testing.T) {
		state := agent.createInitialState("BTC", "2024-01-01")

		assert.Equal(t, "BTC", state.CompanyOfInterest)
		assert.Equal(t, "crypto", state.AssetType)
		assert.Contains(t, state.InstrumentContext, "加密资产")
	})
}

func TestIntegration_ExtractDecision(t *testing.T) {
	agent := NewTradingAgentsAgent(nil)

	tests := []struct {
		name           string
		input          string
		expectedAction types.DecisionAction
		expectedNil    bool
	}{
		{"Empty decision", "", types.DecisionActionHold, true},
		{"Buy decision", "最终评级: Buy", types.DecisionActionBuy, false},
		{"Sell decision", "最终评级: Sell", types.DecisionActionSell, false},
		{"Hold decision", "最终评级: Hold", types.DecisionActionHold, false},
		{"Overweight decision", "最终评级: Overweight", types.DecisionActionBuy, false},
		{"Underweight decision", "最终评级: Underweight", types.DecisionActionSell, false},
		{"Lowercase buy", "建议: buy", types.DecisionActionBuy, false},
		{"Lowercase sell", "建议: sell", types.DecisionActionSell, false},
		{"Mixed case BUY", "结论: BUY", types.DecisionActionBuy, false},
		{"Mixed case SELL", "结论: SELL", types.DecisionActionSell, false},
		{"No rating word", "这是一份分析报告", types.DecisionActionHold, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := agent.extractDecision(tt.input)

			if tt.expectedNil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assert.Equal(t, tt.expectedAction, result.Action)
				assert.Equal(t, "TradingAgents", result.AgentName)
				assert.Equal(t, tt.input, result.Rationale)
				assert.Equal(t, 1.0, result.Quantity)
			}
		})
	}
}

func TestIntegration_BuildResult(t *testing.T) {
	agent := NewTradingAgentsAgent(nil)

	state := &states.AgentState{
		CompanyOfInterest:    "600519",
		TradeDate:            "2024-01-01",
		MarketReport:         "市场报告内容",
		SentimentReport:      "情绪报告内容",
		NewsReport:           "新闻报告内容",
		FundamentalsReport:   "基本面报告内容",
		InvestmentPlan:       "投资计划内容",
		TraderInvestmentPlan: "交易员计划内容",
		FinalTradeDecision:   "最终评级: Buy",
		RiskDebateState: &states.RiskDebateState{
			History: "风险辩论历史",
		},
	}

	result := agent.buildResult(state, "600519", "2024-01-01")

	assert.Equal(t, "TradingAgents", result.AgentName)
	assert.NotNil(t, result.Decision)
	assert.Equal(t, types.DecisionActionBuy, result.Decision.Action)
	assert.Equal(t, "最终评级: Buy", result.Decision.Rationale)

	assert.NotNil(t, result.Outputs)
	assert.Equal(t, "市场报告内容", result.Outputs["market_report"])
	assert.Equal(t, "情绪报告内容", result.Outputs["sentiment_report"])
	assert.Equal(t, "新闻报告内容", result.Outputs["news_report"])
	assert.Equal(t, "基本面报告内容", result.Outputs["fundamentals_report"])
	assert.Equal(t, "投资计划内容", result.Outputs["investment_plan"])
	assert.Equal(t, "交易员计划内容", result.Outputs["trader_plan"])
	assert.Equal(t, "风险辩论历史", result.Outputs["risk_debate_history"])
	assert.Equal(t, "最终评级: Buy", result.Outputs["final_trade_decision"])
}

func TestIntegration_RunValidation(t *testing.T) {
	agent := NewTradingAgentsAgent(nil)

	req := &types.AgentRunRequest{
		Symbol:    "600519",
		TradeDate: "2024-01-01",
	}
	ctx := &types.AgentExecutionContext{}

	_, err := agent.run(req, ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "LLM clients not initialized")
}

func TestIntegration_ExecuteAnalysts(t *testing.T) {
	agent := NewTradingAgentsAgent(nil)

	state := &states.AgentState{
		CompanyOfInterest: "600519",
		AssetType:         "stock",
	}

	err := agent.executeAnalysts(state)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "market analyst not initialized")
}

func TestIntegration_ContainsToolCalls(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"No tool calls", "普通文本", false},
		{"Empty string", "", false},
		{"With tool call", "{\"name\": \"get_stock_data\", \"args\": []}", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsToolCalls(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
