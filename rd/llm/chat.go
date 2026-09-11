// llm 包本文件：OpenAI 兼容 Chat Completions 传输内核（novel be/llm/chat.go
// C1 原样 copy，错误构造换 rd pkg/errs 哨兵）。DSH 网关对外形态见 deepseek-harness
// cli e2e：POST {baseURL}/chat/completions + Bearer key，OpenAI 请求/响应形状。
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/huzilin/wbflow/rd/pkg/errs"
)

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
	// Usage 网关回传用量段（OpenAI 兼容形状；cost_yuan 为 0 时回落 cost 字段——
	// 网关用 cost 承载计价时兼容读取，两者都无则 0 不编造）
	Usage struct {
		PromptTokens     int64   `json:"prompt_tokens"`
		CompletionTokens int64   `json:"completion_tokens"`
		CostYuan         float64 `json:"cost_yuan"`
		Cost             float64 `json:"cost"`
	} `json:"usage"`
}

// ChatClient OpenAI 兼容客户端（DSH 网关传输内核）。
type ChatClient struct {
	baseURL string // 如 http://127.0.0.1:3080（本地 DSH；/v1 归 baseURL 携带）
	apiKey  string
	model   string
	// extraHeaders 附加请求头（如网关要求的会话/路由头），逐请求透传
	extraHeaders map[string]string
	httpc        *http.Client
}

// NewChatClient 构造。
// DisableKeepAlives（实证必需件，copy 时必须保留）：LLM 网关侧断连后 Go 连接池
// 会持续复用半开连接，表现为每次请求挂起直到超时（novel 真机 E2E 实测；curl 每次
// 新连接故无此症）。本客户端调用频度低，禁复用无成本。
func NewChatClient(baseURL string, apiKey string, model string, timeout time.Duration, extraHeaders map[string]string) *ChatClient {
	return &ChatClient{
		baseURL:      strings.TrimSuffix(baseURL, "/"),
		apiKey:       apiKey,
		model:        model,
		extraHeaders: extraHeaders,
		httpc: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				DisableKeepAlives: true,
			},
		},
	}
}

// Model 自报模型名（构造时注入的配置值，日志/埋点按此分组）。
func (c *ChatClient) Model() string { return c.model }

// Complete 单轮对话补全 → 文本与用量。连接失败/非 2xx/解析失败/空 Choices 全部
// 归哨兵 errs.ErrLLM（502 语义，调用方据此做失败分流——重试/回退）；非 2xx 时
// 错误体截断 512 字节防大响应刷屏。
func (c *ChatClient) Complete(ctx context.Context, system string, user string) (string, Usage, error) {
	body, err := json.Marshal(chatRequest{
		Model:    c.model,
		Messages: []chatMessage{{Role: "system", Content: system}, {Role: "user", Content: user}},
	})
	if err != nil {
		return "", Usage{}, errs.Trace(fmt.Errorf("%w: 请求序列化失败: %v", errs.ErrLLM, err))
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", Usage{}, errs.Trace(fmt.Errorf("%w: 构造请求失败: %v", errs.ErrLLM, err))
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	for k, v := range c.extraHeaders {
		req.Header.Set(k, v)
	}
	resp, err := c.httpc.Do(req)
	if err != nil {
		// 连接/超时：DSH 网关不可用 → ErrLLM
		return "", Usage{}, errs.Trace(fmt.Errorf("%w: 大模型服务不可用: %v", errs.ErrLLM, err))
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", Usage{}, errs.Trace(fmt.Errorf("%w: 大模型服务异常(%d): %s", errs.ErrLLM, resp.StatusCode, strings.TrimSpace(string(b))))
	}
	var cr chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&cr); err != nil {
		return "", Usage{}, errs.Trace(fmt.Errorf("%w: 响应解析失败: %v", errs.ErrLLM, err))
	}
	if len(cr.Choices) == 0 {
		return "", Usage{}, errs.Trace(fmt.Errorf("%w: 响应无内容", errs.ErrLLM))
	}
	cost := cr.Usage.CostYuan
	if cost == 0 {
		cost = cr.Usage.Cost
	}
	return cr.Choices[0].Message.Content, Usage{
		PromptTokens:     cr.Usage.PromptTokens,
		CompletionTokens: cr.Usage.CompletionTokens,
		CostYuan:         cost,
	}, nil
}
