// T12 domain 裁决链纯逻辑单测：仲裁结论词表 × replies 不可变追加（裁决链通道
// 续号）× 上升/裁决条目定位（issue↔session 关联载体）× 通知文案装配。
package domain

import (
	"strings"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/huzilin/wbflow/rd/repo"
)

// TestUnit_AdjudicationVerdict 仲裁结论枚举：uphold/override 合法，其余（含
// approved/rejected 拍板词表）越集——拍板是节点级门禁结论，仲裁是 issue 级冲突
// 结论，词表互不借用。
func TestUnit_AdjudicationVerdict(t *testing.T) {
	assert.Assert(t, ValidAdjudicationVerdict(AdjudicationVerdictUphold))
	assert.Assert(t, ValidAdjudicationVerdict(AdjudicationVerdictOverride))
	for _, verdict := range []string{"", "approved", "rejected", "agree", "fix", "UPHOLD"} {
		assert.Assert(t, !ValidAdjudicationVerdict(verdict), "verdict %q 应越集", verdict)
	}
}

// TestUnit_AppendIssueReply 不可变追加：空线程起号 r-1、续号线内自增、
// 原 JSON 文本不变形（不可变追加语义）、escalation/adjudication 条目携带
// verdict/sessionId。
func TestUnit_AppendIssueReply(t *testing.T) {
	text, replyID, err := AppendIssueReply("[]", repo.IssueReply{By: "arch", RoleID: "arch", Text: "上升", Kind: ReplyKindEscalation, SessionID: "ad-1"})
	assert.NilError(t, err)
	assert.Equal(t, replyID, "r-1")
	assert.Assert(t, contains(text, `"kind":"escalation"`) && contains(text, `"sessionId":"ad-1"`), "上升条目字段: %s", text)

	text2, replyID2, err := AppendIssueReply(text, repo.IssueReply{By: "user", RoleID: "user", Text: "裁决", Kind: ReplyKindAdjudication, Verdict: AdjudicationVerdictOverride, SessionID: "ad-1"})
	assert.NilError(t, err)
	assert.Equal(t, replyID2, "r-2")
	assert.Assert(t, contains(text2, `"verdict":"override"`), "裁决条目 verdict: %s", text2)
	assert.Assert(t, contains(text2, `"text":"上升"`) && contains(text2, `"text":"裁决"`), "既有条目保留: %s", text2)

	// 普通 reply 条目（无 kind）round-trip 不引入多余键
	text3, _, err := AppendIssueReply(text2, repo.IssueReply{By: "pm", RoleID: "pm", Text: "回复"})
	assert.NilError(t, err)
	assert.Assert(t, !contains(text3, `"kind":""`) && !contains(text3, `"verdict":""`), "空 omitempty 键不落: %s", text3)

	// 非法 JSON 拒绝（实例数据损坏）
	_, _, err = AppendIssueReply("{broken", repo.IssueReply{})
	assert.Assert(t, err != nil, "非法 replies JSON 应报错")
}

// TestUnit_LatestReplyOfKind 条目定位：命中末条即最新、混入普通回复不干扰、
// 未命中 ok=false、非法 JSON 报错。
func TestUnit_LatestReplyOfKind(t *testing.T) {
	replies := "[]"
	text, _, err := AppendIssueReply(replies, repo.IssueReply{By: "a", Text: "普通回复"})
	assert.NilError(t, err)
	text, _, err = AppendIssueReply(text, repo.IssueReply{By: "a", Text: "上升一", Kind: ReplyKindEscalation, SessionID: "ad-1"})
	assert.NilError(t, err)
	text, _, err = AppendIssueReply(text, repo.IssueReply{By: "b", Text: "普通回复"})
	assert.NilError(t, err)
	text, _, err = AppendIssueReply(text, repo.IssueReply{By: "a", Text: "上升二", Kind: ReplyKindEscalation, SessionID: "ad-2"})
	assert.NilError(t, err)

	entry, ok, err := LatestReplyOfKind(text, ReplyKindEscalation)
	assert.NilError(t, err)
	assert.Assert(t, ok)
	assert.Equal(t, entry.SessionID, "ad-2", "末条即最新")
	assert.Equal(t, entry.Text, "上升二")

	_, ok, err = LatestReplyOfKind(text, ReplyKindAdjudication)
	assert.NilError(t, err)
	assert.Assert(t, !ok, "未命中 ok=false")

	_, ok, err = LatestReplyOfKind("{broken", ReplyKindEscalation)
	assert.Assert(t, err != nil && !ok, "非法 JSON 报错")
}

// TestUnit_EscalationNoticeText 通知文案装配：源 issue id 与触发来源进文案
// （T11 收件箱读侧直接呈现）。
func TestUnit_EscalationNoticeText(t *testing.T) {
	auto := BuildEscalationNoticeText("cm-1", AutoEscalateTriggerText(5))
	assert.Assert(t, contains(auto, "cm-1") && contains(auto, "reopen 达阈值 5"), "自动触发文案: %s", auto)
	manual := BuildEscalationNoticeText("cm-2", ManualEscalateTriggerText("arch"))
	assert.Assert(t, contains(manual, "cm-2") && contains(manual, "operator=arch"), "手动触发文案: %s", manual)
	assert.Equal(t, EscalationReplyText(""), "冲突不收敛，上升 User 仲裁", "reason 空=默认文案")
	assert.Equal(t, EscalationReplyText("双方僵持"), "双方僵持", "reason 非空直用")
}

// contains strings.Contains 别名（断言可读性）。
func contains(s string, substr string) bool {
	return strings.Contains(s, substr)
}
