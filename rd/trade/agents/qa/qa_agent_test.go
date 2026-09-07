package qa

import (
	"testing"

	"github.com/huzilin/trade/skills"
)

func TestSessionManagement(t *testing.T) {
	session := NewQASession("AAPL", "2024-01-15")

	if session.Ticker != "AAPL" {
		t.Errorf("expected AAPL, got %s", session.Ticker)
	}

	if session.TradeDate != "2024-01-15" {
		t.Errorf("expected 2024-01-15, got %s", session.TradeDate)
	}

	if len(session.History) != 0 {
		t.Errorf("expected empty history, got %d", len(session.History))
	}

	// Test adding messages
	session.AddMessage("user", "test question")
	session.AddMessage("assistant", "test answer")

	if len(session.History) != 2 {
		t.Errorf("expected 2 messages, got %d", len(session.History))
	}

	// Test history limit (keep last 20 = 10 rounds)
	for i := 0; i < 25; i++ {
		session.AddMessage("user", "msg")
	}
	if len(session.History) > 20 {
		t.Errorf("history should be limited to 20, got %d", len(session.History))
	}
}

func TestAnalysisContext(t *testing.T) {
	session := NewQASession("AAPL", "2024-01-15")
	session.SetAnalysisContext(
		"market report content",
		"sentiment report",
		"news report",
		"fundamentals report",
		"investment plan",
		"final decision",
	)

	ctx := session.BuildContextForLLM()

	if ctx == "" {
		t.Error("expected non-empty context")
	}

	if !contains(ctx, "market report content") {
		t.Error("context should contain market report")
	}
}

func TestConversationHistory(t *testing.T) {
	session := NewQASession("AAPL", "2024-01-15")
	session.AddMessage("user", "question")
	session.AddMessage("assistant", "answer")

	history := session.BuildConversationHistory()
	if !contains(history, "user: question") {
		t.Errorf("history should contain user message, got: %s", history)
	}
	if !contains(history, "assistant: answer") {
		t.Errorf("history should contain assistant message, got: %s", history)
	}
}

func TestIntentKeywordMatch(t *testing.T) {
	classifier := NewIntentClassifier(nil, "")

	tests := []struct {
		question string
		expected IntentType
	}{
		{"这个股票的市盈率是多少？", IntentDataQuery},
		{"有什么风险？", IntentRiskInquiry},
		{"和同行业相比如何？", IntentComparison},
		{"建议怎么操作？", IntentStrategyAdvice},
		{"当前市场趋势如何？", IntentMarketInsight},
	}

	for _, tt := range tests {
		result, err := classifier.Classify(tt.question, "")
		if err != nil {
			t.Errorf("classify error: %v", err)
		}
		if result.Intent != tt.expected {
			t.Errorf("for '%s': expected %s, got %s", tt.question, tt.expected, result.Intent)
		}
	}
}

func TestIntentDefaultFallback(t *testing.T) {
	classifier := NewIntentClassifier(nil, "")

	result, err := classifier.Classify("你好，今天天气怎么样？", "")
	if err != nil {
		t.Errorf("classify error: %v", err)
	}
	if result.Intent != IntentGeneral {
		t.Errorf("expected general intent, got %s", result.Intent)
	}
}

func TestRetriever(t *testing.T) {
	r := NewRetriever()

	ctx := AnalysisContext{
		MarketReport:       "股票处于上升趋势，MACD金叉，RSI为65。",
		SentimentReport:    "社交媒体情绪积极，讨论热度上升。",
		FundamentalsReport: "市盈率15倍，净利润增长20%。",
		FinalDecision:      "建议买入，目标价200。",
	}

	r.IndexAnalysisContext(ctx)

	// Test retrieval
	docs := r.Retrieve("市盈率 PE 估值", 3)
	if len(docs) == 0 {
		t.Error("expected non-empty results")
	}

	// Test context building
	contextText := BuildContextText(docs)
	if contextText == "" {
		t.Error("expected non-empty context text")
	}

	// Test retrieval with no match
	emptyDocs := r.Retrieve("xyzabc notfound", 3)
	if len(emptyDocs) != 0 {
		t.Errorf("expected empty results, got %d", len(emptyDocs))
	}
}

