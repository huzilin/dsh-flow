// Package domain 本文件：collect 聚合判定 + 门禁公式 + 结论协议装配（T8 编排核心，
// plugin 内置逻辑的纯逻辑投影）。无 IO，可单测。
//
// 定稿出处（docs-tmp/capability-model-final.md）：
//   - §1.2a collect 三段口径：①收票——是否全表态完成；②判定——全 agree → approved，
//     有 unagree → owner 澄清循环；③再收票——澄清后重新等待表态。
//     判定顺序=先齐后判：未齐（任一评审位无当前轮表态）恒 pending，即使已有个别 unagree；
//   - §1.2 per-agent 三态：reviewing（评审+表态进行中）/ agree / unagree；
//   - §1.3 裁决规则：user_gate=1 唤醒 User 终审（approved 迁移下一节点）、user_gate=0
//     机械 approved；节点内 CR 异化永不 rejected（打回链路 T9 消费）；
//   - §6 推进契约 rule4 空转规则：相位无挂载 → plugin 直接轮转（视为通过）；
//   - last_decision 结论协议（migration 0003 / arch_design v0.29）：
//     {"verdict":"approved|rejected","reason":"…","at":毫秒,"branch":"…"}。
package domain

import (
	perr "github.com/pingcap/errors"

	"github.com/huzilin/wbflow/rd/pkg/utils"
	"github.com/huzilin/wbflow/rd/repo"
)

// collect 判定结果（service 层据此装配节点相位迁移与门禁落定；
// per-agent reviewState 不含 approved/rejected——§1.2a「它们是 collect 的输出」）。
type CollectOutcome string

// collect 判定结果三值。
const (
	// CollectPending 未齐：当前轮仍有评审位未表态（reviewing），继续收票。
	CollectPending CollectOutcome = "pending"
	// CollectApproved 全表态且全 agree：门禁通过（user_gate 分流由 service 层装配）。
	CollectApproved CollectOutcome = "approved"
	// CollectClarify 全表态且存在 unagree：owner 澄清循环（节点进 respond 相位）。
	CollectClarify CollectOutcome = "clarify"
)

// 门禁裁决 verdict（flow_nodes.last_decision JSON 与 wb_decision 入参共用词表）。
const (
	GateVerdictApproved = "approved"
	GateVerdictRejected = "rejected"
)

// 拍板卡门禁侧状态（ui-interaction-final §1.3 三态中的后两态；
// 进度态=CollectPending/Clarify，不入本词表）。
const (
	GateStateAgreed   = "agreed"
	GateStateApproved = "approved"
)

// GateDecision flow_nodes.last_decision JSON 结构（结论协议单一口径，
// migration 0003 列注释同源）。
type GateDecision struct {
	// Verdict 裁决结论：approved | rejected
	Verdict string `json:"verdict"`
	// Reason 裁决理由（collect 汇总 / User 打回理由）
	Reason string `json:"reason"`
	// At 裁决时间，unix 毫秒
	At int64 `json:"at"`
	// Branch 评审敲定分支名；空=无
	Branch string `json:"branch,omitempty"`
}

// CurrentRound 当前评审轮：条目最大 round（review_results 空数组=轮 0）。
// round 口径与评审轮绑定——节点首轮 0，澄清后重入新轮 +1；
// 打回重入（T9）复用同一自增语义，base 对齐 rolled_back_count 由 T9 装配。
func CurrentRound(entries []repo.ReviewResult) int64 {
	current := int64(0)
	for _, entry := range entries {
		if entry.Round > current {
			current = entry.Round
		}
	}
	return current
}

// CurrentRoundEntries 当前轮条目过滤（collect 判定只看当前轮——
// 历史轮条目仅留痕，不参与判定与聚合计数）。
func CurrentRoundEntries(entries []repo.ReviewResult) []repo.ReviewResult {
	current := CurrentRound(entries)
	kept := make([]repo.ReviewResult, 0, len(entries))
	for _, entry := range entries {
		if entry.Round == current {
			kept = append(kept, entry)
		}
	}
	return kept
}

