# 团队项目工作台 · 文档体系

> 目录标准：版本 → 需求 → 角色；设计文档按版本归档；产物模板全局共享。
> 标准细则见 `v1.0/arch_design.md` §10；本文展示当前实际落位的目录结构。

## 目录结构（当前实际）

```
docs/
├── v1.0/                            # v1.0 设计文档（团队项目工作台平台自身）
│   ├── arch_design.md               # 版本架构设计总集（架构/数据/制度/执行/规范/里程碑，§7~§11）
│   ├── roadmap.md                   # 版本规划（v1.0 MVP / v1.1 增强）
│   ├── diagrams/                    # 架构图（可交互 HTML + 源 JSON）
│   ├── 01-module-board/                # 模块 I：版本看板 / 需求池 / 需求管理 / 版本管理
│   │   ├── prd/prd.md                 # 产品需求文档
│   │   ├── rd/design.md             # RD 技术设计（数据结构/审计/版本状态机/RPC）
│   │   └── fe/design.md             # FE 技术设计（看板/列表/弹窗/筛选）
│   ├── 02-module-flow-tree/            # 模块 II：需求 OA 流程树（评审/锚点/门禁）
│   │   ├── prd/prd.md                 # 产品需求文档
│   │   ├── rd/design.md             # RD 技术设计（门禁判定/评论状态机/锚点防漂移/RPC）
│   │   └── fe/design.md             # FE 技术设计（流程树/阅读器/锚点交互/对话区）
│   ├── 03-module-inbox-todo/           # 模块 III/IV：门禁通知收件箱 / TODO
│   │   ├── prd/prd.md                 # 产品需求文档
│   │   ├── rd/design.md             # RD 技术设计（查询投影/断点续接/待回复推导）
│   │   └── fe/design.md             # FE 技术设计（badge/收件箱列表/TODO dock）
│   └── _archive/                    # 已废弃归档的旧稿（01-09，内容已并入上文/合并到 arch_design）
│       ├── 01-architecture.md
│       ├── 02-module-board.md
│       ├── 03-module-flow-tree.md
│       ├── 04-module-inbox-todo.md
│       ├── 05-domain-model.md
│       ├── 06-flows-and-roles.md
│       ├── 07-tools-and-orchestration.md
│       ├── 08-doc-standards.md
│       └── 09-milestones.md
├── templates/                       # 产物模板（与 capabilities 挂接，全局共享）
│   ├── prd-template.md              # PRD 模板（11 节）
│   ├── backend-design-doc-template.md
│   ├── frontend-design-doc-template.md
│   ├── arch-version-design-template.md  # 版本架构模板（12 节）
│   ├── arch-version-scope-align-template.md
│   ├── research-open-source-tech-template.md
│   ├── qa-acceptance-template.md
│   ├── qa-testcase-review-template.md
│   ├── qa-defect-management-template.md
│   └── *-demo.md                    # 各模板填充示例
└── README.md                        # 本文件
```

## 阅读路径（v1.0 设计文档）

| 顺序 | 文档 | 内容 |
|---|---|---|
| 1 | `v1.0/arch_design.md` | 版本架构总集：§1-§6 总纲/决策/模块/需求拆解 |
| 2 | `v1.0/arch_design.md` §7 | 数据层（领域模型与存储） |
| 3 | `v1.0/arch_design.md` §8 | 制度层：流程模板 / 角色 / 能力 |
| 4 | `v1.0/arch_design.md` §9 | 执行层：工具 / 编排 |
| 5 | `v1.0/arch_design.md` §10 | 文档规范：目录 / 模板 / 门禁 |
| 6 | `v1.0/arch_design.md` §11 | 里程碑与落地状态 |
| 7 | `v1.0/roadmap.md` | 版本规划：目标/范围/需求清单 |

**需求级（按模块）**：

| 模块 | PRD | RD 设计 | FE 设计 |
|---|---|---|---|
| I 看板/需求/版本 | `01-module-board/prd/prd.md` | `01-module-board/rd/design.md` | `01-module-board/fe/design.md` |
| II OA 流程树 | `02-module-flow-tree/prd/prd.md` | `02-module-flow-tree/rd/design.md` | `02-module-flow-tree/fe/design.md` |
| III/IV 收件箱/TODO | `03-module-inbox-todo/prd/prd.md` | `03-module-inbox-todo/rd/design.md` | `03-module-inbox-todo/fe/design.md` |

## 术语表

| 术语 | 含义 |
|---|---|
| DSH | DeepSeek Harness，本工作台的运行平台 |
| 需求（Requirement） | 最小工作单元，`category` 决定走哪类流程 |
| 流程（Flow） | 需求对应的流程实例，由「待启动 + N 个节点」组成 |
| 评论/问题（Issue） | 统一模型：一切评论都视为待解决问题 |
| 能力（Capability） | 角色的专业行为单元（role→capability 1 对多） |
| wb_* 工具 | 领域模型工具（22 个），agent 与 UI 共用同一套领域逻辑 |
