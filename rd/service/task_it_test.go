// T5 涉库集成测试（AC）：wb_assign_task 幂等派发（同键重派无害/未激活 400/取号
// 并发唯一/branch 泳道）+ wb_task_update 三事件全分支（重复 ack/progress 覆写/
// 谎报 complete 判据拦截回 in-progress 且 missing 列出/未 ack complete 409/
// 已终态再上报 no-op/乐观锁 409）+ my_task relatedTasks 摘要 + 写路径审计。
// env 门控与 scratch 库口径沿用 T2/T3：WBFLOW_MYSQL_IT=1 + WBFLOW_MYSQL_* 注入，
// 独立 scratch 库（wbflow_it_*）测后 DROP——禁写 wbflow 库。密码仅经 env 零落地。
package service_test

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"gotest.tools/v3/assert"
)

// assignTask POST /api/tasks 薄封装。
func assignTask(t *testing.T, router *gin.Engine, reqID string, nodeID string, actionID string, assignee string) testResp {
	t.Helper()
	return reqJSON(t, router, http.MethodPost, "/api/tasks", map[string]any{
		"requirementId": reqID, "nodeId": nodeID, "actionId": actionID, "assignee": assignee,
	})
}

// updateTask PATCH /api/tasks/:taskId 薄封装（extra 携带 progress/updatedAt）。
func updateTask(t *testing.T, router *gin.Engine, taskID string, event string, extra map[string]any) testResp {
	t.Helper()
	body := map[string]any{"event": event}
	for k, v := range extra {
		body[k] = v
	}
	return reqJSON(t, router, http.MethodPatch, "/api/tasks/"+taskID, body)
}

// arrStrings 提取字符串数组出参（missing 等）。
func arrStrings(t *testing.T, tr testResp, key string) []string {
	t.Helper()
	list, ok := tr.body[key].([]any)
	assert.Assert(t, ok, "字段 %s 应为字符串数组，实测 %v", key, tr.body[key])
	out := make([]string, 0, len(list))
	for _, e := range list {
		s, ok := e.(string)
		assert.Assert(t, ok)
		out = append(out, s)
	}
	return out
}

