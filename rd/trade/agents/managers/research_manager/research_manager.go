package research_manager

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/huzilin/trade/agents/states"
	"github.com/huzilin/trade/agents/utils"
	"github.com/huzilin/trade/llm"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

var policySensitiveReplacements = []struct {
	source string
	target string
}{
	{"自杀式", "高风险"},
	{"自残式", "高风险"},
	{"血洗", "大幅回撤"},
	{"屠杀", "大幅抛压"},
	{"绞杀", "快速压缩"},
	{"杀跌", "恐慌性下跌"},
	{"砸盘", "大额卖出"},
	{"踩踏", "集中抛售"},
	{"暴雷", "突发利空"},
	{"爆仓", "强平风险"},
	{"腰斩", "大幅下跌"},
	{"死亡交叉", "下行交叉"},
	{"死叉", "下行交叉"},
	{"自救", "修复"},
}

var pastMistakesRegex = regexp.MustCompile(`(?i)\bpast mistakes\b`)
var mistakesRegex = regexp.MustCompile(`(?i)\bmistakes\b`)

// ResearchManager 研究经理
type ResearchManager struct {
	llm      *llm.OpenAIClientV2
	language string
	memory   *utils.TradingMemoryLog
}

// NewResearchManager 创建研究经理
func NewResearchManager(llmClient *llm.OpenAIClientV2, language string, memory ...*utils.TradingMemoryLog) *ResearchManager {
	var mem *utils.TradingMemoryLog
	if len(memory) > 0 {
		mem = memory[0]
	}
	return &ResearchManager{
		llm:      llmClient,
		language: language,
		memory:   mem,
	}
}

// ResearchManagerNode 研究经理节点
func (rm *ResearchManager) ResearchManagerNode(state *states.AgentState) (*states.AgentState, error) {
	ticker := state.CompanyOfInterest
	instrumentContext := utils.GetInstrumentContextFromState(state)
	marketDescriptor := utils.GetMarketDescriptor(ticker)
	executionConstraints := utils.GetExecutionConstraintsPrompt(ticker)

	var history string
	if state.InvestmentDebateState != nil {
		history = sanitizeFinancialPromptText(state.InvestmentDebateState.History)
	}

	marketReport := state.MarketReport
	sentimentReport := state.SentimentReport
	newsReport := state.NewsReport
	fundamentalsReport := state.FundamentalsReport

	currSituation := fmt.Sprintf("%s\n\n%s\n\n%s\n\n%s", marketReport, sentimentReport, newsReport, fundamentalsReport)
	pastMemories := rm.getPastMemories(currSituation)

	prompt := rm.buildPrompt(instrumentContext, marketDescriptor, executionConstraints, history, pastMemories)
	llm.Logger().Debug("Research Manager prompt", zap.String("prompt", prompt))

	messages := []*llm.ChatMessage{
		{Role: "user", Content: prompt},
	}

	resp, err := rm.llm.Chat(nil, messages, &llm.ChatOptions{})
	if err != nil {
		llm.Logger().Error("Research Manager LLM call failed", zap.Error(err))
		return state, errors.Wrap(err, "Research Manager LLM call failed")
	}

	investmentPlan := resp.Content

	newDebateState := &states.InvestDebateState{
		History:         history,
		BullHistory:     "",
		BearHistory:     "",
		CurrentResponse: investmentPlan,
		JudgeDecision:   investmentPlan,
		Count:           0,
	}
	if state.InvestmentDebateState != nil {
		newDebateState.Count = state.InvestmentDebateState.Count
	}

	state.InvestmentDebateState = newDebateState
	state.InvestmentPlan = investmentPlan

	return state, nil
}

// getPastMemories 获取历史记忆
func (rm *ResearchManager) getPastMemories(currSituation string) string {
	if rm.memory == nil {
		return ""
	}
	memories := rm.memory.GetMemories(currSituation, 2)
	return buildMemoryText(memories)
}

// buildPrompt 构建研究经理提示词
func (rm *ResearchManager) buildPrompt(instrumentContext, marketDescriptor, executionConstraints, history, pastMemories string) string {
	ratingScale := `**评级标准** (选择其中一个):
- **Buy**: 强烈看多; 建议建仓或加仓
- **Overweight**: 建设性观点; 建议逐步增加敞口
- **Hold**: 平衡观点; 建议维持当前仓位
- **Underweight**: 谨慎观点; 建议减仓
- **Sell**: 强烈看空; 建议退出或避免建仓

只要辩论中最强有力的论点支持明确立场，就应坚持明确立场；只有在双方证据真正平衡的情况下才保留Hold。`

	pastMemoriesLine := ""
	if pastMemories != "" {
		pastMemoriesLine = fmt.Sprintf("以下是你过去的反思和经验教训:\n\"%s\"\n\n", pastMemories)
	}

	return fmt.Sprintf(`作为%s研究经理和辩论主持人，批判性地评估本轮辩论并做出明确决策：支持看空分析师、支持看多分析师，或仅在设置确实缺乏优势时选择Hold。

简要总结双方的关键点，重点关注最有说服力的证据或推理。你的建议（Buy、Sell或Hold）必须清晰且可操作。不要仅仅因为双方都有合理观点就默认选择Hold；应基于辩论中最强有力的论点和执行约束（如%s）来坚持立场。

此外，为交易员制定详细的投资计划。应包括：

你的建议: 由最有说服力的论点支持的明确立场。
理由: 解释为什么这些论点导致你的结论。
战略行动: 实施建议的具体步骤，明确指出交易是否适合次日执行、短线操作或仅观察。

考虑你过去在类似情况下的经验教训。利用这些见解来完善你的决策并确保持续改进。避免暴力、自残或灾难隐喻；使用中性市场语言，如急剧下跌、下行风险、强制卖出或快速回撤。以对话方式呈现你的分析，就像自然说话一样，无需特殊格式。

%s
%s

---

%s

---

**辩论历史:**
%s`, marketDescriptor, executionConstraints, pastMemoriesLine, instrumentContext, ratingScale, history)
}

// sanitizeFinancialPromptText 将容易触发策略过滤的金融隐喻替换为中性表达
func sanitizeFinancialPromptText(text string) string {
	if text == "" {
		return ""
	}
	cleaned := text
	for _, replacement := range policySensitiveReplacements {
		cleaned = strings.ReplaceAll(cleaned, replacement.source, replacement.target)
	}
	cleaned = pastMistakesRegex.ReplaceAllString(cleaned, "past lessons")
	cleaned = mistakesRegex.ReplaceAllString(cleaned, "lessons")
	return cleaned
}

// buildMemoryText 将历史记忆整理为单段文本，并提前做降敏清洗
func buildMemoryText(memories []*utils.MemoryEntry) string {
	if len(memories) == 0 {
		return ""
	}
	var parts []string
	for _, rec := range memories {
		if rec.Recommendation != "" {
			parts = append(parts, sanitizeFinancialPromptText(rec.Recommendation))
		}
	}
	return strings.Join(parts, "\n\n")
}
