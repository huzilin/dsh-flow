// rd 服务入口：配置加载 → zap 初始化 → GORM MySQL 连接 → Gin 路由注册 →
// 优雅退出（SIGTERM/SIGINT：server.Shutdown + 连接关闭）。
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/huzilin/wbflow/rd/config"
	"github.com/huzilin/wbflow/rd/database"
	"github.com/huzilin/wbflow/rd/handler"
	"github.com/huzilin/wbflow/rd/llm"
	"github.com/huzilin/wbflow/rd/logger"
	"github.com/huzilin/wbflow/rd/migration"
	"github.com/huzilin/wbflow/rd/repo"
	"github.com/huzilin/wbflow/rd/service"
)

func main() {
	// migrate 子命令：应用 schema 迁移后退出，不启动 HTTP 服务。
	// 用法：rd migrate [-config 主配置] [-config-local 本地覆盖]
	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		runMigrate(os.Args[2:])
		return
	}
	runServer()
}

// runMigrate 执行 schema 迁移（Bootstrap 建库 + Run 按版本应用），成功即退出码 0。
func runMigrate(args []string) {
	fs := flag.NewFlagSet("migrate", flag.ExitOnError)
	mainPath := fs.String("config", config.DefaultMainConfigPath, "主配置文件路径")
	localPath := fs.String("config-local", config.DefaultLocalConfigPath, "本地覆盖配置路径（gitignored，可不存在）")
	_ = fs.Parse(args)

	cfg, err := config.Load(*mainPath, *localPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "配置加载失败: "+err.Error())
		os.Exit(1)
	}
	logger.InitZap(cfg.Log.Path, logger.Mode(cfg.Log.Mode), cfg.Log.Level)
	defer logger.Sync()
	lg := logger.L()

	ctx := context.Background()
	db, report, err := migration.OpenForMigrate(ctx, &cfg.MySQL)
	if err != nil {
		lg.Error("迁移失败", zap.Error(err))
		os.Exit(1)
	}
	defer func() {
		_ = db.Close()
	}()
	lg.Info("迁移完成",
		zap.Int64("from_version", report.FromVersion),
		zap.Int64("to_version", report.ToVersion),
		zap.Int64s("applied", report.Applied),
	)
}

// runServer 原 HTTP 服务路径：配置加载 → zap 初始化 → GORM MySQL 连接 → Gin 路由注册 →
// 优雅退出（SIGTERM/SIGINT：server.Shutdown + 连接关闭）。
func runServer() {
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
	var taskSvc *service.TaskService
	var issueSvc *service.IssueService
	var escalateSvc *service.EscalationService
	var inboxSvc *service.InboxService
	var reviewSvc *service.ReviewService
	var advanceSvc *service.AdvanceService
	var llmFlowSvc *service.LLMFlowService
	var repoHandle *repo.Repo
	if db.Gorm != nil {
		repoHandle = repo.NewRepo(db.Gorm)
		reqSvc = service.NewRequirementService(repoHandle)
		verSvc = service.NewVersionService(repoHandle)
		flowSvc = service.NewFlowService(repoHandle)
		artSvc = service.NewArtifactService(repoHandle, cfg.Artifacts.DocsRoot)
		taskSvc = service.NewTaskService(repoHandle)
		issueSvc = service.NewIssueService(repoHandle, cfg.Escalate.LoopThreshold)
		escalateSvc = service.NewEscalationService(repoHandle)
		inboxSvc = service.NewInboxService(repoHandle)
		reviewSvc = service.NewReviewService(repoHandle)
		advanceSvc = service.NewAdvanceService(repoHandle)
		// LLM 通路装配（-L4 + C11 装配模式）：base_url 空 = FakeClient 替身演示态；
		// 非 = ChatClient 经 DSH 网关（api_key 环境变量注入，空串=不带鉴权头）
		var llmClient llm.Completer
		if cfg.LLM.BaseURL == "" {
			llmClient = &llm.FakeClient{}
			lg.Warn("llm.base_url 未配置，LLM 通路运行于替身演示态（FakeClient，不外呼）")
		} else {
			timeout := time.Duration(cfg.LLM.TimeoutSec) * time.Second
			if timeout <= 0 {
				timeout = 120 * time.Second
			}
			llmClient = llm.NewChatClient(cfg.LLM.BaseURL, cfg.LLM.APIKey, cfg.LLM.Model, timeout, nil)
			lg.Info("LLM 通路已装配（DSH 网关）",
				zap.String("base_url", cfg.LLM.BaseURL),
				zap.String("model", cfg.LLM.Model),
				zap.Duration("timeout", timeout),
			)
		}
		llmFlowSvc = service.NewLLMFlowService(repoHandle, taskSvc, artSvc, issueSvc, reviewSvc, llmClient, service.LLMFlowOptions{
			Enabled:  cfg.LLM.Enabled,
			RetryMax: int(cfg.LLM.RetryMax),
			Actions:  cfg.LLM.Actions,
		}).WithLogger(lg)
	} else {
		lg.Warn("gorm 句柄不可用，API 路由降级不注册（仅 /healthz）")
	}
	router := handler.NewRouter(&handler.Deps{
		Logger:      lg,
		Health:      service.NewHealth(db),
		Token:       cfg.Auth.Token,
		Reqs:        reqSvc,
		Vers:        verSvc,
		Flows:       flowSvc,
		Artifacts:   artSvc,
		Tasks:       taskSvc,
		Comments:    issueSvc,
		Escalations: escalateSvc,
		Inbox:       inboxSvc,
		Reviews:     reviewSvc,
		Advance:     advanceSvc,
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

	// 6. 扫描器 goroutine（T10）：阈值/间隔参数化（config scanner 段）；优雅退出联动
	scannerStop := make(chan struct{})
	var scannerDone sync.WaitGroup
	if cfg.Scanner.Enabled && repoHandle != nil {
		scanner := service.NewScanner(repoHandle, db, service.ScannerOptions{
			Interval:       time.Duration(cfg.Scanner.IntervalSec) * time.Second,
			FollowupAfter:  time.Duration(cfg.Scanner.FollowupAfterSec) * time.Second,
			InterveneAfter: time.Duration(cfg.Scanner.InterveneAfterSec) * time.Second,
			RetryBackoff:   time.Duration(cfg.Scanner.RetryBackoffSec) * time.Second,
		}).WithLogger(lg).WithLLMFlow(llmFlowSvc)
		scannerDone.Add(1)
		go func() {
			defer scannerDone.Done()
			scanner.Run(context.Background(), scannerStop)
		}()
		lg.Info("扫描器已启动",
			zap.Int64("interval_sec", cfg.Scanner.IntervalSec),
			zap.Int64("followup_after_sec", cfg.Scanner.FollowupAfterSec),
			zap.Int64("intervene_after_sec", cfg.Scanner.InterveneAfterSec),
			zap.Int64("retry_backoff_sec", cfg.Scanner.RetryBackoffSec),
		)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	sig := <-quit
	lg.Info("收到退出信号，开始优雅退出", zap.String("signal", sig.String()))

	close(scannerStop)
	scannerDone.Wait()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		lg.Error("http 优雅退出超时", zap.Error(err))
	}
	lg.Info("rd 服务已退出")
}
