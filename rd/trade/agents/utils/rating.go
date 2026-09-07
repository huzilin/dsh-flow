package utils

import (
	"strings"
)

// Ratings5Tier 标准5级评级（从最看涨到最看跌）
var Ratings5Tier = []string{"Buy", "Overweight", "Hold", "Underweight", "Sell"}

var ratingSet = map[string]struct{}{
	"buy":        {},
	"overweight": {},
	"hold":       {},
	"underweight": {},
	"sell":       {},
}

// ParseRating 从文本中启发式提取5级评级
// 策略：
//   1. 查找显式的 "Rating: X" 标签（兼容 markdown 加粗）
//   2. 回退到在文本中查找第一个5级评级词
// 返回首字母大写的评级字符串，若未找到则返回 defaultRating
func ParseRating(text, defaultRating string) string {
	if defaultRating == "" {
		defaultRating = "Hold"
	}

	lines := strings.Split(text, "\n")
	for _, line := range lines {
		// 尝试匹配 "Rating: X" 或 "rating - X" 或 "Rating: **X**"
		lowerLine := strings.ToLower(line)
		idx := strings.Index(lowerLine, "rating")
		if idx == -1 {
			continue
		}
		rest := line[idx+len("rating"):]
		rest = strings.TrimLeft(rest, " \t*")
		if len(rest) == 0 {
			continue
		}
		if rest[0] != ':' && rest[0] != '-' {
			continue
		}
		rest = rest[1:]
		rest = strings.TrimLeft(rest, " \t*")
		words := strings.Fields(rest)
		if len(words) == 0 {
			continue
		}
		word := strings.ToLower(strings.Trim(words[0], "*:.,"))
		if _, ok := ratingSet[word]; ok {
			return capitalize(word)
		}
	}

	// 回退：在文本中查找第一个5级评级词
	for _, line := range lines {
		words := strings.Fields(line)
		for _, w := range words {
			word := strings.ToLower(strings.Trim(w, "*:.,"))
			if _, ok := ratingSet[word]; ok {
				return capitalize(word)
			}
		}
	}

	return defaultRating
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
