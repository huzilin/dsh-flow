// Package handler Gin 路由薄壳。本文件：收件箱端点（wb_list_inbox）——
// query 绑定 + 转发 service，业务编排一律在 service 层。
package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/huzilin/wbflow/rd/api"
)

// InboxAPI 收件箱端点对 service 的依赖接口（consumer 侧定义，
// service.InboxService 天然满足，便于 httptest 打桩）。
type InboxAPI interface {
	// List wb_list_inbox（七类通知 + TODO 投影双通道）
	List(ctx context.Context, req *api.ListInboxReq) (*api.ListInboxResp, error)
}

// RegisterInboxRoutes 注册收件箱路由（group=/api）。
func RegisterInboxRoutes(group *gin.RouterGroup, svc InboxAPI) {
	group.GET("/inbox", ListInbox(svc))
}

// ListInbox GET /api/inbox（query 绑定）。
func ListInbox(svc InboxAPI) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := &api.ListInboxReq{
			Role:          c.Query("role"),
			ReadFilter:    c.Query("read_filter"),
			Kind:          c.Query("kind"),
			RequirementId: c.Query("requirement_id"),
			Page:          queryInt64(c, "page"),
			PageSize:      queryInt64(c, "page_size"),
		}
		resp, err := svc.List(c.Request.Context(), req)
		if err != nil {
			AbortWithErr(c, err)
			return
		}
		respondProto(c, http.StatusOK, resp)
	}
}
