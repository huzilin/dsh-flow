// Package handler Gin 路由薄壳。本文件：冲突上升裁决链端点
// （wb_escalate / wb_adjudicate，T12）——参数绑定 + 转发 service，业务编排与
// 裁决状态机判定一律在 service 层。
package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/huzilin/wbflow/rd/api"
)

// EscalationAPI 冲突上升裁决链端点对 service 的依赖接口（consumer 侧定义，
// service.EscalationService 天然满足，便于 httptest 打桩）。
type EscalationAPI interface {
	// Escalate wb_escalate（手动上升 + 拉起裁决 session）
	Escalate(ctx context.Context, req *api.EscalateReq) (*api.EscalateResp, error)
	// Adjudicate wb_adjudicate（User 仲裁结论回填 + reviewer 重表态）
	Adjudicate(ctx context.Context, req *api.AdjudicateReq) (*api.AdjudicateResp, error)
}

// RegisterEscalationRoutes 注册冲突上升裁决链路由（group 形如 /api/comments，
// 与评论线程侧四端点同组挂载）。
func RegisterEscalationRoutes(group *gin.RouterGroup, svc EscalationAPI) {
	group.POST("/:commentId/escalate", Escalate(svc))
	group.POST("/:commentId/adjudicate", Adjudicate(svc))
}

// Escalate POST /api/comments/:commentId/escalate（body 绑定，路径 id 优先）。
func Escalate(svc EscalationAPI) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := &api.EscalateReq{}
		if err := bindProtoBody(c, req); err != nil {
			AbortWithErr(c, err)
			return
		}
		req.CommentId = c.Param("commentId")
		resp, err := svc.Escalate(c.Request.Context(), req)
		if err != nil {
			AbortWithErr(c, err)
			return
		}
		respondProto(c, http.StatusOK, resp)
	}
}

// Adjudicate POST /api/comments/:commentId/adjudicate（body 绑定，路径 id 优先）。
func Adjudicate(svc EscalationAPI) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := &api.AdjudicateReq{}
		if err := bindProtoBody(c, req); err != nil {
			AbortWithErr(c, err)
			return
		}
		req.CommentId = c.Param("commentId")
		resp, err := svc.Adjudicate(c.Request.Context(), req)
		if err != nil {
			AbortWithErr(c, err)
			return
		}
		respondProto(c, http.StatusOK, resp)
	}
}
