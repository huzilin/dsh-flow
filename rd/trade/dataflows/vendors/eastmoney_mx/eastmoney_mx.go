package eastmoney_mx

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/huzilin/trade/dataflows/errs"
	"github.com/huzilin/trade/dataflows/interfaces"
	"github.com/huzilin/trade/dataflows/market_symbols"
	"github.com/huzilin/trade/dataflows/normalizer"
	"github.com/huzilin/trade/dataflows/vendors/utils"
	"github.com/pkg/errors"
)

const (
	MXQueryURL   = "https://mkapi2.dfcfs.com/finskillshub/api/claw/query"
	MXSearchURL  = "https://mkapi2.dfcfs.com/finskillshub/api/claw/news-search"
	MXTimeout    = 30
	MXRetryDelay = 1.0
	MXMaxRetries = 3
)

var MXInfoTypeLabels = map[string]string{
	"REPORT":       "研报",
	"NEWS":         "新闻",
	"ANNOUNCEMENT": "公告",
}

var MXAPIKeyValue = "mkt_t-eOQuwV5b7B4yvND9xL93thl_wpeNZtxHlD46Eyn3Y"

func MXAPIKey() (string, error) {
	if MXAPIKeyValue != "" {
		return MXAPIKeyValue, nil
	}
	return "", errs.NewVendorNotConfiguredError("eastmoney_mx", "MX_APIKEY")
}

func postMX(url string, payload map[string]any) ([]byte, error) {
	key, err := MXAPIKey()
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: time.Duration(MXTimeout) * time.Second}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal payload")
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", key)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")

	for attempt := 1; attempt <= MXMaxRetries; attempt++ {
		resp, err := client.Do(req)
		if err != nil {
			if attempt < MXMaxRetries && utils.IsRetryableError(err) {
				time.Sleep(time.Duration(MXRetryDelay*float64(attempt)) * time.Second)
				continue
			}
			return nil, errors.Wrap(err, "MX request failed")
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusTooManyRequests {
			if attempt < MXMaxRetries {
				time.Sleep(time.Duration(MXRetryDelay*float64(attempt)*2) * time.Second)
				continue
			}
			return nil, errs.NewVendorRateLimitError("eastmoney_mx", url)
		}

		if resp.StatusCode != http.StatusOK {
			return nil, errors.Errorf("MX API returned status %d", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, errors.Wrap(err, "failed to read response")
		}

		var result map[string]any
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, errors.Wrap(err, "failed to parse MX response")
		}

		if status, ok := result["status"].(float64); ok && status != 0 {
			msg := fmt.Sprintf("%v", result["message"])
			if msg == "" {
				msg = fmt.Sprintf("MX API returned status=%v", status)
			}
			return nil, errors.New(msg)
		}

		return body, nil
	}

	return nil, errors.New("max retries exceeded")
}

func extractQueryTables(result map[string]any) []map[string]any {
	var tables []map[string]any

	data, _ := result["data"].(map[string]any)
	inner, _ := data["data"].(map[string]any)

	var searchResult map[string]any
	if sr, ok := inner["searchDataResultDTO"].(map[string]any); ok {
		searchResult = sr
	} else if sr, ok := data["searchDataResultDTO"].(map[string]any); ok {
		searchResult = sr
	}

	if dtList, ok := searchResult["dataTableDTOList"].([]any); ok {
		for _, item := range dtList {
			if dt, ok := item.(map[string]any); ok {
				tables = append(tables, dt)
			}
		}
	} else if dtList, ok := data["dataTableDTOList"].([]any); ok {
		for _, item := range dtList {
			if dt, ok := item.(map[string]any); ok {
				tables = append(tables, dt)
			}
		}
	}

	return tables
}

