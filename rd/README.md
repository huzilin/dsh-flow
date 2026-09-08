# rd · 后端（Go + Gin + MySQL 8.0）

wbflow 后端 root。技术规格：`.plan/wbflow-impl/spec.md`。

## 分层（Handler 模式）

```
cmd/rd/     入口（main：配置加载 → 路由注册 → 优雅退出）
handler/    Gin 路由薄壳：参数绑定 + 入口日志 + 转发 service（每个 wb_* 端点一个 Handler）
service/    业务编排：判据校验、collect 聚合、门禁公式、幂等、审计（GoDoc 带错误集）
domain/     状态机纯逻辑：评审轮聚合判定 / issue 状态机 v2 / 门禁公式 / 结论协议装配（无 IO 可单测）
repo/       GORM MySQL：PreparedStmt + SkipDefaultTransaction（写路径显式事务 + 行锁）
configs/    yaml 配置
pkg/utils/  通用工具（TEXT 列 JSON 序列化 helper、StrPtr/BoolToInt64 标量微 helper——收敛于此，业务代码不内联）
```

## 约定（详见 spec.md）

- 错误：errors.Trace 包裹 + 哨兵错误（ErrGateBlocked/ErrOptimisticLock/ErrStaleReport…），HTTP 出参集中映射，接口永不 panic
- 事务：写路径单事务 + flow_nodes 行锁；通知派发移出事务
- 幂等键：派发 reqId+nodeId+action_id+assignee；回退 reqId+nodeId+round
- 类型一致：any 不用 interface{}；int 统一 int64；结构体入参一律指针；json tag 与 field name 一致
