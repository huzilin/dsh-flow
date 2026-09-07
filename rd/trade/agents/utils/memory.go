package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/huzilin/trade/logger"
	"go.uber.org/zap"
)

// separator HTML 注释分隔符，不会出现在 LLM 输出中
const separator = "\n\n<!-- ENTRY_END -->\n\n"

// MemoryLogConfig 记忆日志配置
type MemoryLogConfig struct {
	LogPath    string
	MaxEntries int
}

// MemoryEntry 记忆条目（用于检索结果）
type MemoryEntry struct {
	Recommendation string `json:"recommendation"`
	Decision       string `json:"decision"`
	Ticker         string `json:"ticker"`
	Date           string `json:"date"`
	Rating         string `json:"rating"`
}

// TradingMemoryLog 交易决策与复盘日志
// 采用追加式 markdown 格式，支持原子写入和条目轮转
type TradingMemoryLog struct {
	logPath    string
	maxEntries int
	mu         sync.Mutex
}

// NewTradingMemoryLog 创建交易记忆日志
func NewTradingMemoryLog(cfg *MemoryLogConfig) *TradingMemoryLog {
	if cfg == nil {
		cfg = &MemoryLogConfig{}
	}
	tml := &TradingMemoryLog{
		logPath:    cfg.LogPath,
		maxEntries: cfg.MaxEntries,
	}
	if tml.logPath != "" {
		dir := filepath.Dir(tml.logPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			logger.L().Warn("Failed to create memory log dir", zap.String("dir", dir), zap.Error(err))
		}
	}
	return tml
}

// --- Write path (Phase A) ---

// StoreDecision 在 propagate() 结束时追加 pending 条目，不调用 LLM
func (m *TradingMemoryLog) StoreDecision(ticker, tradeDate, finalTradeDecision string) {
	if m.logPath == "" {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// 幂等性检查：快速文本扫描，避免重复写入
	if _, err := os.Stat(m.logPath); err == nil {
		data, err := os.ReadFile(m.logPath)
		if err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				if strings.HasPrefix(line, fmt.Sprintf("[%s | %s |", tradeDate, ticker)) &&
					strings.HasSuffix(line, "| pending]") {
					return
				}
			}
		}
	}

	rating := ParseRating(finalTradeDecision, "Hold")
	tag := fmt.Sprintf("[%s | %s | %s | pending]", tradeDate, ticker, rating)
	entry := fmt.Sprintf("%s\n\nDECISION:\n%s%s", tag, finalTradeDecision, separator)

	f, err := os.OpenFile(m.logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logger.L().Error("Failed to open memory log", zap.Error(err))
		return
	}
	defer f.Close()

	if _, err := f.WriteString(entry); err != nil {
		logger.L().Error("Failed to write memory log", zap.Error(err))
	}
}

// --- Read path ---

// LogEntry 单条日志条目
type LogEntry struct {
	Date       string
	Ticker     string
	Rating     string
	Pending    bool
	Raw        string
	Alpha      string
	Holding    string
	Decision   string
	Reflection string
}

// LoadEntries 解析日志中所有条目
func (m *TradingMemoryLog) LoadEntries() []*LogEntry {
	if m.logPath == "" {
		return nil
	}

	data, err := os.ReadFile(m.logPath)
	if err != nil {
		if !os.IsNotExist(err) {
			logger.L().Error("Failed to read memory log", zap.Error(err))
		}
		return nil
	}

	rawEntries := strings.Split(string(data), separator)
	var entries []*LogEntry
	for _, raw := range rawEntries {
		entry := m.parseEntry(raw)
		if entry != nil {
			entries = append(entries, entry)
		}
	}
	return entries
}

// GetPendingEntries 返回所有 pending 状态的条目（用于 Phase B）
func (m *TradingMemoryLog) GetPendingEntries() []*LogEntry {
	var pending []*LogEntry
	for _, e := range m.LoadEntries() {
		if e.Pending {
			pending = append(pending, e)
		}
	}
	return pending
}