func normalizeNameMap(nameMap any) map[string]string {
	result := make(map[string]string)

	switch v := nameMap.(type) {
	case map[string]any:
		for k, val := range v {
			if valStr := fmt.Sprintf("%v", val); valStr != "" {
				result[k] = valStr
			}
		}
	case []any:
		for i, val := range v {
			if valStr := fmt.Sprintf("%v", val); valStr != "" {
				result[fmt.Sprintf("%d", i)] = valStr
			}
		}
	}

	return result
}

func resolveIndicatorLabel(key string, nameMap map[string]string) string {
	if label, ok := nameMap[key]; ok {
		return label
	}
	return key
}

func dtoToRows(dto map[string]any) []map[string]any {
	table, _ := dto["table"].(map[string]any)
	if table == nil || len(table) == 0 {
		return nil
	}

	nameMap := normalizeNameMap(dto["nameMap"])
	indicatorOrder, _ := dto["indicatorOrder"].([]any)

	dataKeys := make([]string, 0)
	for k := range table {
		if k != "headName" {
			dataKeys = append(dataKeys, k)
		}
	}

	preferred := make([]string, 0)
	seen := make(map[string]bool)
	for _, key := range indicatorOrder {
		keyStr := fmt.Sprintf("%v", key)
		if _, ok := table[keyStr]; ok && !seen[keyStr] {
			preferred = append(preferred, keyStr)
			seen[keyStr] = true
		}
	}
	for _, key := range dataKeys {
		if !seen[key] {
			preferred = append(preferred, key)
			seen[key] = true
		}
	}

	headValues, _ := table["headName"].([]any)
	if headValues != nil && len(headValues) > 0 {
		var rows []map[string]any
		for rowIndex, headValue := range headValues {
			row := map[string]any{"日期": fmt.Sprintf("%v", headValue)}
			for _, key := range preferred {
				rawValues, _ := table[key].([]any)
				var value string
				if rawValues != nil && rowIndex < len(rawValues) {
					value = fmt.Sprintf("%v", rawValues[rowIndex])
				}
				row[resolveIndicatorLabel(key, nameMap)] = value
			}
			rows = append(rows, row)
		}
		return rows
	}

	var rows []map[string]any
	for _, key := range preferred {
		rawValues, _ := table[key].([]any)
		var value string
		if rawValues != nil && len(rawValues) > 0 {
			value = fmt.Sprintf("%v", rawValues[0])
		} else if v, ok := table[key]; ok {
			value = fmt.Sprintf("%v", v)
		}
		rows = append(rows, map[string]any{
			"指标": resolveIndicatorLabel(key, nameMap),
			"数值": value,
		})
	}
	return rows
}

func extractSearchItems(result map[string]any) []map[string]any {
	var items []map[string]any

	data, _ := result["data"].(map[string]any)
	inner, _ := data["data"].(map[string]any)

	var llmResponse map[string]any
	if lr, ok := inner["llmSearchResponse"].(map[string]any); ok {
		llmResponse = lr
	} else if lr, ok := inner["searchResponse"].(map[string]any); ok {
		llmResponse = lr
	}

	if dataList, ok := llmResponse["data"].([]any); ok {
		for _, item := range dataList {
			if dt, ok := item.(map[string]any); ok {
				items = append(items, dt)
			}
		}
	}

	return items
}

func filterMXItemsByDate(items []map[string]any, startDate, endDate string) []map[string]any {
	start, _ := time.Parse("2006-01-02", startDate)
	end, _ := time.Parse("2006-01-02", endDate)
	end = end.AddDate(0, 0, 1).Add(-time.Second)

	var filtered []map[string]any
	for _, item := range items {
		dateStr := fmt.Sprintf("%v", item["date"])
		var published time.Time
		var err error

		patterns := []string{"2006-01-02 15:04:05", "2006-01-02T15:04:05", "2006-01-02"}
		for _, pattern := range patterns {
			published, err = time.Parse(pattern, dateStr)
			if err == nil {
				break
			}
		}

		if err != nil {
			continue
		}

		if published.After(start.Add(-time.Second)) && published.Before(end.Add(time.Second)) {
			enriched := make(map[string]any)
			for k, v := range item {
				enriched[k] = v
			}
			enriched["_published"] = published
			filtered = append(filtered, enriched)
		}
	}

	sort.Slice(filtered, func(i, j int) bool {
		pi, _ := filtered[i]["_published"].(time.Time)
		pj, _ := filtered[j]["_published"].(time.Time)
		return pi.After(pj)
	})

	return filtered
}

