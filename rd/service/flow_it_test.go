// T4 涉库集成测试（AC）：三种 scope 实例化矩阵（full/rd/fe × business/tech/task/plan，
// 被裁不占位）+ wb_my_task 三向校验全分支 + 依赖树（含环检测）。
// env 门控与 scratch 库口径沿用 T2/T3：WBFLOW_MYSQL_IT=1 + WBFLOW_MYSQL_* 注入，
// 独立 scratch 库（wbflow_qa_it_*）测后 DROP——冒烟与测试禁写 wbflow 库。
// 密码仅经 env 零落地。
package service_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/huzilin/wbflow/rd/database"
	"github.com/huzilin/wbflow/rd/repo"
)

// TestIntegration_FlowInstantiation_RealMySQL 实例化矩阵：节点集与序号符合渲染矩阵
// （ui-interaction-final §1.1 渲染示例：full=13 全渲染、rd=9 节点单主链、fe 对称、
// QA 验收仅 full），被裁节点不占位；scope 越集 400；审计留痕。
func TestIntegration_FlowInstantiation_RealMySQL(t *testing.T) {
	router, handles := openScratch(t)

	create := func(title string, category string, flowDefsID string, scope string) string {
		body := map[string]any{"title": title, "category": category, "createdBy": "user-a"}
		if flowDefsID != "" {
			body["flowDefsId"] = flowDefsID
		}
		if scope != "" {
			body["scope"] = scope
		}
		resp := reqJSON(t, router, http.MethodPost, "/api/requirements", body)
		assert.Equal(t, resp.code, http.StatusOK, "创建失败: %v", resp.body)
		return item(t, resp, "requirement").str("requirementId")
	}
	snapshot := func(reqID string) ([]testResp, testResp) {
		resp := reqJSON(t, router, http.MethodGet, "/api/requirements/"+reqID, nil)
		assert.Equal(t, resp.code, http.StatusOK)
		return arr(t, resp, "flowNodes"), arr(t, resp, "flows")[0]
	}
	defIDs := func(nodes []testResp) []string {
		out := make([]string, 0, len(nodes))
		for _, n := range nodes {
			out = append(out, n.str("defNodeId"))
			assert.Equal(t, n.str("state"), "", "实例节点未进入编排: "+n.str("nodeId"))
			assert.Equal(t, n.str("reviewResults"), "[]")
		}
		return out
	}

	// business × full（显式）：13 节点全渲染，序号 1..13，node_id n-NN 与渲染序一致
	reqFull := create("全栈需求", "需求任务", "business", "full")
	nodes, flow := snapshot(reqFull)
	assert.Equal(t, flow.str("scope"), "full")
	assert.Equal(t, len(nodes), 13)
	assert.DeepEqual(t, defIDs(nodes), []string{
		"bus-00", "bus-01", "bus-02", "bus-03", "bus-04", "bus-05", "bus-06",
		"bus-07", "bus-08", "bus-09", "bus-10", "bus-11", "bus-12",
	})
	for i, n := range nodes {
		assert.Equal(t, n.number(t, "nodeOrder"), int64(i+1), "full 序号连续")
		assert.Equal(t, n.str("nodeId"), fmt.Sprintf("n-%02d", i+1), "node_id n-NN 与渲染序一致")
	}
	// full 下 bus-03 评审位全量（arch+rd+fe）
	assert.Assert(t, strings.Contains(nodes[3].str("reviewers"), `"role":"fe"`), "bus-03 fe 位 full 保留")
	// 定义属性填充：bus-10 gate=1 produce、bus-11 user_gate=1
	assert.Equal(t, nodes[10].str("nodeType"), "produce")
	assert.Equal(t, nodes[10].number(t, "gate"), int64(1))
	assert.Equal(t, nodes[11].str("nodeType"), "gate")
	assert.Equal(t, nodes[11].number(t, "userGate"), int64(1), "bus-11 用户验收 user_gate=1")

	// business × rd：9 节点单主链（FE 侧与 QA 不渲染、不占位、不占评审位）
	reqRD := create("纯后端需求", "需求任务", "business", "rd")
	nodes, flow = snapshot(reqRD)
	assert.Equal(t, flow.str("scope"), "rd")
	assert.DeepEqual(t, defIDs(nodes), []string{
		"bus-00", "bus-01", "bus-02", "bus-03", "bus-04", "bus-05", "bus-08", "bus-11", "bus-12",
	})
	for i, n := range nodes {
		assert.Equal(t, n.number(t, "nodeOrder"), int64(i+1), "rd 序号重排连续（被裁不占位）")
	}
	assert.Assert(t, !strings.Contains(nodes[3].str("reviewers"), `"role":"fe"`), "rd 实例 bus-03 fe 位被裁")
	assert.Assert(t, strings.Contains(nodes[3].str("reviewers"), `"role":"rd"`), "rd 实例 bus-03 rd 位保留")
	assert.Assert(t, !strings.Contains(nodes[5].str("reviewers"), `"role":"fe"`), "rd 实例 bus-05 fe 位被裁")
	assert.Assert(t, strings.Contains(nodes[5].str("reviewers"), `"role":"qa"`), "rd 实例 bus-05 qa 位保留")

	// business × fe：对称 9 节点
	reqFE := create("纯前端需求", "需求任务", "business", "fe")
	nodes, _ = snapshot(reqFE)
	assert.DeepEqual(t, defIDs(nodes), []string{
		"bus-00", "bus-01", "bus-02", "bus-03", "bus-06", "bus-07", "bus-09", "bus-11", "bus-12",
	})

	// tech × rd：7 节点（tec-b4/b5/b8b FE 侧被裁）
	reqTech := create("技术需求", "需求任务", "tech", "rd")
	nodes, _ = snapshot(reqTech)
	assert.DeepEqual(t, defIDs(nodes), []string{
		"tec-b0", "tec-b1", "tec-b2", "tec-b3", "tec-b8a", "tec-b10", "tec-b11",
	})

	// task × rd / × fe：rd|fe 表达式全渲染（7 节点无 QA）
	reqTaskRD := create("任务需求RD", "需求任务", "task", "rd")
	nodes, _ = snapshot(reqTaskRD)
	assert.DeepEqual(t, defIDs(nodes), []string{
		"tsk-c0", "tsk-c1", "tsk-c2", "tsk-c3", "tsk-c4", "tsk-c6", "tsk-c7",
	})
	reqTaskFE := create("任务需求FE", "需求任务", "task", "fe")
	nodes, _ = snapshot(reqTaskFE)
	assert.Equal(t, len(nodes), 7, "task fe 全渲染（rd|fe 表达式）")

	// plan：无 scope（""）全渲染 9 节点
	reqPlan := create("规划需求", "规划审批任务", "", "")
	nodes, flow = snapshot(reqPlan)
	assert.Equal(t, flow.str("scope"), "", "plan 无 scope")
	assert.Equal(t, len(nodes), 9)
	assert.Equal(t, nodes[0].str("defNodeId"), "pln-d0")

	// scope 拦截：task 不接受 full / plan 不接受任何 scope / business 越集
	resp := reqJSON(t, router, http.MethodPost, "/api/requirements", map[string]any{
		"title": "越集", "category": "需求任务", "flowDefsId": "task", "scope": "full", "createdBy": "user-a",
	})
	assert.Equal(t, resp.code, http.StatusBadRequest)
	assert.Equal(t, resp.str("code"), "INVALID_ARGUMENT")
	resp = reqJSON(t, router, http.MethodPost, "/api/requirements", map[string]any{
		"title": "plan带scope", "category": "规划审批任务", "scope": "rd", "createdBy": "user-a",
	})
	assert.Equal(t, resp.code, http.StatusBadRequest)
	resp = reqJSON(t, router, http.MethodPost, "/api/requirements", map[string]any{
		"title": "be不存在", "category": "需求任务", "scope": "be", "createdBy": "user-a",
	})
	assert.Equal(t, resp.code, http.StatusBadRequest)

	// 审计留痕：requirements/flows 各 1 条 INSERT；flow_nodes 逐节点
	assert.Equal(t, auditCount(t, handles, "requirements", reqRD, "INSERT"), int64(1))
	assert.Equal(t, auditCount(t, handles, "flows", reqRD, "INSERT"), int64(1))
	assert.Equal(t, auditCount(t, handles, "flow_nodes", "n-14", "INSERT"), int64(1),
		"第二需求（rd 9 节点）首节点，基值 13 续号")
}

