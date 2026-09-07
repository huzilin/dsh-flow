package errs

import (
	"errors"
	"fmt"
	"testing"

	"gotest.tools/v3/assert"
)

func TestUnit_Trace_KeepsSentinelChain(t *testing.T) {
	// pingcap tracer 未实现 Unwrap：std errors.Is 走不通 Trace 链，errs.Is 双链可穿透
	assert.Assert(t, Is(Trace(ErrOptimisticLock), ErrOptimisticLock))

	// std error 经 %w + Trace 包裹后同样穿透到哨兵
	plain := errors.New("db write failed")
	wrapped := Trace(fmt.Errorf("repo update: %w", plain))
	assert.Assert(t, Is(wrapped, plain))
	assert.Assert(t, wrapped != plain, "Trace 对无堆栈错误应产生新包裹层")
	assert.Assert(t, Is(Trace(Trace(ErrGateBlocked)), ErrGateBlocked))
}

func TestUnit_Trace_NilPassthrough(t *testing.T) {
	assert.Assert(t, Trace(nil) == nil)
}

func TestUnit_MapToHTTP_AllSentinels(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode Code
		wantHTTP int
		wantBlk  bool
	}{
		{"gate blocked", ErrGateBlocked, CodeGateBlocked, 409, true},
		{"gate blocked wrapped", Trace(ErrGateBlocked), CodeGateBlocked, 409, true},
		{"idempotent conflict", ErrIdempotentConflict, CodeIdempotentConflict, 409, false},
		{"optimistic lock", ErrOptimisticLock, CodeOptimisticLock, 409, false},
		{"stale report", ErrStaleReport, CodeStaleReport, 409, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mapping := MapToHTTP(tc.err)
			assert.Equal(t, mapping.Code, tc.wantCode)
			assert.Equal(t, mapping.HTTPStatus, tc.wantHTTP)
			assert.Equal(t, mapping.Blocked, tc.wantBlk)
		})
	}
}

func TestUnit_MapToHTTP_UnknownIsInternal(t *testing.T) {
	mapping := MapToHTTP(errors.New("boom"))
	assert.Equal(t, mapping.Code, CodeInternal)
	assert.Equal(t, mapping.HTTPStatus, 500)
	assert.Equal(t, mapping.Blocked, false)
}

func TestUnit_MapToHTTP_Nil(t *testing.T) {
	assert.Equal(t, MapToHTTP(nil).HTTPStatus, 200)
}
