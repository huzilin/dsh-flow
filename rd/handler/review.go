// Package handler Gin 路由薄壳。本文件：评审表态与 User 拍板端点
// （wb_review / wb_decision，T8）——参数绑定 + 转发 service，业务编排一律在
// service 层。
package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/huzilin/wbflow/rd/api"
)

// ReviewAPI 评审门禁端点对 service 的依赖接口（consumer 侧定义，
// service.ReviewService 天然满足，便于 httptest 打桩）。
type ReviewAPI interface {
	// Review wb_review（agent 表态 + collect 聚合门禁推进）
	Review(ctx context.Context, req *api.ReviewReq) (*api.ReviewResp, error)
	// Decide wb_decision（User 终审拍板专用路由）
	Decide(ctx context.Context, req *api.DecisionReq) (*api.DecisionResp, error)
}

// RegisterReviewRoutes 注册评审门禁路由（group 形如 /api/requirements，
// 与 artifacts 端点同组挂载节点子路径）。
func RegisterReviewRoutes(group *gin.RouterGroup, svc ReviewAPI) {
	group.POST("/:reqId/nodes/:nodeId/review", Review(svc))
	group.POST("/:reqId/nodes/:nodeId/decision", Decide(svc))
}

// Review POST /api/requirements/:reqId/nodes/:nodeId/review（body 绑定，路径 id 优先）。
func Review(svc ReviewAPI) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := &api.ReviewReq{}
		if err := bindProtoBody(c, req); err != nil {
			AbortWithErr(c, err)
			return
		}
		req.RequirementId = c.Param("reqId")
		req.NodeId = c.Param("nodeId")
		resp, err := svc.Review(c.Request.Context(), req)
		if err != nil {
			AbortWithErr(c, err)
			return
		}
		respondProto(c, http.StatusOK, resp)
	}
}

// Decide POST /api/requirements/:reqId/nodes/:nodeId/decision（body 绑定，路径 id 优先）。
func Decide(svc ReviewAPI) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := &api.DecisionReq{}
		if err := bindProtoBody(c, req); err != nil {
			AbortWithErr(c, err)
			return
		}
		req.RequirementId = c.Param("reqId")
		req.NodeId = c.Param("nodeId")
		resp, err := svc.Decide(c.Request.Context(), req)
		if err != nil {
			AbortWithErr(c, err)
			return
		}
		respondProto(c, http.StatusOK, resp)
	}
}
