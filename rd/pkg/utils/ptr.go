// Package utils 本文件：标量微 helper（指针化 / 指针解引用 / 审计文本化 / TINYINT(1)
// 布尔转换）——service/repo 多包共用，按 rd/README.md 口径收敛于此，业务代码不内联。
package utils

import "strconv"

// StrPtr 字符串指针（NULL 保留清单列 / 审计文本构造）。
func StrPtr(s string) *string { return &s }

// DerefStr 字符串指针零值收敛（nil → 空串，NULL 列出参与审计旧值口径）。
func DerefStr(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

// FormatMillis 毫秒时间戳/序号审计文本（audit_log old/new 值 TEXT 列字符串口径）。
func FormatMillis(v int64) string {
	return strconv.FormatInt(v, 10)
}

// FormatMillisPtr 毫秒时间戳指针文本（nil → 空串）。
func FormatMillisPtr(v *int64) string {
	if v == nil {
		return ""
	}
	return FormatMillis(*v)
}

// BoolToInt64 TINYINT(1) 列取值转换。
func BoolToInt64(b bool) int64 {
	if b {
		return 1
	}
	return 0
}
