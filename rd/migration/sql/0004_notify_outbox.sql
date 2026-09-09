-- 迁移 v4（通知可靠投递 outbox 化）：消息事务表。只增不改，禁止 DROP。
--
-- 1) notify_outbox：通知意图先行落表（与业务写路径同事务 enqueue），事务提交后
--    flush 推送为 issues 通知行（issue_type=notification），失败由 scanner 周期
--    补投（retry 上限后置 dead 终态 + audit 留痕）。依据：
--      - spec §1.3「消息/通知派发移出事务」口径的可靠性升级：原 Dispatch 事务外
--        吞错（失败仅记日志）存在静默丢失窗口（联调实证 req-002 gate_pending
--        一次未落）——本表使通知意图与业务提交原子，flush/补投失败不丢意图；
--      - 「失败不阻断业务路径」口径不变：flush 失败不影响业务结果。
--    收件箱读侧不变（仍读 issues 通知行投影），FE 零改动。
--
-- 状态机：pending → sending（认领推送中）→ sent | failed（retry_count++）；
--   failed 且 retry_count < 5 可再认领；retry 耗尽置 dead（终态，audit 告警一次）；
--   sending 停滞超阈值（进程崩溃）由 scanner 按 updated_at 恢复认领。
--
-- 幂等（与 0001/0003 对称）：CREATE TABLE IF NOT EXISTS，重复应用零效应。
CREATE TABLE IF NOT EXISTS notify_outbox (
  id             BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  kind           VARCHAR(32)  NOT NULL COMMENT '通知类别（七类枚举，anchor_kind 列位同源）',
  requirement_id VARCHAR(32)  NOT NULL COMMENT '跳转一级定位',
  node_id        VARCHAR(32)  NULL     COMMENT '跳转二级定位；会话类可空',
  target_id      VARCHAR(64)  NULL     COMMENT '锚点增强（retest=issueId / task 类=taskId）',
  recipient_role VARCHAR(32)  NOT NULL COMMENT '收件角色',
  sender         VARCHAR(64)  NOT NULL DEFAULT 'plugin' COMMENT '派发者；空=plugin',
  sender_role    VARCHAR(32)  NOT NULL DEFAULT 'plugin' COMMENT '派发者角色；空=plugin',
  text           TEXT         NOT NULL COMMENT '通知文案',
  status         VARCHAR(16)  NOT NULL DEFAULT 'pending' COMMENT 'pending/sending/sent/failed/dead',
  retry_count    INT          NOT NULL DEFAULT 0 COMMENT '补投次数；上限 5',
  last_error     TEXT         NULL COMMENT '最近一次推送失败原因',
  created_at     BIGINT       NOT NULL COMMENT '创建时间，unix 毫秒',
  updated_at     BIGINT       NOT NULL COMMENT '更新时间，unix 毫秒（认领/停滞判定基准）',
  sent_at        BIGINT       NULL COMMENT '推送成功时间；NULL=未送达',
  INDEX idx_outbox_status (status, updated_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
  COMMENT='通知可靠投递 outbox（业务事务内 enqueue，flush/scanner 推送至 issues 通知行）';
