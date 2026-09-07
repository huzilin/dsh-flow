package skills

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSkillRegistry(t *testing.T) {
	reg := NewSkillRegistry()
	if reg == nil {
		t.Fatal("expected non-nil registry")
	}

	if len(reg.List()) != 0 {
		t.Errorf("expected empty registry, got %d skills", len(reg.List()))
	}
}

func TestSkillRegisterAndList(t *testing.T) {
	reg := NewSkillRegistry()
	loader := NewPluginLoaderWithPaths(reg, "./plugin", "./registry")
	loader.LoadPlugins()

	all := reg.List()
	if len(all) != 20 {
		t.Errorf("expected 20 builtin skills, got %d skills", len(all))
	}

	names := reg.Names()
	if len(names) != 20 {
		t.Errorf("expected 20 names, got %d", len(names))
	}
}

func TestSkillGet(t *testing.T) {
	reg := NewSkillRegistry()
	loader := NewPluginLoaderWithPaths(reg, "./plugin", "./registry")
	loader.LoadPlugins()

	s, ok := reg.Get("master:buffett")
	if ok {
		t.Errorf("expected master:buffett skill not to exist, got %s", s.Name())
	}

	s, ok = reg.Get("buffett")
	if ok {
		t.Errorf("expected buffett alias not to resolve, got %s", s.Name())
	}

	reg.Register(&BaseSkill{
		NameVal:        "test:skill",
		DisplayNameVal: "Test Skill",
	})

	s, ok = reg.Get("test:skill")
	if !ok {
		t.Fatal("expected test:skill to exist")
	}
	if s.Name() != "test:skill" {
		t.Errorf("expected test:skill, got %s", s.Name())
	}
}

func TestSkillInterface(t *testing.T) {
	skill := &BaseSkill{
		NameVal:          "test:skill",
		DisplayNameVal:   "Test Skill",
		DescriptionVal:   "Test Description",
		SystemPromptVal:  "Test System Prompt",
		FocusAreasVal:    []string{"area1", "area2"},
		RiskToleranceVal: "moderate",
		CategoryVal:      "analyst",
	}

	if skill.Name() == "" {
		t.Error("expected non-empty name")
	}
	if skill.DisplayName() == "" {
		t.Error("expected non-empty display name")
	}
	if skill.Description() == "" {
		t.Error("expected non-empty description")
	}
	if skill.SystemPrompt() == "" {
		t.Error("expected non-empty system prompt")
	}
	if len(skill.FocusAreas()) == 0 {
		t.Error("expected non-empty focus areas")
	}
	if skill.RiskTolerance() == "" {
		t.Error("expected non-empty risk tolerance")
	}
	if skill.Category() == "" {
		t.Error("expected non-empty category")
	}
}

func TestBuildSkillPrompt(t *testing.T) {
	skill := &BaseSkill{
		NameVal:          "test:skill",
		DisplayNameVal:   "Test Skill",
		DescriptionVal:   "Test Description",
		SystemPromptVal:  "Test System Prompt",
		RiskToleranceVal: "moderate",
	}
	prompt := BuildSkillPrompt(skill, "test context")

	if len(prompt) == 0 {
		t.Error("expected non-empty prompt")
	}
	if !contains(prompt, skill.DisplayName()) {
		t.Error("prompt should contain display name")
	}
	if !contains(prompt, "test context") {
		t.Error("prompt should contain analysis context")
	}
}

func TestSkillSearch(t *testing.T) {
	reg := NewSkillRegistry()
	loader := NewPluginLoaderWithPaths(reg, "./plugin", "./registry")
	loader.LoadPlugins()

	reg.Register(&BaseSkill{
		NameVal:        "test:skill1",
		DisplayNameVal: "价值投资技能",
		DescriptionVal: "价值投资相关的技能",
	})
	reg.Register(&BaseSkill{
		NameVal:        "test:skill2",
		DisplayNameVal: "成长投资技能",
		DescriptionVal: "成长投资相关的技能",
	})

	results := reg.SearchSkills("价值")
	if len(results) != 1 {
		t.Errorf("expected 1 result for '价值', got %d", len(results))
	}

	results = reg.SearchSkills("nonexistent")
	if len(results) != 0 {
		t.Error("expected no results for nonexistent keyword")
	}

	results = reg.SearchSkills("")
	if len(results) != 22 {
		t.Errorf("empty keyword should return all (20 builtin + 2 test), got %d", len(results))
	}
}

