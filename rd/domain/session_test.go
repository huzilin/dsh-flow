// T10 domain 会话状态迁移表驱动单测（AC：status 迁移全分支 + 非法迁移拦截，
// 无 IO 全覆盖）：合法链 active→ended→disposed、同态幂等 no-op、跳段/回退/终态
// 后再迁一律拒绝。
package domain

import (
	"testing"

	"gotest.tools/v3/assert"
)

// TestUnit_TransitionSessionStatus_Legal 合法迁移逐条（与 sessionTransitions 定义
// 一一对应，防手误漂移）。
func TestUnit_TransitionSessionStatus_Legal(t *testing.T) {
	cases := []struct {
		name        string
		from        string
		to          string
		wantChanged bool
	}{
		{"active→ended", SessionStatusActive, SessionStatusEnded, true},
		{"ended→disposed", SessionStatusEnded, SessionStatusDisposed, true},
		{"active 同态 no-op", SessionStatusActive, SessionStatusActive, false},
		{"ended 同态 no-op", SessionStatusEnded, SessionStatusEnded, false},
		{"disposed 同态 no-op", SessionStatusDisposed, SessionStatusDisposed, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			allowed, changed := TransitionSessionStatus(tc.from, tc.to)
			assert.Assert(t, allowed, "%s→%s 应合法", tc.from, tc.to)
			assert.Equal(t, changed, tc.wantChanged)
		})
	}
}

// TestUnit_TransitionSessionStatus_Illegal 非法迁移逐条：跳段（active→disposed）、
// 回退（ended→active / disposed→ended / disposed→active）、未知态。
func TestUnit_TransitionSessionStatus_Illegal(t *testing.T) {
	cases := []struct {
		name string
		from string
		to   string
	}{
		{"active→disposed 跳段", SessionStatusActive, SessionStatusDisposed},
		{"ended→active 回退", SessionStatusEnded, SessionStatusActive},
		{"disposed→ended 回退", SessionStatusDisposed, SessionStatusEnded},
		{"disposed→active 回退", SessionStatusDisposed, SessionStatusActive},
		{"未知态迁 active", "zombie", SessionStatusActive},
		{"active 迁未知态", SessionStatusActive, "zombie"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			allowed, changed := TransitionSessionStatus(tc.from, tc.to)
			assert.Assert(t, !allowed, "%s→%s 应拒绝", tc.from, tc.to)
			assert.Assert(t, !changed)
		})
	}
}
