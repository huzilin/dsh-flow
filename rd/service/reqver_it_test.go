// T3 涉库集成测试（AC）：httptest 直打 requirements/versions 10 端点——
// 正常 / 拦截（blocked/401/409）/ 幂等分支 + 审计落库断言 + req-NNN/v-NNN 取号并发安全。
//
// env 门控与 scratch 库口径沿用 T2：WBFLOW_MYSQL_IT=1 + WBFLOW_MYSQL_* 注入；
// 独立 scratch 库（wbflow_it_* 后缀）应用迁移后测、测后 DROP——
// 不污染 wbflow 库、不改其数据。密码仅经 env 零落地。
package service_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync"
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

const itToken = "it-test-token"

// openScratch 每测试独立 scratch 库（建库 → 迁移 → 装配真实 service 的路由），测后 DROP。
func openScratch(t *testing.T) (*gin.Engine, *database.DB) {
	t.Helper()
	if os.Getenv("WBFLOW_MYSQL_IT") == "" {
		t.Skip("真实 MySQL 集成：设 WBFLOW_MYSQL_IT=1 且注入 WBFLOW_MYSQL_PASSWORD 后运行")
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
	cfg.DBName = fmt.Sprintf("wbflow_it_%d", time.Now().UnixNano())
	t.Logf("scratch 库: %s", cfg.DBName)

	handles, err := migration.Bootstrap(context.Background(), cfg)
	assert.NilError(t, err)
	t.Cleanup(func() {
		_ = handles.Close()
		dropScratch(t, cfg)
	})
	report, err := migration.Run(context.Background(), handles.Gorm)
	assert.NilError(t, err)
	assert.Equal(t, report.ToVersion, migration.LatestVersion())

	r := repo.NewRepo(handles.Gorm)
	gin.SetMode(gin.TestMode)
	router := handler.NewRouter(&handler.Deps{
		Logger: zap.NewNop(),
		Health: service.NewHealth(handles),
		Token:  itToken,
		Reqs:   service.NewRequirementService(r),
		Vers:   service.NewVersionService(r),
		Flows:  service.NewFlowService(r),
		Tasks:  service.NewTaskService(r),
	})
	return router, handles
}

// ---- HTTP 助手（统一带 Bearer token） ----

type testResp struct {
	code   int
	header http.Header
	body   map[string]any
}

// reqJSON 执行带 token 的请求（body 为 nil 时不带请求体）。
func reqJSON(t *testing.T, r *gin.Engine, method string, path string, body any) testResp {
	t.Helper()
	return reqJSONHeader(t, r, method, path, body, map[string]string{
		"Authorization": "Bearer " + itToken,
	})
}

// reqJSONHeader 显式指定请求头（401 分支用）。
func reqJSONHeader(t *testing.T, r *gin.Engine, method string, path string, body any, header map[string]string) testResp {
	t.Helper()
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		assert.NilError(t, err)
		reader = strings.NewReader(string(data))
	}
	req := httptest.NewRequest(method, path, reader)
	for k, v := range header {
		req.Header.Set(k, v)
	}
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	out := testResp{code: resp.Code, header: resp.Header()}
	if resp.Body.Len() > 0 {
		assert.NilError(t, json.Unmarshal(resp.Body.Bytes(), &out.body))
	} else {
		out.body = map[string]any{}
	}
	return out
}

// ---- 断言助手（protojson 出参：int64 为 JSON 字符串） ----

func (tr testResp) str(key string) string {
	if v, ok := tr.body[key].(string); ok {
		return v
	}
	return ""
}

func (tr testResp) boolean(key string) bool {
	if v, ok := tr.body[key].(bool); ok {
		return v
	}
	return false
}

func (tr testResp) number(t *testing.T, key string) int64 {
	t.Helper()
	switch v := tr.body[key].(type) {
	case string:
		n, err := strconv.ParseInt(v, 10, 64)
		assert.NilError(t, err, "字段 %s 应为数字字符串，实测 %v", key, tr.body[key])
		return n
	case float64:
		return int64(v)
	default:
		t.Fatalf("字段 %s 缺失或类型非法: %v", key, tr.body[key])
		return 0
	}
}

