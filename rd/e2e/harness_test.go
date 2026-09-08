//go:build e2e

// T17 端到端演练 harness（roadmap P5「seed 注入 → HTTP 驱动全流程」）：
//   - 每条剧本独立真实 MySQL scratch 库（wbflow_e2e_* 前缀，新建 → 迁移+seed →
//     演练 → DROP，禁写 wbflow 库，天然可重复执行）；
//   - httptest.NewServer 启动完整 Gin 栈（26 wb_* 端点全装配，T1-T16 能力面）；
//   - HTTP client 携 Bearer token 以真实 HTTP 协议驱动。
//
// env 门控与注入（密码零落地）：WBFLOW_E2E=1 + WBFLOW_MYSQL_HOST/PORT/USER/PASSWORD。
// 运行命令与剧本索引见 rd/e2e/README.md。
package e2e

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gotest.tools/v3/assert"

	"github.com/huzilin/wbflow/rd/config"
	"github.com/huzilin/wbflow/rd/database"
	"github.com/huzilin/wbflow/rd/handler"
	"github.com/huzilin/wbflow/rd/migration"
	"github.com/huzilin/wbflow/rd/repo"
	"github.com/huzilin/wbflow/rd/service"
)

// e2eToken 演练 API token（进程内装配值，对应 config.Auth.Token 装配位）。
const e2eToken = "e2e-rehearsal-token"

// harness 一条剧本的运行环境：独立 scratch 库 + 完整 Gin 栈 HTTP server。
type harness struct {
	t          *testing.T
	cfg        *config.MySQLConfig
	db         *database.DB
	server     *httptest.Server
	token      string
	httpClient *http.Client
}

// startHarness 建库 → 迁移+seed → 装配全栈路由 → 起 HTTP server；测后关服 +
// 关连接池 + DROP scratch 库（每轮独立 scratch 库即天然可重复）。
func startHarness(t *testing.T, label string) *harness {
	t.Helper()
	if os.Getenv("WBFLOW_E2E") == "" {
		t.Skip("端到端演练：设 WBFLOW_E2E=1 且注入 WBFLOW_MYSQL_PASSWORD 后运行（见 rd/e2e/README.md）")
	}
	cfg := &config.MySQLConfig{
		Host:               envOr("WBFLOW_MYSQL_HOST", "192.168.3.100"),
		Port:               3306,
		User:               envOr("WBFLOW_MYSQL_USER", "root"),
		Password:           os.Getenv("WBFLOW_MYSQL_PASSWORD"),
		MaxOpenConns:       10,
		MaxIdleConns:       5,
		ConnMaxLifetimeSec: 3600,
	}
	if p := os.Getenv("WBFLOW_MYSQL_PORT"); p != "" {
		port, err := strconv.ParseInt(p, 10, 64)
		assert.NilError(t, err)
		cfg.Port = port
	}
	cfg.DBName = fmt.Sprintf("wbflow_e2e_%s_%d", label, time.Now().UnixNano())
	t.Logf("scratch 库: %s", cfg.DBName)

	handles, err := migration.Bootstrap(context.Background(), cfg)
	assert.NilError(t, err)
	t.Cleanup(func() {
		_ = handles.Close()
		dropScratch(t, cfg)
	})
	report, err := migration.Run(context.Background(), handles.Gorm)
	assert.NilError(t, err)
	assert.Equal(t, report.ToVersion, migration.LatestVersion(), "迁移应至最新版本")

	r := repo.NewRepo(handles.Gorm)
	gin.SetMode(gin.TestMode)
	router := handler.NewRouter(&handler.Deps{
		Logger:      zap.NewNop(),
		Health:      service.NewHealth(handles),
		Token:       e2eToken,
		Reqs:        service.NewRequirementService(r),
		Vers:        service.NewVersionService(r),
		Flows:       service.NewFlowService(r),
		Artifacts:   service.NewArtifactService(r, t.TempDir()),
		Tasks:       service.NewTaskService(r),
		Comments:    service.NewIssueService(r, 5),
		Reviews:     service.NewReviewService(r),
		Escalations: service.NewEscalationService(r),
		Inbox:       service.NewInboxService(r),
	})
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	return &harness{
		t:          t,
		cfg:        cfg,
		db:         handles,
		server:     server,
		token:      e2eToken,
		httpClient: server.Client(),
	}
}

// dropScratch 测后清理 scratch 库（绝不触 wbflow 库）。
func dropScratch(t *testing.T, cfg *config.MySQLConfig) {
	t.Helper()
	handles, err := database.Open(cfg)
	if err != nil {
		t.Fatalf("清理句柄打开失败: %v", err)
	}
	defer func() { _ = handles.Close() }()
	if _, err := handles.Server.Exec("DROP DATABASE IF EXISTS `" + cfg.DBName + "`"); err != nil {
		t.Logf("scratch 库清理失败（%s）: %v", cfg.DBName, err)
	}
}

