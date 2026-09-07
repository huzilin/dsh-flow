// 真实 MySQL 迁移集成测试（T2 AC）：迁移幂等（连跑两次 no-op）、14 表/索引落库、
// seed 行数与 docs-tmp 定稿逐项核对（scope_expr/reviewers/node_actions/expected_outputs）。
//
// env 门控，默认跳过；运行方式（密码经 env 注入，不落地仓库）：
//
//	WBFLOW_MYSQL_IT=1 \
//	WBFLOW_MYSQL_HOST=192.168.3.100 \
//	WBFLOW_MYSQL_PORT=3306 \
//	WBFLOW_MYSQL_USER=root \
//	WBFLOW_MYSQL_PASSWORD='...' \
//	go test ./migration/ -run TestIntegration_Migration -v
//
// 可选 WBFLOW_MYSQL_DB=<库名>：对指定库直接迁移并校验（不建不删，用于 wbflow 真实库验收）；
// 缺省每次建独立 scratch 库（wbflow_it_<时间戳>）并在测后 DROP，绝不污染既有库。
package migration_test

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"gorm.io/gorm"
	"gotest.tools/v3/assert"

	"github.com/huzilin/wbflow/rd/config"
	"github.com/huzilin/wbflow/rd/database"
	"github.com/huzilin/wbflow/rd/migration"
	"github.com/huzilin/wbflow/rd/pkg/utils"
	"github.com/huzilin/wbflow/rd/repo"
)

// 全部 14 表名（spec §0 + §0.3）。
var allTables = []string{
	"capabilities", "roles", "flow_defs", "flow_def_nodes", "flows", "flow_nodes",
	"tasks", "artifacts", "issues", "sessions", "audit_log", "meta", "requirements", "versions",
}

// 全部声明显式索引（DDL 内 UNIQUE KEY / KEY；主键除外）。
var allIndexes = map[string][]string{
	"capabilities":   {"uk_capabilities_cid", "idx_capabilities_role"},
	"roles":          {"uk_roles_rid"},
	"flow_defs":      {"uk_flow_defs_fid"},
	"flow_def_nodes": {"uk_fdnode_tnid", "uk_fdnode_order", "idx_fdnode_tpl"},
	"flows":          {"uk_flows_req"},
	"flow_nodes":     {"uk_flow_nodes_nid", "idx_flow_nodes_req", "idx_flow_nodes_def"},
	"tasks":          {"uk_tasks_tid", "idx_tasks_req", "idx_tasks_node", "idx_tasks_role", "idx_tasks_status", "idx_tasks_action", "idx_tasks_dispatch", "idx_tasks_last_reply"},
	"artifacts":      {"uk_artifacts_aid", "idx_artifacts_req", "idx_artifacts_ptype"},
	"issues":         {"uk_issues_iid", "idx_issues_node", "idx_issues_assignee", "idx_issues_state", "idx_issues_escalate"},
	"sessions":       {"uk_sessions_sid", "idx_sessions_req", "idx_sessions_node"},
	"audit_log":      {"idx_audit_row"},
	"meta":           {"uk_meta_key"},
	"requirements":   {"uk_requirements_rid", "idx_requirements_ver", "idx_requirements_status"},
	"versions":       {"uk_versions_vid"},
}