// UpsertReviewResult 同轮同角色覆写取最后（spec §2 幂等行）：命中 (role, round)
// 既有条目原地覆写 state/capabilityId/at；未命中追加（reviewers 内位置唯一，
// role+round 即天然幂等键）。
func UpsertReviewResult(entries []repo.ReviewResult, entry repo.ReviewResult) []repo.ReviewResult {
	for i := range entries {
		if entries[i].Role == entry.Role && entries[i].Round == entry.Round {
			entries[i].CapabilityID = entry.CapabilityID
			entries[i].State = entry.State
			entries[i].At = entry.At
			return entries
		}
	}
	return append(entries, entry)
}

// InitReviewRound 评审轮初始化：全部评审位置初始化为 reviewing（scope 裁剪后的
// 实例评审位声明为位置全集）。消费方：节点激活进 review 相位（首轮）。
//
// 错误集：reviewers JSON 非法（实例数据损坏）。
func InitReviewRound(reviewersJSON string, round int64, at int64) ([]repo.ReviewResult, error) {
	reviewers, err := ParseReviewers(reviewersJSON)
	if err != nil {
		return nil, err
	}
	entries := make([]repo.ReviewResult, 0, len(reviewers))
	for _, reviewer := range reviewers {
		entries = append(entries, repo.ReviewResult{
			Role:         reviewer.Role,
			CapabilityID: reviewer.CapabilityID,
			State:        repo.ReviewStateReviewing,
			Round:        round,
			At:           at,
		})
	}
	return entries, nil
}

// ReenterRoundEntries 澄清后重入新轮（§1.2 内置机制——owner 澄清后对应 reviewer
// 回 reviewing 重新表态，其余评审位保持已表态结论不重复打扰）：
// 全部位升轮（round+1 语义由调用方传入目标轮），agree 条目携结论进新轮，
// unagree/reviewing/缺位复位 reviewing——collect 三段③再收票只等未收敛位。
// 重入以全量替换语义落库（字段恒承载当前轮快照，历史轮留痕在审计 old_value）。
//
// 错误集：reviewers/review_results JSON 非法（实例数据损坏）。
func ReenterRoundEntries(reviewersJSON string, reviewResultsJSON string, round int64, at int64) ([]repo.ReviewResult, error) {
	reviewers, err := ParseReviewers(reviewersJSON)
	if err != nil {
		return nil, err
	}
	var previous []repo.ReviewResult
	if err := utils.UnmarshalJSONText(reviewResultsJSON, &previous); err != nil {
		return nil, perr.Trace(err)
	}
	conclusion := make(map[string]repo.ReviewResult, len(previous))
	for _, entry := range previous {
		if entry.Round != CurrentRound(previous) {
			continue
		}
		if _, seen := conclusion[entry.Role]; !seen || entry.At >= conclusion[entry.Role].At {
			conclusion[entry.Role] = entry
		}
	}
	entries := make([]repo.ReviewResult, 0, len(reviewers))
	for _, reviewer := range reviewers {
		state := repo.ReviewStateReviewing
		capabilityID := reviewer.CapabilityID
		if prior, ok := conclusion[reviewer.Role]; ok && prior.State == repo.ReviewStateAgree {
			state = repo.ReviewStateAgree
			capabilityID = prior.CapabilityID
		}
		entries = append(entries, repo.ReviewResult{
			Role:         reviewer.Role,
			CapabilityID: capabilityID,
			State:        state,
			Round:        round,
			At:           at,
		})
	}
	return entries, nil
}

// ParseReviewers 实例评审位声明解析（reviewers JSON → 有序位置集）。
//
// 错误集：reviewers JSON 非法（实例数据损坏）。
func ParseReviewers(reviewersJSON string) ([]repo.ReviewerDef, error) {
	reviewers := make([]repo.ReviewerDef, 0)
	if err := utils.UnmarshalJSONText(reviewersJSON, &reviewers); err != nil {
		return nil, perr.Trace(err)
	}
	return reviewers, nil
}

