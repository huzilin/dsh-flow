package schemas

// TraderProposal 交易员生成的结构化交易提案
type TraderProposal struct {
	Action         TraderAction `json:"action" description:"交易方向"`
	Reasoning     string        `json:"reasoning" description:"行动理由"`
	EntryPrice    *float64      `json:"entry_price,omitempty" description:"入场目标价"`
	StopLoss      *float64      `json:"stop_loss,omitempty" description:"止损价格"`
	PositionSizing *string       `json:"position_sizing,omitempty" description:"仓位指导"`
}

// RenderTraderProposal 将 TraderProposal 渲染为 markdown
// 保留最后的 "FINAL TRANSACTION PROPOSAL: **BUY/HOLD/SELL**" 行以保持向后兼容
func RenderTraderProposal(proposal *TraderProposal) string {
	var parts []string

	parts = append(parts, "**Action**: "+string(proposal.Action))
	parts = append(parts, "")
	parts = append(parts, "**Reasoning**: "+proposal.Reasoning)

	if proposal.EntryPrice != nil {
		parts = append(parts, "")
		parts = append(parts, "**Entry Price**: "+Float64ToString(*proposal.EntryPrice))
	}

	if proposal.StopLoss != nil {
		parts = append(parts, "")
		parts = append(parts, "**Stop Loss**: "+Float64ToString(*proposal.StopLoss))
	}

	if proposal.PositionSizing != nil {
		parts = append(parts, "")
		parts = append(parts, "**Position Sizing**: "+*proposal.PositionSizing)
	}

	parts = append(parts, "")
	parts = append(parts, "FINAL TRANSACTION PROPOSAL: **"+string(proposal.Action)+"**")

	return JoinLines(parts...)
}
