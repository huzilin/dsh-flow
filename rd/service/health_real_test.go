// 真实 MySQL 探活测试（T1 AC1：MySQL 8.0 假设实测确认）。
//
// env 门控，默认跳过；运行方式（密码经 env 注入，不落地仓库）：
//
//	WBFLOW_MYSQL_IT=1 \
//	WBFLOW_MYSQL_HOST=192.168.3.100 \
//	WBFLOW_MYSQL_PORT=3306 \
//	WBFLOW_MYSQL_USER=root \
//	WBFLOW_MYSQL_PASSWORD='...' \
//	go test ./service/ -run TestIntegration_Health_RealMySQL -v
package service_test

import (
	"context"
	"os"
	"strconv"
	"strings"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/huzilin/wbflow/rd/config"
	"github.com/huzilin/wbflow/rd/database"
	"github.com/huzilin/wbflow/rd/service"
)

func TestIntegration_Health_RealMySQL(t *testing.T) {
	if os.Getenv("WBFLOW_MYSQL_IT") == "" {
		t.Skip("真实 MySQL 探活：设 WBFLOW_MYSQL_IT=1 且注入 WBFLOW_MYSQL_PASSWORD 后运行")
	}

	cfg := &config.MySQLConfig{
		Host:               envOr("WBFLOW_MYSQL_HOST", "192.168.3.100"),
		Port:               3306,
		User:               envOr("WBFLOW_MYSQL_USER", "root"),
		Password:           os.Getenv("WBFLOW_MYSQL_PASSWORD"),
		DBName:             envOr("WBFLOW_MYSQL_DBNAME", "wbflow"),
		MaxOpenConns:       5,
		MaxIdleConns:       2,
		ConnMaxLifetimeSec: 3600,
	}
	if p := os.Getenv("WBFLOW_MYSQL_PORT"); p != "" {
		port, err := strconv.ParseInt(p, 10, 64)
		assert.NilError(t, err)
		cfg.Port = port
	}

	db, err := database.Open(cfg)
	assert.NilError(t, err)
	defer func() { _ = db.Close() }()

	// T1 本工单不建库：wbflow 库不存在属预期，gorm 打开失败不阻断探活
	if db.OpenErr != nil {
		t.Logf("gorm 打开失败（预期内，库可能未建）: %v", db.OpenErr)
	}

	status := service.NewHealth(db).Check(context.Background())
	t.Logf("healthz: state=%s reachable=%v version=%q db_state=%s err=%q",
		status.State, status.MySQL.Reachable, status.MySQL.Version, status.MySQL.DBState, status.MySQL.Error)

	// 服务器必须可达
	assert.Assert(t, status.MySQL.Reachable, "MySQL 服务器不可达")
	// 版本断言：主版本 8.x（AC1；若实际非 8.0，如实报告不改表）
	assert.Assert(t, status.MySQL.Version != "", "SELECT VERSION() 应返回版本号")
	assert.Assert(t, strings.HasPrefix(status.MySQL.Version, "8."),
		"MySQL 主版本非 8.x，实测: "+status.MySQL.Version)
}

func envOr(key string, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