func TestSkillRiskTolerances(t *testing.T) {
	reg := NewSkillRegistry()
	loader := NewPluginLoaderWithPaths(reg, "./plugin", "./registry")
	loader.LoadPlugins()

	reg.Register(&BaseSkill{
		NameVal:          "test:conservative",
		RiskToleranceVal: "conservative",
	})
	reg.Register(&BaseSkill{
		NameVal:          "test:aggressive",
		RiskToleranceVal: "aggressive",
	})
	reg.Register(&BaseSkill{
		NameVal:          "test:moderate",
		RiskToleranceVal: "moderate",
	})

	tolerances := map[string]string{
		"test:conservative": "conservative",
		"test:aggressive":   "aggressive",
		"test:moderate":     "moderate",
	}

	for name, expected := range tolerances {
		s, ok := reg.Get(name)
		if !ok {
			t.Errorf("skill not found: %s", name)
			continue
		}
		if s.RiskTolerance() != expected {
			t.Errorf("skill %s: expected risk %s, got %s", name, expected, s.RiskTolerance())
		}
	}
}

func TestSkillCategory(t *testing.T) {
	reg := NewSkillRegistry()
	loader := NewPluginLoaderWithPaths(reg, "./plugin", "./registry")
	loader.LoadPlugins()

	reg.Register(&BaseSkill{
		NameVal:     "test:analyst",
		CategoryVal: "analyst",
	})
	reg.Register(&BaseSkill{
		NameVal:     "test:strategy",
		CategoryVal: "strategy",
	})

	categories := map[string]string{
		"test:analyst":  "analyst",
		"test:strategy": "strategy",
	}

	for name, expected := range categories {
		s, ok := reg.Get(name)
		if !ok {
			t.Errorf("skill not found: %s", name)
			continue
		}
		if s.Category() != expected {
			t.Errorf("skill %s: expected category %s, got %s", name, expected, s.Category())
		}
	}
}

func TestSkillParams(t *testing.T) {
	skill := &BaseSkill{
		NameVal: "test:skill",
		RequiredParamsVal: []SkillParam{
			{Name: "symbol", Type: "string", Required: true},
		},
		OptionalParamsVal: []SkillParam{
			{Name: "depth", Type: "string", Default: "normal"},
		},
	}

	err := skill.ValidateParams(map[string]interface{}{"symbol": "AAPL"})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	err = skill.ValidateParams(map[string]interface{}{})
	if err == nil {
		t.Error("expected error for missing required param")
	}
}

func TestMarkdownParser_Parse(t *testing.T) {
	parser := NewMarkdownParser()

	content := `# 测试技能

## 描述
这是一个测试技能

## 系统提示词
你是一个测试分析师。

## 流程
1. 第一步：分析
2. 第二步：总结
`

	parsed := parser.Parse(content)

	if parsed.Title != "测试技能" {
		t.Errorf("expected title '测试技能', got '%s'", parsed.Title)
	}
	if parsed.SystemPrompt == "" {
		t.Error("expected non-empty system prompt")
	}
	if len(parsed.Workflow) != 2 {
		t.Errorf("expected 2 workflow steps, got %d", len(parsed.Workflow))
	}
}

func TestMarkdownParser_GetSection(t *testing.T) {
	parser := NewMarkdownParser()

	content := `# 标题

## 描述
内容描述

## 分析框架
框架内容
`

	parsed := parser.Parse(content)

	desc := parser.GetSection(parsed, "描述")
	if desc != "内容描述" {
		t.Errorf("expected '内容描述', got '%s'", desc)
	}

	nonexistent := parser.GetSection(parsed, "不存在")
	if nonexistent != "" {
		t.Errorf("expected empty for nonexistent section, got '%s'", nonexistent)
	}
}