// TestIntegration_TaskAssign_RealMySQL 派发端点：校验拦截 / 幂等命中 / 上下文装配 /
// my_task relatedTasks / node action 派发 / 泳道 branch / 取号并发唯一。
func TestIntegration_TaskAssign_RealMySQL(t *testing.T) {
	router, handles := openScratch(t)

	resp := reqJSON(t, router, http.MethodPost, "/api/requirements", map[string]any{
		"title": "任务总线需求", "category": "需求任务", "createdBy": "user-a",
	})
	assert.Equal(t, resp.code, http.StatusOK, "创建失败: %v", resp.body)
	reqID := item(t, resp, "requirement").str("requirementId")

	// 1. 入参拦截：缺必填 → 400
	for _, body := range []map[string]any{
		{"nodeId": "n-03", "actionId": "product-design.produce", "assignee": "pm"},
		{"requirementId": reqID, "actionId": "product-design.produce", "assignee": "pm"},
		{"requirementId": reqID, "nodeId": "n-03", "assignee": "pm"},
		{"requirementId": reqID, "nodeId": "n-03", "actionId": "product-design.produce"},
	} {
		resp := reqJSON(t, router, http.MethodPost, "/api/tasks", body)
		assert.Equal(t, resp.code, http.StatusBadRequest, "缺必填应 400: %v", body)
	}
	// context 非法 JSON → 400
	resp = reqJSON(t, router, http.MethodPost, "/api/tasks", map[string]any{
		"requirementId": reqID, "nodeId": "n-03", "actionId": "product-design.produce",
		"assignee": "pm", "context": "{bad",
	})
	assert.Equal(t, resp.code, http.StatusBadRequest)

	// 2. 目标校验：未知需求/节点 → 404；未激活动作/角色不符 → 400
	resp = assignTask(t, router, "req-999", "n-03", "product-design.produce", "pm")
	assert.Equal(t, resp.code, http.StatusNotFound)
	resp = assignTask(t, router, reqID, "n-99", "product-design.produce", "pm")
	assert.Equal(t, resp.code, http.StatusNotFound)
	// submit 不在 bus-02 激活清单（produce/revise）
	resp = assignTask(t, router, reqID, "n-03", "product-design.submit", "pm")
	assert.Equal(t, resp.code, http.StatusBadRequest)
	assert.Equal(t, resp.str("code"), "INVALID_ARGUMENT", "未激活动作 400")
	// 能力不在该节点激活（backend-dev 不在 bus-02 激活清单）
	resp = assignTask(t, router, reqID, "n-03", "backend-dev.dev", "rd")
	assert.Equal(t, resp.code, http.StatusBadRequest)
	// assignee 与能力归属角色不符
	resp = assignTask(t, router, reqID, "n-03", "product-design.produce", "rd")
	assert.Equal(t, resp.code, http.StatusBadRequest)

	// 3. 正常派发：bus-02（n-03）product-design.produce → pm
	resp = reqJSON(t, router, http.MethodPost, "/api/tasks", map[string]any{
		"requirementId": reqID, "nodeId": "n-03", "actionId": "product-design.produce", "assignee": "pm",
		"context": `{"extra":"补充"}`,
	})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	assert.Equal(t, resp.boolean("ok"), true)
	assert.Equal(t, resp.boolean("reused"), false)
	task1 := item(t, resp, "task")
	assert.Assert(t, strings.HasPrefix(task1.str("taskId"), "t-"), "task_id t-<ts>-<seq>: %s", task1.str("taskId"))
	assert.Equal(t, task1.str("status"), "todo", "派发后未 ack=todo")
	assert.Equal(t, task1.str("roleId"), "pm", "role_id 按 action 定位")
	assert.Equal(t, task1.str("taskType"), "produce")
	assert.Equal(t, task1.str("branch"), "", "full 双泳道 branch 空")
	assert.Equal(t, task1.number(t, "dispatchAt"), int64(0), "dispatch_at ack 落值")
	assert.Assert(t, strings.Contains(task1.str("context"), "product-design.produce"), "context 装配动作")
	assert.Assert(t, strings.Contains(task1.str("context"), "补充"), "context 透传补充")
	assert.Equal(t, auditCount(t, handles, "tasks", task1.str("taskId"), "INSERT"), int64(1))

	// 4. 幂等派发：同键重派无害（同任务、reused=true、不重复建、不重复审计）
	resp = assignTask(t, router, reqID, "n-03", "product-design.produce", "pm")
	assert.Equal(t, resp.code, http.StatusOK)
	assert.Equal(t, resp.boolean("reused"), true, "同键重派幂等命中")
	assert.Equal(t, item(t, resp, "task").str("taskId"), task1.str("taskId"))
	assert.Equal(t, auditCount(t, handles, "tasks", task1.str("taskId"), "INSERT"), int64(1), "幂等命中不写审计")

	// 5. my_task relatedTasks：该需求该节点在途任务摘要
	resp = reqJSON(t, router, http.MethodPost, "/api/requirements/"+reqID+"/my-task", map[string]any{
		"role": "pm", "nodeId": "n-03",
	})
	assert.Equal(t, resp.code, http.StatusOK)
	related := arr(t, resp, "relatedTasks")
	assert.Equal(t, len(related), 1)
	assert.Equal(t, related[0].str("taskId"), task1.str("taskId"))
	assert.Equal(t, related[0].str("actionId"), "product-design.produce")
	assert.Equal(t, related[0].str("status"), "todo")
	// 其他节点 relatedTasks 空
	resp = reqJSON(t, router, http.MethodPost, "/api/requirements/"+reqID+"/my-task", map[string]any{
		"role": "pm", "nodeId": "n-02",
	})
	assert.Equal(t, len(arr(t, resp, "relatedTasks")), 0)

	// 6. node action 派发：bus-01（n-02）to-brd（executorRole=pm）
	resp = assignTask(t, router, reqID, "n-02", "to-brd", "pm")
	assert.Equal(t, resp.code, http.StatusOK, "node action 派发: %v", resp.body)
	assert.Equal(t, item(t, resp, "task").str("taskType"), "produce")
	resp = assignTask(t, router, reqID, "n-02", "to-brd", "rd")
	assert.Equal(t, resp.code, http.StatusBadRequest, "node action 响应者角色不符 400")

	// 7. 泳道 branch：task×rd 需求（开发节点 tsk-c4=渲染序 5）派发 backend-dev.dev → branch rd
	resp = reqJSON(t, router, http.MethodPost, "/api/requirements", map[string]any{
		"title": "rd泳道任务", "category": "需求任务", "flowDefsId": "task", "scope": "rd", "createdBy": "user-a",
	})
	assert.Equal(t, resp.code, http.StatusOK)
	taskReqID := item(t, resp, "requirement").str("requirementId")
	resp = reqJSON(t, router, http.MethodGet, "/api/requirements/"+taskReqID, nil)
	devNodeID := arr(t, resp, "flowNodes")[4].str("nodeId")
	resp = assignTask(t, router, taskReqID, devNodeID, "backend-dev.dev", "rd")
	assert.Equal(t, resp.code, http.StatusOK, "task×rd 开发派发: %v", resp.body)
	assert.Equal(t, item(t, resp, "task").str("branch"), "rd", "rd 泳道 branch=rd")
	assert.Equal(t, item(t, resp, "task").str("taskType"), "dev")

	// 8. 取号并发：同键并发（6 worker）幂等收敛为单任务；异键并发号码唯一。
	const workers = 6
	taskIDs := make(chan string, workers)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(seq int) {
			defer wg.Done()
			<-start
			resp := assignTask(t, router, taskReqID, devNodeID, "backend-dev.diagnosing-bugs", "rd")
			assert.Equal(t, resp.code, http.StatusOK, "并发派发 %d: %v", seq, resp.body)
			taskIDs <- item(t, resp, "task").str("taskId")
		}(i)
	}
	close(start)
	wg.Wait()
	close(taskIDs)
	seen := map[string]bool{}
	for id := range taskIDs {
		seen[id] = true
	}
	assert.Equal(t, len(seen), 1, "同键并发派发幂等收敛为单任务")
	for id := range seen {
		assert.Equal(t, auditCount(t, handles, "tasks", id, "INSERT"), int64(1), "并发同键仅一次 INSERT")
	}

	// 异键并发：dev / fix_on_review（respond 族）同节点并行派发，号码唯一
	dualIDs := make(chan string, 2)
	start2 := make(chan struct{})
	var wg2 sync.WaitGroup
	for _, action := range []string{"backend-dev.dev", "backend-dev.fix_on_review"} {
		wg2.Add(1)
		go func(actionID string) {
			defer wg2.Done()
			<-start2
			resp := assignTask(t, router, taskReqID, devNodeID, actionID, "rd")
			assert.Equal(t, resp.code, http.StatusOK, "异键并发派发 %s: %v", actionID, resp.body)
			dualIDs <- item(t, resp, "task").str("taskId")
		}(action)
	}
	close(start2)
	wg2.Wait()
	close(dualIDs)
	dualSeen := map[string]bool{}
	for id := range dualIDs {
		dualSeen[id] = true
	}
	assert.Equal(t, len(dualSeen), 2, "异键并发取号唯一")
}