func TestRetrieverEmpty(t *testing.T) {
	r := NewRetriever()
	docs := r.Retrieve("test", 3)
	if len(docs) != 0 {
		t.Errorf("expected empty results, got %d", len(docs))
	}

	text := BuildContextText(nil)
	if text != "" {
		t.Errorf("expected empty text, got %s", text)
	}
}

func TestNewQAAgent(t *testing.T) {
	agent := NewQAAgent(nil, &QAConfig{Model: "test-model"})

	if agent.GetName() != "QA" {
		t.Errorf("expected QA, got %s", agent.GetName())
	}

	if agent.model != "test-model" {
		t.Errorf("expected test-model, got %s", agent.model)
	}

	if agent.skillRegistry == nil {
		t.Error("expected non-nil skill registry")
	}
}

func TestQAAgent_SkillRegistry(t *testing.T) {
	agent := NewQAAgent(nil, &QAConfig{Model: "test-model"})

	registry := agent.SkillRegistry()
	if registry == nil {
		t.Error("expected non-nil skill registry")
	}

	if len(registry.List()) != 0 {
		t.Errorf("expected empty registry, got %d skills", len(registry.List()))
	}
}

func TestQAAgent_RegisterSkill(t *testing.T) {
	agent := NewQAAgent(nil, &QAConfig{Model: "test-model"})

	skill := &skills.BaseSkill{
		NameVal:          "test:skill",
		DisplayNameVal:   "Test Skill",
		DescriptionVal:   "Test Description",
		SystemPromptVal:  "Test Prompt",
		FocusAreasVal:    []string{"test"},
		RiskToleranceVal: "moderate",
		CategoryVal:      "analyst",
	}

	agent.RegisterSkill(skill)

	registry := agent.SkillRegistry()
	if len(registry.List()) != 1 {
		t.Errorf("expected 1 skill, got %d", len(registry.List()))
	}

	found, ok := registry.Get("test:skill")
	if !ok {
		t.Error("expected test:skill to be registered")
	}
	if found.Name() != "test:skill" {
		t.Errorf("expected name 'test:skill', got '%s'", found.Name())
	}
}

func TestQAAgent_ListSkills(t *testing.T) {
	agent := NewQAAgent(nil, &QAConfig{Model: "test-model"})

	skill1 := &skills.BaseSkill{
		NameVal:          "test:skill1",
		DisplayNameVal:   "Skill 1",
		DescriptionVal:   "Description 1",
		SystemPromptVal:  "Prompt 1",
		FocusAreasVal:    []string{"area1"},
		RiskToleranceVal: "conservative",
		CategoryVal:      "analyst",
	}

	skill2 := &skills.BaseSkill{
		NameVal:          "test:skill2",
		DisplayNameVal:   "Skill 2",
		DescriptionVal:   "Description 2",
		SystemPromptVal:  "Prompt 2",
		FocusAreasVal:    []string{"area2"},
		RiskToleranceVal: "aggressive",
		CategoryVal:      "strategy",
	}

	agent.RegisterSkill(skill1)
	agent.RegisterSkill(skill2)

	result := agent.listSkills()
	outputs := result.Outputs

	count, ok := outputs["count"].(int)
	if !ok {
		t.Error("expected count in outputs")
	}
	if count != 2 {
		t.Errorf("expected 2 skills, got %d", count)
	}

	skillList, ok := outputs["skills"].([]map[string]any)
	if !ok {
		t.Error("expected skills array in outputs")
	}
	if len(skillList) != 2 {
		t.Errorf("expected 2 skills in list, got %d", len(skillList))
	}
}

