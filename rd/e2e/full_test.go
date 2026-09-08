//go:build e2e

// T17 剧本 full：business scope=full 全链 13 节点（bus-00 → bus-12）——
// 对齐（entry hold → 用户确认 → to-brd 落盘 → 迁移）→ 产品设计 → 三轮评审
// （首轮含 unagree → 澄清 → 重入轮）→ RD∥FE 并行开发（CR 内嵌）→ QA 验收 →
// 用户验收拍板 → 完成；断言每步节点推进正确、flows 终态、审计行存在、
// 收件箱出现 gate_pending / clarify_task / re_review 通知（§4.1 P5 主 AC）。
package e2e

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestE2E_BusinessFull_Flow(t *testing.T) {
	h := startHarness(t, "full")
	s := h.newScript()

	// ---- 建需求：business × full，13 节点全渲染 ----
	s.create("全栈端到端演练需求", "business", "full", "pm")
	s.assertInstantiated([]string{
		"bus-00", "bus-01", "bus-02", "bus-03", "bus-04", "bus-05", "bus-06",
		"bus-07", "bus-08", "bus-09", "bus-10", "bus-11", "bus-12",
	})
	assert.Equal(t, s.flow().str("scope"), "full")
	assert.Equal(t, s.flow().str("currentNodeId"), "", "未进入编排 currentNodeId 空")
	// bus-10 QA 验收仅 full 渲染、bus-11 用户验收 user_gate=1（定义属性填充对照）
	assert.Equal(t, s.snapshotNode("bus-10").str("nodeType"), "produce")
	assert.Equal(t, s.snapshotNode("bus-11").number(t, "userGate"), int64(1))

	// ---- bus-00 待启动：User 确认投入资源（门禁直判，空转 collect） ----
	t.Log("bus-00 待启动：User 确认投入资源")
	s.decide("bus-00", "approved", "确认投入资源")
	s.assertCurrent("bus-01")
	s.assertNode("bus-01", "entry")

	// ---- bus-01 需求对齐：entry hold（迎宾驻留）→ to-brd 落盘 → 用户确认迁移 ----
	t.Log("bus-01 需求对齐：entry 迎宾 hold → to-brd 完成上报 → 用户确认")
	s.execute("bus-01", "to-brd", "pm", "brd")
	s.decide("bus-01", "approved", "对齐结论用户确认")
	s.assertCurrent("bus-02")
	s.assertNode("bus-02", "exec")

	// ---- bus-02 产品设计（produce 无内审）→ 提评迁移 bus-03 ----
	t.Log("bus-02 产品设计：product-design.produce（wb_my_task 三向投影对照）")
	myTask := h.ok(h.do("POST", "/api/requirements/"+s.reqID+"/my-task", map[string]any{
		"requirementId": s.reqID, "role": "pm", "nodeId": s.node("bus-02"),
	}), "my_task pm@bus-02")
	assert.Assert(t, len(myTask.arr(t, "activeActions")) > 0, "pm 在 exec 相位应有激活动作")
	s.execute("bus-02", "product-design.produce", "pm", "product_doc")
	s.activateNext("bus-02")
	s.assertCurrent("bus-03")
	s.assertNode("bus-03", "review")

	// ---- bus-03 产品设计评审：三轮（首轮 fe unagree → 澄清循环 → 复核通过） ----
	t.Log("bus-03 产品设计评审：arch/rd agree，fe 报 unagree（open 工单）→ collect=clarify")
	s.review("bus-03", "arch", "arch.prd-review", "agree", "pending")
	s.review("bus-03", "rd", "rd.prd-review", "agree", "pending")
	issueFe := s.commentIssue("bus-03", "fe-dev", "fe", "登录态刷新口径未定义，需澄清", "pm")
	s.review("bus-03", "fe", "fe.prd-review", "unagree", "clarify")
	s.assertNode("bus-03", "respond")
	s.assertInboxKind("pm", "clarify_task", 1)
	feSnapshot := s.snapshotNode("bus-03")
	assert.Assert(t, containsString(feSnapshot.str("reviewResults"), `"state":"unagree"`),
		"review_results 应留痕 unagree: %s", feSnapshot.str("reviewResults"))

	t.Log("bus-03 澄清循环：pm 澄清闭环 → review 重入新轮（re_review 通知 fe）→ fe 复核 agree")
	s.respondClarify("bus-03", "product-design.clarify", "pm", issueFe)
	s.assertInboxKind("fe", "re_review", 1)
	s.review("bus-03", "fe", "fe.prd-review", "agree", "approved")
	s.assertNode("bus-03", "review")
	s.assertInboxKind("pm", "gate_pending", 1)
	s.decide("bus-03", "approved", "产品设计终审通过")
	s.assertCurrent("bus-04")
	s.assertLastDecision("bus-03", "approved")

	// ---- bus-04 RD 技术设计 → 提评迁移 bus-05 ----
	t.Log("bus-04 RD 技术设计：backend-design.produce")
	s.execute("bus-04", "backend-design.produce", "rd", "rd_design")
	s.activateNext("bus-04")
	s.assertNode("bus-05", "review")

	// ---- bus-05 RD 技术评审：arch/fe/qa/pm 四位 straight agree → User 拍板 ----
	t.Log("bus-05 RD 技术评审：四位全 agree → agreed → User 拍板")
	s.reviewAndCarry("bus-05", [][2]string{
		{"arch", "arch.rd-design-review"},
		{"fe", "fe.rd-design-review"},
		{"qa", "qa.rd-design-review"},
		{"pm", "pm.rd-design-review"},
	})
	s.assertInboxKind("pm", "gate_pending", 2)
	decideR := s.decide("bus-05", "approved", "技术评审通过")
	assert.Equal(t, decideR.str("nextNodeId"), s.node("bus-06"), "拍板通过应迁移 bus-06")
	s.assertCurrent("bus-06")
	s.assertNode("bus-06", "exec")

	// ---- bus-06 FE 技术设计 → 提评迁移 bus-07 ----
	t.Log("bus-06 FE 技术设计：frontend-design.produce")
	s.execute("bus-06", "frontend-design.produce", "fe", "fe_design")
	s.activateNext("bus-06")
	s.assertNode("bus-07", "review")

	// ---- bus-07 FE 设计评审：qa 单评审位，user_gate=0 机械 approved 即时推进 ----
	t.Log("bus-07 FE 设计评审：qa 单位 agree → 机械 approved（current 前移 bus-08）")
	s.reviewAndCarry("bus-07", [][2]string{{"qa", "qa.fe-design-review"}})
	s.assertLastDecision("bus-07", "approved")
	s.assertCurrent("bus-08")
	s.assertNode("bus-08", "exec")

	// ---- bus-08 RD 开发（内嵌 CR——节点内评审异化） ----
	t.Log("bus-08 RD 开发：backend-dev.dev → CR 轮转（节点内接缝）→ cr agree 机械 approved")
	s.execute("bus-08", "backend-dev.dev", "rd", "dev_notes", "code_mr")
	s.rotateSelf("bus-08")
	s.assertNode("bus-08", "review")
	s.reviewAndCarry("bus-08", [][2]string{{"cr", "code-review"}})
	s.assertLastDecision("bus-08", "approved")
	s.assertCurrent("bus-09")
	s.assertNode("bus-09", "exec")

	// ---- bus-09 FE 开发（同构；RD∥FE 并行组按渲染序驱动，join=逐节点机械收口） ----
	t.Log("bus-09 FE 开发：frontend-dev.dev → CR 轮转 → cr agree 机械 approved")
	s.execute("bus-09", "frontend-dev.dev", "fe", "dev_notes", "code_mr")
	s.rotateSelf("bus-09")
	s.assertNode("bus-09", "review")
	s.reviewAndCarry("bus-09", [][2]string{{"cr", "code-review"}})
	s.assertLastDecision("bus-09", "approved")
	s.assertCurrent("bus-10")
	s.assertNode("bus-10", "exec")

	// ---- bus-10 QA 验收（仅 full 渲染）：acceptance-test.accept → 机械收口 → bus-11 ----
	t.Log("bus-10 QA 验收：acceptance-test.accept（test_result+defect）→ 收口迁移 bus-11")
	s.execute("bus-10", "acceptance-test.accept", "qa", "test_result", "defect")
	s.activateNext("bus-10")
	s.assertCurrent("bus-11")
	s.assertNode("bus-11", "entry")

	// ---- bus-11 用户验收：User 终拍板 → bus-12 完成（尾节点驻留） ----
	t.Log("bus-11 用户验收：User 拍板 approved → bus-12 完成")
	s.decide("bus-11", "approved", "用户验收通过")
	s.assertCurrent("bus-12")
	s.assertNode("bus-12", "entry")

	// ---- 终态与对账断言 ----
	t.Log("终态断言：flows 驻留 bus-12、审计对账、收件箱通知族、TODO 投影清零")
	assert.Assert(t, s.snapshotNode("bus-12").number(t, "enteredAt") > 0, "尾节点应已进入")
	s.assertLastDecision("bus-11", "approved")
	s.assertLastDecision("bus-03", "approved")
	// 审计（同库同事务——存在即留痕一致）：实例化/任务/产物 INSERT + 编排迁移
	// plugin 留痕 + 拍板 user 留痕，条数与步骤计数对账
	assert.Equal(t, h.auditCount("requirements", s.reqID, "INSERT"), int64(1))
	assert.Equal(t, h.auditCount("flows", s.reqID, "INSERT"), int64(1))
	for _, defID := range []string{"bus-00", "bus-06", "bus-12"} {
		assert.Equal(t, h.auditCount("flow_nodes", s.node(defID), "INSERT"), int64(1),
			"实例节点 %s 应有 INSERT 审计", defID)
	}
	assert.Equal(t, h.auditCountAll("tasks", "INSERT"), int64(s.tasksAssigned),
		"任务 INSERT 审计应与派发步数对账")
	assert.Equal(t, h.auditCountAll("artifacts", "INSERT"), int64(s.artifactsAdded),
		"产物 INSERT 审计应与登记步数对账")
	assert.Equal(t, h.auditFieldCountBy("flow_nodes", s.node("bus-05"), "lastDecision", "user"), int64(1),
		"bus-05 拍板应写 user 审计")
	assert.Assert(t, h.auditFieldCountBy("flow_nodes", s.node("bus-03"), "state", "plugin") >= 1,
		"bus-03 澄清轮转应有 plugin 编排审计")
	assert.Assert(t, h.auditFieldCountBy("flows", s.reqID, "currentNodeId", "plugin") >= 1,
		"flow 推进应有 plugin 审计")
	// 收件箱：gate_pending（bus-03/bus-05 agreed）、clarify_task（bus-03）、
	// re_review（fe 复核请求）——通知族在 full 链全触发
	s.assertInboxKind("pm", "gate_pending", 2)
	s.assertInboxKind("pm", "clarify_task", 1)
	s.assertInboxKind("fe", "re_review", 1)
	// TODO 投影：全链收口后在途任务不残留（completed/rejected 不投影）
	inbox := h.ok(h.do("GET", "/api/inbox?role=pm", nil), "收件箱 pm")
	for _, todo := range inbox.arr(t, "todos") {
		assert.Assert(t, todo.str("requirementId") != s.reqID,
			"pm TODO 不应残留本需求在途任务: %v", todo.body)
	}
}
