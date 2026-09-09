// T10 涉库集成测试（AC）：扫描器 5min 催办 / 10min 介入按时触发（注入时钟模拟，
// domain 判定单测之外的真库一层）+ 催办幂等（同任务同轮重复扫描不重复建）+
// active 会话绑定（running 代理）等待 + 会话结束后续扫 + MySQL 断连降级退避不崩 +
// 恢复后从账面状态续扫（无内存态依赖）+ sessions 绑定 status 迁移全分支与非法迁移
// 拦截 + 写路径审计。env 门控与 scratch 库口径沿用 T2/T3：WBFLOW_MYSQL_IT=1 +
// WBFLOW_MYSQL_* 注入，独立 scratch 库（wbflow_it_*）测后 DROP——禁写 wbflow 库。
// 密码仅经 env 零落地。
package service_test

import (
	"context"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"gotest.tools/v3/assert"

	"github.com/gin-gonic/gin"
	"github.com/huzilin/wbflow/rd/config"
	"github.com/huzilin/wbflow/rd/database"
	"github.com/huzilin/wbflow/rd/pkg/errs"
	"github.com/huzilin/wbflow/rd/repo"
	"github.com/huzilin/wbflow/rd/service"
)

// scannerTestOpts IT 扫描参数（spec 标准阈值 5min/10min；间隔/退避仅 Run 循环消费，
// RunOnce 直驱不受影响）。
func scannerTestOpts() service.ScannerOptions {
	return service.ScannerOptions{
		Interval:       30 * time.Second,
		FollowupAfter:  5 * time.Minute,
		InterveneAfter: 10 * time.Minute,
		RetryBackoff:   5 * time.Second,
	}
}

// countIssues 统计 issues 表命中行数（通知行断言用）。
func countIssues(t *testing.T, handles *database.DB, where string, args ...any) int64 {
	t.Helper()
	var n int64
	assert.NilError(t, handles.Gorm.Raw("SELECT COUNT(*) FROM issues WHERE "+where, args...).Scan(&n).Error)
	return n
}

// countNotification 统计指定任务+kind 的通知行数。
func countNotification(t *testing.T, handles *database.DB, kind string, taskID string) int64 {
	t.Helper()
	return countIssues(t, handles, "issue_type = 'notification' AND text LIKE ?",
		"["+kind+"] task="+taskID+" %")
}

// countAudit 审计行计数（table_name + row_id 前缀 + action）。
func countAudit(t *testing.T, handles *database.DB, table string, rowIDPrefix string, action string) int64 {
	t.Helper()
	var n int64
	assert.NilError(t, handles.Gorm.Raw(
		"SELECT COUNT(*) FROM audit_log WHERE table_name = ? AND action = ? AND row_id LIKE ?",
		table, action, rowIDPrefix+"%").Scan(&n).Error)
	return n
}

// createReqAndTask 造需求 + 派发 produce 任务（扫描对象），返回 (reqID, taskID)。
func createReqAndTask(t *testing.T, router *gin.Engine, handles *database.DB) (string, string) {
	t.Helper()
	ctx := context.Background()
	resp := reqJSON(t, router, http.MethodPost, "/api/requirements", map[string]any{
		"title": "扫描器需求", "category": "需求任务", "createdBy": "user-a",
	})
	assert.Equal(t, resp.code, http.StatusOK, "创建失败: %v", resp.body)
	reqID := item(t, resp, "requirement").str("requirementId")
	resp = reqJSON(t, router, http.MethodPost, "/api/tasks", map[string]any{
		"requirementId": reqID, "nodeId": "n-03", "actionId": "product-design.produce", "assignee": "pm",
	})
	assert.Equal(t, resp.code, http.StatusOK, "派发失败: %v", resp.body)
	taskID := item(t, resp, "task").str("taskId")
	row, err := repo.NewRepo(handles.Gorm).GetTaskByTaskID(ctx, taskID)
	assert.NilError(t, err)
	assert.Assert(t, row != nil)
	return reqID, row.TaskID
}

