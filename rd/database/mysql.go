// Package database 负责 MySQL 接入：DSN 构造（mysql.NewConfig，禁手拼字符串）、
// GORM 打开（PrepareStmt + SkipDefaultTransaction）与连接池参数装配。
//
// 返回三个句柄分层：
//   - Gorm：业务读写句柄（全量 DSN，含 dbname），打开失败不致命（如库未建），
//     返回 nil + 记录 OpenErr，服务仍可启动并由 /healthz 上报 db missing；
//   - Server：无 dbname 的服务器级连接，用于服务器可达性探测与 SELECT VERSION()；
//   - Database：全量 DSN 连接，用于 dbname 存在性判定（区分 1049 unknown database）。
package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	mysqldrv "github.com/go-sql-driver/mysql"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/huzilin/wbflow/rd/config"
)

// DB MySQL 连接句柄集合。
type DB struct {
	// Gorm 业务读写句柄；打开失败为 nil
	Gorm *gorm.DB
	// Server 服务器级连接（DSN 无 dbname）
	Server *sql.DB
	// Database 全量 DSN 连接（DSN 含 dbname）
	Database *sql.DB
	// OpenErr Gorm 打开失败原因；nil=打开成功
	OpenErr error
}

// PingTimeout 建连超时。
const PingTimeout = 5 * time.Second

// Open 建立 MySQL 连接。Gorm 打开失败不返回错误（库不存在等场景服务仍须可探活），
// 仅当连接句柄无法创建（DSN 构造/驱动注册失败）时返回错误。
func Open(cfg *config.MySQLConfig) (*DB, error) {
	server, err := sql.Open("mysql", buildDSN(cfg, false))
	if err != nil {
		return nil, err
	}
	full, err := sql.Open("mysql", buildDSN(cfg, true))
	if err != nil {
		_ = server.Close()
		return nil, err
	}

	db := &DB{
		Server:   server,
		Database: full,
	}

	gormDB, gormErr := gorm.Open(gormmysql.New(gormmysql.Config{
		DSN: buildDSN(cfg, true),
	}), &gorm.Config{
		PrepareStmt:            true,
		SkipDefaultTransaction: true,
	})
	if gormErr != nil {
		db.OpenErr = gormErr
	} else {
		sqlDB, poolErr := gormDB.DB()
		if poolErr != nil {
			db.OpenErr = poolErr
		} else {
			applyPool(sqlDB, cfg)
			db.Gorm = gormDB
		}
	}
	return db, nil
}

// Close 关闭全部连接句柄。
func (d *DB) Close() error {
	firstErr := error(nil)
	if d.Gorm != nil {
		if sqlDB, err := d.Gorm.DB(); err == nil {
			firstErr = sqlDB.Close()
		}
	}
	if err := d.Server.Close(); err != nil && firstErr == nil {
		firstErr = err
	}
	if err := d.Database.Close(); err != nil && firstErr == nil {
		firstErr = err
	}
	return firstErr
}

// PingContext 服务器级连通性探测。
func (d *DB) PingContext(ctx context.Context) error {
	return d.Server.PingContext(ctx)
}

// DatabasePingContext 全量 DSN（含 dbname）连通性探测，
// 调用方据错误码区分「库不存在」与「连不上」。
func (d *DB) DatabasePingContext(ctx context.Context) error {
	return d.Database.PingContext(ctx)
}

// VersionContext 执行 SELECT VERSION()。
func (d *DB) VersionContext(ctx context.Context) (string, error) {
	var version string
	err := d.Server.QueryRowContext(ctx, "SELECT VERSION()").Scan(&version)
	return version, err
}

// applyPool 装配连接池参数（按部署配置）。
func applyPool(sqlDB *sql.DB, cfg *config.MySQLConfig) {
	sqlDB.SetMaxOpenConns(int(cfg.MaxOpenConns))
	sqlDB.SetMaxIdleConns(int(cfg.MaxIdleConns))
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetimeSec) * time.Second)
}

// buildDSN 用 mysql.NewConfig 填字段后 FormatDSN 构造（密码含 @ 等特殊字符，
// 禁止手拼 DSN 字符串）。withDB=false 时不含 dbname（服务器级探测用）。
func buildDSN(cfg *config.MySQLConfig, withDB bool) string {
	dsnCfg := mysqldrv.NewConfig()
	dsnCfg.Net = "tcp"
	dsnCfg.Addr = fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	dsnCfg.User = cfg.User
	dsnCfg.Passwd = cfg.Password
	if withDB {
		dsnCfg.DBName = cfg.DBName
	}
	dsnCfg.Timeout = PingTimeout
	dsnCfg.ReadTimeout = 30 * time.Second
	dsnCfg.WriteTimeout = 30 * time.Second
	dsnCfg.ParseTime = true
	dsnCfg.Loc = time.Local
	dsnCfg.Params = map[string]string{"charset": "utf8mb4"}
	return dsnCfg.FormatDSN()
}
