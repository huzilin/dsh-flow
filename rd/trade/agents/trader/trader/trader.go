package trader

import (
	"fmt"
	"strings"

	"github.com/huzilin/trade/agents/states"
	"github.com/huzilin/trade/agents/utils"
	"github.com/huzilin/trade/llm"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// Trader 交易员
type Trader struct {
	llm      *llm.OpenAIClientV2
	language string
	memory   *utils.TradingMemoryLog
}

// NewTrader 创建交易员
func NewTrader(llmClient *llm.OpenAIClientV2, language string, memory ...*utils.TradingMemoryLog) *Trader {
	var mem *utils.TradingMemoryLog
	if len(memory) > 0 {
		mem = memory[0]
	}
	return &Trader{
		llm:      llmClient,
		language: language,
		memory:   mem,
	}
}

// TraderNode 交易员节点
func (t *Trader) TraderNode(state *states.AgentState) (*states.AgentState, error) {
	companyName := state.CompanyOfInterest
	instrumentContext := utils.GetInstrumentContextFromState(state)
	marketDescriptor := utils.GetMarketDescriptor(companyName)
	executionConstraints := utils.GetExecutionConstraintsPrompt(companyName)
	investmentPlan := state.InvestmentPlan

	marketReport := state.MarketReport
	sentimentReport := state.SentimentReport
	newsReport := state.NewsReport
	fundamentalsReport := state.FundamentalsReport

	currSituation := fmt.Sprintf("%s\n\n%s\n\n%s\n\n%s", marketReport, sentimentReport, newsReport, fundamentalsReport)
	pastMemories := t.getPastMemories(currSituation)

	messages := []*llm.ChatMessage{
		{
			Role: "system",
			Content: fmt.Sprintf(`你是一名%s交易员。根据分析，提供买入、卖出或持有的具体建议。你的推理必须反映执行约束，包括%s，以及该设置是否可能继续或消退。应用过去决策的经验教训来加强你的分析。以下是你交易过的类似情况的反思和经验教训：%s%s保持所有解释性内容为中文，但将最终机器可读行保留为'FINAL TRANSACTION PROPOSAL: **BUY/HOLD/SELL**'。`,
				marketDescriptor, executionConstraints, pastMemories, utils.GetUserFacingReportInstruction()),
		},
		{
			Role: "user",
			Content: fmt.Sprintf(`基于对%s的%s综合分析，以下是投资计划。%s该计划包含技术结构、政策和市场新闻、基本面和情绪。使用此计划作为你下一个%s交易决策的基础。

建议投资计划: %s

利用这些见解做出明智和战略性的决策。`,
				companyName, marketDescriptor, instrumentContext, marketDescriptor, investmentPlan),
		},
	}

	resp, err := t.llm.Chat(nil, messages, &llm.ChatOptions{})
	if err != nil {
		llm.Logger().Error("Trader LLM call failed", zap.Error(err))
		return state, errors.Wrap(err, "Trader LLM call failed")
	}

	traderPlan := resp.Content
	state.TraderInvestmentPlan = traderPlan

	return state, nil
}

// getPastMemories 获取历史记忆
func (t *Trader) getPastMemories(currSituation string) string {
	if t.memory == nil {
		return "No past memories found."
	}
	memories := t.memory.GetMemories(currSituation, 2)
	var parts []string
	for _, rec := range memories {
		if rec.Recommendation != "" {
			parts = append(parts, rec.Recommendation)
		}
	}
	if len(parts) == 0 {
		return "No past memories found."
	}
	return strings.Join(parts, "\n\n")
}
