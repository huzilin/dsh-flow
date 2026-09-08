// Package handler Gin 路由薄壳。本文件：requirements 五端点
// （wb_list / wb_create / wb_get / wb_update / wb_delete_requirement）——
// 参数绑定 + 转发 service，业务编排一律在 service 层。
package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/huzilin/wbflow/rd/api"
)

// RequirementAPI requirements 端点对 service 的依赖接口（consumer 侧定义，
// service.RequirementService 天然满足，便于 httptest 打桩）。
type RequirementAPI interface {
	// List wb_list_requirements
	List(ctx context.Context, req *api.ListRequirementsReq) (*api.ListRequirementsResp, error)
	// Get wb_get_requirement（聚合快照基础版）
	Get(ctx context.Context, req *api.GetRequirementReq) (*api.GetRequirementResp, error)
	// Create wb_create_requirement（category→流程类型绑定派生）
	Create(ctx context.Context, req *api.CreateRequirementReq) (*api.CreateRequirementResp, error)
	// Update wb_update_requirement（乐观锁 + 自动 diff 变更历史）
	Update(ctx context.Context, req *api.UpdateRequirementReq) (*api.UpdateRequirementResp, error)
	// Delete wb_delete_requirement ⚠️（需 User 明确指令口径）
	Delete(ctx context.Context, req *api.DeleteRequirementReq) (*api.DeleteRequirementResp, error)
}

// RegisterRequirementRoutes 注册 requirements 路由（group 形如 /api/requirements）。
func RegisterRequirementRoutes(group *gin.RouterGroup, svc RequirementAPI) {
	group.GET("", ListRequirements(svc))
	group.POST("", CreateRequirement(svc))
	group.GET("/:reqId", GetRequirement(svc))
	group.PUT("/:reqId", UpdateRequirement(svc))
	group.DELETE("/:reqId", DeleteRequirement(svc))
}

// ListRequirements GET /api/requirements（query 绑定）。
func ListRequirements(svc RequirementAPI) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := &api.ListRequirementsReq{
			VersionId: c.Query("version_id"),
			Backlog:   c.Query("backlog") == "true",
			Priority:  c.Query("priority"),
			Status:    c.Query("status"),
			Scope:     c.Query("scope"),
			Page:      queryInt64(c, "page"),
			PageSize:  queryInt64(c, "page_size"),
		}
		resp, err := svc.List(c.Request.Context(), req)
		if err != nil {
			AbortWithErr(c, err)
			return
		}
		respondProto(c, http.StatusOK, resp)
	}
}

// GetRequirement GET /api/requirements/:reqId。
func GetRequirement(svc RequirementAPI) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := &api.GetRequirementReq{RequirementId: c.Param("reqId")}
		resp, err := svc.Get(c.Request.Context(), req)
		if err != nil {
			AbortWithErr(c, err)
			return
		}
		respondProto(c, http.StatusOK, resp)
	}
}

// CreateRequirement POST /api/requirements（body 绑定）。
func CreateRequirement(svc RequirementAPI) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := &api.CreateRequirementReq{}
		if err := bindProtoBody(c, req); err != nil {
			AbortWithErr(c, err)
			return
		}
		resp, err := svc.Create(c.Request.Context(), req)
		if err != nil {
			AbortWithErr(c, err)
			return
		}
		respondProto(c, http.StatusOK, resp)
	}
}

// UpdateRequirement PUT /api/requirements/:reqId（body 绑定，路径 id 优先）。
func UpdateRequirement(svc RequirementAPI) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := &api.UpdateRequirementReq{}
		if err := bindProtoBody(c, req); err != nil {
			AbortWithErr(c, err)
			return
		}
		req.RequirementId = c.Param("reqId")
		resp, err := svc.Update(c.Request.Context(), req)
		if err != nil {
			AbortWithErr(c, err)
			return
		}
		respondProto(c, http.StatusOK, resp)
	}
}

// DeleteRequirement DELETE /api/requirements/:reqId（confirm 走 query 参数）。
func DeleteRequirement(svc RequirementAPI) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := &api.DeleteRequirementReq{
			RequirementId: c.Param("reqId"),
			Confirm:       c.Query("confirm") == "true",
		}
		resp, err := svc.Delete(c.Request.Context(), req)
		if err != nil {
			AbortWithErr(c, err)
			return
		}
		respondProto(c, http.StatusOK, resp)
	}
}

// queryInt64 query 参数解析（空/非法按 0 处理，语义校验归 service）。
func queryInt64(c *gin.Context, key string) int64 {
	raw := c.Query(key)
	if raw == "" {
		return 0
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0
	}
	return value
}
