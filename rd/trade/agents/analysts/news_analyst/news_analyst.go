package news_analyst

import (
	"context"
	"fmt"
	"strings"

	"github.com/huzilin/trade/agents/states"
	"github.com/huzilin/trade/agents/utils"
	"github.com/huzilin/trade/llm"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// NewsAnalyst 新闻分析师
type NewsAnalyst struct {
	llm      *llm.OpenAIClientV2
	language string
}

// NewNewsAnalyst 创建新闻分析师
func NewNewsAnalyst(llmClient *llm.OpenAIClientV2, language string) *NewsAnalyst {
	return &NewsAnalyst{
		llm:      llmClient,
		language: language,
	}
}

// NewsAnalystNode 新闻分析师节点
func (na *NewsAnalyst) NewsAnalystNode(state *states.AgentState) (*states.AgentState, error) {
	currentDate := state.TradeDate
	ticker := state.CompanyOfInterest
	instrumentContext := utils.GetInstrumentContextFromState(state)
	marketDescriptor := utils.GetMarketIndefiniteDescriptor(ticker)
	marketScope := utils.GetMarketScope(ticker)
	marketNewsFocus := utils.GetMarketNewsFocusPrompt(ticker)

	systemMessage := na.buildSystemMessage(marketDescriptor, marketScope, marketNewsFocus)

	tools := utils.GetToolDefinitionsForAnalyst([]string{"get_news", "get_market_news", "get_company_announcements"})

	prompt := fmt.Sprintf(`You are a helpful AI assistant, collaborating with other assistants.
 Use the provided tools to progress towards answering the question.
 If you are unable to fully answer, that's OK; another assistant with different tools
 will help where you left off. Execute what you can to make progress.
 If you or any other assistant has the FINAL TRANSACTION PROPOSAL: **BUY/HOLD/SELL** or deliverable,
 prefix your response with FINAL TRANSACTION PROPOSAL: **BUY/HOLD/SELL** so the team knows to stop.
 You have access to the following tools: %s
%s
For your reference, the current date is %s. %s`, getToolNames(tools), systemMessage, currentDate, instrumentContext)

	messages := []*llm.ChatMessage{
		{Role: "system", Content: prompt},
	}

	maxToolRounds := 5
	for round := 0; round < maxToolRounds; round++ {
		resp, err := na.llm.Chat(context.Background(), messages, &llm.ChatOptions{
			Temperature: 0.7,
			Tools:       convertToLLMTools(tools),
		})
		if err != nil {
			llm.Logger().Error("News Analyst LLM call failed", zap.Error(err))
			return state, errors.Wrap(err, "News Analyst LLM call failed")
		}

		if len(resp.ToolCalls) == 0 {
			state.NewsReport = resp.Content
			return state, nil
		}

		for _, tc := range resp.ToolCalls {
			toolResult, err := na.executeToolCall(tc)
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

	state.NewsReport = messages[len(messages)-1].Content
	return state, nil
}

// buildSystemMessage 构建新闻分析师系统消息
func (na *NewsAnalyst) buildSystemMessage(marketDescriptor, marketScope, marketNewsFocus string) string {
	return fmt.Sprintf(`你是%s新闻研究员。分析公司新闻、上市公司公告、行业政策变化，以及对交易重要的%s范围内的政策和宏观信号。使用get_news(ticker, start_date, end_date)获取公司新闻，get_company_announcements(ticker, start_date, end_date, category)获取披露记录，get_market_news(curr_date, look_back_days, limit)获取更广泛的市场、政策和宏观新闻。%s确保在报告末尾附加Markdown表格，将报告中的关键点组织起来，条理清晰易读。`, marketDescriptor, marketScope, marketNewsFocus)
}

// executeToolCall 执行单个工具调用
func (na *NewsAnalyst) executeToolCall(tc llm.ToolCall) (string, error) {
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
