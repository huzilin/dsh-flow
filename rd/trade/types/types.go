package types

// DecisionAction 标准化的交易动作枚举
type DecisionAction string

const (
	DecisionActionBuy  DecisionAction = "BUY"
	DecisionActionSell DecisionAction = "SELL"
	DecisionActionHold DecisionAction = "HOLD"
)

// AgentDecision 定义 Agent 输出给回测层的标准化决策
type AgentDecision struct {
	AgentName         string         `json:"AgentName"`
	Symbol            string         `json:"Symbol"`
	TradeDate         string         `json:"TradeDate"`
	Action            DecisionAction `json:"Action"`
	Rationale         string         `json:"Rationale"`
	Confidence        *float64       `json:"Confidence,omitempty"`
	Quantity          float64        `json:"Quantity"`
	DecisionTime      *string        `json:"DecisionTime,omitempty"`
	HoldingPeriodBars int64          `json:"HoldingPeriodBars"`
	Metadata          map[string]any `json:"Metadata"`
}

// AgentRunRequest 描述一次 Agent 独立运行请求
type AgentRunRequest struct {
	Symbol    string         `json:"Symbol"`
	TradeDate string         `json:"TradeDate"`
	Context   map[string]any `json:"Context"`
}

// AgentRunResult 描述一次 Agent 运行结果
type AgentRunResult struct {
	AgentName string         `json:"AgentName"`
	Decision  *AgentDecision `json:"Decision,omitempty"`
	Outputs   map[string]any `json:"Outputs"`
}

// AgentExecutionContext 描述 Agent 运行时可访问的共享能力
type AgentExecutionContext struct {
	Config      map[string]any `json:"Config"`
	DataTools   any            `json:"DataTools"`
	MarketTools any            `json:"MarketTools"`
}

// TradingAgentConfig 交易 Agent 配置
type TradingAgentConfig struct {
	LLMProvider          string `json:"LLMProvider"`
	DeepThinkLLM         string `json:"DeepThinkLLM"`
	QuickThinkLLM        string `json:"QuickThinkLLM"`
	AllowVendorFallback  bool   `json:"AllowVendorFallback"`
	MaxDebateRounds      int64  `json:"MaxDebateRounds"`
	MaxRiskDiscussRounds int64  `json:"MaxRiskDiscussRounds"`
	ResearchDepth        int64  `json:"ResearchDepth"`

	// 目录配置
	LocalDataDir         string `json:"LocalDataDir"`
	DataToolsCacheDir    string `json:"DataToolsCacheDir"`
	DataToolsSnapshotDir string `json:"DataToolsSnapshotDir"`
	MarketDataDir        string `json:"MarketDataDir"`
	AgentOutputDir       string `json:"AgentOutputDir"`
	BacktestOutputDir    string `json:"BacktestOutputDir"`
	ReportOutputDir      string `json:"ReportOutputDir"`
	DataCacheDir         string `json:"DataCacheDir"`
}

// ReportMetadata 报告元数据
type ReportMetadata struct {
	ReportFile    string `json:"ReportFile"`
	ReportDir     string `json:"ReportDir"`
	ReportPDFFile string `json:"ReportPdfFile"`
}

// RuntimeConfig 运行时配置
type RuntimeConfig struct {
	Config           map[string]any
	SelectedAnalysts []string
	Debug            bool
}
