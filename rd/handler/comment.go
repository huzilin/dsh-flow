// Package handler Gin 路由薄壳。本文件：评论/issue 端点
// （wb_comment / wb_clarified / wb_declare_fixed / wb_settle / wb_qualify /
// wb_confirm / wb_reopen / wb_revise / wb_override_role；NAM-09 契约改名，
// 路由名对齐标准领域事件词，行为不变）——
// 参数绑定 + 转发 service，业务编排与状态机判定一律在 service/domain 层。
package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/huzilin/wbflow/rd/api"
)

// CommentAPI 评论/issue 端点对 service 的依赖接口（consumer 侧定义，
// service.IssueService 天然满足，便于 httptest 打桩）。
type CommentAPI interface {
	// Comment wb_comment（四级锚点评论建行）
	Comment(ctx context.Context, req *api.CommentReq) (*api.CommentResp, error)
	// Clarified wb_clarified（owner 澄清完成申报；与 wb_declare_fixed 共用 reply 核心，
	// 服务端不校验 from-state，路由名即调用方意图）
	Clarified(ctx context.Context, req *api.ReplyReq) (*api.ReplyResp, error)
	// DeclareFixed wb_declare_fixed（owner 修复完成申报；与 wb_clarified 共用 reply 核心）
	DeclareFixed(ctx context.Context, req *api.ReplyReq) (*api.ReplyResp, error)
	// Settle wb_settle（协商一致关闭；与 wb_qualify 共用 resolve 核心，path 由入口固定注入）
	Settle(ctx context.Context, req *api.ResolveReq) (*api.ResolveResp, error)
	// Qualify wb_qualify（复核定性真问题、触发打回；与 wb_settle 共用 resolve 核心）
	Qualify(ctx context.Context, req *api.ResolveReq) (*api.ResolveResp, error)
	// Confirm wb_confirm（提评者复核确认；原词面 confirm_review，NAM-09 改名）
	Confirm(ctx context.Context, req *api.ConfirmReviewReq) (*api.ConfirmReviewResp, error)
	// Reopen wb_reopen（复核退回·未解释清楚 pending-confirm→open；与 wb_revise 共用 reopen 核心）
	Reopen(ctx context.Context, req *api.ReopenReq) (*api.ReopenResp, error)
	// Revise wb_revise（修复侧复核退回 fixed→to-fix；与 wb_reopen 共用 reopen 核心）
	Revise(ctx context.Context, req *api.ReopenReq) (*api.ReopenResp, error)
	// OverrideRole wb_override_role（注记-L1 强权通道：User 整角色批量置 resolved
	// ——09-12 裁决 overridden 态废除，override 动作=批量 resolve，resolve_path=override）
	OverrideRole(ctx context.Context, req *api.OverrideRoleReq) (*api.OverrideRoleResp, error)
}

// RegisterIssueRoutes 注册评论/issue 路由：requirements 组挂需求侧建评
// （POST /:reqId/comments），comments 组挂线程侧端点 + 强权通道。
// NAM-09：原回复端点拆 clarified/declare-fixed、原解决端点拆 settle/qualify
// （path 参数取消）、原 reopen 端点拆 reopen（澄清侧）/revise（修复侧）；旧路由不留兼容。
func RegisterIssueRoutes(requirements *gin.RouterGroup, comments *gin.RouterGroup, svc CommentAPI) {
	requirements.POST("/:reqId/comments", Comment(svc))
	comments.POST("/:commentId/clarified", replyAction(svc, svc.Clarified))
	comments.POST("/:commentId/declare-fixed", replyAction(svc, svc.DeclareFixed))
	comments.POST("/:commentId/settle", resolveAction(svc, svc.Settle))
	comments.POST("/:commentId/qualify", resolveAction(svc, svc.Qualify))
	comments.POST("/:commentId/confirm", Confirm(svc))
	comments.POST("/:commentId/reopen", reopenAction(svc, svc.Reopen))
	comments.POST("/:commentId/revise", reopenAction(svc, svc.Revise))
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

// replyAction clarified/declare-fixed 共用薄壳（原 /replies 拆分，body/出参同型）。
func replyAction(svc CommentAPI, action func(context.Context, *api.ReplyReq) (*api.ReplyResp, error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := &api.ReplyReq{}
		if err := bindProtoBody(c, req); err != nil {
			AbortWithErr(c, err)
			return
		}
		req.CommentId = c.Param("commentId")
		resp, err := action(c.Request.Context(), req)
		if err != nil {
			AbortWithErr(c, err)
			return
		}
		respondProto(c, http.StatusOK, resp)
	}
}

// resolveAction settle/qualify 共用薄壳（解决族拆分，path 参数取消、路由名承载语义）。
func resolveAction(svc CommentAPI, action func(context.Context, *api.ResolveReq) (*api.ResolveResp, error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := &api.ResolveReq{}
		if err := bindProtoBody(c, req); err != nil {
			AbortWithErr(c, err)
			return
		}
		req.CommentId = c.Param("commentId")
		resp, err := action(c.Request.Context(), req)
		if err != nil {
			AbortWithErr(c, err)
			return
		}
		respondProto(c, http.StatusOK, resp)
	}
}

// reopenAction reopen/revise 共用薄壳（原 /reopen 拆分：澄清退回 / 修复退回）。
func reopenAction(svc CommentAPI, action func(context.Context, *api.ReopenReq) (*api.ReopenResp, error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := &api.ReopenReq{}
		if err := bindProtoBody(c, req); err != nil {
			AbortWithErr(c, err)
			return
		}
		req.CommentId = c.Param("commentId")
		resp, err := action(c.Request.Context(), req)
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
