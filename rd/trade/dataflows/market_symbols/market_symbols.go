package market_symbols

import (
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

var (
	ErrEmptySymbol        = errors.New("ticker symbol cannot be empty")
	ErrUnsupportedFormat  = errors.New("unsupported symbol format")
	ErrUnsupportedAshare  = errors.New("unsupported A-share symbol")
	ErrUnsupportedHKShare = errors.New("unsupported Hong Kong stock symbol")
)

func NormalizeAshareSymbol(symbol string) (string, error) {
	normalized := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(symbol), " ", ""))
	if normalized == "" {
		return "", ErrEmptySymbol
	}

	exchangePrefixRegex := regexp.MustCompile(`^(SH|SZ|BJ)(\d{6})$`)
	if match := exchangePrefixRegex.FindStringSubmatch(normalized); match != nil {
		return match[2] + "." + match[1], nil
	}

	exchangeSuffixRegex := regexp.MustCompile(`^(\d{6})\.(SH|SZ|BJ)$`)
	if match := exchangeSuffixRegex.FindStringSubmatch(normalized); match != nil {
		return match[1] + "." + match[2], nil
	}

	digitsRegex := regexp.MustCompile(`^\d{6}$`)
	if match := digitsRegex.FindStringSubmatch(normalized); match != nil {
		code := match[0]
		exchange := inferExchange(code)
		if exchange == "" {
			return "", ErrUnsupportedAshare
		}
		return code + "." + exchange, nil
	}

	return "", ErrUnsupportedAshare
}

func inferExchange(code string) string {
	if len(code) < 3 {
		return ""
	}
	prefix := code[:3]
	switch prefix[:2] {
	case "60":
		if prefix == "600" || prefix == "601" || prefix == "603" || prefix == "605" || prefix == "68" {
			return "SH"
		}
	case "00":
		if prefix == "000" || prefix == "001" || prefix == "002" || prefix == "003" || prefix == "300" || prefix == "301" {
			return "SZ"
		}
	case "30":
		if prefix == "300" || prefix == "301" {
			return "SZ"
		}
	case "43", "83", "87", "92":
		return "BJ"
	}
	return ""
}

func NormalizeHKSymbol(symbol string) (string, error) {
	normalized := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(symbol), " ", ""))
	if normalized == "" {
		return "", ErrEmptySymbol
	}

	exchangePrefixRegex := regexp.MustCompile(`^HK(\d{4,5})$`)
	if match := exchangePrefixRegex.FindStringSubmatch(normalized); match != nil {
		code := match[1]
		for len(code) < 5 {
			code = "0" + code
		}
		return code + ".HK", nil
	}

	exchangeSuffixRegex := regexp.MustCompile(`^(\d{4,5})\.HK$`)
	if match := exchangeSuffixRegex.FindStringSubmatch(normalized); match != nil {
		code := match[1]
		for len(code) < 5 {
			code = "0" + code
		}
		return code + ".HK", nil
	}

	digitsRegex := regexp.MustCompile(`^\d{4,5}$`)
	if match := digitsRegex.FindStringSubmatch(normalized); match != nil {
		code := match[0]
		for len(code) < 5 {
			code = "0" + code
		}
		return code + ".HK", nil
	}

	return "", ErrUnsupportedHKShare
}

func NormalizeMarketSymbol(symbol string) (string, error) {
	if result, err := NormalizeAshareSymbol(symbol); err == nil {
		return result, nil
	}
	return NormalizeHKSymbol(symbol)
}

func IsHKSymbol(symbol string) bool {
	result, err := NormalizeHKSymbol(symbol)
	return err == nil && strings.HasSuffix(result, ".HK")
}

func IsAshareSymbol(symbol string) bool {
	result, err := NormalizeAshareSymbol(symbol)
	return err == nil && (strings.HasSuffix(result, ".SH") || strings.HasSuffix(result, ".SZ") || strings.HasSuffix(result, ".BJ"))
}

func GetMarketLabel(symbol string) string {
	if IsHKSymbol(symbol) {
		return "Hong Kong stock"
	}
	return "A-share"
}

func ToPlainSymbol(symbol string) (string, error) {
	normalized, err := NormalizeAshareSymbol(symbol)
	if err != nil {
		return "", err
	}
	parts := strings.SplitN(normalized, ".", 2)
	return parts[0], nil
}

func ToExchangePrefixedSymbol(symbol string) (string, error) {
	normalized, err := NormalizeAshareSymbol(symbol)
	if err != nil {
		return "", err
	}
	parts := strings.SplitN(normalized, ".", 2)
	if len(parts) != 2 {
		return "", ErrUnsupportedAshare
	}
	return parts[1] + parts[0], nil
}
