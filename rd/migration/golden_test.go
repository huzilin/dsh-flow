// seed 黄金样本：40 定义节点定稿口径（business-flow-derivation.md v3.1 +
// flow-derivation-tech-task-plan.md v3.1 逐节点），与 0002_seed.sql 交叉核对。
// 本文件=docs-tmp 定稿的 Go 化转录；DB 落库值逐字段与之比对（TestIntegration_SeedGolden）。
package migration_test

import (
	"github.com/huzilin/wbflow/rd/pkg/utils"
	"github.com/huzilin/wbflow/rd/repo"
)

// goldenNodes 返回定稿口径的全部定义节点（与 seed SQL 独立转录，交叉验证）。
func goldenNodes() []repo.FlowDefNode {
	nodes := make([]repo.FlowDefNode, 0, 40)
	nodes = append(nodes, businessGolden()...)
	nodes = append(nodes, techGolden()...)
	nodes = append(nodes, taskGolden()...)
	nodes = append(nodes, planGolden()...)
	return nodes
}

// ---- 构造 helper（nil 切片经 MarshalJSONText 统一为 "[]"，与 seed 口径一致） ----

func j(v any) string {
	s, err := utils.MarshalJSONText(v)
	if err != nil {
		panic(err)
	}
	return s
}

func rev(role string, capabilityID string, scopeExpr ...string) repo.ReviewerDef {
	out := repo.ReviewerDef{Role: role, CapabilityID: capabilityID}
	if len(scopeExpr) > 0 {
		out.ScopeExpr = scopeExpr[0]
	}
	return out
}

func actCap(capabilityID string, actions ...string) repo.NodeActionDef {
	return repo.NodeActionDef{Kind: repo.NodeActionKindCapability, CapabilityID: capabilityID, Actions: actions}
}

func actCapScope(capabilityID string, scopeExpr string, actions ...string) repo.NodeActionDef {
	return repo.NodeActionDef{Kind: repo.NodeActionKindCapability, CapabilityID: capabilityID, ScopeExpr: scopeExpr, Actions: actions}
}

func actNode(actionID string, on string, executor string, executorRole string, skill string, inline string) repo.NodeActionDef {
	return repo.NodeActionDef{Kind: repo.NodeActionKindNode, ActionID: actionID, On: on, Executor: executor, ExecutorRole: executorRole, Skill: skill, Inline: inline}
}

func out(productType string, label string, kind string) repo.ExpectedOutput {
	return repo.ExpectedOutput{ProductType: productType, Label: label, Kind: kind}
}

func node(flow, tnid string, order int64, key, nodeType string, gate, userGate int64, scope string, reviewers []repo.ReviewerDef, actions []repo.NodeActionDef, outputs []repo.ExpectedOutput) repo.FlowDefNode {
	return repo.FlowDefNode{
		FlowDefsID:      flow,
		TemplateNodeID:  tnid,
		NodeOrder:       order,
		NodeKey:         key,
		NodeType:        nodeType,
		Gate:            gate,
		UserGate:        userGate,
		ScopeExpr:       scope,
		Reviewers:       j(reviewers),
		NodeActions:     j(actions),
		ExpectedOutputs: j(outputs),
	}
}

// ---- business 13 节点（business-flow-derivation.md v3.1） ----

