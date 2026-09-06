# 团队项目工作台 · 文档地图（docs-tmp 重构版）

> 本目录是设计文档体系的重构产物（暂存版，经确认后替换 `docs/design/`）。
> 重构目标：**一份整体架构 + 按模块组织的需求与设计**；去掉多轮交互产生的版本历史、重复描述与已废弃方案，只保留定稿结论。

## 阅读路径

| 顺序 | 文档 | 内容 | 取代（旧文件） |
|---|---|---|---|
| 0 | `README.md`（本文件） | 文档地图、术语表 | — |
| 1 | `01-architecture.md` | **整体架构**：定位、设计原则、四层架构、四模块、关键决策 | `team-workbench.md` |
| 2 | `02-module-board.md` | **模块 I**：版本看板 / 需求池 / 需求管理（CRUD+审计）/ 版本管理 | `m3-board*.md` + `m4-req-crud*.md` + `m5-version*.md` |
| 3 | `03-module-flow-tree.md` | **模块 II**：需求 OA 流程树 / 评审闭环 / 产物阅读器 / 锚点评论 | `m2-flow-tree*.md` + `m7-comment-design.md` |
| 4 | `04-module-inbox-todo.md` | **模块 III/IV**：收件箱 / TODO | `team-workbench.md` §2.3–2.4、3.4（收编） |
| 5 | `05-domain-model.md` | 领域模型与存储：实体 / 状态机 / sqlite 11 表 / 存储演进 | `m2-flow-tree-tech.md` §3 + `m8-flow-simplify.md` §3.6 |
| 6 | `06-flows-and-roles.md` | **制度层**：4 类流程模板 / 9+1 角色 / 19 能力 / 评审分工 | `m6-team-preset.md` + `m6-flow-templates.md` + `m8` §2.4/§10 |
| 7 | `07-tools-and-orchestration.md` | **执行层**：22 个 wb_* 工具 / agent 编排协议 / 协作铁律 | `m6-agent-tools.md` + `m6-gap-closure.md` + `m7-agent-protocol.md` + `m8` §2/§4/§11 |
| 8 | `08-doc-standards.md` | 文档规范：目录结构 / 产物模板 / 质量门禁 | `m7-doc-structure.md` + `m8-doc-templates.md` |
| 9 | `09-milestones.md` | 里程碑定义与落地状态 | `team-workbench.md` §5 + 实现记录 |
| — | `diagrams/team-workbench-architecture.html` | **架构图**（可交互 HTML，浏览器打开；源 JSON 见 `diagrams/architecture.json`） | — |

## 术语表

| 术语 | 含义 |
|---|---|
| DSH | DeepSeek Harness，本工作台的运行平台 |
| 工作台（workbench） | 本系统整体：「项目管理 + OA 流程 + 评审协作」三合一交互系统 |
| 项目 | DSH 的一个 workspace（目录），各自一套独立工作台数据 |
| 需求（Requirement） | 最小工作单元，`category` 决定走哪类流程（需求任务/任务/版本规划任务） |
| 流程（Flow） | 需求对应的流程实例，由「待启动 + N 个节点」组成 |
| 节点（Node） | 一条流程标签一个独立节点，可独立回退重入 |
| 门禁（Gate） | 需评审通过的节点；User 终审 = 通过/打回 |
| 评论/问题（Issue） | 统一模型：一切评论都视为待解决问题，有状态与轮次 |
| 锚点（Anchor） | 评论定位到产物文档的位置：文档级 / 行级 / 范围级 / 块级 |
| 产物（Artifact） | 节点绑定产出物（文档 doc / 链接 link），是评审对象 |
| 能力（Capability） | 角色的专业行为单元（role→capability 1 对多），节点声明用什么能力执行 |
| preset | DSH agent 预设（制度层载体）；team preset = 本团队制度 |
| plugin | DSH 插件（交互层载体）；wbflow = 工作台插件 |
| wb_* 工具 | 领域模型工具（22 个），agent 与 UI 共用同一套领域逻辑 |
| wb_my_task | 确定性任务上下文工具：程序计算「你在哪、看什么、产什么、通知谁」 |

## 写读约定（本目录内）

- **唯一真相源**：实体字段以 `05-domain-model.md` 为准；流程节点以 `06-flows-and-roles.md` 为准；工具清单以 `07-tools-and-orchestration.md` 为准；其它文档只引用不重写。
- 模块文档（02/03/04）= 该模块的**需求（交互/验收）＋设计（实现）**合并稿。
- 制度/执行/规范类文档（06/07/08）为跨模块支撑层。
- 所有文档只写**定稿结论**，不保留版本迭代过程；确需说明的演进只留一行「演进注记」。