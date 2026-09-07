// jsontext 单元测试：TEXT 列 JSON 文本序列化/反序列化（无 IO）。
package utils_test

import (
	"testing"

	"gotest.tools/v3/assert"

	"github.com/huzilin/wbflow/rd/pkg/utils"
)

type demoPayload struct {
	Name  string   `json:"name"`
	Items []string `json:"items"`
}

func TestUnit_MarshalJSONText_NilSliceIsEmptyArray(t *testing.T) {
	// nil 切片统一 "[]"（TEXT 列 NOT NULL DEFAULT '[]' 约定）
	got, err := utils.MarshalJSONText(nilStringSlice())
	assert.NilError(t, err)
	assert.Equal(t, got, "[]")

	var nilMap map[string]string // 非 nil 切片语义不受影响：nil map 原样 null
	got, err = utils.MarshalJSONText(nilMap)
	assert.NilError(t, err)
	assert.Equal(t, got, "null")
}

func TestUnit_MarshalJSONText_RoundTrip(t *testing.T) {
	in := demoPayload{Name: "评审", Items: []string{"a", "b"}}
	text, err := utils.MarshalJSONText(in)
	assert.NilError(t, err)
	assert.Equal(t, text, `{"name":"评审","items":["a","b"]}`)

	var out demoPayload
	assert.NilError(t, utils.UnmarshalJSONText(text, &out))
	assert.DeepEqual(t, in, out)
}

func TestUnit_UnmarshalJSONText_BlankIsNoop(t *testing.T) {
	var out demoPayload
	assert.NilError(t, utils.UnmarshalJSONText("", &out))
	assert.NilError(t, utils.UnmarshalJSONText("   ", &out))
	assert.Equal(t, out.Name, "")
}

func TestUnit_UnmarshalJSONText_InvalidTextErrors(t *testing.T) {
	var out demoPayload
	err := utils.UnmarshalJSONText("{not-json", &out)
	assert.ErrorContains(t, err, "invalid character")
}

func nilStringSlice() []string { return nil }
