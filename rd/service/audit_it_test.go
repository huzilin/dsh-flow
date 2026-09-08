// T13 涉库集成测试（AC，spec §4.1「审计：全部写路径落 audit_log」）：
//  1. 覆盖断言——穷举驱动 rd/service 全部业务写路径各一次，逐路径断言 audit_log
//     出现对应行（table_name/row_id/action/by_who）；append-only 回归（UPDATE 后
//     旧值行保留、created_at=updated_at 恒同值）；
//  2. 审计失败回滚——故障注入（audit_log 表不可写：RENAME 移出）验证业务事务
//     整体回滚（INSERT 多表路径与 UPDATE 路径各一），恢复后业务可重放。
//
// by_who 口径裁定背书（裁定表见工单报告）：管理面=user、agent/参与角色=role_id
// 原样、User 拍板/仲裁=user、编排迁移（collect/激活/前移/轮重入）=plugin、
// 会话生命周期=system、通知投递通道（repo.InsertNotification）=plugin（触发者
// 身份留 issues.author/role_id，如扫描器=system——T11 归一口径维持）。
//
// env 门控与 scratch 库口径沿用 T2：WBFLOW_MYSQL_IT=1 + WBFLOW_MYSQL_* 注入，
// 独立 scratch 库（wbflow_it_*）测后 DROP——禁写 wbflow 库，密码仅经 env 零落地。
//
// 实例节点 id 与需求序对应（T4 取号规则，全局取号跨需求递增）：
// 第 1 条需求 n-01..n-13、第 2 条 n-14..n-26（评审需求，bus-03=n-17）。
package service_test

import (
	"context"
	"fmt"
	"net/http"
	"os"
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

// auditITThreshold 注入小阈值（reopen≥2 自动 escalate，避免 5 轮拉锯）。
const auditITThreshold = 2

// openScratchAudit T13 全量 service 装配（含 Artifacts docs root 与 escalate 阈值注入）：
// 每测试独立 scratch 库，测后 DROP。
func openScratchAudit(t *testing.T) (*gin.Engine, *database.DB, string) {
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
		Logger:      zap.NewNop(),
		Health:      service.NewHealth(handles),
		Token:       itToken,
		Reqs:        service.NewRequirementService(r),
		Vers:        service.NewVersionService(r),
		Flows:       service.NewFlowService(r),
		Artifacts:   service.NewArtifactService(r, docsRoot),
		Tasks:       service.NewTaskService(r),
		Comments:    service.NewIssueService(r, auditITThreshold),
		Reviews:     service.NewReviewService(r),
		Escalations: service.NewEscalationService(r),
		Inbox:       service.NewInboxService(r),
	})
	return router, handles, docsRoot
}

// auditWhoCount 按 (table, row_id, action, by_who) 精确计审计行（覆盖断言主数据源）。
func auditWhoCount(t *testing.T, handles *database.DB, table string, rowID string, action string, byWho string) int64 {
	t.Helper()
	count := int64(0)
	err := handles.Gorm.WithContext(context.Background()).
		Raw("SELECT COUNT(*) FROM audit_log WHERE table_name = ? AND row_id = ? AND action = ? AND by_who = ?",
			table, rowID, action, byWho).Scan(&count).Error
	assert.NilError(t, err)
	return count
}

// auditTimestampInvariant 断言审计行 append-only 时间戳不变量：created_at=updated_at
// 恒同值（spec §0 表 #11）。
func auditTimestampInvariant(t *testing.T, handles *database.DB, table string, rowID string) {
	t.Helper()
	count := int64(0)
	err := handles.Gorm.WithContext(context.Background()).
		Raw("SELECT COUNT(*) FROM audit_log WHERE table_name = ? AND row_id = ? AND created_at <> updated_at",
			table, rowID).Scan(&count).Error
	assert.NilError(t, err)
	assert.Equal(t, count, int64(0), "审计行 created_at=updated_at 恒同值（append-only）")
}

