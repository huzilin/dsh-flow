package bull_researcher

import (
	"fmt"
	"strings"

	"github.com/huzilin/trade/agents/states"
	"github.com/huzilin/trade/agents/utils"
	"github.com/huzilin/trade/llm"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// BullResearcher 看涨研究员
type BullResearcher struct {
	llm      *llm.OpenAIClientV2
	language string
	memory   *utils.TradingMemoryLog
}

// NewBullResearcher 创建看涨研究员
func NewBullResearcher(llmClient *llm.OpenAIClientV2, language string, memory ...*utils.TradingMemoryLog) *BullResearcher {
	var mem *utils.TradingMemoryLog
	if len(memory) > 0 {
		mem = memory[0]
	}
	return &BullResearcher{
		llm:      llmClient,
		language: language,
		memory:   mem,
	}
}

// BullNode 看涨节点
func (br *BullResearcher) BullNode(state *states.AgentState) (*states.AgentState, error) {
	investmentDebateState := state.InvestmentDebateState
	if investmentDebateState == nil {
		investmentDebateState = &states.InvestDebateState{}
	}

	history := investmentDebateState.History
	bullHistory := investmentDebateState.BullHistory
	currentResponse := investmentDebateState.CurrentResponse

	marketReport := state.MarketReport
	sentimentReport := state.SentimentReport
	newsReport := state.NewsReport
	fundamentalsReport := state.FundamentalsReport

	ticker := state.CompanyOfInterest
	instrumentContext := utils.GetInstrumentContextFromState(state)
	marketDescriptor := utils.GetMarketDescriptor(ticker)
	marketPolicyReportLabel := utils.GetMarketPolicyReportLabel(ticker)

	currSituation := fmt.Sprintf("%s\n\n%s\n\n%s\n\n%s", marketReport, sentimentReport, newsReport, fundamentalsReport)
	pastMemories := br.getPastMemories(currSituation)

	prompt := br.buildPrompt(instrumentContext, marketDescriptor, marketPolicyReportLabel,
		marketReport, sentimentReport, newsReport, fundamentalsReport,
		history, currentResponse, pastMemories)

	llm.Logger().Debug("Bull Researcher prompt", zap.String("prompt", prompt))

	resp, err := br.llm.Chat(nil, []*llm.ChatMessage{{Role: "user", Content: prompt}}, &llm.ChatOptions{})
	if err != nil {
		llm.Logger().Error("Bull Researcher LLM call failed", zap.Error(err))
		return state, errors.Wrap(err, "Bull Researcher LLM call failed")
	}

	argument := fmt.Sprintf("多头研究员: %s", resp.Content)

	newDebateState := &states.InvestDebateState{
		History:         history + "\n" + argument,
		BullHistory:     bullHistory + "\n" + argument,
		BearHistory:     investmentDebateState.BearHistory,
		CurrentResponse: argument,
		Count:           investmentDebateState.Count + 1,
	}

	state.InvestmentDebateState = newDebateState
	return state, nil
}

// getPastMemories 获取历史记忆
func (br *BullResearcher) getPastMemories(currSituation string) string {
	if br.memory == nil {
		return ""
	}
	memories := br.memory.GetMemories(currSituation, 2)
	var parts []string
	for _, rec := range memories {
		if rec.Recommendation != "" {
			parts = append(parts, rec.Recommendation)
		}
	}
	return strings.Join(parts, "\n\n")
}

// buildPrompt 构建看涨研究员提示词
func (br *BullResearcher) buildPrompt(instrumentContext, marketDescriptor, marketPolicyReportLabel,
	marketReport, sentimentReport, newsReport, fundamentalsReport,
	history, currentResponse, pastMemories string) string {

	pastMemoriesLine := ""
	if pastMemories != "" {
		pastMemoriesLine = fmt.Sprintf("类似情况的反思和经验教训: %s\n", pastMemories)
	}

	return fmt.Sprintf(`你是一位看多分析师，倡导投资该股票。你的任务是建立一个强有力的、基于证据的论点，强调增长潜力、竞争优势和积极的市场指标。利用提供的研究和数据来有效解决担忧并反驳看空论点。

重点关注:
- 增长潜力: 突出公司的市场机会、收入预测和可扩展性。
- 竞争优势: 强调独特产品、强大品牌或主导市场定位等因素。
- 积极指标: 使用财务健康、行业趋势和近期正面新闻作为证据。
- 看空反驳: 用具体数据和合理推理批判性分析看空论点，彻底解决担忧，展示看多观点为何更有价值。
- 互动: 以对话风格呈现论点，直接回应看空分析师的观点并有效辩论，而非仅列出数据。
- 语气: 避免暴力、自残或灾难隐喻。使用中性市场措辞，如上涨空间、回撤、下行风险、抛售压力或估值压缩。

可用资源:
市场研究报告: %s
社交媒体情绪报告: %s
%s: %s
公司基本面报告: %s
辩论对话历史: %s
最新看空论点: %s
%s使用这些信息为%s市场提供一个有说服力的看多论点。讨论为什么政策支持、行业景气、估值修复、资金风格或催化事件可能证明上涨合理。反驳看空的担忧并从以往经验中学习。`, marketReport, sentimentReport, marketPolicyReportLabel, newsReport, fundamentalsReport, history, currentResponse, pastMemoriesLine, marketDescriptor)
}
