// T10 domain 扫描器表驱动单测（AC：5min/10min 按时触发 + 催办幂等 + session
// running 等待，无 IO 全覆盖）。时间以参数注入（ScanInput.Now），不依赖真实时钟。
package domain

import (
	"testing"

	"gotest.tools/v3/assert"

	"github.com/huzilin/wbflow/rd/repo"
)

const (
	testFollowupAfter  int64 = 5 * 60 * 1000 // 5min
	testInterveneAfter int64 = 10 * 60 * 1000
)

// scanTaskFixture 构造在途任务行（created=1000000，ack/更新时刻可覆写）。
func scanTaskFixture(mod func(*repo.Task)) *repo.Task {
	created := int64(1_000_000)
	task := &repo.Task{
		TaskID:        "t-1000000-001",
		RequirementID: "req-001",
		NodeID:        "n-03",
		RoleID:        "rd",
		Status:        repo.TaskStatusInProgress,
		CreatedAt:     created,
		UpdatedAt:     created,
	}
	if mod != nil {
		mod(task)
	}
	return task
}

// TestUnit_ScanTask_Thresholds 阈值边界逐点：未 ack 以创建时刻为基线——4min59s
// 不动、5min 整催办、9min59s 仍催办档、10min 整介入升级；已 ack 以最近上报为基线。
func TestUnit_ScanTask_Thresholds(t *testing.T) {
	created := int64(1_000_000)
	cases := []struct {
		name       string
		task       *repo.Task
		now        int64
		wantAction string
		wantIdle   int64
	}{
		{"未 ack 4min59s 不动", scanTaskFixture(nil), created + testFollowupAfter - 1, "", 0},
		{"未 ack 5min 整催办", scanTaskFixture(nil), created + testFollowupAfter, NotifKindFollowup, testFollowupAfter},
		{"未 ack 9min59s 仍催办档", scanTaskFixture(nil), created + testInterveneAfter - 1, NotifKindFollowup, testInterveneAfter - 1},
		{"未 ack 10min 整介入", scanTaskFixture(nil), created + testInterveneAfter, NotifKindIntervene, testInterveneAfter},
		// 已 ack：基线=最近上报（updated_at）；ack 后无推进 5min 催办
		{"已 ack 上报后 4min59s 不动", scanTaskFixture(func(t *repo.Task) {
			t.LastAck = ptrInt64(created + 60_000)
			t.UpdatedAt = created + 60_000
		}), created + 60_000 + testFollowupAfter - 1, "", 0},
		{"已 ack 上报后 5min 催办", scanTaskFixture(func(t *repo.Task) {
			t.LastAck = ptrInt64(created + 60_000)
			t.UpdatedAt = created + 60_000
		}), created + 60_000 + testFollowupAfter, NotifKindFollowup, testFollowupAfter},
		{"已 ack 上报后 10min 介入", scanTaskFixture(func(t *repo.Task) {
			t.LastAck = ptrInt64(created + 60_000)
			t.UpdatedAt = created + 60_000
		}), created + 60_000 + testInterveneAfter, NotifKindIntervene, testInterveneAfter},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			verdict := ScanTask(tc.task, ScanInput{
				Now:                  tc.now,
				FollowupAfterMillis:  testFollowupAfter,
				InterveneAfterMillis: testInterveneAfter,
			})
			assert.Equal(t, verdict.Action, tc.wantAction)
			if tc.wantAction != "" {
				assert.Equal(t, verdict.IdleMillis, tc.wantIdle)
			}
		})
	}
}

// TestUnit_ScanTask_Skip 终态任务不扫；绑定 session active（running 代理）→ 等待。
func TestUnit_ScanTask_Skip(t *testing.T) {
	created := int64(1_000_000)
	now := created + testInterveneAfter*10
	for _, status := range repo.TerminalTaskStatuses {
		verdict := ScanTask(scanTaskFixture(func(t *repo.Task) { t.Status = status }), ScanInput{
			Now: now, FollowupAfterMillis: testFollowupAfter, InterveneAfterMillis: testInterveneAfter,
		})
		assert.Equal(t, verdict.Action, "", "终态 %s 不应触发", status)
	}
	// session active → 等待（阈值再大也不催）
	verdict := ScanTask(scanTaskFixture(nil), ScanInput{
		Now: now, FollowupAfterMillis: testFollowupAfter, InterveneAfterMillis: testInterveneAfter,
		SessionActive: true,
	})
	assert.Equal(t, verdict.Action, "")
}