func TestMarkdownParser_HasSection(t *testing.T) {
	parser := NewMarkdownParser()

	content := `# 标题

## 描述
内容
`

	parsed := parser.Parse(content)

	if !parser.HasSection(parsed, "描述") {
		t.Error("expected section '描述' to exist")
	}
	if parser.HasSection(parsed, "不存在") {
		t.Error("expected section '不存在' to not exist")
	}
}

func TestMarkdownParser_ExtractParameters(t *testing.T) {
	parser := NewMarkdownParser()

	content := `- symbol: 股票代码 (required)
- trade_date: 交易日期
- depth: 分析深度 (optional)`

	params := parser.ExtractParameters(content)

	if len(params) != 3 {
		t.Errorf("expected 3 parameters, got %d", len(params))
	}

	if params[0].Name != "symbol" {
		t.Errorf("expected first param 'symbol', got '%s'", params[0].Name)
	}
	if !params[0].Required {
		t.Error("expected symbol to be required")
	}
}

func TestPluginLoader_EnsureDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	pluginsDir := filepath.Join(tmpDir, "plugins")
	registryDir := filepath.Join(tmpDir, "registry")

	loader := NewPluginLoaderWithPaths(NewSkillRegistry(), pluginsDir, registryDir)

	err := loader.ensureDirectories()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if _, err := os.Stat(pluginsDir); os.IsNotExist(err) {
		t.Error("plugins dir should exist")
	}
	if _, err := os.Stat(registryDir); os.IsNotExist(err) {
		t.Error("registry dir should exist")
	}
}

