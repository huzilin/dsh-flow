// llm 包本文件：测试替身（novel be/llm/fake.go C6 copy，方法集对齐 rd 通路面
// 裁剪后的最小接口）。colleague 规范「假模型返回确定产物」的标准实现：
// ChatText/ChatErr 驱动 Complete 通路，固定 Usage 注入；base_url 空的演示态装配
// 亦落在此实现（main 装配 WARN 提示）。
package llm

import "context"

// FakeClient 测试替身：Complete 通路返回固定产物/固定错误/固定 Usage。
// ChatText 空串时回落占位正文（只配错误场景的替身也能驱动非错误分支）。
type FakeClient struct {
	// ChatText Complete 通路固定产物
	ChatText string
	// ChatErr Complete 通路固定错误（非 nil 时每次调用返回）
	ChatErr error
	// Usage 各次调用统一回传的固定用量（零值=网关未给用量）
	Usage Usage
	// ModelName 自报模型名；空=缺省 "fake"（替身链路日志可辨识）
	ModelName string
}

// Model 自报模型名。
func (f *FakeClient) Model() string {
	if f.ModelName == "" {
		return "fake"
	}
	return f.ModelName
}

// Complete 裸对话补全替身（与 ChatClient.Complete 同形）。
func (f *FakeClient) Complete(_ context.Context, _ string, _ string) (string, Usage, error) {
	text := f.ChatText
	if text == "" {
		text = "（替身产物占位）"
	}
	return text, f.Usage, f.ChatErr
}