// item / arr 取嵌套对象/数组。
func item(t *testing.T, tr testResp, key string) testResp {
	t.Helper()
	m, ok := tr.body[key].(map[string]any)
	assert.Assert(t, ok, "字段 %s 应为对象，实测 %v", key, tr.body[key])
	return testResp{code: tr.code, body: m}
}

func arr(t *testing.T, tr testResp, key string) []testResp {
	t.Helper()
	list, ok := tr.body[key].([]any)
	assert.Assert(t, ok, "字段 %s 应为数组，实测 %v", key, tr.body[key])
	out := make([]testResp, 0, len(list))
	for _, e := range list {
		m, ok := e.(map[string]any)
		assert.Assert(t, ok)
		out = append(out, testResp{body: m})
	}
	return out
}

// auditCount 审计行计数断言数据源。
func auditCount(t *testing.T, handles *database.DB, table string, rowID string, action string) int64 {
	t.Helper()
	count := int64(0)
	err := handles.Gorm.WithContext(context.Background()).
		Raw("SELECT COUNT(*) FROM audit_log WHERE table_name = ? AND row_id = ? AND action = ?",
			table, rowID, action).Scan(&count).Error
	assert.NilError(t, err)
	return count
}

// ---- requirements 五端点 ----

// TestIntegration_RequirementsEndpoints_RealMySQL 创建/列表筛选分页/聚合快照/
// 乐观锁更新+diff 审计/删除确认口径全分支。
func TestIntegration_RequirementsEndpoints_RealMySQL(t *testing.T) {
	router, handles := openScratch(t)
	createBody := func(title, category string) map[string]any {
		return map[string]any{"title": title, "category": category, "createdBy": "user-a"}
	}

	// 1. 创建：需求任务默认派生 business；req-001 起号；默认待分配池 + 待启动 + P2
	resp := reqJSON(t, router, http.MethodPost, "/api/requirements", createBody("登录功能", "需求任务"))
	assert.Equal(t, resp.code, http.StatusOK)
	assert.Equal(t, resp.boolean("ok"), true)
	req1 := item(t, resp, "requirement")
	assert.Equal(t, req1.str("requirementId"), "req-001")
	assert.Equal(t, req1.str("flowDefsId"), "business")
	assert.Equal(t, req1.str("status"), "待启动")
	assert.Equal(t, req1.str("versionId"), "")
	assert.Equal(t, req1.str("priority"), "P2")
	assert.Assert(t, req1.number(t, "updatedAt") > 0, "updated_at 应为应用层毫秒值")
	assert.Equal(t, auditCount(t, handles, "requirements", "req-001", "INSERT"), int64(1), "创建写 INSERT 审计")

	// 2. 创建：允许集内显式指定 tech
	resp = reqJSON(t, router, http.MethodPost, "/api/requirements", map[string]any{
		"title": "缓存优化", "category": "需求任务", "flowDefsId": "tech",
		"priority": "P0", "createdBy": "user-a",
	})
	assert.Equal(t, resp.code, http.StatusOK)
	assert.Equal(t, item(t, resp, "requirement").str("requirementId"), "req-002")
	assert.Equal(t, item(t, resp, "requirement").str("flowDefsId"), "tech")

	// 3. 创建：规划审批任务/版本规划任务 派生 plan
	for i, tc := range []struct {
		title    string
		category string
		wantID   string
	}{
		{"制度评审", "规划审批任务", "req-003"},
		{"下版本规划", "版本规划任务", "req-004"},
	} {
		_ = i
		resp = reqJSON(t, router, http.MethodPost, "/api/requirements", createBody(tc.title, tc.category))
		assert.Equal(t, resp.code, http.StatusOK)
		got := item(t, resp, "requirement")
		assert.Equal(t, got.str("requirementId"), tc.wantID)
		assert.Equal(t, got.str("flowDefsId"), "plan")
	}

	// 4. 创建拦截：越集（审批类别绑 business）/ 未知类别 / 缺 title → 400
	resp = reqJSON(t, router, http.MethodPost, "/api/requirements", map[string]any{
		"title": "越集", "category": "规划审批任务", "flowDefsId": "business", "createdBy": "user-a",
	})
	assert.Equal(t, resp.code, http.StatusBadRequest)
	assert.Equal(t, resp.str("code"), "INVALID_ARGUMENT")

	resp = reqJSON(t, router, http.MethodPost, "/api/requirements", createBody("未知类别", "别的任务"))
	assert.Equal(t, resp.code, http.StatusBadRequest)

	resp = reqJSON(t, router, http.MethodPost, "/api/requirements", map[string]any{"category": "需求任务", "createdBy": "user-a"})
	assert.Equal(t, resp.code, http.StatusBadRequest)

	// 5. 列表：总数/默认分页/id DESC/筛选
	resp = reqJSON(t, router, http.MethodGet, "/api/requirements", nil)
	assert.Equal(t, resp.code, http.StatusOK)
	assert.Equal(t, resp.number(t, "total"), int64(4))
	assert.Equal(t, resp.number(t, "page"), int64(1))
	assert.Equal(t, resp.number(t, "pageSize"), int64(20))
	items := arr(t, resp, "items")
	assert.Equal(t, len(items), 4)
	assert.Equal(t, items[0].str("requirementId"), "req-004", "id DESC 新需求在前")

	resp = reqJSON(t, router, http.MethodGet, "/api/requirements?page=2&page_size=2", nil)
	assert.Equal(t, resp.number(t, "total"), int64(4))
	items = arr(t, resp, "items")
	assert.Equal(t, len(items), 2)
	assert.Equal(t, items[0].str("requirementId"), "req-002")
	assert.Equal(t, items[1].str("requirementId"), "req-001")

	resp = reqJSON(t, router, http.MethodGet, "/api/requirements?priority=P0", nil)
	assert.Equal(t, resp.number(t, "total"), int64(1))
	assert.Equal(t, arr(t, resp, "items")[0].str("requirementId"), "req-002")

	resp = reqJSON(t, router, http.MethodGet, "/api/requirements?status=待启动", nil)
	assert.Equal(t, resp.number(t, "total"), int64(4))

	// 待分配池口径：version_id IS NULL
	resp = reqJSON(t, router, http.MethodGet, "/api/requirements?backlog=true", nil)
	assert.Equal(t, resp.number(t, "total"), int64(4))

	// 6. 聚合快照：T4 起创建即实例化——business 默认 scope=full，13 实例节点
	resp = reqJSON(t, router, http.MethodGet, "/api/requirements/req-001", nil)
	assert.Equal(t, resp.code, http.StatusOK)
	assert.Equal(t, item(t, resp, "requirement").str("requirementId"), "req-001")
	assert.Assert(t, resp.body["version"] == nil, "待分配池需求 version 应为 null")
	flows := arr(t, resp, "flows")
	assert.Equal(t, len(flows), 1, "T4 创建即实例化 flows 1 行")
	assert.Equal(t, flows[0].str("flowDefsId"), "business")
	assert.Equal(t, flows[0].str("scope"), "full", "business 缺省 scope=full")
	assert.Equal(t, flows[0].str("currentNodeId"), "", "未进入编排 currentNodeId 空")
	assert.Equal(t, flows[0].str("dependencies"), "[]")
	flowNodes := arr(t, resp, "flowNodes")
	assert.Equal(t, len(flowNodes), 13, "business full 13 实例节点")
	assert.Equal(t, flowNodes[0].str("defNodeId"), "bus-00")
	assert.Equal(t, flowNodes[0].str("state"), "", "未进入编排 state 空")
	assert.Equal(t, flowNodes[0].number(t, "nodeOrder"), int64(1))
	assert.Equal(t, flowNodes[0].str("nodeType"), "gate", "定义属性经 def_node_id 关联填充")
	assert.Equal(t, flowNodes[1].str("nodeType"), "align")
	assert.Equal(t, flowNodes[3].str("nodeId"), "n-04", "node_id n-NN 与渲染序一致")
	assert.Equal(t, flowNodes[12].str("defNodeId"), "bus-12")
	assert.Equal(t, flowNodes[2].str("reviewResults"), "[]")
	assert.Assert(t, len(arr(t, resp, "artifacts")) == 0, "T6 前 artifacts 空集")
	assert.Assert(t, len(arr(t, resp, "issues")) == 0, "T7 前 issues 空集")
	assert.Assert(t, strings.Contains(resp.str("reviewResults"), "["), "reviewResults 汇总为 JSON 数组文本")

	// 7. 快照 404
	resp = reqJSON(t, router, http.MethodGet, "/api/requirements/req-999", nil)
	assert.Equal(t, resp.code, http.StatusNotFound)
	assert.Equal(t, resp.str("code"), "NOT_FOUND")

	// 8. 乐观锁更新 + 自动 diff 审计（逐变更字段）
	baseline := req1.number(t, "updatedAt")
	resp = reqJSON(t, router, http.MethodPut, "/api/requirements/req-001", map[string]any{
		"title": "登录功能 v2", "description": "", "category": "需求任务",
		"priority": "P2", "status": "待启动", "owner": "pm",
		"versionId": "", "updatedAt": baseline,
	})
	assert.Equal(t, resp.code, http.StatusOK)
	updated := item(t, resp, "requirement")
	assert.Equal(t, updated.str("title"), "登录功能 v2")
	assert.Equal(t, updated.str("owner"), "pm")
	assert.Assert(t, updated.number(t, "updatedAt") > baseline, "updated_at 应回前移")
	assert.Equal(t, auditCount(t, handles, "requirements", "req-001", "UPDATE"), int64(2),
		"变更 title/owner 两字段各写一条 UPDATE 审计")

	// 9. 乐观锁 409：旧基线重放
	resp = reqJSON(t, router, http.MethodPut, "/api/requirements/req-001", map[string]any{
		"title": "旧基线写", "description": "", "category": "需求任务",
		"priority": "P2", "status": "待启动", "owner": "pm",
		"versionId": "", "updatedAt": baseline,
	})
	assert.Equal(t, resp.code, http.StatusConflict)
	assert.Equal(t, resp.str("code"), "OPTIMISTIC_LOCK")

	// 10. 幂等 no-op：全字段同值更新成功且不写审计
	noChangeBaseline := updated.number(t, "updatedAt")
	resp = reqJSON(t, router, http.MethodPut, "/api/requirements/req-001", map[string]any{
		"title": "登录功能 v2", "description": "", "category": "需求任务",
		"priority": "P2", "status": "待启动", "owner": "pm",
		"versionId": "", "updatedAt": noChangeBaseline,
	})
	assert.Equal(t, resp.code, http.StatusOK)
	assert.Equal(t, auditCount(t, handles, "requirements", "req-001", "UPDATE"), int64(2), "no-op 不写审计")

	// 11. 更新拦截：枚举外 status → 400
	resp = reqJSON(t, router, http.MethodPut, "/api/requirements/req-001", map[string]any{
		"title": "x", "description": "", "category": "需求任务",
		"priority": "P2", "status": "不存在的状态", "owner": "",
		"versionId": "", "updatedAt": noChangeBaseline,
	})
	assert.Equal(t, resp.code, http.StatusBadRequest)

	// 12. 删除口径：未 confirm 默认 blocked
	resp = reqJSON(t, router, http.MethodDelete, "/api/requirements/req-004", nil)
	assert.Equal(t, resp.code, http.StatusConflict)
	assert.Equal(t, resp.boolean("blocked"), true, "未 confirm 删除应 blocked")
	assert.Equal(t, auditCount(t, handles, "requirements", "req-004", "DELETE"), int64(0))

	// 13. 删除：显式 confirm 生效 + DELETE 审计 + 复删 404（幂等=资源已不存在）
	resp = reqJSON(t, router, http.MethodDelete, "/api/requirements/req-004?confirm=true", nil)
	assert.Equal(t, resp.code, http.StatusOK)
	assert.Equal(t, resp.str("requirementId"), "req-004")
	assert.Equal(t, auditCount(t, handles, "requirements", "req-004", "DELETE"), int64(1))
	resp = reqJSON(t, router, http.MethodDelete, "/api/requirements/req-004?confirm=true", nil)
	assert.Equal(t, resp.code, http.StatusNotFound)

	// 14. 删除拦截：进行中需求 confirm=true 仍拒绝
	resp = reqJSON(t, router, http.MethodPut, "/api/requirements/req-001", map[string]any{
		"title": "登录功能 v2", "description": "", "category": "需求任务",
		"priority": "P2", "status": "进行中", "owner": "pm",
		"versionId": "", "updatedAt": noChangeBaseline,
	})
	assert.Equal(t, resp.code, http.StatusOK)
	resp = reqJSON(t, router, http.MethodDelete, "/api/requirements/req-001?confirm=true", nil)
	assert.Equal(t, resp.code, http.StatusConflict)
	assert.Equal(t, resp.boolean("blocked"), true, "进行中需求删除应 blocked")
}