func TestPluginLoader_ValidateSkillID(t *testing.T) {
	testCases := []struct {
		name  string
		id    string
		valid bool
	}{
		{"valid with namespace", "master:buffett", true},
		{"valid plugin", "ai-berkshire:investment-research", true},
		{"invalid no colon", "buffett", false},
		{"invalid empty namespace", ":buffett", false},
		{"invalid empty name", "master:", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateSkillID(tc.id)
			if tc.valid && err != nil {
				t.Errorf("expected valid, got error: %v", err)
			}
			if !tc.valid && err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestHealthChecker_Check(t *testing.T) {
	registry := NewSkillRegistry()
	registry.Register(&BaseSkill{
		NameVal:          "test:skill1",
		DisplayNameVal:   "测试技能1",
		DescriptionVal:   "测试",
		SystemPromptVal:  "prompt",
		FocusAreasVal:    []string{"test"},
		RiskToleranceVal: "moderate",
		CategoryVal:      "analyst",
	})

	checker := NewHealthChecker(registry)
	results := checker.Check()

	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != StatusWarning {
		t.Errorf("expected warning status (no params), got %s", results[0].Status)
	}
}

func TestHealthChecker_IsAllHealthy(t *testing.T) {
	registry := NewSkillRegistry()
	registry.Register(&BaseSkill{
		NameVal:          "test:skill1",
		DisplayNameVal:   "测试技能",
		DescriptionVal:   "测试",
		SystemPromptVal:  "prompt",
		FocusAreasVal:    []string{"test"},
		RiskToleranceVal: "moderate",
	})

	checker := NewHealthChecker(registry)

	if !checker.IsAllHealthy() {
		t.Error("expected all healthy (no critical errors)")
	}
}

func TestHealthChecker_GetCriticalSkills(t *testing.T) {
	registry := NewSkillRegistry()
	registry.Register(&BaseSkill{
		NameVal:        "test:skill1",
		DisplayNameVal: "测试技能",
		SkillFileVal:   "/non/existent/path.md",
	})

	checker := NewHealthChecker(registry)
	critical := checker.GetCriticalSkills()

	if len(critical) != 1 {
		t.Errorf("expected 1 critical skill, got %d", len(critical))
	}
}

func TestBaseSkill_Getters(t *testing.T) {
	skill := &BaseSkill{
		NameVal:          "test:skill",
		DisplayNameVal:   "测试技能",
		DescriptionVal:   "description",
		SystemPromptVal:  "prompt",
		FocusAreasVal:    []string{"area1", "area2"},
		RiskToleranceVal: "conservative",
		CategoryVal:      "analyst",
	}

	if skill.Name() != "test:skill" {
		t.Errorf("expected name 'test:skill', got '%s'", skill.Name())
	}
	if skill.DisplayName() != "测试技能" {
		t.Errorf("expected display name '测试技能', got '%s'", skill.DisplayName())
	}
	if skill.Category() != "analyst" {
		t.Errorf("expected category 'analyst', got '%s'", skill.Category())
	}
}

func TestBaseSkill_InvokeExample(t *testing.T) {
	tests := []struct {
		name     string
		skill    *BaseSkill
		expected string
	}{
		{
			name: "no params",
			skill: &BaseSkill{
				NameVal:        "test:skill",
				DisplayNameVal: "Test Skill",
			},
			expected: "/skill",
		},
		{
			name: "with required params using description",
			skill: &BaseSkill{
				NameVal:        "test:skill",
				DisplayNameVal: "Test Skill",
				RequiredParamsVal: []SkillParam{
					{Name: "symbol", Type: "string", Description: "股票代码"},
					{Name: "date", Type: "string", Description: "交易日期"},
				},
			},
			expected: "/skill <股票代码> <交易日期>",
		},
		{
			name: "with optional params using description and default",
			skill: &BaseSkill{
				NameVal:        "test:skill",
				DisplayNameVal: "Test Skill",
				OptionalParamsVal: []SkillParam{
					{Name: "limit", Type: "int", Description: "筛选数量", Default: 10},
				},
			},
			expected: "/skill [筛选数量=10]",
		},
		{
			name: "with both required and optional",
			skill: &BaseSkill{
				NameVal:        "test:skill",
				DisplayNameVal: "Test Skill",
				RequiredParamsVal: []SkillParam{
					{Name: "symbol", Type: "string", Description: "股票代码"},
				},
				OptionalParamsVal: []SkillParam{
					{Name: "depth", Type: "string", Description: "分析深度"},
					{Name: "limit", Type: "int", Description: "筛选数量", Default: 5},
				},
			},
			expected: "/skill <股票代码> [分析深度] [筛选数量=5]",
		},
		{
			name: "simple name without colon",
			skill: &BaseSkill{
				NameVal:           "simple-skill",
				DisplayNameVal:    "Simple Skill",
				RequiredParamsVal: []SkillParam{{Name: "input", Description: "输入内容"}},
			},
			expected: "/simple-skill <输入内容>",
		},
		{
			name: "fallback to name when description is empty",
			skill: &BaseSkill{
				NameVal:        "test:skill",
				DisplayNameVal: "Test Skill",
				RequiredParamsVal: []SkillParam{
					{Name: "symbol", Type: "string"},
				},
			},
			expected: "/skill <symbol>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.skill.InvokeExample()
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestSkillRegistry_Empty(t *testing.T) {
	reg := NewSkillRegistry()

	if len(reg.List()) != 0 {
		t.Errorf("expected empty registry, got %d", len(reg.List()))
	}
	if len(reg.Names()) != 0 {
		t.Errorf("expected empty names, got %d", len(reg.Names()))
	}
	if _, ok := reg.Get("nonexistent"); ok {
		t.Error("expected no skill for nonexistent")
	}
}

func TestSkillRegistry_MultipleRegister(t *testing.T) {
	reg := NewSkillRegistry()

	skill1 := &BaseSkill{NameVal: "test:skill1", DisplayNameVal: "Skill 1"}
	skill2 := &BaseSkill{NameVal: "test:skill2", DisplayNameVal: "Skill 2"}

	reg.Register(skill1)
	reg.Register(skill2)
	reg.Register(skill1)

	if len(reg.List()) != 2 {
		t.Errorf("expected 2 unique skills, got %d", len(reg.List()))
	}
}

func TestSkillRegistry_GetSkillHints(t *testing.T) {
	reg := NewSkillRegistry()

	skill1 := &BaseSkill{
		NameVal:          "test:skill1",
		DisplayNameVal:   "行业分析",
		DescriptionVal:   "行业深度分析",
		CategoryVal:      "analyst",
		RiskToleranceVal: "moderate",
		RequiredParamsVal: []SkillParam{
			{Name: "sector", Type: "string", Description: "行业板块"},
		},
		OptionalParamsVal: []SkillParam{
			{Name: "limit", Type: "int", Description: "筛选数量", Default: 10},
		},
	}
	reg.Register(skill1)

	hints := reg.GetSkillHints()
	if len(hints) != 1 {
		t.Errorf("expected 1 hint, got %d", len(hints))
	}

	hint := hints[0]
	if hint.ID != "test:skill1" {
		t.Errorf("expected ID 'test:skill1', got '%s'", hint.ID)
	}
	if hint.DisplayName != "行业分析" {
		t.Errorf("expected DisplayName '行业分析', got '%s'", hint.DisplayName)
	}
	if hint.InvokeExample != "/skill1 <行业板块> [筛选数量=10]" {
		t.Errorf("expected InvokeExample '/skill1 <行业板块> [筛选数量=10]', got '%s'", hint.InvokeExample)
	}
	if len(hint.RequiredParams) != 1 {
		t.Errorf("expected 1 required param, got %d", len(hint.RequiredParams))
	}
	if hint.RequiredParams[0].Description != "行业板块" {
		t.Errorf("expected required param description '行业板块', got '%s'", hint.RequiredParams[0].Description)
	}
	if len(hint.OptionalParams) != 1 {
		t.Errorf("expected 1 optional param, got %d", len(hint.OptionalParams))
	}
	if hint.OptionalParams[0].Description != "筛选数量" {
		t.Errorf("expected optional param description '筛选数量', got '%s'", hint.OptionalParams[0].Description)
	}
	if hint.OptionalParams[0].Default != 10 {
		t.Errorf("expected optional param default 10, got %v", hint.OptionalParams[0].Default)
	}
}

func TestSkillRegistry_GetSkillHint(t *testing.T) {
	reg := NewSkillRegistry()

	skill := &BaseSkill{
		NameVal:          "test:skill",
		DisplayNameVal:   "测试技能",
		DescriptionVal:   "测试描述",
		CategoryVal:      "test",
		RiskToleranceVal: "low",
	}
	reg.Register(skill)

	hint, ok := reg.GetSkillHint("test:skill")
	if !ok {
		t.Error("expected skill hint to exist")
	}
	if hint.ID != "test:skill" {
		t.Errorf("expected ID 'test:skill', got '%s'", hint.ID)
	}
	if hint.DisplayName != "测试技能" {
		t.Errorf("expected DisplayName '测试技能', got '%s'", hint.DisplayName)
	}

	_, ok = reg.GetSkillHint("nonexistent")
	if ok {
		t.Error("expected no skill hint for nonexistent")
	}
}

func TestPluginLoader_ListPlugins(t *testing.T) {
	tmpDir := t.TempDir()
	pluginsDir := filepath.Join(tmpDir, "plugins")
	registryDir := filepath.Join(tmpDir, "registry")

	err := os.MkdirAll(registryDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	configFile := filepath.Join(registryDir, "test-plugin.yaml")
	err = os.WriteFile(configFile, []byte(`name: test-plugin
version: "1.0.0"
repo_url: https://example.com/test.git
skills: []
`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	loader := NewPluginLoaderWithPaths(NewSkillRegistry(), pluginsDir, registryDir)
	plugins, err := loader.ListPlugins()
	if err != nil {
		t.Fatal(err)
	}

	if len(plugins) != 1 {
		t.Errorf("expected 1 plugin, got %d", len(plugins))
	}
	if plugins[0] != "test-plugin" {
		t.Errorf("expected test-plugin, got %s", plugins[0])
	}
}

func TestPluginLoader_GetPluginInfo(t *testing.T) {
	tmpDir := t.TempDir()
	pluginsDir := filepath.Join(tmpDir, "plugins")
	registryDir := filepath.Join(tmpDir, "registry")

	err := os.MkdirAll(registryDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	configFile := filepath.Join(registryDir, "test-plugin.yaml")
	err = os.WriteFile(configFile, []byte(`name: test-plugin
version: "1.0.0"
repo_url: https://example.com/test.git
branch: main
skills: []
`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	loader := NewPluginLoaderWithPaths(NewSkillRegistry(), pluginsDir, registryDir)
	info, err := loader.GetPluginInfo("test-plugin")
	if err != nil {
		t.Fatal(err)
	}

	if info.Name != "test-plugin" {
		t.Errorf("expected name test-plugin, got %s", info.Name)
	}
	if info.Version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", info.Version)
	}
	if info.Branch != "main" {
		t.Errorf("expected branch main, got %s", info.Branch)
	}
}

func TestPluginLoader_GetSkillDetail(t *testing.T) {
	reg := NewSkillRegistry()
	skill := &BaseSkill{
		NameVal:          "test:skill",
		DisplayNameVal:   "Test Skill",
		DescriptionVal:   "Description",
		CategoryVal:      "analyst",
		RiskToleranceVal: "moderate",
		FocusAreasVal:    []string{"area1"},
	}
	reg.Register(skill)

	loader := NewPluginLoaderWithPaths(reg, "/tmp/plugins", "/tmp/registry")
	detail, err := loader.GetSkillDetail("test:skill")
	if err != nil {
		t.Fatal(err)
	}

	if detail == "" {
		t.Error("expected non-empty detail")
	}
	if !contains(detail, "Test Skill") {
		t.Error("detail should contain display name")
	}
	if !contains(detail, "analyst") {
		t.Error("detail should contain category")
	}
}

func TestHealthChecker_GetWarningSkills(t *testing.T) {
	registry := NewSkillRegistry()
	registry.Register(&BaseSkill{
		NameVal:          "test:skill1",
		DisplayNameVal:   "测试技能",
		DescriptionVal:   "测试",
		SystemPromptVal:  "prompt",
		FocusAreasVal:    []string{"test"},
		RiskToleranceVal: "moderate",
	})

	checker := NewHealthChecker(registry)
	warnings := checker.GetWarningSkills()

	if len(warnings) != 1 {
		t.Errorf("expected 1 warning skill (no params), got %d", len(warnings))
	}
}

func TestHealthChecker_MixedStatus(t *testing.T) {
	registry := NewSkillRegistry()
	registry.Register(&BaseSkill{
		NameVal:        "test:critical",
		DisplayNameVal: "Critical Skill",
		SkillFileVal:   "/non/existent/file.md",
	})
	registry.Register(&BaseSkill{
		NameVal:          "test:warning",
		DisplayNameVal:   "Warning Skill",
		DescriptionVal:   "test",
		SystemPromptVal:  "prompt",
		FocusAreasVal:    []string{"test"},
		RiskToleranceVal: "moderate",
	})
	registry.Register(&BaseSkill{
		NameVal:           "test:healthy",
		DisplayNameVal:    "Healthy Skill",
		DescriptionVal:    "test",
		SystemPromptVal:   "prompt",
		FocusAreasVal:     []string{"test"},
		RiskToleranceVal:  "moderate",
		RequiredParamsVal: []SkillParam{{Name: "symbol", Type: "string"}},
	})

	checker := NewHealthChecker(registry)
	results := checker.Check()

	criticalCount := 0
	warningCount := 0
	healthyCount := 0
	for _, r := range results {
		switch r.Status {
		case StatusCritical:
			criticalCount++
		case StatusWarning:
			warningCount++
		case StatusHealthy:
			healthyCount++
		}
	}

	if criticalCount != 1 {
		t.Errorf("expected 1 critical, got %d", criticalCount)
	}
	if warningCount != 1 {
		t.Errorf("expected 1 warning, got %d", warningCount)
	}
	if healthyCount != 1 {
		t.Errorf("expected 1 healthy, got %d", healthyCount)
	}
}

func TestIntegration_LoadAiBerkshirePlugin(t *testing.T) {
	_, err := os.Stat("./plugin/ai-berkshire")
	if os.IsNotExist(err) {
		t.Skip("ai-berkshire plugin not installed, skipping integration test")
	}

	registry := NewSkillRegistry()
	loader := NewPluginLoaderWithPaths(registry, "./plugin", "./registry")

	err = loader.LoadPlugins()
	if err != nil {
		t.Fatalf("Failed to load plugins: %v", err)
	}

	skills := registry.List()
	if len(skills) == 0 {
		t.Error("Expected at least one skill to be loaded")
	}

	foundInvestmentResearch := false
	foundDypAsk := false
	for _, s := range skills {
		if s.Name() == "ai-berkshire:investment-research" {
			foundInvestmentResearch = true
		}
		if s.Name() == "ai-berkshire:dyp-ask" {
			foundDypAsk = true
		}
	}

	if !foundInvestmentResearch {
		t.Error("Expected ai-berkshire:investment-research to be loaded")
	}
	if !foundDypAsk {
		t.Error("Expected ai-berkshire:dyp-ask to be loaded")
	}

	testSkill, ok := registry.Get("ai-berkshire:investment-research")
	if !ok {
		t.Fatal("Expected ai-berkshire:investment-research to exist")
	}

	if testSkill.DisplayName() != "深度投资研究" {
		t.Errorf("Expected display name '深度投资研究', got '%s'", testSkill.DisplayName())
	}
	if testSkill.Category() != "analyst" {
		t.Errorf("Expected category 'analyst', got '%s'", testSkill.Category())
	}
	if len(testSkill.RequiredParams()) != 2 {
		t.Errorf("Expected 2 required params, got %d", len(testSkill.RequiredParams()))
	}
	if testSkill.SkillFile() == "" {
		t.Error("Expected skill file path to be set")
	}
}

func TestIntegration_SkillSearchWithPlugin(t *testing.T) {
	_, err := os.Stat("./plugin/ai-berkshire")
	if os.IsNotExist(err) {
		t.Skip("ai-berkshire plugin not installed, skipping integration test")
	}

	registry := NewSkillRegistry()
	loader := NewPluginLoaderWithPaths(registry, "./plugin", "./registry")

	err = loader.LoadPlugins()
	if err != nil {
		t.Fatalf("Failed to load plugins: %v", err)
	}

	results := registry.SearchSkills("投资")
	if len(results) == 0 {
		t.Error("Expected results for '投资'")
	}

	results = registry.SearchSkills("深度")
	if len(results) == 0 {
		t.Error("Expected results for '深度'")
	}

	results = registry.SearchSkills("nonexistentxyz")
	if len(results) != 0 {
		t.Errorf("Expected 0 results for 'nonexistentxyz', got %d", len(results))
	}
}

func TestIntegration_HealthCheckWithPlugin(t *testing.T) {
	_, err := os.Stat("./plugin/ai-berkshire")
	if os.IsNotExist(err) {
		t.Skip("ai-berkshire plugin not installed, skipping integration test")
	}

	registry := NewSkillRegistry()
	loader := NewPluginLoaderWithPaths(registry, "./plugin", "./registry")

	err = loader.LoadPlugins()
	if err != nil {
		t.Fatalf("Failed to load plugins: %v", err)
	}

	checker := NewHealthChecker(registry)
	results := checker.Check()

	if len(results) == 0 {
		t.Error("Expected health check results")
	}

	criticalCount := 0
	for _, r := range results {
		if r.Status == StatusCritical {
			criticalCount++
		}
	}

	if criticalCount > 0 {
		t.Errorf("Expected 0 critical skills, got %d", criticalCount)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
