package conservative_debator

import (
	"fmt"

	"github.com/huzilin/trade/agents/states"
	"github.com/huzilin/trade/agents/utils"
	"github.com/huzilin/trade/llm"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// ConservativeDebator 保守风险分析师
type ConservativeDebator struct {
	llm      *llm.OpenAIClientV2
	language string
}

// NewConservativeDebator 创建保守风险分析师
func NewConservativeDebator(llmClient *llm.OpenAIClientV2, language string) *ConservativeDebator {
	return &ConservativeDebator{
		llm:      llmClient,
		language: language,
	}
}

// ConservativeNode 保守风险分析师节点
func (cd *ConservativeDebator) ConservativeNode(state *states.AgentState) (*states.AgentState, error) {
	riskDebateState := state.RiskDebateState
	if riskDebateState == nil {
		riskDebateState = &states.RiskDebateState{}
	}

	history := riskDebateState.History
	conservativeHistory := riskDebateState.ConservativeHistory
	currentAggressiveResponse := riskDebateState.CurrentAggressiveResponse
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

	prompt := cd.buildPrompt(traderDecision, instrumentContext, marketDescriptor, executionConstraints, marketPolicyReportLabel,
		marketReport, sentimentReport, newsReport, fundamentalsReport,
		history, currentAggressiveResponse, currentNeutralResponse)

	llm.Logger().Debug("Conservative Debator prompt", zap.String("prompt", prompt))

	resp, err := cd.llm.Chat(nil, []*llm.ChatMessage{{Role: "user", Content: prompt}}, &llm.ChatOptions{})
	if err != nil {
		llm.Logger().Error("Conservative Debator LLM call failed", zap.Error(err))
		return state, errors.Wrap(err, "Conservative Debator LLM call failed")
	}

	argument := fmt.Sprintf("保守派分析师: %s", resp.Content)

	newRiskDebateState := &states.RiskDebateState{
		History:                     history + "\n" + argument,
		AggressiveHistory:           riskDebateState.AggressiveHistory,
		ConservativeHistory:         conservativeHistory + "\n" + argument,
		NeutralHistory:              riskDebateState.NeutralHistory,
		LatestSpeaker:               "Conservative",
		CurrentAggressiveResponse:   riskDebateState.CurrentAggressiveResponse,
		CurrentConservativeResponse: argument,
		CurrentNeutralResponse:      riskDebateState.CurrentNeutralResponse,
		Count:                       riskDebateState.Count + 1,
	}

	state.RiskDebateState = newRiskDebateState
	return state, nil
}

// buildPrompt 构建保守风险分析师提示词
func (cd *ConservativeDebator) buildPrompt(traderDecision, instrumentContext, marketDescriptor, executionConstraints, marketPolicyReportLabel,
	marketReport, sentimentReport, newsReport, fundamentalsReport,
	history, currentAggressiveResponse, currentNeutralResponse string) string {

	return fmt.Sprintf(`作为%s的保守风险分析师，你的主要目标是保护资产、最小化波动性，并避免被高换手率主题、疲软流动性和执行风险所困。评估交易员的决策或计划时，批判性地检查高风险要素，指出决策可能使投资组合面临不当风险的地方，以及更谨慎的替代方案可以获得更好风险调整后回报的地方。以下是交易员的决策：

%s

你的任务是积极反驳激进分析师和中立分析师的论点，突出他们的观点可能忽视潜在威胁或未能优先考虑可持续性的地方。直接回应他们的观点，从以下数据源中汲取信息，为对交易员决策进行低风险调整构建一个有说服力的案例：

%s
市场研究报告: %s
社交媒体情绪报告: %s
%s: %s
公司基本面报告: %s
当前对话历史: %s 激进分析师的最新回应: %s 中立分析师的最新回应: %s。如果其他观点尚未有回应，请根据可用数据提出自己的论点。

通过质疑他们的乐观态度并强调他们可能忽视的潜在下行风险来参与，包括监管风险、情绪退潮、缩量下跌，以及在%s等约束下的执行摩擦。以对话方式输出，无需特殊格式。`,
		marketDescriptor, traderDecision, instrumentContext, marketReport, sentimentReport,
		marketPolicyReportLabel, newsReport, fundamentalsReport, history, currentAggressiveResponse, currentNeutralResponse,
		executionConstraints)
}