// GetMemories 根据当前情况检索历史记忆
// currSituation: 当前情况描述文本
// nMatches: 返回匹配数量
func (m *TradingMemoryLog) GetMemories(currSituation string, nMatches int) []*MemoryEntry {
	if nMatches <= 0 {
		return nil
	}

	entries := m.LoadEntries()
	if len(entries) == 0 {
		return nil
	}

	var resolved []*LogEntry
	for _, e := range entries {
		if !e.Pending {
			resolved = append(resolved, e)
		}
	}
	if len(resolved) == 0 {
		return nil
	}

	result := make([]*MemoryEntry, 0, nMatches)
	for i := len(resolved) - 1; i >= 0 && len(result) < nMatches; i-- {
		e := resolved[i]
		recommendation := e.Reflection
		if recommendation == "" {
			recommendation = e.Decision
		}
		result = append(result, &MemoryEntry{
			Recommendation: recommendation,
			Decision:       e.Decision,
			Ticker:         e.Ticker,
			Date:           e.Date,
			Rating:         e.Rating,
		})
	}

	return result
}

// GetPastContext 返回格式化的历史上下文字符串，用于注入 agent prompt
// nSame: 同一 ticker 的最近条目数，nCross: 跨 ticker 的最近条目数
func (m *TradingMemoryLog) GetPastContext(ticker string, nSame, nCross int) string {
	if nSame <= 0 && nCross <= 0 {
		return ""
	}

	entries := m.LoadEntries()
	if len(entries) == 0 {
		return ""
	}

	// 过滤掉 pending 条目
	var resolved []*LogEntry
	for _, e := range entries {
		if !e.Pending {
			resolved = append(resolved, e)
		}
	}
	if len(resolved) == 0 {
		return ""
	}

	var same, cross []*LogEntry
	for i := len(resolved) - 1; i >= 0; i-- {
		if len(same) >= nSame && len(cross) >= nCross {
			break
		}
		e := resolved[i]
		if e.Ticker == ticker && len(same) < nSame {
			same = append(same, e)
		} else if e.Ticker != ticker && len(cross) < nCross {
			cross = append(cross, e)
		}
	}

	if len(same) == 0 && len(cross) == 0 {
		return ""
	}

	var parts []string
	if len(same) > 0 {
		parts = append(parts, fmt.Sprintf("Past analyses of %s (most recent first):", ticker))
		for _, e := range same {
			parts = append(parts, m.formatFull(e))
		}
	}
	if len(cross) > 0 {
		parts = append(parts, "Recent cross-ticker lessons:")
		for _, e := range cross {
			parts = append(parts, m.formatReflectionOnly(e))
		}
	}
	return strings.Join(parts, "\n\n")
}

// --- Update path (Phase B) ---