// TestIntegration_Scanner_FollowupIntervene_RealMySQL 注入时钟全链路：0 点不触发 /
// active 会话等待 / 结束后 5min 催办 / 幂等不重复 / 10min 介入升级 / 介入幂等 /
// 通知与审计留痕。
func TestIntegration_Scanner_FollowupIntervene_RealMySQL(t *testing.T) {
	router, handles := openScratch(t)
	ctx := context.Background()
	reqID, taskID := createReqAndTask(t, router, handles)

	baseRepo := repo.NewRepo(handles.Gorm)
	task, err := baseRepo.GetTaskByTaskID(ctx, taskID)
	assert.NilError(t, err)

	// 注入时钟：起点=任务创建时刻（账面基线），测试内手动推进
	clock := task.CreatedAt
	scanner := service.NewScanner(baseRepo, handles, scannerTestOpts()).WithClock(func() int64 { return clock })

	// 1. 闲置 0：不触发
	report, err := scanner.RunOnce(ctx)
	assert.NilError(t, err)
	assert.Equal(t, report.Degraded, false)
	assert.Equal(t, len(report.Followups)+len(report.Intervenes), 0)

	// 2. active 会话绑定（running 代理判据）→ 闲置 11min 仍等待
	sessSvc := service.NewSessionService(baseRepo, nil)
	session, err := sessSvc.Bind(ctx, service.SessionBindInput{
		RequirementID: reqID, NodeID: "n-03", RoleID: "pm",
	})
	assert.NilError(t, err)
	clock = task.CreatedAt + 11*60*1000
	report, err = scanner.RunOnce(ctx)
	assert.NilError(t, err)
	assert.Equal(t, len(report.Followups)+len(report.Intervenes), 0, "active 会话应等待: %v", report)

	// 3. 结束会话 → 时钟 +5min 档：催办触发（assignee=执行角色 pm，state=open）
	_, noop, err := sessSvc.End(ctx, session.SessionID)
	assert.NilError(t, err)
	assert.Equal(t, noop, false)
	clock = task.CreatedAt + 5*60*1000
	report, err = scanner.RunOnce(ctx)
	assert.NilError(t, err)
	assert.DeepEqual(t, report.Followups, []string{taskID})
	assert.Equal(t, countNotification(t, handles, "followup", taskID), int64(1))
	var assignee string
	assert.NilError(t, handles.Gorm.Raw(
		"SELECT COALESCE(assignee,'') FROM issues WHERE issue_type='notification' AND text LIKE ? ORDER BY id LIMIT 1",
		"[followup] task="+taskID+" %").Scan(&assignee).Error)
	assert.Equal(t, assignee, "pm")

	// 4. 同任务同轮重复扫描：不重复建（催办幂等）
	clock = task.CreatedAt + 6*60*1000
	report, err = scanner.RunOnce(ctx)
	assert.NilError(t, err)
	assert.Equal(t, len(report.Followups), 0)
	assert.Equal(t, countNotification(t, handles, "followup", taskID), int64(1))

	// 5. 越介入阈值：升级介入（assignee=user），催办行不重复
	clock = task.CreatedAt + 10*60*1000
	report, err = scanner.RunOnce(ctx)
	assert.NilError(t, err)
	assert.DeepEqual(t, report.Intervenes, []string{taskID})
	assert.Equal(t, countNotification(t, handles, "followup", taskID), int64(1))
	assert.Equal(t, countNotification(t, handles, "intervene", taskID), int64(1))

	// 6. 介入幂等：再扫不再建
	clock = task.CreatedAt + 12*60*1000
	report, err = scanner.RunOnce(ctx)
	assert.NilError(t, err)
	assert.Equal(t, len(report.Followups)+len(report.Intervenes), 0)
	assert.Equal(t, countNotification(t, handles, "intervene", taskID), int64(1))

	// 7. 通知行写路径审计：两条 INSERT（row_id=通知 issue_id），by_who=system
	assert.Equal(t, countIssues(t, handles, "issue_type = 'notification'"), int64(2))
	var notifIDs []string
	assert.NilError(t, handles.Gorm.Raw(
		"SELECT issue_id FROM issues WHERE issue_type = 'notification' ORDER BY id").Scan(&notifIDs).Error)
	assert.Equal(t, len(notifIDs), 2)
	for _, issueID := range notifIDs {
		assert.Equal(t, countAudit(t, handles, "issues", issueID, "INSERT"), int64(1))
	}
}

// TestIntegration_Scanner_Recovery_RealMySQL 续跑 AC：断连（句柄关闭）降级不崩、
// 重建句柄（模拟进程重启）后从账面状态续扫——介入按时升级、催办不重复。
func TestIntegration_Scanner_Recovery_RealMySQL(t *testing.T) {
	router, handles := openScratch(t)
	ctx := context.Background()
	reqID, taskID := createReqAndTask(t, router, handles)
	_ = reqID

	baseRepo := repo.NewRepo(handles.Gorm)
	task, err := baseRepo.GetTaskByTaskID(ctx, taskID)
	assert.NilError(t, err)

	clock := task.CreatedAt
	scanner := service.NewScanner(baseRepo, handles, scannerTestOpts()).WithClock(func() int64 { return clock })

	// 账面 dbname 先取（Close 后 Server 句柄不可用）
	var dbName string
	assert.NilError(t, handles.Gorm.Raw("SELECT DATABASE()").Scan(&dbName).Error)

	// 1. 断连前：+5min 催办落账
	clock = task.CreatedAt + 5*60*1000
	report, err := scanner.RunOnce(ctx)
	assert.NilError(t, err)
	assert.DeepEqual(t, report.Followups, []string{taskID})

	// 2. 断连：句柄全关，ping 失败 → 降级返回（Degraded，进程不崩、无 panic）
	assert.NilError(t, handles.Close())
	report, err = scanner.RunOnce(ctx)
	assert.NilError(t, err)
	assert.Equal(t, report.Degraded, true)
	assert.Assert(t, report.Error != "")

	// 3. 恢复：按账面 dbname 重建句柄（无内存态），+11min 续扫——介入升级一次、
	// 催办不因账面已有而重复
	reopenCfg := &config.MySQLConfig{
		Host:     envOr("WBFLOW_MYSQL_HOST", "192.168.3.100"),
		Port:     3306,
		User:     envOr("WBFLOW_MYSQL_USER", "root"),
		Password: os.Getenv("WBFLOW_MYSQL_PASSWORD"),
		DBName:   dbName,
	}
	if p := os.Getenv("WBFLOW_MYSQL_PORT"); p != "" {
		port, convErr := strconv.ParseInt(p, 10, 64)
		assert.NilError(t, convErr)
		reopenCfg.Port = port
	}
	reopened, err := database.Open(reopenCfg)
	assert.NilError(t, err)
	t.Cleanup(func() { _ = reopened.Close() })

	scanner2 := service.NewScanner(repo.NewRepo(reopened.Gorm), reopened, scannerTestOpts()).
		WithClock(func() int64 { return task.CreatedAt + 11*60*1000 })
	report, err = scanner2.RunOnce(ctx)
	assert.NilError(t, err)
	assert.Equal(t, report.Degraded, false)
	assert.DeepEqual(t, report.Intervenes, []string{taskID})
	// 账面恰两条通知：断连前催办 1 + 恢复后介入 1（催办幂等跨重启成立）
	assert.Equal(t, countNotification(t, reopened, "followup", taskID), int64(1))
	assert.Equal(t, countNotification(t, reopened, "intervene", taskID), int64(1))
}

