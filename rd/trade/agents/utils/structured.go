package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/huzilin/trade/llm"
	"go.uber.org/zap"
)

// StructuredResult 结构化结果包装器，包含解析后的数据或原始文本
type StructuredResult[T any] struct {
	Data         *T
	Raw          string
	IsStructured bool
}

// TryStructured 尝试将 LLM 响应解析为结构化数据，失败时回退到原始文本
// 适用于 Portfolio Manager、Trader、Research Manager 等需要结构化输出的代理
func TryStructured[T any](
	ctx context.Context,
	llmClient *llm.OpenAIClientV2,
	messages []*llm.ChatMessage,
	options *llm.ChatOptions,
	agentName string,
) *StructuredResult[T] {
	resp, err := llmClient.Chat(ctx, messages, options)
	if err != nil {
		llm.Logger().Error(fmt.Sprintf("%s LLM call failed", agentName), zap.Error(err))
		return &StructuredResult[T]{Raw: "", IsStructured: false}
	}

	var data T

	// 直接尝试解析 JSON
	if err := json.Unmarshal([]byte(resp.Content), &data); err == nil {
		return &StructuredResult[T]{Data: &data, Raw: resp.Content, IsStructured: true}
	}

	// 尝试从 markdown code block 中提取 JSON
	if jsonStr := extractJSONFromMarkdown(resp.Content); jsonStr != "" {
		if err := json.Unmarshal([]byte(jsonStr), &data); err == nil {
			return &StructuredResult[T]{Data: &data, Raw: resp.Content, IsStructured: true}
		}
	}

	llm.Logger().Warn(
		fmt.Sprintf("%s structured output parsing failed, using free text", agentName),
		zap.String("content_preview", truncate(resp.Content, 200)),
	)
	return &StructuredResult[T]{Raw: resp.Content, IsStructured: false}
}

// extractJSONFromMarkdown 从 markdown code block 中提取 JSON 字符串
func extractJSONFromMarkdown(content string) string {
	// 查找 ```json ... ``` 或 ``` ... ```
	markers := []string{"```json", "```"}
	for _, marker := range markers {
		idx := strings.Index(content, marker)
		if idx == -1 {
			continue
		}
		start := idx + len(marker)
		// 跳过换行符
		for start < len(content) && (content[start] == '\n' || content[start] == '\r') {
			start++
		}
		endMarker := "```"
		end := strings.Index(content[start:], endMarker)
		if end == -1 {
			continue
		}
		return strings.TrimSpace(content[start : start+end])
	}

	// 查找 { ... } 或 [ ... ] 包裹的内容
	start := strings.Index(content, "{")
	if start == -1 {
		start = strings.Index(content, "[")
	}
	if start == -1 {
		return ""
	}
	end := strings.LastIndex(content, "}")
	if end == -1 {
		end = strings.LastIndex(content, "]")
	}
	if end == -1 || end <= start {
		return ""
	}
	return strings.TrimSpace(content[start : end+1])
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
