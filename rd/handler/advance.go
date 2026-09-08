// Package handler Gin 路由薄壳。本文件：wb_advance 编排推进端点
// （2026-09-09 拍板-B，spec §1.2 注记-B/C）——参数绑定 + 转发 service。
package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/huzilin/wbflow/rd/api"
)

// AdvanceAPI 编排推进端点对 service 的依赖接口（consumer 侧定义，
// service.AdvanceService 天然满足，便于 httptest 打桩）。
type AdvanceAPI interface {
	// Advance wb_advance（编排推进：跨节点 / 节点内 CR 轮转 / 机械收口，幂等）
	Advance(ctx context.Context, req *api.AdvanceReq) (*api.AdvanceResp, error)
}

// RegisterAdvanceRoutes 注册编排推进路由（group 形如 /api/requirements，
// 与 review/decision/artifacts 同挂节点子路径）。
func RegisterAdvanceRoutes(group *gin.RouterGroup, svc AdvanceAPI) {
	group.POST("/:reqId/nodes/:nodeId/advance", Advance(svc))
}

// Advance POST /api/requirements/:reqId/nodes/:nodeId/advance
// （body 绑定，路径 id 优先；错误映射沿用 errs.MapToHTTP）。
func Advance(svc AdvanceAPI) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := &api.AdvanceReq{}
		if err := bindProtoBody(c, req); err != nil {
			AbortWithErr(c, err)
			return
		}
		req.RequirementId = c.Param("reqId")
		req.NodeId = c.Param("nodeId")
		resp, err := svc.Advance(c.Request.Context(), req)
		if err != nil {
			AbortWithErr(c, err)
			return
		}
		respondProto(c, http.StatusOK, resp)
	}
}
