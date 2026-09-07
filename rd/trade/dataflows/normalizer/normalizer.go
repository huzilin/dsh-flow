// Package normalizer 提供股票符号标准化功能
package normalizer

import (
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

// MarketType 市场类型
type MarketType string

const (
	MarketAshare  MarketType = "A-share"  // A股市场
	MarketHKShare MarketType = "HK-share" // 港股市场
	MarketUSShare MarketType = "US-share" // 美股市场
	MarketGlobal  MarketType = "Global"   // 全球市场
	MarketUnknown MarketType = "unknown"  // 未知市场
)

// NormalizedSymbol 标准化后的符号
type NormalizedSymbol struct {
	Original string     // 原始输入
	Standard string     // 标准格式 (如 000001.SZ)
	Plain    string     // 纯代码 (如 000001)
	Market   MarketType // 市场类型
	Exchange string     // 交易所 (SH, SZ, BJ, HK, NASDAQ, NYSE)
}

// SymbolNormalizer 符号标准化接口
type SymbolNormalizer interface {
	// Normalize 标准化符号
	Normalize(symbol string) (NormalizedSymbol, error)
	// DetectMarket 检测市场类型
	DetectMarket(symbol string) MarketType
}

// DefaultNormalizer 默认标准化器
type DefaultNormalizer struct{}

// NewDefaultNormalizer 创建默认标准化器
func NewDefaultNormalizer() *DefaultNormalizer {
	return &DefaultNormalizer{}
}

// Normalize 标准化符号，自动检测市场类型
func (n *DefaultNormalizer) Normalize(symbol string) (NormalizedSymbol, error) {
	normalized := strings.ToUpper(strings.TrimSpace(symbol))
	if normalized == "" {
		return NormalizedSymbol{}, ErrEmptySymbol
	}

	// 尝试 A股标准化
	if result, err := normalizeAshareSymbol(normalized); err == nil {
		return result, nil
	}

	// 尝试港股标准化
	if result, err := normalizeHKSymbol(normalized); err == nil {
		return result, nil
	}

	// 尝试美股标准化（假设为美股）
	if isUSSymbol(normalized) {
		return NormalizedSymbol{
			Original: symbol,
			Standard: normalized,
			Plain:    normalized,
			Market:   MarketUSShare,
			Exchange: inferUSExchange(normalized),
		}, nil
	}

	return NormalizedSymbol{}, errors.Errorf("unsupported symbol format: %s", symbol)
}

// DetectMarket 检测市场类型
func (n *DefaultNormalizer) DetectMarket(symbol string) MarketType {
	normalized := strings.ToUpper(strings.TrimSpace(symbol))

	// A股检测
	if _, err := normalizeAshareSymbol(normalized); err == nil {
		return MarketAshare
	}

	// 港股检测
	if _, err := normalizeHKSymbol(normalized); err == nil {
		return MarketHKShare
	}

	// 美股检测
	if isUSSymbol(normalized) {
		return MarketUSShare
	}

	return MarketUnknown
}

// 错误类型
var (
	ErrEmptySymbol        = errors.New("ticker symbol cannot be empty")
	ErrUnsupportedFormat  = errors.New("unsupported symbol format")
	ErrUnsupportedAshare  = errors.New("unsupported A-share symbol")
	ErrUnsupportedHKShare = errors.New("unsupported Hong Kong stock symbol")
)

// normalizeAshareSymbol 标准化A股符号
func normalizeAshareSymbol(symbol string) (NormalizedSymbol, error) {
	normalized := strings.ToUpper(strings.ReplaceAll(symbol, " ", ""))

	// 匹配带交易所前缀的格式 (SH600000, SZ000001)
	exchangePrefixRegex := regexp.MustCompile(`^(SH|SZ|BJ)(\d{6})$`)
	if match := exchangePrefixRegex.FindStringSubmatch(normalized); match != nil {
		return NormalizedSymbol{
			Original: symbol,
			Standard: match[2] + "." + match[1],
			Plain:    match[2],
			Market:   MarketAshare,
			Exchange: match[1],
		}, nil
	}

	// 匹配带交易所后缀的格式 (600000.SH, 000001.SZ)
	exchangeSuffixRegex := regexp.MustCompile(`^(\d{6})\.(SH|SZ|BJ)$`)
	if match := exchangeSuffixRegex.FindStringSubmatch(normalized); match != nil {
		return NormalizedSymbol{
			Original: symbol,
			Standard: match[1] + "." + match[2],
			Plain:    match[1],
			Market:   MarketAshare,
			Exchange: match[2],
		}, nil
	}

	// 匹配纯数字格式 (600000, 000001)
	digitsRegex := regexp.MustCompile(`^\d{6}$`)
	if match := digitsRegex.FindStringSubmatch(normalized); match != nil {
		code := match[0]
		exchange := inferAshareExchange(code)
		if exchange == "" {
			return NormalizedSymbol{}, ErrUnsupportedAshare
		}
		return NormalizedSymbol{
			Original: symbol,
			Standard: code + "." + exchange,
			Plain:    code,
			Market:   MarketAshare,
			Exchange: exchange,
		}, nil
	}

	return NormalizedSymbol{}, ErrUnsupportedAshare
}

// inferAshareExchange 根据代码推断A股交易所
func inferAshareExchange(code string) string {
	if len(code) < 3 {
		return ""
	}
	prefix := code[:3]
	switch prefix[:2] {
	case "60":
		if prefix == "600" || prefix == "601" || prefix == "603" || prefix == "605" {
			return "SH"
		}
	case "68":
		return "SH" // 科创板
	case "00":
		if prefix == "000" || prefix == "001" || prefix == "002" || prefix == "003" {
			return "SZ"
		}
	case "30":
		if prefix == "300" || prefix == "301" {
			return "SZ" // 创业板
		}
	case "43", "83", "87", "92":
		return "BJ" // 北交所
	}
	return ""
}

// normalizeHKSymbol 标准化港股符号
func normalizeHKSymbol(symbol string) (NormalizedSymbol, error) {
	normalized := strings.ToUpper(strings.ReplaceAll(symbol, " ", ""))

	// 匹配带前缀的格式 (HK0700, HK00700)
	exchangePrefixRegex := regexp.MustCompile(`^HK(\d{4,5})$`)
	if match := exchangePrefixRegex.FindStringSubmatch(normalized); match != nil {
		code := padHKCode(match[1])
		return NormalizedSymbol{
			Original: symbol,
			Standard: code + ".HK",
			Plain:    code,
			Market:   MarketHKShare,
			Exchange: "HK",
		}, nil
	}

	// 匹配带后缀的格式 (00700.HK, 0700.HK)
	exchangeSuffixRegex := regexp.MustCompile(`^(\d{4,5})\.HK$`)
	if match := exchangeSuffixRegex.FindStringSubmatch(normalized); match != nil {
		code := padHKCode(match[1])
		return NormalizedSymbol{
			Original: symbol,
			Standard: code + ".HK",
			Plain:    code,
			Market:   MarketHKShare,
			Exchange: "HK",
		}, nil
	}

	// 匹带纯数字格式 (00700, 700)
	digitsRegex := regexp.MustCompile(`^\d{4,5}$`)
	if match := digitsRegex.FindStringSubmatch(normalized); match != nil {
		code := padHKCode(match[0])
		return NormalizedSymbol{
			Original: symbol,
			Standard: code + ".HK",
			Plain:    code,
			Market:   MarketHKShare,
			Exchange: "HK",
		}, nil
	}

	return NormalizedSymbol{}, ErrUnsupportedHKShare
}

// padHKCode 补齐港股代码到5位
func padHKCode(code string) string {
	for len(code) < 5 {
		code = "0" + code
	}
	return code
}

// isUSSymbol 检测是否为美股符号
func isUSSymbol(symbol string) bool {
	// 美股符号通常是字母组合，长度1-5
	letterRegex := regexp.MustCompile(`^[A-Z]{1,5}$`)
	return letterRegex.MatchString(symbol)
}

// inferUSExchange 推断美股交易所
func inferUSExchange(symbol string) string {
	// 简化处理，假设为 NASDAQ
	// 实际应该根据更多信息判断
	return "NASDAQ"
}

// Normalize 快捷函数，使用默认标准化器
func Normalize(symbol string) (NormalizedSymbol, error) {
	return NewDefaultNormalizer().Normalize(symbol)
}

// DetectMarket 快捷函数，使用默认标准化器检测市场
func DetectMarket(symbol string) MarketType {
	return NewDefaultNormalizer().DetectMarket(symbol)
}