// CollectVerdict collect 聚合判定（§1.2a 三段：先齐后判）：
//
//   - 评审位为空（无评审位节点）→ approved（空转规则：无挂载相位直过，§6 rule4）；
//   - 当前轮未齐（任一位置无条目或 state=reviewing）→ pending；
//   - 全 agree → approved；
//   - 存在 unagree → clarify（owner 澄清循环；确认真问题 to-fix 的 rejected 打回
//     由 T9 消费，节点内 CR 永不 rejected）。
//
// 入参：reviewersJSON=实例评审位声明（scope 裁剪后）；reviewResultsJSON=该节点
// review_results 全量 JSON（多轮共存时按 CurrentRound 过滤）。
//
// 错误集：reviewers/review_results JSON 非法（实例数据损坏）。
func CollectVerdict(reviewersJSON string, reviewResultsJSON string) (CollectOutcome, error) {
	reviewers, err := ParseReviewers(reviewersJSON)
	if err != nil {
		return "", err
	}
	if len(reviewers) == 0 {
		return CollectApproved, nil
	}
	var entries []repo.ReviewResult
	if err := utils.UnmarshalJSONText(reviewResultsJSON, &entries); err != nil {
		return "", perr.Trace(err)
	}
	return collectVerdict(reviewers, CurrentRoundEntries(entries)), nil
}

// collectVerdict 纯判定核（位置集 × 当前轮条目 → 三值；表驱动单测锚点）。
func collectVerdict(reviewers []repo.ReviewerDef, current []repo.ReviewResult) CollectOutcome {
	stated := make(map[string]string, len(current))
	for _, entry := range current {
		if entry.State == repo.ReviewStateAgree || entry.State == repo.ReviewStateUnagree {
			stated[entry.Role] = entry.State
		}
	}
	if len(stated) < len(reviewers) {
		return CollectPending
	}
	for _, state := range stated {
		if state == repo.ReviewStateUnagree {
			return CollectClarify
		}
	}
	return CollectApproved
}

// ActivationPhase 节点激活相位（门禁通过后下一节点的 state 落值口径，
// §2 节点类型实例化 + §6 rule4 空转规则）：
//
//   - gate（门禁类）→ entry（迎宾呈现 + user_gate 拍板卡驻留）；
//   - review（审批类）→ review（产出动作位即评审，exec 不配置；评审位经
//     reviewers 声明挂载）；
//   - produce / align → entry 有挂载动作用 entry，否则空转直过 exec。
//
// 错误集：node_actions JSON 非法（实例数据损坏）。
func ActivationPhase(nodeType string, nodeActionsJSON string) (string, error) {
	switch nodeType {
	case repo.NodeTypeGate:
		return repo.PhaseEntry, nil
	case repo.NodeTypeReview:
		return repo.PhaseReview, nil
	default:
		mounted, err := phaseMounted(repo.PhaseEntry, nodeActionsJSON)
		if err != nil {
			return "", err
		}
		if mounted {
			return repo.PhaseEntry, nil
		}
		return repo.PhaseExec, nil
	}
}

// phaseMounted 相位挂载判定（entry 专用）：kind=node 条目 on 命中该相位。
// entry 挂载=开场白 node action（交互型专属，capability-model-final §1.1）；
// capability 动作只挂 exec/review/respond，不参与 entry 判定。评审相位挂载经
// reviewers 声明，不入本判定——ActivationPhase 对 review 型节点直返 review。
func phaseMounted(phase string, nodeActionsJSON string) (bool, error) {
	var entries []repo.NodeActionDef
	if err := utils.UnmarshalJSONText(nodeActionsJSON, &entries); err != nil {
		return false, perr.Trace(err)
	}
	for _, entry := range entries {
		if entry.Kind == repo.NodeActionKindNode && entry.On == phase {
			return true, nil
		}
	}
	return false, nil
}

// BuildGateDecision 门禁裁决结论协议装配（last_decision JSON 文本，
// migration 0003 列注释 schema：{verdict,reason,at,branch?}——branch 空=省略键）。
func BuildGateDecision(verdict string, reason string, at int64, branch string) (string, error) {
	return utils.MarshalJSONText(GateDecision{
		Verdict: verdict,
		Reason:  reason,
		At:      at,
		Branch:  branch,
	})
}

// ParseGateDecision last_decision JSON 解析（NULL/空串=未判定，返回零值）。
//
// 错误集：JSON 非法（实例数据损坏）。
func ParseGateDecision(lastDecision string) (GateDecision, error) {
	decision := GateDecision{}
	if lastDecision == "" {
		return decision, nil
	}
	if err := utils.UnmarshalJSONText(lastDecision, &decision); err != nil {
		return GateDecision{}, perr.Trace(err)
	}
	return decision, nil
}
