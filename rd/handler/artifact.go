// Package handler Gin 路由薄壳。本文件：产物端点（wb_add_artifact /
// wb_get_artifact_content，T6）——参数绑定 + 转发 service，业务编排一律在 service 层。
package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/huzilin/wbflow/rd/api"
)

// ArtifactAPI 产物端点对 service 的依赖接口（consumer 侧定义，
// service.ArtifactService 天然满足，便于 httptest 打桩）。
type ArtifactAPI interface {
	// Add wb_add_artifact（product_type 受控枚举校验 + 幂等登记）
	Add(ctx context.Context, req *api.AddArtifactReq) (*api.AddArtifactResp, error)
	// GetContent wb_get_artifact_content（document 正文 / link 外链）
	GetContent(ctx context.Context, req *api.GetArtifactContentReq) (*api.GetArtifactContentResp, error)
}

// RegisterArtifactRoutes 注册产物路由（group 形如 /api/requirements，
// 与 requirements/flows 端点同组挂载子路径）。
func RegisterArtifactRoutes(group *gin.RouterGroup, svc ArtifactAPI) {
	group.POST("/:reqId/nodes/:nodeId/artifacts", AddArtifact(svc))
	group.GET("/:reqId/artifacts/:artifactId/content", GetArtifactContent(svc))
}

// AddArtifact POST /api/requirements/:reqId/nodes/:nodeId/artifacts
// （body 绑定，路径 id 优先）。
func AddArtifact(svc ArtifactAPI) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := &api.AddArtifactReq{}
		if err := bindProtoBody(c, req); err != nil {
			AbortWithErr(c, err)
			return
		}
		req.RequirementId = c.Param("reqId")
		req.NodeId = c.Param("nodeId")
		resp, err := svc.Add(c.Request.Context(), req)
		if err != nil {
			AbortWithErr(c, err)
			return
		}
		respondProto(c, http.StatusOK, resp)
	}
}

// GetArtifactContent GET /api/requirements/:reqId/artifacts/:artifactId/content。
func GetArtifactContent(svc ArtifactAPI) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := &api.GetArtifactContentReq{
			RequirementId: c.Param("reqId"),
			ArtifactId:    c.Param("artifactId"),
		}
		resp, err := svc.GetContent(c.Request.Context(), req)
		if err != nil {
			AbortWithErr(c, err)
			return
		}
		respondProto(c, http.StatusOK, resp)
	}
}