// TestIntegration_MyTask_RealMySQL my_task 三向校验全分支：角色 × 节点 × 相位，
// 不属于该节点/该相位的角色返回空动作集而非报错（spec §1.2 口径）。
func TestIntegration_MyTask_RealMySQL(t *testing.T) {
	router, handles := openScratch(t)

	// business full：n-01..n-13 ↔ bus-00..bus-12
	resp := reqJSON(t, router, http.MethodPost, "/api/requirements", map[string]any{
		"title": "mytask需求", "category": "需求任务", "createdBy": "user-a",
	})
	assert.Equal(t, resp.code, http.StatusOK)
	reqID := item(t, resp, "requirement").str("requirementId")

	// 入参拦截：缺 role → 400；未知需求 → 404
	resp = reqJSON(t, router, http.MethodPost, "/api/requirements/"+reqID+"/my-task", map[string]any{})
	assert.Equal(t, resp.code, http.StatusBadRequest, "缺 role 应 400")
	resp = reqJSON(t, router, http.MethodPost, "/api/requirements/req-999/my-task", map[string]any{"role": "pm"})
	assert.Equal(t, resp.code, http.StatusNotFound)

	// 未进入编排：node 未填充、相位空、动作空集；relatedNodes=pm 相关集
	resp = reqJSON(t, router, http.MethodPost, "/api/requirements/"+reqID+"/my-task", map[string]any{"role": "pm"})
	assert.Equal(t, resp.code, http.StatusOK)
	assert.Equal(t, resp.boolean("ok"), true)
	assert.Assert(t, resp.body["node"] == nil, "未进入编排 node 应 null")
	assert.Equal(t, resp.str("currentPhase"), "")
	assert.Equal(t, len(arr(t, resp, "activeActions")), 0, "未进入编排动作空集")
	// pm 相关节点：bus-01(align.opening 响应者)/bus-02,bus-03(product-design 能力归属)/bus-05(pm 评审位)
	assert.DeepEqual(t, relatedIDs(t, arr(t, resp, "relatedNodes")), []string{"n-02", "n-03", "n-04", "n-06"})

	// 编排模拟（scratch 库内）：当前节点 n-03（bus-02 产品设计）进入 exec
	execSQL(t, handles, "UPDATE flows SET current_node_id = 'n-03' WHERE requirement_id = ?", reqID)
	execSQL(t, handles, "UPDATE flow_nodes SET state = 'exec' WHERE node_id = 'n-03'")

	// 三维全中：pm × n-03 × exec 拿首产配置（T9：exec 多配置按 rolled_back_count
	// 选择——rbc=0 出 produce，重入集 revise 剔除；submit 未激活）
	resp = reqJSON(t, router, http.MethodPost, "/api/requirements/"+reqID+"/my-task", map[string]any{"role": "pm"})
	assert.Equal(t, resp.code, http.StatusOK)
	node := item(t, resp, "node")
	assert.Equal(t, node.str("nodeId"), "n-03")
	assert.Equal(t, node.str("defNodeId"), "bus-02")
	assert.Equal(t, node.str("nodeType"), "produce")
	assert.Equal(t, node.number(t, "rolledBackCount"), int64(0))
	assert.Equal(t, resp.str("currentPhase"), "exec")
	active := arr(t, resp, "activeActions")
	assert.Equal(t, len(active), 1, "exec 首产配置 produce（revise 按 rbc=0 剔除，T9）")
	assert.Equal(t, active[0].str("actionId"), "product-design.produce", "BUG-006 口径：与 assign 入参同口径全名")
	assert.Equal(t, active[0].str("skill"), "to-prd")
	assert.Equal(t, active[0].str("capabilityId"), "product-design")

	// 角色维度不中：rd 同节点同相位 → 空动作集（不报错）
	resp = reqJSON(t, router, http.MethodPost, "/api/requirements/"+reqID+"/my-task", map[string]any{"role": "rd"})
	assert.Equal(t, resp.code, http.StatusOK)
	assert.Equal(t, len(arr(t, resp, "activeActions")), 0, "非能力归属角色空动作集")

	// 显式 nodeId 覆盖当前节点：n-04（bus-03 评审）未进入编排 → 空
	resp = reqJSON(t, router, http.MethodPost, "/api/requirements/"+reqID+"/my-task",
		map[string]any{"role": "pm", "nodeId": "n-04"})
	assert.Equal(t, resp.code, http.StatusOK)
	assert.Equal(t, item(t, resp, "node").str("defNodeId"), "bus-03")
	assert.Equal(t, resp.str("currentPhase"), "")
	assert.Equal(t, len(arr(t, resp, "activeActions")), 0)

	// 评审相位：review_results 摘要（per-agent 聚合计数与明细）
	execSQL(t, handles, "UPDATE flow_nodes SET state = 'review', review_results = ? WHERE node_id = 'n-04'",
		`[{"role":"arch","capabilityId":"arch.prd-review","state":"agree","round":0,"at":123},
		  {"role":"rd","capabilityId":"rd.prd-review","state":"unagree","round":0,"at":456},
		  {"role":"fe","capabilityId":"fe.prd-review","state":"reviewing","round":0,"at":789}]`)
	resp = reqJSON(t, router, http.MethodPost, "/api/requirements/"+reqID+"/my-task",
		map[string]any{"role": "arch", "nodeId": "n-04"})
	assert.Equal(t, resp.str("currentPhase"), "review")
	assert.Equal(t, len(arr(t, resp, "activeActions")), 0, "review 相位无 agent 动作（collect 内置）")
	summary := item(t, resp, "reviewSummary")
	assert.Equal(t, summary.number(t, "agree"), int64(1))
	assert.Equal(t, summary.number(t, "unagree"), int64(1))
	assert.Equal(t, summary.number(t, "reviewing"), int64(1))
	assert.Equal(t, len(arr(t, summary, "details")), 3)

	// respond 相位：pm（product-design 归属）拿 clarify
	execSQL(t, handles, "UPDATE flow_nodes SET state = 'respond' WHERE node_id = 'n-04'")
	resp = reqJSON(t, router, http.MethodPost, "/api/requirements/"+reqID+"/my-task",
		map[string]any{"role": "pm", "nodeId": "n-04"})
	active = arr(t, resp, "activeActions")
	assert.Equal(t, len(active), 1, "respond 相位 clarify 归 product-design(pm)")
	assert.Equal(t, active[0].str("actionId"), "product-design.clarify")

	// 节点维度不中：nodeId 不存在 → 404
	resp = reqJSON(t, router, http.MethodPost, "/api/requirements/"+reqID+"/my-task",
		map[string]any{"role": "pm", "nodeId": "n-99"})
	assert.Equal(t, resp.code, http.StatusNotFound)
}

