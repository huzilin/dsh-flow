// Package domain 本文件：会话绑定纯逻辑（T10）——sessions.status 生命周期迁移表
// （spec §0 表 #10 列注释 active/ended/disposed）。
//
// 迁移口径：单向链 active→ended→disposed。
//   - 同态重入 = 幂等 no-op（返回 ok 且 changed=false）；
//   - active→ended / ended→disposed 为合法迁移（changed=true）；
//   - 其余（含 active→disposed 跳段、任何回退、disposed 后任何迁移）一律非法
//     （返回 false，service 层转 errs.ErrInvalidTransition，HTTP 409）。
//
// 无 IO，可单测；迁移表为单一口径，service/repo 不重复维护判定。
package domain

// 会话状态族（sessions.status 三值，spec §0 表 #10 列注释为单一口径）。
const (
	SessionStatusActive   = "active"
	SessionStatusEnded    = "ended"
	SessionStatusDisposed = "disposed"
)

// sessionTransitions 合法迁移边集（表驱动单一口径）。
var sessionTransitions = map[string]string{
	SessionStatusActive: SessionStatusEnded,
	SessionStatusEnded:  SessionStatusDisposed,
}

// TransitionSessionStatus 会话状态迁移判定：
//   - to == from：幂等 no-op，返回 (true, false)；
//   - 合法边（active→ended / ended→disposed）：返回 (true, true)；
//   - 其余（跳段/回退/终态后再迁）：非法迁移，返回 (false, false)。
func TransitionSessionStatus(from string, to string) (allowed bool, changed bool) {
	if from == to {
		return true, false
	}
	if sessionTransitions[from] == to {
		return true, true
	}
	return false, false
}
