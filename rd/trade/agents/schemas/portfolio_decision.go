package schemas

// PortfolioDecision 投资组合经理生成的结构化决策
type PortfolioDecision struct {
	Rating           PortfolioRating `json:"rating" description:"最终仓位评级"`
	ExecutiveSummary string          `json:"executive_summary" description:"简洁的行动计划"`
	InvestmentThesis string          `json:"investment_thesis" description:"基于分析师辩论的详细推理"`
	PriceTarget      *float64        `json:"price_target,omitempty" description:"目标价格"`
	TimeHorizon      *string         `json:"time_horizon,omitempty" description:"建议持有期限"`
}

// RenderPMDecision 将 PortfolioDecision 渲染为 markdown
func RenderPMDecision(decision *PortfolioDecision) string {
	var parts []string

	parts = append(parts, "**Rating**: "+string(decision.Rating))
	parts = append(parts, "")
	parts = append(parts, "**Executive Summary**: "+decision.ExecutiveSummary)
	parts = append(parts, "")
	parts = append(parts, "**Investment Thesis**: "+decision.InvestmentThesis)

	if decision.PriceTarget != nil {
		parts = append(parts, "")
		parts = append(parts, "**Price Target**: "+Float64ToString(*decision.PriceTarget))
	}

	if decision.TimeHorizon != nil {
		parts = append(parts, "")
		parts = append(parts, "**Time Horizon**: "+*decision.TimeHorizon)
	}

	return JoinLines(parts...)
}
