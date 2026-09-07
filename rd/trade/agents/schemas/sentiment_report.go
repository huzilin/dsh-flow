package schemas

// SentimentReport 情绪分析师生成的结构化情绪报告
type SentimentReport struct {
	OverallBand  SentimentBand `json:"overall_band" description:"总体情绪方向"`
	OverallScore float64       `json:"overall_score" description:"情绪强度，0-10"`
	Confidence   Confidence    `json:"confidence" description:"评估置信度"`
	Narrative    string        `json:"narrative" description:"完整情绪报告"`
}

// RenderSentimentReport 将 SentimentReport 渲染为 markdown
func RenderSentimentReport(report *SentimentReport) string {
	return JoinLines(
		"**Overall Sentiment:** **"+string(report.OverallBand)+"** (Score: "+
			FormatFloat(report.OverallScore)+"/10)",
		"**Confidence:** "+Capitalize(string(report.Confidence)),
		"",
		report.Narrative,
	)
}