// TestIntegration_Migration_RealMySQL 迁移幂等 + 14 表/索引 + seed 行数。
func TestIntegration_Migration_RealMySQL(t *testing.T) {
	handles := openTarget(t)
	ctx := context.Background()
	g := handles.Gorm

	// AC：迁移可重复执行——空库首跑应用 v1+v2；二跑幂等 no-op
	// （WBFLOW_MYSQL_DB 指定库可能已迁移过，首跑 applied 允许为空）
	report1, err := migration.Run(ctx, g)
	assert.NilError(t, err)
	assert.Equal(t, report1.ToVersion, migration.SeedVersion)
	if os.Getenv("WBFLOW_MYSQL_DB") == "" {
		assert.DeepEqual(t, report1.Applied, []int64{migration.SchemaVersion, migration.SeedVersion})
	}

	report2, err := migration.Run(ctx, g)
	assert.NilError(t, err)
	assert.Equal(t, len(report2.Applied), 0, "重复执行应 no-op")
	assert.Equal(t, report2.ToVersion, migration.SeedVersion)

	// meta.schema_version 账本
	var version string
	assert.NilError(t, g.Raw("SELECT meta_value FROM meta WHERE meta_key = ?", migration.MetaKeySchemaVersion).Scan(&version).Error)
	assert.Equal(t, version, strconv.FormatInt(migration.SeedVersion, 10))

	// 14 表齐备（information_schema 口径）
	for _, table := range allTables {
		count := countRows(t, g,
			"SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?", table)
		assert.Equal(t, count, int64(1), "表应存在: "+table)
	}

	// 索引齐备（唯一键 + 二级索引逐个核对）
	for table, indexes := range allIndexes {
		for _, idx := range indexes {
			count := countRows(t, g,
				"SELECT COUNT(DISTINCT INDEX_NAME) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND INDEX_NAME = ?", table, idx)
			assert.Equal(t, count, int64(1), fmt.Sprintf("索引应存在: %s.%s", table, idx))
		}
	}

	// 引擎与字符集抽核（DDL 约定：InnoDB + utf8mb4）
	var tableCollation string
	assert.NilError(t, g.Raw(
		"SELECT TABLE_COLLATION FROM information_schema.TABLES WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'flow_def_nodes'").
		Scan(&tableCollation).Error)
	assert.Assert(t, len(tableCollation) >= 7 && tableCollation[:7] == "utf8mb4", "定义节点表字符集应 utf8mb4，实测: "+tableCollation)

	// seed 行数：roles 全集 10 / 能力 21 / 流程定义 4 / 定义节点 40（business 13 + tech 11 + task 7 + plan 9）
	// 注：工单文案「41」为 task C5 删除前陈旧计数，docs-tmp 定稿为 40（见交付报告）。
	assert.Equal(t, countRows(t, g, "SELECT COUNT(*) FROM roles"), int64(10), "roles 行数")
	assert.Equal(t, countRows(t, g, "SELECT COUNT(*) FROM capabilities"), int64(21), "capabilities 行数")
	assert.Equal(t, countRows(t, g, "SELECT COUNT(*) FROM flow_defs"), int64(4), "flow_defs 行数")
	assert.Equal(t, countRows(t, g, "SELECT COUNT(*) FROM flow_def_nodes"), int64(40), "flow_def_nodes 行数")
	for flow, want := range map[string]int64{"business": 13, "tech": 11, "task": 7, "plan": 9} {
		assert.Equal(t, countRows(t, g, "SELECT COUNT(*) FROM flow_def_nodes WHERE flow_defs_id = ?", flow),
			want, "流程节点数: "+flow)
	}
	assert.Equal(t, countRows(t, g, "SELECT COUNT(*) FROM flow_defs WHERE scope_options = ? AND flow_defs_id = 'plan'", "[]"),
		int64(1), "plan 无 scope")
}

// TestIntegration_SeedGolden_RealMySQL seed 与 docs-tmp 定稿逐项核对：
// 40 定义节点的 node_type/gate/user_gate/scope_expr/reviewers/node_actions/expected_outputs。
func TestIntegration_SeedGolden_RealMySQL(t *testing.T) {
	handles := openTarget(t)
	g := handles.Gorm
	_, err := migration.Run(context.Background(), g) // 目标库已迁移则 no-op
	assert.NilError(t, err)

	for _, want := range goldenNodes() {
		var row repo.FlowDefNode
		err := g.Raw("SELECT * FROM flow_def_nodes WHERE template_node_id = ?", want.TemplateNodeID).Scan(&row).Error
		assert.NilError(t, err, "定义节点应存在: "+want.TemplateNodeID)
		assert.Equal(t, row.FlowDefsID, want.FlowDefsID, want.TemplateNodeID+": flow_defs_id")
		assert.Equal(t, row.NodeOrder, want.NodeOrder, want.TemplateNodeID+": node_order")
		assert.Equal(t, row.NodeKey, want.NodeKey, want.TemplateNodeID+": node_key")
		assert.Equal(t, row.NodeType, want.NodeType, want.TemplateNodeID+": node_type")
		assert.Equal(t, row.Gate, want.Gate, want.TemplateNodeID+": gate")
		assert.Equal(t, row.UserGate, want.UserGate, want.TemplateNodeID+": user_gate")
		assert.Equal(t, row.ScopeExpr, want.ScopeExpr, want.TemplateNodeID+": scope_expr")
		assert.Equal(t, row.Reviewers, want.Reviewers, want.TemplateNodeID+": reviewers")
		assert.Equal(t, row.NodeActions, want.NodeActions, want.TemplateNodeID+": node_actions")
		assert.Equal(t, row.ExpectedOutputs, want.ExpectedOutputs, want.TemplateNodeID+": expected_outputs")
	}

	// flow_defs.scope_options 四流程核对
	for flow, wantScope := range map[string]string{
		"business": `["full","rd","fe"]`, "tech": `["full","rd","fe"]`, "task": `["rd","fe"]`, "plan": `[]`,
	} {
		var got string
		assert.NilError(t, g.Raw("SELECT scope_options FROM flow_defs WHERE flow_defs_id = ?", flow).Scan(&got).Error)
		assert.Equal(t, got, wantScope, "scope_options: "+flow)
	}

	// capabilities JSON 载荷可按 Go struct 口径反序列化（21 能力全量抽验动作挂载）
	var caps []repo.Capability
	assert.NilError(t, g.Raw("SELECT * FROM capabilities ORDER BY capability_id").Scan(&caps).Error)
	assert.Equal(t, len(caps), 21)
	actionsByCap := map[string][]repo.ActionDef{}
	for _, cap := range caps {
		var inputs []repo.CapabilityInput
		var actions []repo.ActionDef
		var products []repo.ProductDef
		assert.NilError(t, utils.UnmarshalJSONText(cap.Inputs, &inputs), cap.CapabilityID+": inputs")
		assert.NilError(t, utils.UnmarshalJSONText(cap.Actions, &actions), cap.CapabilityID+": actions")
		assert.NilError(t, utils.UnmarshalJSONText(cap.Products, &products), cap.CapabilityID+": products")
		assert.Assert(t, len(actions) > 0, cap.CapabilityID+" 应有动作")
		for _, action := range actions {
			assert.Assert(t, action.On == repo.PhaseExec || action.On == repo.PhaseReview || action.On == repo.PhaseRespond,
				cap.CapabilityID+"."+action.ActionID+" on 非法: "+action.On)
		}
		actionsByCap[cap.CapabilityID] = actions
	}
	// 定稿抽核：decide-cap 收缩为 create_reqs 单动作；backend-dev 五动作；评审能力=review+confirm
	assert.Equal(t, len(actionsByCap["decide-cap"]), 1)
	assert.Equal(t, actionsByCap["decide-cap"][0].ActionID, "create_reqs")
	assert.Equal(t, actionsByCap["decide-cap"][0].Skill, "to-tickets")
	assert.Equal(t, len(actionsByCap["backend-dev"]), 5)
	assert.Equal(t, len(actionsByCap["arch.prd-review"]), 2)
	assert.Equal(t, actionsByCap["arch.prd-review"][0].Skill, "prd-review-arch-skill")
}