func filterMXItemsBySymbol(items []map[string]any, plainSymbol string) []map[string]any {
	var matched []map[string]any
	for _, item := range items {
		secuList, _ := item["secuList"].([]any)
		if secuList == nil {
			continue
		}
		for _, secu := range secuList {
			if entry, ok := secu.(map[string]any); ok {
				code := strings.TrimSpace(fmt.Sprintf("%v", entry["secuCode"]))
				if code == plainSymbol {
					matched = append(matched, item)
					break
				}
			}
		}
	}
	if len(matched) > 0 {
		return matched
	}
	return items
}

func preferMXInformationType(items []map[string]any, infoType string) []map[string]any {
	if infoType == "" {
		return items
	}
	var matched []map[string]any
	for _, item := range items {
		itype := strings.ToUpper(strings.TrimSpace(fmt.Sprintf("%v", item["informationType"])))
		if itype == strings.ToUpper(infoType) {
			matched = append(matched, item)
		}
	}
	if len(matched) > 0 {
		return matched
	}
	return items
}

func formatMXSearchRows(items []map[string]any, title string, rows int) string {
	if len(items) == 0 {
		return fmt.Sprintf("%s\n\n暂无数据。", title)
	}

	if rows > len(items) {
		rows = len(items)
	}

	var formattedRows []map[string]any
	for _, item := range items[:rows] {
		var secuNames []string
		secuList, _ := item["secuList"].([]any)
		for _, secu := range secuList {
			if entry, ok := secu.(map[string]any); ok {
				name := strings.TrimSpace(fmt.Sprintf("%v", entry["secuName"]))
				code := strings.TrimSpace(fmt.Sprintf("%v", entry["secuCode"]))
				if name != "" || code != "" {
					if name != "" && code != "" {
						secuNames = append(secuNames, fmt.Sprintf("%s(%s)", name, code))
					} else {
						secuNames = append(secuNames, name+code)
					}
				}
			}
		}

		var publishedStr string
		if t, ok := item["_published"].(time.Time); ok {
			publishedStr = t.Format("2006-01-02 15:04:05")
		} else {
			publishedStr = fmt.Sprintf("%v", item["date"])
		}

		infoType := strings.ToUpper(strings.TrimSpace(fmt.Sprintf("%v", item["informationType"])))
		typeLabel := MXInfoTypeLabels[infoType]
		if typeLabel == "" {
			typeLabel = infoType
		}

		content := fmt.Sprintf("%v", item["content"])
		if content == "" {
			content = fmt.Sprintf("%v", item["trunk"])
		}

		formattedRows = append(formattedRows, map[string]any{
			"发布时间": publishedStr,
			"类型":   typeLabel,
			"标题":   fmt.Sprintf("%v", item["title"]),
			"内容":   utils.SafeTruncate(content, 180),
			"机构":   fmt.Sprintf("%v", item["insName"]),
			"关联证券": strings.Join(secuNames, "；"),
		})
	}

	return utils.FormatTable(formattedRows, title, rows)
}

