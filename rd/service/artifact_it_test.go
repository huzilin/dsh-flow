// T6 涉库集成测试（AC）：wb_add_artifact 受控枚举/归属/路径穿越/幂等/并发取号 +
// wb_get_artifact_content document/link/缺失三分支 + my_task relatedArtifacts
// 三分支（有产物/无产物/节点无关）。
//
// env 门控与 scratch 库口径沿用 T2/T3：WBFLOW_MYSQL_IT=1 + WBFLOW_MYSQL_* 注入，
// 独立 scratch 库（wbflow_it_*）测后 DROP——禁写 wbflow 库。
// 测试用 docs root 为临时目录（t.TempDir，测后自动清理）。
// 密码仅经 env 零落地。装配 helper 自包含（openScratchWithDocs），不动既有
// openScratch 签名，便于并行工单三方合并。
package service_test

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
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

// openScratchWithDocs 每测试独立 scratch 库 + 临时 docs root 的全量路由装配，
// 返回 docsRoot 供用例写文档文件。测后 DROP scratch 库。
func openScratchWithDocs(t *testing.T) (*gin.Engine, *database.DB, string) {
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

	docsRoot := t.TempDir()
	r := repo.NewRepo(handles.Gorm)
	gin.SetMode(gin.TestMode)
	router := handler.NewRouter(&handler.Deps{
		Logger:    zap.NewNop(),
		Health:    service.NewHealth(handles),
		Token:     itToken,
		Reqs:      service.NewRequirementService(r),
		Vers:      service.NewVersionService(r),
		Flows:     service.NewFlowService(r),
		Artifacts: service.NewArtifactService(r, docsRoot),
	})
	return router, handles, docsRoot
}

// createReqForArtifacts 建需求（business × full，13 实例节点）返回需求 id。
func createReqForArtifacts(t *testing.T, router *gin.Engine, title string) string {
	t.Helper()
	resp := reqJSON(t, router, http.MethodPost, "/api/requirements", map[string]any{
		"title": title, "category": "需求任务", "flowDefsId": "business", "scope": "full",
		"createdBy": "user-a",
	})
	assert.Equal(t, resp.code, http.StatusOK, "创建失败: %v", resp.body)
	return item(t, resp, "requirement").str("requirementId")
}

// addArtifact 登记产物快捷函数。
func addArtifact(t *testing.T, router *gin.Engine, reqID string, nodeID string, body map[string]any) testResp {
	t.Helper()
	full := map[string]any{"addedBy": "pm"}
	for k, v := range body {
		full[k] = v
	}
	return reqJSON(t, router, http.MethodPost,
		"/api/requirements/"+reqID+"/nodes/"+nodeID+"/artifacts", full)
}

// artifactCount 按幂等键计数（DB 侧行数断言数据源）。
func artifactCount(t *testing.T, handles *database.DB, reqID string, nodeID string, productType string, href string) int64 {
	t.Helper()
	count := int64(0)
	err := handles.Gorm.WithContext(context.Background()).
		Raw("SELECT COUNT(*) FROM artifacts WHERE requirement_id = ? AND node_id = ? AND product_type = ? AND href = ?",
			reqID, nodeID, productType, href).Scan(&count).Error
	assert.NilError(t, err)
	return count
}

