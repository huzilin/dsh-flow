package platform

import (
	"encoding/json"
	"os"

	"github.com/huzilin/trade/agentcore"
	"github.com/huzilin/trade/agents"
	"github.com/huzilin/trade/agents/qa"
	"github.com/huzilin/trade/llm"
	"github.com/huzilin/trade/types"
	"github.com/huzilin/trade/utils"
)

var (
	defConfig = &Config{
		// LLM 配置默认值
		LLMProvider:         "zhipu",
		DeepThinkLLM:        "GLM-5.1",
		QuickThinkLLM:       "GLM-5.1",
		AllowVendorFallback: true,

		// 辩论与讨论轮次配置默认值
		MaxDebateRounds:      3,
		MaxRiskDiscussRounds: 3,
		ResearchDepth:        3,

		// 目录配置默认值
		LocalDataDir:         "./local_data",
		DataToolsCacheDir:    "./local_data/data_tools/cache",
		DataToolsSnapshotDir: "./local_data/data_tools/snapshots",
		MarketDataDir:        "./local_data/market",
		AgentOutputDir:       "./outputs/agents",
		BacktestOutputDir:    "./outputs/backtests",
		ReportOutputDir:      "./reports",
		DataCacheDir:         "./data_cache",
	}
)

// Config 交易平台配置（支持 JSON 反序列化）
type Config struct {
	// LLM 配置
	LLMProvider         string `json:"LLMProvider"`
	DeepThinkLLM        string `json:"DeepThinkLLM"`
	QuickThinkLLM       string `json:"QuickThinkLLM"`
	AllowVendorFallback bool   `json:"AllowVendorFallback"`

	// 辩论与讨论轮次配置
	MaxDebateRounds      int64 `json:"MaxDebateRounds"`
	MaxRiskDiscussRounds int64 `json:"MaxRiskDiscussRounds"`
	ResearchDepth        int64 `json:"ResearchDepth"`

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

// TradingPlatform 组合数据工具、市场工具、Agent 与回测层的新平台入口
type TradingPlatform struct {
	Config *Config

	// 运行时组件
	dataToolRegistry      any
	artifactStore         any
	dataToolExecutor      any
	dataCollectionService any
	marketTools           any
	agentRegistry         *agentcore.AgentRegistry
	backtestEngine        any

	// 动态配置（用于扩展配置）
	extraConfig map[string]any
}

// NewTradingPlatform 创建新的 TradingPlatform 实例
// 如果 config 参数为 nil，则使用默认配置
// 如果 config 中某些字段未赋值（零值），则使用默认配置中对应字段的值
func NewTradingPlatform(config *Config) (*TradingPlatform, error) {
	cfg := mergeConfig(defConfig, config)

	p := &TradingPlatform{
		Config:        cfg,
		agentRegistry: agentcore.NewAgentRegistry(),
		extraConfig:   make(map[string]any),
	}

	p.ensureDirectories()
	p.initializeComponents()

	return p, nil
}

// mergeConfig 将 config 的值合并到 defaults 中
// 如果 config 中某个字段是零值，则保留 defaults 中的值
func mergeConfig(defaults, config *Config) *Config {
	if config == nil {
		return defaults
	}

	result := &Config{}

	// LLM 配置
	result.LLMProvider = utils.Coalesce(config.LLMProvider, defaults.LLMProvider)
	result.DeepThinkLLM = utils.Coalesce(config.DeepThinkLLM, defaults.DeepThinkLLM)
	result.QuickThinkLLM = utils.Coalesce(config.QuickThinkLLM, defaults.QuickThinkLLM)
	result.AllowVendorFallback = utils.Coalesce(config.AllowVendorFallback, defaults.AllowVendorFallback)

	// 辩论与讨论轮次配置
	result.MaxDebateRounds = utils.Coalesce(config.MaxDebateRounds, defaults.MaxDebateRounds)
	result.MaxRiskDiscussRounds = utils.Coalesce(config.MaxRiskDiscussRounds, defaults.MaxRiskDiscussRounds)
	result.ResearchDepth = utils.Coalesce(config.ResearchDepth, defaults.ResearchDepth)

	// 目录配置
	result.LocalDataDir = utils.Coalesce(config.LocalDataDir, defaults.LocalDataDir)
	result.DataToolsCacheDir = utils.Coalesce(config.DataToolsCacheDir, defaults.DataToolsCacheDir)
	result.DataToolsSnapshotDir = utils.Coalesce(config.DataToolsSnapshotDir, defaults.DataToolsSnapshotDir)
	result.MarketDataDir = utils.Coalesce(config.MarketDataDir, defaults.MarketDataDir)
	result.AgentOutputDir = utils.Coalesce(config.AgentOutputDir, defaults.AgentOutputDir)
	result.BacktestOutputDir = utils.Coalesce(config.BacktestOutputDir, defaults.BacktestOutputDir)
	result.ReportOutputDir = utils.Coalesce(config.ReportOutputDir, defaults.ReportOutputDir)
	result.DataCacheDir = utils.Coalesce(config.DataCacheDir, defaults.DataCacheDir)

	return result
}

// ensureDirectories 创建平台运行所需目录
func (p *TradingPlatform) ensureDirectories() {
	directories := []string{
		p.Config.LocalDataDir,
		p.Config.DataToolsCacheDir,
		p.Config.DataToolsSnapshotDir,
		p.Config.MarketDataDir,
		p.Config.AgentOutputDir,
		p.Config.BacktestOutputDir,
		p.Config.ReportOutputDir,
		p.Config.DataCacheDir,
	}

	for _, dir := range directories {
		if dir != "" {
			os.MkdirAll(dir, 0755)
		}
	}
}

// initializeComponents 初始化平台组件
func (p *TradingPlatform) initializeComponents() {
	p.dataToolRegistry = p.createDataToolRegistry()
	p.artifactStore = p.createArtifactStore()
	p.dataToolExecutor = p.createDataToolExecutor()
	p.dataCollectionService = p.createDataCollectionService()
	p.marketTools = p.createMarketTools()
	p.backtestEngine = p.createBacktestEngine()
}

// createDataToolRegistry 创建数据工具注册表（占位实现）
func (p *TradingPlatform) createDataToolRegistry() any {
	return nil
}

// createArtifactStore 创建制品存储（占位实现）
func (p *TradingPlatform) createArtifactStore() any {
	return nil
}

// createDataToolExecutor 创建数据工具执行器（占位实现）
func (p *TradingPlatform) createDataToolExecutor() any {
	return nil
}

// createDataCollectionService 创建数据采集服务（占位实现）
func (p *TradingPlatform) createDataCollectionService() any {
	return nil
}

// createMarketTools 创建市场工具（占位实现）
func (p *TradingPlatform) createMarketTools() any {
	return nil
}

// createBacktestEngine 创建回测引擎（占位实现）
func (p *TradingPlatform) createBacktestEngine() any {
	return nil
}

// RegisterDataTool 注册自定义数据工具
func (p *TradingPlatform) RegisterDataTool(definition any) {
}

// CollectData 独立运行数据采集任务
func (p *TradingPlatform) CollectData(jobs []any) []any {
	return []any{}
}

// RegisterAgent 注册一个可独立运行的 Agent
func (p *TradingPlatform) RegisterAgent(agent agentcore.BaseAgent) {
	p.agentRegistry.Register(agent)
}

// RegisterTradingAgentsAgent 注册当前仓库内置的 TradingAgents 实现
func (p *TradingPlatform) RegisterTradingAgentsAgent(opts ...*agents.TradingAgentsAgentOptions) agentcore.BaseAgent {
	var opt *agents.TradingAgentsAgentOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	// 如果没有传入配置或配置为空，则创建新配置并从 platform.Config 复制
	if opt == nil || opt.Config == nil {
		opt = &agents.TradingAgentsAgentOptions{
			Config: agents.NewTradingAgentsConfig(),
		}
		opt.Config.MaxDebateRounds = p.Config.MaxDebateRounds
		opt.Config.MaxRiskDiscussRounds = p.Config.MaxRiskDiscussRounds
		opt.Config.ResearchDepth = p.Config.ResearchDepth
		opt.Config.ReportOutputDir = p.Config.ReportOutputDir
	}

	agent := agents.NewTradingAgentsAgent(opt)
	p.RegisterAgent(agent)
	return agent
}

// RegisterQAAgent 注册 QA 问答代理
func (p *TradingPlatform) RegisterQAAgent(llmClient *llm.OpenAIClientV2) agentcore.BaseAgent {
	agent := qa.NewQAAgent(llmClient, &qa.QAConfig{
		Model: p.Config.QuickThinkLLM,
	})
	p.RegisterAgent(agent)
	return agent
}

// RunAgent 独立运行指定 Agent
func (p *TradingPlatform) RunAgent(agentName string, request *types.AgentRunRequest) (types.AgentRunResult, error) {
	agent, ok := p.agentRegistry.Get(agentName)
	if !ok {
		return types.AgentRunResult{}, ErrAgentNotFound(agentName)
	}
	result, err := agent.Run(request, p.buildExecutionContext())
	if err != nil {
		return types.AgentRunResult{}, err
	}
	return result.(types.AgentRunResult), nil
}

// BacktestAgent 运行指定 Agent 并对其结果执行回测
func (p *TradingPlatform) BacktestAgent(agentName string, requests []types.AgentRunRequest, barRule string) any {
	return nil
}

// BuildExecutionContext 构建 Agent 运行上下文
func (p *TradingPlatform) BuildExecutionContext() *types.AgentExecutionContext {
	return p.buildExecutionContext()
}

func (p *TradingPlatform) buildExecutionContext() *types.AgentExecutionContext {
	return &types.AgentExecutionContext{
		Config:      p.ToMap(),
		DataTools:   p.dataToolExecutor,
		MarketTools: p.marketTools,
	}
}

// ToMap 将 TradingPlatform 配置转换为 map
func (p *TradingPlatform) ToMap() map[string]any {
	data, _ := json.Marshal(p.Config)
	m := make(map[string]any)
	json.Unmarshal(data, &m)

	// 动态配置
	for k, v := range p.extraConfig {
		m[k] = v
	}

	return m
}

// SetExtraConfig 设置额外的配置项
func (p *TradingPlatform) SetExtraConfig(key string, value any) {
	p.extraConfig[key] = value
}

// ErrAgentNotFound Agent 未找到错误
type ErrAgentNotFound string

func (e ErrAgentNotFound) Error() string {
	return "agent not found: " + string(e)
}

// CreateDefaultPlatform 创建一个带默认组件的新平台实例
func CreateDefaultPlatform() *TradingPlatform {
	p, _ := NewTradingPlatform(nil)
	return p
}