func EastmoneyMXGetFundamentals(ctx context.Context, args ...any) (string, error) {
	if len(args) < 1 {
		return "", errors.New("get_fundamentals requires at least 1 argument: ticker")
	}

	ticker := fmt.Sprintf("%v", args[0])

	var currDate string
	if len(args) > 1 {
		currDate = fmt.Sprintf("%v", args[1])
	}

	normalizedSymbol, err := market_symbols.NormalizeAshareSymbol(ticker)
	if err != nil {
		return "", errors.Wrap(err, "invalid A-share symbol")
	}

	plainSymbol, err := market_symbols.ToPlainSymbol(ticker)
	if err != nil {
		return "", errors.Wrap(err, "failed to get plain symbol")
	}

	query := fmt.Sprintf("%s 公司概况 主营业务 财务指标 归母净利润 扣非净利润 经营现金流 资产负债率 ROE", plainSymbol)
	if currDate != "" {
		query += fmt.Sprintf(" 截至%s", currDate)
	}

	body, err := postMX(MXQueryURL, map[string]any{"toolQuery": query})
	if err != nil {
		return utils.FormatDataError(fmt.Sprintf("# A-share company profile for %s", normalizedSymbol), err), nil
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return utils.FormatDataError(fmt.Sprintf("# A-share company profile for %s", normalizedSymbol), err), nil
	}

	tables := extractQueryTables(result)
	if len(tables) == 0 {
		return fmt.Sprintf("MX fundamentals returned no structured tables for %s.", normalizedSymbol), nil
	}

	var sections []string
	sections = append(sections, fmt.Sprintf("# A-share company profile for %s", normalizedSymbol))

	for _, dto := range tables {
		title := strings.TrimSpace(fmt.Sprintf("%v", dto["title"]))
		if title == "" {
			title = strings.TrimSpace(fmt.Sprintf("%v", dto["entityName"]))
		}
		if title == "" {
			title = "妙想基本面数据"
		}

		rows := dtoToRows(dto)
		if len(rows) > 0 {
			sections = append(sections, utils.FormatTable(rows, fmt.Sprintf("## %s", title), 12))
		}
	}

	return strings.Join(sections, "\n\n"), nil
}

func EastmoneyMXGetNews(ctx context.Context, args ...any) (string, error) {
	if len(args) < 3 {
		return "", errors.New("get_news requires 3 arguments: ticker, start_date, end_date")
	}

	ticker := fmt.Sprintf("%v", args[0])
	startDate := fmt.Sprintf("%v", args[1])
	endDate := fmt.Sprintf("%v", args[2])

	normalizedSymbol, err := market_symbols.NormalizeAshareSymbol(ticker)
	if err != nil {
		return "", errors.Wrap(err, "invalid A-share symbol")
	}

	plainSymbol, err := market_symbols.ToPlainSymbol(ticker)
	if err != nil {
		return "", errors.Wrap(err, "failed to get plain symbol")
	}

	query := fmt.Sprintf("%s 个股新闻 公司资讯 研报 %s 至 %s", plainSymbol, startDate, endDate)

	body, err := postMX(MXSearchURL, map[string]any{"query": query})
	if err != nil {
		return utils.FormatDataError(fmt.Sprintf("# A-share company news for %s", normalizedSymbol), err), nil
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return utils.FormatDataError(fmt.Sprintf("# A-share company news for %s", normalizedSymbol), err), nil
	}

	items := extractSearchItems(result)
	dated := filterMXItemsByDate(items, startDate, endDate)
	filtered := filterMXItemsBySymbol(dated, plainSymbol)

	if len(filtered) == 0 {
		return fmt.Sprintf("%s 在 %s 到 %s 之间没有匹配的新闻。", normalizedSymbol, startDate, endDate), nil
	}

	return formatMXSearchRows(filtered, fmt.Sprintf("# A-share company news for %s", normalizedSymbol), 20), nil
}