// TestIntegration_AuditAllWritePaths_RealMySQL 全写路径覆盖断言：requirements/
// versions/artifacts/tasks/issues（评论五端点+escalate 手动置位）/flow_nodes（表态/
// 拍板/编排迁移）/flows（前移）/sessions（绑定生命周期+裁决建绑终局）/通知行（投递
// 通道 plugin、扫描器触发者 system 身份留行上）各驱动一次，逐路径断言审计行。
func TestIntegration_AuditAllWritePaths_RealMySQL(t *testing.T) {
	router, handles, _ := openScratchAudit(t)
	ctx := context.Background()

	// ---- versions：Create / Update / Advance / Deprecate（管理面 by_who=user）----
	resp := reqJSON(t, router, http.MethodPost, "/api/versions", map[string]any{"name": "v1"})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	v1 := item(t, resp, "version").str("versionId")
	assert.Equal(t, auditWhoCount(t, handles, "versions", v1, "INSERT", "user"), int64(1), "版本创建 INSERT 审计 by user")

	resp = reqJSON(t, router, http.MethodPost, "/api/versions", map[string]any{"name": "v2"})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	v2 := item(t, resp, "version").str("versionId")

	resp = reqJSON(t, router, http.MethodPut, "/api/versions/"+v2, map[string]any{"name": "v2-renamed"})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	assert.Equal(t, auditWhoCount(t, handles, "versions", v2, "UPDATE", "user"), int64(1), "版本改名 UPDATE 审计 by user")

	resp = reqJSON(t, router, http.MethodPost, "/api/versions/"+v2+"/advance", nil)
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	assert.Equal(t, auditFieldCount(t, handles, "versions", v2, "state"), int64(1), "版本发布 state 审计一次")
	assert.Equal(t, auditWhoCount(t, handles, "versions", v2, "UPDATE", "user"), int64(2), "版本 UPDATE 审计累计 by user")

	resp = reqJSON(t, router, http.MethodPost, "/api/versions/"+v2+"/deprecate", nil)
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	assert.Equal(t, auditFieldCount(t, handles, "versions", v2, "state"), int64(2), "版本废弃 state 审计累计")
	auditTimestampInvariant(t, handles, "versions", v2)

	// ---- requirements：Create（含 flows/flow_nodes 实例化）/ Update / Delete ----
	resp = reqJSON(t, router, http.MethodPost, "/api/requirements", map[string]any{
		"title": "审计主需求", "category": "需求任务", "createdBy": "user-a", "versionId": v1,
	})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	req1 := item(t, resp, "requirement").str("requirementId")
	assert.Equal(t, auditWhoCount(t, handles, "requirements", req1, "INSERT", "user"), int64(1), "需求创建 INSERT 审计 by user")
	assert.Equal(t, auditWhoCount(t, handles, "flows", req1, "INSERT", "user"), int64(1), "流程实例化 INSERT 审计 by user")
	assert.Equal(t, auditWhoCount(t, handles, "flow_nodes", "n-01", "INSERT", "user"), int64(1), "节点投影 INSERT 审计 by user")

	resp = reqJSON(t, router, http.MethodPut, "/api/requirements/"+req1, map[string]any{
		"title": "审计主需求（改名）", "category": "需求任务", "priority": "P2",
		"status": "待启动", "createdBy": "user-a", "versionId": v1,
	})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	assert.Equal(t, auditFieldCount(t, handles, "requirements", req1, "title"), int64(1), "需求改名 diff 审计一次")
	assert.Equal(t, auditWhoCount(t, handles, "requirements", req1, "UPDATE", "user"), int64(1), "需求更新 UPDATE 审计 by user")
	auditTimestampInvariant(t, handles, "requirements", req1)

	// ---- artifacts：Add（by_who=added_by，登记角色原样）----
	resp = reqJSON(t, router, http.MethodPost, "/api/requirements/"+req1+"/nodes/n-01/artifacts", map[string]any{
		"productType": "discussion", "label": "外链产物", "href": "https://example.com/a",
		"kind": "link", "addedBy": "pm",
	})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	artID := item(t, resp, "artifact").str("artifactId")
	assert.Equal(t, auditWhoCount(t, handles, "artifacts", artID, "INSERT", "pm"), int64(1), "产物登记 INSERT 审计 by added_by")

	// ---- tasks：Assign（管理面派发 by user）/ Update ack+progress（执行角色原样）----
	resp = reqJSON(t, router, http.MethodPost, "/api/tasks", map[string]any{
		"requirementId": req1, "nodeId": "n-03", "actionId": "product-design.produce", "assignee": "pm",
	})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	taskID := item(t, resp, "task").str("taskId")
	assert.Equal(t, auditWhoCount(t, handles, "tasks", taskID, "INSERT", "user"), int64(1), "任务派发 INSERT 审计 by user")

	resp = reqJSON(t, router, http.MethodPatch, "/api/tasks/"+taskID, map[string]any{"event": "ack"})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	assert.Equal(t, auditWhoCount(t, handles, "tasks", taskID, "UPDATE", "pm"), int64(3), "ack 逐字段审计（dispatchAt/lastAck/status）by 执行角色")

	resp = reqJSON(t, router, http.MethodPatch, "/api/tasks/"+taskID, map[string]any{"event": "progress", "progress": "推进中"})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	assert.Equal(t, auditFieldCount(t, handles, "tasks", taskID, "lastProgress"), int64(1), "progress 字段审计一次")
	assert.Equal(t, auditWhoCount(t, handles, "tasks", taskID, "UPDATE", "pm"), int64(4), "任务 UPDATE 审计累计 by 执行角色")
	auditTimestampInvariant(t, handles, "tasks", taskID)

	// ---- review：表态（by 表态角色）→ collect approved → 拍板（by user）→
	// 推进编排迁移（下一节点激活/前移 by plugin）。第 2 条需求第 4 节点 = n-17。
	// （顺序约束：需求删除仅删 requirements 行——flows 孤儿行占住 uk_flows_req，
	// 删除后再建需求会撞 req-NNN 取号复用；删除路径断言移到全测末尾。）
	req3 := setupReviewNode(t, router, handles, "2", "n-17")
	for _, position := range []struct{ role, capabilityID string }{
		{"arch", "arch.prd-review"}, {"rd", "rd.prd-review"}, {"fe", "fe.prd-review"},
	} {
		resp := reqJSON(t, router, http.MethodPost,
			"/api/requirements/"+req3+"/nodes/n-17/review",
			map[string]any{"role": position.role, "capabilityId": position.capabilityID, "state": "agree"})
		assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
		assert.Assert(t, auditWhoCount(t, handles, "flow_nodes", "n-17", "UPDATE", position.role) >= 1,
			"表态审计 by 表态角色（%s）", position.role)
	}
	assert.Equal(t, auditFieldCount(t, handles, "flow_nodes", "n-17", "reviewResults"), int64(3), "表态逐次审计")

	resp = reqJSON(t, router, http.MethodPost, "/api/requirements/"+req3+"/nodes/n-17/decision",
		map[string]any{"verdict": "approved", "reason": "评审通过", "branch": "feat/audit"})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	assert.Equal(t, auditWhoCount(t, handles, "flow_nodes", "n-17", "UPDATE", "user"), int64(1), "User 拍板审计 by user")
	assert.Equal(t, auditWhoCount(t, handles, "flow_nodes", "n-18", "UPDATE", "plugin"), int64(2), "下一节点激活审计 by plugin（state+enteredAt）")
	assert.Equal(t, auditWhoCount(t, handles, "flows", req3, "UPDATE", "plugin"), int64(1), "current_node_id 前移审计 by plugin")
	auditTimestampInvariant(t, handles, "flow_nodes", "n-17")

	// ---- issues：Comment（by 提交角色）/ Reply / Resolve(fix) / Confirm / Reopen ----
	newIssue := func(text string) string {
		t.Helper()
		resp := reqJSON(t, router, http.MethodPost, "/api/requirements/"+req3+"/comments", map[string]any{
			"nodeId": "n-17", "author": "arch", "roleId": "arch", "assignee": "pm", "text": text,
		})
		assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
		return item(t, resp, "issue").str("issueId")
	}
	issueReply := func(issueID string) {
		t.Helper()
		resp := reqJSON(t, router, http.MethodPost, "/api/comments/"+issueID+"/replies", map[string]any{
			"by": "pm", "roleId": "pm", "text": "回复说明",
		})
		assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	}

	issueA := newIssue("审计问题A（fix 路径）")
	assert.Equal(t, auditWhoCount(t, handles, "issues", issueA, "INSERT", "arch"), int64(1), "评论建行 INSERT 审计 by 提交角色")
	issueReply(issueA)
	assert.Equal(t, auditWhoCount(t, handles, "issues", issueA, "UPDATE", "pm"), int64(2), "reply 状态+追加审计 by 回复角色")
	resp = reqJSON(t, router, http.MethodPost, "/api/comments/"+issueA+"/resolve", map[string]any{
		"path": "fix", "operator": "pm", "note": "修复申报",
	})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	assert.Equal(t, auditFieldCount(t, handles, "issues", issueA, "state"), int64(2), "resolve 状态迁移审计（open→pending-confirm→to-fix）")
	assert.Equal(t, auditWhoCount(t, handles, "issues", issueA, "UPDATE", "pm"), int64(3), "resolve 审计 by 操作者")
	auditTimestampInvariant(t, handles, "issues", issueA)

	issueB := newIssue("审计问题B（confirm 复核）")
	issueReply(issueB)
	resp = reqJSON(t, router, http.MethodPost, "/api/comments/"+issueB+"/confirm", map[string]any{"operator": "arch"})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	assert.Equal(t, auditWhoCount(t, handles, "issues", issueB, "UPDATE", "arch"), int64(1), "confirm 复核状态迁移审计 by 提评者")

	issueC := newIssue("审计问题C（reopen+手动上升裁决）")
	issueReply(issueC)
	resp = reqJSON(t, router, http.MethodPost, "/api/comments/"+issueC+"/reopen", map[string]any{
		"operator": "arch", "text": "仍不满意",
	})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	assert.Equal(t, auditFieldCount(t, handles, "issues", issueC, "loop_count"), int64(1), "reopen 计数审计一次")
	assert.Equal(t, auditWhoCount(t, handles, "issues", issueC, "UPDATE", "arch"), int64(3), "reopen 审计（state+loop_count+replies）by 操作者")

	// ---- escalate：手动上升（operator=arch）——session 建绑/置位/上升条目/通知位 ----
	resp = reqJSON(t, router, http.MethodPost, "/api/comments/"+issueC+"/escalate", map[string]any{
		"operator": "arch", "reason": "拉锯不收敛",
	})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	adjudicationSession := resp.str("sessionId")
	assert.Equal(t, auditWhoCount(t, handles, "sessions", adjudicationSession, "INSERT", "arch"), int64(1), "裁决 session 建绑 INSERT 审计 by 操作者")
	assert.Equal(t, auditFieldCount(t, handles, "issues", issueC, "escalate"), int64(1), "手动置位 escalate 审计一次")
	assert.Equal(t, auditWhoCount(t, handles, "issues", issueC, "UPDATE", "arch"), int64(5), "escalate 写路径审计累计 by 操作者（reopen 3+置位+上升条目）")
	noticeC := scratchCol(t, handles,
		"SELECT issue_id FROM issues WHERE requirement_id = ? AND issue_type = 'notification' AND text LIKE ? ORDER BY id DESC LIMIT 1",
		req3, "%"+issueC+"%")
	assert.Assert(t, noticeC != "", "手动上升通知位行存在")
	assert.Equal(t, auditWhoCount(t, handles, "issues", noticeC, "INSERT", "arch"), int64(1), "通知位 INSERT 审计 by 触发操作者（同事务动作口径）")

	// ---- adjudicate：User 仲裁 override——issue 终态回填 + session 终局 + 轮重入 ----
	resp = reqJSON(t, router, http.MethodPost, "/api/comments/"+issueC+"/adjudicate", map[string]any{
		"verdict": "override", "conclusion": "议题不成立", "decided_by": "user",
	})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	assert.Assert(t, auditWhoCount(t, handles, "issues", issueC, "UPDATE", "user") >= 1, "仲裁回填审计 by user")
	assert.Equal(t, auditWhoCount(t, handles, "sessions", adjudicationSession, "UPDATE", "user"), int64(1), "session 终局审计 by user")
	assert.Assert(t, auditWhoCount(t, handles, "flow_nodes", "n-17", "UPDATE", "plugin") >= 2, "轮重入编排审计 by plugin")
	auditTimestampInvariant(t, handles, "issues", issueC)

	// ---- 自动阈值上升（reopen≥2，每轮 reply→reopen 拉锯）：置位审计 + 通知位
	// INSERT 审计 by plugin ----
	issueD := newIssue("审计问题D（自动上升）")
	for i := 0; i < auditITThreshold; i++ {
		issueReply(issueD)
		resp = reqJSON(t, router, http.MethodPost, "/api/comments/"+issueD+"/reopen", map[string]any{
			"operator": "arch", "text": "不收敛",
		})
		assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	}
	assert.Equal(t, auditFieldCount(t, handles, "issues", issueD, "escalate"), int64(1), "自动置位 escalate 审计一次")
	noticeD := scratchCol(t, handles,
		"SELECT issue_id FROM issues WHERE requirement_id = ? AND issue_type = 'notification' AND text LIKE ? ORDER BY id DESC LIMIT 1",
		req3, "%"+issueD+"%")
	assert.Assert(t, noticeD != "", "自动上升通知位行存在")
	assert.Equal(t, auditWhoCount(t, handles, "issues", noticeD, "INSERT", "plugin"), int64(1), "自动上升通知位 INSERT 审计 by plugin")

	// ---- sessions：Bind（by plugin）/ End·Dispose 生命周期（by system）----
	baseRepo := repo.NewRepo(handles.Gorm)
	sessSvc := service.NewSessionService(baseRepo, nil)
	session, err := sessSvc.Bind(ctx, service.SessionBindInput{RequirementID: req1, NodeID: "n-01", RoleID: "rd"})
	assert.NilError(t, err)
	assert.Equal(t, auditWhoCount(t, handles, "sessions", session.SessionID, "INSERT", "plugin"), int64(1), "会话绑定 INSERT 审计 by plugin")
	_, _, err = sessSvc.End(ctx, session.SessionID)
	assert.NilError(t, err)
	_, _, err = sessSvc.Dispose(ctx, session.SessionID)
	assert.NilError(t, err)
	assert.Equal(t, auditWhoCount(t, handles, "sessions", session.SessionID, "UPDATE", "system"), int64(2), "会话生命周期审计 by system（ended+disposed）")
	auditTimestampInvariant(t, handles, "sessions", session.SessionID)

	// ---- 通知投递通道（扫描器同款 Notify 事务外派发）：INSERT 审计恒 by plugin，
	// 触发者 system 身份留 issues.author/role_id（T11 归一口径裁定背书）----
	notifier := service.NewNotifier(baseRepo)
	_, err = notifier.Notify(ctx, service.NotificationSpec{
		Kind:          service.NotificationKindStallFollowup,
		RequirementID: req1,
		NodeID:        "n-03",
		TargetID:      taskID,
		RecipientRole: "pm",
		Sender:        "system",
		SenderRole:    "system",
		Text:          "[stall_followup] task=" + taskID + " 已结束 5min 未表态",
	})
	assert.NilError(t, err)
	scanNotice := scratchCol(t, handles,
		"SELECT issue_id FROM issues WHERE requirement_id = ? AND issue_type = 'notification' AND text LIKE ? ORDER BY id DESC LIMIT 1",
		req1, "%[stall_followup] task="+taskID+"%")
	assert.Assert(t, scanNotice != "", "扫描器同款通知行存在")
	assert.Equal(t, auditWhoCount(t, handles, "issues", scanNotice, "INSERT", "plugin"), int64(1), "通知投递 INSERT 审计 by plugin（通道口径）")
	assert.Equal(t, scratchCol(t, handles,
		"SELECT author FROM issues WHERE issue_id = ?", scanNotice), "system", "触发者身份留 issues.author（行上可见）")

	// ---- requirements：Delete（管理面 by_who=user；放末尾——删除仅删 requirements
	// 行，孤儿 flows 占住 uk_flows_req，其后再建需求会撞取号复用，见头部顺序约束）----
	resp = reqJSON(t, router, http.MethodPost, "/api/requirements", map[string]any{
		"title": "待删需求", "category": "需求任务", "createdBy": "user-a",
	})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	reqDel := item(t, resp, "requirement").str("requirementId")
	resp = reqJSON(t, router, http.MethodDelete, "/api/requirements/"+reqDel+"?confirm=true", nil)
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	assert.Equal(t, auditWhoCount(t, handles, "requirements", reqDel, "DELETE", "user"), int64(1), "需求删除 DELETE 审计 by user")
}

