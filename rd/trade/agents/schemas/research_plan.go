package schemas

// ResearchPlan 研究经理生成的结构化投资计划
type ResearchPlan struct {
	Recommendation   PortfolioRating `json:"recommendation" description:"投资建议"`
	Rationale       string          `json:"rationale" description:"辩论双方的关键论点总结"`
	StrategicActions string          `json:"strategic_actions" description:"交易员实施建议的具体步骤"`
}

// RenderResearchPlan 将 ResearchPlan 渲染为 markdown
func RenderResearchPlan(plan *ResearchPlan) string {
	return JoinLines(
		"**Recommendation**: "+string(plan.Recommendation),
		"",
		"**Rationale**: "+plan.Rationale,
		"",
		"**Strategic Actions**: "+plan.StrategicActions,
	)
}
