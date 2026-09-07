// Package handler Gin 路由薄壳。本文件：任务总线端点
// （wb_assign_task / wb_task_update）——参数绑定 + 转发 service，业务编排一律在
// service 层。
package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/huzilin/wbflow/rd/api"
)

// TaskAPI 任务总线端点对 service 的依赖接口（consumer 侧定义，
// service.TaskService 天然满足，便于 httptest 打桩）。
type TaskAPI interface {
	// Assign wb_assign_task（幂等派发：reqId+nodeId+action_id+assignee）
	Assign(ctx context.Context, req *api.AssignTaskReq) (*api.AssignTaskResp, error)
	// Update wb_task_update（event: ack|progress|complete，complete 触发判据校验）
	Update(ctx context.Context, req *api.TaskUpdateReq) (*api.TaskUpdateResp, error)
}

// RegisterTaskRoutes 注册任务总线路由（group=/api）。
func RegisterTaskRoutes(group *gin.RouterGroup, svc TaskAPI) {
	group.POST("/tasks", AssignTask(svc))
	group.PATCH("/tasks/:taskId", TaskUpdate(svc))
}

// AssignTask POST /api/tasks（body 绑定）。
func AssignTask(svc TaskAPI) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := &api.AssignTaskReq{}
		if err := bindProtoBody(c, req); err != nil {
			AbortWithErr(c, err)
			return
		}
		resp, err := svc.Assign(c.Request.Context(), req)
		if err != nil {
			AbortWithErr(c, err)
			return
		}
		respondProto(c, http.StatusOK, resp)
	}
}

// TaskUpdate PATCH /api/tasks/:taskId（body 绑定，路径 id 优先）。
func TaskUpdate(svc TaskAPI) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := &api.TaskUpdateReq{}
		if err := bindProtoBody(c, req); err != nil {
			AbortWithErr(c, err)
			return
		}
		req.TaskId = c.Param("taskId")
		resp, err := svc.Update(c.Request.Context(), req)
		if err != nil {
			AbortWithErr(c, err)
			return
		}
		respondProto(c, http.StatusOK, resp)
	}
}
