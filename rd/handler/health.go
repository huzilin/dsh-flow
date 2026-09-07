// Package handler Gin 路由薄壳。本文件：healthz 健康检查端点。
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/huzilin/wbflow/rd/service"
)

// Healthz GET /healthz
//
// 返回服务状态 + MySQL 连通状态（服务器可达 / dbname 存在可区分上报）。
// 进程存活即 200（liveness 口径），数据库降级在出参 state/db_state 体现。
func Healthz(health service.HealthChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, health.Check(c.Request.Context()))
	}
}
