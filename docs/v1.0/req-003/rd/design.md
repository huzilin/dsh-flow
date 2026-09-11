# 《评审通知复现需求》RD 技术设计

> 联调样例文档（BUG-015 数据修复入库，2026-09-10）。

## 1. 方案概述

本需求不新增服务端能力，全部复用既有编排与通知链路；技术设计仅约定样例推进的构造方式与观测点。

## 2. 数据设计

- 复用 requirements / flows / flow_nodes 既有表，不新增字段。
- 通知观测点：notifications 表按 anchor_kind 与 target_id 过滤。

## 3. 接口设计

| 步骤 | 接口 | 断言 |
|---|---|---|
| 提评 | POST /comments | issue state=open |
| 表态 | wb_review | collect 判定正确 |
| 拍板 | wb_decision | last_decision 落值 |
| 上升 | wb_escalate | session 建绑 |

## 4. 关键时序

1. pm 登记 brd → advance → 停等用户确认；
2. 产品设计登记 prd → 提评 → 三位逐位 agree → gate 停等；
3. RD 技术设计登记 → 提评 → 四位 agree → 拍板迁移。

## 5. 风险与对策

- 风险：样例推进与 scanner 巡检竞速。对策：回退窗口内不依赖巡检推进（BUG-009 口径）。
