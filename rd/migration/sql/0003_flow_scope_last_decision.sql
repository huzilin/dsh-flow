-- 迁移 v3（T4 流程实例化）：spec §0 漏项补列 + scope 选择值落库。只增不改，禁止 DROP。
--
-- 1) flow_nodes.last_decision：门禁裁决 approved/rejected 承载列（TEXT NULL）。
--    依据：
--      - docs/v1.0/arch_design.md v0.29 flow_nodes.last_decision
--        （JSON:{verdict,reason,at,branch?}，NULL=未决策）；
--      - docs-tmp/capability-model-final.md §1.2a「approved/rejected 是 collect 的输出结果，
--        写入 last_decision」；
--      - docs-tmp/data-tools-final.md「last_decision 保留，JSON 扩展承载 reject 结论协议
--        （branches/confirmed_issues/context）」。
--    spec §0 flow_nodes 表尾注「门禁裁决 approved/rejected 由 last_decision JSON 承载
--    （见 flow_nodes 扩展或独立列）」即指向本列——T2 落库时发现的 spec 漏项，本版本补齐。
--
-- 2) flows.scope：需求所选渲染 scope（full/rd/fe；''=无 scope，plan 流程）。
--    依据（「需求所选 scope」需要落库而 flows 表无列）：
--      - docs-tmp/ui-interaction-final.md §1.1「节点集 = flow_def_nodes.scope 表达式
--        ∩ requirement.scope」——交集运算的右操作数是持久化选择值；
--      - 同文 §2「scope 徽标：需求卡/列表行/详情补 scope 徽标（full/rd/fe 三色区分）
--        ——需求 scope 三值」——选择值需跨会话展示；
--      - spec §1.2 wb_list_requirements「version/priority/status/scope 筛选」——可筛选
--        ⇒ 必落库。
--    flows 与 requirement 一比一（uk_flows_req），承载于流程实例侧（工单 T4 裁决口径）；
--    实例节点集 = flow_def_nodes.scope_expr ∩ flows.scope（实例化投影，不另存选择值副本）。
--
-- 幂等（与 0001 CREATE TABLE IF NOT EXISTS / 0002 INSERT IGNORE 对称）：MySQL 8 无
-- ADD COLUMN IF NOT EXISTS，改用 INFORMATION_SCHEMA.COLUMNS 存在性检查 + PREPARE/EXECUTE
-- 条件 DDL 惯用法——applyStep 拆句逐条执行（migration.go splitStatements），中断重跑时
-- 已加列走 SELECT 1 空操作分支，不再报 1060 重复列错误。
-- 注意：本文件为 MySQL 专用 DDL（PREPARE/INFORMATION_SCHEMA 语法 sqlite 无对应），
-- 本地替身测试不可覆盖；真库行为由 migration_it_test.go（env 门控）scratch 库连跑两次
-- 「首跑 applied=[1 2 3] + 二跑 no-op」断言守护。

SET @fnd_last_decision_exists = (
  SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'flow_nodes' AND COLUMN_NAME = 'last_decision'
);
SET @fnd_ddl = IF(
  @fnd_last_decision_exists = 0,
  'ALTER TABLE flow_nodes ADD COLUMN last_decision TEXT NULL COMMENT ''门禁裁决结果 JSON 文本：{"verdict":"approved|rejected","reason":"…","at":毫秒,"branch":"…"}；NULL=未判定（approved/rejected=collect 聚合输出，capability-model-final §1.2a；reject 结论协议扩展承载 branches/confirmed_issues/context）'' AFTER pending_scope',
  'SELECT 1'
);
PREPARE fnd_stmt FROM @fnd_ddl;
EXECUTE fnd_stmt;
DEALLOCATE PREPARE fnd_stmt;

SET @flow_scope_exists = (
  SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'flows' AND COLUMN_NAME = 'scope'
);
SET @flow_ddl = IF(
  @flow_scope_exists = 0,
  'ALTER TABLE flows ADD COLUMN scope VARCHAR(8) NOT NULL DEFAULT '''' COMMENT ''需求所选渲染 scope：full/rd/fe；''''=无 scope（plan 流程）。实例化节点集=flow_def_nodes.scope_expr ∩ 本列（ui-interaction-final §1.1）；scope 徽标/列表筛选的消费证据见迁移头注'' AFTER dependencies',
  'SELECT 1'
);
PREPARE flow_stmt FROM @flow_ddl;
EXECUTE flow_stmt;
DEALLOCATE PREPARE flow_stmt;
