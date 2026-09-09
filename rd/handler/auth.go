// Package handler Gin 路由薄壳。本文件：简单 token 认证中间件
// （spec §1.1：本期本地部署简单 token；DSH 嵌入期替换为 DSH session 适配器，
// API 路由不变）。
package handler

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/huzilin/wbflow/rd/pkg/errs"
)

// bearerPrefix Authorization 头前缀。
const bearerPrefix = "Bearer "

// TokenAuth Bearer token 校验：Authorization: Bearer <token>，失败 401。
//
// 口径：
//   - token 未配置（空串）= fail closed：全部请求 401（安全默认，防裸奔部署）；
//   - 比对用 constant-time（subtle.ConstantTimeCompare）防时序侧信道；
//   - /healthz 不挂本中间件（探活豁免，路由注册层面区分）。
func TokenAuth(token string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if token == "" {
			unauthorized(c, "auth token 未配置（fail closed，设 WBFLOW_AUTH_TOKEN 后可用）")
			return
		}
		authHeader := c.GetHeader("Authorization")
		if !strings.HasPrefix(authHeader, bearerPrefix) ||
			subtle.ConstantTimeCompare([]byte(strings.TrimPrefix(authHeader, bearerPrefix)), []byte(token)) != 1 {
			unauthorized(c, "无效或缺失 Bearer token")
			return
		}
		c.Next()
	}
}

// unauthorized 401 出参（api.ErrorResp 契约，code=UNAUTHORIZED）。
func unauthorized(c *gin.Context, msg string) {
	writeErrJSON(c, http.StatusUnauthorized, errs.CodeUnauthorized, msg, false, nil)
}
