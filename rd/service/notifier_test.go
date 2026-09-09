// T11 通知收敛写入口单测（无 IO）：Notify 行装配（七类枚举/收件角色/跳转定位
// 映射/派发者口径）+ 入参校验拦截 + outbox 可靠投递（enqueue 校验/即时推送/
// retry 耗尽 dead——spec §1.2 注记-H 的单元侧验证）。
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

// fakeOutbox outbox 通道替身（内存行集；记录状态迁移与告警调用）。
type fakeOutbox struct {
	rows   map[int64]*repo.Outbox
	alerts []int64
	seq    int64
}

func newFakeOutbox(rows ...*repo.Outbox) *fakeOutbox {
	f := &fakeOutbox{rows: map[int64]*repo.Outbox{}}
	for _, r := range rows {
		f.seq++
		r.ID = f.seq
		f.rows[r.ID] = r
	}
	return f
}

func (f *fakeOutbox) ListPendingOutbox(ctx context.Context, limit int) ([]*repo.Outbox, error) {
	out := []*repo.Outbox{}
	for _, r := range f.rows {
		if r.Status == repo.OutboxStatusPending {
			out = append(out, r)
		}
	}
	return out, nil
}

func (f *fakeOutbox) ListRetryableOutbox(ctx context.Context, staleBeforeMs int64, limit int) ([]*repo.Outbox, error) {
	out := []*repo.Outbox{}
	for _, r := range f.rows {
		switch {
		case r.Status == repo.OutboxStatusFailed && r.RetryCount < repo.OutboxRetryLimit:
			out = append(out, r)
		case r.Status == repo.OutboxStatusPending && r.UpdatedAt <= staleBeforeMs:
			out = append(out, r)
		case r.Status == repo.OutboxStatusSending && r.UpdatedAt <= staleBeforeMs:
			out = append(out, r)
		}
	}
	return out, nil
}

func (f *fakeOutbox) ClaimOutbox(ctx context.Context, id int64, fromStatuses []string) (bool, error) {
	r, ok := f.rows[id]
	if !ok {
		return false, nil
	}
	for _, s := range fromStatuses {
		if r.Status == s {
			r.Status = repo.OutboxStatusSending
			return true, nil
		}
	}
	return false, nil
}

func (f *fakeOutbox) SettleOutbox(ctx context.Context, id int64, success bool, sendErr string) (int64, error) {
	r, ok := f.rows[id]
	if !ok {
		return 0, errors.New("no such row")
	}
	if success {
		r.Status = repo.OutboxStatusSent
		return 0, nil
	}
	r.RetryCount++
	r.Status = repo.OutboxStatusFailed
	r.LastError = &sendErr
	return r.RetryCount, nil
}

func (f *fakeOutbox) DeadOutboxWithAlert(ctx context.Context, id int64) error {
	f.rows[id].Status = repo.OutboxStatusDead
	f.alerts = append(f.alerts, id)
	return nil
}

// outboxRow 测试行工厂（pending）。
func outboxRow() *repo.Outbox {
	return &repo.Outbox{
		Kind:          NotificationKindGatePending,
		RequirementID: "req-001",
		NodeID:        "n-04",
		RecipientRole: "pm",
		Status:        repo.OutboxStatusPending,
	}
}

// TestUnit_NotifierOutboxEnqueueValidation EnqueueNotificationTx 入参校验：与 Notify
// 同口径（类别越集/缺需求/缺收件人拦截）；合法入参装配默认值与 pending 态。
func TestUnit_NotifierOutboxEnqueueValidation(t *testing.T) {
	for _, tc := range []struct {
		name string
		spec NotificationSpec
	}{
		{"类别越集", NotificationSpec{Kind: "no_such", RequirementID: "req-001", RecipientRole: "pm"}},
		{"缺需求", NotificationSpec{Kind: NotificationKindGatePending, RecipientRole: "pm"}},
		{"缺收件人", NotificationSpec{Kind: NotificationKindGatePending, RequirementID: "req-001"}},
	} {
		err := EnqueueNotificationTx(context.Background(), nil, tc.spec)
		assert.Assert(t, err != nil, "%s 应拦截（装配缺陷先于 IO 触发，不触 tx）", tc.name)
	}
	row, err := outboxRowOf(NotificationSpec{
		Kind: NotificationKindReReview, RequirementID: "req-001", NodeID: "n-04",
		TargetID: "cm-101", RecipientRole: "arch",
	})
	assert.NilError(t, err)
	assert.Equal(t, row.Kind, NotificationKindReReview)
	assert.Equal(t, row.Sender, "plugin", "派发者缺省 plugin")
	assert.Equal(t, row.SenderRole, "plugin", "派发角色缺省 plugin")
	assert.Equal(t, row.Status, repo.OutboxStatusPending, "enqueue 即 pending")
	assert.Assert(t, row.TargetID != nil && *row.TargetID == "cm-101", "锚点保留（retest 子型）")
}

// TestUnit_NotifierFlushPendingHappyPath 即时推送：pending 行经认领 → Notify 落
// issues 通知行 → settle sent。
func TestUnit_NotifierFlushPendingHappyPath(t *testing.T) {
	store := &fakeStore{issueID: "cm-200"}
	ob := newFakeOutbox(outboxRow())
	n := &Notifier{store: store, outbox: ob}
	n.FlushPending(context.Background())
	assert.Equal(t, store.calls, 1, "Notify 已投递一次")
	for _, r := range ob.rows {
		assert.Equal(t, r.Status, repo.OutboxStatusSent, "outbox 行置 sent")
	}
	assert.Assert(t, store.captured != nil, "issues 通知行已落库")
}

// TestUnit_NotifierPushOneRetryThenDead retry 耗尽：连续失败 5 次内保持 failed 可
// 补投，第 5 次失败后置 dead + audit 告警一次，不再被候选选取。
func TestUnit_NotifierPushOneRetryThenDead(t *testing.T) {
	store := &fakeStore{issueID: "cm-1", insertErr: errors.New("db down（注入）")}
	row := outboxRow()
	ob := newFakeOutbox(row)
	n := &Notifier{store: store, outbox: ob}
	for i := 0; i < repo.OutboxRetryLimit; i++ {
		assert.Assert(t, !n.pushOne(context.Background(), row,
			[]string{repo.OutboxStatusPending, repo.OutboxStatusFailed}), "注入故障下推送失败")
	}
	assert.Equal(t, row.RetryCount, int64(repo.OutboxRetryLimit), "retry 计数自增至上限")
	assert.Assert(t, len(ob.alerts) == 1, "耗尽置 dead + audit 告警一次")
	assert.Equal(t, row.Status, repo.OutboxStatusDead)
	// dead 不再被补投候选选取
	got, err := ob.ListRetryableOutbox(context.Background(), 0, 10)
	assert.NilError(t, err)
	assert.Equal(t, len(got), 0, "dead 行退出候选")
}
