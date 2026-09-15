// 契约快照自检（C3 双测试互锁的 Go 侧）：默认比对已提交 golden——真相源被改而未
// 再生即红灯（强制 -update 重生成并提交两侧）；-update 重写 golden。FE 侧
// fe/src/api/contract/contract.test.ts 消费同一文件（后端改判定未同步 FE → FE 红灯）。
package contract_test

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/huzilin/wbflow/rd/contract"
)

// goldenPath 相对 contract 包目录（go test cwd=包目录）。
const goldenPath = "../../fe/src/api/contract/golden.json"

var update = flag.Bool("update", false, "重写契约 golden fixture（改真相源后重生成并提交）")

// TestUnit_ContractSnapshot 契约快照自检：生成器输出与已提交 golden 归一化比对
// （键序/缩进不敏感；内容增删改均 FAIL）。
func TestUnit_ContractSnapshot(t *testing.T) {
	got, err := json.Marshal(contract.Build())
	if err != nil {
		t.Fatalf("契约快照序列化失败: %v", err)
	}
	if *update {
		abs, _ := filepath.Abs(goldenPath)
		if err := os.WriteFile(abs, got, 0o644); err != nil {
			t.Fatalf("golden 写入失败（%s）: %v", abs, err)
		}
		t.Logf("golden 已重写: %s", abs)
		return
	}
	wantRaw, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("golden 读取失败（先跑 go test ./contract/ -run TestUnit_ContractSnapshot -update 重生成）: %v", err)
	}
	var want, gotAny any
	if err := json.Unmarshal(wantRaw, &want); err != nil {
		t.Fatalf("golden 非法 JSON: %v", err)
	}
	if err := json.Unmarshal(got, &gotAny); err != nil {
		t.Fatalf("生成器输出非法 JSON: %v", err)
	}
	if !reflect.DeepEqual(want, gotAny) {
		t.Fatalf("契约快照漂移：真相源已变而 golden 未再生——跑 go test ./contract/ -run TestUnit_ContractSnapshot -update 重写并提交，同时检查 FE 消费面是否需要跟进")
	}
}
