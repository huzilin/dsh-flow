// llm 包本文件：指数退避重试器（spec 注记 -L4 执行拍板 MIG-09 失败策略 B+：
// 指数退避重试 N 次（llm.retry_max 参数化）仍失败才回退+need_intervene；重试
// 骨架参考 novel be/task/worker.go 退避序列，模式 copy 非直接 copy——rd 无
// worker 轮询，重试内聚在执行器调用点）。确定性错误（预检 fail fast）不进重试器。
package llm

import (
	"context"
	"fmt"
	"time"

	"github.com/huzilin/wbflow/rd/pkg/errs"
)

// RetryOptions 重试器参数（service 层按 config llm 段装配；测试注入 Sleep 时钟）。
type RetryOptions struct {
	// Max 总尝试次数上限（含首次；拍板口径「重试 6 次」= 6 次尝试，默认 6）
	Max int
	// Base 首次退避间隔（默认 1s）
	Base time.Duration
	// MaxBackoff 单次退避上限（默认 30s，指数封顶防长尾）
	MaxBackoff time.Duration
	// Sleep 退避等待注入点（生产 time 等待；单测注入即时返回采集退避序列）。
	// 返回 false=ctx 取消/停止，重试器立即终止并返回最后一次错误。
	Sleep func(ctx context.Context, d time.Duration) bool
	// OnAttempt 每次尝试结束回调（attempt 从 1 起；err=nil=该次成功）——
	// 执行器借此写任务 last_progress 心跳（scanner 停滞判定的活性证据）。
	OnAttempt func(attempt int, err error)
}

// normalize 补默认值（零值安全：Max≤0=1 次即不重试；Base/MaxBackoff 缺省）。
func (o RetryOptions) normalize() RetryOptions {
	if o.Max < 1 {
		o.Max = 1
	}
	if o.Base <= 0 {
		o.Base = time.Second
	}
	if o.MaxBackoff <= 0 {
		o.MaxBackoff = 30 * time.Second
	}
	if o.Sleep == nil {
		o.Sleep = func(ctx context.Context, d time.Duration) bool {
			timer := time.NewTimer(d)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return false
			case <-timer.C:
				return true
			}
		}
	}
	return o
}

// CompleteWithRetry 带指数退避的重试补全：至多 opts.Max 次调用 complete，失败按
// base×2^(n-1) 退避（封顶 MaxBackoff）后重试；全部耗尽返回包裹错误（调用方据
// errs.Is(err, errs.ErrLLM) 分流回退）。ctx 取消立即终止（服务优雅退出联动）。
//
// 每次尝试结束触发 opts.OnAttempt（含失败次）——执行器写心跳，scanner 据此区分
// 「重试中（活性）」与「异常中断（停滞，恢复执行而非回退）」。
func CompleteWithRetry(ctx context.Context, c Completer, system string, user string, opts RetryOptions) (string, Usage, error) {
	opts = opts.normalize()
	var (
		text    string
		usage   Usage
		lastErr error
	)
	for attempt := 1; attempt <= opts.Max; attempt++ {
		text, usage, lastErr = c.Complete(ctx, system, user)
		if opts.OnAttempt != nil {
			opts.OnAttempt(attempt, lastErr)
		}
		if lastErr == nil {
			return text, usage, nil
		}
		if ctx.Err() != nil {
			return "", usage, errs.Trace(lastErr)
		}
		if attempt == opts.Max {
			break
		}
		backoff := opts.Base * (1 << (attempt - 1))
		if backoff > opts.MaxBackoff {
			backoff = opts.MaxBackoff
		}
		if !opts.Sleep(ctx, backoff) {
			return "", usage, errs.Trace(lastErr)
		}
	}
	return "", usage, errs.Trace(fmt.Errorf("LLM 补全重试耗尽（%d 次）: %w", opts.Max, lastErr))
}