// TestIntegration_Artifacts_RegisterAndRead 登记正常路径 + 审计 + 阅读三分支
// （document 有正文 / link 外链 / document 文件缺失 404）。
func TestIntegration_Artifacts_RegisterAndRead(t *testing.T) {
	router, handles, docsRoot := openScratchWithDocs(t)
	reqID := createReqForArtifacts(t, router, "产物登记全链路")

	// document：登记（href 相对 docs root，文件已落盘）
	docPath := filepath.Join(docsRoot, "brd-v1.md")
	assert.NilError(t, os.WriteFile(docPath, []byte("# BRD\n目标与范围\n"), 0o644))
	resp := addArtifact(t, router, reqID, "n-01", map[string]any{
		"productType": "brd", "label": "《BRD v1》", "href": "brd-v1.md",
		"kind": "document", "note": "首版", "addedBy": "pm",
	})
	assert.Equal(t, resp.code, http.StatusOK, "登记失败: %v", resp.body)
	assert.Equal(t, resp.boolean("ok"), true)
	art := item(t, resp, "artifact")
	artID := art.str("artifactId")
	assert.Assert(t, len(artID) > len("art-"), "artifact_id 应为 art-<ts>: %q", artID)
	assert.Equal(t, art.str("kind"), "document")
	assert.Equal(t, art.str("addedBy"), "pm")
	assert.Assert(t, art.number(t, "createdAt") > 0, "登记时间应落库")
	// 审计：INSERT by_who=added_by
	assert.Equal(t, auditCount(t, handles, "artifacts", artID, "INSERT"), int64(1), "登记写 INSERT 审计")

	// document：读取正文 + 锚点定位元数据齐备
	resp = reqJSON(t, router, http.MethodGet, "/api/requirements/"+reqID+"/artifacts/"+artID+"/content", nil)
	assert.Equal(t, resp.code, http.StatusOK)
	assert.Equal(t, resp.str("artifactId"), artID)
	assert.Equal(t, resp.str("nodeId"), "n-01")
	assert.Equal(t, resp.str("productType"), "brd")
	assert.Equal(t, resp.str("label"), "《BRD v1》")
	assert.Equal(t, resp.str("kind"), "document")
	assert.Equal(t, resp.str("href"), "brd-v1.md")
	assert.Equal(t, resp.str("content"), "# BRD\n目标与范围\n")
	assert.Equal(t, resp.str("mimeType"), "text/markdown")

	// link：外链登记 + 读取（content 恒空，href 直出）
	resp = addArtifact(t, router, reqID, "n-01", map[string]any{
		"productType": "discussion", "label": "会议纪要", "href": "https://example.com/notes/1",
		"kind": "link", "addedBy": "pm",
	})
	assert.Equal(t, resp.code, http.StatusOK, "link 登记失败: %v", resp.body)
	linkID := item(t, resp, "artifact").str("artifactId")
	resp = reqJSON(t, router, http.MethodGet, "/api/requirements/"+reqID+"/artifacts/"+linkID+"/content", nil)
	assert.Equal(t, resp.code, http.StatusOK)
	assert.Equal(t, resp.str("kind"), "link")
	assert.Equal(t, resp.str("href"), "https://example.com/notes/1")
	assert.Equal(t, resp.str("content"), "", "link 正文恒空")

	// 缺失分支：登记后文件被删 → 404 + blocked:false
	assert.NilError(t, os.Remove(docPath))
	resp = reqJSON(t, router, http.MethodGet, "/api/requirements/"+reqID+"/artifacts/"+artID+"/content", nil)
	assert.Equal(t, resp.code, http.StatusNotFound)
	assert.Equal(t, resp.boolean("blocked"), false, "缺失分支非门禁拦截")
}

// TestIntegration_Artifacts_RegisterRejects 受控枚举越集（blocked + reason）、
// kind 越集、link href 非外链、node 归属不存在 404、需求不存在 404。
func TestIntegration_Artifacts_RegisterRejects(t *testing.T) {
	router, handles, docsRoot := openScratchWithDocs(t)
	reqID := createReqForArtifacts(t, router, "登记拦截分支")
	assert.NilError(t, os.WriteFile(filepath.Join(docsRoot, "a.md"), []byte("x"), 0o644))

	// product_type 越集（case_design 已移除）：400 + blocked:true + reason
	resp := addArtifact(t, router, reqID, "n-01", map[string]any{
		"productType": "case_design", "label": "x", "href": "a.md", "addedBy": "pm",
	})
	assert.Equal(t, resp.code, http.StatusBadRequest)
	assert.Equal(t, resp.boolean("blocked"), true, "越集拦截出参 blocked:true")
	assert.Assert(t, resp.str("error") != "", "拦截应带 reason")
	// 未受控词表值同样拦截
	resp = addArtifact(t, router, reqID, "n-01", map[string]any{
		"productType": "BRD", "label": "x", "href": "a.md", "addedBy": "pm",
	})
	assert.Equal(t, resp.code, http.StatusBadRequest)
	assert.Equal(t, artifactCount(t, handles, reqID, "n-01", "case_design", "a.md"), int64(0), "越集不得落库")

	// kind 越集
	resp = addArtifact(t, router, reqID, "n-01", map[string]any{
		"productType": "brd", "label": "x", "href": "a.md", "kind": "url", "addedBy": "pm",
	})
	assert.Equal(t, resp.code, http.StatusBadRequest)
	assert.Equal(t, resp.boolean("blocked"), true)

	// link href 非外链（相对路径冒充）
	resp = addArtifact(t, router, reqID, "n-01", map[string]any{
		"productType": "discussion", "label": "x", "href": "a.md", "kind": "link", "addedBy": "pm",
	})
	assert.Equal(t, resp.code, http.StatusBadRequest)

	// document href 文件不存在（存在性校验）
	resp = addArtifact(t, router, reqID, "n-01", map[string]any{
		"productType": "brd", "label": "x", "href": "nope.md", "addedBy": "pm",
	})
	assert.Equal(t, resp.code, http.StatusBadRequest)

	// node 归属不存在 404；需求不存在 404
	resp = addArtifact(t, router, reqID, "n-99", map[string]any{
		"productType": "brd", "label": "x", "href": "a.md", "addedBy": "pm",
	})
	assert.Equal(t, resp.code, http.StatusNotFound)
	resp = addArtifact(t, router, "req-ghost", "n-01", map[string]any{
		"productType": "brd", "label": "x", "href": "a.md", "addedBy": "pm",
	})
	assert.Equal(t, resp.code, http.StatusNotFound)
}

