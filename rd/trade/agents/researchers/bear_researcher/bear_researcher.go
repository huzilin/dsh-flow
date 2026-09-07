package bear_researcher

import (
	"fmt"
	"strings"

	"github.com/huzilin/trade/agents/states"
	"github.com/huzilin/trade/agents/utils"
	"github.com/huzilin/trade/llm"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// BearResearcher 看跌研究员
type BearResearcher struct {
	llm      *llm.OpenAIClientV2
	language string
	memory   *utils.TradingMemoryLog
}

// NewBearResearcher 创建看跌研究员
func NewBearResearcher(llmClient *llm.OpenAIClientV2, language string, memory ...*utils.TradingMemoryLog) *BearResearcher {
	var mem *utils.TradingMemoryLog
	if len(memory) > 0 {
		mem = memory[0]
	}
	return &BearResearcher{
		llm:      llmClient,
		language: language,
		memory:   mem,
	}
}

// BearNode 看跌节点
func (br *BearResearcher) BearNode(state *states.AgentState) (*states.AgentState, error) {
	investmentDebateState := state.InvestmentDebateState
	if investmentDebateState == nil {
		investmentDebateState = &states.InvestDebateState{}
	}

	history := investmentDebateState.History
	bearHistory := investmentDebateState.BearHistory
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

	llm.Logger().Debug("Bear Researcher prompt", zap.String("prompt", prompt))

	resp, err := br.llm.Chat(nil, []*llm.ChatMessage{{Role: "user", Content: prompt}}, &llm.ChatOptions{})
	if err != nil {
		llm.Logger().Error("Bear Researcher LLM call failed", zap.Error(err))
		return state, errors.Wrap(err, "Bear Researcher LLM call failed")
	}

	argument := fmt.Sprintf("空头研究员: %s", resp.Content)

	newDebateState := &states.InvestDebateState{
		History:         history + "\n" + argument,
		BullHistory:     investmentDebateState.BullHistory,
		BearHistory:     bearHistory + "\n" + argument,
		CurrentResponse: argument,
		Count:           investmentDebateState.Count + 1,
	}

	state.InvestmentDebateState = newDebateState
	return state, nil
}

// getPastMemories 获取历史记忆
func (br *BearResearcher) getPastMemories(currSituation string) string {
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

// buildPrompt 构建看跌研究员提示词
func (br *BearResearcher) buildPrompt(instrumentContext, marketDescriptor, marketPolicyReportLabel,
	marketReport, sentimentReport, newsReport, fundamentalsReport,
	history, currentResponse, pastMemories string) string {

	pastMemoriesLine := ""
	if pastMemories != "" {
		pastMemoriesLine = fmt.Sprintf("类似情况的反思和经验教训: %s\n", pastMemories)
	}

	return fmt.Sprintf(`你是一位看空分析师，主张不投资该股票。你的目标是提出一个理由充分的论点，强调风险、挑战和负面指标。利用提供的研究和数据来突出潜在的下行风险并有效反驳看多论点。

重点关注:

- 风险和挑战: 突出市场饱和、财务不稳定或宏观经济威胁等可能阻碍股票表现的因素。
- 竞争劣势: 强调较弱的市场定位、创新下降或竞争对手威胁等脆弱性。
- 负面指标: 使用财务数据、市场趋势或近期不利新闻的证据来支持你的立场。
- 看多反驳: 用具体数据和合理推理批判性分析看多论点，揭露弱点或过度乐观的假设。
- 互动: 以对话风格呈现论点，直接回应看多分析师的观点并有效辩论，而非仅列出事实。
- 语气: 避免暴力、自残或灾难隐喻。使用中性市场措辞，如下行风险、回撤、集中抛售、流动性疲软或估值压力。

可用资源:

市场研究报告: %s
社交媒体情绪报告: %s
%s: %s
公司基本面报告: %s
辩论对话历史: %s
最新看多论点: %s
%s使用这些信息为%s市场提供一个有说服力的看空论点，包括政策不及预期、估值过高、业绩不及预期、流动性退潮或题材退潮风险。反驳看多的主张并从以往经验中学习。`, marketReport, sentimentReport, marketPolicyReportLabel, newsReport, fundamentalsReport, history, currentResponse, pastMemoriesLine, marketDescriptor)
}
