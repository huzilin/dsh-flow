// Package handler Gin 路由薄壳。本文件：流程实例读端点
// （wb_my_task / 依赖树）——参数绑定 + 转发 service，业务编排一律在 service 层。
package handler

import (
	"net/http"

	"context"
	"github.com/gin-gonic/gin"

	"github.com/huzilin/wbflow/rd/api"
)

// FlowAPI 流程实例端点对 service 的依赖接口（consumer 侧定义，
// service.FlowService 天然满足，便于 httptest 打桩）。
type FlowAPI interface {
	// MyTask wb_my_task（三向校验 role × node × 相位）
	MyTask(ctx context.Context, req *api.MyTaskReq) (*api.MyTaskResp, error)
	// DependencyTree 依赖树（分析师视图，动态加载，环检测）
	DependencyTree(ctx context.Context, req *api.DependencyTreeReq) (*api.DependencyTreeResp, error)
	// Reproject 范围重定投影（T9 打回链路：pending_scope 消费端点）
	Reproject(ctx context.Context, req *api.ReprojectScopeReq) (*api.ReprojectScopeResp, error)
}

// RegisterFlowRoutes 注册流程实例路由（group 形如 /api/requirements，
// 与 requirements 端点同组挂载子路径）。
func RegisterFlowRoutes(group *gin.RouterGroup, svc FlowAPI) {
	group.POST("/:reqId/my-task", MyTask(svc))
	group.GET("/:reqId/dependencies/tree", DependencyTree(svc))
	group.POST("/:reqId/reproject", Reproject(svc))
}

// MyTask POST /api/requirements/:reqId/my-task（body 绑定，路径 id 优先）。
func MyTask(svc FlowAPI) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := &api.MyTaskReq{}
		if err := bindProtoBody(c, req); err != nil {
			AbortWithErr(c, err)
			return
		}
		req.RequirementId = c.Param("reqId")
		resp, err := svc.MyTask(c.Request.Context(), req)
		if err != nil {
			AbortWithErr(c, err)
			return
		}
		respondProto(c, http.StatusOK, resp)
	}
}

// DependencyTree GET /api/requirements/:reqId/dependencies/tree。
func DependencyTree(svc FlowAPI) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := &api.DependencyTreeReq{RequirementId: c.Param("reqId")}
		resp, err := svc.DependencyTree(c.Request.Context(), req)
		if err != nil {
			AbortWithErr(c, err)
			return
		}
		respondProto(c, http.StatusOK, resp)
	}
}

// Reproject POST /api/requirements/:reqId/reproject（body 绑定，路径 id 优先）。
// T9 打回链路：pending_scope 待重定标记的消费端点。
func Reproject(svc FlowAPI) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := &api.ReprojectScopeReq{}
		if err := bindProtoBody(c, req); err != nil {
			AbortWithErr(c, err)
			return
		}
		req.RequirementId = c.Param("reqId")
		resp, err := svc.Reproject(c.Request.Context(), req)
		if err != nil {
			AbortWithErr(c, err)
			return
		}
		respondProto(c, http.StatusOK, resp)
	}
}