func businessGolden() []repo.FlowDefNode {
	opening := actNode("align.opening", "entry", repo.ExecutorProgram, "pm", "", "渲染节点开场白并等待用户指令（内置强制等待）")
	toBRD := actNode("to-brd", "exec", repo.ExecutorSubagent, "pm", "to-brd", "")
	devOut := func(devLabel, mrLabel string) []repo.ExpectedOutput {
		return []repo.ExpectedOutput{out("dev_notes", devLabel, "document"), out("code_mr", mrLabel, "link")}
	}
	qaOut := []repo.ExpectedOutput{out("test_result", "《测试用例与验收记录》", "document"), out("defect", "《缺陷文档》", "document")}
	return []repo.FlowDefNode{
		node("business", "bus-00", 1, "待启动", repo.NodeTypeGate, 0, 1, "", nil, nil, nil),
		node("business", "bus-01", 2, "需求对齐", repo.NodeTypeAlign, 0, 1, "", nil,
			[]repo.NodeActionDef{opening, toBRD}, []repo.ExpectedOutput{out("brd", "《业务需求文档》", "document")}),
		node("business", "bus-02", 3, "产品设计", repo.NodeTypeProduce, 0, 0, "", nil,
			[]repo.NodeActionDef{actCap("product-design", "produce", "revise")},
			[]repo.ExpectedOutput{out("product_doc", "《产品设计文档》", "document")}),
		node("business", "bus-03", 4, "产品设计评审", repo.NodeTypeReview, 1, 1, "",
			[]repo.ReviewerDef{rev("arch", "arch.prd-review"), rev("rd", "rd.prd-review", "rd|full"), rev("fe", "fe.prd-review", "full")},
			[]repo.NodeActionDef{actCap("product-design", "clarify")}, nil),
		node("business", "bus-04", 5, "RD 技术设计", repo.NodeTypeProduce, 0, 0, "rd|full", nil,
			[]repo.NodeActionDef{actCap("backend-design", "produce", "revise")},
			[]repo.ExpectedOutput{out("rd_design", "《RD 技术设计》", "document")}),
		node("business", "bus-05", 6, "RD 技术评审", repo.NodeTypeReview, 1, 1, "rd|full",
			[]repo.ReviewerDef{rev("arch", "arch.rd-design-review"), rev("fe", "fe.rd-design-review", "full"), rev("qa", "qa.rd-design-review"), rev("pm", "pm.rd-design-review")},
			[]repo.NodeActionDef{actCap("backend-design", "clarify")}, nil),
		node("business", "bus-06", 7, "FE 技术设计", repo.NodeTypeProduce, 0, 0, "fe|full", nil,
			[]repo.NodeActionDef{actCap("frontend-design", "produce", "revise")},
			[]repo.ExpectedOutput{out("fe_design", "《FE 技术设计》", "document")}),
		node("business", "bus-07", 8, "FE 设计评审", repo.NodeTypeReview, 1, 0, "fe|full",
			[]repo.ReviewerDef{rev("qa", "qa.fe-design-review")},
			[]repo.NodeActionDef{actCap("frontend-design", "clarify")}, nil),
		node("business", "bus-08", 9, "RD 开发", repo.NodeTypeProduce, 1, 0, "rd|full",
			[]repo.ReviewerDef{rev("cr", "code-review")},
			[]repo.NodeActionDef{actCap("backend-dev", "dev", "diagnosing-bugs", "fix_on_review")},
			devOut("《RD 开发说明》", "《RD 代码 MR》")),
		node("business", "bus-09", 10, "FE 开发", repo.NodeTypeProduce, 1, 0, "fe|full",
			[]repo.ReviewerDef{rev("cr", "code-review")},
			[]repo.NodeActionDef{actCap("frontend-dev", "dev", "diagnosing-bugs", "fix_on_review")},
			devOut("《FE 开发说明》", "《FE 代码 MR》")),
		node("business", "bus-10", 11, "QA 验收（含联调）", repo.NodeTypeProduce, 1, 0, "full", nil,
			[]repo.NodeActionDef{actCap("acceptance-test", "accept", "retest"), actCap("backend-dev", "clarify"), actCap("frontend-dev", "clarify")},
			qaOut),
		node("business", "bus-11", 12, "用户验收", repo.NodeTypeGate, 1, 1, "", nil, nil, nil),
		node("business", "bus-12", 13, "完成", repo.NodeTypeGate, 0, 0, "", nil, nil, nil),
	}
}

// ---- tech 11 节点（flow-derivation-tech-task-plan.md §1） ----

