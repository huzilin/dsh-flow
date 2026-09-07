package neutral_debator

import (
	"fmt"

	"github.com/huzilin/trade/agents/states"
	"github.com/huzilin/trade/agents/utils"
	"github.com/huzilin/trade/llm"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// NeutralDebator 中立风险分析师
type NeutralDebator struct {
	llm      *llm.OpenAIClientV2
	language string
}

// NewNeutralDebator 创建中立风险分析师
func NewNeutralDebator(llmClient *llm.OpenAIClientV2, language string) *NeutralDebator {
	return &NeutralDebator{
		llm:      llmClient,
		language: language,
	}
}

// NeutralNode 中立风险分析师节点
func (nd *NeutralDebator) NeutralNode(state *states.AgentState) (*states.AgentState, error) {
	riskDebateState := state.RiskDebateState
	if riskDebateState == nil {
		riskDebateState = &states.RiskDebateState{}
	}

	history := riskDebateState.History
	neutralHistory := riskDebateState.NeutralHistory
	currentAggressiveResponse := riskDebateState.CurrentAggressiveResponse
	currentConservativeResponse := riskDebateState.CurrentConservativeResponse

	marketReport := state.MarketReport
	sentimentReport := state.SentimentReport
	newsReport := state.NewsReport
	fundamentalsReport := state.FundamentalsReport
	instrumentContext := utils.GetInstrumentContextFromState(state)
	traderDecision := state.TraderInvestmentPlan

	ticker := state.CompanyOfInterest
	marketDescriptor := utils.GetMarketDescriptor(ticker)
	executionConstraints := utils.GetExecutionConstraintsPrompt(ticker)
	marketPolicyReportLabel := utils.GetMarketPolicyReportLabel(ticker)

	prompt := nd.buildPrompt(traderDecision, instrumentContext, marketDescriptor, executionConstraints, marketPolicyReportLabel,
		marketReport, sentimentReport, newsReport, fundamentalsReport,
		history, currentAggressiveResponse, currentConservativeResponse)

	llm.Logger().Debug("Neutral Debator prompt", zap.String("prompt", prompt))

	resp, err := nd.llm.Chat(nil, []*llm.ChatMessage{{Role: "user", Content: prompt}}, &llm.ChatOptions{})
	if err != nil {
		llm.Logger().Error("Neutral Debator LLM call failed", zap.Error(err))
		return state, errors.Wrap(err, "Neutral Debator LLM call failed")
	}

	argument := fmt.Sprintf("中性派分析师: %s", resp.Content)

	newRiskDebateState := &states.RiskDebateState{
		History:                     history + "\n" + argument,
		AggressiveHistory:           riskDebateState.AggressiveHistory,
		ConservativeHistory:         riskDebateState.ConservativeHistory,
		NeutralHistory:              neutralHistory + "\n" + argument,
		LatestSpeaker:               "Neutral",
		CurrentAggressiveResponse:   riskDebateState.CurrentAggressiveResponse,
		CurrentConservativeResponse: riskDebateState.CurrentConservativeResponse,
		CurrentNeutralResponse:      argument,
		Count:                       riskDebateState.Count + 1,
	}

	state.RiskDebateState = newRiskDebateState
	return state, nil
}

// buildPrompt 构建中立风险分析师提示词
func (nd *NeutralDebator) buildPrompt(traderDecision, instrumentContext, marketDescriptor, executionConstraints, marketPolicyReportLabel,
	marketReport, sentimentReport, newsReport, fundamentalsReport,
	history, currentAggressiveResponse, currentConservativeResponse string) string {

	return fmt.Sprintf(`作为%s的中立风险分析师，你的角色是提供平衡的视角，权衡交易员决策或计划的潜在收益和风险。判断在%s市场条件下，该设置是否支持参与、减仓或仅观察。以下是交易员的决策：

%s

你的任务是挑战激进分析师和保守分析师，指出每个视角可能过于乐观或过于谨慎的地方。使用以下数据源的见解来支持一个适度、可持续的策略，以调整交易员的决策：

%s
市场研究报告: %s
社交媒体情绪报告: %s
%s: %s
公司基本面报告: %s
当前对话历史: %s 激进分析师的最新回应: %s 保守分析师的最新回应: %s。如果其他观点尚未有回应，请根据可用数据提出自己的论点。

通过批判性分析双方来积极参与，解决激进和保守论点中的弱点，倡导更平衡的方法。明确讨论仓位大小、触发价格，以及考虑流动性和执行约束（如%s）后，交易是否仍然有意义。以对话方式输出，无需特殊格式。`,
		marketDescriptor, marketDescriptor, traderDecision, instrumentContext,
		marketReport, sentimentReport, marketPolicyReportLabel, newsReport,
		fundamentalsReport, history, currentAggressiveResponse, currentConservativeResponse,
		executionConstraints)
}
