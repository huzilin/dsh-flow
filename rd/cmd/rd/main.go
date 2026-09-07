// rd 服务入口：配置加载 → zap 初始化 → GORM MySQL 连接 → Gin 路由注册 →
// 优雅退出（SIGTERM/SIGINT：server.Shutdown + 连接关闭）。
package main

import (
	"context"
	"errors"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/huzilin/wbflow/rd/config"
	"github.com/huzilin/wbflow/rd/database"
	"github.com/huzilin/wbflow/rd/handler"
	"github.com/huzilin/wbflow/rd/logger"
	"github.com/huzilin/wbflow/rd/repo"
	"github.com/huzilin/wbflow/rd/service"
)

func main() {
	mainPath := flag.String("config", config.DefaultMainConfigPath, "主配置文件路径")
	localPath := flag.String("config-local", config.DefaultLocalConfigPath, "本地覆盖配置路径（gitignored，可不存在）")
	flag.Parse()

	// 1. 配置加载（含 ${VAR} 环境变量展开与本地覆盖合并）
	cfg, err := config.Load(*mainPath, *localPath)
	if err != nil {
		panic("配置加载失败: " + err.Error())
	}

	// 2. zap 初始化
	logger.InitZap(cfg.Log.Path, logger.Mode(cfg.Log.Mode), cfg.Log.Level)
	defer logger.Sync()
	lg := logger.L()

	// 3. GORM MySQL 连接（库未建等场景不阻断启动，由 /healthz 上报 db missing）
	db, err := database.Open(&cfg.MySQL)
	if err != nil {
		lg.Error("mysql 连接初始化失败", zap.Error(err))
		panic("mysql 连接初始化失败: " + err.Error())
	}
	defer func() {
		_ = db.Close()
	}()
	if db.OpenErr != nil {
		lg.Warn("gorm 打开失败（服务降级启动，healthz 将上报）", zap.Error(db.OpenErr))
	} else {
		lg.Info("mysql 连接就绪",
			zap.String("host", cfg.MySQL.Host),
			zap.String("dbname", cfg.MySQL.DBName),
		)
	}

	// 4. Gin 路由注册（DB 未就绪时服务降级：仅 /healthz，API 路由不注册）
	var reqSvc *service.RequirementService
	var verSvc *service.VersionService
	var flowSvc *service.FlowService
	var artSvc *service.ArtifactService
	if db.Gorm != nil {
		r := repo.NewRepo(db.Gorm)
		reqSvc = service.NewRequirementService(r)
		verSvc = service.NewVersionService(r)
		flowSvc = service.NewFlowService(r)
		artSvc = service.NewArtifactService(r, cfg.Artifacts.DocsRoot)
	} else {
		lg.Warn("gorm 句柄不可用，API 路由降级不注册（仅 /healthz）")
	}
	router := handler.NewRouter(&handler.Deps{
		Logger:    lg,
		Health:    service.NewHealth(db),
		Token:     cfg.Auth.Token,
		Reqs:      reqSvc,
		Vers:      verSvc,
		Flows:     flowSvc,
		Artifacts: artSvc,
	})
	srv := &http.Server{
		Addr:    ":" + strconv.FormatInt(cfg.HTTP.Port, 10),
		Handler: router,
	}

	// 5. 启动 + 优雅退出
	go func() {
		lg.Info("rd 服务启动", zap.Int64("port", cfg.HTTP.Port))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			lg.Fatal("http 服务异常退出", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	sig := <-quit
	lg.Info("收到退出信号，开始优雅退出", zap.String("signal", sig.String()))

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		lg.Error("http 优雅退出超时", zap.Error(err))
	}
	lg.Info("rd 服务已退出")
}
