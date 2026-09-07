// token 认证中间件单测：401 分支（缺头/错 token/未配置 fail closed）、
// 正常放行、/healthz 豁免。不依赖 DB。
package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"gotest.tools/v3/assert"
)

// newAuthTestRouter 构造仅含 auth 探针路由的引擎。
func newAuthTestRouter(token string) *gin.Engine {
	r := gin.New()
	api := r.Group("/api")
	api.Use(TokenAuth(token))
	api.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	return r
}

// doGet 执行 GET 请求并返回 recorder。
func doGet(r *gin.Engine, path string, header map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	for k, v := range header {
		req.Header.Set(k, v)
	}
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	return resp
}

// TestUnit_TokenAuth_RejectBranches 401 三分支：缺头 / 错 token / fail closed。
func TestUnit_TokenAuth_RejectBranches(t *testing.T) {
	r := newAuthTestRouter("secret-1")

	// 1. 缺 Authorization 头
	resp := doGet(r, "/api/ping", nil)
	assert.Equal(t, resp.Code, http.StatusUnauthorized)
	var body map[string]any
	assert.NilError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	assert.Equal(t, body["ok"], false)
	assert.Equal(t, body["code"], "UNAUTHORIZED")

	// 2. 错 token
	resp = doGet(r, "/api/ping", map[string]string{"Authorization": "Bearer wrong"})
	assert.Equal(t, resp.Code, http.StatusUnauthorized)

	// 3. 非 Bearer scheme
	resp = doGet(r, "/api/ping", map[string]string{"Authorization": "Basic secret-1"})
	assert.Equal(t, resp.Code, http.StatusUnauthorized)
}

// TestUnit_TokenAuth_FailClosedWhenUnconfigured token 未配置（空串）= 全 401。
func TestUnit_TokenAuth_FailClosedWhenUnconfigured(t *testing.T) {
	r := newAuthTestRouter("")
	resp := doGet(r, "/api/ping", nil)
	assert.Equal(t, resp.Code, http.StatusUnauthorized)
	resp = doGet(r, "/api/ping", map[string]string{"Authorization": "Bearer anything"})
	assert.Equal(t, resp.Code, http.StatusUnauthorized)
}

// TestUnit_TokenAuth_AcceptAndHealthzExempt 正确 token 放行；/healthz 豁免。
func TestUnit_TokenAuth_AcceptAndHealthzExempt(t *testing.T) {
	r := newAuthTestRouter("secret-1")

	resp := doGet(r, "/api/ping", map[string]string{"Authorization": "Bearer secret-1"})
	assert.Equal(t, resp.Code, http.StatusOK)

	// /healthz 无头直通
	resp = doGet(r, "/healthz", nil)
	assert.Equal(t, resp.Code, http.StatusOK)
}