func EastmoneyMXGetMarketNews(ctx context.Context, args ...any) (string, error) {
	currDate := utils.FormatDate(time.Now())
	lookBackDays := 7
	limit := 10

	if len(args) > 0 {
		currDate = fmt.Sprintf("%v", args[0])
	}
	if len(args) > 1 {
		if lb, ok := args[1].(int); ok {
			lookBackDays = lb
		} else if lb, ok := args[1].(float64); ok {
			lookBackDays = int(lb)
		}
	}
	if len(args) > 2 {
		if l, ok := args[2].(int); ok {
			limit = l
		} else if l, ok := args[2].(float64); ok {
			limit = int(l)
		}
	}

	end, _ := time.Parse("2006-01-02", currDate)
	start := end.AddDate(0, 0, -lookBackDays)
	startDate := start.Format("2006-01-02")

	query := fmt.Sprintf("A股 市场 政策 宏观 新闻 %s 至 %s", startDate, currDate)

	body, err := postMX(MXSearchURL, map[string]any{"query": query})
	if err != nil {
		return utils.FormatDataError("# A-share market and policy news", err), nil
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return utils.FormatDataError("# A-share market and policy news", err), nil
	}

	items := extractSearchItems(result)
	filtered := filterMXItemsByDate(items, startDate, currDate)

	if len(filtered) == 0 {
		return fmt.Sprintf("%s 前 %d 天没有可用的市场快讯。", currDate, lookBackDays), nil
	}

	return formatMXSearchRows(filtered, "# A-share market and policy news", limit), nil
}

func EastmoneyMXGetCompanyAnnouncements(ctx context.Context, args ...any) (string, error) {
	if len(args) < 3 {
		return "", errors.New("get_company_announcements requires 3 arguments: ticker, start_date, end_date")
	}

	ticker := fmt.Sprintf("%v", args[0])
	startDate := fmt.Sprintf("%v", args[1])
	endDate := fmt.Sprintf("%v", args[2])

	category := ""
	if len(args) > 3 {
		category = fmt.Sprintf("%v", args[3])
	}

	normalizedSymbol, err := market_symbols.NormalizeAshareSymbol(ticker)
	if err != nil {
		return "", errors.Wrap(err, "invalid A-share symbol")
	}

	plainSymbol, err := market_symbols.ToPlainSymbol(ticker)
	if err != nil {
		return "", errors.Wrap(err, "failed to get plain symbol")
	}

	categoryFragment := ""
	if category != "" && category != "全部" {
		categoryFragment = category + " "
	}

	query := fmt.Sprintf("%s %s公告 %s 至 %s", plainSymbol, categoryFragment, startDate, endDate)

	body, err := postMX(MXSearchURL, map[string]any{"query": query})
	if err != nil {
		return utils.FormatDataError(fmt.Sprintf("# A-share company announcements for %s", normalizedSymbol), err), nil
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return utils.FormatDataError(fmt.Sprintf("# A-share company announcements for %s", normalizedSymbol), err), nil
	}

	items := extractSearchItems(result)
	dated := filterMXItemsByDate(items, startDate, endDate)
	filtered := filterMXItemsBySymbol(dated, plainSymbol)
	filtered = preferMXInformationType(filtered, "ANNOUNCEMENT")

	if len(filtered) == 0 {
		return fmt.Sprintf("%s 在 %s 到 %s 之间没有匹配的公告。", normalizedSymbol, startDate, endDate), nil
	}

	return formatMXSearchRows(filtered, fmt.Sprintf("# A-share company announcements for %s", normalizedSymbol), 20), nil
}

func init() {
}

type EastmoneyMXProvider struct{}

func (p *EastmoneyMXProvider) Name() string {
	return "eastmoney_mx"
}

func (p *EastmoneyMXProvider) Capabilities() []interfaces.Capability {
	return []interfaces.Capability{
		interfaces.CapabilityFundamentals,
		interfaces.CapabilityNews,
		interfaces.CapabilityCompanyAnnouncements,
	}
}

func (p *EastmoneyMXProvider) Markets() []normalizer.MarketType {
	return []normalizer.MarketType{normalizer.MarketAshare}
}

var _ interfaces.Provider = (*EastmoneyMXProvider)(nil)
