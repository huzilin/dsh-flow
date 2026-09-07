package sentiment_analyst

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/huzilin/trade/agents/states"
	"github.com/huzilin/trade/agents/utils"
	"github.com/huzilin/trade/llm"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// SentimentAnalyst 情绪分析师
type SentimentAnalyst struct {
	llm      *llm.OpenAIClientV2
	language string
}

// NewSentimentAnalyst 创建情绪分析师
func NewSentimentAnalyst(llmClient *llm.OpenAIClientV2, language string) *SentimentAnalyst {
	return &SentimentAnalyst{
		llm:      llmClient,
		language: language,
	}
}

// SentimentAnalystNode 情绪分析师节点
func (sa *SentimentAnalyst) SentimentAnalystNode(state *states.AgentState) (*states.AgentState, error) {
	ticker := state.CompanyOfInterest
	endDate := state.TradeDate
	startDate := sa.sevenDaysBack(endDate)
	instrumentContext := utils.GetInstrumentContextFromState(state)
	marketDescriptor := utils.GetMarketIndefiniteDescriptor(ticker)
	marketScope := utils.GetMarketScope(ticker)

	systemMessage := sa.buildSystemMessage(marketDescriptor, marketScope, ticker, startDate, endDate)

	tools := utils.GetToolDefinitionsForAnalyst([]string{"get_news"})

	prompt := fmt.Sprintf(`You are a helpful AI assistant, collaborating with other assistants.
 Use the provided tools to progress towards answering the question.
 If you are unable to fully answer, that's OK; another assistant with different tools
 will help where you left off. Execute what you can to make progress.
 If you or any other assistant has the FINAL TRANSACTION PROPOSAL: **BUY/HOLD/SELL** or deliverable,
 prefix your response with FINAL TRANSACTION PROPOSAL: **BUY/HOLD/SELL** so the team knows to stop.
 You have access to the following tools: %s
%s
For your reference, the current date is %s. %s`, getToolNames(tools), systemMessage, endDate, instrumentContext)

	messages := []*llm.ChatMessage{
		{Role: "system", Content: prompt},
	}

	maxToolRounds := 3
	for round := 0; round < maxToolRounds; round++ {
		resp, err := sa.llm.Chat(context.Background(), messages, &llm.ChatOptions{
			Temperature: 0.7,
			Tools:       convertToLLMTools(tools),
		})
		if err != nil {
			llm.Logger().Error("Sentiment Analyst LLM call failed", zap.Error(err))
			return state, errors.Wrap(err, "Sentiment Analyst LLM call failed")
		}

		if len(resp.ToolCalls) == 0 {
			state.SentimentReport = resp.Content
			return state, nil
		}

		for _, tc := range resp.ToolCalls {
			toolResult, err := sa.executeToolCall(tc)
			if err != nil {
				llm.Logger().Warn("Tool execution failed, continuing", zap.Error(err))
				continue
			}
			messages = append(messages, &llm.ChatMessage{
				Role:    "assistant",
				Content: fmt.Sprintf("Tool call: %s(%s)", tc.Function.Name, tc.Function.Arguments),
			})
			messages = append(messages, &llm.ChatMessage{
				Role:    "tool",
				Content: toolResult,
			})
		}
	}

	state.SentimentReport = messages[len(messages)-1].Content
	return state, nil
}

// sevenDaysBack 计算7天前的日期
func (sa *SentimentAnalyst) sevenDaysBack(tradeDate string) string {
	t, err := time.Parse("2006-01-02", tradeDate)
	if err != nil {
		return tradeDate
	}
	return t.Add(-7 * 24 * time.Hour).Format("2006-01-02")
}

// buildSystemMessage 构建情绪分析师系统消息
func (sa *SentimentAnalyst) buildSystemMessage(marketDescriptor, marketScope, ticker, startDate, endDate string) string {
	return fmt.Sprintf(`你是%s情绪分析师，专注于散户情绪、媒体框架和公司特定的公众关注度。使用get_news(ticker, start_date, end_date)来总结代表%s投资者情绪的信号，如媒体语气、重复叙事、产品热度、资金流向关注度以及围绕股票的情绪化反应。不要假设直接访问海外社交平台；从财经媒体和公司特定新闻流中推断情绪。突出情绪是改善、过热还是恶化，并解释对%s交易可能产生的短期影响。确保在报告末尾附加Markdown表格，将报告中的关键点组织起来，条理清晰易读。`, marketDescriptor, marketScope, marketScope)
}

// executeToolCall 执行单个工具调用
func (sa *SentimentAnalyst) executeToolCall(tc llm.ToolCall) (string, error) {
	return utils.ExecuteToolCall(context.Background(), utils.ToolCall{
		Name:      tc.Function.Name,
		Arguments: []byte(tc.Function.Arguments),
	})
}

// getToolNames 获取工具名称列表
func getToolNames(tools []map[string]any) string {
	names := make([]string, 0, len(tools))
	for _, t := range tools {
		if fn, ok := t["function"].(map[string]any); ok {
			if name, ok := fn["name"].(string); ok {
				names = append(names, name)
			}
		}
	}
	return strings.Join(names, ", ")
}

// convertToLLMTools 转换工具定义为 llm.Tool 格式
func convertToLLMTools(tools []map[string]any) []*llm.Tool {
	result := make([]*llm.Tool, 0, len(tools))
	for _, t := range tools {
		if fn, ok := t["function"].(map[string]any); ok {
			name, _ := fn["name"].(string)
			desc, _ := fn["description"].(string)
			params := fn["parameters"]

			result = append(result, &llm.Tool{
				Type: "function",
				Function: &llm.ToolFunction{
					Name:        name,
					Description: desc,
					Parameters:  params,
				},
			})
		}
	}
	return result
}
