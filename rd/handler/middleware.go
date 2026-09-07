// Package handler Gin 路由薄壳。本文件：全局中间件——
// 入口/出口日志埋点、panic 兜底（接口永不 panic）、错误集中映射出参。
package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/huzilin/wbflow/rd/pkg/errs"
)

// ErrorResp 统一失败响应体（spec §1.1：{ok:false, error, code, blocked?, missing?}）。
type ErrorResp struct {
	Ok      bool     `json:"ok"`
	Error   string   `json:"error"`
	Code    string   `json:"code"`
	Blocked bool     `json:"blocked,omitempty"`
	Missing []string `json:"missing,omitempty"`
}

// AccessLog 入口/出口日志埋点：请求进入打一条，响应完成（defer）打一条。
func AccessLog(lg *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		lg.Info("http request start",
			zap.String("path", path),
			zap.String("method", method),
		)

		c.Next()

		lg.Info("http request done",
			zap.String("path", path),
			zap.String("method", method),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", time.Since(start)),
		)
	}
}

// Recovery panic 兜底：任何 handler panic 转统一失败出参，接口永不 panic。
func Recovery(lg *zap.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		lg.Error("http handler panic",
			zap.Any("panic", recovered),
			zap.String("path", c.Request.URL.Path),
		)
		c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResp{
			Ok:    false,
			Error: "internal server error",
			Code:  string(errs.CodeInternal),
		})
	})
}

// AbortWithErr 错误集中映射出参：业务 handler 捕获 service 错误后统一走此写出。
func AbortWithErr(c *gin.Context, err error) {
	mapping := errs.MapToHTTP(err)
	c.AbortWithStatusJSON(mapping.HTTPStatus, ErrorResp{
		Ok:      false,
		Error:   err.Error(),
		Code:    string(mapping.Code),
		Blocked: mapping.Blocked,
	})
}
