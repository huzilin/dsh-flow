package skills

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// SkillParam 技能参数定义
type SkillParam struct {
	Name        string      `yaml:"name"`
	Type        string      `yaml:"type"`
	Description string      `yaml:"description"`
	Default     interface{} `yaml:"default,omitempty"`
	Required    bool        `yaml:"required,omitempty"`
}

// SkillParamHint 前端参数提示结构体
type SkillParamHint struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"`
	Description string      `json:"description"`
	Default     interface{} `json:"default,omitempty"`
}

// SkillHint 前端技能提示结构体
type SkillHint struct {
	ID             string           `json:"id"`
	DisplayName    string           `json:"display_name"`
	Description    string           `json:"description"`
	InvokeExample  string           `json:"invoke_example"`
	Category       string           `json:"category"`
	RiskTolerance  string           `json:"risk_tolerance"`
	RequiredParams []SkillParamHint `json:"required_params"`
	OptionalParams []SkillParamHint `json:"optional_params"`
}

// Skill 可扩展的分析技能接口
// 每新增一个分析师视角，只需实现此接口并注册即可
type Skill interface {
	Name() string
	DisplayName() string
	Description() string
	SystemPrompt() string
	FocusAreas() []string
	RiskTolerance() string
	RequiredParams() []SkillParam
	OptionalParams() []SkillParam
	SkillFile() string
	Category() string
	ValidateParams(params map[string]interface{}) error
	InvokeExample() string
}

// BaseSkill 基础技能实现，用于简化内置技能和插件式技能的实现
type BaseSkill struct {
	NameVal           string
	DisplayNameVal    string
	DescriptionVal    string
	SystemPromptVal   string
	FocusAreasVal     []string
	RiskToleranceVal  string
	RequiredParamsVal []SkillParam
	OptionalParamsVal []SkillParam
	SkillFileVal      string
	CategoryVal       string
}

func (b *BaseSkill) Name() string                 { return b.NameVal }
func (b *BaseSkill) DisplayName() string          { return b.DisplayNameVal }
func (b *BaseSkill) Description() string          { return b.DescriptionVal }
func (b *BaseSkill) SystemPrompt() string         { return b.SystemPromptVal }
func (b *BaseSkill) FocusAreas() []string         { return b.FocusAreasVal }
func (b *BaseSkill) RiskTolerance() string        { return b.RiskToleranceVal }
func (b *BaseSkill) RequiredParams() []SkillParam { return b.RequiredParamsVal }
func (b *BaseSkill) OptionalParams() []SkillParam { return b.OptionalParamsVal }
func (b *BaseSkill) SkillFile() string            { return b.SkillFileVal }
func (b *BaseSkill) Category() string             { return b.CategoryVal }

func (b *BaseSkill) ValidateParams(params map[string]interface{}) error {
	for _, p := range b.RequiredParamsVal {
		if _, exists := params[p.Name]; !exists {
			return fmt.Errorf("missing required parameter: %s", p.Name)
		}
	}
	return nil
}

func (b *BaseSkill) InvokeExample() string {
	parts := strings.Split(b.NameVal, ":")
	skillID := b.NameVal
	if len(parts) > 1 {
		skillID = parts[len(parts)-1]
	}

	var builder strings.Builder
	builder.WriteString("/")
	builder.WriteString(skillID)

	for _, p := range b.RequiredParamsVal {
		builder.WriteString(" <")
		builder.WriteString(b.getParamDisplayName(p))
		builder.WriteString(">")
	}

	for _, p := range b.OptionalParamsVal {
		builder.WriteString(" [")
		builder.WriteString(b.getParamDisplayName(p))
		if p.Default != nil {
			builder.WriteString("=")
			builder.WriteString(fmt.Sprintf("%v", p.Default))
		}
		builder.WriteString("]")
	}

	return builder.String()
}

func (b *BaseSkill) getParamDisplayName(p SkillParam) string {
	if p.Description != "" {
		return p.Description
	}
	return p.Name
}