func techGolden() []repo.FlowDefNode {
	opening := actNode("align.opening", "entry", repo.ExecutorProgram, "pm", "", "渲染节点开场白并等待用户指令（内置强制等待）")
	toBRD := actNode("to-brd", "exec", repo.ExecutorSubagent, "pm", "to-brd", "")
	qaOut := []repo.ExpectedOutput{out("test_result", "《测试用例与验收记录》", "document"), out("defect", "《缺陷文档》", "document")}
	return []repo.FlowDefNode{
		node("tech", "tec-b0", 1, "待启动", repo.NodeTypeGate, 0, 1, "", nil, nil, nil),
		node("tech", "tec-b1", 2, "需求对齐", repo.NodeTypeAlign, 0, 1, "", nil,
			[]repo.NodeActionDef{opening, toBRD}, []repo.ExpectedOutput{out("brd", "《业务需求文档》", "document")}),
		node("tech", "tec-b2", 3, "RD 技术设计", repo.NodeTypeProduce, 0, 0, "rd|full", nil,
			[]repo.NodeActionDef{actCap("backend-design", "produce", "revise")},
			[]repo.ExpectedOutput{out("rd_design", "《RD 技术设计》", "document")}),
		node("tech", "tec-b3", 4, "RD 技术评审（按需）", repo.NodeTypeReview, 1, 1, "rd|full",
			[]repo.ReviewerDef{rev("arch", "arch.rd-design-review"), rev("fe", "fe.rd-design-review", "full"), rev("qa", "qa.rd-design-review")},
			[]repo.NodeActionDef{actCap("backend-design", "clarify")}, nil),
		node("tech", "tec-b4", 5, "FE 技术设计（按需）", repo.NodeTypeProduce, 0, 0, "fe|full", nil,
			[]repo.NodeActionDef{actCap("frontend-design", "produce", "revise")},
			[]repo.ExpectedOutput{out("fe_design", "《FE 技术设计》", "document")}),
		node("tech", "tec-b5", 6, "FE 技术评审（按需）", repo.NodeTypeReview, 1, 0, "fe|full",
			[]repo.ReviewerDef{rev("qa", "qa.fe-design-review")},
			[]repo.NodeActionDef{actCap("frontend-design", "clarify")}, nil),
		node("tech", "tec-b8a", 7, "RD 开发", repo.NodeTypeProduce, 1, 0, "rd|full",
			[]repo.ReviewerDef{rev("cr", "code-review")},
			[]repo.NodeActionDef{actCap("backend-dev", "dev", "diagnosing-bugs", "fix_on_review")},
			[]repo.ExpectedOutput{out("dev_notes", "《RD 开发说明》", "document"), out("code_mr", "《RD 代码 MR》", "link")}),
		node("tech", "tec-b8b", 8, "FE 开发", repo.NodeTypeProduce, 1, 0, "fe|full",
			[]repo.ReviewerDef{rev("cr", "code-review")},
			[]repo.NodeActionDef{actCap("frontend-dev", "dev", "diagnosing-bugs", "fix_on_review")},
			[]repo.ExpectedOutput{out("dev_notes", "《FE 开发说明》", "document"), out("code_mr", "《FE 代码 MR》", "link")}),
		node("tech", "tec-b9", 9, "QA 验收（含联调）", repo.NodeTypeProduce, 1, 0, "full", nil,
			[]repo.NodeActionDef{actCap("acceptance-test", "accept", "retest"), actCap("backend-dev", "clarify"), actCap("frontend-dev", "clarify")},
			qaOut),
		node("tech", "tec-b10", 10, "用户验收", repo.NodeTypeGate, 1, 1, "", nil, nil, nil),
		node("tech", "tec-b11", 11, "完成", repo.NodeTypeGate, 0, 0, "", nil, nil, nil),
	}
}

// ---- task 7 节点（无 QA；C5 已删） ----

