// Package handler Gin 路由薄壳：参数绑定 + 日志（入口/出口埋点）+ 转发 service。
// 本文件：路由注册总入口。
package handler

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/huzilin/wbflow/rd/service"
)

// Deps handler 层依赖集合。
type Deps struct {
	Logger *zap.Logger
	Health service.HealthChecker
}

// NewRouter 注册全部路由（含全局中间件）。
func NewRouter(deps *Deps) *gin.Engine {
	r := gin.New()
	r.Use(AccessLog(deps.Logger), Recovery(deps.Logger))

	r.GET("/healthz", Healthz(deps.Health))
	return r
}