// TestIntegration_AuditAppendOnly_RealMySQL append-only 回归：同一行两次 UPDATE 后
// 旧审计行保留可追溯（前次 new_value=后次 old_value）、时间戳恒同值不变量成立。
func TestIntegration_AuditAppendOnly_RealMySQL(t *testing.T) {
	router, handles, _ := openScratchAudit(t)

	resp := reqJSON(t, router, http.MethodPost, "/api/requirements", map[string]any{
		"title": "T0", "category": "需求任务", "createdBy": "user-a",
	})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	reqID := item(t, resp, "requirement").str("requirementId")

	for _, title := range []string{"T1", "T2"} {
		resp = reqJSON(t, router, http.MethodPut, "/api/requirements/"+reqID, map[string]any{
			"title": title, "category": "需求任务", "priority": "P2",
			"status": "待启动", "createdBy": "user-a",
		})
		assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	}

	rows := auditRowsOrdered(t, handles, "requirements", reqID, "title")
	assert.Equal(t, len(rows), 2, "两次改名两行审计")
	assert.Equal(t, rows[0]["old_value"], "T0", "首行旧值")
	assert.Equal(t, rows[0]["new_value"], "T1", "首行新值")
	assert.Equal(t, rows[1]["old_value"], "T1", "次行旧值=前次新值（旧值可追溯）")
	assert.Equal(t, rows[1]["new_value"], "T2", "次行新值")
	assert.Equal(t, rows[0]["created_at"], rows[0]["updated_at"], "append-only：created_at=updated_at")
	assert.Equal(t, rows[1]["created_at"], rows[1]["updated_at"], "append-only：created_at=updated_at")
	// 旧行不被改写：审计行只增（INSERT 1 + title 2 + flows/flow_nodes 实例化族）
	total := int64(0)
	assert.NilError(t, handles.Gorm.WithContext(context.Background()).
		Raw("SELECT COUNT(*) FROM audit_log WHERE row_id = ?", reqID).Scan(&total).Error)
	assert.Assert(t, total >= 3, "审计行只增：INSERT+2×UPDATE")
}

