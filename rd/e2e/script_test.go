//go:build e2e

// T17 剧本驱动框架（小 DSL）：演练者扮演全部 agent/User 角色，按节点剧本用 HTTP
// 驱动——assign_task → ack/progress/complete（complete 判据须先 add_artifact 登记
// 期望产物）→ per-agent review 表态 →（unagree 时澄清循环：comment/reply/resolve）→
// collect approved →（user_gate 时 decision 拍板）→ 推进下一节点；并断言节点推进/
// 相位/review_results/收件箱变化。三条剧本（full / rd-only / task）复用本框架。
//
// 编排接缝口径：exec 完成后的节点轮转（提评迁移 exec→review、节点内 CR 轮转、
// QA 机械收口 exec→entry）属 plugin 编排职责，本期无 HTTP 端点承载——演练以
// scratch 库内 SQL 接缝模拟（T9/T12 IT 同口径，域函数 domain.ActivationPhase /
// domain.InitReviewRound 复用，保证与 advanceApprovedNode 激活口径一致）。
package e2e

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/huzilin/wbflow/rd/domain"
	"github.com/huzilin/wbflow/rd/repo"
)

// script 单条剧本：一条需求实例 + 节点索引 + 步骤计数（审计对账数据源）。
type script struct {
	h     *harness
	reqID string
	order []string          // nodeID 按 node_order 升序（0 基：order[i] 序号 i+1）
	byDef map[string]string // defNodeID → nodeID
	defOf map[string]string // nodeID → defNodeID
	// 步骤计数：审计行数对账断言（tasks/artifacts INSERT 审计应与步骤数一致）
	tasksAssigned  int
	artifactsAdded int
}

// newScript 空剧本。
func (h *harness) newScript() *script {
	return &script{
		h:     h,
		byDef: map[string]string{},
		defOf: map[string]string{},
	}
}

// ---- 剧本步骤 ----

// create 建需求（HTTP wb_create_requirement）并装载节点索引；断言 scope 落库。
func (s *script) create(title string, flowDefsID string, scope string, owner string) {
	s.h.t.Helper()
	body := map[string]any{
		"title": title, "category": "需求任务", "createdBy": "user-a", "owner": owner,
	}
	if flowDefsID != "" {
		body["flowDefsId"] = flowDefsID
	}
	if scope != "" {
		body["scope"] = scope
	}
	r := s.h.ok(s.h.do("POST", "/api/requirements", body), "创建需求")
	s.reqID = r.item(s.h.t, "requirement").str("requirementId")
	assert.Assert(s.h.t, s.reqID != "", "需求 id 应非空")
	s.reload()
}

// reload 拉取聚合快照并重建节点索引（顺序 / def 映射）。
func (s *script) reload() {
	s.h.t.Helper()
	r := s.h.ok(s.h.do("GET", "/api/requirements/"+s.reqID, nil), "聚合快照")
	nodes := r.arr(s.h.t, "flowNodes")
	ordered := make([]resp, 0, len(nodes))
	ordered = append(ordered, nodes...)
	// 按 node_order 升序（protojson int64 字符串，逐个解析比较）
	for i := 0; i < len(ordered); i++ {
		for j := i + 1; j < len(ordered); j++ {
			if ordered[j].number(s.h.t, "nodeOrder") < ordered[i].number(s.h.t, "nodeOrder") {
				ordered[i], ordered[j] = ordered[j], ordered[i]
			}
		}
	}
	s.order = make([]string, 0, len(ordered))
	s.byDef = map[string]string{}
	s.defOf = map[string]string{}
	for _, n := range ordered {
		nodeID := n.str("nodeId")
		s.order = append(s.order, nodeID)
		s.byDef[n.str("defNodeId")] = nodeID
		s.defOf[nodeID] = n.str("defNodeId")
	}
}

// snapshotNode 拉取指定定义节点的最新快照行。
func (s *script) snapshotNode(defNodeID string) resp {
	s.h.t.Helper()
	r := s.h.ok(s.h.do("GET", "/api/requirements/"+s.reqID, nil), "聚合快照")
	for _, n := range r.arr(s.h.t, "flowNodes") {
		if n.str("defNodeId") == defNodeID {
			return n
		}
	}
	s.h.t.Fatalf("快照无节点 %s", defNodeID)
	return resp{}
}

