package models

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/shopspring/decimal"
)

func formatDecimal(v decimal.Decimal) string {
	if v.IsZero() {
		return "0"
	}

	abs := v.Abs()
	var prefix string
	if v.LessThan(decimal.Zero) {
		prefix = "-"
	}

	if abs.GreaterThanOrEqual(decimal.NewFromInt(1000000000)) {
		return prefix + abs.Div(decimal.NewFromInt(1000000000)).StringFixed(2) + "B"
	}
	if abs.GreaterThanOrEqual(decimal.NewFromInt(1000000)) {
		return prefix + abs.Div(decimal.NewFromInt(1000000)).StringFixed(2) + "M"
	}
	if abs.GreaterThanOrEqual(decimal.NewFromInt(1000)) {
		return prefix + abs.Div(decimal.NewFromInt(1000)).StringFixed(2) + "K"
	}

	return v.StringFixed(4)
}

func formatInt(v int64) string {
	return strconv.FormatInt(v, 10)
}

func formatIntInt(v int) string {
	return strconv.Itoa(v)
}

func FormatTable(rows []map[string]any, title string, limit int) string {
	if len(rows) == 0 {
		return title + "\n\n无数据"
	}

	output := title + "\n\n"

	keys := extractKeys(rows)
	if len(keys) == 0 {
		return output + "无数据"
	}

	output += "| " + strings.Join(keys, " | ") + " |\n"
	output += "| " + strings.Repeat("--- | ", len(keys)) + "\n"

	count := min(len(rows), limit)
	for i := 0; i < count; i++ {
		row := rows[i]
		var values []string
		for _, key := range keys {
			val := row[key]
			var strVal string
			switch v := val.(type) {
			case decimal.Decimal:
				strVal = formatDecimal(v)
			case float64:
				strVal = formatDecimal(decimal.NewFromFloat(v))
			case int:
				strVal = formatIntInt(v)
			case int64:
				strVal = formatInt(v)
			default:
				strVal = fmt.Sprintf("%v", val)
			}
			values = append(values, strVal)
		}
		output += "| " + strings.Join(values, " | ") + " |\n"
	}

	if len(rows) > limit {
		output += "\n*Showing " + formatIntInt(limit) + " of " + formatIntInt(len(rows)) + " items*\n"
	}

	return output
}

func extractKeys(rows []map[string]any) []string {
	if len(rows) == 0 {
		return []string{}
	}
	keys := make([]string, 0, len(rows[0]))
	for k := range rows[0] {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
