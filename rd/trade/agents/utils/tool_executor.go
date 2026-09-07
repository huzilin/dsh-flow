package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/huzilin/trade/dataflows/tools"
	"github.com/huzilin/trade/logger"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// ToolCall LLM 工具调用
type ToolCall struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// ToolResult 工具执行结果
type ToolResult struct {
	Content string `json:"content"`
	IsError bool   `json:"is_error,omitempty"`
}

// ExecuteToolCall 执行单个工具调用
func ExecuteToolCall(ctx context.Context, toolCall ToolCall) (string, error) {
	logger.L().Info("Executing tool", zap.String("name", toolCall.Name))

	var args []interface{}
	if err := json.Unmarshal(toolCall.Arguments, &args); err != nil {
		// 如果解析失败，尝试解析为 map
		var argsMap map[string]interface{}
		if err2 := json.Unmarshal(toolCall.Arguments, &argsMap); err2 == nil {
			args = make([]interface{}, 0, len(argsMap))
			for _, v := range argsMap {
				args = append(args, v)
			}
		} else {
			return "", errors.Wrap(err, "failed to parse tool arguments")
		}
	}

	switch toolCall.Name {
	case "get_stock_data":
		if len(args) >= 3 {
			return tools.GetStockData(ctx,
				fmt.Sprintf("%v", args[0]),
				fmt.Sprintf("%v", args[1]),
				fmt.Sprintf("%v", args[2]))
		}
	case "get_indicators":
		if len(args) >= 4 {
			lookBack := 30
			if len(args) > 4 {
				if lb, ok := args[4].(float64); ok {
					lookBack = int(lb)
				}
			}
			return tools.GetIndicators(ctx,
				fmt.Sprintf("%v", args[0]),
				fmt.Sprintf("%v", args[1]),
				fmt.Sprintf("%v", args[2]),
				lookBack)
		}
	case "get_fundamentals":
		if len(args) >= 1 {
			currDate := ""
			if len(args) > 1 {
				currDate = fmt.Sprintf("%v", args[1])
			}
			return tools.GetFundamentals(ctx, fmt.Sprintf("%v", args[0]), currDate)
		}
	case "get_balance_sheet":
		if len(args) >= 1 {
			freq := "quarterly"
			currDate := ""
			if len(args) > 1 {
				freq = fmt.Sprintf("%v", args[1])
			}
			if len(args) > 2 {
				currDate = fmt.Sprintf("%v", args[2])
			}
			return tools.GetBalanceSheet(ctx, fmt.Sprintf("%v", args[0]), freq, currDate)
		}
	case "get_cashflow":
		if len(args) >= 1 {
			freq := "quarterly"
			currDate := ""
			if len(args) > 1 {
				freq = fmt.Sprintf("%v", args[1])
			}
			if len(args) > 2 {
				currDate = fmt.Sprintf("%v", args[2])
			}
			return tools.GetCashflow(ctx, fmt.Sprintf("%v", args[0]), freq, currDate)
		}
	case "get_income_statement":
		if len(args) >= 1 {
			freq := "quarterly"
			currDate := ""
			if len(args) > 1 {
				freq = fmt.Sprintf("%v", args[1])
			}
			if len(args) > 2 {
				currDate = fmt.Sprintf("%v", args[2])
			}
			return tools.GetIncomeStatement(ctx, fmt.Sprintf("%v", args[0]), freq, currDate)
		}
	case "get_news":
		if len(args) >= 3 {
			return tools.GetNews(ctx,
				fmt.Sprintf("%v", args[0]),
				fmt.Sprintf("%v", args[1]),
				fmt.Sprintf("%v", args[2]))
		}
	case "get_company_announcements":
		if len(args) >= 3 {
			category := "全部"
			if len(args) > 3 {
				category = fmt.Sprintf("%v", args[3])
			}
			return tools.GetCompanyAnnouncements(ctx,
				fmt.Sprintf("%v", args[0]),
				fmt.Sprintf("%v", args[1]),
				fmt.Sprintf("%v", args[2]),
				category)
		}
	case "get_market_news":
		if len(args) >= 1 {
			lookBack := 7
			limit := 10
			if len(args) > 1 {
				if lb, ok := args[1].(float64); ok {
					lookBack = int(lb)
				}
			}
			if len(args) > 2 {
				if l, ok := args[2].(float64); ok {
					limit = int(l)
				}
			}
			return tools.GetMarketNews(ctx, fmt.Sprintf("%v", args[0]), lookBack, limit)
		}
	case "get_global_news":
		if len(args) >= 1 {
			lookBack := 7
			limit := 10
			if len(args) > 1 {
				if lb, ok := args[1].(float64); ok {
					lookBack = int(lb)
				}
			}
			if len(args) > 2 {
				if l, ok := args[2].(float64); ok {
					limit = int(l)
				}
			}
			return tools.GetGlobalNews(ctx, fmt.Sprintf("%v", args[0]), lookBack, limit)
		}
	case "get_insider_transactions":
		if len(args) >= 1 {
			return tools.GetInsiderTransactions(ctx, fmt.Sprintf("%v", args[0]))
		}
	case "get_macro_indicators":
		if len(args) >= 2 {
			lookBack := 365
			if len(args) > 2 {
				if lb, ok := args[2].(float64); ok {
					lookBack = int(lb)
				}
			}
			return tools.GetMacroIndicators(ctx,
				fmt.Sprintf("%v", args[0]),
				fmt.Sprintf("%v", args[1]),
				lookBack)
		}
	case "get_prediction_markets":
		if len(args) >= 1 {
			limit := 6
			if len(args) > 1 {
				if l, ok := args[1].(float64); ok {
					limit = int(l)
				}
			}
			return tools.GetPredictionMarkets(ctx, fmt.Sprintf("%v", args[0]), limit)
		}
	case "get_verified_market_snapshot":
		if len(args) >= 2 {
			lookBack := 30
			if len(args) > 2 {
				if lb, ok := args[2].(float64); ok {
					lookBack = int(lb)
				}
			}
			return tools.GetVerifiedMarketSnapshot(ctx,
				fmt.Sprintf("%v", args[0]),
				fmt.Sprintf("%v", args[1]),
				lookBack)
		}
	default:
		return "", errors.Errorf("unknown tool: %s", toolCall.Name)
	}

	return "", errors.Errorf("insufficient arguments for tool: %s", toolCall.Name)
}

