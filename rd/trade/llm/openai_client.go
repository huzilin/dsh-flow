package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/huzilin/trade/logger"
	"github.com/pkg/errors"
	"github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
)

// Logger 获取日志记录器
func Logger() *zap.Logger {
	return logger.L()
}

// OpenAIClient OpenAI 客户端（使用 go-openai 库）
type OpenAIClientV2 struct {
	client          *openai.Client
	config          *ClientConfig
	useResponsesAPI bool
}

// NewOpenAIClientV2 创建新的 OpenAI 客户端
func NewOpenAIClientV2(config *ClientConfig) (*OpenAIClientV2, error) {
	resolvedConfig, err := resolveConfig(config)
	if err != nil {
		return nil, err
	}

	// 创建 OpenAI 客户端配置
	clientConfig := openai.DefaultConfig(resolvedConfig.APIKey)
	if resolvedConfig.BaseURL != "" {
		clientConfig.BaseURL = resolvedConfig.BaseURL
	}

	// 创建客户端
	client := openai.NewClientWithConfig(clientConfig)

	// 检查是否使用 Responses API（仅原生 OpenAI）
	spec, _ := providerSpecs[config.Provider]
	useResponsesAPI := spec.UseResponsesAPI && isNativeOpenAIBaseURL(resolvedConfig.BaseURL)

	c := &OpenAIClientV2{
		client:          client,
		config:          resolvedConfig,
		useResponsesAPI: useResponsesAPI,
	}
	c.WarnIfUnknownModel()
	return c, nil
}

// Chat 发送聊天请求
func (c *OpenAIClientV2) Chat(ctx context.Context, messages []*ChatMessage, options *ChatOptions) (*LLMResponse, error) {
	// 转换消息格式
	openaiMessages := convertToOpenAIMessages(messages)

	// 构建请求
	req := openai.ChatCompletionRequest{
		Model:       c.config.Model,
		Messages:    openaiMessages,
		Temperature: float32(options.Temperature),
		MaxTokens:   int(options.MaxTokens),
	}

	// 处理停止序列
	if len(options.Stop) > 0 {
		req.Stop = options.Stop
	}

	// 处理工具
	if len(options.Tools) > 0 {
		req.Tools = convertToOpenAITools(options.Tools)
	}

	// 发送请求
	resp, err := c.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "OpenAI API error")
	}

	// 转换响应
	return convertFromOpenAIResponse(&resp), nil
}

// ChatWithReasoning 发送带推理内容的聊天请求（用于 DeepSeek）
func (c *OpenAIClientV2) ChatWithReasoning(ctx context.Context, messages []*ChatMessage, options *ChatOptions, reasoningContents map[int]string) (*LLMResponse, error) {
	// 转换消息格式，附加 reasoning_content
	openaiMessages := convertToOpenAIMessages(messages)
	for i, msg := range messages {
		// 如果有 reasoning_content，添加到消息中
		if rc, ok := reasoningContents[i]; ok && rc != "" {
			// DeepSeek 需要将 reasoning_content 作为额外字段传递
			// go-openai 不直接支持，需要使用自定义请求
			openaiMessages[i] = openai.ChatCompletionMessage{
				Role:    msg.Role,
				Content: msg.Content,
			}
		}
	}

	// 构建请求
	req := openai.ChatCompletionRequest{
		Model:       c.config.Model,
		Messages:    openaiMessages,
		Temperature: float32(options.Temperature),
		MaxTokens:   int(options.MaxTokens),
	}

	// 处理停止序列
	if len(options.Stop) > 0 {
		req.Stop = options.Stop
	}

	// 处理工具
	if len(options.Tools) > 0 {
		req.Tools = convertToOpenAITools(options.Tools)
	}

	// 发送请求
	resp, err := c.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "OpenAI API error")
	}

	// 转换响应
	return convertFromOpenAIResponse(&resp), nil
}

// ChatWithReasoningSplit 发送带 reasoning_split 的聊天请求（用于 MiniMax M2.x）
func (c *OpenAIClientV2) ChatWithReasoningSplit(ctx context.Context, messages []*ChatMessage, options *ChatOptions) (*LLMResponse, error) {
	// 转换消息格式
	openaiMessages := convertToOpenAIMessages(messages)

	// 构建请求
	req := openai.ChatCompletionRequest{
		Model:       c.config.Model,
		Messages:    openaiMessages,
		Temperature: float32(options.Temperature),
		MaxTokens:   int(options.MaxTokens),
	}

	// 处理停止序列
	if len(options.Stop) > 0 {
		req.Stop = options.Stop
	}

	// 处理工具
	if len(options.Tools) > 0 {
		req.Tools = convertToOpenAITools(options.Tools)
	}

	// MiniMax M2.x 需要 reasoning_split 参数
	// go-openai 不直接支持，需要使用自定义 HTTP 请求
	return c.createChatCompletionWithExtraBody(ctx, &req, map[string]any{"reasoning_split": true})
}

