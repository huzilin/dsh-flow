# 《逐步推进验证需求（正常流程构造）》产品设计文档（PRD）

> 联调样例文档（BUG-015 数据修复入库，2026-09-10）。

## 1. 问题陈述

BUG-001 复现需要逐节点推进的真实编排数据；速推样本无法区分「状态未联动」与「数据伪影」。

## 2. 目标

提供一条七步推进样本：待启动 → 需求对齐 → 产品设计 → 产品评审 → RD 技术设计 → RD 技术评审停等。

## 3. 功能要点

1. 建 business×full 需求（状态「待启动」）。
2. 待启动节点拍板 → 对齐节点 to-brd 落盘 + 拍板。
3. 产品设计 produce 登记 brd/product_doc → 提评 → arch/rd/fe 逐位 agree → 拍板。
4. RD 技术设计登记 rd_design → 提评 → arch/fe/qa/pm 逐位 agree，停等。

## 4. 观测口径

- 每步 `GET /api/requirements/:id` 快照；
- audit_log 中 requirements 表应仅有 INSERT（BUG-001 修复前）或含 status UPDATE（修复后）。