// flow 拉取流程实例行（currentNodeId/scope）。
func (s *script) flow() resp {
	s.h.t.Helper()
	r := s.h.ok(s.h.do("GET", "/api/requirements/"+s.reqID, nil), "聚合快照")
	flows := r.arr(s.h.t, "flows")
	assert.Equal(s.h.t, len(flows), 1, "应恰有 1 条流程实例")
	return flows[0]
}

// node 定义节点 → 实例节点 id。
func (s *script) node(defNodeID string) string {
	s.h.t.Helper()
	nodeID, ok := s.byDef[defNodeID]
	assert.Assert(s.h.t, ok, "节点索引缺 %s（reload 未执行或 scope 被裁）", defNodeID)
	return nodeID
}

// assertInstantiated 断言实例化矩阵：节点集与序号（scope 渲染后重排、连续不占位）。
func (s *script) assertInstantiated(wantDefs []string) {
	s.h.t.Helper()
	assert.Equal(s.h.t, len(s.order), len(wantDefs), "实例节点数应与渲染矩阵一致")
	for i, wantDef := range wantDefs {
		assert.Equal(s.h.t, s.order[i], s.byDef[wantDef], "第 %d 位应为 %s（被裁节点不占位）", i+1, wantDef)
		node := s.snapshotNode(wantDef)
		assert.Equal(s.h.t, node.number(s.h.t, "nodeOrder"), int64(i+1), "%s 序号应连续", wantDef)
		assert.Equal(s.h.t, node.str("state"), "", "实例节点未进入编排 state 空: "+wantDef)
	}
}

// assertNode 断言节点当前相位。
func (s *script) assertNode(defNodeID string, wantState string) {
	s.h.t.Helper()
	got := s.snapshotNode(defNodeID).str("state")
	assert.Equal(s.h.t, got, wantState, "节点 %s 相位", defNodeID)
}

// assertCurrent 断言 flows.current_node_id。
func (s *script) assertCurrent(defNodeID string) {
	s.h.t.Helper()
	got := s.flow().str("currentNodeId")
	assert.Equal(s.h.t, got, s.node(defNodeID), "flows.current_node_id 应驻留 %s", defNodeID)
}

// assertLastDecision 断言节点门禁结论 verdict。
func (s *script) assertLastDecision(defNodeID string, wantVerdict string) {
	s.h.t.Helper()
	decision, err := domain.ParseGateDecision(s.snapshotNode(defNodeID).str("lastDecision"))
	assert.NilError(s.h.t, err)
	assert.Equal(s.h.t, decision.Verdict, wantVerdict, "节点 %s 门禁结论", defNodeID)
}

// assign wb_assign_task：派发动作并断言新建（非幂等命中），返回 task_id。
func (s *script) assign(defNodeID string, actionID string, assignee string) string {
	s.h.t.Helper()
	r := s.h.ok(s.h.do("POST", "/api/tasks", map[string]any{
		"requirementId": s.reqID, "nodeId": s.node(defNodeID),
		"actionId": actionID, "assignee": assignee,
	}), "派发 "+actionID)
	assert.Equal(s.h.t, r.boolean("reused"), false, "派发应新建任务（非幂等命中）")
	s.tasksAssigned++
	return r.item(s.h.t, "task").str("taskId")
}

// ack wb_task_update ack（todo → in-progress）。
func (s *script) ack(taskID string) {
	s.h.t.Helper()
	r := s.h.ok(s.h.do("PATCH", "/api/tasks/"+taskID, map[string]any{"event": "ack"}), "ack "+taskID)
	assert.Equal(s.h.t, r.str("status"), "in-progress", "ack 后应 in-progress")
}

// progress wb_task_update progress（演练覆盖上报通道）。
func (s *script) progress(taskID string, text string) {
	s.h.t.Helper()
	s.h.ok(s.h.do("PATCH", "/api/tasks/"+taskID, map[string]any{"event": "progress", "progress": text}), "progress "+taskID)
}