// createChatCompletionWithExtraBody 使用额外参数发送请求
func (c *OpenAIClientV2) createChatCompletionWithExtraBody(ctx context.Context, req *openai.ChatCompletionRequest, extraBody map[string]any) (*LLMResponse, error) {
	// 将请求序列化为 JSON
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal request")
	}

	// 合并 extraBody
	var reqMap map[string]any
	if err := json.Unmarshal(reqBytes, &reqMap); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal request")
	}

	for k, v := range extraBody {
		reqMap[k] = v
	}

	// 重新序列化
	finalReqBytes, err := json.Marshal(reqMap)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal final request")
	}

	// 使用自定义 HTTP 请求
	httpReq, err := c.buildChatCompletionRequest(ctx, finalReqBytes)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build chat completion request")
	}

	// 发送请求
	httpClient := &http.Client{}
	httpResp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, errors.Wrap(err, "HTTP request error")
	}
	defer httpResp.Body.Close()

	// 解析响应
	var resp openai.ChatCompletionResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&resp); err != nil {
		return nil, errors.Wrap(err, "failed to decode response")
	}

	return convertFromOpenAIResponse(&resp), nil
}

// buildChatCompletionRequest 构建 HTTP 请求
func (c *OpenAIClientV2) buildChatCompletionRequest(ctx context.Context, body []byte) (*http.Request, error) {
	baseURL := c.config.BaseURL
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	reqURL := fmt.Sprintf("%s/chat/completions", strings.TrimSuffix(baseURL, "/"))
	httpReq, err := http.NewRequestWithContext(ctx, "POST", reqURL, bytes.NewReader(body))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.config.APIKey))

	return httpReq, nil
}

// Name 返回客户端名称
func (c *OpenAIClientV2) Name() string {
	return string(c.config.Provider)
}

// Provider 返回服务提供商
func (c *OpenAIClientV2) Provider() LLMProvider {
	return c.config.Provider
}

// GetProviderName 获取提供商名称（用于警告消息）
func (c *OpenAIClientV2) GetProviderName() string {
	return string(c.config.Provider)
}

// WarnIfUnknownModel 如果模型未知则发出警告
func (c *OpenAIClientV2) WarnIfUnknownModel() {
	if c.ValidateModel() {
		return
	}
	// 发出警告
	logger.L().Warn("Model is not in the known model list",
		zap.String("model", c.config.Model),
		zap.String("provider", c.GetProviderName()))
}

// GetLLM 获取配置的 LLM 客户端实例
func (c *OpenAIClientV2) GetLLM() *OpenAIClientV2 {
	c.WarnIfUnknownModel()
	return c
}

// Helper functions
func getTimeout(timeout int) int {
	if timeout <= 0 {
		return 60
	}
	return timeout
}

func convertToOpenAIMessages(messages []*ChatMessage) []openai.ChatCompletionMessage {
	result := make([]openai.ChatCompletionMessage, len(messages))
	for i, msg := range messages {
		result[i] = openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}
	return result
}

func convertToOpenAITools(tools []*Tool) []openai.Tool {
	result := make([]openai.Tool, len(tools))
	for i, tool := range tools {
		result[i] = openai.Tool{
			Type: openai.ToolType(tool.Type),
			Function: &openai.FunctionDefinition{
				Name:        tool.Function.Name,
				Description: tool.Function.Description,
				Parameters:  tool.Function.Parameters,
			},
		}
	}
	return result
}

func convertFromOpenAIResponse(resp *openai.ChatCompletionResponse) *LLMResponse {
	result := &LLMResponse{
		Model: resp.Model,
		Usage: Usage{
			PromptTokens:     int64(resp.Usage.PromptTokens),
			CompletionTokens: int64(resp.Usage.CompletionTokens),
			TotalTokens:      int64(resp.Usage.TotalTokens),
		},
	}

	if len(resp.Choices) > 0 {
		choice := resp.Choices[0]
		result.Content = choice.Message.Content
		result.FinishReason = string(choice.FinishReason)

		// 转换工具调用
		if len(choice.Message.ToolCalls) > 0 {
			result.ToolCalls = make([]ToolCall, len(choice.Message.ToolCalls))
			for i, tc := range choice.Message.ToolCalls {
				result.ToolCalls[i] = ToolCall{
					ID:   tc.ID,
					Type: string(tc.Type),
					Function: FunctionCall{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				}
			}
		}
	}

	return result
}

// ProviderSpec 提供商配置
type ProviderSpec struct {
	BaseURL         string // 默认 base URL
	BaseURLEnv      string // 环境变量名，覆盖 BaseURL
	KeyOptional     bool   // API key 是否可选
	PlaceholderKey  string // 无 key 时使用的占位符
	RequireBaseURL  bool   // 是否必须提供 base_url
	UseResponsesAPI bool   // 是否使用 Responses API
}

// IsOpenAICompatible 检查是否为 OpenAI 兼容提供商
func IsOpenAICompatible(provider LLMProvider) bool {
	_, ok := providerSpecs[provider]
	return ok
}

// isNativeOpenAIBaseURL 检查是否为原生 OpenAI URL
func isNativeOpenAIBaseURL(baseURL string) bool {
	if baseURL == "" {
		return true
	}
	// 解析 URL
	if !strings.Contains(baseURL, "://") {
		baseURL = "https://" + baseURL
	}
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return false
	}
	host := parsedURL.Hostname()
	return host == "api.openai.com" || strings.HasSuffix(host, ".openai.com")
}

// ValidateModel 验证模型是否有效
func (c *OpenAIClientV2) ValidateModel() bool {
	return validateModel(c.config.Provider, c.config.Model)
}