// SkillRegistry 技能注册中心，线程安全
type SkillRegistry struct {
	mu     sync.RWMutex
	skills map[string]Skill
	order  []string
}

func NewSkillRegistry() *SkillRegistry {
	return &SkillRegistry{
		skills: make(map[string]Skill),
	}
}

func (r *SkillRegistry) Register(s Skill) {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := s.Name()
	if _, exists := r.skills[name]; !exists {
		r.order = append(r.order, name)
	}
	r.skills[name] = s
}

func (r *SkillRegistry) Get(name string) (Skill, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if s, ok := r.skills[name]; ok {
		return s, ok
	}

	if !strings.Contains(name, ":") {
		if s, ok := r.skills["master:"+name]; ok {
			return s, ok
		}
	}

	return nil, false
}

func (r *SkillRegistry) List() []Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Skill, 0, len(r.skills))
	for _, name := range r.order {
		result = append(result, r.skills[name])
	}
	return result
}

func (r *SkillRegistry) GetSkillHints() []SkillHint {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var hints []SkillHint
	for _, name := range r.order {
		s := r.skills[name]
		hints = append(hints, r.skillToHint(s))
	}
	return hints
}

func (r *SkillRegistry) GetSkillHint(name string) (*SkillHint, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	s, ok := r.skills[name]
	if !ok {
		return nil, false
	}
	hint := r.skillToHint(s)
	return &hint, true
}

func (r *SkillRegistry) skillToHint(s Skill) SkillHint {
	var requiredParams []SkillParamHint
	for _, p := range s.RequiredParams() {
		requiredParams = append(requiredParams, SkillParamHint{
			Name:        p.Name,
			Type:        p.Type,
			Description: p.Description,
			Default:     p.Default,
		})
	}

	var optionalParams []SkillParamHint
	for _, p := range s.OptionalParams() {
		optionalParams = append(optionalParams, SkillParamHint{
			Name:        p.Name,
			Type:        p.Type,
			Description: p.Description,
			Default:     p.Default,
		})
	}

	return SkillHint{
		ID:             s.Name(),
		DisplayName:    s.DisplayName(),
		Description:    s.Description(),
		InvokeExample:  s.InvokeExample(),
		Category:       s.Category(),
		RiskTolerance:  s.RiskTolerance(),
		RequiredParams: requiredParams,
		OptionalParams: optionalParams,
	}
}

func (r *SkillRegistry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, len(r.order))
	copy(names, r.order)
	return names
}

func BuildSkillPrompt(skill Skill, analysisContext string) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("你现在以 **%s** 的分析视角回答问题。\n", skill.DisplayName()))
	b.WriteString(fmt.Sprintf("风格特征：%s\n", skill.Description()))
	b.WriteString(fmt.Sprintf("风险偏好：%s\n\n", skill.RiskTolerance()))

	b.WriteString(skill.SystemPrompt())
	b.WriteString("\n\n---\n\n")

	b.WriteString(`规则：
1. 回答必须基于提供的分析报告内容，不得编造数据
2. 引用来源必须标注具体报告名称
3. 涉及具体操作建议时，必须附加投资风险提示
4. 不确定的信息明确说明"根据现有数据无法确定"
5. 使用中文回答

`)

	if len(analysisContext) > 0 {
		b.WriteString("分析报告内容：\n")
		b.WriteString(analysisContext)
	}

	return b.String()
}

func (r *SkillRegistry) SearchSkills(keyword string) []Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if keyword == "" {
		return r.List()
	}

	var result []Skill
	kw := strings.ToLower(keyword)
	for _, name := range r.order {
		s := r.skills[name]
		if strings.Contains(strings.ToLower(s.Name()), kw) ||
			strings.Contains(strings.ToLower(s.DisplayName()), kw) ||
			strings.Contains(strings.ToLower(s.Description()), kw) {
			result = append(result, s)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Name() < result[j].Name()
	})

	return result
}