// TestIntegration_AuditFailureRollback_RealMySQL 故障注入：audit_log 不可写
// （RENAME 移出）→ 审计写失败必须回滚业务事务（同库同事务，spec §2 审计失败行）。
// INSERT 多表路径（requirements+flows+flow_nodes）与 UPDATE 路径（issues 状态迁移）
// 各验一次；恢复表后业务可重放。
func TestIntegration_AuditFailureRollback_RealMySQL(t *testing.T) {
	router, handles, _ := openScratchAudit(t)

	// 基线：正常建一条需求 + 一条评论
	resp := reqJSON(t, router, http.MethodPost, "/api/requirements", map[string]any{
		"title": "回滚基线需求", "category": "需求任务", "createdBy": "user-a",
	})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	reqID := item(t, resp, "requirement").str("requirementId")
	resp = reqJSON(t, router, http.MethodPost, "/api/requirements/"+reqID+"/comments", map[string]any{
		"nodeId": "n-01", "author": "arch", "roleId": "arch", "text": "回滚基线评论",
	})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	issueID := item(t, resp, "issue").str("issueId")

	// 故障注入：audit_log 移出（同名表不存在 → 审计 INSERT 报错）
	execSQL(t, handles, "RENAME TABLE audit_log TO audit_log_bak")

	// INSERT 多表路径：需求创建整体回滚（requirements/flows/flow_nodes 全部不落）
	resp = reqJSON(t, router, http.MethodPost, "/api/requirements", map[string]any{
		"title": "审计故障需求", "category": "需求任务", "createdBy": "user-a",
	})
	assert.Equal(t, resp.code, http.StatusInternalServerError, "审计不可写应 500: %v", resp.body)
	assert.Equal(t, resp.boolean("ok"), false)
	failed := int64(0)
	assert.NilError(t, handles.Gorm.WithContext(context.Background()).
		Raw("SELECT COUNT(*) FROM requirements WHERE title = ?", "审计故障需求").Scan(&failed).Error)
	assert.Equal(t, failed, int64(0), "业务行回滚：需求未落库")
	orphan := int64(0)
	assert.NilError(t, handles.Gorm.WithContext(context.Background()).
		Raw("SELECT COUNT(*) FROM flows WHERE requirement_id NOT IN (SELECT requirement_id FROM requirements)").Scan(&orphan).Error)
	assert.Equal(t, orphan, int64(0), "同事务流程实例回滚：无孤儿 flows")

	// UPDATE 路径：reply 状态迁移整体回滚（issues 行不变）
	resp = reqJSON(t, router, http.MethodPost, "/api/comments/"+issueID+"/replies", map[string]any{
		"by": "pm", "roleId": "pm", "text": "审计故障回复",
	})
	assert.Equal(t, resp.code, http.StatusInternalServerError, "审计不可写应 500: %v", resp.body)
	assert.Equal(t, scratchCol(t, handles,
		"SELECT state FROM issues WHERE issue_id = ?", issueID), "open", "业务回滚：issue 状态未迁移")

	// 恢复：重命名回来后业务可重放（审计随行落库）
	execSQL(t, handles, "RENAME TABLE audit_log_bak TO audit_log")
	resp = reqJSON(t, router, http.MethodPost, "/api/requirements", map[string]any{
		"title": "审计故障需求", "category": "需求任务", "createdBy": "user-a",
	})
	assert.Equal(t, resp.code, http.StatusOK, "恢复后重放成功: %v", resp.body)
	replayed := item(t, resp, "requirement").str("requirementId")
	assert.Equal(t, auditWhoCount(t, handles, "requirements", replayed, "INSERT", "user"), int64(1), "重放审计落库")
}

