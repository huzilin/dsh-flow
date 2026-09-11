// llm 包本文件：围栏剥离与平衡块提取（novel be/llm/dsh.go C2 原样 copy）。
// 模型实测会包 ```json 围栏或夹带解释文字，剥离不依赖围栏字面量——按大括号/
// 方括号配对深度取首个平衡块；StripFencesAny 带字符串内转义感知（inStr 状态机）。
package llm

import "strings"

// StripFences 剥 markdown 代码围栏与首尾杂 text，取首个 {..} 平衡块
// （真机 GLM 实测会包 ```json）。导出供各 JSON 通路复用同一解析口径。
func StripFences(s string) string {
	i := strings.Index(s, "{")
	if i < 0 {
		return s
	}
	depth := 0
	for j := i; j < len(s); j++ {
		switch s[j] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return s[i : j+1]
			}
		}
	}
	return s
}

// StripFencesAny 剥围栏与首尾杂 text，取首个 {..} 或 [..] 平衡块（产物可为裸数组；
// 与 StripFences 的差异仅在支持数组顶层——评审通路保持对象口径不动）。
func StripFencesAny(s string) string {
	i := strings.IndexAny(s, "{[")
	if i < 0 {
		return s
	}
	open, close := s[i], map[byte]byte{'{': '}', '[': ']'}[s[i]]
	depth := 0
	inStr := false
	for j := i; j < len(s); j++ {
		c := s[j]
		if inStr {
			if c == '\\' {
				j++
			} else if c == '"' {
				inStr = false
			}
			continue
		}
		switch {
		case c == '"':
			inStr = true
		case c == open:
			depth++
		case c == close:
			depth--
			if depth == 0 {
				return s[i : j+1]
			}
		}
	}
	return s[i:]
}

// truncate 错误信息截断工具（防模型异常产出刷爆错误字段）。
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
