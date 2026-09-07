package qa

import (
	"sort"
	"strings"
)

// RetrievedDoc 检索到的文档片段
type RetrievedDoc struct {
	Source  string  // 来源（market_report / sentiment_report / news_report 等）
	Content string  // 内容片段
	Score   float64 // 相关性分数
}

// Retriever 简易检索器，基于关键词匹配（初版，后续升级 RAG）
type Retriever struct {
	sections []docSection
}

type docSection struct {
	source  string
	content string
}

// NewRetriever 创建检索器
func NewRetriever() *Retriever {
	return &Retriever{}
}

// IndexAnalysisContext 索引分析上下文
func (r *Retriever) IndexAnalysisContext(ctx AnalysisContext) {
	r.sections = nil

	if ctx.MarketReport != "" {
		r.sections = append(r.sections, docSection{source: "market_report", content: ctx.MarketReport})
	}
	if ctx.SentimentReport != "" {
		r.sections = append(r.sections, docSection{source: "sentiment_report", content: ctx.SentimentReport})
	}
	if ctx.NewsReport != "" {
		r.sections = append(r.sections, docSection{source: "news_report", content: ctx.NewsReport})
	}
	if ctx.FundamentalsReport != "" {
		r.sections = append(r.sections, docSection{source: "fundamentals_report", content: ctx.FundamentalsReport})
	}
	if ctx.InvestmentPlan != "" {
		r.sections = append(r.sections, docSection{source: "investment_plan", content: ctx.InvestmentPlan})
	}
	if ctx.FinalDecision != "" {
		r.sections = append(r.sections, docSection{source: "final_decision", content: ctx.FinalDecision})
	}
}

// Retrieve 检索最相关的文档片段
// 使用关键词匹配 + 简单 TF 评分（初版实现）
func (r *Retriever) Retrieve(query string, topK int) []RetrievedDoc {
	if len(r.sections) == 0 {
		return nil
	}

	queryWords := tokenize(query)
	if len(queryWords) == 0 {
		return nil
	}

	var results []RetrievedDoc

	for _, section := range r.sections {
		// 将长文本分段
		chunks := splitChunks(section.content, 500)
		for _, chunk := range chunks {
			score := computeTFScore(chunk, queryWords)
			if score > 0 {
				results = append(results, RetrievedDoc{
					Source:  section.source,
					Content: chunk,
					Score:   score,
				})
			}
		}
	}

	// 按分数降序排序
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// 返回 topK 个结果
	if len(results) > topK {
		results = results[:topK]
	}

	return results
}

// BuildContextText 将检索结果拼接为上下文文本
func BuildContextText(docs []RetrievedDoc) string {
	if len(docs) == 0 {
		return ""
	}

	var builder strings.Builder
	sourceLabels := map[string]string{
		"market_report":       "市场分析报告",
		"sentiment_report":    "情绪分析报告",
		"news_report":         "新闻分析报告",
		"fundamentals_report": "基本面分析报告",
		"investment_plan":     "投资计划",
		"final_decision":      "最终决策",
	}

	for i, doc := range docs {
		label, ok := sourceLabels[doc.Source]
		if !ok {
			label = doc.Source
		}
		builder.WriteString("【来源：")
		builder.WriteString(label)
		builder.WriteString("】\n")
		builder.WriteString(doc.Content)
		if i < len(docs)-1 {
			builder.WriteString("\n\n")
		}
	}

	return builder.String()
}

// tokenize 简单分词（按空格和标点分割）
func tokenize(text string) []string {
	text = strings.ToLower(text)
	// 按常见分隔符分割
	words := strings.FieldsFunc(text, func(r rune) bool {
		return r == ' ' || r == '\n' || r == '\t' ||
			r == '，' || r == '。' || r == '？' || r == '！' ||
			r == ',' || r == '.' || r == '?' || r == '!' ||
			r == '；' || r == '：' || r == '"' || r == '\''
	})

	// 过滤短词和停用词
	var filtered []string
	stopWords := map[string]bool{
		"的": true, "是": true, "了": true, "在": true, "和": true,
		"the": true, "is": true, "a": true, "an": true, "of": true,
		"to": true, "in": true, "for": true, "on": true, "with": true,
	}
	for _, w := range words {
		if len(w) > 1 && !stopWords[w] {
			filtered = append(filtered, w)
		}
	}
	return filtered
}

// splitChunks 将文本按句分割为多个 chunk
func splitChunks(text string, maxLen int) []string {
	if len(text) <= maxLen {
		return []string{text}
	}

	var chunks []string
	runes := []rune(text)

	start := 0
	for start < len(runes) {
		end := start + maxLen
		if end > len(runes) {
			end = len(runes)
		}

		// 尝试在句号或换行处断开
		chunk := string(runes[start:end])
		lastPeriod := maxLastIndex(chunk, '。', '\n', '.')
		if lastPeriod > maxLen/2 {
			end = start + lastPeriod + 1
			chunk = string(runes[start:end])
		}

		chunks = append(chunks, chunk)
		start = end
	}

	return chunks
}

// computeTFScore 计算词频得分
func computeTFScore(text string, queryWords []string) float64 {
	textLower := strings.ToLower(text)
	var score float64
	for _, word := range queryWords {
		count := strings.Count(textLower, word)
		if count > 0 {
			score += float64(count)
		}
	}
	return score
}

func maxLastIndex(s string, chars ...rune) int {
	maxIdx := -1
	for _, ch := range chars {
		idx := strings.LastIndexFunc(s, func(r rune) bool { return r == ch })
		if idx > maxIdx {
			maxIdx = idx
		}
	}
	return maxIdx
}
