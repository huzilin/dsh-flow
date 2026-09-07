// 真实 MySQL repo 基线集成测试（T2 AC）：乐观锁并发一胜一败（败者收 ErrOptimisticLock）、
// append-only 审计写入 created_at=updated_at 恒同值 + 同事务回滚不残留。
//
// 始终使用独立 scratch 库（wbflow_it_<时间戳>）并在测后 DROP——本文件会写入业务行，
// 绝不落 WBFLOW_MYSQL_DB 指定库，也绝不污染 wbflow 库。env 门控同 migration 测试。
package repo_test

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"gotest.tools/v3/assert"

	"github.com/huzilin/wbflow/rd/config"
	"github.com/huzilin/wbflow/rd/database"
	"github.com/huzilin/wbflow/rd/migration"
	"github.com/huzilin/wbflow/rd/pkg/errs"
	"github.com/huzilin/wbflow/rd/repo"
)

// TestIntegration_OptimisticLock_RealMySQL 并发更新同一行：一胜一败，败者收 ErrOptimisticLock。
func TestIntegration_OptimisticLock_RealMySQL(t *testing.T) {
	handles := openScratch(t)
	ctx := context.Background()
	r := repo.NewRepo(handles.Gorm)

	now := repo.NowMillis()
	seed := &repo.Task{
		TaskID: "t-it-lock-1", RequirementID: "req-it", NodeID: "n-1", ActionID: "backend-dev.dev",
		RoleID: "rd", TaskType: "dev", Goal: "乐观锁集成测试", Context: "{}",
		Status: "todo", CreatedAt: now, UpdatedAt: now,
	}
	assert.NilError(t, handles.Gorm.WithContext(ctx).Create(seed).Error)

	// 两个并发方读到同一基线版本
	var base repo.Task
	assert.NilError(t, handles.Gorm.WithContext(ctx).
		Raw("SELECT * FROM tasks WHERE task_id = ?", seed.TaskID).Scan(&base).Error)

	type result struct {
		winner bool
		err    error
	}
	start := make(chan struct{})
	results := make(chan result, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(seq int) {
			defer wg.Done()
			<-start
			err := r.UpdateWithOptimisticLock(ctx, &repo.Task{}, base.ID, base.UpdatedAt, map[string]any{
				"status": fmt.Sprintf("in-progress-%d", seq),
			})
			results <- result{err == nil, err}
		}(i)
	}
	close(start)
	wg.Wait()
	close(results)

	winners, losers := 0, 0
	for res := range results {
		if res.winner {
			winners++
			continue
		}
		losers++
		assert.Assert(t, errs.Is(res.err, errs.ErrOptimisticLock),
			"败者应命中 ErrOptimisticLock，实测: "+res.err.Error())
	}
	assert.Equal(t, winners, 1, "并发更新应恰好一胜")
	assert.Equal(t, losers, 1, "并发更新应恰好一败")

	// 胜者生效 + updated_at 前移（新基线），旧基线再试必败
	var after repo.Task
	assert.NilError(t, handles.Gorm.WithContext(ctx).
		Raw("SELECT * FROM tasks WHERE task_id = ?", seed.TaskID).Scan(&after).Error)
	assert.Assert(t, after.UpdatedAt != base.UpdatedAt, "updated_at 应前移")
	assert.Assert(t, after.Status != "todo", "胜者 status 应生效")
	err := r.UpdateWithOptimisticLock(ctx, &repo.Task{}, after.ID, base.UpdatedAt, map[string]any{"status": "completed"})
	assert.Assert(t, errs.Is(err, errs.ErrOptimisticLock), "旧基线更新应命中 ErrOptimisticLock")
}

