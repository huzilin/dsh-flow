package logger

import (
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// 全局单例logger
var log *zap.Logger
var sugar *zap.SugaredLogger

// Mode 运行环境：dev / prod
type Mode string

const (
	ModeDev  Mode = "dev"
	ModeProd Mode = "prod"
)

// InitZap 初始化zap全局日志
// logPath 日志文件路径, mode 运行模式, level 日志级别（debug/info/warn/error，非法值回退 info）
func InitZap(logPath string, mode Mode, level string) {
	// 1. 分割日志配置
	lj := &lumberjack.Logger{
		Filename:   filepath.Clean(logPath),
		MaxSize:    100,  // 单个文件最大100MB
		MaxBackups: 10,   // 最多保留10个备份
		MaxAge:     7,    // 保留7天日志
		Compress:   true, // 旧日志gzip压缩
	}

	// 2. 编码器配置
	encoderCfg := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stack",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder, // 简短文件行号
	}

	var core zapcore.Core
	switch mode {
	case ModeDev:
		// 开发环境：控制台彩色文本输出
		consoleEncoder := zapcore.NewConsoleEncoder(encoderCfg)
		core = zapcore.NewTee(
			zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), parseLevel(level)),
		)
	case ModeProd:
		// 生产环境：JSON格式输出到文件
		jsonEncoder := zapcore.NewJSONEncoder(encoderCfg)
		writeSyncer := zapcore.NewMultiWriteSyncer(zapcore.AddSync(lj), zapcore.AddSync(os.Stdout))
		core = zapcore.NewTee(
			zapcore.NewCore(jsonEncoder, writeSyncer, parseLevel(level)),
		)
	default:
		// 未知模式回退 dev 行为
		consoleEncoder := zapcore.NewConsoleEncoder(encoderCfg)
		core = zapcore.NewTee(
			zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), parseLevel(level)),
		)
	}

	// 3. 创建logger，开启调用栈、堆栈打印
	log = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	sugar = log.Sugar()
}

// L 获取原生zap.Logger 强类型高性能打印
func L() *zap.Logger {
	return log
}

// S 获取SugaredLogger 方便printf风格
func S() *zap.SugaredLogger {
	return sugar
}

// Sync 刷新缓冲区，程序退出前必须调用
func Sync() {
	_ = log.Sync()
}

// parseLevel 解析日志级别字符串，非法值回退 info。
func parseLevel(level string) zapcore.Level {
	lv, err := zapcore.ParseLevel(level)
	if err != nil {
		return zapcore.InfoLevel
	}
	return lv
}
