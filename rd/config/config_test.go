package config

import (
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
)

func TestUnit_Load_MainAndLocalOverlay(t *testing.T) {
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "config.yaml")
	localPath := filepath.Join(dir, "config.local.yaml")

	mainYAML := `
http:
  port: 8080
mysql:
  host: 127.0.0.1
  port: 3306
  user: root
  password: ${WBFLOW_TEST_PASSWORD}
  dbname: wbflow
log:
  level: info
  mode: dev
  path: logs/rd.log
`
	localYAML := `
mysql:
  host: 192.168.3.100
  password: ${WBFLOW_TEST_PASSWORD}
log:
  level: debug
`
	assert.NilError(t, os.WriteFile(mainPath, []byte(mainYAML), 0o600))
	assert.NilError(t, os.WriteFile(localPath, []byte(localYAML), 0o600))

	t.Setenv("WBFLOW_TEST_PASSWORD", "p@ss:word")

	cfg, err := Load(mainPath, localPath)
	assert.NilError(t, err)
	assert.Equal(t, cfg.HTTP.Port, int64(8080))
	// 本地覆盖生效
	assert.Equal(t, cfg.MySQL.Host, "192.168.3.100")
	assert.Equal(t, cfg.Log.Level, "debug")
	// env 展开生效（含特殊字符 @ :）
	assert.Equal(t, cfg.MySQL.Password, "p@ss:word")
	// 未覆盖字段保持主配置
	assert.Equal(t, cfg.MySQL.DBName, "wbflow")
	// 默认值兜底生效
	assert.Equal(t, cfg.MySQL.MaxOpenConns, int64(20))
}

func TestUnit_Load_MissingLocalIsOK(t *testing.T) {
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "config.yaml")
	assert.NilError(t, os.WriteFile(mainPath, []byte("http:\n  port: 9090\n"), 0o600))

	cfg, err := Load(mainPath, filepath.Join(dir, "not-exist.yaml"))
	assert.NilError(t, err)
	assert.Equal(t, cfg.HTTP.Port, int64(9090))
	assert.Equal(t, cfg.MySQL.DBName, "wbflow")
	// auth 段缺省=空 token（fail closed 口径由 handler 层执行）
	assert.Equal(t, cfg.Auth.Token, "")
}

func TestUnit_Load_AuthTokenEnvExpand(t *testing.T) {
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "config.yaml")
	mainYAML := "auth:\n  token: ${WBFLOW_TEST_AUTH_TOKEN}\n"
	assert.NilError(t, os.WriteFile(mainPath, []byte(mainYAML), 0o600))

	t.Setenv("WBFLOW_TEST_AUTH_TOKEN", "tok-abc")
	cfg, err := Load(mainPath, "")
	assert.NilError(t, err)
	assert.Equal(t, cfg.Auth.Token, "tok-abc")
}

func TestUnit_Load_MainMissingIsErr(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "no.yaml"), "")
	assert.Assert(t, err != nil)
}

// T10 扫描器段：阈值参数化可覆盖（演练加速口径）+ 默认值兜底（5min 催办 / 10min
// 介入，spec §2「agent 无响应」行参数化配置）。
func TestUnit_Load_ScannerOverlay(t *testing.T) {
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "config.yaml")
	mainYAML := `
http:
  port: 8080
scanner:
  enabled: true
  interval_sec: 1
  followup_after_sec: 5
  intervene_after_sec: 10
  retry_backoff_sec: 1
`
	assert.NilError(t, os.WriteFile(mainPath, []byte(mainYAML), 0o600))

	cfg, err := Load(mainPath, "")
	assert.NilError(t, err)
	assert.Equal(t, cfg.Scanner.IntervalSec, int64(1))
	assert.Equal(t, cfg.Scanner.FollowupAfterSec, int64(5))
	assert.Equal(t, cfg.Scanner.InterveneAfterSec, int64(10))

	// 无 scanner 段：默认值兜底
	minimal := filepath.Join(dir, "minimal.yaml")
	assert.NilError(t, os.WriteFile(minimal, []byte("http:\n  port: 8080\n"), 0o600))
	cfg2, err := Load(minimal, "")
	assert.NilError(t, err)
	assert.Equal(t, cfg2.Scanner.Enabled, true)
	assert.Equal(t, cfg2.Scanner.IntervalSec, int64(30))
	assert.Equal(t, cfg2.Scanner.FollowupAfterSec, int64(300))
	assert.Equal(t, cfg2.Scanner.InterveneAfterSec, int64(600))
	assert.Equal(t, cfg2.Scanner.RetryBackoffSec, int64(5))
}