// TestIntegration_TaskUpdate_RealMySQL 上报端点全分支：ack/progress/complete 判据
// 框架（produce/review/respond/accept 四族实测）+ 幂等 no-op + 时序/乐观锁 409。
func TestIntegration_TaskUpdate_RealMySQL(t *testing.T) {
	router, handles := openScratch(t)

	resp := reqJSON(t, router, http.MethodPost, "/api/requirements", map[string]any{
		"title": "上报全分支需求", "category": "需求任务", "createdBy": "user-a",
	})
	assert.Equal(t, resp.code, http.StatusOK)
	reqID := item(t, resp, "requirement").str("requirementId")

	// 入参拦截：event 非法/缺失 → 400；任务不存在 → 404
	resp = reqJSON(t, router, http.MethodPatch, "/api/tasks/t-1-001", map[string]any{"event": "no-such"})
	assert.Equal(t, resp.code, http.StatusBadRequest)
	resp = reqJSON(t, router, http.MethodPatch, "/api/tasks/t-1-001", nil)
	assert.Equal(t, resp.code, http.StatusBadRequest)
	resp = updateTask(t, router, "t-1-001", "ack", nil)
	assert.Equal(t, resp.code, http.StatusNotFound)

	// 派发 produce 任务（bus-02/n-03，期望产物 product_doc）
	resp = reqJSON(t, router, http.MethodPost, "/api/tasks", map[string]any{
		"requirementId": reqID, "nodeId": "n-03", "actionId": "product-design.produce", "assignee": "pm",
	})
	assert.Equal(t, resp.code, http.StatusOK)
	taskID := item(t, resp, "task").str("taskId")

	// 时序违规：未 ack 先 complete → 409 STALE_REPORT
	resp = updateTask(t, router, taskID, "complete", nil)
	assert.Equal(t, resp.code, http.StatusConflict)
	assert.Equal(t, resp.str("code"), "STALE_REPORT")

	// progress 覆写（未 ack 也可报进度）：last_progress 落值 + 审计，状态不变
	resp = updateTask(t, router, taskID, "progress", map[string]any{"progress": "30% 初稿"})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	assert.Equal(t, item(t, resp, "task").str("lastProgress"), "30% 初稿")
	assert.Equal(t, item(t, resp, "task").str("status"), "todo", "progress 不改状态")
	assert.Equal(t, auditCount(t, handles, "tasks", taskID, "UPDATE"), int64(1))

	// ack：dispatch_at/last_ack 落值 + todo→in-progress（3 字段审计）
	resp = updateTask(t, router, taskID, "ack", nil)
	assert.Equal(t, resp.code, http.StatusOK)
	acked := item(t, resp, "task")
	assert.Equal(t, acked.str("status"), "in-progress")
	assert.Assert(t, acked.number(t, "dispatchAt") > 0, "dispatch_at 落值")
	assert.Assert(t, acked.number(t, "lastAck") > 0, "last_ack 落值")
	assert.Equal(t, auditCount(t, handles, "tasks", taskID, "UPDATE"), int64(4), "progress 1 + ack 3")

	// 重复 ack 幂等无害：noop=true、last_ack 不变、无新审计
	resp = updateTask(t, router, taskID, "ack", nil)
	assert.Equal(t, resp.code, http.StatusOK)
	assert.Equal(t, resp.boolean("noop"), true)
	assert.Equal(t, item(t, resp, "task").number(t, "lastAck"), acked.number(t, "lastAck"), "last_ack 不变")
	assert.Equal(t, auditCount(t, handles, "tasks", taskID, "UPDATE"), int64(4), "重复 ack 不写审计")

	// 乐观锁 409：陈旧基线（updated_at=1）
	resp = updateTask(t, router, taskID, "progress", map[string]any{"progress": "x", "updatedAt": 1})
	assert.Equal(t, resp.code, http.StatusConflict)
	assert.Equal(t, resp.str("code"), "OPTIMISTIC_LOCK")

	// 谎报完成：无产物登记 → 409 GATE_BLOCKED + missing=[artifact:product_doc]，任务回 in-progress
	resp = updateTask(t, router, taskID, "complete", nil)
	assert.Equal(t, resp.code, http.StatusConflict, "%v", resp.body)
	assert.Equal(t, resp.str("code"), "GATE_BLOCKED")
	assert.Equal(t, resp.boolean("blocked"), true)
	assert.DeepEqual(t, arrStrings(t, resp, "missing"), []string{"artifact:product_doc"})
	resp = updateTask(t, router, taskID, "progress", map[string]any{"progress": "仍在改"})
	assert.Equal(t, item(t, resp, "task").str("status"), "in-progress", "判据拦截后任务在途")

	// 登记产物（T6 前落库模拟）→ complete 放行：completed + done_at
	execSQL(t, handles, `INSERT INTO artifacts
		(artifact_id, requirement_id, node_id, product_type, label, href, added_by, created_at, updated_at)
		VALUES ('art-1', ?, 'n-03', 'product_doc', '《产品设计文档》', 'docs/x.md', 'pm', 1, 1)`, reqID)
	resp = updateTask(t, router, taskID, "complete", nil)
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	done := item(t, resp, "task")
	assert.Equal(t, done.str("status"), "completed")
	assert.Assert(t, done.number(t, "doneAt") > 0, "done_at 落值")
	assert.Equal(t, auditCount(t, handles, "tasks", taskID, "UPDATE"), int64(6), "+status/+done_at")

	// 已终态再上报幂等 no-op（重复 complete / ack / progress）
	for _, event := range []string{"complete", "ack", "progress"} {
		extra := map[string]any{}
		if event == "progress" {
			extra["progress"] = "迟到上报"
		}
		resp = updateTask(t, router, taskID, event, extra)
		assert.Equal(t, resp.code, http.StatusOK, "终态 %s 幂等 no-op", event)
		assert.Equal(t, resp.boolean("noop"), true)
	}
	assert.Equal(t, auditCount(t, handles, "tasks", taskID, "UPDATE"), int64(6), "终态再上报不写审计")

	// review 族（bus-03/n-04 评审位 rd）：无表态 → missing=review_statement:rd；表态后放行
	execSQL(t, handles, "UPDATE flow_nodes SET state = 'review' WHERE node_id = 'n-04'")
	resp = assignTask(t, router, reqID, "n-04", "rd.prd-review.review", "rd")
	assert.Equal(t, resp.code, http.StatusOK, "评审位激活 review 派发: %v", resp.body)
	reviewTask := item(t, resp, "task").str("taskId")
	assert.Equal(t, item(t, resp, "task").str("taskType"), "review")
	resp = updateTask(t, router, reviewTask, "ack", nil)
	assert.Equal(t, resp.code, http.StatusOK)
	resp = updateTask(t, router, reviewTask, "complete", nil)
	assert.Equal(t, resp.code, http.StatusConflict)
	assert.DeepEqual(t, arrStrings(t, resp, "missing"), []string{"review_statement:rd"})
	execSQL(t, handles,
		"UPDATE flow_nodes SET review_results = ? WHERE node_id = 'n-04'",
		`[{"role":"rd","capabilityId":"rd.prd-review","state":"agree","round":0,"at":1}]`)
	resp = updateTask(t, router, reviewTask, "complete", nil)
	assert.Equal(t, resp.code, http.StatusOK, "表态后 complete 放行: %v", resp.body)
	assert.Equal(t, item(t, resp, "task").str("status"), "completed")

	// respond 族（bus-03 clarify，激活清单）：无 open issue 放行；有指派 open issue 拦截
	execSQL(t, handles, "UPDATE flow_nodes SET state = 'respond' WHERE node_id = 'n-04'")
	resp = assignTask(t, router, reqID, "n-04", "product-design.clarify", "pm")
	assert.Equal(t, resp.code, http.StatusOK)
	respondTask := item(t, resp, "task").str("taskId")
	resp = updateTask(t, router, respondTask, "ack", nil)
	assert.Equal(t, resp.code, http.StatusOK)
	resp = updateTask(t, router, respondTask, "complete", nil)
	assert.Equal(t, resp.code, http.StatusOK, "无 open issue 放行: %v", resp.body)
	// 终态后同键可再派（重开语义）→ 新任务；指派 open issue → 判据拦截
	resp = assignTask(t, router, reqID, "n-04", "product-design.clarify", "pm")
	assert.Equal(t, resp.code, http.StatusOK)
	assert.Equal(t, resp.boolean("reused"), false, "终态后同键可再派")
	reopened := item(t, resp, "task").str("taskId")
	assert.Assert(t, reopened != respondTask, "应派发新任务")
	execSQL(t, handles, `INSERT INTO issues
		(issue_id, requirement_id, node_id, author, role_id, text, replies, assignee, created_at, updated_at)
		VALUES ('cm-1', ?, 'n-04', 'arch', 'arch', '第 2 节依据？', '[]', 'pm', 1, 1)`, reqID)
	resp = updateTask(t, router, reopened, "ack", nil)
	assert.Equal(t, resp.code, http.StatusOK)
	resp = updateTask(t, router, reopened, "complete", nil)
	assert.Equal(t, resp.code, http.StatusConflict)
	assert.DeepEqual(t, arrStrings(t, resp, "missing"), []string{"open_issues:pm(1)"})

	// accept 族（bus-10/n-11 QA 验收）：期望 test_result+defect 双缺 → 登记后放行
	resp = assignTask(t, router, reqID, "n-11", "acceptance-test.accept", "qa")
	assert.Equal(t, resp.code, http.StatusOK, "accept 派发: %v", resp.body)
	acceptTask := item(t, resp, "task").str("taskId")
	assert.Equal(t, item(t, resp, "task").str("taskType"), "accept")
	resp = updateTask(t, router, acceptTask, "ack", nil)
	assert.Equal(t, resp.code, http.StatusOK)
	resp = updateTask(t, router, acceptTask, "complete", nil)
	assert.Equal(t, resp.code, http.StatusConflict)
	assert.DeepEqual(t, arrStrings(t, resp, "missing"), []string{"artifact:test_result", "artifact:defect"})
	execSQL(t, handles, `INSERT INTO artifacts
		(artifact_id, requirement_id, node_id, product_type, label, href, added_by, created_at, updated_at)
		VALUES ('art-2', ?, 'n-11', 'test_result', '《测试用例与验收记录》', 'docs/t.md', 'qa', 1, 1)`, reqID)
	execSQL(t, handles, `INSERT INTO artifacts
		(artifact_id, requirement_id, node_id, product_type, label, href, added_by, created_at, updated_at)
		VALUES ('art-3', ?, 'n-11', 'defect', '《缺陷文档》', 'docs/d.md', 'qa', 1, 1)`, reqID)
	resp = updateTask(t, router, acceptTask, "complete", nil)
	assert.Equal(t, resp.code, http.StatusOK, "验收产物齐后放行: %v", resp.body)

	// 审计 by_who 口径：上报审计=执行角色（任务行更新由 agent 上报触发）
	var byWho string
	assert.NilError(t, handles.Gorm.WithContext(context.Background()).
		Raw("SELECT by_who FROM audit_log WHERE table_name='tasks' AND row_id=? AND action='UPDATE' LIMIT 1", taskID).
		Scan(&byWho).Error)
	assert.Equal(t, byWho, "pm", "上报审计 by_who=执行角色")

	// my_task relatedTasks：目标节点任务全部终态后摘要清零（n-03 的 produce 任务已 completed）
	resp = reqJSON(t, router, http.MethodPost, "/api/requirements/"+reqID+"/my-task", map[string]any{
		"role": "pm", "nodeId": "n-03",
	})
	assert.Equal(t, len(arr(t, resp, "relatedTasks")), 0, "终态任务不进 relatedTasks")
}
