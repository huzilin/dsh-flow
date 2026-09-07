package consts

// AnalystType 分析师类型枚举
type AnalystType string

const (
	// AnalystMarket 市场分析师
	AnalystMarket AnalystType = "market"
	// AnalystSocial 社交情绪分析师
	AnalystSocial AnalystType = "social"
	// AnalystNews 新闻分析师
	AnalystNews AnalystType = "news"
	// AnalystFundamentals 基本面分析师
	AnalystFundamentals AnalystType = "fundamentals"
)

// AllAnalystTypes 返回所有分析师类型
func AllAnalystTypes() []AnalystType {
	return []AnalystType{
		AnalystMarket,
		AnalystSocial,
		AnalystNews,
		AnalystFundamentals,
	}
}

// AnalystTypeStrings 返回所有分析师类型的字符串切片
func AnalystTypeStrings() []string {
	return []string{
		string(AnalystMarket),
		string(AnalystSocial),
		string(AnalystNews),
		string(AnalystFundamentals),
	}
}
