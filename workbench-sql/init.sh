#!/usr/bin/env bash
# =====================================================================
# 团队工作台 · 一键初始化数据库脚本（init.sh）
# 用法：bash workbench-sql/init.sh
# 行为：rm -f workbench.db → 执行 schema.sql → 执行 seed.sql → 自检提示
# 目标库：/Users/huzilin/workdir/test-flow/.workbench/workbench.db
# 可用环境变量覆盖：WORKBENCH_DB=/path/to/db bash workbench-sql/init.sh
# =====================================================================
set -euo pipefail

# --- 路径解析 ----------------------------------------------------------
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
WORKBENCH_DIR="$PROJECT_ROOT/.workbench"
DB_PATH="${WORKBENCH_DB:-$WORKBENCH_DIR/workbench.db}"

# --- 前置检查 ----------------------------------------------------------
command -v sqlite3 >/dev/null 2>&1 || { echo "错误：未找到 sqlite3 CLI（macOS 自带）"; exit 1; }
[ -f "$SCRIPT_DIR/schema.sql" ] || { echo "错误：缺少 schema.sql（$SCRIPT_DIR/schema.sql）"; exit 1; }
[ -f "$SCRIPT_DIR/seed.sql" ]   || { echo "错误：缺少 seed.sql（$SCRIPT_DIR/seed.sql）"; exit 1; }

mkdir -p "$WORKBENCH_DIR"

# --- 执行 --------------------------------------------------------------
echo ">> 1/3 删除旧库: $DB_PATH"
rm -f "$DB_PATH"

echo ">> 2/3 建表: schema.sql"
sqlite3 "$DB_PATH" < "$SCRIPT_DIR/schema.sql"

echo ">> 3/3 种子数据: seed.sql"
sqlite3 "$DB_PATH" < "$SCRIPT_DIR/seed.sql"

# --- 自检 --------------------------------------------------------------
echo
echo ">> 自检：表清单"
sqlite3 "$DB_PATH" ".tables"

echo
echo ">> 自检：种子行数"
sqlite3 "$DB_PATH" "SELECT 'roles', count(*) FROM roles
UNION ALL SELECT 'capabilities', count(*) FROM capabilities
UNION ALL SELECT 'flow_templates', count(*) FROM flow_templates
UNION ALL SELECT 'flow_template_nodes', count(*) FROM flow_template_nodes
UNION ALL SELECT 'meta', count(*) FROM meta;"

echo
echo ">> 自检：capabilities 按 role_id 分布（roles→capabilities 1 对多）"
sqlite3 -header -column "$DB_PATH" "SELECT role_id, count(*) AS caps FROM capabilities GROUP BY role_id ORDER BY role_id;"

echo
echo ">> 自检：flow_template_nodes 分布（按模板，flow_templates_id 关联）"
sqlite3 -header -column "$DB_PATH" "SELECT t.flow_templates_id, count(*) AS nodes FROM flow_template_nodes n JOIN flow_templates t ON t.flow_templates_id = n.flow_templates_id GROUP BY t.flow_templates_id;"

echo
echo ">> 自检：JSON 字段合法性（json_valid=0 即非法）"
sqlite3 -header -column "$DB_PATH" "
SELECT 'capabilities.how' AS col, count(*) AS total, sum(json_valid(how)) AS valid FROM capabilities
UNION ALL SELECT 'capabilities.products', count(*), sum(json_valid(products)) FROM capabilities
UNION ALL SELECT 'flow_templates.scope_options', count(*), sum(json_valid(scope_options)) FROM flow_templates
UNION ALL SELECT 'flow_template_nodes.reviewers', count(*), sum(json_valid(reviewers)) FROM flow_template_nodes
UNION ALL SELECT 'flow_template_nodes.expected_outputs', count(*), sum(json_valid(expected_outputs)) FROM flow_template_nodes;"

echo
echo ">> 自检：产出能力 products 完整性（v0.28：产出型 11 项 products 非空且有 doc_path；acceptance-test 挂 2 产物）"
sqlite3 -header -column "$DB_PATH" "
SELECT
  sum(CASE WHEN json_array_length(products) > 0 THEN 1 ELSE 0 END) AS caps_with_products,
  sum(CASE WHEN json_array_length(products) > 0 AND json_extract(products, '\$[0].doc_path') IS NOT NULL THEN 1 ELSE 0 END) AS caps_with_doc,
  sum(CASE WHEN json_array_length(products) = 2 THEN 1 ELSE 0 END) AS caps_with_2_products
FROM capabilities;"

echo
echo ">> 自检：评审/决策类能力无产物（v0.28：products 应为空数组）"
sqlite3 -header -column "$DB_PATH" "
SELECT count(*) AS review_caps_no_products FROM capabilities
WHERE json_array_length(products) = 0;"

echo
echo ">> 自检：flow_template_nodes.mode 取值合法（produce/align；v0.28）"
sqlite3 -header -column "$DB_PATH" "SELECT mode, count(*) AS nodes FROM flow_template_nodes GROUP BY mode ORDER BY mode;"
sqlite3 -header -column "$DB_PATH" "SELECT count(*) AS invalid_mode FROM flow_template_nodes WHERE mode NOT IN ('produce','align');"
sqlite3 -header -column "$DB_PATH" "SELECT count(*) AS align_should_null_cap FROM flow_template_nodes WHERE mode='align' AND capability_id IS NOT NULL;"

echo
echo ">> 自检：flow_template_nodes.scope 取值合法（''/be/fe；v0.27 scope 裁剪列）"
sqlite3 -header -column "$DB_PATH" "SELECT scope, count(*) AS nodes FROM flow_template_nodes GROUP BY scope ORDER BY scope;"
sqlite3 -header -column "$DB_PATH" "SELECT count(*) AS invalid_scope FROM flow_template_nodes WHERE scope NOT IN ('','be','fe');"

echo
echo ">> 自检：时间戳完整性（v0.22：全部行 created_at/updated_at 非空）"
sqlite3 -header -column "$DB_PATH" "
SELECT 'roles' AS tbl, count(*) AS total, sum(created_at IS NOT NULL AND updated_at IS NOT NULL) AS ts_ok FROM roles
UNION ALL SELECT 'capabilities', count(*), sum(created_at IS NOT NULL AND updated_at IS NOT NULL) FROM capabilities
UNION ALL SELECT 'flow_templates', count(*), sum(created_at IS NOT NULL AND updated_at IS NOT NULL) FROM flow_templates
UNION ALL SELECT 'meta', count(*), sum(created_at IS NOT NULL AND updated_at IS NOT NULL) FROM meta;"

echo
echo ">> 完成 ✓ 数据库就绪: $DB_PATH"
