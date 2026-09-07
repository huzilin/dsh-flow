package report_finalizer

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/huzilin/trade/llm"
	"go.uber.org/zap"
)

var reportFinalizationSpecs = []struct {
	sourceKey   string
	targetKey   string
	reportLabel string
}{
	{"market_report", "final_market_report", "market analysis report"},
	{"sentiment_report", "final_sentiment_report", "sentiment analysis report"},
	{"news_report", "final_news_report", "news analysis report"},
	{"fundamentals_report", "final_fundamentals_report", "fundamentals analysis report"},
	{"investment_plan", "final_investment_plan_report", "research manager investment plan"},
	{"trader_investment_plan", "final_trader_investment_plan_report", "trader execution plan"},
}

var englishWordRE = regexp.MustCompile(`\b[A-Za-z][A-Za-z/+_-]{2,}\b`)
var englishLeakPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?im)^(excellent!|now let me|let me compile|here is|based on a comprehensive)`),
	regexp.MustCompile(`(?m)^# .*\b(report|analysis|plan|decision)\b`),
	regexp.MustCompile(`(?m)^## [IVX]+\.`),
}

// ReportFinalizer 报告整理器
type ReportFinalizer struct {
	llm *llm.OpenAIClientV2
}

// NewReportFinalizer 创建报告整理器
func NewReportFinalizer(llmClient *llm.OpenAIClientV2) *ReportFinalizer {
	return &ReportFinalizer{
		llm: llmClient,
	}
}

// needsLanguageCleanup 判断报告是否明显泄漏了与最终语言不一致的英文内容
func needsLanguageCleanup(text string, finalLanguage string) bool {
	if text == "" {
		return false
	}

	if strings.TrimSpace(finalLanguage) == "" || strings.ToLower(strings.TrimSpace(finalLanguage)) == "english" {
		return false
	}

	for _, pattern := range englishLeakPatterns {
		if pattern.MatchString(text) {
			return true
		}
	}

	return len(englishWordRE.FindAllString(text, -1)) >= 20
}

// rewriteReportInFinalLanguage 将泄漏英文的最终报告重写为中文
func (rf *ReportFinalizer) rewriteReportInFinalLanguage(reportText, reportLabel string) string {
	if !needsLanguageCleanup(reportText, "Chinese") {
		return reportText
	}

	prompt := fmt.Sprintf(`将以下%s重写为中文供最终用户阅读。

要求:
- 保留所有事实、数字、日期、股票代码和Markdown表格。
- 将所有叙述性文本、标题、列表标签和摘要翻译为中文。
- 删除英文工作流叙述或元评论。
- 仅在明确需要时保留机器可读标记，包括股票代码以及Buy / Overweight / Hold / Underweight / Sell（如果已出现）。
- 仅返回清理后的Markdown报告正文。

报告:
%s`, reportLabel, reportText)

	resp, err := rf.llm.Chat(nil, []*llm.ChatMessage{{Role: "user", Content: prompt}}, &llm.ChatOptions{})
	if err != nil {
		llm.Logger().Error("最终报告清理失败", zap.String("report_label", reportLabel), zap.Error(err))
		return reportText
	}

	cleaned := strings.TrimSpace(resp.Content)
	if cleaned != "" {
		return cleaned
	}

	return reportText
}

// ReportFinalizerNode 将中间态内容整理为最终对外报告
func (rf *ReportFinalizer) ReportFinalizerNode(state map[string]string) (map[string]string, error) {
	finalReports := make(map[string]string)

	for _, spec := range reportFinalizationSpecs {
		sourceText := state[spec.sourceKey]
		finalReports[spec.targetKey] = rf.rewriteReportInFinalLanguage(sourceText, spec.reportLabel)
	}

	finalReports["final_trade_decision_report"] = rf.rewriteReportInFinalLanguage(
		state["final_trade_decision"],
		"portfolio manager final decision report",
	)

	return finalReports, nil
}