func taskGolden() []repo.FlowDefNode {
	return []repo.FlowDefNode{
		node("task", "tsk-c0", 1, "待启动", repo.NodeTypeGate, 0, 1, "", nil, nil, nil),
		node("task", "tsk-c1", 2, "需求对齐", repo.NodeTypeAlign, 0, 1, "", nil,
			[]repo.NodeActionDef{actNode("align.opening", "entry", repo.ExecutorProgram, "executor", "", "渲染节点开场白并等待用户指令（内置强制等待）")}, nil),
		node("task", "tsk-c2", 3, "技术设计", repo.NodeTypeProduce, 0, 0, "rd|fe", nil,
			[]repo.NodeActionDef{actCapScope("backend-design", "rd", "produce", "revise"), actCapScope("frontend-design", "fe", "produce", "revise")},
			[]repo.ExpectedOutput{out("rd_design", "《RD 技术设计》", "document"), out("fe_design", "《FE 技术设计》", "document")}),
		node("task", "tsk-c3", 4, "技术评审", repo.NodeTypeReview, 1, 1, "rd|fe",
			[]repo.ReviewerDef{rev("arch", "arch.rd-design-review", "rd"), rev("arch", "arch.fe-design-review", "fe")},
			[]repo.NodeActionDef{actCapScope("backend-design", "rd", "clarify"), actCapScope("frontend-design", "fe", "clarify")}, nil),
		node("task", "tsk-c4", 5, "开发", repo.NodeTypeProduce, 1, 0, "rd|fe",
			[]repo.ReviewerDef{rev("cr", "code-review")},
			[]repo.NodeActionDef{actCapScope("backend-dev", "rd", "dev", "diagnosing-bugs", "fix_on_review"), actCapScope("frontend-dev", "fe", "dev", "diagnosing-bugs", "fix_on_review")},
			[]repo.ExpectedOutput{out("dev_notes", "《开发说明》", "document"), out("code_mr", "《代码 MR》", "link")}),
		node("task", "tsk-c6", 6, "用户验收", repo.NodeTypeGate, 1, 1, "", nil, nil, nil),
		node("task", "tsk-c7", 7, "完成", repo.NodeTypeGate, 0, 0, "", nil, nil, nil),
	}
}

// ---- plan 9 节点（无 scope） ----

func planGolden() []repo.FlowDefNode {
	return []repo.FlowDefNode{
		node("plan", "pln-d0", 1, "待启动", repo.NodeTypeGate, 0, 1, "", nil, nil, nil),
		node("plan", "pln-d1", 2, "立项需求对齐", repo.NodeTypeAlign, 0, 1, "", nil,
			[]repo.NodeActionDef{
				actNode("align.opening", "entry", repo.ExecutorProgram, "arch", "", "渲染节点开场白并等待用户指令（内置强制等待）"),
				actNode("to-alignment", "exec", repo.ExecutorSubagent, "arch", "to-alignment", ""),
			},
			[]repo.ExpectedOutput{out("alignment", "《需求对齐文档》", "document")}),
		node("plan", "pln-d2", 3, "立项评审", repo.NodeTypeGate, 1, 1, "", nil, nil, nil),
		node("plan", "pln-d3", 4, "调研", repo.NodeTypeProduce, 0, 0, "", nil,
			[]repo.NodeActionDef{actCap("research-cap", "produce", "revise")},
			[]repo.ExpectedOutput{out("research", "《调研文档》", "document")}),
		node("plan", "pln-d4", 5, "调研评审", repo.NodeTypeReview, 1, 0, "",
			[]repo.ReviewerDef{rev("arch", "arch.research-review")},
			[]repo.NodeActionDef{actCap("research-cap", "clarify")}, nil),
		node("plan", "pln-d5", 6, "架构规划", repo.NodeTypeProduce, 0, 0, "", nil,
			[]repo.NodeActionDef{actCap("arch-cap", "produce", "revise")},
			[]repo.ExpectedOutput{out("architecture", "《架构规划》", "document")}),
		node("plan", "pln-d6", 7, "架构规划评审", repo.NodeTypeReview, 1, 1, "",
			[]repo.ReviewerDef{rev("pm", "pm.architecture-review")},
			[]repo.NodeActionDef{actCap("arch-cap", "clarify")}, nil),
		node("plan", "pln-d7", 8, "创建需求·启动版本", repo.NodeTypeProduce, 0, 0, "", nil,
			[]repo.NodeActionDef{actCap("decide-cap", "create_reqs")}, nil),
		node("plan", "pln-d8", 9, "完成", repo.NodeTypeGate, 0, 0, "", nil, nil, nil),
	}
}
