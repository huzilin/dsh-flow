// Package llm LLM 通路客户端内核（spec 注记 -L4：exec/review 环节的真执行必须经
// DSH 网关调用底层大模型）。
//
// 分层（novel be/llm 取证调研 C1-C6 copy 纪律）：
//   - chat.go：OpenAI 兼容传输内核（POST {baseURL}/chat/completions + Bearer，
//     DisableKeepAlives 实证必需件——网关断连后 Go 连接池复用半开连接表现为每次
//     请求挂起直到超时，真机 E2E 实测）；
//   - fence.go：围栏剥离/平衡块提取（模型实测会包 ```json，不依赖围栏字面量）；
//   - review.go：评审通路 JSON 形状契约 prompt 装配 + findings 宽口径解析
//     （数组/字符串/null 三收，解析降级不中断）；
//   - retry.go：指数退避重试器（-L4 执行拍板 MIG-09 失败策略 B+：重试耗尽才回退）；
//   - fake.go：测试替身（确定性产物注入，全分支单测接缝）。
//
// 定位（MIG-08）：adapter = 「DSH 模型通路的客户端」——模型路由/账号/鉴权归
// DSH 管，本包只做单轮补全与结构化输出解析；prompt 素材（skill/inline）由
// service 层按能力数据组装传入，不在本包写死业务文本。
package llm

import (
	"context"
)

// Completer 最小补全接口（与完整客户端解耦的传输面）：单轮 system+user 补全，
// 返回文本与用量。生产实现 ChatClient（经 DSH 网关）与替身 FakeClient 均实现之。
type Completer interface {
	// Complete 单轮对话补全：system=角色/形状契约段，user=素材段。
	Complete(ctx context.Context, system string, user string) (string, Usage, error)
}

// ModelReporter 自报模型名（埋点/日志分组用；替身缺省 "fake" 可辨识）。
type ModelReporter interface {
	Model() string
}

// Usage 单次调用用量（响应 usage 段解析；cost_yuan 缺省回落 cost 字段，
// 网关均未给则 0 不编造）。一期不落库（ai_call_log 裁剪，调研 D9），仅日志观测。
type Usage struct {
	PromptTokens     int64   `json:"prompt_tokens"`
	CompletionTokens int64   `json:"completion_tokens"`
	CostYuan         float64 `json:"cost_yuan"`
}

// IsZero 零值判定（无用量信息）。
func (u Usage) IsZero() bool {
	return u.PromptTokens == 0 && u.CompletionTokens == 0 && u.CostYuan == 0
}

// 编译期断言：生产实现与替身都必须满足最小接口（novel BUG-2 教训——
// 接口实现完整性要有编译期断言，缺方法编译期即红）。
var (
	_ Completer     = (*ChatClient)(nil)
	_ Completer     = (*FakeClient)(nil)
	_ ModelReporter = (*ChatClient)(nil)
	_ ModelReporter = (*FakeClient)(nil)
)