// openTarget 按环境准备目标库句柄：WBFLOW_MYSQL_DB 指定库直接迁移（不删），
// 否则建 scratch 库并在测后 DROP（绝不污染既有库）。密码仅经 WBFLOW_MYSQL_PASSWORD env 注入。
func openTarget(t *testing.T) *database.DB {
	t.Helper()
	if os.Getenv("WBFLOW_MYSQL_IT") == "" {
		t.Skip("真实 MySQL 集成：设 WBFLOW_MYSQL_IT=1 且注入 WBFLOW_MYSQL_PASSWORD 后运行")
	}
	cfg := &config.MySQLConfig{
		Host:               envOr("WBFLOW_MYSQL_HOST", "192.168.3.100"),
		Port:               3306,
		User:               envOr("WBFLOW_MYSQL_USER", "root"),
		Password:           os.Getenv("WBFLOW_MYSQL_PASSWORD"),
		DBName:             envOr("WBFLOW_MYSQL_DB", ""),
		MaxOpenConns:       5,
		MaxIdleConns:       2,
		ConnMaxLifetimeSec: 3600,
	}
	if p := os.Getenv("WBFLOW_MYSQL_PORT"); p != "" {
		port, err := strconv.ParseInt(p, 10, 64)
		assert.NilError(t, err)
		cfg.Port = port
	}
	if cfg.DBName == "" {
		cfg.DBName = fmt.Sprintf("wbflow_it_%d", time.Now().UnixNano())
		t.Logf("scratch 库: %s", cfg.DBName)
		t.Cleanup(func() {
			dropDatabase(t, cfg)
		})
	} else {
		t.Logf("目标库（不建不删）: %s", cfg.DBName)
	}
	handles, err := migration.Bootstrap(context.Background(), cfg)
	assert.NilError(t, err)
	t.Cleanup(func() {
		_ = handles.Close()
	})
	return handles
}

func dropDatabase(t *testing.T, cfg *config.MySQLConfig) {
	t.Helper()
	handles, err := database.Open(cfg)
	if err != nil {
		t.Fatalf("清理句柄打开失败: %v", err)
	}
	defer func() { _ = handles.Close() }()
	_, err = handles.Server.Exec("DROP DATABASE IF EXISTS `" + cfg.DBName + "`")
	if err != nil {
		t.Logf("scratch 库清理失败（%s）: %v", cfg.DBName, err)
	}
}

func countRows(t *testing.T, g dbLike, query string, args ...any) int64 {
	t.Helper()
	var count int64
	assert.NilError(t, g.Raw(query, args...).Scan(&count).Error)
	return count
}

type dbLike interface {
	Raw(query string, args ...any) *gorm.DB
	Exec(sql string, values ...any) *gorm.DB
}

func envOr(key string, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
