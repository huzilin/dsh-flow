// llm 包本文件：评审/产出通路 prompt 形状契约与 findings 宽口径解析。
//
// prompt 纪律（novel dsh.go 写法范本借桜，调研 C12/D6）：system 内嵌完整 JSON
// 形状契约 + 硬约束（findings 必须数组、desc 一句中文人话、禁围栏禁解释），
// 业务素材（capability skill/inline、上游产物正文）由 service 层组装传入 material——
// 本包不写死业务文本。解析侧宽口径（C3）：数组形状优先，字符串形状回退单条纯文本
// 意见不中断评审，null/空/缺失归空数组。
package llm

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ReviewFinding 单条评审意见（-L3 锚点两级映射的载体）：
// StartLine/EndLine 均 >0 且 StartLine≤EndLine → range（划词）锚；
// 否则 → document（文档级）锚。行号按被评审产物正文行号（1 起）。
type ReviewFinding struct {
	// Desc 意见内容（一句中文人话；模型塞 JSON/围栏时净化降级）
	Desc string `json:"desc"`
	// StartLine 范围起行（1 起；0=文档级）
	StartLine int64 `json:"startLine,omitempty"`
	// EndLine 范围终行（1 起；0=文档级）
	EndLine int64 `json:"endLine,omitempty"`
}

// findingsJSON 模型输出的 findings 条目宽松形状（行号字段名按契约 startLine/endLine；
// 数字或字符串数字都收——推理型模型宽口径）。
type findingsJSON struct {
	Desc      string `json:"desc"`
	StartLine any    `json:"startLine,omitempty"`
	EndLine   any    `json:"endLine,omitempty"`
}

// ProduceSystemPrompt produce 通路 system 段装配：角色身份 + 动作素材
// （capability skill/inline）+ 输出纪律（仅产物正文 Markdown，无解释无围栏包裹）。
func ProduceSystemPrompt(role string, material string) string {
	var b strings.Builder
	b.WriteString("你是团队项目工作台的 AI 执行代理，以「" + role + "」角色完成任务：依据任务目标与上游产物产出文档。")
	if material != "" {
		b.WriteString("\n执行要点：" + material)
	}
	b.WriteString("\n输出纪律：只输出产物文档正文本身（Markdown 文本）；禁止输出任何解释、前言、结语或元信息；不要用代码围栏包裹整篇文档。")
	return b.String()
}

// ReviewSystemPrompt review 通路 system 段装配：W3 定形纪律——JSON 形状契约内嵌
// + findings 数组硬约束 + desc 净化要求 + 划词定位要求（-L3 range/document 两级）。
func ReviewSystemPrompt(role string, capabilityName string, material string) string {
	var b strings.Builder
	b.WriteString("你是团队项目工作台的 AI 评审代理，以「" + role + "」角色按评审能力「" + capabilityName + "」评审被评审产物。")
	if material != "" {
		b.WriteString("\n评审要点：" + material)
	}
	b.WriteString("\n输出仅一个 JSON 对象，禁止 markdown 围栏与任何解释文字，形状必须为：" +
		`{"pass":bool,"findings":[{"desc":"…","startLine":1,"endLine":1}]}` +
		"。findings 必须是数组（无意见=[]）：每条只讲一件独立的事；每条必带 desc（一句中文人话，≤80字，直接给结论，" +
		"desc 内禁止出现 JSON、键值对、markdown 围栏或代码）；针对具体行/词的意见必须带 startLine/endLine" +
		"（按正文行号，1 起，startLine≤endLine）；只有全篇级问题才允许省略行号；禁止把整个报告或多个问题塞进单条 findings。" +
		"pass=true 表示通过（findings 通常为空数组）。")
	return b.String()
}

// LineNumbered 产物正文行号标注（评审通路 user 段用）：每行前缀 `N| `（N=1 起
// 行号），供模型按行定位 findings；被评审产物与划词锚共享同一行号口径。
func LineNumbered(content string) string {
	lines := strings.Split(content, "\n")
	numbered := make([]string, 0, len(lines))
	for i, line := range lines {
		numbered = append(numbered, fmt.Sprintf("%d| %s", i+1, line))
	}
	return strings.Join(numbered, "\n")
}

// ParseReviewFindings 评审 findings 解析（宽口径，C3 copy 纪律）：
//   - 剥围栏取首个 {..} 平衡块后按 {"pass":…,"findings":…} 解析；
//   - findings 数组形状优先（逐条映射，行级带 startLine/endLine）；
//   - 字符串形状回退为单条纯文本意见（旧契约容忍），不中断评审；
//   - null/空/缺失归空数组；desc 空条目剔除；desc 塞 JSON/围栏时净化降级；
//   - 整体非 JSON → 单条兜底意见（原文截断留档），评审不因解析失败中断。
func ParseReviewFindings(raw string) []ReviewFinding {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []ReviewFinding{}
	}
	var envelope struct {
		Findings json.RawMessage `json:"findings"`
	}
	if err := json.Unmarshal([]byte(StripFences(raw)), &envelope); err == nil {
		if fs := parseFindingsField(envelope.Findings); fs != nil {
			return fs
		}
	}
	// 整体非预期 JSON：单条兜底意见（净化截断留档），不中断评审
	if text := sanitizeDesc(raw); text != "" {
		return []ReviewFinding{{Desc: text}}
	}
	return []ReviewFinding{}
}

// parseFindingsField findings 字段宽口径：数组形状逐条映射；字符串形状回退单条；
// null/空/解析失败归空数组（返回空切片非 nil——调用方以 len 判定，nil 与空同义）。
func parseFindingsField(raw json.RawMessage) []ReviewFinding {
	if len(raw) == 0 || string(raw) == "null" {
		return []ReviewFinding{}
	}
	var items []findingsJSON
	if err := json.Unmarshal(raw, &items); err == nil {
		fs := make([]ReviewFinding, 0, len(items))
		for _, item := range items {
			desc := sanitizeDesc(item.Desc)
			if desc == "" {
				continue
			}
			fs = append(fs, ReviewFinding{Desc: desc, StartLine: looseInt(item.StartLine), EndLine: looseInt(item.EndLine)})
		}
		return fs
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil && strings.TrimSpace(text) != "" {
		return []ReviewFinding{{Desc: sanitizeDesc(text)}}
	}
	return []ReviewFinding{}
}

// looseInt 宽松整数提取：数字直取；字符串数字解析（推理型模型宽口径）；其余 0。
func looseInt(v any) int64 {
	switch n := v.(type) {
	case float64:
		return int64(n)
	case string:
		var out int64
		_, _ = fmt.Sscanf(strings.TrimSpace(n), "%d", &out)
		return out
	}
	return 0
}

// sanitizeDesc 意见文本净化（novel sanitizeFindingDesc 简化口径）：剥围栏残迹、
// 压成单行、截断 200 字符——desc 塞 JSON/围栏时降级为人话短句不原样落库。
func sanitizeDesc(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	// 围栏残迹清理：剥首行围栏（```json / ```text 等）+ 去除残余围栏标记
	if strings.HasPrefix(s, "```") {
		rest := s[3:]
		if nl := strings.IndexByte(rest, '\n'); nl >= 0 {
			s = strings.TrimSpace(rest[nl+1:])
		}
	}
	s = strings.ReplaceAll(s, "```json", "")
	s = strings.ReplaceAll(s, "```", "")
	s = strings.Join(strings.Fields(s), " ")
	return truncate(s, 200)
}
