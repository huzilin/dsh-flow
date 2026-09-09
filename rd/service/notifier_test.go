// T11 通知收敛写入口单测（无 IO）：Notify 行装配（七类枚举/收件角色/跳转定位
// 映射/派发者口径）+ 入参校验拦截 + Dispatch 故障注入（落库失败吞错记日志、
// 不 panic 不上抛——「通知写入失败不阻断业务路径」AC 的单元侧验证）。
package service

import (
	"context"
	"errors"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/huzilin/wbflow/rd/repo"
)

// fakeStore 通知落库通道替身（记录调用与行装配；可注入失败）。
type fakeStore struct {
	// issueID 取号返回值
	issueID string
	// insertErr 注入 InsertNotification 失败
	insertErr error
	// allocErr 注入取号失败
	allocErr error
	// captured 捕获的落库行
	captured *repo.Issue
	// calls InsertNotification 调用次数
	calls int
}

func (f *fakeStore) AllocIssueID(ctx context.Context) (string, error) {
	if f.allocErr != nil {
		return "", f.allocErr
	}
	return f.issueID, nil
}

func (f *fakeStore) InsertNotification(ctx context.Context, row *repo.Issue) error {
	f.calls++
	if f.insertErr != nil {
		return f.insertErr
	}
	f.captured = row
	return nil
}

// TestUnit_NotifierNotifyValidation 入参校验：类别越集/缺需求/缺收件人拦截。
func TestUnit_NotifierNotifyValidation(t *testing.T) {
	n := &Notifier{store: &fakeStore{}}
	for _, tc := range []struct {
		name string
		spec NotificationSpec
	}{
		{"类别越集", NotificationSpec{Kind: "no_such", RequirementID: "req-001", RecipientRole: "pm"}},
		{"缺需求", NotificationSpec{Kind: NotificationKindGatePending, RecipientRole: "pm"}},
		{"缺收件人", NotificationSpec{Kind: NotificationKindGatePending, RequirementID: "req-001"}},
	} {
		_, err := n.Notify(context.Background(), tc.spec)
		assert.Assert(t, err != nil, "%s 应拦截", tc.name)
	}
}

// TestUnit_NotifierNotifyAssembly 行装配：七类载体字段与跳转定位映射
// （anchor_kind=类别、target_type=node 未传锚点时空 target、task/issue 类落
// 目标 id、派发者缺省 plugin、state=未读、issue_type=notification）。
func TestUnit_NotifierNotifyAssembly(t *testing.T) {
	// node 类：gate_pending——未传锚点，target_id 列位空（定位取 node_id 列）
	store := &fakeStore{issueID: "cm-100"}
	n := &Notifier{store: store}
	row, err := n.Notify(context.Background(), NotificationSpec{
		Kind:          NotificationKindGatePending,
		RequirementID: "req-001",
		NodeID:        "n-04",
		RecipientRole: "pm",
	})
	assert.NilError(t, err)
	assert.Equal(t, row.IssueID, "cm-100")
	assert.Equal(t, store.captured.IssueType, "notification", "载体=issues notification 分区")
	assert.Equal(t, store.captured.State, repo.NotificationStateUnread, "建行未读")
	assert.Equal(t, store.captured.AnchorKind, NotificationKindGatePending, "类别落 anchor_kind 复用列")
	assert.Assert(t, store.captured.AnchorArtifactID == nil, "node 类 target_id 置空")
	assert.Assert(t, store.captured.Assignee != nil, "收件角色落 assignee")
	assert.Equal(t, *store.captured.Assignee, "pm")
	assert.Equal(t, store.captured.Author, "plugin", "派发者缺省 plugin")
	assert.Equal(t, store.captured.RoleID, "plugin", "派发角色缺省 plugin")
	assert.Equal(t, store.captured.Replies, "[]")

	// node 类显式锚点（注记-D1：re_review retest 子型传 issueId）——保留不置空
	storeRt := &fakeStore{issueID: "cm-102"}
	_, err = (&Notifier{store: storeRt}).Notify(context.Background(), NotificationSpec{
		Kind:          NotificationKindReReview,
		RequirementID: "req-001",
		NodeID:        "n-04",
		TargetID:      "cm-101",
		RecipientRole: "arch",
	})
	assert.NilError(t, err)
	assert.Assert(t, storeRt.captured.AnchorArtifactID != nil, "显式锚点保留（retest 子型）")
	assert.Equal(t, *storeRt.captured.AnchorArtifactID, "cm-101")

	// task 类：悬置催办——target id 落 anchor_artifact_id
	store2 := &fakeStore{issueID: "cm-101"}
	row2, err := (&Notifier{store: store2}).Notify(context.Background(), NotificationSpec{
		Kind:          NotificationKindStallFollowup,
		RequirementID: "req-001",
		NodeID:        "n-04",
		TargetID:      "t-1730000000000-001",
		RecipientRole: "rd",
		Sender:        "scanner",
		SenderRole:    "plugin",
	})
	assert.NilError(t, err)
	assert.Assert(t, row2.AnchorArtifactID != nil, "task 类目标 id 落 anchor 列")
	assert.Equal(t, *row2.AnchorArtifactID, "t-1730000000000-001")
	assert.Equal(t, row2.Author, "scanner")

	// 会话类映射 session（target_type 恒确定性映射，无显式覆写口）
	store3 := &fakeStore{issueID: "cm-102"}
	_, err = (&Notifier{store: store3}).Notify(context.Background(), NotificationSpec{
		Kind:          NotificationKindSessionMessage,
		RequirementID: "req-001",
		RecipientRole: "pm",
	})
	assert.NilError(t, err)
	assert.Equal(t, notificationTargetTypes[NotificationKindSessionMessage], "session", "会话类映射 session")
}

// TestUnit_DispatchSwallowsNotifierFailure 故障注入：落库失败 Dispatch 吞错记日志，
// 不 panic 不上抛、业务路径无感（AC：通知写入失败不阻断业务路径）。
func TestUnit_DispatchSwallowsNotifierFailure(t *testing.T) {
	store := &fakeStore{issueID: "cm-1", insertErr: errors.New("db down（注入）")}
	n := &Notifier{store: store}
	assert.Assert(t, n.store != nil)
	Dispatch(n, NotificationSpec{
		Kind:          NotificationKindGatePending,
		RequirementID: "req-001",
		NodeID:        "n-04",
		RecipientRole: "pm",
	})
	assert.Equal(t, store.calls, 1, "失败前已尝试落库一次")
	assert.Assert(t, store.captured == nil, "失败行不落地")

	// nil notifier 兜底：不 panic
	Dispatch(nil, NotificationSpec{Kind: NotificationKindGatePending})
}