// TestIntegration_Artifacts_PathTraversalBlocked 路径穿越登记全拦截
// （../、绝对路径）；读取侧 symlink 逃逸防线（登记后文件被替换为根外 symlink）。
func TestIntegration_Artifacts_PathTraversalBlocked(t *testing.T) {
	router, handles, docsRoot := openScratchWithDocs(t)
	reqID := createReqForArtifacts(t, router, "穿越拦截")

	// ../ 上跳越界登记拦截
	resp := addArtifact(t, router, reqID, "n-01", map[string]any{
		"productType": "brd", "label": "x", "href": "../evil.md", "addedBy": "pm",
	})
	assert.Equal(t, resp.code, http.StatusBadRequest)
	assert.Equal(t, resp.boolean("blocked"), true, "穿越拦截出参 blocked:true")

	// 绝对路径登记拦截
	resp = addArtifact(t, router, reqID, "n-01", map[string]any{
		"productType": "brd", "label": "x", "href": "/etc/passwd", "addedBy": "pm",
	})
	assert.Equal(t, resp.code, http.StatusBadRequest)
	assert.Equal(t, artifactCount(t, handles, reqID, "n-01", "brd", "/etc/passwd"), int64(0), "越界不得落库")

	// 读取侧防线：合法登记后，文件被替换为指向根外的 symlink → 400 拦截
	docPath := filepath.Join(docsRoot, "victim.md")
	assert.NilError(t, os.WriteFile(docPath, []byte("v"), 0o644))
	resp = addArtifact(t, router, reqID, "n-02", map[string]any{
		"productType": "research", "label": "v", "href": "victim.md", "addedBy": "rd",
	})
	assert.Equal(t, resp.code, http.StatusOK)
	victimID := item(t, resp, "artifact").str("artifactId")

	outside := filepath.Join(t.TempDir(), "secret.md")
	assert.NilError(t, os.WriteFile(outside, []byte("s"), 0o644))
	assert.NilError(t, os.Remove(docPath))
	assert.NilError(t, os.Symlink(outside, docPath))

	resp = reqJSON(t, router, http.MethodGet, "/api/requirements/"+reqID+"/artifacts/"+victimID+"/content", nil)
	assert.Equal(t, resp.code, http.StatusBadRequest, "symlink 逃逸读取应拦截: %v", resp.body)
	assert.Equal(t, resp.boolean("blocked"), true)
}

