// llm 包单测：FakeClient/ChatClient/围栏剥离/findings 宽解析/重试器全分支
// （-L4 工单闸门：成功/围栏/解析失败/重试耗尽/恢复）。传输层用 httptest 起
// 本地端点，不依赖外部 DSH 网关。
package llm

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/huzilin/wbflow/rd/pkg/errs"
)

// --- FakeClient 全分支 ---

func TestUnit_FakeClient_Success(t *testing.T) {
	fake := &FakeClient{ChatText: "产物正文", Usage: Usage{PromptTokens: 11, CompletionTokens: 7}}
	text, usage, err := fake.Complete(context.Background(), "system", "user")
	if err != nil {
		t.Fatalf("Complete 返回错误: %v", err)
	}
	if text != "产物正文" {
		t.Fatalf("产物不匹配: %q", text)
	}
	if usage.PromptTokens != 11 || usage.CompletionTokens != 7 {
		t.Fatalf("用量不匹配: %+v", usage)
	}
	if fake.Model() != "fake" {
		t.Fatalf("替身缺省模型名应可辨识: %q", fake.Model())
	}
	if (&FakeClient{ModelName: "gpt-x"}).Model() != "gpt-x" {
		t.Fatal("显式模型名应原样自报")
	}
}

func TestUnit_FakeClient_Error(t *testing.T) {
	sentinel := errors.New("注入失败")
	_, _, err := (&FakeClient{ChatErr: sentinel}).Complete(context.Background(), "s", "u")
	if !errors.Is(err, sentinel) {
		t.Fatalf("应原样回传注入错误: %v", err)
	}
}

// --- ChatClient（httptest 端点）---

func TestUnit_ChatClient_SuccessAndUsageFallback(t *testing.T) {
	var gotAuth, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		// cost_yuan 缺失回落 cost 字段
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"你好"}}],"usage":{"prompt_tokens":3,"completion_tokens":5,"cost":0.42}}`))
	}))
	defer srv.Close()
	client := NewChatClient(srv.URL+"/", "sk-test", "deepseek-v4-flash", 5*time.Second, nil)
	text, usage, err := client.Complete(context.Background(), "s", "u")
	if err != nil {
		t.Fatalf("Complete 返回错误: %v", err)
	}
	if text != "你好" {
		t.Fatalf("正文不匹配: %q", text)
	}
	if usage.PromptTokens != 3 || usage.CompletionTokens != 5 || usage.CostYuan != 0.42 {
		t.Fatalf("用量不匹配: %+v", usage)
	}
	if gotPath != "/chat/completions" {
		t.Fatalf("端点拼法应为基础 URL 追加 /chat/completions: %q", gotPath)
	}
	if gotAuth != "Bearer sk-test" {
		t.Fatalf("Bearer 鉴权头缺失: %q", gotAuth)
	}
	if client.Model() != "deepseek-v4-flash" {
		t.Fatalf("模型名自报不匹配: %q", client.Model())
	}
}

func TestUnit_ChatClient_NoKeyNoHeader(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"}}]}`))
	}))
	defer srv.Close()
	client := NewChatClient(srv.URL, "", "m", time.Second, nil)
	if _, _, err := client.Complete(context.Background(), "s", "u"); err != nil {
		t.Fatalf("免鉴权调用应成功: %v", err)
	}
	if gotAuth != "" {
		t.Fatalf("apiKey 空串不应设 Authorization 头: %q", gotAuth)
	}
	// 实证必需件：DisableKeepAlives 必须保留（漏掉必复现网关断连后挂起超时坑）
	tr, ok := client.httpc.Transport.(*http.Transport)
	if !ok || !tr.DisableKeepAlives {
		t.Fatal("传输内核必须 DisableKeepAlives: true（novel 真机 E2E 实证件）")
	}
}