// TestIntegration_MyTaskReviewArtifacts_RealMySQL BUG-011（2026-09-10 拍板）：评审
// 节点 wb_my_task relatedArtifacts 注入被评审产物——直接上游 produce 节点已登记
// 产物置前，本节点产物殿后；非评审节点不注入上游。
func TestIntegration_MyTaskReviewArtifacts_RealMySQL(t *testing.T) {
	router, handles := openScratch(t)
	reqID := createBusinessFullReq(t, router)

	// 夹具：上游 produce n-03（bus-02 产品设计）登记被评审产物；评审 n-04（bus-03）
	// 本节点登记一份
	for _, art := range []repo.Artifact{
		{ArtifactID: "art-it-rv-up", RequirementID: reqID, NodeID: "n-03",
			ProductType: "product_doc", Label: "《产品设计文档》", Href: "docs/product.md",
			Kind: "document", AddedBy: "pm"},
		{ArtifactID: "art-it-rv-own", RequirementID: reqID, NodeID: "n-04",
			ProductType: "product_doc", Label: "《评审纪要》", Href: "docs/review.md",
			Kind: "document", AddedBy: "arch"},
	} {
		row := art
		assert.NilError(t, handles.Gorm.Create(&row).Error)
	}

	// 评审节点：被评审产物（上游）置前 + 本节点产物殿后
	resp := reqJSON(t, router, http.MethodPost, "/api/requirements/"+reqID+"/my-task",
		map[string]any{"role": "arch", "nodeId": "n-04"})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	assert.Equal(t, item(t, resp, "node").str("nodeType"), "review")
	artifacts := arr(t, resp, "relatedArtifacts")
	assert.Equal(t, len(artifacts), 2, "上游被评审产物 + 本节点产物: %v", resp.body["relatedArtifacts"])
	assert.Equal(t, artifacts[0].str("artifactId"), "art-it-rv-up", "被评审产物置前（零跳转读上游文档）")
	assert.Equal(t, artifacts[1].str("artifactId"), "art-it-rv-own", "本节点产物殿后")

	// 非 review 节点（n-03 produce）只回本节点产物
	resp = reqJSON(t, router, http.MethodPost, "/api/requirements/"+reqID+"/my-task",
		map[string]any{"role": "pm", "nodeId": "n-03"})
	assert.Equal(t, resp.code, http.StatusOK)
	artifacts = arr(t, resp, "relatedArtifacts")
	assert.Equal(t, len(artifacts), 1)
	assert.Equal(t, artifacts[0].str("artifactId"), "art-it-rv-up")
}