// complete wb_task_update complete（判据通过 → completed）。
func (s *script) complete(taskID string) {
	s.h.t.Helper()
	r := s.h.ok(s.h.do("PATCH", "/api/tasks/"+taskID, map[string]any{"event": "complete"}), "complete "+taskID)
	assert.Equal(s.h.t, r.str("status"), "completed", "complete 判据通过应 completed")
}

// addArtifact wb_add_artifact：登记 link 类产物（演练不落文档文件，外链校验最简）。
func (s *script) addArtifact(defNodeID string, productType string, addedBy string) {
	s.h.t.Helper()
	r := s.h.ok(s.h.do("POST", "/api/requirements/"+s.reqID+"/nodes/"+s.node(defNodeID)+"/artifacts", map[string]any{
		"productType": productType,
		"label":       "《e2e-" + productType + "》",
		"href":        fmt.Sprintf("https://docs.example.local/%s/%s", s.node(defNodeID), productType),
		"kind":        "link",
		"addedBy":     addedBy,
	}), "登记产物 "+productType)
	assert.Equal(s.h.t, r.item(s.h.t, "artifact").str("productType"), productType, "产物类型回显")
	s.artifactsAdded++
}

// execute 产出类节点标准步：派发 → ack → progress → 登记期望产物 → complete
// （complete 判据=期望产物已登记；同任务同 key 幂等由 T5 IT 覆盖，此处不重复）。
func (s *script) execute(defNodeID string, actionID string, assignee string, productTypes ...string) {
	s.h.t.Helper()
	s.h.t.Logf("  [%s] %s.by %s 产物=%v", defNodeID, actionID, assignee, productTypes)
	taskID := s.assign(defNodeID, actionID, assignee)
	s.ack(taskID)
	s.progress(taskID, "e2e 演练推进 "+actionID)
	for _, productType := range productTypes {
		s.addArtifact(defNodeID, productType, assignee)
	}
	s.complete(taskID)
}

// review wb_review：agent 自判表态并断言 collect 判定结果；
// approved 时断言 gate_state ∈ {agreed（user_gate 停等拍板）, approved（机械推进）}。
func (s *script) review(defNodeID string, role string, capabilityID string, state string, wantOutcome string) resp {
	s.h.t.Helper()
	s.h.t.Logf("  [%s] review.by %s(%s)=%s", defNodeID, role, capabilityID, state)
	r := s.h.ok(s.h.do("POST", "/api/requirements/"+s.reqID+"/nodes/"+s.node(defNodeID)+"/review", map[string]any{
		"requirementId": s.reqID, "nodeId": s.node(defNodeID),
		"role": role, "capabilityId": capabilityID, "state": state,
	}), "表态 "+role)
	assert.Equal(s.h.t, r.str("collectOutcome"), wantOutcome, "节点 %s collect 判定", defNodeID)
	if wantOutcome == "approved" {
		gateState := r.str("gateState")
		assert.Assert(s.h.t, gateState == "agreed" || gateState == "approved",
			"collect=approved 时 gate_state 应为 agreed|approved，实测 %q", gateState)
	}
	return r
}

// decide wb_decision：User 终审拍板（user_gate 位），断言 verdict 回显。
func (s *script) decide(defNodeID string, verdict string, reason string) resp {
	s.h.t.Helper()
	s.h.t.Logf("  [%s] decision.%s by user（%s）", defNodeID, verdict, reason)
	r := s.h.ok(s.h.do("POST", "/api/requirements/"+s.reqID+"/nodes/"+s.node(defNodeID)+"/decision", map[string]any{
		"requirementId": s.reqID, "nodeId": s.node(defNodeID),
		"verdict": verdict, "reason": reason,
	}), "拍板 "+defNodeID)
	assert.Equal(s.h.t, r.str("verdict"), verdict, "拍板结论回显")
	return r
}

