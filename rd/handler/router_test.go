package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"gotest.tools/v3/assert"

	"github.com/huzilin/wbflow/rd/pkg/errs"
	"github.com/huzilin/wbflow/rd/service"
)

// errsTraceGateBlocked 构造经 Trace 包裹的门禁拦截错误。
func errsTraceGateBlocked() error {
	return errs.Trace(errs.ErrGateBlocked)
}

func init() {
	gin.SetMode(gin.TestMode)
}

// fakeHealth httptest 打桩的 HealthChecker。
type fakeHealth struct {
	status  *service.HealthStatus
	panicAt bool
}

func (f *fakeHealth) Check(ctx context.Context) *service.HealthStatus {
	if f.panicAt {
		panic("boom")
	}
	return f.status
}

func newTestLogger() (*zap.Logger, *observer.ObservedLogs) {
	core, logs := observer.New(zapcore.InfoLevel)
	return zap.New(core), logs
}

func TestUnit_Healthz_RouteAndResponse(t *testing.T) {
	lg, _ := newTestLogger()
	deps := &Deps{
		Logger: lg,
		Health: &fakeHealth{status: &service.HealthStatus{
			Ok:    false,
			State: service.StateDegraded,
			MySQL: service.MySQLStatus{
				Reachable: true,
				Version:   "8.0.40",
				DBState:   service.DBStateMissing,
				Error:     "Error 1049",
			},
		}},
	}
	router := NewRouter(deps)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, resp.Code, http.StatusOK)

	var body service.HealthStatus
	assert.NilError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	// server ok / db missing 可区分上报
	assert.Equal(t, body.MySQL.Reachable, true)
	assert.Equal(t, body.MySQL.DBState, "missing")
	assert.Equal(t, body.State, "degraded")
}

func TestUnit_AccessLog_EntryAndExit(t *testing.T) {
	lg, logs := newTestLogger()
	router := NewRouter(&Deps{Logger: lg, Health: &fakeHealth{status: &service.HealthStatus{}}})

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	entries := logs.All()
	assert.Assert(t, len(entries) >= 2, "入口+出口埋点应各有一条")
	assert.Equal(t, entries[0].Message, "http request start")
	assert.Equal(t, entries[len(entries)-1].Message, "http request done")
	// 出口埋点带 status
	statusField := entries[len(entries)-1].ContextMap()["status"]
	assert.Assert(t, statusField != nil, "出口埋点应带 status 字段")
}

func TestUnit_Recovery_NeverPanic(t *testing.T) {
	lg, logs := newTestLogger()
	router := NewRouter(&Deps{Logger: lg, Health: &fakeHealth{panicAt: true}})

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	resp := httptest.NewRecorder()
	assert.NilError(t, func() error { router.ServeHTTP(resp, req); return nil }())

	assert.Equal(t, resp.Code, http.StatusInternalServerError)

	var body map[string]any
	assert.NilError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	assert.Equal(t, body["ok"], false)
	assert.Equal(t, body["code"], "INTERNAL")
	assert.Assert(t, len(logs.FilterMessage("http handler panic").All()) == 1)
}

func TestUnit_AbortWithErr_SentinelMapping(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/blocked", func(c *gin.Context) {
		AbortWithErr(c, errsTraceGateBlocked())
	})

	req := httptest.NewRequest(http.MethodGet, "/blocked", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	assert.Equal(t, resp.Code, http.StatusConflict)

	var body map[string]any
	assert.NilError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	assert.Equal(t, body["ok"], false)
	assert.Equal(t, body["code"], "GATE_BLOCKED")
	assert.Equal(t, body["blocked"], true)
}
