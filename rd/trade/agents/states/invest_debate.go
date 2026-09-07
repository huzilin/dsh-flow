package states

// InvestDebateState 研究团队辩论状态
type InvestDebateState struct {
	BullHistory      string `json:"bull_history" description:"看涨辩论历史"`
	BearHistory      string `json:"bear_history" description:"看跌辩论历史"`
	History          string `json:"history" description:"对话历史"`
	CurrentResponse  string `json:"current_response" description:"最新回复"`
	JudgeDecision    string `json:"judge_decision" description:"最终裁决"`
	Count            int64  `json:"count" description:"当前对话轮数"`
}