func TestUnit_ChatClient_ErrLLMBranches(t *testing.T) {
	// 非 2xx（错误体截断 512 字节内）
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("upstream exploded"))
	}))
	defer srv.Close()
	client := NewChatClient(srv.URL, "k", "m", time.Second, nil)
	_, _, err := client.Complete(context.Background(), "s", "u")
	if !errs.Is(err, errs.ErrLLM) {
		t.Fatalf("非 2xx 应归 ErrLLM 哨兵: %v", err)
	}
	if !strings.Contains(err.Error(), "502") {
		t.Fatalf("错误应携带状态码: %v", err)
	}

	// 空 Choices
	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[]}`))
	}))
	defer srv2.Close()
	_, _, err = NewChatClient(srv2.URL, "", "m", time.Second, nil).Complete(context.Background(), "s", "u")
	if !errs.Is(err, errs.ErrLLM) {
		t.Fatalf("空 Choices 应归 ErrLLM: %v", err)
	}

	// 连接失败（坏端口）
	_, _, err = NewChatClient("http://127.0.0.1:1", "", "m", time.Second, nil).Complete(context.Background(), "s", "u")
	if !errs.Is(err, errs.ErrLLM) {
		t.Fatalf("连接失败应归 ErrLLM: %v", err)
	}
}

// --- 围栏剥离 ---

func TestUnit_StripFences(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{`{"a":1}`, `{"a":1}`},
		{"```json\n{\"a\":1}\n```", `{"a":1}`},
		{"评审结论如下：\n{\"agree\":true,\"note\":\"结论\"}\n以上。", `{"agree":true,"note":"结论"}`},
		{"无 JSON 文本", "无 JSON 文本"},
	}
	for _, c := range cases {
		if got := StripFences(c.in); got != c.want {
			t.Fatalf("StripFences(%q)=%q want %q", c.in, got, c.want)
		}
	}
}

func TestUnit_StripFencesAny_ArrayAndEscape(t *testing.T) {
	if got := StripFencesAny("```json\n[{\"a\":\"x\\\"y\"},{\"b\":2}]\n```"); got != `[{"a":"x\"y"},{"b":2}]` {
		t.Fatalf("数组顶层+转义剥离失败: %q", got)
	}
	if got := StripFencesAny("说明 {\"a\":1} 尾"); got != `{"a":1}` {
		t.Fatalf("对象剥离失败: %q", got)
	}
	// 字符串内的 }/帶转义引号不破坏配对（inStr 状态机；StripFences 无此感知，
	// 字符串含 } 的产物走本函数口径）
	if got := StripFencesAny(`{"note":"含 } 花括号","n":1}`); got != `{"note":"含 } 花括号","n":1}` {
		t.Fatalf("字符串内 } 应保持配对: %q", got)
	}
}

// --- findings 宽口径解析 ---

func TestUnit_ParseReviewFindings_ArrayShape(t *testing.T) {
	raw := "```json\n{\"pass\":false,\"findings\":[{\"desc\":\"第三行缺少错误处理\",\"startLine\":3,\"endLine\":3},{\"desc\":\"整体结构松散\",\"startLine\":\"5\",\"endLine\":\"8\"}]}\n```"
	fs := ParseReviewFindings(raw)
	if len(fs) != 2 {
		t.Fatalf("应解析出 2 条意见: %+v", fs)
	}
	if fs[0].StartLine != 3 || fs[0].EndLine != 3 {
		t.Fatalf("数字行号不匹配: %+v", fs[0])
	}
	if fs[1].StartLine != 5 || fs[1].EndLine != 8 {
		t.Fatalf("字符串行号应宽松收下: %+v", fs[1])
	}
}

func TestUnit_ParseReviewFindings_Degradations(t *testing.T) {
	// 字符串形状回退单条
	if fs := ParseReviewFindings(`{"findings":"整体偏长"}`); len(fs) != 1 || fs[0].Desc != "整体偏长" {
		t.Fatalf("字符串形状应回退单条: %+v", ParseReviewFindings(`{"findings":"整体偏长"}`))
	}
	// null/空/缺失归空数组
	for _, raw := range []string{`{"agree":true,"findings":null}`, `{"agree":true}`, ""} {
		if fs := ParseReviewFindings(raw); len(fs) != 0 {
			t.Fatalf("%q 应归空数组: %+v", raw, fs)
		}
	}
	// 整体非 JSON → 单条兜底不中断
	fs := ParseReviewFindings("这段实现没有处理空列表场景。")
	if len(fs) != 1 || !strings.Contains(fs[0].Desc, "空列表") {
		t.Fatalf("非 JSON 应单条兜底: %+v", fs)
	}
	// desc 塞围栏净化为单行（desc 内嵌 ```json 围栏与转义 JSON——真机缺陷形态）
	desc := "```json\\n{\\\"x\\\":1}\\n``` 册除"
	fs = ParseReviewFindings("{\"findings\":[{\"desc\":\"" + desc + "\"}]}")
	if len(fs) != 1 || strings.Contains(fs[0].Desc, "```") || strings.Contains(fs[0].Desc, "\n") {
		t.Fatalf("desc 围栏应净化为单行: %+v", fs)
	}
}

