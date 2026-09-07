package llm

import (
	"os"
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

// API 密钥环境变量映射
var providerAPIKeyEnv = map[LLMProvider]string{
	LLMProviderOpenAI:     "OPENAI_API_KEY",
	LLMProviderXAI:        "XAI_API_KEY",
	LLMProviderDeepSeek:   "DEEPSEEK_API_KEY",
	LLMProviderQwen:       "QWEN_API_KEY",
	LLMProviderQwenCN:     "QWEN_API_KEY",
	LLMProviderGLM:        "ZHIPU_API_KEY",
	LLMProviderGLMCN:      "ZHIPU_API_KEY",
	LLMProviderZhipu:      "ZHIPU_API_KEY",
	LLMProviderMiniMax:    "MINIMAX_API_KEY",
	LLMProviderMiniMaxCN:  "MINIMAX_API_KEY",
	LLMProviderOpenRouter: "OPENROUTER_API_KEY",
	LLMProviderMistral:    "MISTRAL_API_KEY",
	LLMProviderKimi:       "KIMI_API_KEY",
	LLMProviderGroq:       "GROQ_API_KEY",
	LLMProviderNVIDIA:     "NVIDIA_API_KEY",
	LLMProviderOllama:     "", // local server, no key
}

// 默认提供商配置
var providerSpecs = map[LLMProvider]ProviderSpec{
	LLMProviderOpenAI:     {UseResponsesAPI: true},
	LLMProviderXAI:        {BaseURL: "https://api.x.ai/v1"},
	LLMProviderDeepSeek:   {BaseURL: "https://api.deepseek.com"},
	LLMProviderQwen:       {BaseURL: "https://dashscope-intl.aliyuncs.com/compatible-mode/v1"},
	LLMProviderQwenCN:     {BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1"},
	LLMProviderGLM:        {BaseURL: "https://api.z.ai/api/paas/v4/"},
	LLMProviderGLMCN:      {BaseURL: "https://open.bigmodel.cn/api/paas/v4/"},
	LLMProviderZhipu:      {BaseURL: "https://api.z.ai/api/paas/v4/"},
	LLMProviderMiniMax:    {BaseURL: "https://api.minimax.io/v1"},
	LLMProviderMiniMaxCN:  {BaseURL: "https://api.minimaxi.com/v1"},
	LLMProviderOpenRouter: {BaseURL: "https://openrouter.ai/api/v1"},
	LLMProviderMistral:    {BaseURL: "https://api.mistral.ai/v1"},
	LLMProviderKimi:       {BaseURL: "https://api.moonshot.ai/v1"},
	LLMProviderGroq:       {BaseURL: "https://api.groq.com/openai/v1"},
	LLMProviderNVIDIA:     {BaseURL: "https://integrate.api.nvidia.com/v1"},
	LLMProviderOllama: {
		BaseURL:        "http://localhost:11434/v1",
		BaseURLEnv:     "OLLAMA_BASE_URL",
		KeyOptional:    true,
		PlaceholderKey: "ollama",
	},
}

// ModelCapabilities 模型能力
type ModelCapabilities struct {
	SupportsToolChoice                bool   // 是否支持 tool_choice 参数
	SupportsJSONMode                  bool   // 是否支持 json_mode
	SupportsJSONSchema                bool   // 是否支持 json_schema
	PreferredStructuredMethod         string // 首选的结构化输出方法
	RequiresReasoningContentRoundtrip bool   // 是否需要 reasoning_content 回传（DeepSeek）
	RequiresReasoningSplit            bool   // 是否需要 reasoning_split 参数（MiniMax M2.x）
}

// StructuredMethod 结构化输出方法常量
const (
	StructuredMethodFunctionCalling = "function_calling"
	StructuredMethodJSONMode        = "json_mode"
	StructuredMethodJSONSchema      = "json_schema"
	StructuredMethodNone            = "none"
)

// GetCapabilities 获取模型能力
func GetCapabilities(model string) ModelCapabilities {
	// 精确匹配
	if caps, ok := capabilitiesByID[model]; ok {
		return caps
	}

	// 模式匹配
	for pattern, caps := range capabilitiesByPattern {
		if pattern.MatchString(model) {
			return caps
		}
	}

	// 默认能力
	return defaultCapabilities
}

// 模型能力表（精确匹配）
var capabilitiesByID = map[string]ModelCapabilities{
	// DeepSeek
	"deepseek-chat": ModelCapabilities{
		SupportsToolChoice:        true,
		SupportsJSONMode:          true,
		SupportsJSONSchema:        false,
		PreferredStructuredMethod: StructuredMethodFunctionCalling,
	},
	"deepseek-reasoner": ModelCapabilities{
		SupportsToolChoice:                false,
		SupportsJSONMode:                  true,
		SupportsJSONSchema:                false,
		PreferredStructuredMethod:         StructuredMethodFunctionCalling,
		RequiresReasoningContentRoundtrip: true,
	},
	"deepseek-v4-flash": ModelCapabilities{
		SupportsToolChoice:                false,
		SupportsJSONMode:                  true,
		SupportsJSONSchema:                false,
		PreferredStructuredMethod:         StructuredMethodFunctionCalling,
		RequiresReasoningContentRoundtrip: true,
	},
	"deepseek-v4-pro": ModelCapabilities{
		SupportsToolChoice:                false,
		SupportsJSONMode:                  true,
		SupportsJSONSchema:                false,
		PreferredStructuredMethod:         StructuredMethodFunctionCalling,
		RequiresReasoningContentRoundtrip: true,
	},
	// MiniMax M2.x 系列
	"MiniMax-M2.7": ModelCapabilities{
		SupportsToolChoice:        false,
		SupportsJSONMode:          false,
		SupportsJSONSchema:        false,
		PreferredStructuredMethod: StructuredMethodFunctionCalling,
		RequiresReasoningSplit:    true,
	},
	"MiniMax-M2.7-highspeed": ModelCapabilities{
		SupportsToolChoice:        false,
		SupportsJSONMode:          false,
		SupportsJSONSchema:        false,
		PreferredStructuredMethod: StructuredMethodFunctionCalling,
		RequiresReasoningSplit:    true,
	},
	"MiniMax-M2.5": ModelCapabilities{
		SupportsToolChoice:        false,
		SupportsJSONMode:          false,
		SupportsJSONSchema:        false,
		PreferredStructuredMethod: StructuredMethodFunctionCalling,
		RequiresReasoningSplit:    true,
	},
	"MiniMax-M2.5-highspeed": ModelCapabilities{
		SupportsToolChoice:        false,
		SupportsJSONMode:          false,
		SupportsJSONSchema:        false,
		PreferredStructuredMethod: StructuredMethodFunctionCalling,
		RequiresReasoningSplit:    true,
	},
	"MiniMax-M2.1": ModelCapabilities{
		SupportsToolChoice:        false,
		SupportsJSONMode:          false,
		SupportsJSONSchema:        false,
		PreferredStructuredMethod: StructuredMethodFunctionCalling,
		RequiresReasoningSplit:    true,
	},
	"MiniMax-M2.1-highspeed": ModelCapabilities{
		SupportsToolChoice:        false,
		SupportsJSONMode:          false,
		SupportsJSONSchema:        false,
		PreferredStructuredMethod: StructuredMethodFunctionCalling,
		RequiresReasoningSplit:    true,
	},
	"MiniMax-M2": ModelCapabilities{
		SupportsToolChoice:        false,
		SupportsJSONMode:          false,
		SupportsJSONSchema:        false,
		PreferredStructuredMethod: StructuredMethodFunctionCalling,
		RequiresReasoningSplit:    true,
	},
}

// 模型能力表（模式匹配）
var capabilitiesByPattern = map[*regexp.Regexp]ModelCapabilities{
	regexp.MustCompile(`^deepseek-v\d`): ModelCapabilities{
		SupportsToolChoice:                false,
		SupportsJSONMode:                  true,
		SupportsJSONSchema:                false,
		PreferredStructuredMethod:         StructuredMethodFunctionCalling,
		RequiresReasoningContentRoundtrip: true,
	},
	regexp.MustCompile(`^deepseek-reasoner`): ModelCapabilities{
		SupportsToolChoice:                false,
		SupportsJSONMode:                  true,
		SupportsJSONSchema:                false,
		PreferredStructuredMethod:         StructuredMethodFunctionCalling,
		RequiresReasoningContentRoundtrip: true,
	},
	regexp.MustCompile(`^MiniMax-M\d`): ModelCapabilities{
		SupportsToolChoice:        false,
		SupportsJSONMode:          false,
		SupportsJSONSchema:        false,
		PreferredStructuredMethod: StructuredMethodFunctionCalling,
		RequiresReasoningSplit:    true,
	},
}

// 默认模型能力
var defaultCapabilities = ModelCapabilities{
	SupportsToolChoice:        true,
	SupportsJSONMode:          true,
	SupportsJSONSchema:        true,
	PreferredStructuredMethod: StructuredMethodFunctionCalling,
}

// validateModel 验证V模型是否有效（内部函数）
func validateModel(provider LLMProvider, model string) bool {
	// 基本验证：模型名不能为空
	if model == "" {
		return false
	}

	// 根据提供商进行特定验证
	switch provider {
	case LLMProviderOpenAI:
		return isOpenAIModel(model)
	case LLMProviderDeepSeek:
		return isDeepSeekModel(model)
	case LLMProviderQwen, LLMProviderQwenCN:
		return isQwenModel(model)
	case LLMProviderMiniMax, LLMProviderMiniMaxCN:
		return isMiniMaxModel(model)
	case LLMProviderZhipu, LLMProviderGLM, LLMProviderGLMCN:
		return isZhipuModel(model)
	default:
		// 其他提供商不做严格验证
		return true
	}
}

// isOpenAIModel 检查是否为有效的 OpenAI 模型
func isOpenAIModel(model string) bool {
	validPrefixes := []string{
		"gpt-4", "gpt-4o", "gpt-3.5", "o1-", "o3-", "chatgpt-4o",
	}
	for _, prefix := range validPrefixes {
		if strings.HasPrefix(model, prefix) {
			return true
		}
	}
	return false
}

// isDeepSeekModel 检查是否为有效的 DeepSeek 模型
func isDeepSeekModel(model string) bool {
	validPrefixes := []string{
		"deepseek-chat", "deepseek-coder", "deepseek-reasoner",
	}
	for _, prefix := range validPrefixes {
		if strings.HasPrefix(model, prefix) {
			return true
		}
	}
	return false
}

// isQwenModel 检查是否为有效的 Qwen 模型
func isQwenModel(model string) bool {
	validPrefixes := []string{
		"qwen-", "qwen2-", "qwen2.5-", "qwen3-",
	}
	for _, prefix := range validPrefixes {
		if strings.HasPrefix(model, prefix) {
			return true
		}
	}
	return false
}

// isMiniMaxModel 检查是否为有效的 MiniMax 模型
func isMiniMaxModel(model string) bool {
	validPrefixes := []string{
		"MiniMax-Text", "abab6", "abab7",
	}
	for _, prefix := range validPrefixes {
		if strings.HasPrefix(model, prefix) {
			return true
		}
	}
	return false
}

// isZhipuModel 检查是否为有效的智谱模型
func isZhipuModel(model string) bool {
	validPrefixes := []string{
		"glm-", "chatglm-",
	}
	for _, prefix := range validPrefixes {
		if strings.HasPrefix(model, prefix) {
			return true
		}
	}
	return false
}

// resolveConfig 解析配置（处理 base_url 和 API key）
func resolveConfig(config *ClientConfig) (*ClientConfig, error) {
	if config == nil {
		return nil, errors.New("config is nil")
	}

	// 复制配置以避免修改原值
	resolved := *config

	spec, ok := providerSpecs[config.Provider]
	if !ok {
		// 如果没有 spec，至少设置默认值
		if resolved.BaseURL == "" {
			resolved.BaseURL = "https://api.openai.com/v1"
		}
		return &resolved, nil
	}

	// base_url 优先级: 显式配置 > 环境变量 > 默认值
	if resolved.BaseURL == "" {
		if spec.BaseURLEnv != "" {
			if envURL := os.Getenv(spec.BaseURLEnv); envURL != "" {
				resolved.BaseURL = envURL
			}
		}
		if resolved.BaseURL == "" {
			resolved.BaseURL = spec.BaseURL
		}
	}

	// 验证 base_url
	if spec.RequireBaseURL && resolved.BaseURL == "" {
		return &resolved, errors.Errorf("provider '%s' requires a base_url", resolved.Provider)
	}

	// API key 处理
	apiKeyEnv := providerAPIKeyEnv[config.Provider]
	if apiKeyEnv != "" && resolved.APIKey == "" {
		resolved.APIKey = os.Getenv(apiKeyEnv)
	}

	if spec.KeyOptional && resolved.APIKey == "" {
		resolved.APIKey = spec.PlaceholderKey
	} else if !spec.KeyOptional && resolved.APIKey == "" && apiKeyEnv != "" {
		return &resolved, errors.Errorf("API key for provider '%s' is not set. Please set the %s environment variable", resolved.Provider, apiKeyEnv)
	}

	// 设置默认值
	if resolved.Timeout == 0 {
		resolved.Timeout = 60
	}
	if resolved.MaxRetries == 0 {
		resolved.MaxRetries = 3
	}

	return &resolved, nil
}