// ParseToolCalls 解析 LLM 返回的工具调用
func ParseToolCalls(content string) []ToolCall {
	var toolCalls []ToolCall

	// 尝试解析 JSON 数组格式
	if strings.HasPrefix(strings.TrimSpace(content), "[") {
		if err := json.Unmarshal([]byte(content), &toolCalls); err == nil {
			return toolCalls
		}
	}

	// 尝试解析单个工具调用
	var singleCall ToolCall
	if err := json.Unmarshal([]byte(content), &singleCall); err == nil {
		if singleCall.Name != "" {
			return []ToolCall{singleCall}
		}
	}

	return nil
}

// GetAvailableToolsDescription 返回可用工具的描述（用于提示词）
func GetAvailableToolsDescription() string {
	return tools.ToolDescriptions()
}

// GetToolDefinitionsForAnalyst 根据工具名称列表过滤工具定义
func GetToolDefinitionsForAnalyst(toolNames []string) []map[string]any {
	allTools := GetToolDefinitions()
	toolSet := make(map[string]bool)
	for _, name := range toolNames {
		toolSet[name] = true
	}

	var filtered []map[string]any
	for _, tool := range allTools {
		if fn, ok := tool["function"].(map[string]any); ok {
			if name, ok := fn["name"].(string); ok {
				if toolSet[name] {
					filtered = append(filtered, tool)
				}
			}
		}
	}
	return filtered
}

