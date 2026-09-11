//go:build e2e

// T17 剧本 task：task scope=rd 单侧 7 节点无 QA（tsk-c0/c1/c2/c3/c4/c6/c7）——
// 走通至用户验收；报障（issue 挂 exec 节点 tsk-c4）assignee 直连开发执行者
// （T7 推导口径：显式指派 > 锚点产物登记人 > 节点 exec 能力归属）（§4.1 P5 AC）。
package e2e

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestE2E_Task_Flow(t *testing.T) {
	h := startHarness(t, "task")
	s := h.newScript()

	// ---- 建需求：task × rd，7 节点无 QA ----
	s.create("任务端到端演练需求", "task", "rd", "rd")
	s.assertInstantiated([]string{
		"tsk-c0", "tsk-c1", "tsk-c2", "tsk-c3", "tsk-c4", "tsk-c6", "tsk-c7",
	})
	assert.Equal(t, s.flow().str("scope"), "rd")

	// ---- tsk-c0 待启动：User 确认 → tsk-c1 需求对齐（entry 迎宾 hold） ----
	t.Log("tsk-c0 User 确认 → tsk-c1 对齐 entry hold")
	s.decide("tsk-c0", "approved", "确认投入资源")
	s.assertNode("tsk-c1", "entry")
	s.assertCurrent("tsk-c1")
	// tsk-c1 entry 挂 align.opening（响应者=executor）；exec 跳过（✅D2 维持无产物）
	myTask := h.ok(h.do("POST", "/api/requirements/"+s.reqID+"/my-task", map[string]any{
		"requirementId": s.reqID, "role": "executor", "nodeId": s.node("tsk-c1"),
	}), "my_task executor@tsk-c1")
	actions := myTask.arr(t, "activeActions")
	assert.Equal(t, len(actions), 1, "c1 entry 应仅挂迎宾动作")
	if len(actions) == 1 {
		assert.Equal(t, actions[0].str("actionId"), "align.opening", "c1 entry 迎宾动作")
	}

	// ---- tsk-c1 用户确认对齐完成（空转 collect）→ tsk-c2 技术设计 ----
	// （entry 迎宾 align.opening=program 渲染交互，executor 响应者投影见上）
	t.Log("tsk-c1 用户确认对齐完成 → tsk-c2 技术设计")
	s.decide("tsk-c1", "approved", "对齐结论用户确认")
	s.assertCurrent("tsk-c2")
	s.assertNode("tsk-c2", "exec")

	// ---- tsk-c2 技术设计（backend-design by scope=rd）→ 提评迁移 tsk-c3 ----
	t.Log("tsk-c2 技术设计：backend-design.produce → 提评迁移")
	s.execute("tsk-c2", "backend-design.produce", "rd", "rd_design")
	s.advanceNode("tsk-c2")
	s.assertNode("tsk-c3", "review")

	// ---- tsk-c3 技术评审（arch 单评审位 by scope=rd，user_gate=1）→ User 拍板 ----
	t.Log("tsk-c3 技术评审：arch agree → User 拍板")
	s.reviewAndCarry("tsk-c3", [][2]string{{"arch", "arch.rd-design-review"}})
	s.decide("tsk-c3", "approved", "技术评审通过")
	s.assertCurrent("tsk-c4")
	s.assertNode("tsk-c4", "exec")

	// ---- tsk-c4 开发（CR 内嵌）→ 机械 approved → tsk-c6 用户验收 ----
	t.Log("tsk-c4 开发：backend-dev.dev → CR agree 机械 approved")
	s.execute("tsk-c4", "backend-dev.dev", "rd", "dev_notes", "code_mr")
	s.advanceNode("tsk-c4")
	s.assertNode("tsk-c4", "review")
	s.reviewAndCarry("tsk-c4", [][2]string{{"cr", "code-review"}})
	s.assertCurrent("tsk-c6")
	s.assertNode("tsk-c6", "entry")

	// ---- 报障直连：issue 挂 exec 节点 tsk-c4，assignee 推导直连开发执行者 rd ----
	t.Log("tsk-c6 用户验收报障：issue 挂 tsk-c4（无显式指派）→ assignee=rd（T7 L2 推导）")
	commentR := h.ok(h.do("POST", "/api/requirements/"+s.reqID+"/comments", map[string]any{
		"requirementId": s.reqID, "nodeId": s.node("tsk-c4"),
		"author": "user-a", "roleId": "user", "text": "验收发现边界条件报错，请确认",
	}), "用户报障")
	report := commentR.item(t, "issue")
	issueID := report.str("issueId")
	assert.Equal(t, report.str("assignee"), "rd", "报障应直连开发执行者 rd（节点 exec 能力归属推导）")
	// 聚合快照回读：报障挂 exec 节点、open 未闭合
	reqSnapshot := h.ok(h.do("GET", "/api/requirements/"+s.reqID, nil), "聚合快照")
	found := false
	for _, iss := range reqSnapshot.arr(t, "issues") {
		if iss.str("issueId") == issueID {
			found = true
			assert.Equal(t, iss.str("nodeId"), s.node("tsk-c4"), "报障应挂 exec 节点")
			assert.Equal(t, iss.str("state"), "open", "报障应 open")
		}
	}
	assert.Assert(t, found, "聚合快照应含报障 issue %s", issueID)
	// rd 收件箱可感知报障（issue 类通知不强制——open issue 经 my_task relatedIssues 呈现）
	myTaskRD := h.ok(h.do("POST", "/api/requirements/"+s.reqID+"/my-task", map[string]any{
		"requirementId": s.reqID, "role": "rd", "nodeId": s.node("tsk-c4"),
	}), "my_task rd@tsk-c4")
	assert.Assert(t, len(myTaskRD.arr(t, "relatedIssues")) > 0, "rd my_task 应呈现未闭合报障")

	// ---- rd 澄清闭环（协商一致）→ User 终拍板 → tsk-c7 完成 ----
	t.Log("rd 报障澄清闭环 → tsk-c6 User 拍板 approved → tsk-c7 完成")
	s.settle(issueID, "rd")
	s.decide("tsk-c6", "approved", "报障已澄清，用户验收通过")
	s.assertCurrent("tsk-c7")
	s.assertNode("tsk-c7", "entry")

	// ---- 终态与对账断言 ----
	t.Log("终态断言：7 节点全链走通、报障直连口径、审计对账")
	assert.Assert(t, s.snapshotNode("tsk-c7").number(t, "enteredAt") > 0, "尾节点应已进入")
	s.assertLastDecision("tsk-c6", "approved")
	assert.Equal(t, h.auditCountAll("tasks", "INSERT"), int64(s.tasksAssigned),
		"任务 INSERT 审计应与派发步数对账")
	assert.Equal(t, h.auditCountAll("artifacts", "INSERT"), int64(s.artifactsAdded),
		"产物 INSERT 审计应与登记步数对账")
	assert.Equal(t, h.auditCount("flows", s.reqID, "INSERT"), int64(1))
}