func TestQAAgent_HealthCheck(t *testing.T) {
	agent := NewQAAgent(nil, &QAConfig{Model: "test-model"})

	skill := &skills.BaseSkill{
		NameVal:          "test:skill",
		DisplayNameVal:   "Test Skill",
		DescriptionVal:   "Test",
		SystemPromptVal:  "Prompt",
		FocusAreasVal:    []string{"test"},
		RiskToleranceVal: "moderate",
		CategoryVal:      "analyst",
	}
	agent.RegisterSkill(skill)

	results := agent.HealthCheck()
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
	if results[0].SkillID != "test:skill" {
		t.Errorf("expected skill ID 'test:skill', got '%s'", results[0].SkillID)
	}
}

func TestQAAgent_GetSkillDetail(t *testing.T) {
	agent := NewQAAgent(nil, &QAConfig{Model: "test-model"})

	skill := &skills.BaseSkill{
		NameVal:          "test:skill",
		DisplayNameVal:   "Test Skill",
		DescriptionVal:   "Test Description",
		SystemPromptVal:  "Test Prompt",
		FocusAreasVal:    []string{"area1"},
		RiskToleranceVal: "moderate",
		CategoryVal:      "analyst",
	}
	agent.RegisterSkill(skill)

	detail, err := agent.GetSkillDetail("test:skill")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
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

	_, err = agent.GetSkillDetail("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent skill")
	}
}

func TestQASession_ActiveSkill(t *testing.T) {
	session := NewQASession("AAPL", "2024-01-15")

	if session.ActiveSkill != nil {
		t.Error("expected nil ActiveSkill initially")
	}

	skill := &skills.BaseSkill{
		NameVal:        "test:skill",
		DisplayNameVal: "Test Skill",
	}
	session.ActiveSkill = skill

	if session.ActiveSkill.Name() != "test:skill" {
		t.Errorf("expected skill name 'test:skill', got '%s'", session.ActiveSkill.Name())
	}
}

func TestQAAgent_SkillCommand_Help(t *testing.T) {
	agent := NewQAAgent(nil, &QAConfig{Model: "test-model"})

	skill := &skills.BaseSkill{
		NameVal:          "test:skill",
		DisplayNameVal:   "Test Skill",
		DescriptionVal:   "Test Description",
		RiskToleranceVal: "moderate",
		CategoryVal:      "analyst",
		RequiredParamsVal: []skills.SkillParam{
			{Name: "symbol", Type: "string", Description: "股票代码"},
		},
		OptionalParamsVal: []skills.SkillParam{
			{Name: "depth", Type: "string", Description: "分析深度"},
		},
	}
	agent.RegisterSkill(skill)

	result, err := agent.handleSkillCommand("/skill")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	outputs := result.Outputs

	skillHint, ok := outputs["skill_hint"].(*skills.SkillHint)
	if !ok {
		t.Fatal("expected skill_hint to be *skills.SkillHint")
	}
	if skillHint.ID != "test:skill" {
		t.Errorf("expected skill_hint ID 'test:skill', got '%s'", skillHint.ID)
	}
	if skillHint.DisplayName != "Test Skill" {
		t.Errorf("expected skill_hint DisplayName 'Test Skill', got '%s'", skillHint.DisplayName)
	}
	if skillHint.InvokeExample != "/skill <股票代码> [分析深度]" {
		t.Errorf("expected skill_hint InvokeExample '/skill <股票代码> [分析深度]', got '%s'", skillHint.InvokeExample)
	}
	if len(skillHint.RequiredParams) != 1 {
		t.Errorf("expected 1 required param, got %d", len(skillHint.RequiredParams))
	}
	if skillHint.RequiredParams[0].Description != "股票代码" {
		t.Errorf("expected required param description '股票代码', got '%s'", skillHint.RequiredParams[0].Description)
	}
	if len(skillHint.OptionalParams) != 1 {
		t.Errorf("expected 1 optional param, got %d", len(skillHint.OptionalParams))
	}
	if skillHint.OptionalParams[0].Description != "分析深度" {
		t.Errorf("expected optional param description '分析深度', got '%s'", skillHint.OptionalParams[0].Description)
	}
}