// auditRowsOrdered 按审计行 id 序读取指定 (table, row_id, field) 的 UPDATE 行
// 关键列（append-only 回归断言数据源）。
func auditRowsOrdered(t *testing.T, handles *database.DB, table string, rowID string, field string) []map[string]string {
	t.Helper()
	type row struct {
		Field     string
		OldValue  string
		NewValue  string
		ByWho     string
		CreatedAt int64
		UpdatedAt int64
	}
	rows := make([]row, 0)
	err := handles.Gorm.WithContext(context.Background()).
		Raw("SELECT field, old_value, new_value, by_who, created_at, updated_at FROM audit_log "+
			"WHERE table_name = ? AND row_id = ? AND action = 'UPDATE' AND field = ? ORDER BY id",
			table, rowID, field).Scan(&rows).Error
	assert.NilError(t, err)
	out := make([]map[string]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, map[string]string{
			"field":      r.Field,
			"old_value":  r.OldValue,
			"new_value":  r.NewValue,
			"by_who":     r.ByWho,
			"created_at": fmt.Sprintf("%d", r.CreatedAt),
			"updated_at": fmt.Sprintf("%d", r.UpdatedAt),
		})
	}
	return out
}

// scratchCol 单值查询（SELECT 的首列文本；SQL 模板限内部字面量，不收外部输入）。
func scratchCol(t *testing.T, handles *database.DB, query string, args ...any) string {
	t.Helper()
	var out string
	err := handles.Gorm.WithContext(context.Background()).Raw(query, args...).Scan(&out).Error
	assert.NilError(t, err)
	return out
}
