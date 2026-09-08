//go:build e2e

// T17 剧本 rd-only：business scope=rd 动态重排 9 节点
// （bus-00/01/02/03/04/05/08/11/12）——FE 侧（bus-06/07/09）与 QA 验收（bus-10）
// 不渲染、不占位（node_order 连续重排），bus-03/bus-05 评审位 fe 位被裁，
// 走通至用户验收（§4.1 P5 AC）。
package e2e

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestE2E_BusinessRDOnly_Flow(t *testing.T) {
	h := startHarness(t, "rdonly")
	s := h.newScript()

	// ---- 建需求：business × rd，9 节点重排（FE 侧与 QA 不渲染不占位） ----
	s.create("纯后端端到端演练需求", "business", "rd", "pm")
	s.assertInstantiated([]string{
		"bus-00", "bus-01", "bus-02", "bus-03", "bus-04", "bus-05", "bus-08",
		"bus-11", "bus-12",
	})
	assert.Equal(t, s.flow().str("scope"), "rd")
	// 评审位裁剪：bus-03 rd 位保留、fe 位被裁；bus-05 qa/pm 保留、fe 位被裁
	bus03 := s.snapshotNode("bus-03").str("reviewers")
	assert.Assert(t, containsString(bus03, `"role":"rd"`), "bus-03 rd 位应保留")
	assert.Assert(t, !containsString(bus03, `"role":"fe"`), "bus-03 fe 位应被裁")
	bus05 := s.snapshotNode("bus-05").str("reviewers")
	assert.Assert(t, containsString(bus05, `"role":"qa"`), "bus-05 qa 位应保留")
	assert.Assert(t, !containsString(bus05, `"role":"fe"`), "bus-05 fe 位应被裁")

	// ---- bus-00 待启动：User 确认 → bus-01 对齐（entry hold → to-brd → 确认） ----
	t.Log("bus-00 User 确认 → bus-01 对齐（to-brd）→ 用户确认")
	s.decide("bus-00", "approved", "确认投入资源")
	s.assertNode("bus-01", "entry")
	s.execute("bus-01", "to-brd", "pm", "brd")
	s.decide("bus-01", "approved", "对齐结论用户确认")
	s.assertCurrent("bus-02")

	// ---- bus-02 产品设计 → 提评迁移 bus-03 ----
	t.Log("bus-02 产品设计 → 提评迁移")
	s.execute("bus-02", "product-design.produce", "pm", "product_doc")
	s.activateNext("bus-02")
	s.assertNode("bus-03", "review")

	// ---- bus-03 产品设计评审（arch/rd 双位）→ User 拍板 ----
	t.Log("bus-03 产品设计评审：arch/rd 全 agree → User 拍板")
	s.reviewAndCarry("bus-03", [][2]string{
		{"arch", "arch.prd-review"},
		{"rd", "rd.prd-review"},
	})
	s.decide("bus-03", "approved", "产品设计终审通过")
	s.assertCurrent("bus-04")

	// ---- bus-04 RD 技术设计 → 提评迁移 bus-05 ----
	t.Log("bus-04 RD 技术设计 → 提评迁移")
	s.execute("bus-04", "backend-design.produce", "rd", "rd_design")
	s.activateNext("bus-04")
	s.assertNode("bus-05", "review")

	// ---- bus-05 RD 技术评审（arch/qa/pm 三位）→ User 拍板 → 直达 bus-08 RD 开发 ----
	t.Log("bus-05 RD 技术评审：arch/qa/pm 全 agree → User 拍板（FE 侧被裁直达开发）")
	s.reviewAndCarry("bus-05", [][2]string{
		{"arch", "arch.rd-design-review"},
		{"qa", "qa.rd-design-review"},
		{"pm", "pm.rd-design-review"},
	})
	s.decide("bus-05", "approved", "技术评审通过")
	s.assertCurrent("bus-08")
	s.assertNode("bus-08", "exec")

	// ---- bus-08 RD 开发（CR 内嵌）→ 机械 approved → 直达 bus-11 用户验收 ----
	t.Log("bus-08 RD 开发：dev → CR agree 机械 approved（QA 节点不渲染直达用户验收）")
	s.execute("bus-08", "backend-dev.dev", "rd", "dev_notes", "code_mr")
	s.rotateSelf("bus-08")
	s.assertNode("bus-08", "review")
	s.reviewAndCarry("bus-08", [][2]string{{"cr", "code-review"}})
	s.assertCurrent("bus-11")
	s.assertNode("bus-11", "entry")

	// ---- bus-11 用户验收：User 终拍板 → bus-12 完成 ----
	t.Log("bus-11 用户验收：User 拍板 approved → bus-12 完成")
	s.decide("bus-11", "approved", "用户验收通过")
	s.assertCurrent("bus-12")
	s.assertNode("bus-12", "entry")

	// ---- 终态与对账断言 ----
	t.Log("终态断言：9 节点全链走通、审计对账")
	assert.Assert(t, s.snapshotNode("bus-12").number(t, "enteredAt") > 0, "尾节点应已进入")
	s.assertLastDecision("bus-11", "approved")
	assert.Equal(t, h.auditCountAll("tasks", "INSERT"), int64(s.tasksAssigned),
		"任务 INSERT 审计应与派发步数对账")
	assert.Equal(t, h.auditCountAll("artifacts", "INSERT"), int64(s.artifactsAdded),
		"产物 INSERT 审计应与登记步数对账")
	assert.Equal(t, h.auditCount("flows", s.reqID, "INSERT"), int64(1))
}