// ---- versions 五端点 ----

// TestIntegration_VersionsEndpoints_RealMySQL 版本创建/改名/发布软约束/
// 废弃引用拦截全分支 + 状态机终态语义。
func TestIntegration_VersionsEndpoints_RealMySQL(t *testing.T) {
	router, handles := openScratch(t)

	// 1. 创建：v-NNN 起号 + 待分配池标记行
	resp := reqJSON(t, router, http.MethodPost, "/api/versions", map[string]any{"name": "v1.0"})
	assert.Equal(t, resp.code, http.StatusOK)
	v1 := item(t, resp, "version")
	assert.Equal(t, v1.str("versionId"), "v-1")
	assert.Equal(t, v1.str("state"), "待启动")
	assert.Equal(t, v1.boolean("isBacklog"), false)

	resp = reqJSON(t, router, http.MethodPost, "/api/versions", map[string]any{"name": "待分配池", "isBacklog": true})
	assert.Equal(t, resp.code, http.StatusOK)
	v2 := item(t, resp, "version")
	assert.Equal(t, v2.str("versionId"), "v-2")
	assert.Equal(t, v2.boolean("isBacklog"), true)

	resp = reqJSON(t, router, http.MethodPost, "/api/versions", map[string]any{"name": "v2.0"})
	assert.Equal(t, item(t, resp, "version").str("versionId"), "v-3")

	// 创建拦截：缺 name → 400
	resp = reqJSON(t, router, http.MethodPost, "/api/versions", map[string]any{})
	assert.Equal(t, resp.code, http.StatusBadRequest)

	// 2. 列表含待分配池
	resp = reqJSON(t, router, http.MethodGet, "/api/versions", nil)
	assert.Equal(t, resp.code, http.StatusOK)
	assert.Equal(t, len(arr(t, resp, "items")), 3)

	// 3. 改名（乐观锁）+ 409
	baseline := v1.number(t, "updatedAt")
	resp = reqJSON(t, router, http.MethodPut, "/api/versions/v-1", map[string]any{"name": "v1.0（定稿）", "updatedAt": baseline})
	assert.Equal(t, resp.code, http.StatusOK)
	assert.Equal(t, item(t, resp, "version").str("name"), "v1.0（定稿）")
	newBaseline := item(t, resp, "version").number(t, "updatedAt")

	resp = reqJSON(t, router, http.MethodPut, "/api/versions/v-1", map[string]any{"name": "旧基线", "updatedAt": baseline})
	assert.Equal(t, resp.code, http.StatusConflict)
	assert.Equal(t, resp.str("code"), "OPTIMISTIC_LOCK")

	// 改名幂等 no-op（同值不写审计）
	resp = reqJSON(t, router, http.MethodPut, "/api/versions/v-1", map[string]any{"name": "v1.0（定稿）", "updatedAt": newBaseline})
	assert.Equal(t, resp.code, http.StatusOK)
	assert.Equal(t, auditCount(t, handles, "versions", "v-1", "UPDATE"), int64(1), "仅首次改名写审计")

	// 4. 发布：空版本直接发布 ok
	resp = reqJSON(t, router, http.MethodPost, "/api/versions/v-1/advance", nil)
	assert.Equal(t, resp.code, http.StatusOK)
	assert.Equal(t, resp.boolean("ok"), true)
	assert.Equal(t, item(t, resp, "version").str("state"), "已发布")
	assert.Equal(t, len(arr(t, resp, "unfinished")), 0)
	assert.Equal(t, auditCount(t, handles, "versions", "v-1", "UPDATE"), int64(2), "state 迁移写审计")

	// 5. 重复发布幂等无害（no-op 不写审计）
	resp = reqJSON(t, router, http.MethodPost, "/api/versions/v-1/advance", nil)
	assert.Equal(t, resp.code, http.StatusOK)
	assert.Equal(t, auditCount(t, handles, "versions", "v-1", "UPDATE"), int64(2))

	// 6. 发布软约束：v-3 挂 2 条需求（1 待启动 1 已完成）→ ok + unfinished 清单提示
	for _, title := range []string{"软约束A", "软约束B"} {
		resp = reqJSON(t, router, http.MethodPost, "/api/requirements", map[string]any{
			"title": title, "category": "需求任务", "createdBy": "user-a", "versionId": "v-3",
		})
		assert.Equal(t, resp.code, http.StatusOK)
	}

	resp = reqJSON(t, router, http.MethodPost, "/api/versions/v-3/advance", nil)
	assert.Equal(t, resp.code, http.StatusOK)
	assert.Equal(t, resp.boolean("ok"), true, "软约束：unfinished 仅提示不拦截")
	unfinished := arr(t, resp, "unfinished")
	assert.Equal(t, len(unfinished), 2)
	assert.Equal(t, unfinished[0].str("requirementId"), "req-001")
	assert.Equal(t, item(t, resp, "version").str("state"), "已发布")

	// 7. 废弃拦截：有未完成需求引用 → blocked
	resp = reqJSON(t, router, http.MethodPost, "/api/versions/v-3/deprecate", nil)
	assert.Equal(t, resp.code, http.StatusConflict)
	assert.Equal(t, resp.boolean("blocked"), true)
	assert.Equal(t, auditCount(t, handles, "versions", "v-3", "UPDATE"), int64(1), "拦截不写 state 审计")

	// 8. 需求全部取消后废弃放行（引用口径=非 已取消/已完成）
	resp = reqJSON(t, router, http.MethodPut, "/api/requirements/req-001", map[string]any{
		"title": "软约束A", "description": "", "category": "需求任务",
		"priority": "P2", "status": "已取消", "owner": "",
		"versionId": "v-3",
	})
	assert.Equal(t, resp.code, http.StatusOK, "取消需求：%v", resp.body)
	resp = reqJSON(t, router, http.MethodPut, "/api/requirements/req-002", map[string]any{
		"title": "软约束B", "description": "", "category": "需求任务",
		"priority": "P2", "status": "已完成", "owner": "",
		"versionId": "v-3",
	})
	assert.Equal(t, resp.code, http.StatusOK)

	resp = reqJSON(t, router, http.MethodPost, "/api/versions/v-3/deprecate", nil)
	assert.Equal(t, resp.code, http.StatusOK)
	assert.Equal(t, item(t, resp, "version").str("state"), "已废弃")

	// 9. 重复废弃幂等无害
	resp = reqJSON(t, router, http.MethodPost, "/api/versions/v-3/deprecate", nil)
	assert.Equal(t, resp.code, http.StatusOK)

	// 10. 已废弃=终态：advance blocked
	resp = reqJSON(t, router, http.MethodPost, "/api/versions/v-3/advance", nil)
	assert.Equal(t, resp.code, http.StatusConflict)
	assert.Equal(t, resp.boolean("blocked"), true)

	// 11. 任意阶段废弃=终态：已发布版本（无引用）直接废弃放行
	resp = reqJSON(t, router, http.MethodPost, "/api/versions/v-1/deprecate", nil)
	assert.Equal(t, resp.code, http.StatusOK)
	assert.Equal(t, item(t, resp, "version").str("state"), "已废弃")

	// 12. 404 分支
	resp = reqJSON(t, router, http.MethodPost, "/api/versions/v-999/advance", nil)
	assert.Equal(t, resp.code, http.StatusNotFound)
	resp = reqJSON(t, router, http.MethodPost, "/api/versions/v-999/deprecate", nil)
	assert.Equal(t, resp.code, http.StatusNotFound)
	resp = reqJSON(t, router, http.MethodPut, "/api/versions/v-999", map[string]any{"name": "x"})
	assert.Equal(t, resp.code, http.StatusNotFound)
}

