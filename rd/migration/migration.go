// Package migration 负责 schema 迁移与 seed 装载：
// 嵌入 SQL（go:embed）按版本号有序应用，meta.schema_version 记录当前版本，
// 重复执行幂等 no-op；只建不删——禁止任何自动 DROP。
//
// 版本语义：schema_version 是迁移账本版本（本包推进），与 updated_at 乐观锁、
// 业务正文版本号是三类不同「版本」，不得混同（spec §0 裁决）。
package migration

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"

	perr "github.com/pingcap/errors"

	"github.com/huzilin/wbflow/rd/config"
	"github.com/huzilin/wbflow/rd/database"
	"github.com/huzilin/wbflow/rd/pkg/errs"
)

//go:embed sql/0001_schema.sql
var schemaSQL string

//go:embed sql/0002_seed.sql
var seedSQL string

//go:embed sql/0003_flow_scope_last_decision.sql
var flowScopeDecisionSQL string

//go:embed sql/0004_notify_outbox.sql
var notifyOutboxSQL string

// 版本号常量（迁移账本 meta.schema_version 取值）。
const (
	// SchemaVersion 14 表全量 DDL（spec §0 + §0.3）
	SchemaVersion int64 = 1
	// SeedVersion seed 基线：roles 全集 + 21 能力 + 4 flow_defs + 40 定义节点（docs-tmp 定稿口径）
	SeedVersion int64 = 2
	// FlowScopeDecisionVersion T4 流程实例化补列：flow_nodes.last_decision（spec §0 漏项，
	// arch_design v0.29 同源）+ flows.scope（需求所选渲染 scope 落库）
	FlowScopeDecisionVersion int64 = 3
	// NotifyOutboxVersion 通知可靠投递 outbox 化（spec §1.2 注记-H）：消息事务表
	NotifyOutboxVersion int64 = 4
	// MetaKeySchemaVersion meta 表中迁移账本键名
	MetaKeySchemaVersion = "schema_version"
)

// step 单个迁移版本。
type step struct {
	version int64
	name    string
	sql     string
}

// steps 全量迁移步骤（按版本号升序，只增不改）。
var steps = []step{
	{version: SchemaVersion, name: "create_schema", sql: schemaSQL},
	{version: SeedVersion, name: "seed_baseline", sql: seedSQL},
	{version: FlowScopeDecisionVersion, name: "flow_scope_last_decision", sql: flowScopeDecisionSQL},
	{version: NotifyOutboxVersion, name: "notify_outbox", sql: notifyOutboxSQL},
}

// LatestVersion 迁移账本最新版本（部署校验与测试断言口径，随 steps 尾元素推进）。
func LatestVersion() int64 {
	return steps[len(steps)-1].version
}

// Report 迁移执行结果。
type Report struct {
	// FromVersion 执行前账本版本（空库为 0）
	FromVersion int64
	// ToVersion 执行后账本版本
	ToVersion int64
	// Applied 本次实际应用的版本号（幂等 no-op 时为空）
	Applied []int64
}

// EnsureDatabase 服务器级引导：CREATE DATABASE IF NOT EXISTS（连接 server 句柄执行，
// database.Open 已提供无 dbname 句柄）。只建不删——库已存在即 no-op，禁止 DROP。
func EnsureDatabase(ctx context.Context, server *sql.DB, dbname string) error {
	if !validDBName(dbname) {
		return errs.Trace(perr.Errorf("非法 dbname: %q（仅允许字母数字下划线）", dbname))
	}
	stmt := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci", dbname)
	if _, err := server.ExecContext(ctx, stmt); err != nil {
		return errs.Trace(err)
	}
	return nil
}

// Bootstrap 部署引导入口：确保 dbname 存在，并返回 Gorm 句柄可用的连接集合。
// 首次建库场景 gorm 打开失败（库缺失）时自动重开一次。
func Bootstrap(ctx context.Context, cfg *config.MySQLConfig) (*database.DB, error) {
	handles, err := database.Open(cfg)
	if err != nil {
		return nil, errs.Trace(err)
	}
	if pingErr := handles.PingContext(ctx); pingErr != nil {
		_ = handles.Close()
		return nil, errs.Trace(fmt.Errorf("mysql 服务器不可达: %w", pingErr))
	}
	if ensureErr := EnsureDatabase(ctx, handles.Server, cfg.DBName); ensureErr != nil {
		_ = handles.Close()
		return nil, ensureErr
	}
	if handles.OpenErr == nil {
		return handles, nil
	}
	// 库刚建成，重开全量句柄
	_ = handles.Close()
	handles, err = database.Open(cfg)
	if err != nil {
		return nil, errs.Trace(err)
	}
	if handles.OpenErr != nil {
		_ = handles.Close()
		return nil, errs.Trace(handles.OpenErr)
	}
	return handles, nil
}

// Run 按版本号升序应用全部未执行的迁移步骤；已应用版本跳过（幂等 no-op）。
// 每步先执行该版本全部 SQL 语句，再推进 meta.schema_version 账本。
func Run(ctx context.Context, g *gorm.DB) (*Report, error) {
	current, err := currentVersion(ctx, g)
	if err != nil {
		return nil, err
	}
	report := &Report{FromVersion: current, ToVersion: current}
	for _, s := range steps {
		if s.version <= current {
			continue
		}
		if applyErr := applyStep(ctx, g, s); applyErr != nil {
			return nil, applyErr
		}
		report.Applied = append(report.Applied, s.version)
		report.ToVersion = s.version
	}
	return report, nil
}

