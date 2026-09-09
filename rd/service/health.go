// Package service 业务编排层。本文件：健康检查（/healthz 数据源）。
package service

import (
	"context"
	"errors"
	"time"

	"github.com/go-sql-driver/mysql"

	"github.com/huzilin/wbflow/rd/database"
)

// ErrUnknownDatabaseCode MySQL 错误码 1049：unknown database。
const ErrUnknownDatabaseCode = 1049

// HealthChecker healthz 对 handler 暴露的探测接口（便于 httptest 打桩）。
type HealthChecker interface {
	Check(ctx context.Context) *HealthStatus
}

// Health 健康检查 service：探测服务器可达性、SELECT VERSION()、
// dbname 存在性（server ok / db missing 可区分上报）。
type Health struct {
	db *database.DB
}

// NewHealth 构造健康检查 service。
func NewHealth(db *database.DB) *Health {
	return &Health{db: db}
}

// HealthStatus /healthz 出参。
type HealthStatus struct {
	Ok    bool        `json:"ok"`
	State string      `json:"state"`
	MySQL MySQLStatus `json:"mysql"`
}

// MySQLStatus MySQL 连通状态。
type MySQLStatus struct {
	// Reachable 服务器可达（tcp + 认证通过）
	Reachable bool `json:"reachable"`
	// Version SELECT VERSION() 结果；不可达为空串
	Version string `json:"version"`
	// DBState dbname 状态：ok / missing（服务器可达但库不存在） / down / unknown
	DBState string `json:"db_state"`
	// Error 最近一次探测错误摘要；无错为空串
	Error string `json:"error"`
}

// DBState 取值。
const (
	DBStateOK      = "ok"
	DBStateMissing = "missing"
	DBStateDown    = "down"
	DBStateUnknown = "unknown"
)

// 服务整体状态取值。
const (
	StateOK       = "ok"
	StateDegraded = "degraded"
	StateDown     = "down"
)

// Check 执行健康探测。ctx 控制单次探测超时（调用方传 5s 级超时即可）。
func (h *Health) Check(ctx context.Context) *HealthStatus {
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	status := &HealthStatus{}
	mysqlStatus := h.checkMySQL(checkCtx)
	status.MySQL = mysqlStatus

	switch {
	case !mysqlStatus.Reachable:
		status.State = StateDown
	case mysqlStatus.DBState != DBStateOK:
		status.State = StateDegraded
	default:
		status.State = StateOK
	}
	status.Ok = status.State == StateOK
	return status
}

// checkMySQL 服务器探测 + 版本探测 + dbname 存在性判定。
func (h *Health) checkMySQL(ctx context.Context) MySQLStatus {
	out := MySQLStatus{DBState: DBStateUnknown}

	if h.db == nil {
		out.Error = "mysql 连接未初始化"
		return out
	}

	// 1. 服务器可达性
	if err := h.db.PingContext(ctx); err != nil {
		out.Error = err.Error()
		return out
	}
	out.Reachable = true

	// 2. SELECT VERSION()
	version, err := h.db.VersionContext(ctx)
	if err != nil {
		out.Error = err.Error()
	} else {
		out.Version = version
	}

	// 3. dbname 存在性：全量 DSN ping；1049 = 库不存在（server ok / db missing）
	if err := h.db.DatabasePingContext(ctx); err == nil {
		out.DBState = DBStateOK
	} else if isUnknownDatabaseErr(err) {
		out.DBState = DBStateMissing
		if out.Error == "" {
			out.Error = err.Error()
		}
	} else {
		out.DBState = DBStateDown
		out.Error = err.Error()
	}
	return out
}

// isUnknownDatabaseErr 判定是否 MySQL 1049 unknown database。
func isUnknownDatabaseErr(err error) bool {
	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == ErrUnknownDatabaseCode
}