func TestQAAgent_SkillCommand_List(t *testing.T) {
	agent := NewQAAgent(nil, &QAConfig{Model: "test-model"})

	skill1 := &skills.BaseSkill{
		NameVal:        "test:skill1",
		DisplayNameVal: "Skill 1",
		CategoryVal:    "analyst",
	}
	skill2 := &skills.BaseSkill{
		NameVal:        "test:skill2",
		DisplayNameVal: "Skill 2",
		CategoryVal:    "strategy",
	}
	agent.RegisterSkill(skill1)
	agent.RegisterSkill(skill2)

	result, err := agent.handleSkillCommand("/")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	outputs := result.Outputs

	count := outputs["count"].(int)
	if count != 2 {
		t.Errorf("expected count 2, got %d", count)
	}

	skillHints, ok := outputs["skill_hints"].([]skills.SkillHint)
	if !ok {
		t.Fatal("expected skill_hints to be []skills.SkillHint")
	}
	if len(skillHints) != 2 {
		t.Errorf("expected 2 skill hints, got %d", len(skillHints))
	}

	foundSkill1 := false
	foundSkill2 := false
	for _, hint := range skillHints {
		if hint.ID == "test:skill1" {
			foundSkill1 = true
			if hint.DisplayName != "Skill 1" {
				t.Errorf("expected DisplayName 'Skill 1', got '%s'", hint.DisplayName)
			}
			if hint.InvokeExample != "/skill1" {
				t.Errorf("expected InvokeExample '/skill1', got '%s'", hint.InvokeExample)
			}
		}
		if hint.ID == "test:skill2" {
			foundSkill2 = true
			if hint.DisplayName != "Skill 2" {
				t.Errorf("expected DisplayName 'Skill 2', got '%s'", hint.DisplayName)
			}
			if hint.InvokeExample != "/skill2" {
				t.Errorf("expected InvokeExample '/skill2', got '%s'", hint.InvokeExample)
			}
		}
	}
	if !foundSkill1 {
		t.Error("expected skill1 to be found in skill_hints")
	}
	if !foundSkill2 {
		t.Error("expected skill2 to be found in skill_hints")
	}
}

func TestQAAgent_SkillCommand_Invoke(t *testing.T) {
	agent := NewQAAgent(nil, &QAConfig{Model: "test-model"})

	skill := &skills.BaseSkill{
		NameVal:          "test:skill",
		DisplayNameVal:   "Test Skill",
		RequiredParamsVal: []skills.SkillParam{
			{Name: "symbol", Type: "string", Description: "股票代码"},
		},
	}
	agent.RegisterSkill(skill)

	result, err := agent.handleSkillCommand("/skill AAPL")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	outputs := result.Outputs

	if outputs["action"] != "skill_invoke" {
		t.Errorf("expected action 'skill_invoke', got '%v'", outputs["action"])
	}

	skillHint, ok := outputs["skill_hint"].(*skills.SkillHint)
	if !ok {
		t.Fatal("expected skill_hint to be *skills.SkillHint")
	}
	if skillHint.DisplayName != "Test Skill" {
		t.Errorf("expected skill_hint DisplayName 'Test Skill', got '%s'", skillHint.DisplayName)
	}

	params := outputs["params"].(map[string]interface{})
	if params["symbol"] != "AAPL" {
		t.Errorf("expected symbol 'AAPL', got '%v'", params["symbol"])
	}
}

func TestQAAgent_SkillCommand_MissingParams(t *testing.T) {
	agent := NewQAAgent(nil, &QAConfig{Model: "test-model"})

	skill := &skills.BaseSkill{
		NameVal:          "test:skill",
		DisplayNameVal:   "Test Skill",
		RequiredParamsVal: []skills.SkillParam{
			{Name: "symbol", Type: "string", Description: "股票代码"},
			{Name: "date", Type: "string", Description: "交易日期"},
		},
	}
	agent.RegisterSkill(skill)

	result, err := agent.handleSkillCommand("/skill AAPL")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	outputs := result.Outputs

	if outputs["error"] != "参数错误" {
		t.Errorf("expected error '参数错误', got '%v'", outputs["error"])
	}

	skillHint, ok := outputs["skill_hint"].(*skills.SkillHint)
	if !ok {
		t.Fatal("expected skill_hint to be *skills.SkillHint")
	}
	if skillHint.InvokeExample != "/skill <股票代码> <交易日期>" {
		t.Errorf("expected InvokeExample '/skill <股票代码> <交易日期>', got '%s'", skillHint.InvokeExample)
	}
}

