# tech / task / plan 三流程 · 逐节点推演表 v2.1（举一反三）

> **状态**：2026-09-05 **定稿**（决策点 D1-D5 已裁决，见 §5；⚠︎ 行已定稿）。模型权威见 `capability-model-final.md` v2.1；挂载映射见 `capability-actions-final.md`。business 已定稿（`business-flow-derivation.md`）；本文件把同一套 8-trigger 词汇表 + 轮次制循环 + 推进契约套到其余三流程，并完成跨流程复用验证（工单 03 第 4 项）。节点结构基准 = flow-alignment.md（tech 删用例两节点 13→11）。

---

## 通用复用原则（三流程与 business 的差异只有三处）

1. **节点类型归入同一四类**：产出型 / 评审型 / 交互型 / 门禁型——动作集、轮次循环、推进契约、会话通道全部复用 business 定稿，无流程专属机制。
2. **scope 裁剪**：business/tech 有 scope（be/fe，按需节点与评审位随 scope 存在）；task 执行者单角色（rd|fe by scope）；plan 无 scope。
3. **inputs 跨流程动态注入**：capability.inputs[] 是**跨流程可见性上限**（声明能力可能消费的产物集合）；流程实例化时按该节点实际可用产物注入 my_task 上下文——如 backend-design 在 business 拿 product_doc、在 tech 拿 brd，skill 依赖动态取用。

---

## 1. tech 技术需求流程（11 节点：删用例设计/用例设计评审）

```
B0 待启动 → B1 需求对齐 → B2 RD技术设计 → B3 RD技术评审(按需) → B4 FE技术设计(按需)
→ B5 FE技术评审(按需) → B8a RD开发 ∥ B8b FE开发 → B9 QA验收 → B10 用户验收 → B11 完成
```

| 节点 | 类型 | 能力/动作挂载 | 评审位 / 门禁 | 复用说明 |
|---|---|---|---|---|
| B0 待启动 | 门禁型 | —— | User 确认投入资源 → approve | 同 business 节点 0 |
| B1 需求对齐 | 交互型 | align.opening(entry) + 自由对话 + **to-brd(exec)** + approve | 响应者=PM；完成=用户告知 PM→上报 | ✅D1：统一产 brd（喂 rd_design）；**to-brd=公共 skill**（对齐节点 node action 挂载，不属 pm 能力，business/tech 复用） |
| B2 RD技术设计 | 产出型 | backend-design(rd)：produce(exec→to-rd-techdesign) / submit / clarify / fix_on_review | scope=be | inputs 动态：brd（tech）/ product_doc（business） |
| B3 RD技术评审(按需) | 评审型 | arch / rd(自评) / fe / qa：review(after-exec) / confirm(replied)；final-approve(approve) | **无 pm 位**（business 节点 5 有）+ User 拍板 | 评审位集合随流程配置；轮次制同 |
| B4 FE技术设计(按需) | 产出型 | frontend-design(fe)：同 B2 模式 | scope=fe | — |
| B5 FE技术评审(按需) | 评审型 | qa-tech-review：review / confirm | qa 单评审位→**机械门禁**（同 business 节点 7） | — |
| B8a/B8b 并行开发 | 产出型+内嵌评审 | backend-dev / frontend-dev：dev / submit / respond_to_cr / fix_on_review；code-review：review / confirm | CR 机械门禁；并行组 join；reject(targetBranch) 部分回滚 | 同 business 节点 8/9 |
| B9 QA验收 | 产出型 | acceptance-test：accept / submit；rd/fe：clarify（缺陷确认/归属争议）/ fix_on_review | 轮次制；机械 reject（qa 分派 targetBranch） | 同 business 节点 10 |
| B10 用户验收 | 门禁型+会话通道 | final-approve(approve)；会话通道绑 qa | User 终拍板 | 同 business 节点 11 |
| B11 完成 | 门禁型 | —— | DONE 占位 | — |

## 2. task 任务流程（8 节点）

```
C0 待启动 → C1 需求对齐 → C2 技术设计 → C3 技术评审 → C4 开发 → C5 QA验收 → C6 用户验收 → C7 完成
```

| 节点 | 类型 | 能力/动作挂载 | 评审位 / 门禁 | 复用说明 |
|---|---|---|---|---|
| C0 待启动 | 门禁型 | —— | User 确认 → approve | — |
| C1 需求对齐 | 交互型 | align.opening(entry) + 自由对话 + approve | 响应者=**执行者**（rd|fe by scope）；**无产物** | ✅D2：维持无产物——**C2 技术设计的基础=C1 对齐简单描述**（无 prd/brd，mytask 上下文注入） |
| C2 技术设计 | 产出型 | **plan-cap**(执行者实例化)：produce(exec→to-plan) / submit / clarify / fix_on_review | — | plan-cap 跨流程实例化：role=执行者（rd|fe by scope）；**基础=C1 简单描述**（✅D2） |
| C3 技术评审 | 评审型 | arch-tech-review：review / confirm；final-approve(approve) | arch 单评审位 + **User 拍板**（✅D4 对照：任务级唯一方向确认点，保留 User） | — |
| C4 开发 | 产出型+内嵌评审 | backend-dev 或 frontend-dev（scope 单分支）：dev / submit / respond_to_cr / fix_on_review；code-review：review / confirm | CR 机械门禁；无并行组（单分支） | business 节点 8 的单分支退化形 |
| C5 QA验收 | 产出型 | acceptance-test：accept / submit；执行者：clarify / fix_on_review | 轮次制；机械 reject（qa 分派，单分支直接 re-entry） | — |
| C6 用户验收 | 门禁型+会话通道 | final-approve(approve)；会话通道绑 qa | User 终拍板 | — |
| C7 完成 | 门禁型 | —— | DONE 占位 | — |