// commentIssue wb_comment：评审/报障意见建行（issues.state=open），显式指派 assignee，
// 返回 issue_id。
func (s *script) commentIssue(defNodeID string, author string, roleID string, text string, assignee string) string {
	s.h.t.Helper()
	r := s.h.ok(s.h.do("POST", "/api/requirements/"+s.reqID+"/comments", map[string]any{
		"requirementId": s.reqID, "nodeId": s.node(defNodeID),
		"author": author, "roleId": roleID, "text": text, "assignee": assignee,
	}), "发表意见")
	issue := r.item(s.h.t, "issue")
	assert.Equal(s.h.t, issue.str("state"), "open", "新意见应为 open")
	return issue.str("issueId")
}

// resolveAgree wb_resolve path=agree：协商一致关闭（operator=assignee）。
func (s *script) resolveAgree(issueID string, operator string) {
	s.h.t.Helper()
	r := s.h.ok(s.h.do("POST", "/api/comments/"+issueID+"/resolve", map[string]any{
		"commentId": issueID, "path": "agree", "operator": operator,
	}), "澄清闭环 "+issueID)
	assert.Equal(s.h.t, r.item(s.h.t, "issue").str("state"), "resolved", "agree 关闭后应 resolved")
}

// respondClarify owner 澄清步：派发 respond 动作 → ack → 逐条闭环 open issue →
// complete（判据=该角色该节点 open 清零，同事务 review 重入新轮 + re_review 通知）。
func (s *script) respondClarify(defNodeID string, actionID string, assignee string, issueIDs ...string) {
	s.h.t.Helper()
	s.h.t.Logf("  [%s] respond %s.by %s（闭环 %d 条澄清）", defNodeID, actionID, assignee, len(issueIDs))
	taskID := s.assign(defNodeID, actionID, assignee)
	s.ack(taskID)
	for _, issueID := range issueIDs {
		s.resolveAgree(issueID, assignee)
	}
	s.complete(taskID)
	s.assertNode(defNodeID, "review")
}

// reviewAndCarry 全评审位 straight-agree 收口：逐位 agree（中间位 pending，末位
// 收齐触发 collect）；末位后按节点 user_gate 断言停等拍板（agreed）或机械通过
// （approved——current 已前移）。
func (s *script) reviewAndCarry(defNodeID string, positions [][2]string) {
	s.h.t.Helper()
	for i, pos := range positions {
		outcome := "pending"
		if i == len(positions)-1 {
			outcome = "approved"
		}
		r := s.review(defNodeID, pos[0], pos[1], "agree", outcome)
		if i == len(positions)-1 {
			if s.snapshotNode(defNodeID).number(s.h.t, "userGate") == 1 {
				assert.Equal(s.h.t, r.str("gateState"), "agreed", "user_gate=1 应停等拍板")
			} else {
				assert.Equal(s.h.t, r.str("gateState"), "approved", "user_gate=0 应机械通过")
			}
		}
	}
}

// activateNext 编排接缝：当前节点 exec 完成后的跨节点轮转（提评迁移 exec→review、
// QA 机械收口 exec→gate entry）——plugin 职责本期无 HTTP 承载，scratch 库内 SQL
// 模拟，激活口径复用 domain.ActivationPhase + domain.InitReviewRound（与
// advanceApprovedNode 一致）。
func (s *script) activateNext(fromDefNodeID string) {
	s.h.t.Helper()
	t := s.h.t
	fromNodeID := s.node(fromDefNodeID)
	fromOrder := 0
	for i, nodeID := range s.order {
		if nodeID == fromNodeID {
			fromOrder = i
			break
		}
	}
	assert.Assert(t, fromOrder+1 < len(s.order), "节点 %s 无后继（尾节点不该走接缝）", fromDefNodeID)
	nextNodeID := s.order[fromOrder+1]
	nextDefID := s.defOf[nextNodeID]

	nodeType := s.h.queryText(
		"SELECT node_type FROM flow_def_nodes WHERE template_node_id = ?", nextDefID)
	nodeActions := s.h.queryText(
		"SELECT node_actions FROM flow_def_nodes WHERE template_node_id = ?", nextDefID)
	reviewers := s.h.queryText(
		"SELECT reviewers FROM flow_def_nodes WHERE template_node_id = ?", nextDefID)

	phase, err := domain.ActivationPhase(nodeType, nodeActions)
	assert.NilError(t, err)
	now := repo.NowMillis()
	if phase == repo.PhaseReview {
		seeded, seedErr := domain.InitReviewRound(reviewers, 0, now)
		assert.NilError(t, seedErr)
		seededJSON := marshalJSON(t, seeded)
		s.h.execSQL(
			"UPDATE flow_nodes SET state = ?, entered_at = ?, review_results = ? WHERE requirement_id = ? AND node_id = ?",
			phase, now, seededJSON, s.reqID, nextNodeID)
	} else {
		s.h.execSQL(
			"UPDATE flow_nodes SET state = ?, entered_at = ? WHERE requirement_id = ? AND node_id = ?",
			phase, now, s.reqID, nextNodeID)
	}
	s.h.execSQL("UPDATE flows SET current_node_id = ? WHERE requirement_id = ?", nextNodeID, s.reqID)
	t.Logf("  [接缝] 编排轮转 %s → %s（state=%s）——exec 后跨节点推进 SQL 模拟（T9/T12 同口径）",
		fromNodeID, nextNodeID, phase)
}

