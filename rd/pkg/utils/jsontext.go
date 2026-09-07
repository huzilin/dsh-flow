// Package utils 通用工具（rd/README.md 约定：序列化等工具收敛于此，业务代码不内联）。
// 本文件：TEXT 列 JSON 文本序列化/反序列化 helper（spec §0.2——TEXT 列存 JSON 文本，
// 结构定义以 Go struct 为单一口径，载体见 repo/payload.go）。
package utils

import (
	"encoding/json"
	"reflect"
	"strings"

	perr "github.com/pingcap/errors"
)

// MarshalJSONText 序列化为 JSON 文本（TEXT 列口径）。
// nil 切片统一输出 "[]"（TEXT 列 NOT NULL DEFAULT '[]' 约定）；其余值原样序列化。
func MarshalJSONText(v any) (string, error) {
	if isNilSlice(v) {
		return "[]", nil
	}
	data, err := json.Marshal(v)
	if err != nil {
		return "", perr.Trace(err)
	}
	return string(data), nil
}

// UnmarshalJSONText 反序列化 JSON 文本到 v（传指针）。
// 空白文本按「列未填充」处理：v 保持零值不报错。
func UnmarshalJSONText(text string, v any) error {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	if err := json.Unmarshal([]byte(text), v); err != nil {
		return perr.Trace(err)
	}
	return nil
}

// isNilSlice 判定 v 是否 nil 切片（非切片/nil 接口均 false）。
func isNilSlice(v any) bool {
	rv := reflect.ValueOf(v)
	return rv.Kind() == reflect.Slice && rv.IsNil()
}
