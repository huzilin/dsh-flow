# DSH 插件安装 / 启动相关知识：pnpm 构建脚本拦截（allowBuilds）故障复盘

> 场景：`dsh plugin --profile web add git+https://...` 安装 git-hosted 插件时报
> `ERR_PNPM_GIT_DEP_PREPARE_NOT_ALLOWED`，要求把包加入 `allowBuilds`。
> 本文固化根因、机制、解法与验证方法，供后续分析同类问题使用。

---

## 1. DSH 插件管理机制（本质是 pnpm 转发器）

- 入口命令：`dsh plugin --profile <name> <args...>`
- 在 DSH checkout 下通过 `pnpm dsh` 运行，等价于
  `node --import tsx/esm apps/cli/src/bin.ts plugin --profile <name> ...`
- 核心实现见 `apps/cli/src/plugin.ts`：
  - `runPlugin()` 本质上只是 `spawnSync('pnpm', args, { cwd: profileDir })` —— 把参数**原样转发**给 pnpm。
  - `cwd` = profile 目录（见下）。
  - pnpm 成功后调用 `reconcilePlugins()`：根据"已安装的依赖是否声明 `dsh.bundle.patch`"来维护 `dsh.profile.bundles`（layer 列表）。
  - 若 pnpm 失败且参数匹配 `git+` / `github:` / `.git`，会提示去 profile 的 `pnpm-workspace.yaml` 里加 `allowBuilds`。
- 本质结论：**DSH 不干预 pnpm 的构建审批**，问题完全由 pnpm 版本行为引起。

## 2. Profile 目录结构

- 位置：`~/.dsh/profiles/<name>/`（本机示例：`/Users/huzilin/.dsh/profiles/web/`）
- 关键文件：
  - `package.json` —— 依赖列表 + `dsh.profile.bundles`（已激活的 layer）
  - `pnpm-workspace.yaml` —— pnpm 配置（`packages`、`nodeLinker`、`allowBuilds`、`dangerouslyAllowAllBuilds` 等）
  - `pnpm-lock.yaml`、`node_modules/`
  - `cordis.yml` / `cordis.patch.yml`
- 首次运行某 profile 时若目录无 `package.json`，DSH 会 `initProfile()` 初始化。

## 3. pnpm 构建脚本审批（根因）

- **触发条件**：pnpm ≥ 10 默认**只允许"已信任"的依赖运行构建脚本**（`preinstall` / `install` / `postinstall` / `prepare`）。
- git-hosted 插件（`git+https://...`、`github:owner/repo`）安装时通过 `prepare` 脚本**构建**（如 tsdown/vite 打包），因此被 pnpm 视为"需要运行构建脚本"，未放行则报：
  - `[ERR_PNPM_GIT_DEP_PREPARE_NOT_ALLOWED] ... needs to execute build scripts but is not in the "allowBuilds" allowlist.`
- 这是 **pnpm 的安全机制**（防供应链脚本攻击），不是 DSH 的 bug。

## 4. allowBuilds 的键匹配规则（关键坑）

- pnpm 11 中 `allowBuilds` 是一个 **map**，键 = 依赖 spec，值 = `true` 放行。
- **对 git 依赖**：pnpm 会把它下载为 **codeload tarball**，因此匹配键必须是**完整 tarball URL（含 commit 哈希）**，例如：
  ```
  dsh-sidebar-qa@https://codeload.github.com/ChenRuoT/dsh-sidebar-qa/tar.gz/<commit-sha>: true
  ```
- **对 registry 依赖**：键是 `<name>@<version>`（如 `dsh-sidebar-qa@0.2.0`）。
- **`"*": true` / `"**": true` 通配不生效** —— 不加引号、写成 `"*": true` 均无法匹配 git-hosted 的 tarball URL。早期有此误解，实际无效。
- 报错信息里 pnpm 会打印它想要的**精确键**，直接复制最稳妥。

## 5. 两类"放行"方案

### 方案 A：完全放开（一行，全局放行所有依赖构建脚本）
```yaml
dangerouslyAllowAllBuilds: true
```
- 生效版本：pnpm ≥ 10.9.0 （官方设置见 https://pnpm.io/settings/build#dangerouslyallowallbuilds）
- 所有依赖的 `preinstall/install/postinstall` 自动运行，不再逐个审批。
- 注意：名字里的 "dangerously" 名副其实 —— 任何第三方包都能直接跑脚本，有供应链风险；不建议长期开启。

### 方案 B：精准白名单（推荐，逐个放行）
```yaml
allowBuilds:
  dsh-sidebar-qa@https://codeload.github.com/ChenRuoT/dsh-sidebar-qa/tar.gz/<commit-sha>: true
  node-pty: true
  cloudflared: true
  cpu-features: true
  ssh2: true
```
- 只放行明确列出的包，其余仍被拦截。
- 维护成本：git 插件更新到新 commit 时，需同步更新键里的 commit 哈希（新安装报错时 pnpm 会打印新键）。

## 6. 验证方法

1. 改完 `pnpm-workspace.yaml` 后重新运行安装命令，不再出现 `ERR_PNPM_GIT_DEP_PREPARE_NOT_ALLOWED` 即成功。
2. 更严谨：`pnpm uninstall dsh-sidebar-qa` + 重新 `pnpm add "github:owner/repo"`，确认 prepare 脚本正常执行。
3. `pnpm rebuild <pkg>` 可用来触发/检查构建脚本是否被批准（纯 JS 包可能是 no-op，输出为空属正常）。
4. 查看已解析配置：
   - `pnpm config get allowBuilds`
   - `pnpm config get dangerouslyAllowAllBuilds`（返回 `undefined` 表示 false/未开启）

## 7. 环境事实（本机，cmacOS arm64）

- pnpm `11.7.0`，node `v25.6.1`，darwin arm64
- registry 指向 `https://registry.npmmirror.com`
- DSH checkkotouri：`/Users/huzilin/workdir/deepseek-harness`
- 仓库 `pnpm dsh` script = `node --import tsx/esm apps/cli/src/bin.ts`
- profile 目录：`/Users/huzilin/.dsh/profiles/`

## 8. 常见误区 / 坑位提醒

- ❌ 以为 `allowBuilds: "*": true` 能通配 —— **不能**，git 依赖需精确 tarball URL。
- ❌ 以为这是 DSH 的 bug，去改 DSH 源码 —— 实为 pnpm 标准安全机制。
- ⚠️ 在这类被沙箱监控的 agent 环境里执行安装会出现 `EPERM ... /Users/huzilin/.dsh/profiles/web/_tmp_...`，那是 **agent 沙箱** 不能写 `~/.dsh` 造成的，**用户本机终端没有此限制**。
- ⚠️ 未来 git 插件升级换 commit 后，精确白名单键要跟着更新。

## 附：推荐长期策略

- 维护阶段用**方案 B（精准白名单）**最安全。
- 若常装各种 git 插件、嫌维护 commit 麻烦，可临时用方案 A，装完再改回。
- 官方参考：pnpm Build Script Security（https://pnpm.io/settings/build）、pnpm v10→v11 迁移文档（https://pnpm.io/zh/migration）。
