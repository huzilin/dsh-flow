package llm

import (
	"context"

	"github.com/pkg/errors"
)

// LLMProvider LLM 服务提供商枚举
type LLMProvider string

const (
	LLMProviderZhipu      LLMProvider = "zhipu"
	LLMProviderOpenAI     LLMProvider = "openai"
	LLMProviderAzure      LLMProvider = "azure"
	LLMProviderAnthropic  LLMProvider = "anthropic"
	LLMProviderGoogle     LLMProvider = "google"
	LLMProviderXAI        LLMProvider = "xai"
	LLMProviderOpenRouter LLMProvider = "openrouter"
	LLMProviderOllama     LLMProvider = "ollama"
	LLMProviderQwen       LLMProvider = "qwen"
	LLMProviderQwenCN     LLMProvider = "qwen-cn"
	LLMProviderGLM        LLMProvider = "glm"
	LLMProviderGLMCN      LLMProvider = "glm-cn"
	LLMProviderMiniMax    LLMProvider = "minimax"
	LLMProviderMiniMaxCN  LLMProvider = "minimax-cn"
	LLMProviderDeepSeek   LLMProvider = "deepseek"
	LLMProviderMistral    LLMProvider = "mistral"
	LLMProviderKimi       LLMProvider = "kimi"
	LLMProviderGroq       LLMProvider = "groq"
	LLMProviderNVIDIA     LLMProvider = "nvidia"
)

// LLMClient LLM 客户端接口
type LLMClient interface {
	// Chat 发送聊天请求
	Chat(ctx context.Context, messages []*ChatMessage, options *ChatOptions) (*LLMResponse, error)

	// Name 返回客户端名称
	Name() string

	// Provider 返回服务提供商
	Provider() LLMProvider
}

// ChatMessage 聊天消息
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatOptions 聊天选项
type ChatOptions struct {
	Model            string   `json:"model"`
	Temperature      float64  `json:"temperature"`
	MaxTokens        int64    `json:"max_tokens"`
	TopP             float64  `json:"top_p"`
	FrequencyPenalty float64  `json:"frequency_penalty"`
	PresencePenalty  float64  `json:"presence_penalty"`
	Stop             []string `json:"stop,omitempty"`
	Tools            []*Tool  `json:"tools,omitempty"`
}

// Tool 函数工具定义
type Tool struct {
	Type     string        `json:"type"`
	Function *ToolFunction `json:"function"`
}

type ToolFunction struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parameters  any    `json:"parameters"`
}

// ToolCall 工具调用
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// LLMResponse LLM 响应
type LLMResponse struct {
	Content      string     `json:"content"`
	Model        string     `json:"model"`
	Usage        Usage      `json:"usage"`
	FinishReason string     `json:"finish_reason"`
	ToolCalls    []ToolCall `json:"tool_calls,omitempty"`
}

// Usage 用法统计
type Usage struct {
	PromptTokens     int64 `json:"prompt_tokens"`
	CompletionTokens int64 `json:"completion_tokens"`
	TotalTokens      int64 `json:"total_tokens"`
}

// ClientConfig 客户端配置
type ClientConfig struct {
	Provider        LLMProvider
	APIKey          string
	BaseURL         string
	Model           string
	Temperature     float64
	MaxTokens       int64
	Timeout         int    `json:"timeout"`
	MaxRetries      int    `json:"max_retries"`
	ReasoningEffort string `json:"reasoning_effort"`
}

// Factory LLM 客户端工厂
type Factory struct {
	clients map[LLMProvider]LLMClient
}

// NewFactory 创建新的工厂实例
func NewFactory() *Factory {
	return &Factory{
		clients: make(map[LLMProvider]LLMClient),
	}
}

// CreateClient 根据配置创建 LLM 客户端
func (f *Factory) CreateClient(config ClientConfig) (LLMClient, error) {
	if client, ok := f.clients[config.Provider]; ok {
		return client, nil
	}

	// 如果是 OpenAI 兼容提供商，使用 go-openai 库
	if IsOpenAICompatible(config.Provider) {
		client, err := NewOpenAIClientV2(&config)
		if err != nil {
			return nil, err
		}
		f.clients[config.Provider] = client
		return client, nil
	}

	var client LLMClient
	switch config.Provider {
	case LLMProviderZhipu:
		client = NewZhipuClient(config)
	case LLMProviderAzure:
		client = NewAzureClient(config)
	case LLMProviderAnthropic:
		client = NewAnthropicClient(config)
	case LLMProviderGoogle:
		client = NewGoogleClient(config)
	default:
		return nil, errors.Errorf("unsupported LLM provider: %s", config.Provider)
	}

	f.clients[config.Provider] = client
	return client, nil
}

// GetClient 获取已创建的客户端
func (f *Factory) GetClient(provider LLMProvider) (LLMClient, bool) {
	client, ok := f.clients[provider]
	return client, ok
}

// 各种 LLM 客户端实现（占位）

// ZhipuClient 智谱客户端
type ZhipuClient struct {
	config ClientConfig
}

func NewZhipuClient(config ClientConfig) *ZhipuClient {
	return &ZhipuClient{config: config}
}

func (c *ZhipuClient) Chat(ctx context.Context, messages []*ChatMessage, options *ChatOptions) (*LLMResponse, error) {
	return &LLMResponse{}, nil
}

func (c *ZhipuClient) Name() string          { return "zhipu" }
func (c *ZhipuClient) Provider() LLMProvider { return LLMProviderZhipu }

// AzureClient Azure 客户端
type AzureClient struct {
	config ClientConfig
}

func NewAzureClient(config ClientConfig) *AzureClient {
	return &AzureClient{config: config}
}

func (c *AzureClient) Chat(ctx context.Context, messages []*ChatMessage, options *ChatOptions) (*LLMResponse, error) {
	return &LLMResponse{}, nil
}

func (c *AzureClient) Name() string          { return "azure" }
func (c *AzureClient) Provider() LLMProvider { return LLMProviderAzure }

// AnthropicClient Anthropic 客户端
type AnthropicClient struct {
	config ClientConfig
}

func NewAnthropicClient(config ClientConfig) *AnthropicClient {
	return &AnthropicClient{config: config}
}

func (c *AnthropicClient) Chat(ctx context.Context, messages []*ChatMessage, options *ChatOptions) (*LLMResponse, error) {
	return &LLMResponse{}, nil
}

func (c *AnthropicClient) Name() string          { return "anthropic" }
func (c *AnthropicClient) Provider() LLMProvider { return LLMProviderAnthropic }

// GoogleClient Google 客户端
type GoogleClient struct {
	config ClientConfig
}

func NewGoogleClient(config ClientConfig) *GoogleClient {
	return &GoogleClient{config: config}
}

func (c *GoogleClient) Chat(ctx context.Context, messages []*ChatMessage, options *ChatOptions) (*LLMResponse, error) {
	return &LLMResponse{}, nil
}

func (c *GoogleClient) Name() string          { return "google" }
func (c *GoogleClient) Provider() LLMProvider { return LLMProviderGoogle }
