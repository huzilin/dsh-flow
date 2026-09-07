package states

// RiskDebateState 风险管理团队辩论状态
type RiskDebateState struct {
	AggressiveHistory      string `json:"aggressive_history" description:"激进代理对话历史"`
	ConservativeHistory   string `json:"conservative_history" description:"保守代理对话历史"`
	NeutralHistory        string `json:"neutral_history" description:"中立代理对话历史"`
	History                string `json:"history" description:"对话历史"`
	LatestSpeaker          string `json:"latest_speaker" description:"最后发言的分析师"`
	CurrentAggressiveResponse string `json:"current_aggressive_response" description:"激进分析师最新回复"`
	CurrentConservativeResponse string `json:"current_conservative_response" description:"保守分析师最新回复"`
	CurrentNeutralResponse string `json:"current_neutral_response" description:"中立分析师最新回复"`
	JudgeDecision          string `json:"judge_decision" description:"裁决者决策"`
	Count                  int64  `json:"count" description:"当前对话轮数"`
}