// TestIntegration_DependencyTree_RealMySQL 依赖树：多层展开 + 悬挂边 found=false + 环检测 409。
func TestIntegration_DependencyTree_RealMySQL(t *testing.T) {
	router, handles := openScratch(t)

	create := func(title string) string {
		resp := reqJSON(t, router, http.MethodPost, "/api/requirements", map[string]any{
			"title": title, "category": "需求任务", "createdBy": "user-a",
		})
		assert.Equal(t, resp.code, http.StatusOK)
		return item(t, resp, "requirement").str("requirementId")
	}
	reqA := create("主需求A")
	reqB := create("前置需求B")
	reqC := create("前置需求C")

	// 依赖边：A → [B, C]；B → [C]（flows.dependencies 由 plan create_reqs 写入，
	// 本期无写端点，scratch 库内直接落库模拟）
	execSQL(t, handles, "UPDATE flows SET dependencies = ? WHERE requirement_id = ?", `["`+reqB+`","`+reqC+`"]`, reqA)
	execSQL(t, handles, "UPDATE flows SET dependencies = ? WHERE requirement_id = ?", `["`+reqC+`"]`, reqB)

	resp := reqJSON(t, router, http.MethodGet, "/api/requirements/"+reqA+"/dependencies/tree", nil)
	assert.Equal(t, resp.code, http.StatusOK)
	assert.Equal(t, resp.boolean("ok"), true)
	root := item(t, resp, "root")
	assert.Equal(t, root.str("requirementId"), reqA)
	assert.Equal(t, root.str("title"), "主需求A")
	assert.Equal(t, root.str("status"), "待启动")
	assert.Equal(t, root.boolean("found"), true)
	children := arr(t, root, "children")
	assert.Equal(t, len(children), 2)
	assert.Equal(t, children[0].str("requirementId"), reqB)
	assert.Equal(t, children[0].str("title"), "前置需求B")
	grandChildren := arr(t, children[0], "children")
	assert.Equal(t, len(grandChildren), 1)
	assert.Equal(t, grandChildren[0].str("requirementId"), reqC, "B 的子依赖 C 展开")
	assert.Equal(t, len(arr(t, children[1], "children")), 0)

	// 悬挂依赖：指向不存在需求 → found=false、title 空、不再展开
	execSQL(t, handles, "UPDATE flows SET dependencies = ? WHERE requirement_id = ?", `["req-009"]`, reqC)
	resp = reqJSON(t, router, http.MethodGet, "/api/requirements/"+reqA+"/dependencies/tree", nil)
	assert.Equal(t, resp.code, http.StatusOK)
	children = arr(t, item(t, resp, "root"), "children")
	assert.Equal(t, len(children), 2)
	cChildren := arr(t, children[1], "children")
	assert.Equal(t, len(cChildren), 1)
	assert.Equal(t, cChildren[0].str("requirementId"), "req-009")
	assert.Equal(t, cChildren[0].boolean("found"), false)
	assert.Equal(t, cChildren[0].str("title"), "")

	// 环检测：C → A 构成环（A→C→A）→ 409 DEPENDENCY_CYCLE
	execSQL(t, handles, "UPDATE flows SET dependencies = ? WHERE requirement_id = ?", `["`+reqA+`"]`, reqC)
	resp = reqJSON(t, router, http.MethodGet, "/api/requirements/"+reqA+"/dependencies/tree", nil)
	assert.Equal(t, resp.code, http.StatusConflict)
	assert.Equal(t, resp.str("code"), "DEPENDENCY_CYCLE")

	// 自环同样拦截
	execSQL(t, handles, "UPDATE flows SET dependencies = ? WHERE requirement_id = ?", `["`+reqC+`"]`, reqC)
	resp = reqJSON(t, router, http.MethodGet, "/api/requirements/"+reqA+"/dependencies/tree", nil)
	assert.Equal(t, resp.code, http.StatusConflict)
	assert.Equal(t, resp.str("code"), "DEPENDENCY_CYCLE")

	// 根需求 404
	resp = reqJSON(t, router, http.MethodGet, "/api/requirements/req-999/dependencies/tree", nil)
	assert.Equal(t, resp.code, http.StatusNotFound)
}

// ---- helper ----

// execSQL scratch 库内执行编排模拟/依赖边落库（绝不触 wbflow 库）。
func execSQL(t *testing.T, handles *database.DB, query string, args ...any) {
	t.Helper()
	assert.NilError(t, handles.Gorm.WithContext(context.Background()).Exec(query, args...).Error)
}

// relatedIDs 提取 relatedNodes 的 node_id 序列。
func relatedIDs(t *testing.T, related []testResp) []string {
	t.Helper()
	out := make([]string, 0, len(related))
	for _, r := range related {
		out = append(out, r.str("nodeId"))
	}
	return out
}