// currentVersion 读取 meta.schema_version；meta 表不存在（首库）或无记录按版本 0。
func currentVersion(ctx context.Context, g *gorm.DB) (int64, error) {
	var value string
	err := g.WithContext(ctx).
		Raw("SELECT meta_value FROM meta WHERE meta_key = ?", MetaKeySchemaVersion).
		Scan(&value).Error
	if err != nil {
		if isTableMissingErr(err) {
			return 0, nil
		}
		return 0, errs.Trace(err)
	}
	if value == "" {
		return 0, nil
	}
	version, parseErr := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if parseErr != nil {
		return 0, errs.Trace(fmt.Errorf("meta.schema_version 非法值 %q: %w", value, parseErr))
	}
	return version, nil
}

// applyStep 执行单版本：拆语句逐条执行后推进账本（DDL 隐式提交，无法包事务；
// 中断重跑安全性由各版本 SQL 自带幂等保证——0001 CREATE TABLE IF NOT EXISTS /
// 0002 INSERT IGNORE / 0003 裸 ALTER + 本函数对「已存在」类错误宽容，见各 SQL 文件头注）。
// 宽容口径：迁移 SQL 受「只增不改，禁止 DROP」约束，1060/1061 唯一来源是重复应用
// （列/索引已在）＝该语句已生效，跳过继续，账本正常推进。
func applyStep(ctx context.Context, g *gorm.DB, s step) error {
	for _, stmt := range splitStatements(s.sql) {
		if execErr := g.WithContext(ctx).Exec(stmt).Error; execErr != nil {
			if isDuplicateObjErr(execErr) {
				continue
			}
			return errs.Trace(fmt.Errorf("迁移 v%d(%s) 语句执行失败: %w", s.version, s.name, execErr))
		}
	}
	return bumpVersion(ctx, g, s.version)
}

// bumpVersion 推进 meta.schema_version（upsert，时间戳应用层维护）。
func bumpVersion(ctx context.Context, g *gorm.DB, version int64) error {
	now := gormNowExpr(g)
	err := g.WithContext(ctx).Exec(
		"INSERT INTO meta (meta_key, meta_value, created_at, updated_at) VALUES (?, ?, "+now+", "+now+") "+
			"ON DUPLICATE KEY UPDATE meta_value = VALUES(meta_value), updated_at = VALUES(updated_at)",
		MetaKeySchemaVersion, strconv.FormatInt(version, 10),
	).Error
	if err != nil {
		return errs.Trace(err)
	}
	return nil
}

// gormNowExpr 取数据库侧 unix 毫秒表达式（账本时间戳与业务列同口径，MySQL 专用）。
func gormNowExpr(_ *gorm.DB) string {
	return "(ROUND(UNIX_TIMESTAMP(NOW(3)) * 1000))"
}

// isTableMissingErr 判定 MySQL 1146（表不存在）。
func isTableMissingErr(err error) bool {
	var mysqlErr *mysql.MySQLError
	return asMySQLError(err, &mysqlErr) && mysqlErr.Number == 1146
}

// isDuplicateObjErr 判定 MySQL「对象已存在」类错误：1060（重复列名）/ 1061（重复索引名）。
// 迁移重跑幂等的实现基础（0003 裸 ALTER 重跑场景）；若未来版本引入 ADD INDEX 等新
// 已存在形态，扩充此判定即可，SQL 文件无需回改。
func isDuplicateObjErr(err error) bool {
	var mysqlErr *mysql.MySQLError
	return asMySQLError(err, &mysqlErr) &&
		(mysqlErr.Number == 1060 || mysqlErr.Number == 1061)
}

// asMySQLError errors.As 薄封装（收拢依赖）。
func asMySQLError(err error, target **mysql.MySQLError) bool {
	for cur := err; cur != nil; {
		if e, ok := cur.(*mysql.MySQLError); ok {
			*target = e
			return true
		}
		causer, ok := cur.(interface{ Cause() error })
		if !ok {
			return false
		}
		cur = causer.Cause()
	}
	return false
}

// validDBName 库名白名单（防注入：EnsureDatabase 拼接标识符）。
var dbNamePattern = regexp.MustCompile(`^[A-Za-z0-9_]+$`)

func validDBName(name string) bool {
	return name != "" && len(name) <= 64 && dbNamePattern.MatchString(name)
}

// splitStatements 拆分 SQL 脚本为逐条语句：剥离 -- 行注释，
// 识别单引号字符串（含 ” 转义）内的分号，按 ';' 切分，丢弃空语句。
func splitStatements(script string) []string {
	var (
		statements []string
		current    strings.Builder
		inString   bool
	)
	flush := func() {
		stmt := strings.TrimSpace(current.String())
		current.Reset()
		if stmt != "" {
			statements = append(statements, stmt)
		}
	}
	lines := strings.Split(script, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !inString && strings.HasPrefix(trimmed, "--") {
			continue
		}
		i := 0
		for i < len(line) {
			ch := line[i]
			switch {
			case inString && ch == '\'' && i+1 < len(line) && line[i+1] == '\'':
				current.WriteString("''")
				i += 2
				continue
			case ch == '\'':
				inString = !inString
			case ch == ';' && !inString:
				flush()
				i++
				continue
			}
			current.WriteByte(ch)
			i++
		}
		current.WriteByte('\n')
	}
	flush()
	return statements
}

// OpenForMigrate 便捷封装：Bootstrap + Run（部署/验证脚本一步到位）。
func OpenForMigrate(ctx context.Context, cfg *config.MySQLConfig) (*database.DB, *Report, error) {
	handles, err := Bootstrap(ctx, cfg)
	if err != nil {
		return nil, nil, err
	}
	report, err := Run(ctx, handles.Gorm)
	if err != nil {
		_ = handles.Close()
		return nil, nil, err
	}
	return handles, report, nil
}
