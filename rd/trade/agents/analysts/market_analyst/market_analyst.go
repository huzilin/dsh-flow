package market_analyst

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

// MarketAnalyst 市场/技术分析师
type MarketAnalyst struct {
	llm      *llm.OpenAIClientV2
	language string
}

// NewMarketAnalyst 创建市场分析师
func NewMarketAnalyst(llmClient *llm.OpenAIClientV2, language string) *MarketAnalyst {
	return &MarketAnalyst{
		llm:      llmClient,
		language: language,
	}
}

// MarketAnalystNode 市场分析师节点
func (ma *MarketAnalyst) MarketAnalystNode(state *states.AgentState) (*states.AgentState, error) {
	currentDate := state.TradeDate
	ticker := state.CompanyOfInterest
	instrumentContext := utils.GetInstrumentContextFromState(state)
	marketScope := utils.GetMarketScope(ticker)
	executionConstraints := utils.GetExecutionConstraintsPrompt(ticker)

	systemMessage := ma.buildSystemMessage(marketScope, executionConstraints)

	tools := utils.GetToolDefinitionsForAnalyst([]string{"get_stock_data", "get_indicators", "get_verified_market_snapshot"})

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
		resp, err := ma.llm.Chat(context.Background(), messages, &llm.ChatOptions{
			Temperature: 0.7,
			Tools:       convertToLLMTools(tools),
		})
		if err != nil {
			llm.Logger().Error("Market Analyst LLM call failed", zap.Error(err))
			return state, errors.Wrap(err, "Market Analyst LLM call failed")
		}

		if len(resp.ToolCalls) == 0 {
			state.MarketReport = resp.Content
			return state, nil
		}

		for _, tc := range resp.ToolCalls {
			toolResult, err := ma.executeToolCall(tc)
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

	state.MarketReport = messages[len(messages)-1].Content
	return state, nil
}

// buildSystemMessage 构建市场分析师系统消息
func (ma *MarketAnalyst) buildSystemMessage(marketScope, executionConstraints string) string {
	return fmt.Sprintf(`你是%s的技术分析师。你的任务是为%s当前市场状况或交易策略选择**最相关的指标**。目标是选择最多**8个指标**，提供互补的洞察力而不冗余。各类别及其指标如下：

移动平均线:
- close_50_sma: 50 SMA: 中期趋势指标。用途：识别趋势方向并作为动态支撑/阻力。提示：它滞后于价格；与更快的指标结合使用以获得及时信号。
- close_200_sma: 200 SMA: 长期趋势基准。用途：确认整体市场趋势并识别金叉/死叉形态。提示：反应较慢；最适合战略性趋势确认而非频繁交易入场。
- close_10_ema: 10 EMA: 响应迅速的短期平均线。用途：捕捉动量的快速变化和潜在入场点。提示：在震荡市场中容易产生噪音；与较长平均线一起使用以过滤虚假信号。

MACD相关:
- macd: MACD: 通过EMA差值计算动量。用途：寻找交叉和背离作为趋势变化信号。提示：在低波动或横盘市场中与其他指标确认。
- macds: MACD Signal: MACD线的EMA平滑。用途：使用与MACD线的交叉触发交易。提示：应作为更广泛策略的一部分以避免假阳性。
- macdh: MACD Histogram: 显示MACD线与其信号之间的差距。用途：可视化动量强度并及早发现背离。提示：可能波动较大；在快速波动市场中补充额外过滤器。

动量指标:
- rsi: RSI: 测量动量以标记超买/超卖状况。用途：应用70/30阈值并观察背离以发出反转信号。提示：在强趋势中，RSI可能保持极端；始终与趋势分析交叉检查。

波动性指标:
- boll: Bollinger Middle: 20 SMA作为布林带的基础。用途：作为价格变动的动态基准。提示：与上下轨结合使用以有效识别突破或反转。
- boll_ub: Bollinger Upper Band: 通常比中线高2个标准差。用途：发出潜在超买状况和突破区域信号。提示：用其他工具确认信号；在强趋势中价格可能沿轨道运行。
- boll_lb: Bollinger Lower Band: 通常比中线低2个标准差。用途：指示潜在超卖状况。提示：使用额外分析以避免虚假反转信号。
- atr: ATR: 平均真实波幅来衡量波动性。用途：设置止损水平并根据当前市场波动性调整仓位大小。提示：这是一种反应性指标，因此将其作为更广泛风险管理策略的一部分。

成交量指标:
- vwma: VWMA: 按成交量加权的移动平均线。用途：通过整合价格走势与成交量数据确认趋势。提示：注意成交量激增导致的扭曲结果；与其他成交量分析结合使用。

- 选择提供多样化和互补信息的指标。避免冗余。调用工具时，请使用上面提供的确切指标名称，因为它们是定义的参数，否则调用将失败。请确保先调用get_stock_data获取生成指标所需的CSV，然后使用get_indicators和特定指标名称。撰写一份详细报告，同时讨论执行问题，如%s，以及该设置是否有利于趋势跟踪或均值回归。确保在报告末尾附加Markdown表格，将报告中的关键点组织起来，条理清晰易读。`, marketScope, marketScope, executionConstraints)
}

// executeToolCall 执行单个工具调用
func (ma *MarketAnalyst) executeToolCall(tc llm.ToolCall) (string, error) {
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