// TestIntegration_SessionBinding_RealMySQL sessions 绑定：status 迁移全分支 +
// 非法迁移拦截 + 同态幂等 no-op + 审计 + persona 解析器占位 + 读端。
func TestIntegration_SessionBinding_RealMySQL(t *testing.T) {
	router, handles := openScratch(t)
	_ = router
	ctx := context.Background()
	baseRepo := repo.NewRepo(handles.Gorm)
	sessSvc := service.NewSessionService(baseRepo, &stubPersona{})

	// 1. 绑定：占位句柄 + persona 注入；status=active
	session, err := sessSvc.Bind(ctx, service.SessionBindInput{
		RequirementID: "req-it-s", NodeID: "n-03", RoleID: "rd",
	})
	assert.NilError(t, err)
	assert.Assert(t, session.SessionID != "")
	assert.Equal(t, session.Status, "active")
	assert.Assert(t, strings.Contains(session.DshRef, `"kind":"placeholder"`), "占位句柄: %s", session.DshRef)
	assert.Assert(t, strings.Contains(session.DshRef, `"persona":"节点 persona rd"`), "persona 注入: %s", session.DshRef)

	// 2. 非法迁移拦截：active→disposed 跳段（ErrInvalidTransition，HTTP 409 语义）
	_, _, err = sessSvc.Dispose(ctx, session.SessionID)
	assert.Assert(t, errs.Is(err, errs.ErrInvalidTransition), "跳段应拦截: %v", err)

	// 3. active→ended 合法；同态再 End 幂等 no-op（库不变）
	_, noop, err := sessSvc.End(ctx, session.SessionID)
	assert.NilError(t, err)
	assert.Equal(t, noop, false)
	row, noop, err := sessSvc.End(ctx, session.SessionID)
	assert.NilError(t, err)
	assert.Equal(t, noop, true)
	assert.Equal(t, row.Status, "ended")

	// 4. ended→disposed 合法；disposed 后再 End 拦截
	_, _, err = sessSvc.Dispose(ctx, session.SessionID)
	assert.NilError(t, err)
	_, _, err = sessSvc.End(ctx, session.SessionID)
	assert.Assert(t, errs.Is(err, errs.ErrInvalidTransition), "disposed 应终态: %v", err)
	fresh, err := sessSvc.Get(ctx, session.SessionID)
	assert.NilError(t, err)
	assert.Equal(t, fresh.Status, "disposed")

	// 5. 读端：Get 未命中 ErrNotFound；按需求列表
	_, _, err = sessSvc.End(ctx, "s-not-exist")
	assert.Assert(t, errs.Is(err, errs.ErrNotFound))
	list, err := sessSvc.ListByRequirement(ctx, "req-it-s")
	assert.NilError(t, err)
	assert.Equal(t, len(list), 1)

	// 6. 入参拦截：role_id 缺失 ErrInvalidArgument
	_, err = sessSvc.Bind(ctx, service.SessionBindInput{RequirementID: "req-it-s"})
	assert.Assert(t, errs.Is(err, errs.ErrInvalidArgument))

	// 7. 写路径审计：INSERT 1 + status UPDATE 2（ended+disposed）
	assert.Equal(t, countAudit(t, handles, "sessions", session.SessionID, "INSERT"), int64(1))
	assert.Equal(t, countAudit(t, handles, "sessions", session.SessionID, "UPDATE"), int64(2))
}

// stubPersona persona 解析器占位桩（验证接口接线；真实数据源 T15+）。
type stubPersona struct{}

// PersonaFor 返回确定性 persona 文本。
func (p *stubPersona) PersonaFor(_ context.Context, _ string, _ string, roleID string) (string, error) {
	return "节点 persona " + roleID, nil
}