// --- 重试器（成功/恢复/耗尽）---

func TestUnit_CompleteWithRetry_FirstAttemptSuccess(t *testing.T) {
	calls := 0
	client := &countingClient{onCall: func() error { calls++; return nil }}
	text, _, err := CompleteWithRetry(context.Background(), client, "s", "u", RetryOptions{Max: 6, Sleep: noSleep})
	if err != nil || text != "ok" {
		t.Fatalf("首次成功不应重试: err=%v text=%q", err, text)
	}
	if calls != 1 {
		t.Fatalf("调用次数应 1: %d", calls)
	}
}

func TestUnit_CompleteWithRetry_Recovery(t *testing.T) {
	calls := 0
	var backoffs []time.Duration
	client := &countingClient{onCall: func() error {
		calls++
		if calls < 3 {
			return errs.Trace(errors.New("临时故障"))
		}
		return nil
	}}
	text, _, err := CompleteWithRetry(context.Background(), client, "s", "u", RetryOptions{Max: 6, Sleep: recordSleep(&backoffs)})
	if err != nil || text != "ok" {
		t.Fatalf("第 3 次应恢复成功: err=%v", err)
	}
	if calls != 3 {
		t.Fatalf("调用次数应 3: %d", calls)
	}
	if len(backoffs) != 2 || backoffs[0] != time.Second || backoffs[1] != 2*time.Second {
		t.Fatalf("指数退避序列不匹配: %v", backoffs)
	}
}

func TestUnit_CompleteWithRetry_Exhausted(t *testing.T) {
	var backoffs []time.Duration
	attemptErrs := 0
	client := &countingClient{onCall: func() error {
		return errs.Trace(fmt.Errorf("%w: 网关持续 502", errs.ErrLLM))
	}}
	_, _, err := CompleteWithRetry(context.Background(), client, "s", "u", RetryOptions{
		Max: 4, Sleep: recordSleep(&backoffs),
		OnAttempt: func(_ int, attemptErr error) {
			if attemptErr != nil {
				attemptErrs++
			}
		},
	})
	if !errs.Is(err, errs.ErrLLM) {
		t.Fatalf("耗尽错误应保留 ErrLLM 哨兵链: %v", err)
	}
	if !strings.Contains(err.Error(), "重试耗尽") || !strings.Contains(err.Error(), "4 次") {
		t.Fatalf("耗尽文案不匹配: %v", err)
	}
	if attemptErrs != 4 {
		t.Fatalf("OnAttempt 应逐次回调失败: %d", attemptErrs)
	}
	// 1s 2s 4s（封顶前），封顶 30s
	if len(backoffs) != 3 || backoffs[0] != time.Second || backoffs[2] != 4*time.Second {
		t.Fatalf("退避序列不匹配: %v", backoffs)
	}
}

func TestUnit_CompleteWithRetry_BackoffCap(t *testing.T) {
	var backoffs []time.Duration
	client := &countingClient{onCall: func() error { return errs.Trace(errors.New("x")) }}
	_, _, _ = CompleteWithRetry(context.Background(), client, "s", "u", RetryOptions{
		Max: 7, Base: 10 * time.Second, MaxBackoff: 20 * time.Second, Sleep: recordSleep(&backoffs),
	})
	for _, d := range backoffs {
		if d > 20*time.Second {
			t.Fatalf("退避应封顶 20s: %v", backoffs)
		}
	}
}

// countingClient 调用计数替身（onCall 返回 nil=成功产出 "ok"）。
type countingClient struct {
	onCall func() error
	mu     sync.Mutex
}

func (c *countingClient) Complete(_ context.Context, _ string, _ string) (string, Usage, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.onCall(); err != nil {
		return "", Usage{}, err
	}
	return "ok", Usage{}, nil
}

// noSleep 单测即时退避（不采集）。
func noSleep(_ context.Context, _ time.Duration) bool { return true }

// recordSleep 采集退避序列的即时退避。
func recordSleep(sink *[]time.Duration) func(context.Context, time.Duration) bool {
	return func(_ context.Context, d time.Duration) bool {
		*sink = append(*sink, d)
		return true
	}
}
