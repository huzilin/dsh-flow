package agents

import (
	"context"
	"strings"

	"github.com/huzilin/trade/agents/analysts/fundamentals_analyst"
	"github.com/huzilin/trade/agents/analysts/market_analyst"
	"github.com/huzilin/trade/agents/analysts/news_analyst"
	"github.com/huzilin/trade/agents/analysts/sentiment_analyst"
	"github.com/huzilin/trade/agents/managers/portfolio_manager"
	"github.com/huzilin/trade/agents/managers/research_manager"
	"github.com/huzilin/trade/agents/researchers/bear_researcher"
	"github.com/huzilin/trade/agents/researchers/bull_researcher"
	"github.com/huzilin/trade/agents/risk_mgmt/aggressive_debator"
	"github.com/huzilin/trade/agents/risk_mgmt/conservative_debator"
	"github.com/huzilin/trade/agents/risk_mgmt/neutral_debator"
	"github.com/huzilin/trade/agents/states"
	"github.com/huzilin/trade/agents/trader/trader"
	"github.com/huzilin/trade/agents/utils"
	"github.com/huzilin/trade/llm"
	"github.com/huzilin/trade/types"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// TradingAgentsConfig TradingAgents 配置
type TradingAgentsConfig struct {
	MaxDebateRounds      int64  `json:"MaxDebateRounds"`
	MaxRiskDiscussRounds int64  `json:"MaxRiskDiscussRounds"`
	ResearchDepth        int64  `json:"ResearchDepth"`
	ReportOutputDir      string `json:"ReportOutputDir"`
}

// NewTradingAgentsConfig 创建 TradingAgents 配置
func NewTradingAgentsConfig() *TradingAgentsConfig {
	return &TradingAgentsConfig{
		MaxDebateRounds:      1,
		MaxRiskDiscussRounds: 1,
		ResearchDepth:        30,
	}
}

// TradingAgentsAgentOptions TradingAgents 代理选项
type TradingAgentsAgentOptions struct {
	Config           *TradingAgentsConfig
	QuickLLM         *llm.OpenAIClientV2
	DeepLLM          *llm.OpenAIClientV2
	SelectedAnalysts []string
	Debug            bool
}

// TradingAgentsAgent TradingAgents 代理
type TradingAgentsAgent struct {
	config           *TradingAgentsConfig
	quickLLM         *llm.OpenAIClientV2
	deepLLM          *llm.OpenAIClientV2
	selectedAnalysts []string
	debug            bool

	marketAnalyst       *market_analyst.MarketAnalyst
	sentimentAnalyst    *sentiment_analyst.SentimentAnalyst
	newsAnalyst         *news_analyst.NewsAnalyst
	fundamentalsAnalyst *fundamentals_analyst.FundamentalsAnalyst
	bullResearcher      *bull_researcher.BullResearcher
	bearResearcher      *bear_researcher.BearResearcher
	researchManager     *research_manager.ResearchManager
	trader              *trader.Trader
	aggressiveDebator   *aggressive_debator.AggressiveDebator
	conservativeDebator *conservative_debator.ConservativeDebator
	neutralDebator      *neutral_debator.NeutralDebator
	portfolioManager    *portfolio_manager.PortfolioManager
}

// NewTradingAgentsAgent 创建 TradingAgents 代理
func NewTradingAgentsAgent(opts *TradingAgentsAgentOptions) *TradingAgentsAgent {
	if opts == nil {
		opts = &TradingAgentsAgentOptions{}
	}
	if opts.Config == nil {
		opts.Config = NewTradingAgentsConfig()
	}
	if opts.SelectedAnalysts == nil || len(opts.SelectedAnalysts) == 0 {
		opts.SelectedAnalysts = []string{"market", "social", "news", "fundamentals"}
	}

	agent := &TradingAgentsAgent{
		config:           opts.Config,
		quickLLM:         opts.QuickLLM,
		deepLLM:          opts.DeepLLM,
		selectedAnalysts: opts.SelectedAnalysts,
		debug:            opts.Debug,
	}

	agent.initAgents()

	return agent
}

func (a *TradingAgentsAgent) initAgents() {
	if a.quickLLM != nil {
		a.marketAnalyst = market_analyst.NewMarketAnalyst(a.quickLLM, "")
		a.sentimentAnalyst = sentiment_analyst.NewSentimentAnalyst(a.quickLLM, "")
		a.newsAnalyst = news_analyst.NewNewsAnalyst(a.quickLLM, "")
		a.fundamentalsAnalyst = fundamentals_analyst.NewFundamentalsAnalyst(a.quickLLM, "")
		a.bullResearcher = bull_researcher.NewBullResearcher(a.quickLLM, "")
		a.bearResearcher = bear_researcher.NewBearResearcher(a.quickLLM, "")
		a.trader = trader.NewTrader(a.quickLLM, "")
		a.aggressiveDebator = aggressive_debator.NewAggressiveDebator(a.quickLLM, "")
		a.conservativeDebator = conservative_debator.NewConservativeDebator(a.quickLLM, "")
		a.neutralDebator = neutral_debator.NewNeutralDebator(a.quickLLM, "")
	}

	if a.deepLLM != nil {
		a.researchManager = research_manager.NewResearchManager(a.deepLLM, "")
		a.portfolioManager = portfolio_manager.NewPortfolioManager(a.deepLLM, "")
	}
}

// Run 运行代理（实现 agentcore.BaseAgent 接口）
func (a *TradingAgentsAgent) Run(request any, context any) (any, error) {
	req, ok := request.(*types.AgentRunRequest)
	if !ok {
		llm.Logger().Error("Invalid request type")
		return types.AgentRunResult{}, errors.New("invalid request type")
	}

	ctx, ok := context.(*types.AgentExecutionContext)
	if !ok {
		llm.Logger().Error("Invalid context type")
		return types.AgentRunResult{}, errors.New("invalid context type")
	}

	return a.run(req, ctx)
}

// run 运行代理（内部实现）
// 执行完整的 TradingAgents 工作流，包括：
// 1. 创建初始状态
// 2. 执行分析师阶段（市场、情绪、新闻、基本面）
// 3. 执行研究员辩论阶段（多头/空头辩论）
// 4. 执行交易员阶段（生成交易计划）
// 5. 执行风险辩论阶段（激进/保守/中立辩论）
// 6. 执行投资组合经理阶段（生成最终决策）
// 参数：
//
//	request: 代理运行请求，包含股票代码和交易日期
//	context: 代理执行上下文
//
// 返回：
//
//	AgentRunResult: 代理运行结果，包含决策和各类报告
//	error: 执行过程中的错误
func (a *TradingAgentsAgent) run(request *types.AgentRunRequest, context *types.AgentExecutionContext) (types.AgentRunResult, error) {
	llm.Logger().Info("Starting TradingAgents workflow",
		zap.String("symbol", request.Symbol),
		zap.String("trade_date", request.TradeDate))

	if a.marketAnalyst == nil || a.researchManager == nil || a.portfolioManager == nil {
		return types.AgentRunResult{}, errors.New("LLM clients not initialized")
	}

	state := a.createInitialState(request.Symbol, request.TradeDate)

	if err := a.executeAnalysts(state); err != nil {
		llm.Logger().Error("Analysts execution failed", zap.Error(err))
		return types.AgentRunResult{}, err
	}

	if err := a.executeResearchDebate(state); err != nil {
		llm.Logger().Error("Research debate failed", zap.Error(err))
		return types.AgentRunResult{}, err
	}

	if err := a.executeTrader(state); err != nil {
		llm.Logger().Error("Trader execution failed", zap.Error(err))
		return types.AgentRunResult{}, err
	}

	if err := a.executeRiskDebate(state); err != nil {
		llm.Logger().Error("Risk debate failed", zap.Error(err))
		return types.AgentRunResult{}, err
	}

	if err := a.executePortfolioManager(state); err != nil {
		llm.Logger().Error("Portfolio manager execution failed", zap.Error(err))
		return types.AgentRunResult{}, err
	}

	return a.buildResult(state, request.Symbol, request.TradeDate), nil
}

// createInitialState 创建初始状态
func (a *TradingAgentsAgent) createInitialState(symbol, tradeDate string) *states.AgentState {
	assetType := "stock"
	if ctx, ok := symbolContextMap[symbol]; ok {
		assetType = ctx
	}

	return &states.AgentState{
		CompanyOfInterest: symbol,
		AssetType:         assetType,
		TradeDate:         tradeDate,
		InstrumentContext: utils.BuildInstrumentContext(symbol, assetType),
		InvestmentDebateState: &states.InvestDebateState{
			BullHistory:     "",
			BearHistory:     "",
			History:         "",
			CurrentResponse: "",
			JudgeDecision:   "",
			Count:           0,
		},
		RiskDebateState: &states.RiskDebateState{
			AggressiveHistory:           "",
			ConservativeHistory:         "",
			NeutralHistory:              "",
			History:                     "",
			LatestSpeaker:               "",
			CurrentAggressiveResponse:   "",
			CurrentConservativeResponse: "",
			CurrentNeutralResponse:      "",
			JudgeDecision:               "",
			Count:                       0,
		},
	}
}

// executeAnalysts 执行分析师阶段
func (a *TradingAgentsAgent) executeAnalysts(state *states.AgentState) error {
	for _, analystType := range a.selectedAnalysts {
		llm.Logger().Info("Executing analyst", zap.String("type", analystType))

		switch analystType {
		case "market":
			if a.marketAnalyst == nil {
				return errors.New("market analyst not initialized")
			}
			if err := a.executeAnalystWithTools(func() (*states.AgentState, error) {
				return a.marketAnalyst.MarketAnalystNode(state)
			}, func(content string) { state.MarketReport = content }, state); err != nil {
				return err
			}
		case "social":
			if a.sentimentAnalyst == nil {
				return errors.New("sentiment analyst not initialized")
			}
			if err := a.executeAnalystWithTools(func() (*states.AgentState, error) {
				return a.sentimentAnalyst.SentimentAnalystNode(state)
			}, func(content string) { state.SentimentReport = content }, state); err != nil {
				return err
			}
		case "news":
			if a.newsAnalyst == nil {
				return errors.New("news analyst not initialized")
			}
			if err := a.executeAnalystWithTools(func() (*states.AgentState, error) {
				return a.newsAnalyst.NewsAnalystNode(state)
			}, func(content string) { state.NewsReport = content }, state); err != nil {
				return err
			}
		case "fundamentals":
			if a.fundamentalsAnalyst == nil {
				return errors.New("fundamentals analyst not initialized")
			}
			if err := a.executeAnalystWithTools(func() (*states.AgentState, error) {
				return a.fundamentalsAnalyst.FundamentalsAnalystNode(state)
			}, func(content string) { state.FundamentalsReport = content }, state); err != nil {
				return err
			}
		}
	}
	return nil
}

// executeAnalystWithTools 执行分析师并处理工具调用循环
func (a *TradingAgentsAgent) executeAnalystWithTools(
	analystFunc func() (*states.AgentState, error),
	setReportFunc func(string),
	state *states.AgentState,
) error {
	maxToolRounds := 5
	round := 0

	for round < maxToolRounds {
		round++

		resultState, err := analystFunc()
		if err != nil {
			return errors.Wrap(err, "analyst execution failed")
		}

		report := resultState.MarketReport
		if report == "" {
			report = resultState.SentimentReport
		}
		if report == "" {
			report = resultState.NewsReport
		}
		if report == "" {
			report = resultState.FundamentalsReport
		}

		if report == "" {
			return errors.New("analyst returned empty report")
		}

		setReportFunc(report)

		if !containsToolCalls(report) {
			break
		}

		if err := a.executeToolCalls(report, state); err != nil {
			llm.Logger().Warn("Tool execution failed, continuing without tools", zap.Error(err))
			break
		}
	}

	return nil
}

// containsToolCalls 检查报告是否包含工具调用
func containsToolCalls(content string) bool {
	return len(utils.ParseToolCalls(content)) > 0
}

// executeToolCalls 执行工具调用
func (a *TradingAgentsAgent) executeToolCalls(content string, state *states.AgentState) error {
	toolCalls := utils.ParseToolCalls(content)
	if len(toolCalls) == 0 {
		return nil
	}

	ctx := context.Background()
	for _, toolCall := range toolCalls {
		llm.Logger().Info("Executing tool", zap.String("name", toolCall.Name))
		result, err := utils.ExecuteToolCall(ctx, toolCall)
		if err != nil {
			llm.Logger().Error("Tool execution failed", zap.String("name", toolCall.Name), zap.Error(err))
			return err
		}

		state.MarketReport += "\n\nTool Result:\n" + result
	}

	return nil
}

// executeResearchDebate 执行研究员辩论阶段
func (a *TradingAgentsAgent) executeResearchDebate(state *states.AgentState) error {
	maxRounds := 2 * a.config.MaxDebateRounds

	for state.InvestmentDebateState.Count < maxRounds {
		var err error

		if state.InvestmentDebateState.CurrentResponse == "" ||
			strings.HasPrefix(state.InvestmentDebateState.CurrentResponse, "Bear") {
			state, err = a.bullResearcher.BullNode(state)
			if err != nil {
				return errors.Wrap(err, "bull researcher failed")
			}
		} else {
			state, err = a.bearResearcher.BearNode(state)
			if err != nil {
				return errors.Wrap(err, "bear researcher failed")
			}
		}

		if a.debug {
			llm.Logger().Debug("Research debate round",
				zap.Int64("count", state.InvestmentDebateState.Count),
				zap.String("last_response", state.InvestmentDebateState.CurrentResponse[:min(100, len(state.InvestmentDebateState.CurrentResponse))]))
		}
	}

	state, err := a.researchManager.ResearchManagerNode(state)
	if err != nil {
		return errors.Wrap(err, "research manager failed")
	}

	return nil
}

// executeTrader 执行交易员阶段
func (a *TradingAgentsAgent) executeTrader(state *states.AgentState) error {
	var err error
	state, err = a.trader.TraderNode(state)
	if err != nil {
		return errors.Wrap(err, "trader failed")
	}
	return nil
}

// executeRiskDebate 执行风险辩论阶段
func (a *TradingAgentsAgent) executeRiskDebate(state *states.AgentState) error {
	maxRounds := 3 * a.config.MaxRiskDiscussRounds

	for state.RiskDebateState.Count < maxRounds {
		var err error

		switch {
		case state.RiskDebateState.LatestSpeaker == "" ||
			strings.HasPrefix(state.RiskDebateState.LatestSpeaker, "Neutral"):
			state, err = a.aggressiveDebator.AggressiveNode(state)
			if err != nil {
				return errors.Wrap(err, "aggressive debator failed")
			}
		case strings.HasPrefix(state.RiskDebateState.LatestSpeaker, "Aggressive"):
			state, err = a.conservativeDebator.ConservativeNode(state)
			if err != nil {
				return errors.Wrap(err, "conservative debator failed")
			}
		case strings.HasPrefix(state.RiskDebateState.LatestSpeaker, "Conservative"):
			state, err = a.neutralDebator.NeutralNode(state)
			if err != nil {
				return errors.Wrap(err, "neutral debator failed")
			}
		default:
			state, err = a.aggressiveDebator.AggressiveNode(state)
			if err != nil {
				return errors.Wrap(err, "aggressive debator failed")
			}
		}

		if a.debug {
			llm.Logger().Debug("Risk debate round",
				zap.Int64("count", state.RiskDebateState.Count),
				zap.String("latest_speaker", state.RiskDebateState.LatestSpeaker))
		}
	}

	return nil
}

// executePortfolioManager 执行投资组合经理阶段
func (a *TradingAgentsAgent) executePortfolioManager(state *states.AgentState) error {
	var err error
	state, err = a.portfolioManager.PortfolioManagerNode(state)
	if err != nil {
		return errors.Wrap(err, "portfolio manager failed")
	}
	return nil
}

// buildResult 构建最终结果
func (a *TradingAgentsAgent) buildResult(state *states.AgentState, symbol, tradeDate string) types.AgentRunResult {
	decision := a.extractDecision(state.FinalTradeDecision)

	outputs := map[string]any{
		"market_report":        state.MarketReport,
		"sentiment_report":     state.SentimentReport,
		"news_report":          state.NewsReport,
		"fundamentals_report":  state.FundamentalsReport,
		"investment_plan":      state.InvestmentPlan,
		"trader_plan":          state.TraderInvestmentPlan,
		"risk_debate_history":  state.RiskDebateState.History,
		"final_trade_decision": state.FinalTradeDecision,
	}

	return types.AgentRunResult{
		AgentName: "TradingAgents",
		Decision:  decision,
		Outputs:   outputs,
	}
}

// extractDecision 从最终决策中提取标准化交易动作
func (a *TradingAgentsAgent) extractDecision(finalDecision string) *types.AgentDecision {
	if finalDecision == "" {
		return nil
	}

	action := types.DecisionActionHold
	decisionUpper := strings.ToUpper(finalDecision)

	if strings.Contains(decisionUpper, "BUY") {
		action = types.DecisionActionBuy
	} else if strings.Contains(decisionUpper, "SELL") {
		action = types.DecisionActionSell
	} else if strings.Contains(decisionUpper, "OVERWEIGHT") {
		action = types.DecisionActionBuy
	} else if strings.Contains(decisionUpper, "UNDERWEIGHT") {
		action = types.DecisionActionSell
	}

	return &types.AgentDecision{
		AgentName: "TradingAgents",
		Symbol:    "",
		TradeDate: "",
		Action:    action,
		Rationale: finalDecision,
		Quantity:  1.0,
		Metadata: map[string]any{
			"full_decision": finalDecision,
		},
	}
}

// GetName 获取代理名称（实现 agentcore.BaseAgent 接口）
func (a *TradingAgentsAgent) GetName() string {
	return "TradingAgents"
}

// symbolContextMap 简单的资产类型映射（可扩展）
var symbolContextMap = map[string]string{
	"BTC":     "crypto",
	"ETH":     "crypto",
	"BTCUSDT": "crypto",
	"ETHUSDT": "crypto",
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