func TestQAAgent_SkillCommand_NotFound(t *testing.T) {
	agent := NewQAAgent(nil, &QAConfig{Model: "test-model"})

	result, err := agent.handleSkillCommand("/nonexistent")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	outputs := result.Outputs

	count := outputs["count"].(int)
	if count != 0 {
		t.Errorf("expected count 0, got %d", count)
	}

	skillHints, ok := outputs["skill_hints"].([]skills.SkillHint)
	if !ok {
		t.Fatal("expected skill_hints to be []skills.SkillHint")
	}
	if len(skillHints) != 0 {
		t.Errorf("expected 0 skill hints, got %d", len(skillHints))
	}
}

func TestParseIntentResponse(t *testing.T) {
	tests := []struct {
		input    string
		expected IntentType
	}{
		{`{"intent": "data_query", "confidence": 0.95, "reasoning": "asking about PE"}`, IntentDataQuery},
		{`some text {"intent": "risk_inquiry", "confidence": 0.8} more text`, IntentRiskInquiry},
		{`invalid json`, IntentGeneral},
		{`{}`, IntentGeneral},
	}

	for _, tt := range tests {
		result, _ := parseIntentResponse(tt.input)
		if result.Intent != tt.expected {
			t.Errorf("for '%s': expected %s, got %s", tt.input, tt.expected, result.Intent)
		}
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestIntegration_QAAgentWithPluginSkills(t *testing.T) {
	agent := NewQAAgent(nil, &QAConfig{Model: "test-model"})
	agent.pluginLoader = skills.NewPluginLoaderWithPaths(agent.skillRegistry, "../../skills/plugin", "../../skills/registry")

	err := agent.LoadPlugins()
	if err != nil {
		t.Fatalf("Failed to load plugins: %v", err)
	}

	skills := agent.SkillRegistry().List()
	t.Logf("Loaded %d skills", len(skills))
	for _, s := range skills {
		t.Logf("  Skill: %s", s.Name())
	}
	if len(skills) == 0 {
		t.Skip("No skills loaded, skipping integration test")
	}

	found := false
	for _, s := range skills {
		if s.Name() == "ai-berkshire:investment-research" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected ai-berkshire:investment-research to be loaded")
	}
}

func TestIntegration_QAAgentSkillDetail(t *testing.T) {
	agent := NewQAAgent(nil, &QAConfig{Model: "test-model"})
	agent.pluginLoader = skills.NewPluginLoaderWithPaths(agent.skillRegistry, "../../skills/plugin", "../../skills/registry")

	err := agent.LoadPlugins()
	if err != nil {
		t.Fatalf("Failed to load plugins: %v", err)
	}

	if len(agent.SkillRegistry().List()) == 0 {
		t.Skip("No skills loaded, skipping integration test")
	}

	detail, err := agent.GetSkillDetail("ai-berkshire:investment-research")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if detail == "" {
		t.Error("Expected non-empty detail")
	}
	if !contains(detail, "深度投资研究") {
		t.Error("Detail should contain display name")
	}
}

func TestIntegration_QAAgentHealthCheck(t *testing.T) {
	agent := NewQAAgent(nil, &QAConfig{Model: "test-model"})
	agent.pluginLoader = skills.NewPluginLoaderWithPaths(agent.skillRegistry, "../../skills/plugin", "../../skills/registry")

	err := agent.LoadPlugins()
	if err != nil {
		t.Fatalf("Failed to load plugins: %v", err)
	}

	if len(agent.SkillRegistry().List()) == 0 {
		t.Skip("No skills loaded, skipping integration test")
	}

	results := agent.HealthCheck()
	if len(results) == 0 {
		t.Error("Expected health check results")
	}

	for _, r := range results {
		if r.Status == "critical" {
			t.Errorf("Found critical skill: %s - %s", r.SkillID, r.Message)
		}
	}
}