// TestIntegration_Artifacts_IdempotentAndConcurrentAlloc 幂等重复登记 no-op
// （同 req×node×product_type×href 返回既有行，不新取号不重复审计）+
// 并发登记取号唯一（uk_artifacts_aid 兜底）。
func TestIntegration_Artifacts_IdempotentAndConcurrentAlloc(t *testing.T) {
	router, handles, docsRoot := openScratchWithDocs(t)
	reqID := createReqForArtifacts(t, router, "幂等与并发取号")
	assert.NilError(t, os.WriteFile(filepath.Join(docsRoot, "doc.md"), []byte("d"), 0o644))

	// 首次登记
	body := map[string]any{
		"productType": "rd_design", "label": "《RD 设计》", "href": "doc.md", "addedBy": "rd",
	}
	resp := addArtifact(t, router, reqID, "n-01", body)
	assert.Equal(t, resp.code, http.StatusOK)
	firstID := item(t, resp, "artifact").str("artifactId")
	assert.Equal(t, auditCount(t, handles, "artifacts", firstID, "INSERT"), int64(1))

	// 完全同键重复登记 → no-op 返回既有（同 artifact_id、单行、审计不重复）
	resp = addArtifact(t, router, reqID, "n-01", body)
	assert.Equal(t, resp.code, http.StatusOK)
	assert.Equal(t, item(t, resp, "artifact").str("artifactId"), firstID, "幂等命中应返回既有行")
	assert.Equal(t, auditCount(t, handles, "artifacts", firstID, "INSERT"), int64(1), "幂等命中不重复审计")
	assert.Equal(t, artifactCount(t, handles, reqID, "n-01", "rd_design", "doc.md"), int64(1))

	// 并发登记（同节点不同 href）：取号唯一
	const concurrency = 8
	results := make([]testResp, concurrency)
	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(seq int) {
			defer wg.Done()
			results[seq] = reqJSON(t, router, http.MethodPost,
				"/api/requirements/"+reqID+"/nodes/n-02/artifacts",
				map[string]any{
					"productType": "dev_notes", "label": fmt.Sprintf("note-%d", seq),
					"href": fmt.Sprintf("note-%d.md", seq), "addedBy": "rd",
				})
		}(i)
	}
	wg.Wait()

	ids := make(map[string]bool, concurrency)
	for _, r := range results {
		assert.Equal(t, r.code, http.StatusOK, "并发登记失败: %v", r.body)
		ids[item(t, r, "artifact").str("artifactId")] = true
	}
	assert.Equal(t, len(ids), concurrency, "并发取号应全局唯一")

	rows := int64(0)
	err := handles.Gorm.WithContext(context.Background()).
		Raw("SELECT COUNT(*) FROM artifacts WHERE requirement_id = ? AND node_id = ? AND product_type = ?",
			reqID, "n-02", "dev_notes").Scan(&rows).Error
	assert.NilError(t, err)
	assert.Equal(t, rows, int64(concurrency), "并发登记应全部落库")
}

// TestIntegration_MyTask_RelatedArtifacts my_task 产物侧注入三分支：
// 有产物（摘要齐备）/ 无产物（空集）/ 节点无关（404）。
func TestIntegration_MyTask_RelatedArtifacts(t *testing.T) {
	router, _, docsRoot := openScratchWithDocs(t)
	reqID := createReqForArtifacts(t, router, "my_task 产物注入")
	assert.NilError(t, os.WriteFile(filepath.Join(docsRoot, "m.md"), []byte("m"), 0o644))

	// n-01 登记两件产物（document + link）
	resp := addArtifact(t, router, reqID, "n-01", map[string]any{
		"productType": "brd", "label": "《BRD》", "href": "m.md", "addedBy": "pm",
	})
	assert.Equal(t, resp.code, http.StatusOK)
	resp = addArtifact(t, router, reqID, "n-01", map[string]any{
		"productType": "alignment", "label": "对齐结论", "href": "https://example.com/a",
		"kind": "link", "addedBy": "pm",
	})
	assert.Equal(t, resp.code, http.StatusOK)

	// 分支一：目标节点有产物 → related_artifacts 摘要齐备
	resp = reqJSON(t, router, http.MethodPost, "/api/requirements/"+reqID+"/my-task",
		map[string]any{"role": "pm", "nodeId": "n-01"})
	assert.Equal(t, resp.code, http.StatusOK)
	items := arr(t, resp, "relatedArtifacts")
	assert.Equal(t, len(items), 2)
	assert.Equal(t, items[0].str("productType"), "brd")
	assert.Equal(t, items[0].str("label"), "《BRD》")
	assert.Equal(t, items[0].str("kind"), "document")
	assert.Assert(t, items[0].str("artifactId") != "")
	assert.Equal(t, items[1].str("productType"), "alignment")
	assert.Equal(t, items[1].str("kind"), "link")

	// 分支二：目标节点无产物 → 空集（不报错）
	resp = reqJSON(t, router, http.MethodPost, "/api/requirements/"+reqID+"/my-task",
		map[string]any{"role": "rd", "nodeId": "n-02"})
	assert.Equal(t, resp.code, http.StatusOK)
	assert.Equal(t, len(arr(t, resp, "relatedArtifacts")), 0, "无产物应为空集")

	// 分支二补充：未进入编排（未指定 nodeId，current_node_id NULL）→ 空集
	resp = reqJSON(t, router, http.MethodPost, "/api/requirements/"+reqID+"/my-task",
		map[string]any{"role": "rd"})
	assert.Equal(t, resp.code, http.StatusOK)
	assert.Equal(t, len(arr(t, resp, "relatedArtifacts")), 0)

	// 分支三：节点无关（nodeId 不存在于该需求）→ 404
	resp = reqJSON(t, router, http.MethodPost, "/api/requirements/"+reqID+"/my-task",
		map[string]any{"role": "rd", "nodeId": "n-99"})
	assert.Equal(t, resp.code, http.StatusNotFound)
}
