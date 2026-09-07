package qa

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/huzilin/trade/llm"
)

// IntentClassifier 意图分类器
type IntentClassifier struct {
	llmClient *llm.OpenAIClientV2
	model     string
}

// NewIntentClassifier 创建意图分类器
func NewIntentClassifier(client *llm.OpenAIClientV2, model string) *IntentClassifier {
	return &IntentClassifier{llmClient: client, model: model}
}

// Classify 分类用户意图
// 优先使用关键词匹配，匹配不到则调用 LLM
func (c *IntentClassifier) Classify(question string, contextText string) (*IntentResult, error) {
	// 1. 关键词快速匹配
	if result := c.keywordMatch(question); result != nil {
		return result, nil
	}

	// 2. LLM 分类
	if c.llmClient != nil {
		return c.llmClassify(question, contextText)
	}

	// 3. 默认：通用问答
	return &IntentResult{
		Intent:     IntentGeneral,
		Confidence: 0.5,
		Reasoning:  "default fallback",
	}, nil
}

// keywordMatch 关键词匹配
func (c *IntentClassifier) keywordMatch(question string) *IntentResult {
	q := strings.ToLower(question)

	keywordMap := map[IntentType][]string{
		IntentDataQuery:      {"市盈率", "市净率", "pe", "pb", "eps", "roe", "毛利率", "净利率", "市值", "营收", "报表", "财务", "资产负债", "现金流", "利润", "收入", "数据", "多少", "价格", "行情", "走势", "k线", "技术指标"},
		IntentRiskInquiry:    {"风险", "亏损", "止损", "回撤", "波动", "不确定性", "最大回撤", "担心", "危险"},
		IntentComparison:     {"对比", "比较", "相比", "同行业", "同行", "竞争", "哪个好", "vs"},
		IntentStrategyAdvice: {"怎么操作", "该怎么办", "建议", "策略", "仓位", "止损位", "止盈", "持有", "加仓", "减仓", "卖出", "买入"},
		IntentMarketInsight:  {"市场", "大盘", "趋势", "行业", "板块", "政策", "宏观", "gdp", "cpi", "利率", "经济"},
	}

	for intent, keywords := range keywordMap {
		for _, kw := range keywords {
			if strings.Contains(q, kw) {
				return &IntentResult{
					Intent:     intent,
					Confidence: 0.8,
					Reasoning:  fmt.Sprintf("keyword matched: %s", kw),
				}
			}
		}
	}

	return nil
}

// llmClassify 使用 LLM 进行意图分类
func (c *IntentClassifier) llmClassify(question string, contextText string) (*IntentResult, error) {
	prompt := c.buildClassifyPrompt(question, contextText)

	messages := []*llm.ChatMessage{
		{Role: "system", Content: "你是一个精确的意图分类器。只返回 JSON，不要其他内容。"},
		{Role: "user", Content: prompt},
	}

	options := &llm.ChatOptions{
		Model:       c.model,
		Temperature: 0.1,
		MaxTokens:   300,
	}

	resp, err := c.llmClient.Chat(context.Background(), messages, options)
	if err != nil {
		return nil, err
	}

	return parseIntentResponse(resp.Content)
}

func (c *IntentClassifier) buildClassifyPrompt(question, contextText string) string {
	intentList := `report_interpretation: 对分析报告内容的解释
data_query: 查询具体的数据指标（市盈率、K线、财务数据等）
comparison: 多只股票或同行业横向对比
risk_inquiry: 了解投资风险
strategy_advice: 个性化的投资策略和操作建议
market_insight: 宏观市场趋势解读
general: 其他通用问题`

	return fmt.Sprintf(`判断以下用户问题的意图类型。

分析上下文摘要：
%s

用户问题：%s

意图类型：
%s

请返回 JSON：{"intent": "<intent_type>", "confidence": 0.95, "reasoning": "<简短说明>"}`, truncateText(contextText, 500), question, intentList)
}

func parseIntentResponse(content string) (*IntentResult, error) {
	// 尝试提取 JSON
	content = extractJSON(content)

	var result IntentResult
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		// JSON 解析失败，默认通用问答
		return &IntentResult{
			Intent:     IntentGeneral,
			Confidence: 0.5,
			Reasoning:  "parse fallback",
		}, nil
	}

	if !isValidIntent(result.Intent) {
		result.Intent = IntentGeneral
	}

	return &result, nil
}

// extractJSON 从 LLM 响应中提取 JSON
func extractJSON(content string) string {
	content = strings.TrimSpace(content)
	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")
	if start >= 0 && end > start {
		return content[start : end+1]
	}
	return content
}

func isValidIntent(intent IntentType) bool {
	switch intent {
	case IntentReportInterpretation, IntentDataQuery, IntentComparison,
		IntentRiskInquiry, IntentStrategyAdvice, IntentMarketInsight, IntentGeneral:
		return true
	}
	return false
}

func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}
