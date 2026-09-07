package config

import "os"

// DataVendors 数据供应商配置
type DataVendors struct {
	CoreStockAPIS       string `json:"core_stock_apis"`      // 股票数据供应商
	TechnicalIndicators string `json:"technical_indicators"` // 技术指标供应商
	FundamentalData     string `json:"fundamental_data"`     // 基本面数据供应商
	NewsData            string `json:"news_data"`            // 新闻数据供应商
	MacroData           string `json:"macro_data"`           // 宏观数据供应商
	PredictionMarkets   string `json:"prediction_markets"`   // 预测市场供应商
}

// OrchestrationRule 编排规则：市场+能力 -> 供应商列表
type OrchestrationRule struct {
	Market     string   `json:"market"`     // 市场类型
	Capability string   `json:"capability"` // 能力名称
	Vendors    []string `json:"vendors"`    // 供应商列表（按优先级排序）
}

// OrchestrationConfig 编排器配置
type OrchestrationConfig struct {
	Rules []OrchestrationRule `json:"rules"` // 编排规则列表
}

// Config 配置结构
type Config struct {
	DataVendors   DataVendors         `json:"data_vendors"`
	ToolVendors   map[string]string   `json:"tool_vendors"`  // 工具级别的供应商配置，优先级高于 DataVendors
	Orchestration OrchestrationConfig `json:"orchestration"` // 供应商编排配置

	// API Keys
	AlphaVantageAPIKey string `json:"alpha_vantage_api_key"`
	FREDAPIKey         string `json:"fred_api_key"`
	PolymarketAPIKey   string `json:"polymarket_api_key"`
	MXAPIKey           string `json:"mx_api_key"`

	// Language settings
	InternalLanguage    string `json:"internal_language"`     // 内部分析与辩论语言
	FinalOutputLanguage string `json:"final_output_language"` // 最终报告输出语言
	OutputLanguage      string `json:"output_language"`       // 兼容旧接口的输出语言配置
}

// 全局配置实例
var cfg *Config

// DefaultConfig 默认配置
// 注意：基本面和新闻数据默认使用 Alpha Vantage，可通过 ToolVendors 覆盖
// Alpha Vantage API 有调用频率限制（5 calls/min, 500 calls/day），生产环境建议申请 Premium
var DefaultConfig = &Config{
	DataVendors: DataVendors{
		CoreStockAPIS:       "yfinance",
		TechnicalIndicators: "yfinance",
		// 基本面数据需要 Alpha Vantage（Yahoo Finance 不提供完整的基本面 API）
		FundamentalData: "alpha_vantage",
		// 新闻数据需要 Alpha Vantage（Yahoo Finance 不提供新闻 API）
		NewsData:          "alpha_vantage",
		MacroData:         "fred",
		PredictionMarkets: "polymarket",
	},
	ToolVendors:         make(map[string]string),
	InternalLanguage:    "English",
	FinalOutputLanguage: "Chinese",
	OutputLanguage:      "Chinese",
}

// Init 初始化配置
func Init() {
	if cfg == nil {
		cfg = &Config{}
		*cfg = *DefaultConfig
	}
}

// Get 获取当前配置
func Get() *Config {
	if cfg == nil {
		Init()
	}
	return cfg
}

// Set 更新配置
func Set(config *Config) {
	if config == nil {
		return
	}
	if cfg == nil {
		cfg = &Config{}
	}
	*cfg = *config
}

// SetFromEnv 从环境变量加载配置
func SetFromEnv() {
	Init()
	if key := os.Getenv("ALPHA_VANTAGE_API_KEY"); key != "" {
		cfg.AlphaVantageAPIKey = key
	}
	if key := os.Getenv("FRED_API_KEY"); key != "" {
		cfg.FREDAPIKey = key
	}
	if key := os.Getenv("POLYMARKET_API_KEY"); key != "" {
		cfg.PolymarketAPIKey = key
	}
	if key := os.Getenv("MX_APIKEY"); key != "" {
		cfg.MXAPIKey = key
	}
	if key := os.Getenv("EASTMONEY_APIKEY"); key != "" && cfg.MXAPIKey == "" {
		cfg.MXAPIKey = key
	}
}