// TestIntegration_AuditAppendOnly_RealMySQL 审计 helper：created_at=updated_at 恒同值；
// 同事务写入随业务事务回滚不残留（审计失败回滚业务事务的同库同事务前提）。
func TestIntegration_AuditAppendOnly_RealMySQL(t *testing.T) {
	handles := openScratch(t)
	ctx := context.Background()
	r := repo.NewRepo(handles.Gorm)

	// 1. helper 强制 created_at=updated_at（调用方传入值不生效）
	entry := &repo.AuditLog{
		Table: "tasks", RowID: "t-it-audit-1", Action: "INSERT",
		ByWho: "system", CreatedAt: 1, UpdatedAt: 2,
	}
	assert.NilError(t, r.InsertAuditLog(ctx, entry))
	assert.Assert(t, entry.ID > 0, "审计行应落库")
	var stored repo.AuditLog
	assert.NilError(t, handles.Gorm.WithContext(ctx).Raw("SELECT * FROM audit_log WHERE id = ?", entry.ID).Scan(&stored).Error)
	assert.Equal(t, stored.CreatedAt, stored.UpdatedAt, "append-only：created_at=updated_at 恒同值")
	assert.Assert(t, stored.CreatedAt > 0, "时间戳应为应用层毫秒值")

	// 2. 第二条独立写入时间戳单调
	second := &repo.AuditLog{Table: "tasks", RowID: "t-it-audit-1", Action: "UPDATE", Field: strPtr("status"), ByWho: "rd"}
	assert.NilError(t, r.InsertAuditLog(ctx, second))
	assert.Assert(t, second.UpdatedAt >= stored.UpdatedAt, "审计时间应单调")

	// 3. 业务事务内写审计 + 随事务回滚：不残留
	tx := handles.Gorm.WithContext(ctx).Begin()
	assert.NilError(t, tx.Error)
	inTx := &repo.AuditLog{Table: "flows", RowID: "req-it", Action: "INSERT", ByWho: "system"}
	assert.NilError(t, r.InsertAuditLogWith(tx, inTx))
	tx.Rollback()
	count := int64(0)
	assert.NilError(t, handles.Gorm.WithContext(ctx).
		Raw("SELECT COUNT(*) FROM audit_log WHERE table_name = ? AND row_id = ?", "flows", "req-it").Scan(&count).Error)
	assert.Equal(t, count, int64(0), "回滚后审计行不残留")
}

// openScratch 每测试独立 scratch 库（建库 → 迁移），测后 DROP。
func openScratch(t *testing.T) *database.DB {
	t.Helper()
	if os.Getenv("WBFLOW_MYSQL_IT") == "" {
		t.Skip("真实 MySQL 集成：设 WBFLOW_MYSQL_IT=1 且注入 WBFLOW_MYSQL_PASSWORD 后运行")
	}
	cfg := &config.MySQLConfig{
		Host:               envOr("WBFLOW_MYSQL_HOST", "192.168.3.100"),
		Port:               3306,
		User:               envOr("WBFLOW_MYSQL_USER", "root"),
		Password:           os.Getenv("WBFLOW_MYSQL_PASSWORD"),
		MaxOpenConns:       5,
		MaxIdleConns:       2,
		ConnMaxLifetimeSec: 3600,
	}
	if p := os.Getenv("WBFLOW_MYSQL_PORT"); p != "" {
		port, err := strconv.ParseInt(p, 10, 64)
		assert.NilError(t, err)
		cfg.Port = port
	}
	cfg.DBName = fmt.Sprintf("wbflow_it_%d", time.Now().UnixNano())
	t.Logf("scratch 库: %s", cfg.DBName)

	handles, err := migration.Bootstrap(context.Background(), cfg)
	assert.NilError(t, err)
	t.Cleanup(func() {
		_ = handles.Close()
		dropScratch(t, cfg)
	})
	report, err := migration.Run(context.Background(), handles.Gorm)
	assert.NilError(t, err)
	assert.Equal(t, report.ToVersion, migration.SeedVersion)
	return handles
}

func dropScratch(t *testing.T, cfg *config.MySQLConfig) {
	t.Helper()
	handles, err := database.Open(cfg)
	if err != nil {
		t.Fatalf("清理句柄打开失败: %v", err)
	}
	defer func() { _ = handles.Close() }()
	if _, err := handles.Server.Exec("DROP DATABASE IF EXISTS `" + cfg.DBName + "`"); err != nil {
		t.Logf("scratch 库清理失败（%s）: %v", cfg.DBName, err)
	}
}

func strPtr(s string) *string { return &s }

func envOr(key string, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
