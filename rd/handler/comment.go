// Package handler Gin 路由薄壳。本文件：评论/issue 五端点
// （wb_comment / wb_reply / wb_resolve / wb_confirm_review / wb_reopen）——
// 参数绑定 + 转发 service，业务编排与状态机判定一律在 service/domain 层。
package handler

import (
	"net/http"

	"context"
	"github.com/gin-gonic/gin"

	"github.com/huzilin/wbflow/rd/api"
)

// CommentAPI 评论/issue 端点对 service 的依赖接口（consumer 侧定义，
// service.IssueService 天然满足，便于 httptest 打桩）。
type CommentAPI interface {
	// Comment wb_comment（四级锚点评论建行）
	Comment(ctx context.Context, req *api.CommentReq) (*api.CommentResp, error)
	// Reply wb_reply（replies 不可变追加 + roundCount++）
	Reply(ctx context.Context, req *api.ReplyReq) (*api.ReplyResp, error)
	// Resolve wb_resolve（path=fix|agree 双路径）
	Resolve(ctx context.Context, req *api.ResolveReq) (*api.ResolveResp, error)
	// Confirm wb_confirm_review（提评者复核确认）
	Confirm(ctx context.Context, req *api.ConfirmReviewReq) (*api.ConfirmReviewResp, error)
	// Reopen wb_reopen（澄清 reopen / 修复 revise，loop_count++）
	Reopen(ctx context.Context, req *api.ReopenReq) (*api.ReopenResp, error)
	// OverrideRole wb_override_role（注记-L1 强权通道：User 整角色批量置 resolved
	// ——09-12 裁决 overridden 态废除，override 动作=批量 resolve，resolve_path=override）
	OverrideRole(ctx context.Context, req *api.OverrideRoleReq) (*api.OverrideRoleResp, error)
}

// RegisterIssueRoutes 注册评论/issue 路由：requirements 组挂需求侧建评
// （POST /:reqId/comments），comments 组挂线程侧四端点 + 强权通道。
func RegisterIssueRoutes(requirements *gin.RouterGroup, comments *gin.RouterGroup, svc CommentAPI) {
	requirements.POST("/:reqId/comments", Comment(svc))
	comments.POST("/:commentId/replies", Reply(svc))
	comments.POST("/:commentId/resolve", Resolve(svc))
	comments.POST("/:commentId/confirm", Confirm(svc))
	comments.POST("/:commentId/reopen", Reopen(svc))
	comments.POST("/override-role", OverrideRole(svc))
}

// Comment POST /api/requirements/:reqId/comments（body 绑定，路径 id 优先）。
func Comment(svc CommentAPI) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := &api.CommentReq{}
		if err := bindProtoBody(c, req); err != nil {
			AbortWithErr(c, err)
			return
		}
		req.RequirementId = c.Param("reqId")
		resp, err := svc.Comment(c.Request.Context(), req)
		if err != nil {
			AbortWithErr(c, err)
			return
		}
		respondProto(c, http.StatusOK, resp)
	}
}

// Reply POST /api/comments/:commentId/replies。
func Reply(svc CommentAPI) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := &api.ReplyReq{}
		if err := bindProtoBody(c, req); err != nil {
			AbortWithErr(c, err)
			return
		}
		req.CommentId = c.Param("commentId")
		resp, err := svc.Reply(c.Request.Context(), req)
		if err != nil {
			AbortWithErr(c, err)
			return
		}
		respondProto(c, http.StatusOK, resp)
	}
}

// Resolve POST /api/comments/:commentId/resolve。
func Resolve(svc CommentAPI) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := &api.ResolveReq{}
		if err := bindProtoBody(c, req); err != nil {
			AbortWithErr(c, err)
			return
		}
		req.CommentId = c.Param("commentId")
		resp, err := svc.Resolve(c.Request.Context(), req)
		if err != nil {
			AbortWithErr(c, err)
			return
		}
		respondProto(c, http.StatusOK, resp)
	}
}

// Confirm POST /api/comments/:commentId/confirm。
func Confirm(svc CommentAPI) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := &api.ConfirmReviewReq{}
		if err := bindProtoBody(c, req); err != nil {
			AbortWithErr(c, err)
			return
		}
		req.CommentId = c.Param("commentId")
		resp, err := svc.Confirm(c.Request.Context(), req)
		if err != nil {
			AbortWithErr(c, err)
			return
		}
		respondProto(c, http.StatusOK, resp)
	}
}

// Reopen POST /api/comments/:commentId/reopen。
func Reopen(svc CommentAPI) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := &api.ReopenReq{}
		if err := bindProtoBody(c, req); err != nil {
			AbortWithErr(c, err)
			return
		}
		req.CommentId = c.Param("commentId")
		resp, err := svc.Reopen(c.Request.Context(), req)
		if err != nil {
			AbortWithErr(c, err)
			return
		}
		respondProto(c, http.StatusOK, resp)
	}
}

// OverrideRole POST /api/comments/override-role（注记-L1 强权通道）。
func OverrideRole(svc CommentAPI) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := &api.OverrideRoleReq{}
		if err := bindProtoBody(c, req); err != nil {
			AbortWithErr(c, err)
			return
		}
		resp, err := svc.OverrideRole(c.Request.Context(), req)
		if err != nil {
			AbortWithErr(c, err)
			return
		}
		respondProto(c, http.StatusOK, resp)
	}
}
