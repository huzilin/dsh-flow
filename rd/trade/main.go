package main

import (
	"encoding/json"
	"os"

	"github.com/huzilin/trade/consts"
	"github.com/huzilin/trade/logger"
	"github.com/huzilin/trade/platform"
	"github.com/huzilin/trade/types"
	"go.uber.org/zap"
)

func main() {
	// 创建平台，部分字段使用默认值
	cfg := &platform.Config{
		LLMProvider:     "deepseek",
		MaxDebateRounds: 2,
		// 其他字段未设置，使用默认值
	}
	initLog()
	logger.L().Info("=== TradingAgents Go Platform ===")

	p, err := platform.NewTradingPlatform(cfg)
	if err != nil {
		logger.L().Error("Error creating platform", zap.Error(err))
		return
	}

	// 打印配置验证默认值是否生效
	logger.L().Info("Using LLM Provider", zap.String("provider", p.Config.LLMProvider))
	logger.L().Info("Using Max Debate Rounds", zap.Int64("rounds", p.Config.MaxDebateRounds), zap.String("note", "custom"))
	logger.L().Info("Using DeepThink LLM", zap.String("model", p.Config.DeepThinkLLM), zap.String("note", "default"))
	logger.L().Info("Using Report Output Dir", zap.String("dir", p.Config.ReportOutputDir), zap.String("note", "default"))

	p.RegisterTradingAgentsAgent()

	request := &types.AgentRunRequest{
		Symbol:    "600570",
		TradeDate: "2026-04-03",
		Context: map[string]any{
			"quick_mode":     true,
			"persist_report": true,
		},
	}

	result, err := p.RunAgent(consts.TradingAgents, request)
	if err != nil {
		logger.L().Error("Error running agent", zap.Error(err))
		return
	}

	logger.L().Info("Agent Name", zap.String("name", result.AgentName))

	if result.Decision != nil {
		logger.L().Info("Decision",
			zap.String("symbol", result.Decision.Symbol),
			zap.String("trade_date", result.Decision.TradeDate),
			zap.String("action", string(result.Decision.Action)),
			zap.String("rationale", result.Decision.Rationale),
		)
	}

	if reportFile, ok := result.Outputs["report_file"].(string); ok {
		logger.L().Info("Report File", zap.String("path", reportFile))
	}
	if reportDir, ok := result.Outputs["report_dir"].(string); ok {
		logger.L().Info("Report Directory", zap.String("path", reportDir))
	}
	if reportPDF, ok := result.Outputs["report_pdf_file"].(string); ok {
		logger.L().Info("Report PDF", zap.String("path", reportPDF))
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	logger.L().Info("Full Result", zap.String("result", string(data)))

	// 刷新日志缓冲区
	logger.Sync()
}

func initLog() {
	mode := logger.ModeDev
	if os.Getenv("ENV") == "prod" {
		mode = logger.ModeProd
	}
	logger.InitZap("./logs/trade.log", mode)
}
