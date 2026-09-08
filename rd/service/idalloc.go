// 取号重试共享 helper：idAllocRetryLimit 循环同形 6 处收拢（requirement/version/
// task/comment/artifact/escalation）——once 函数返回 retryable=true（唯一键冲突/
// 锁竞争）时整体重试，耗尽返回 ErrIdempotentConflict。单一真相源，新增取号写路径
// 一律复用本函数。
package service

import (
	"github.com/huzilin/wbflow/rd/pkg/errs"
)

// withIDAllocRetry 取号重试骨架：执行 once 直至成功或不可重试错误；retryable
// 错误重试至 idAllocRetryLimit 次，耗尽后返回 ErrIdempotentConflict（409）。
func withIDAllocRetry[T any](once func() (result T, retryable bool, err error)) (T, error) {
	var zero T
	for attempt := 0; attempt < idAllocRetryLimit; attempt++ {
		result, retryable, err := once()
		if err != nil {
			if retryable {
				continue
			}
			return zero, err
		}
		return result, nil
	}
	return zero, errs.Trace(errs.ErrIdempotentConflict)
}
