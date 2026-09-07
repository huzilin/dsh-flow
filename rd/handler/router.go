// Package handler Gin 路由薄壳：参数绑定 + 日志（入口/出口埋点）+ 转发 service。
// 本文件：路由注册总入口（/healthz 豁免认证；/api 组挂 token 中间件）。
package handler

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/huzilin/wbflow/rd/service"
)

// Deps handler 层依赖集合。Reqs/Vers 为 nil 时服务降级（仅 /healthz 可用，
// API 路由不注册——与 DB 未就绪的降级启动口径一致）。
type Deps struct {
	Logger *zap.Logger
	Health service.HealthChecker
	// Token Bearer 令牌（config.Auth.Token）；空=fail closed（API 全 401）
	Token string
	// Reqs requirements 五端点 service；nil=不注册对应路由
	Reqs RequirementAPI
	// Vers versions 五端点 service；nil=不注册对应路由
	Vers VersionAPI
	// Flows 流程实例端点（wb_my_task / 依赖树）service；nil=不注册对应路由
	Flows FlowAPI
	// Comments 评论/issue 五端点 service；nil=不注册对应路由
	Comments CommentAPI
}

// NewRouter 注册全部路由（含全局中间件）。
func NewRouter(deps *Deps) *gin.Engine {
	r := gin.New()
	r.Use(AccessLog(deps.Logger), Recovery(deps.Logger))

	// /healthz 探活豁免 token（认证中间件只挂 /api 组）
	r.GET("/healthz", Healthz(deps.Health))

	api := r.Group("/api")
	api.Use(TokenAuth(deps.Token))
	if deps.Reqs != nil {
		RegisterRequirementRoutes(api.Group("/requirements"), deps.Reqs)
	}
	if deps.Vers != nil {
		RegisterVersionRoutes(api.Group("/versions"), deps.Vers)
	}
	if deps.Flows != nil {
		RegisterFlowRoutes(api.Group("/requirements"), deps.Flows)
	}
	if deps.Comments != nil {
		RegisterIssueRoutes(api.Group("/requirements"), api.Group("/comments"), deps.Comments)
	}
	return r
}
