// Package config 负责 rd 服务 yaml 配置的加载与合并。
//
// 加载顺序：先读主配置（必选），后读本地覆盖配置（可选，gitignored），
// 后者逐字段覆盖前者；两份文件内容均先做环境变量展开（${VAR} 语法），
// 保证 password 等敏感字段的真实值不落地仓库文件。
package config

import (
	"os"

	perr "github.com/pingcap/errors"
	"gopkg.in/yaml.v3"
)

// DefaultMainConfigPath 主配置默认路径。
const DefaultMainConfigPath = "configs/config.yaml"

// DefaultLocalConfigPath 本地覆盖配置默认路径（gitignored，可不存在）。
const DefaultLocalConfigPath = "configs/config.local.yaml"

// Config rd 服务总配置。
type Config struct {
	HTTP      HTTPConfig      `yaml:"http"`
	MySQL     MySQLConfig     `yaml:"mysql"`
	Log       LogConfig       `yaml:"log"`
	Auth      AuthConfig      `yaml:"auth"`
	Artifacts ArtifactsConfig `yaml:"artifacts"`
	Scanner   ScannerConfig   `yaml:"scanner"`
}

// HTTPConfig HTTP 服务配置。
type HTTPConfig struct {
	Port int64 `yaml:"port"`
}

// AuthConfig 认证配置（spec §1.1：本期本地部署简单 token；DSH 嵌入期替换适配器，路由不变）。
type AuthConfig struct {
	// Token Bearer 令牌；经 ${WBFLOW_AUTH_TOKEN} 注入，禁止落地仓库文件。
	// 空=未配置：API 路由全部 401（fail closed），/healthz 豁免。
	Token string `yaml:"token"`
}

// MySQLConfig MySQL 接入配置。
type MySQLConfig struct {
	Host     string `yaml:"host"`
	Port     int64  `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
	// 连接池参数（按部署配置）
	MaxOpenConns       int64 `yaml:"max_open_conns"`
	MaxIdleConns       int64 `yaml:"max_idle_conns"`
	ConnMaxLifetimeSec int64 `yaml:"conn_max_lifetime_sec"`
}

// LogConfig 日志配置。
type LogConfig struct {
	// Level 日志级别：debug/info/warn/error
	Level string `yaml:"level"`
	// Mode 运行环境：dev（控制台彩色） / prod（JSON 文件）
	Mode string `yaml:"mode"`
	Path string `yaml:"path"`
}

// ArtifactsConfig 产物配置（T6 wb_get_artifact_content document 读取口径）。
type ArtifactsConfig struct {
	// DocsRoot document 类产物 href 的解析根目录（默认 docs，相对路径按服务进程
	// 工作目录解析；绝对路径原样使用）。安全默认最小权限：根目录收敛到仓库内
	// 专用目录而非 "/"；读取路径 resolve（含 symlink 展开）后越出该目录一律拒绝。
	DocsRoot string `yaml:"docs_root"`
}

// ScannerConfig 扫描器配置（T10：催办/介入阈值参数化——spec §2「agent 无响应」行
// 参数化配置 + §4.1 P2 AC「阈值参数化：配置改小即可加速演练」）。
type ScannerConfig struct {
	// Enabled 扫描器开关；false 不启动扫描循环
	Enabled bool `yaml:"enabled"`
	// IntervalSec 扫描间隔（秒）
	IntervalSec int64 `yaml:"interval_sec"`
	// FollowupAfterSec 催办阈值（秒；spec 默认 5min=300）
	FollowupAfterSec int64 `yaml:"followup_after_sec"`
	// InterveneAfterSec 介入升级阈值（秒；spec 默认 10min=600；须 ≥ 催办阈值，
	// 否则催办恒被介入取代）
	InterveneAfterSec int64 `yaml:"intervene_after_sec"`
	// RetryBackoffSec MySQL 不可用退避间隔（秒）
	RetryBackoffSec int64 `yaml:"retry_backoff_sec"`
}

// Load 加载配置：mainPath 必选；localPath 可选（不存在时静默跳过）。
// 两份文件内容均做 ${VAR} 环境变量展开。
func Load(mainPath string, localPath string) (*Config, error) {
	cfg := defaults()

	mainData, err := os.ReadFile(mainPath)
	if err != nil {
		return nil, perr.Trace(err)
	}
	if err := yaml.Unmarshal(expandEnv(mainData), cfg); err != nil {
		return nil, perr.Trace(err)
	}

	if localPath != "" {
		localData, err := os.ReadFile(localPath)
		if err == nil {
			if err := yaml.Unmarshal(expandEnv(localData), cfg); err != nil {
				return nil, perr.Trace(err)
			}
		} else if !os.IsNotExist(err) {
			return nil, perr.Trace(err)
		}
	}
	return cfg, nil
}

// defaults 返回带默认值的配置基线，覆盖文件只需写差异字段。
func defaults() *Config {
	return &Config{
		HTTP: HTTPConfig{
			Port: 8080,
		},
		MySQL: MySQLConfig{
			Host:               "127.0.0.1",
			Port:               3306,
			User:               "root",
			Password:           "",
			DBName:             "wbflow",
			MaxOpenConns:       20,
			MaxIdleConns:       10,
			ConnMaxLifetimeSec: 3600,
		},
		Log: LogConfig{
			Level: "info",
			Mode:  "dev",
			Path:  "logs/rd.log",
		},
		Artifacts: ArtifactsConfig{
			DocsRoot: "docs",
		},
		Scanner: ScannerConfig{
			Enabled:           true,
			IntervalSec:       30,
			FollowupAfterSec:  300,
			InterveneAfterSec: 600,
			RetryBackoffSec:   5,
		},
	}
}

// expandEnv 对 yaml 内容做 ${VAR} 展开。
func expandEnv(data []byte) []byte {
	return []byte(os.ExpandEnv(string(data)))
}