// GetToolDefinitions 返回工具定义（用于 LLM 函数调用）
func GetToolDefinitions() []map[string]interface{} {
	return []map[string]interface{}{
		{
			"type": "function",
			"function": map[string]interface{}{
				"name":        "get_stock_data",
				"description": "Retrieve stock price data (OHLCV) for a given ticker symbol",
				"parameters": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"symbol":     map[string]interface{}{"type": "string", "description": "Ticker symbol of the company"},
						"start_date": map[string]interface{}{"type": "string", "description": "Start date in yyyy-mm-dd format"},
						"end_date":   map[string]interface{}{"type": "string", "description": "End date in yyyy-mm-dd format"},
					},
					"required": []string{"symbol", "start_date", "end_date"},
				},
			},
		},
		{
			"type": "function",
			"function": map[string]interface{}{
				"name":        "get_indicators",
				"description": "Retrieve a single technical indicator for a given ticker symbol",
				"parameters": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"symbol":         map[string]interface{}{"type": "string", "description": "Ticker symbol"},
						"indicator":      map[string]interface{}{"type": "string", "description": "Technical indicator name (e.g., rsi, macd, boll)"},
						"curr_date":      map[string]interface{}{"type": "string", "description": "Current trading date in YYYY-mm-dd format"},
						"look_back_days": map[string]interface{}{"type": "integer", "description": "How many days to look back", "default": 30},
					},
					"required": []string{"symbol", "indicator", "curr_date"},
				},
			},
		},
		{
			"type": "function",
			"function": map[string]interface{}{
				"name":        "get_fundamentals",
				"description": "Retrieve comprehensive fundamental data for a given ticker symbol",
				"parameters": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"ticker":    map[string]interface{}{"type": "string", "description": "Ticker symbol"},
						"curr_date": map[string]interface{}{"type": "string", "description": "Current date in yyyy-mm-dd format"},
					},
					"required": []string{"ticker"},
				},
			},
		},
		{
			"type": "function",
			"function": map[string]interface{}{
				"name":        "get_news",
				"description": "Retrieve news data for a given ticker symbol",
				"parameters": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"ticker":     map[string]interface{}{"type": "string", "description": "Ticker symbol"},
						"start_date": map[string]interface{}{"type": "string", "description": "Start date in yyyy-mm-dd format"},
						"end_date":   map[string]interface{}{"type": "string", "description": "End date in yyyy-mm-dd format"},
					},
					"required": []string{"ticker", "start_date", "end_date"},
				},
			},
		},
		{
			"type": "function",
			"function": map[string]interface{}{
				"name":        "get_company_announcements",
				"description": "Retrieve company announcements for a given ticker symbol",
				"parameters": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"ticker":     map[string]interface{}{"type": "string", "description": "Ticker symbol"},
						"start_date": map[string]interface{}{"type": "string", "description": "Start date in yyyy-mm-dd format"},
						"end_date":   map[string]interface{}{"type": "string", "description": "End date in yyyy-mm-dd format"},
						"category":   map[string]interface{}{"type": "string", "description": "Announcement category such as 全部, 财务报告, 风险提示", "default": "全部"},
					},
					"required": []string{"ticker", "start_date", "end_date"},
				},
			},
		},
		{
			"type": "function",
			"function": map[string]interface{}{
				"name":        "get_market_news",
				"description": "Retrieve market-wide news and policy updates",
				"parameters": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"curr_date":      map[string]interface{}{"type": "string", "description": "Current date in yyyy-mm-dd format"},
						"look_back_days": map[string]interface{}{"type": "integer", "description": "Days to look back", "default": 7},
						"limit":          map[string]interface{}{"type": "integer", "description": "Max articles to return", "default": 10},
					},
					"required": []string{"curr_date"},
				},
			},
		},
		{
			"type": "function",
			"function": map[string]interface{}{
				"name":        "get_macro_indicators",
				"description": "Retrieve macroeconomic indicator time series from FRED",
				"parameters": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"indicator":      map[string]interface{}{"type": "string", "description": "Macro indicator (e.g., cpi, unemployment, 10y_treasury)"},
						"curr_date":      map[string]interface{}{"type": "string", "description": "Current date in yyyy-mm-dd format"},
						"look_back_days": map[string]interface{}{"type": "integer", "description": "Trailing window length in days"},
					},
					"required": []string{"indicator", "curr_date"},
				},
			},
		},
		{
			"type": "function",
			"function": map[string]interface{}{
				"name":        "get_prediction_markets",
				"description": "Retrieve live prediction market probabilities",
				"parameters": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"topic": map[string]interface{}{"type": "string", "description": "Event topic/keyword"},
						"limit": map[string]interface{}{"type": "integer", "description": "Max markets to return", "default": 6},
					},
					"required": []string{"topic"},
				},
			},
		},
		{
			"type": "function",
			"function": map[string]interface{}{
				"name":        "get_global_news",
				"description": "Retrieve global news data",
				"parameters": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"curr_date":      map[string]interface{}{"type": "string", "description": "Current date in yyyy-mm-dd format"},
						"look_back_days": map[string]interface{}{"type": "integer", "description": "Days to look back"},
						"limit":          map[string]interface{}{"type": "integer", "description": "Max articles to return"},
					},
					"required": []string{"curr_date"},
				},
			},
		},
		{
			"type": "function",
			"function": map[string]interface{}{
				"name":        "get_insider_transactions",
				"description": "Retrieve insider transaction information about a company",
				"parameters": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"ticker": map[string]interface{}{"type": "string", "description": "Ticker symbol"},
					},
					"required": []string{"ticker"},
				},
			},
		},
		{
			"type": "function",
			"function": map[string]interface{}{
				"name":        "get_balance_sheet",
				"description": "Retrieve balance sheet data for a given ticker symbol",
				"parameters": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"ticker":    map[string]interface{}{"type": "string", "description": "Ticker symbol"},
						"freq":      map[string]interface{}{"type": "string", "description": "Reporting frequency: annual/quarterly", "default": "quarterly"},
						"curr_date": map[string]interface{}{"type": "string", "description": "Current date in yyyy-mm-dd format"},
					},
					"required": []string{"ticker"},
				},
			},
		},
		{
			"type": "function",
			"function": map[string]interface{}{
				"name":        "get_cashflow",
				"description": "Retrieve cash flow statement data for a given ticker symbol",
				"parameters": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"ticker":    map[string]interface{}{"type": "string", "description": "Ticker symbol"},
						"freq":      map[string]interface{}{"type": "string", "description": "Reporting frequency: annual/quarterly", "default": "quarterly"},
						"curr_date": map[string]interface{}{"type": "string", "description": "Current date in yyyy-mm-dd format"},
					},
					"required": []string{"ticker"},
				},
			},
		},
		{
			"type": "function",
			"function": map[string]interface{}{
				"name":        "get_income_statement",
				"description": "Retrieve income statement data for a given ticker symbol",
				"parameters": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"ticker":    map[string]interface{}{"type": "string", "description": "Ticker symbol"},
						"freq":      map[string]interface{}{"type": "string", "description": "Reporting frequency: annual/quarterly", "default": "quarterly"},
						"curr_date": map[string]interface{}{"type": "string", "description": "Current date in yyyy-mm-dd format"},
					},
					"required": []string{"ticker"},
				},
			},
		},
		{
			"type": "function",
			"function": map[string]interface{}{
				"name":        "get_verified_market_snapshot",
				"description": "Retrieve a deterministic verification snapshot of latest OHLCV, technical indicators, and recent closes. Use as source of truth before making exact claims about price levels, indicators, or historical comparisons.",
				"parameters": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"symbol":         map[string]interface{}{"type": "string", "description": "Ticker symbol of the company"},
						"curr_date":      map[string]interface{}{"type": "string", "description": "The current trading date, YYYY-mm-dd"},
						"look_back_days": map[string]interface{}{"type": "integer", "description": "Number of recent trading rows to include for sanity-checking", "default": 30},
					},
					"required": []string{"symbol", "curr_date"},
				},
			},
		},
	}
}
