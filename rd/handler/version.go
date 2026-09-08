// Package handler Gin 路由薄壳。本文件：versions 五端点
// （wb_list / wb_create / wb_update_version、wb_advance_version、wb_deprecate_version）。
package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/huzilin/wbflow/rd/api"
)

// VersionAPI versions 端点对 service 的依赖接口（consumer 侧定义）。
type VersionAPI interface {
	// List wb_list_versions（含待分配池）
	List(ctx context.Context, req *api.ListVersionsReq) (*api.ListVersionsResp, error)
	// Create wb_create_version
	Create(ctx context.Context, req *api.CreateVersionReq) (*api.CreateVersionResp, error)
	// Update wb_update_version（改名，乐观锁）
	Update(ctx context.Context, req *api.UpdateVersionReq) (*api.UpdateVersionResp, error)
	// Advance wb_advance_version（发布软约束：unfinished 仅提示不拦截）
	Advance(ctx context.Context, req *api.AdvanceVersionReq) (*api.AdvanceVersionResp, error)
	// Deprecate wb_deprecate_version ⚠️（条件解绑：任一需求已开始 blocked，全部
	// 未开始废弃成功并解绑回池——2026-09-09 拍板-A）
	Deprecate(ctx context.Context, req *api.DeprecateVersionReq) (*api.DeprecateVersionResp, error)
}

// RegisterVersionRoutes 注册 versions 路由（group 形如 /api/versions）。
func RegisterVersionRoutes(group *gin.RouterGroup, svc VersionAPI) {
	group.GET("", ListVersions(svc))
	group.POST("", CreateVersion(svc))
	group.PUT("/:versionId", UpdateVersion(svc))
	group.POST("/:versionId/advance", AdvanceVersion(svc))
	group.POST("/:versionId/deprecate", DeprecateVersion(svc))
}

// ListVersions GET /api/versions。
func ListVersions(svc VersionAPI) gin.HandlerFunc {
	return func(c *gin.Context) {
		resp, err := svc.List(c.Request.Context(), &api.ListVersionsReq{})
		if err != nil {
			AbortWithErr(c, err)
			return
		}
		respondProto(c, http.StatusOK, resp)
	}
}

// CreateVersion POST /api/versions（body 绑定）。
func CreateVersion(svc VersionAPI) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := &api.CreateVersionReq{}
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

// UpdateVersion PUT /api/versions/:versionId（body 绑定，路径 id 优先）。
func UpdateVersion(svc VersionAPI) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := &api.UpdateVersionReq{}
		if err := bindProtoBody(c, req); err != nil {
			AbortWithErr(c, err)
			return
		}
		req.VersionId = c.Param("versionId")
		resp, err := svc.Update(c.Request.Context(), req)
		if err != nil {
			AbortWithErr(c, err)
			return
		}
		respondProto(c, http.StatusOK, resp)
	}
}

// AdvanceVersion POST /api/versions/:versionId/advance。
func AdvanceVersion(svc VersionAPI) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := &api.AdvanceVersionReq{}
		if err := bindProtoBody(c, req); err != nil {
			AbortWithErr(c, err)
			return
		}
		req.VersionId = c.Param("versionId")
		resp, err := svc.Advance(c.Request.Context(), req)
		if err != nil {
			AbortWithErr(c, err)
			return
		}
		respondProto(c, http.StatusOK, resp)
	}
}

// DeprecateVersion POST /api/versions/:versionId/deprecate。
func DeprecateVersion(svc VersionAPI) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := &api.DeprecateVersionReq{}
		if err := bindProtoBody(c, req); err != nil {
			AbortWithErr(c, err)
			return
		}
		req.VersionId = c.Param("versionId")
		resp, err := svc.Deprecate(c.Request.Context(), req)
		if err != nil {
			AbortWithErr(c, err)
			return
		}
		respondProto(c, http.StatusOK, resp)
	}
}