## 3. plan 版本规划流程（9 节点）

```
D0 待启动 → D1 立项需求对齐 → D2 立项评审 → D3 调研 → D4 调研评审 → D5 架构规划 → D6 架构规划评审 → D7 创建需求·启动版本 → D8 完成
```

| 节点 | 类型 | 能力/动作挂载 | 评审位 / 门禁 | 复用说明 |
|---|---|---|---|---|
| D0 待启动 | 门禁型 | —— | User 确认 → approve | — |
| D1 立项需求对齐 | 交互型 | align.opening(entry) + 自由对话（用户可调多 skills：wayfinder/改进项/遗留 TODO/使用发现）+ **to-alignment(exec，公共 skill)** | 响应者=**arch**；✅D3：交互对齐+**to-alignment 汇总成明确文档**（对齐结论+对话中发现的改进项/TODO/需求） | 产出 alignment（版本方向文档）；to-alignment=公共 skill（node action，同 to-brd） |
| D2 立项评审 | 评审型 | **alignment-cap(arch)**：clarify(reviewed) / fix_on_review(re-exec)——评审循环响应；pm-review：review / confirm；final-approve(approve) | PM 评审 + User 拍板 | 代笔产物（arch 写）需 PM 把关——判据=产出者非 User；**alignment-cap 收缩为评审响应能力**（produce 由 node action to-alignment 承接） |
| D3 调研 | 产出型 | research-cap(inv)：produce(exec→research 已有) / submit / clarify / fix_on_review | — | — |
| D4 调研评审 | 评审型 | arch-tech-review：review / confirm | arch 单评审位→**机械门禁**（✅D4：用户不参与；同节点 7/10 先例） | 打回→D3 re-entry |
| D5 架构规划 | 产出型 | arch-cap(arch)：produce(exec→to-arch) / submit / clarify / fix_on_review | — | 产出 architecture（版本级） |
| D6 架构规划评审 | 评审型 | pm-review：review / confirm；final-approve(approve) | PM + User 拍板 | — |
| D7 创建需求·启动版本 | 产出型（exec） | decide-cap(leader)：**create_reqs(exec→to-tickets)** | ——（程序推进版本） | create_reqs 的唯一挂载点 |
| D8 完成 | 门禁型 | —— | DONE 占位 | — |

---

## 4. 跨流程复用验证矩阵（18 能力 × 4 流程）

| 能力 | business | tech | task | plan |
|---|---|---|---|---|
| product-design | 节点 2 | — | — | — |
| backend-design | 节点 4 | B2 | — | — |
| frontend-design | 节点 6 | B4 | — | — |
| research-cap | — | — | — | D3 |
| arch-cap | — | — | — | D5 |
| plan-cap | — | — | C2 | — |
| alignment-cap | — | — | — | D1 |
| backend-dev | 节点 8 | B8a | C4(scope=rd) | — |
| frontend-dev | 节点 9 | B8b | C4(scope=fe) | — |
| pm-review | 节点 3/5 | — | — | D2/D6 |
| arch-tech-review | 节点 3/5 | B3 | C3 | D4 |
| rd-tech-review | 节点 5(自评+评审) | B3 | — | — |
| fe-tech-review | 节点 5 | B3 | — | — |
| qa-tech-review | 节点 5/7 | B3/B5 | — | — |
| code-review | 节点 8/9 | B8a/B8b | C4 | — |
| acceptance-test | 节点 10 | B9 | C5 | — |
| decide-cap | ——(join) | ——(join) | — | D7(create_reqs) |
| final-approve | 节点 3/5/11 | B3/B10 | C3/C6 | D2/D6 |

**验证结论**：① 18 能力 × 4 流程全覆盖，无流程专属能力、无流程专属动作——动作词汇表（8 trigger + 挂载表）四流程统一；② 差异仅在**节点组合**（流程拓扑）与**评审位集合**（如 tech B3 无 pm、business 节点 5 有）与 **scope 裁剪**；③ 机制层（轮次循环/推进契约/会话通道/兜底扫描）流程无关；④ inputs 跨流程动态注入成立（inputs[]=可见性上限，实际按流程可用产物注入 my_task）。

## 5. 决策点裁决记录（2026-09-05 用户批注，全部定稿）

| # | 决策 | 裁决 |
|---|---|---|
| D1 | tech B1 对齐产物 | **统一产 brd**；to-brd=**公共 skill**（对齐节点 node action 挂载，不属 pm 能力，business/tech 复用） |
| D2 | task C1 对齐 | **维持无产物**；C2 技术设计基础=C1 **对齐简单描述**（无 prd/brd，mytask 上下文注入） |
| D3 | plan D1 形态 | **交互对齐+固化**——用户参与重（对话中可调多 skills：wayfinder/改进项/遗留 TODO/使用发现），**to-alignment=公共 skill** 汇总成明确文档；连带：**alignment-cap 收缩为 D2 评审响应能力**（clarify/fix_on_review，produce 由 node action 承接） |
| D4 | D4 调研评审 | **维持 arch 评审，用户不参与**（机械门禁）；对照 C3 保留 User 拍板=任务级唯一方向确认点 |
| D5 | inputs 动态注入 | **确认**——写入各 skill（skill 中写明上下文依赖来源）；**skill 实现阶段需逐个校验，防止遗漏** |