// rotateSelf 节点内 CR 轮转接缝：开发节点 exec 完成（提评登记判据通过）→ review
// 相位本节点内重开（节点内评审异化，全位 reviewing 轮 0）——plugin 职责本期无
// HTTP 承载，scratch 库内 SQL 模拟；current 不动（CR 机械 approved 后经
// wb_review 内置推进迁移下一节点）。
func (s *script) rotateSelf(defNodeID string) {
	s.h.t.Helper()
	nodeID := s.node(defNodeID)
	reviewers := s.h.queryText(
		"SELECT reviewers FROM flow_def_nodes WHERE template_node_id = ?", defNodeID)
	now := repo.NowMillis()
	seeded, err := domain.InitReviewRound(reviewers, 0, now)
	assert.NilError(s.h.t, err)
	s.h.execSQL(
		"UPDATE flow_nodes SET state = ?, entered_at = ?, review_results = ? WHERE requirement_id = ? AND node_id = ?",
		repo.PhaseReview, now, marshalJSON(s.h.t, seeded), s.reqID, nodeID)
	s.h.t.Logf("  [接缝] 节点内 CR 轮转 %s exec → review——plugin 轮转 SQL 模拟（T9/T12 同口径）", nodeID)
}

// inboxKinds 收件箱按类别计数（按本需求过滤），断言通知触发与跳转口径数据源。
func (s *script) inboxKinds(role string) map[string]int {
	s.h.t.Helper()
	r := s.h.ok(s.h.do("GET", "/api/inbox?role="+role, nil), "收件箱 "+role)
	counts := map[string]int{}
	for _, n := range r.arr(s.h.t, "notifications") {
		if n.str("requirementId") != s.reqID {
			continue
		}
		counts[n.str("kind")]++
	}
	return counts
}

// assertInboxKind 断言某类别通知存在（至少 want 条）。
func (s *script) assertInboxKind(role string, kind string, want int) {
	s.h.t.Helper()
	got := s.inboxKinds(role)[kind]
	assert.Assert(s.h.t, got >= want, "角色 %s 应有 ≥%d 条 %s 通知，实测 %d（全量 %v）",
		role, want, kind, got, s.inboxKinds(role))
}

// assertAuditField 断言审计行存在（table+row+field+by_who）。
func (s *script) assertAuditField(table string, rowID string, field string, byWho string, want int64) {
	s.h.t.Helper()
	got := s.h.auditFieldCountBy(table, rowID, field, byWho)
	assert.Equal(s.h.t, got, want, "审计 %s.%s(%s) by %s", table, rowID, field, byWho)
}

// marshalJSON 紧凑 JSON 文本（接缝播种用）。
func marshalJSON(t *testing.T, v any) string {
	t.Helper()
	data, err := json.Marshal(v)
	assert.NilError(t, err)
	return string(data)
}

// containsString 子串包含判定（review_results 留痕断言用）。
func containsString(haystack string, needle string) bool {
	return strings.Contains(haystack, needle)
}
