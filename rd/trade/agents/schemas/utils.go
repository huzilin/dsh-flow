package schemas

import "strconv"

// JoinLines 将多行字符串连接成一个字符串
func JoinLines(parts ...string) string {
	result := ""
	for i, part := range parts {
		if i > 0 {
			result += "\n"
		}
		result += part
	}
	return result
}

// Float64ToString 将 float64 转换为字符串
func Float64ToString(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

// FormatFloat 格式化浮点数为字符串
func FormatFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', 1, 64)
}

// Capitalize 将字符串首字母大写
func Capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	if s[0] >= 'a' && s[0] <= 'z' {
		return string(s[0]-32) + s[1:]
	}
	return s
}
