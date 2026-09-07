// Package errs 收敛 wbflow 哨兵错误、统一 Trace 包裹入口与集中 HTTP 错误码映射。
//
// 约定（spec §1.3/§2）：
//   - 业务错误链路一律经 Trace 层层包裹（保留堆栈与包裹链）；
//   - service/domain 返回哨兵错误（或以其为 cause 的包裹错误）；
//   - handler 出参经 MapToHTTP 集中映射，接口永不 panic。
//   - 哨兵错误与各 Service 方法 GoDoc「错误集」交叉引用。
package errs

import (
	"errors"
	"net/http"

	perr "github.com/pingcap/errors"
)

// 哨兵错误清单（spec §1.3）。后续工单的 service 层按 GoDoc 错误集引用。
var (
	// ErrOptimisticLock updated_at 乐观锁冲突：UPDATE 0 行命中。
	ErrOptimisticLock = perr.New("wbflow: 乐观锁冲突（updated_at 不匹配）")
	// ErrGateBlocked 门禁拦截：判据不满足，出参带 blocked:true + reason。
	ErrGateBlocked = perr.New("wbflow: 门禁拦截")
	// ErrIdempotentConflict 幂等冲突：幂等键重复且语义不一致。
	ErrIdempotentConflict = perr.New("wbflow: 幂等冲突")
	// ErrStaleReport 过期上报：complete 判据校验不满足或状态过期。
	ErrStaleReport = perr.New("wbflow: 过期上报")
	// ErrInvalidArgument 入参校验失败：必填缺失 / 枚举外值 / 取值非法。
	ErrInvalidArgument = perr.New("wbflow: 入参非法")
	// ErrNotFound 目标资源不存在：业务 id 未命中。
	ErrNotFound = perr.New("wbflow: 资源不存在")
	// ErrDependencyCycle 依赖环：flows.dependencies 依赖图成环，树不可构建（T4）。
	ErrDependencyCycle = perr.New("wbflow: 依赖环")
	// ErrInvalidTransition 非法状态迁移：issue 状态机 v2 拒绝该操作（当前态 ×
	// 动作 × 操作者约束任一不成立，T7）。
	ErrInvalidTransition = perr.New("wbflow: 非法状态迁移")
)

// Trace 统一错误包裹入口：内部错误链路层层包裹，保留堆栈。
// nil 直通；已被 pingcap/errors 带堆栈包裹的错误重复包裹无害（仅在边界包一次）。
func Trace(err error) error {
	if err == nil {
		return nil
	}
	return perr.Trace(err)
}

// Code 机器可读错误码（统一响应出参 code 字段）。
type Code string

// 错误码清单。
const (
	CodeOptimisticLock     Code = "OPTIMISTIC_LOCK"
	CodeGateBlocked        Code = "GATE_BLOCKED"
	CodeIdempotentConflict Code = "IDEMPOTENT_CONFLICT"
	CodeStaleReport        Code = "STALE_REPORT"
	CodeInvalidArgument    Code = "INVALID_ARGUMENT"
	CodeNotFound           Code = "NOT_FOUND"
	CodeDependencyCycle    Code = "DEPENDENCY_CYCLE"
	CodeInvalidTransition  Code = "INVALID_TRANSITION"
	CodeUnauthorized       Code = "UNAUTHORIZED"
	CodeInternal           Code = "INTERNAL"
)

// HTTPMapping 哨兵错误到 HTTP 出参的集中映射结果。
type HTTPMapping struct {
	// HTTPStatus HTTP 状态码
	HTTPStatus int
	// Code 机器可读错误码
	Code Code
	// Blocked 门禁拦截标记（统一响应 blocked 字段）
	Blocked bool
}

// MapToHTTP 集中映射：error 链路里命中哨兵错误即映射；
// 未命中一律 INTERNAL（500），不泄漏内部细节。
func MapToHTTP(err error) HTTPMapping {
	switch {
	case err == nil:
		return HTTPMapping{HTTPStatus: http.StatusOK, Code: ""}
	case Is(err, ErrGateBlocked):
		return HTTPMapping{HTTPStatus: http.StatusConflict, Code: CodeGateBlocked, Blocked: true}
	case Is(err, ErrIdempotentConflict):
		return HTTPMapping{HTTPStatus: http.StatusConflict, Code: CodeIdempotentConflict}
	case Is(err, ErrOptimisticLock):
		return HTTPMapping{HTTPStatus: http.StatusConflict, Code: CodeOptimisticLock}
	case Is(err, ErrStaleReport):
		return HTTPMapping{HTTPStatus: http.StatusConflict, Code: CodeStaleReport}
	case Is(err, ErrInvalidArgument):
		return HTTPMapping{HTTPStatus: http.StatusBadRequest, Code: CodeInvalidArgument}
	case Is(err, ErrNotFound):
		return HTTPMapping{HTTPStatus: http.StatusNotFound, Code: CodeNotFound}
	case Is(err, ErrDependencyCycle):
		return HTTPMapping{HTTPStatus: http.StatusConflict, Code: CodeDependencyCycle}
	case Is(err, ErrInvalidTransition):
		return HTTPMapping{HTTPStatus: http.StatusConflict, Code: CodeInvalidTransition}
	default:
		return HTTPMapping{HTTPStatus: http.StatusInternalServerError, Code: CodeInternal}
	}
}

// Is 判断错误链路是否命中目标。
//
// pingcap/errors v0.11.4 的 tracer 只暴露 Cause() 不实现 Unwrap()，
// std errors.Is 走不通 Trace 包裹链；此处双链遍历：先 std Unwrap 链，
// 再 pingcap Cause 链。rd 内哨兵判定一律走本函数（收拢点），
// handler 出参统一经 MapToHTTP 消费，无旁路。
func Is(err error, target error) bool {
	for cur := err; cur != nil; {
		if cur == target || errors.Is(cur, target) {
			return true
		}
		causer, ok := cur.(interface{ Cause() error })
		if !ok {
			return false
		}
		cur = causer.Cause()
	}
	return false
}