// ---- 取号并发安全 ----

// TestIntegration_IDAlloc_Concurrency_RealMySQL req-NNN / v-NNN 并发取号：
// 全部成功且号码唯一无重复（事务内取号 + 唯一键/锁竞争整体重试）。
func TestIntegration_IDAlloc_Concurrency_RealMySQL(t *testing.T) {
	router, _ := openScratch(t)

	const reqWorkers = 8
	reqIDs := make(chan string, reqWorkers)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < reqWorkers; i++ {
		wg.Add(1)
		go func(seq int) {
			defer wg.Done()
			<-start
			resp := reqJSON(t, router, http.MethodPost, "/api/requirements", map[string]any{
				"title": fmt.Sprintf("并发需求-%d", seq), "category": "需求任务", "createdBy": "user-a",
			})
			assert.Equal(t, resp.code, http.StatusOK)
			reqIDs <- item(t, resp, "requirement").str("requirementId")
		}(i)
	}
	close(start)
	wg.Wait()
	close(reqIDs)

	seen := make(map[string]bool)
	for id := range reqIDs {
		assert.Assert(t, !seen[id], "取号重复: %s", id)
		seen[id] = true
	}
	assert.Equal(t, len(seen), reqWorkers, "应取到全部唯一号码")
	for i := 1; i <= reqWorkers; i++ {
		want := fmt.Sprintf("req-%03d", i)
		assert.Assert(t, seen[want], "应有 %s（实际集: %v）", want, seen)
	}

	const verWorkers = 4
	verIDs := make(chan string, verWorkers)
	start2 := make(chan struct{})
	var wg2 sync.WaitGroup
	for i := 0; i < verWorkers; i++ {
		wg2.Add(1)
		go func(seq int) {
			defer wg2.Done()
			<-start2
			resp := reqJSON(t, router, http.MethodPost, "/api/versions", map[string]any{
				"name": fmt.Sprintf("并发版本-%d", seq),
			})
			assert.Equal(t, resp.code, http.StatusOK)
			verIDs <- item(t, resp, "version").str("versionId")
		}(i)
	}
	close(start2)
	wg2.Wait()
	close(verIDs)

	seenVer := make(map[string]bool)
	for id := range verIDs {
		assert.Assert(t, !seenVer[id], "版本取号重复: %s", id)
		seenVer[id] = true
	}
	assert.Equal(t, len(seenVer), verWorkers)
	for i := 1; i <= verWorkers; i++ {
		want := fmt.Sprintf("v-%d", i)
		assert.Assert(t, seenVer[want], "应有 %s（实际集: %v）", want, seenVer)
	}
}

// ---- openScratch 支撑（env 注入 / scratch 清理） ----

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

func envOrShared(key string, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
