package fundamentals_analyst

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

// FundamentalsAnalyst 基本面分析师
type FundamentalsAnalyst struct {
	llm      *llm.OpenAIClientV2
	language string
}

// NewFundamentalsAnalyst 创建基本面分析师
func NewFundamentalsAnalyst(llmClient *llm.OpenAIClientV2, language string) *FundamentalsAnalyst {
	return &FundamentalsAnalyst{
		llm:      llmClient,
		language: language,
	}
}

// FundamentalsAnalystNode 基本面分析师节点
func (fa *FundamentalsAnalyst) FundamentalsAnalystNode(state *states.AgentState) (*states.AgentState, error) {
	currentDate := state.TradeDate
	ticker := state.CompanyOfInterest
	instrumentContext := utils.GetInstrumentContextFromState(state)
	marketDescriptor := utils.GetMarketIndefiniteDescriptor(ticker)
	marketFocus := utils.GetFundamentalFocusPrompt(ticker)

	systemMessage := fa.buildSystemMessage(marketDescriptor, marketFocus)

	tools := utils.GetToolDefinitionsForAnalyst([]string{"get_fundamentals", "get_balance_sheet", "get_cashflow", "get_income_statement"})

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
		resp, err := fa.llm.Chat(context.Background(), messages, &llm.ChatOptions{
			Temperature: 0.7,
			Tools:       convertToLLMTools(tools),
		})
		if err != nil {
			llm.Logger().Error("Fundamentals Analyst LLM call failed", zap.Error(err))
			return state, errors.Wrap(err, "Fundamentals Analyst LLM call failed")
		}

		if len(resp.ToolCalls) == 0 {
			state.FundamentalsReport = resp.Content
			return state, nil
		}

		for _, tc := range resp.ToolCalls {
			toolResult, err := fa.executeToolCall(tc)
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

	state.FundamentalsReport = messages[len(messages)-1].Content
	return state, nil
}

// buildSystemMessage 构建基本面分析师系统消息
func (fa *FundamentalsAnalyst) buildSystemMessage(marketDescriptor, marketFocus string) string {
	return fmt.Sprintf(`你是%s基本面分析师。分析上市公司的业务概况、主营构成、收入质量、利润质量、现金流、杠杆率、利润率、ROE和资产负债表风险。%s解释最新基本面是否支持交易机会或警示估值和质量风险。确保在报告末尾附加Markdown表格，将报告中的关键点组织起来，条理清晰易读。使用可用工具：get_fundamentals获取综合公司分析，get_balance_sheet、get_cashflow和get_income_statement获取特定财务报表。`, marketDescriptor, marketFocus)
}

// executeToolCall 执行单个工具调用
func (fa *FundamentalsAnalyst) executeToolCall(tc llm.ToolCall) (string, error) {
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