// UpdateWithOutcome 将第一个匹配的 pending 条目更新为 resolved
func (m *TradingMemoryLog) UpdateWithOutcome(ticker, tradeDate string, rawReturn, alphaReturn float64, holdingDays int, reflection string) {
	if m.logPath == "" {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(m.logPath)
	if err != nil {
		if !os.IsNotExist(err) {
			logger.L().Error("Failed to read memory log for update", zap.Error(err))
		}
		return
	}

	blocks := strings.Split(string(data), separator)
	pendingPrefix := fmt.Sprintf("[%s | %s |", tradeDate, ticker)
	rawPct := fmt.Sprintf("%+.1f%%", rawReturn*100)
	alphaPct := fmt.Sprintf("%+.1f%%", alphaReturn*100)

	updated := false
	var newBlocks []string
	for _, block := range blocks {
		stripped := strings.TrimSpace(block)
		if stripped == "" {
			newBlocks = append(newBlocks, block)
			continue
		}
		lines := strings.Split(stripped, "\n")
		tagLine := strings.TrimSpace(lines[0])

		if !updated && strings.HasPrefix(tagLine, pendingPrefix) && strings.HasSuffix(tagLine, "| pending]") {
			fields := splitFields(tagLine[1:len(tagLine)-1], "|")
			rating := "Hold"
			if len(fields) > 2 {
				rating = strings.TrimSpace(fields[2])
			}
			newTag := fmt.Sprintf("[%s | %s | %s | %s | %s | %dd]",
				tradeDate, ticker, rating, rawPct, alphaPct, holdingDays)
			rest := strings.Join(lines[1:], "\n")
			newBlock := fmt.Sprintf("%s\n\n%s\n\nREFLECTION:\n%s",
				newTag, strings.TrimLeft(rest, "\n"), reflection)
			newBlocks = append(newBlocks, newBlock)
			updated = true
		} else {
			newBlocks = append(newBlocks, block)
		}
	}

	if !updated {
		return
	}

	m.writeBlocks(newBlocks)
}

// BatchUpdateWithOutcomes 批量更新多个 pending 条目（单次读取 + 原子写入）
func (m *TradingMemoryLog) BatchUpdateWithOutcomes(updates []map[string]any) {
	if m.logPath == "" || len(updates) == 0 {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(m.logPath)
	if err != nil {
		if !os.IsNotExist(err) {
			logger.L().Error("Failed to read memory log for batch update", zap.Error(err))
		}
		return
	}

	blocks := strings.Split(string(data), separator)

	// 构建查找表
	type updateKey struct {
		tradeDate, ticker string
	}
	updateMap := make(map[updateKey]map[string]any)
	for _, u := range updates {
		key := updateKey{
			tradeDate: fmt.Sprintf("%v", u["trade_date"]),
			ticker:    fmt.Sprintf("%v", u["ticker"]),
		}
		updateMap[key] = u
	}

	var newBlocks []string
	for _, block := range blocks {
		stripped := strings.TrimSpace(block)
		if stripped == "" {
			newBlocks = append(newBlocks, block)
			continue
		}
		lines := strings.Split(stripped, "\n")
		tagLine := strings.TrimSpace(lines[0])

		matched := false
		for key, upd := range updateMap {
			pendingPrefix := fmt.Sprintf("[%s | %s |", key.tradeDate, key.ticker)
			if strings.HasPrefix(tagLine, pendingPrefix) && strings.HasSuffix(tagLine, "| pending]") {
				fields := splitFields(tagLine[1:len(tagLine)-1], "|")
				rating := "Hold"
				if len(fields) > 2 {
					rating = strings.TrimSpace(fields[2])
				}
				rawReturn, _ := upd["raw_return"].(float64)
				alphaReturn, _ := upd["alpha_return"].(float64)
				holdingDays, _ := upd["holding_days"].(int)
				if hd, ok := upd["holding_days"].(int64); ok {
					holdingDays = int(hd)
				}
				rawPct := fmt.Sprintf("%+.1f%%", rawReturn*100)
				alphaPct := fmt.Sprintf("%+.1f%%", alphaReturn*100)
				newTag := fmt.Sprintf("[%s | %s | %s | %s | %s | %dd]",
					key.tradeDate, key.ticker, rating, rawPct, alphaPct, holdingDays)
				rest := strings.Join(lines[1:], "\n")
				reflection := fmt.Sprintf("%v", upd["reflection"])
				newBlock := fmt.Sprintf("%s\n\n%s\n\nREFLECTION:\n%s",
					newTag, strings.TrimLeft(rest, "\n"), reflection)
				newBlocks = append(newBlocks, newBlock)
				delete(updateMap, key)
				matched = true
				break
			}
		}

		if !matched {
			newBlocks = append(newBlocks, block)
		}
	}

	m.writeBlocks(newBlocks)
}

// --- Helpers ---

func (m *TradingMemoryLog) writeBlocks(blocks []string) {
	blocks = m.applyRotation(blocks)
	newText := strings.Join(blocks, separator)
	tmpPath := m.logPath + ".tmp"
	if err := os.WriteFile(tmpPath, []byte(newText), 0644); err != nil {
		logger.L().Error("Failed to write temp memory log", zap.Error(err))
		return
	}
	if err := os.Rename(tmpPath, m.logPath); err != nil {
		logger.L().Error("Failed to replace memory log", zap.Error(err))
	}
}

// applyRotation 当 resolved 条目数超过 maxEntries 时，删除最老的条目
func (m *TradingMemoryLog) applyRotation(blocks []string) []string {
	if m.maxEntries <= 0 {
		return blocks
	}

	type decision struct {
		block      string
		isResolved bool
	}

	var decisions []decision
	for _, block := range blocks {
		stripped := strings.TrimSpace(block)
		if stripped == "" {
			decisions = append(decisions, decision{block: block, isResolved: false})
			continue
		}
		lines := strings.Split(stripped, "\n")
		tagLine := strings.TrimSpace(lines[0])
		isResolved := strings.HasPrefix(tagLine, "[") &&
			strings.HasSuffix(tagLine, "]") &&
			!strings.HasSuffix(tagLine, "| pending]")
		decisions = append(decisions, decision{block: block, isResolved: isResolved})
	}

	resolvedCount := 0
	for _, d := range decisions {
		if d.isResolved {
			resolvedCount++
		}
	}
	if resolvedCount <= m.maxEntries {
		return blocks
	}

	toDrop := resolvedCount - m.maxEntries
	var kept []string
	for _, d := range decisions {
		if d.isResolved && toDrop > 0 {
			toDrop--
			continue
		}
		kept = append(kept, d.block)
	}
	return kept
}

func (m *TradingMemoryLog) parseEntry(raw string) *LogEntry {
	lines := strings.Split(strings.TrimSpace(raw), "\n")
	if len(lines) == 0 {
		return nil
	}
	tagLine := strings.TrimSpace(lines[0])
	if !strings.HasPrefix(tagLine, "[") || !strings.HasSuffix(tagLine, "]") {
		return nil
	}
	fields := splitFields(tagLine[1:len(tagLine)-1], "|")
	if len(fields) < 4 {
		return nil
	}

	entry := &LogEntry{
		Date:    strings.TrimSpace(fields[0]),
		Ticker:  strings.TrimSpace(fields[1]),
		Rating:  strings.TrimSpace(fields[2]),
		Pending: strings.TrimSpace(fields[3]) == "pending",
	}
	if !entry.Pending {
		entry.Raw = strings.TrimSpace(fields[3])
		if len(fields) > 4 {
			entry.Alpha = strings.TrimSpace(fields[4])
		}
		if len(fields) > 5 {
			entry.Holding = strings.TrimSpace(fields[5])
		}
	}

	body := strings.Join(lines[1:], "\n")
	if idx := strings.Index(body, "DECISION:\n"); idx != -1 {
		decisionStart := idx + len("DECISION:\n")
		decisionPart := body[decisionStart:]
		if ridx := strings.Index(decisionPart, "\n\nREFLECTION:"); ridx != -1 {
			entry.Decision = strings.TrimSpace(decisionPart[:ridx])
			entry.Reflection = strings.TrimSpace(decisionPart[ridx+len("\n\nREFLECTION:"):])
		} else {
			entry.Decision = strings.TrimSpace(decisionPart)
		}
	}

	return entry
}

func (m *TradingMemoryLog) formatFull(e *LogEntry) string {
	raw := e.Raw
	if raw == "" {
		raw = "n/a"
	}
	alpha := e.Alpha
	if alpha == "" {
		alpha = "n/a"
	}
	holding := e.Holding
	if holding == "" {
		holding = "n/a"
	}
	tag := fmt.Sprintf("[%s | %s | %s | %s | %s | %s]",
		e.Date, e.Ticker, e.Rating, raw, alpha, holding)
	parts := []string{tag, fmt.Sprintf("DECISION:\n%s", e.Decision)}
	if e.Reflection != "" {
		parts = append(parts, fmt.Sprintf("REFLECTION:\n%s", e.Reflection))
	}
	return strings.Join(parts, "\n\n")
}

func (m *TradingMemoryLog) formatReflectionOnly(e *LogEntry) string {
	raw := e.Raw
	if raw == "" {
		raw = "n/a"
	}
	tag := fmt.Sprintf("[%s | %s | %s | %s]", e.Date, e.Ticker, e.Rating, raw)
	if e.Reflection != "" {
		return fmt.Sprintf("%s\n%s", tag, e.Reflection)
	}
	text := e.Decision
	if len(text) > 300 {
		text = text[:300] + "..."
	}
	return fmt.Sprintf("%s\n%s", tag, text)
}

// splitFields 按分隔符切分字段，保留空字段
func splitFields(s, sep string) []string {
	var parts []string
	for {
		idx := strings.Index(s, sep)
		if idx == -1 {
			parts = append(parts, s)
			break
		}
		parts = append(parts, s[:idx])
		s = s[idx+len(sep):]
	}
	return parts
}
