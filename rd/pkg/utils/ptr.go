// Package utils 本文件：标量微 helper（指针化 / TINYINT(1) 布尔转换）——
// service 多文件共用，按 rd/README.md 口径收敛于此，业务代码不内联。
package utils

// StrPtr 字符串指针（NULL 保留清单列 / 审计文本构造）。
func StrPtr(s string) *string { return &s }

// BoolToInt64 TINYINT(1) 列取值转换。
func BoolToInt64(b bool) int64 {
	if b {
		return 1
	}
	return 0
}
