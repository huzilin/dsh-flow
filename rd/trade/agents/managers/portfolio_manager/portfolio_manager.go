package portfolio_manager

import (
	"fmt"
	"strings"

	"github.com/huzilin/trade/agents/states"
	"github.com/huzilin/trade/agents/utils"
	"github.com/huzilin/trade/llm"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// PortfolioManager 投资组合经理
type PortfolioManager struct {
	llm      *llm.OpenAIClientV2
	language string
	memory   *utils.TradingMemoryLog
}

// NewPortfolioManager 创建投资组合经理
func NewPortfolioManager(llmClient *llm.OpenAIClientV2, language string, memory ...*utils.TradingMemoryLog) *PortfolioManager {
	var mem *utils.TradingMemoryLog
	if len(memory) > 0 {
		mem = memory[0]
	}
	return &PortfolioManager{
		llm:      llmClient,
		language: language,
		memory:   mem,
	}
}

// PortfolioManagerNode 投资组合经理节点
func (pm *PortfolioManager) PortfolioManagerNode(state *states.AgentState) (*states.AgentState, error) {
	ticker := state.CompanyOfInterest
	instrumentContext := utils.GetInstrumentContextFromState(state)
	marketDescriptor := utils.GetMarketDescriptor(ticker)
	executionConstraints := utils.GetExecutionConstraintsPrompt(ticker)

	var history string
	if state.RiskDebateState != nil {
		history = state.RiskDebateState.History
	}

	researchPlan := state.InvestmentPlan
	traderPlan := state.TraderInvestmentPlan

	marketReport := state.MarketReport
	sentimentReport := state.SentimentReport
	newsReport := state.NewsReport
	fundamentalsReport := state.FundamentalsReport

	currSituation := fmt.Sprintf("%s\n\n%s\n\n%s\n\n%s", marketReport, sentimentReport, newsReport, fundamentalsReport)
	pastMemories := pm.getPastMemories(currSituation)

	prompt := pm.buildPrompt(instrumentContext, marketDescriptor, executionConstraints, history, researchPlan, traderPlan, pastMemories)
	llm.Logger().Debug("Portfolio Manager prompt", zap.String("prompt", prompt))

	messages := []*llm.ChatMessage{
		{Role: "user", Content: prompt},
	}

	resp, err := pm.llm.Chat(nil, messages, &llm.ChatOptions{})
	if err != nil {
		llm.Logger().Error("Portfolio Manager LLM call failed", zap.Error(err))
		return state, errors.Wrap(err, "Portfolio Manager LLM call failed")
	}

	finalTradeDecision := resp.Content

	newRiskDebateState := &states.RiskDebateState{
		History:             history,
		AggressiveHistory:   "",
		ConservativeHistory: "",
		NeutralHistory:      "",
		LatestSpeaker:       "Judge",
		JudgeDecision:       finalTradeDecision,
		Count:               0,
	}
	if state.RiskDebateState != nil {
		newRiskDebateState.AggressiveHistory = state.RiskDebateState.AggressiveHistory
		newRiskDebateState.ConservativeHistory = state.RiskDebateState.ConservativeHistory
		newRiskDebateState.NeutralHistory = state.RiskDebateState.NeutralHistory
		newRiskDebateState.CurrentAggressiveResponse = state.RiskDebateState.CurrentAggressiveResponse
		newRiskDebateState.CurrentConservativeResponse = state.RiskDebateState.CurrentConservativeResponse
		newRiskDebateState.CurrentNeutralResponse = state.RiskDebateState.CurrentNeutralResponse
		newRiskDebateState.Count = state.RiskDebateState.Count
	}

	state.RiskDebateState = newRiskDebateState
	state.FinalTradeDecision = finalTradeDecision

	return state, nil
}

// getPastMemories 获取历史记忆
func (pm *PortfolioManager) getPastMemories(currSituation string) string {
	if pm.memory == nil {
		return ""
	}
	memories := pm.memory.GetMemories(currSituation, 2)
	var parts []string
	for _, rec := range memories {
		if rec.Recommendation != "" {
			parts = append(parts, rec.Recommendation)
		}
	}
	return strings.Join(parts, "\n\n")
}

// buildPrompt 构建投资组合经理提示词
func (pm *PortfolioManager) buildPrompt(instrumentContext, marketDescriptor, executionConstraints, history, researchPlan, traderPlan, pastMemories string) string {
	languageInstruction := utils.GetUserFacingReportInstruction()

	lessonsLine := ""
	if pastMemories != "" {
		lessonsLine = fmt.Sprintf("- 过去决策的经验教训: **%s**\n", pastMemories)
	}

	return fmt.Sprintf(`作为%s投资组合经理，综合风险分析师的辩论并提供最终交易决策。

%s

---

**评级标准** (选择其中一个):
- **Buy**: 强烈信心买入或加仓
- **Overweight**: 前景看好，逐步增加敞口
- **Hold**: 维持当前仓位，无需操作
- **Underweight**: 减少敞口，部分获利了结
- **Sell**: 退出仓位或避免入场

**上下文:**
- 交易员的建议计划: **%s**
%s

**必需输出结构:**
1. **评级**: 选择 Buy / Overweight / Hold / Underweight / Sell 之一。
2. **执行摘要**: 涵盖入场策略、仓位大小、关键风险水平、预期持有期和市场特定执行约束的简明行动计划。
3. **投资逻辑**: 基于分析师辩论、政策背景和过去反思的详细推理。

---

**风险分析师辩论历史:**
%s

---

**研究辩论:**
%s

---

做出果断决策，将每个结论建立在分析师提供的具体证据之上。明确讨论%s，以及执行风险是否会改变最终行动。%s评级关键词本身保持英文（Buy / Overweight / Hold / Underweight / Sell）以便下游解析，但将所有标题和解释性文本转换为中文。`,
		marketDescriptor, instrumentContext, traderPlan, lessonsLine,
		history, researchPlan, executionConstraints, languageInstruction)
}
