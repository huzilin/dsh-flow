package aggressive_debator

import (
	"fmt"

	"github.com/huzilin/trade/agents/states"
	"github.com/huzilin/trade/agents/utils"
	"github.com/huzilin/trade/llm"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// AggressiveDebator 激进风险分析师
type AggressiveDebator struct {
	llm      *llm.OpenAIClientV2
	language string
}

// NewAggressiveDebator 创建激进风险分析师
func NewAggressiveDebator(llmClient *llm.OpenAIClientV2, language string) *AggressiveDebator {
	return &AggressiveDebator{
		llm:      llmClient,
		language: language,
	}
}

// AggressiveNode 激进风险分析师节点
func (ad *AggressiveDebator) AggressiveNode(state *states.AgentState) (*states.AgentState, error) {
	riskDebateState := state.RiskDebateState
	if riskDebateState == nil {
		riskDebateState = &states.RiskDebateState{}
	}

	history := riskDebateState.History
	aggressiveHistory := riskDebateState.AggressiveHistory
	currentConservativeResponse := riskDebateState.CurrentConservativeResponse
	currentNeutralResponse := riskDebateState.CurrentNeutralResponse

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

	prompt := ad.buildPrompt(traderDecision, instrumentContext, marketDescriptor, executionConstraints, marketPolicyReportLabel,
		marketReport, sentimentReport, newsReport, fundamentalsReport,
		history, currentConservativeResponse, currentNeutralResponse)

	llm.Logger().Debug("Aggressive Debator prompt", zap.String("prompt", prompt))

	resp, err := ad.llm.Chat(nil, []*llm.ChatMessage{{Role: "user", Content: prompt}}, &llm.ChatOptions{})
	if err != nil {
		llm.Logger().Error("Aggressive Debator LLM call failed", zap.Error(err))
		return state, errors.Wrap(err, "Aggressive Debator LLM call failed")
	}

	argument := fmt.Sprintf("激进派分析师: %s", resp.Content)

	newRiskDebateState := &states.RiskDebateState{
		History:                     history + "\n" + argument,
		AggressiveHistory:           aggressiveHistory + "\n" + argument,
		ConservativeHistory:         riskDebateState.ConservativeHistory,
		NeutralHistory:              riskDebateState.NeutralHistory,
		LatestSpeaker:               "Aggressive",
		CurrentAggressiveResponse:   argument,
		CurrentConservativeResponse: riskDebateState.CurrentConservativeResponse,
		CurrentNeutralResponse:      riskDebateState.CurrentNeutralResponse,
		Count:                       riskDebateState.Count + 1,
	}

	state.RiskDebateState = newRiskDebateState
	return state, nil
}

// buildPrompt 构建激进风险分析师提示词
func (ad *AggressiveDebator) buildPrompt(traderDecision, instrumentContext, marketDescriptor, executionConstraints, marketPolicyReportLabel,
	marketReport, sentimentReport, newsReport, fundamentalsReport,
	history, currentConservativeResponse, currentNeutralResponse string) string {

	return fmt.Sprintf(`作为%s的激进风险分析师，你的角色是积极倡导高回报机会，特别是当政策催化、板块轮动和资金情绪可能推动强劲上涨时。评估交易员的决策或计划时，关注潜在上涨空间以及为什么更快发展的%s主题可能证明接受更高波动性是合理的。直接回应保守分析师和中立分析师的每一个观点，用数据驱动的反驳和有说服力的推理来反击。以下是交易员的决策：

%s

你的任务是通过质疑和批评保守和中立立场来为交易员的决策创造一个有说服力的案例，以展示为什么你的高回报视角提供了最佳前进路径。将以下来源的见解纳入你的论点：

%s
市场研究报告: %s
社交媒体情绪报告: %s
%s: %s
公司基本面报告: %s
当前对话历史: %s 保守分析师的最新论点: %s 中立分析师的最新论点: %s。如果其他观点尚未有回应，请根据可用数据提出自己的论点。

通过解决提出的具体问题、反驳对立逻辑中的弱点以及断言为什么激进的%s方法在市场情绪和政策催化一致时可能表现出色来积极参与。明确权衡执行约束，如%s。以对话方式输出，无需特殊格式。`,
		marketDescriptor, marketDescriptor, traderDecision, instrumentContext, marketReport, sentimentReport,
		marketPolicyReportLabel, newsReport, fundamentalsReport, history, currentConservativeResponse, currentNeutralResponse,
		marketDescriptor, executionConstraints)
}