func envOr(key string, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// ---- HTTP 助手（真实 HTTP client，统一带 Bearer token） ----

// resp HTTP 出参（protojson 编码：字段 camelCase、int64 为 JSON 字符串）。
type resp struct {
	code int
	body map[string]any
}

// do 执行一次带 token 的 HTTP 请求（body 非 nil 时 JSON 编码携带）。
func (h *harness) do(method string, path string, body any) resp {
	h.t.Helper()
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		assert.NilError(h.t, err)
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, h.server.URL+path, reader)
	assert.NilError(h.t, err)
	req.Header.Set("Authorization", "Bearer "+h.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	httpResp, err := h.httpClient.Do(req)
	assert.NilError(h.t, err)
	defer func() { _ = httpResp.Body.Close() }()
	raw, err := io.ReadAll(httpResp.Body)
	assert.NilError(h.t, err)
	out := resp{code: httpResp.StatusCode, body: map[string]any{}}
	if len(raw) > 0 {
		assert.NilError(h.t, json.Unmarshal(raw, &out.body), "出参非合法 JSON: %s", string(raw))
	}
	return out
}

// ok 断言 200 成功出参并返回自身（链式便捷）。
func (h *harness) ok(r resp, msg string) resp {
	h.t.Helper()
	assert.Equal(h.t, r.code, http.StatusOK, "%s: %v", msg, r.body)
	assert.Equal(h.t, r.boolean("ok"), true, "%s: %v", msg, r.body)
	return r
}

// str / boolean / number / item / arr 出参取值助手（protojson：int64 为字符串）。
func (r resp) str(key string) string {
	if v, ok := r.body[key].(string); ok {
		return v
	}
	return ""
}

func (r resp) boolean(key string) bool {
	if v, ok := r.body[key].(bool); ok {
		return v
	}
	return false
}

func (r resp) number(t *testing.T, key string) int64 {
	t.Helper()
	switch v := r.body[key].(type) {
	case string:
		n, err := strconv.ParseInt(v, 10, 64)
		assert.NilError(t, err, "字段 %s 应为数字字符串，实测 %v", key, r.body[key])
		return n
	case float64:
		return int64(v)
	default:
		t.Fatalf("字段 %s 缺失或类型非法: %v", key, r.body[key])
		return 0
	}
}

func (r resp) item(t *testing.T, key string) resp {
	t.Helper()
	m, ok := r.body[key].(map[string]any)
	assert.Assert(t, ok, "字段 %s 应为对象，实测 %v", key, r.body[key])
	return resp{code: r.code, body: m}
}

func (r resp) arr(t *testing.T, key string) []resp {
	t.Helper()
	list, ok := r.body[key].([]any)
	assert.Assert(t, ok, "字段 %s 应为数组，实测 %v", key, r.body[key])
	out := make([]resp, 0, len(list))
	for _, e := range list {
		m, ok := e.(map[string]any)
		assert.Assert(t, ok)
		out = append(out, resp{body: m})
	}
	return out
}

// ---- scratch 库直读助手（编排接缝模拟 / 审计断言专用，绝不触 wbflow 库） ----

// execSQL scratch 库内执行编排接缝 SQL。
func (h *harness) execSQL(query string, args ...any) {
	h.t.Helper()
	assert.NilError(h.t, h.db.Gorm.WithContext(context.Background()).Exec(query, args...).Error)
}

// queryText scratch 库读单列文本（NULL=空串；仅内部字面量拼接，不收外部输入）。
func (h *harness) queryText(query string, args ...any) string {
	h.t.Helper()
	var out sql.NullString
	assert.NilError(h.t, h.db.Gorm.WithContext(context.Background()).
		Raw(query, args...).Scan(&out).Error)
	return out.String
}

// auditCount 审计行计数（table_name+row_id+action 口径）。
func (h *harness) auditCount(table string, rowID string, action string) int64 {
	h.t.Helper()
	count := int64(0)
	assert.NilError(h.t, h.db.Gorm.WithContext(context.Background()).
		Raw("SELECT COUNT(*) FROM audit_log WHERE table_name = ? AND row_id = ? AND action = ?",
			table, rowID, action).Scan(&count).Error)
	return count
}

// auditCountAll 审计行计数（table+action 全表口径——scratch 库单需求隔离，
// 总量对账数据源）。
func (h *harness) auditCountAll(table string, action string) int64 {
	h.t.Helper()
	count := int64(0)
	assert.NilError(h.t, h.db.Gorm.WithContext(context.Background()).
		Raw("SELECT COUNT(*) FROM audit_log WHERE table_name = ? AND action = ?",
			table, action).Scan(&count).Error)
	return count
}

// auditFieldCountBy 审计行计数（table+row+field+by_who 口径——编排迁移 plugin/
// 拍板 user 等行为断言数据源）。
func (h *harness) auditFieldCountBy(table string, rowID string, field string, byWho string) int64 {
	h.t.Helper()
	count := int64(0)
	assert.NilError(h.t, h.db.Gorm.WithContext(context.Background()).
		Raw("SELECT COUNT(*) FROM audit_log WHERE table_name = ? AND row_id = ? AND field = ? AND by_who = ?",
			table, rowID, field, byWho).Scan(&count).Error)
	return count
}