// TestUnit_ScanTask_Dedup 催办幂等判定：已有未闭合同类通知不重复触发；催办在先、
// 升级在后（有未闭合催办但越介入阈值 → 仍发介入）；已有介入通知后催办/介入均不再发。
func TestUnit_ScanTask_Dedup(t *testing.T) {
	created := int64(1_000_000)
	mk := func(now int64) ScanInput {
		return ScanInput{
			Now: now, FollowupAfterMillis: testFollowupAfter, InterveneAfterMillis: testInterveneAfter,
		}
	}
	// 未闭合催办存在 + 5min 档：不重复催
	in := mk(created + testFollowupAfter)
	in.OpenFollowup = true
	assert.Equal(t, ScanTask(scanTaskFixture(nil), in).Action, "")
	// 未闭合催办存在 + 越介入阈值：升级介入（催办不阻断升级）
	in = mk(created + testInterveneAfter)
	in.OpenFollowup = true
	assert.Equal(t, ScanTask(scanTaskFixture(nil), in).Action, NotifKindIntervene)
	// 未闭合介入存在：催办与介入均不再发
	in = mk(created + testInterveneAfter*2)
	in.OpenIntervene = true
	assert.Equal(t, ScanTask(scanTaskFixture(nil), in).Action, "")
	// 未闭合介入存在 + 未闭合催办存在：不重复
	in.OpenFollowup = true
	assert.Equal(t, ScanTask(scanTaskFixture(nil), in).Action, "")
}

// TestUnit_ScanTask_NegativeIdle 时钟回拨（now < 基线）闲置钳 0，不触发。
func TestUnit_ScanTask_NegativeIdle(t *testing.T) {
	verdict := ScanTask(scanTaskFixture(nil), ScanInput{
		Now: int64(999), FollowupAfterMillis: testFollowupAfter, InterveneAfterMillis: testInterveneAfter,
	})
	assert.Equal(t, verdict.Action, "")
	assert.Equal(t, verdict.IdleMillis, int64(0))
}

// TestUnit_ScannerNotificationText 通知文本单一口径：前缀可查重回读 + kind/任务/idle
// 编码无歧义（T11 收件箱按前缀分类的契约面）。
func TestUnit_ScannerNotificationText(t *testing.T) {
	cases := []struct {
		kind string
		want string
	}{
		{NotifKindFollowup, "[followup] task=t-1-001 idle_ms=300000 任务超时未推进，请尽快 ack/progress/complete；遇卡点请上报 user"},
		{NotifKindIntervene, "[intervene] task=t-1-001 idle_ms=600000 任务长时间未推进，需用户介入"},
	}
	for _, tc := range cases {
		text := ScannerNotificationText(tc.kind, "t-1-001", 300000)
		if tc.kind == NotifKindIntervene {
			text = ScannerNotificationText(tc.kind, "t-1-001", 600000)
		}
		assert.Equal(t, text, tc.want)
		// 查重前缀是文本前缀（repo LIKE 契约）
		prefix := ScannerNotificationTextPrefix(tc.kind, "t-1-001")
		assert.Assert(t, len(text) > len(prefix))
		assert.Equal(t, text[:len(prefix)], prefix)
	}
	// 未知 kind 兜底仍保持 [<kind>] task= 前缀形态（不产生无法查重的行）
	text := ScannerNotificationText("other", "t-1-001", 1)
	assert.Equal(t, text, "[other] task=t-1-001 idle_ms=1")
}

// ptrInt64 测试辅助（int64 指针惯例）。
func ptrInt64(v int64) *int64 { return &v }
