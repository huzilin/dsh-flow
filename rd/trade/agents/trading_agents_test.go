package agents

import (
	"testing"

	"github.com/huzilin/trade/agents/states"
	"github.com/stretchr/testify/assert"
)

func TestUnit_RiskDebateStateInitialization(t *testing.T) {
	state := &states.RiskDebateState{}

	assert.Equal(t, "", state.History)
	assert.Equal(t, "", state.AggressiveHistory)
	assert.Equal(t, "", state.ConservativeHistory)
	assert.Equal(t, "", state.NeutralHistory)
	assert.Equal(t, "", state.CurrentAggressiveResponse)
	assert.Equal(t, "", state.CurrentConservativeResponse)
	assert.Equal(t, "", state.CurrentNeutralResponse)
	assert.Equal(t, "", state.LatestSpeaker)
	assert.Equal(t, "", state.JudgeDecision)
	assert.Equal(t, int64(0), state.Count)
}

func TestUnit_InvestDebateStateInitialization(t *testing.T) {
	state := &states.InvestDebateState{}

	assert.Equal(t, "", state.History)
	assert.Equal(t, "", state.BullHistory)
	assert.Equal(t, "", state.BearHistory)
	assert.Equal(t, "", state.CurrentResponse)
	assert.Equal(t, "", state.JudgeDecision)
	assert.Equal(t, int64(0), state.Count)
}

func TestUnit_AgentStateInitialization(t *testing.T) {
	state := &states.AgentState{}

	assert.Nil(t, state.RiskDebateState)
	assert.Nil(t, state.InvestmentDebateState)
	assert.Equal(t, "", state.CompanyOfInterest)
	assert.Equal(t, "", state.AssetType)
	assert.Equal(t, "", state.InstrumentContext)
	assert.Equal(t, "", state.TradeDate)
	assert.Equal(t, "", state.Sender)
	assert.Equal(t, "", state.MarketReport)
	assert.Equal(t, "", state.SentimentReport)
	assert.Equal(t, "", state.NewsReport)
	assert.Equal(t, "", state.FundamentalsReport)
	assert.Equal(t, "", state.InvestmentPlan)
	assert.Equal(t, "", state.TraderInvestmentPlan)
	assert.Equal(t, "", state.FinalTradeDecision)
	assert.Equal(t, "", state.PastContext)
}

func TestUnit_AgentStateWithDebateStates(t *testing.T) {
	state := &states.AgentState{
		CompanyOfInterest: "600519",
		AssetType:         "stock",
		InstrumentContext: "Test context",
		RiskDebateState: &states.RiskDebateState{
			History:       "Test history",
			LatestSpeaker: "Aggressive",
			Count:         1,
		},
		InvestmentDebateState: &states.InvestDebateState{
			History: "Investment debate history",
			Count:   2,
		},
	}

	assert.Equal(t, "600519", state.CompanyOfInterest)
	assert.Equal(t, "stock", state.AssetType)
	assert.NotNil(t, state.RiskDebateState)
	assert.NotNil(t, state.InvestmentDebateState)
	assert.Equal(t, "Test history", state.RiskDebateState.History)
	assert.Equal(t, "Aggressive", state.RiskDebateState.LatestSpeaker)
	assert.Equal(t, int64(1), state.RiskDebateState.Count)
	assert.Equal(t, "Investment debate history", state.InvestmentDebateState.History)
	assert.Equal(t, int64(2), state.InvestmentDebateState.Count)
}
