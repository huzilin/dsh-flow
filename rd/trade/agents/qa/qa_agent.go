package qa

import (
	"context"
	"fmt"
	"strings"

	"github.com/huzilin/trade/llm"
	"github.com/huzilin/trade/skills"
	"github.com/huzilin/trade/types"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type QAAgent struct {
	llmClient     *llm.OpenAIClientV2
	classifier    *IntentClassifier
	retriever     *Retriever
	skillRegistry *skills.SkillRegistry
	sessions      map[string]*QASessionState
	model         string
	pluginLoader  *skills.PluginLoader
	healthChecker *skills.HealthChecker
}

type QAConfig struct {
	Model string
}

func NewQAAgent(client *llm.OpenAIClientV2, config *QAConfig) *QAAgent {
	if config == nil {
		config = &QAConfig{}
	}

	registry := skills.NewSkillRegistry()

	return &QAAgent{
		llmClient:     client,
		classifier:    NewIntentClassifier(client, config.Model),
		retriever:     NewRetriever(),
		skillRegistry: registry,
		sessions:      make(map[string]*QASessionState),
		model:         config.Model,
		pluginLoader:  skills.NewPluginLoader(registry),
		healthChecker: skills.NewHealthChecker(registry),
	}
}

func (a *QAAgent) SkillRegistry() *skills.SkillRegistry {
	return a.skillRegistry
}

func (a *QAAgent) RegisterSkill(s skills.Skill) {
	a.skillRegistry.Register(s)
}

func (a *QAAgent) LoadPlugins() error {
	return a.pluginLoader.LoadPlugins()
}

func (a *QAAgent) HealthCheck() []skills.HealthResult {
	return a.healthChecker.Check()
}

func (a *QAAgent) GetSkillDetail(skillID string) (string, error) {
	return a.pluginLoader.GetSkillDetail(skillID)
}

func (a *QAAgent) Run(request any, execCtx any) (any, error) {
	req, ok := request.(*types.AgentRunRequest)
	if !ok {
		return types.AgentRunResult{}, errors.New("invalid request type for QA agent")
	}

	return a.handleRequest(req)
}

func (a *QAAgent) GetName() string {
	return "QA"
}

func (a *QAAgent) handleRequest(req *types.AgentRunRequest) (types.AgentRunResult, error) {
	question, _ := req.Context["question"].(string)

	if question == "" {
		if skillList, ok := req.Context["list_skills"].(bool); ok && skillList {
			return a.listSkills(), nil
		}
		return types.AgentRunResult{}, errors.New("question is required")
	}

	if strings.HasPrefix(question, "/") {
		return a.handleSkillCommand(question)
	}

	sessionID, _ := req.Context["session_id"].(string)
	if sessionID == "" {
		sessionID = req.Symbol + "_" + req.TradeDate
	}

	skillName, _ := req.Context["skill"].(string)

	llm.Logger().Info("QA Agent processing question",
		zap.String("session", sessionID),
		zap.String("question", question),
		zap.String("skill", skillName))

	session, exists := a.sessions[sessionID]
	if !exists {
		session = NewQASession(req.Symbol, req.TradeDate)
		a.sessions[sessionID] = session
	}

	if skillName != "" {
		if sk, ok := a.skillRegistry.Get(skillName); ok {
			session.ActiveSkill = sk
			llm.Logger().Info("Activated skill", zap.String("skill", sk.DisplayName()))
		}
	}

	a.updateSessionFromContext(session, req.Context)

	response, err := a.answer(question, session)
	if err != nil {
		llm.Logger().Error("QA answer failed", zap.Error(err))
		return types.AgentRunResult{}, err
	}

	session.AddMessage("user", question)
	session.AddMessage("assistant", response.Answer)

	outputs := map[string]any{
		"answer":      response.Answer,
		"intent_type": response.IntentType,
		"confidence":  response.Confidence,
		"citations":   response.Citations,
		"disclaimer":  response.Disclaimer,
		"session_id":  sessionID,
	}
	if session.ActiveSkill != nil {
		outputs["active_skill"] = session.ActiveSkill.DisplayName()
	}

	return types.AgentRunResult{
		AgentName: "QA",
		Outputs:   outputs,
	}, nil
}

func (a *QAAgent) listSkills() types.AgentRunResult {
	skills := a.skillRegistry.List()
	var skillList []map[string]any
	for _, s := range skills {
		skillInfo := map[string]any{
			"name":           s.Name(),
			"display_name":   s.DisplayName(),
			"description":    s.Description(),
			"risk":           s.RiskTolerance(),
			"category":       s.Category(),
			"focus_areas":    s.FocusAreas(),
			"has_skill_file": s.SkillFile() != "",
		}

		if len(s.RequiredParams()) > 0 {
			var params []map[string]string
			for _, p := range s.RequiredParams() {
				params = append(params, map[string]string{
					"name":        p.Name,
					"type":        p.Type,
					"description": p.Description,
					"required":    "true",
				})
			}
			skillInfo["required_params"] = params
		}

		if len(s.OptionalParams()) > 0 {
			var params []map[string]string
			for _, p := range s.OptionalParams() {
				params = append(params, map[string]string{
					"name":        p.Name,
					"type":        p.Type,
					"description": p.Description,
					"required":    "false",
				})
			}
			skillInfo["optional_params"] = params
		}

		skillList = append(skillList, skillInfo)
	}
	return types.AgentRunResult{
		AgentName: "QA",
		Outputs: map[string]any{
			"skills": skillList,
			"count":  len(skillList),
		},
	}
}

func (a *QAAgent) updateSessionFromContext(session *QASessionState, ctx map[string]any) {
	market, _ := ctx["market_report"].(string)
	sentiment, _ := ctx["sentiment_report"].(string)
	news, _ := ctx["news_report"].(string)
	fundamentals, _ := ctx["fundamentals_report"].(string)
	plan, _ := ctx["investment_plan"].(string)
	decision, _ := ctx["final_trade_decision"].(string)

	if market != "" || sentiment != "" || news != "" || fundamentals != "" || plan != "" || decision != "" {
		session.SetAnalysisContext(market, sentiment, news, fundamentals, plan, decision)
		a.retriever.IndexAnalysisContext(session.AnalysisCtx)
	}
}

func (a *QAAgent) answer(question string, session *QASessionState) (*QAResponse, error) {
	intent, err := a.classifier.Classify(question, session.BuildContextForLLM())
	if err != nil {
		llm.Logger().Warn("Intent classification failed, using general", zap.Error(err))
		intent = &IntentResult{Intent: IntentGeneral, Confidence: 0.5}
	}

	docs := a.retrieveWithSkillWeights(question, session.ActiveSkill)
	contextText := BuildContextText(docs)
	if contextText == "" {
		contextText = session.BuildContextForLLM()
	}

	answer, err := a.generateAnswerWithSkill(question, contextText, session.BuildConversationHistory(), intent, session.ActiveSkill)
	if err != nil {
		return nil, errors.Wrap(err, "LLM answer generation failed")
	}

	var citations []string
	for _, doc := range docs {
		if doc.Score > 1.0 {
			citations = append(citations, doc.Source)
		}
	}

	return &QAResponse{
		Answer:     answer,
		IntentType: string(intent.Intent),
		Confidence: intent.Confidence,
		Citations:  citations,
		Disclaimer: defaultDisclaimer,
	}, nil
}

func (a *QAAgent) retrieveWithSkillWeights(question string, skill skills.Skill) []RetrievedDoc {
	docs := a.retriever.Retrieve(question, 5)
	if skill == nil || len(docs) == 0 {
		return docs
	}

	areas := skill.FocusAreas()
	for i := range docs {
		for _, area := range areas {
			if strings.Contains(strings.ToLower(docs[i].Content), strings.ToLower(area)) {
				docs[i].Score *= 1.5
			}
		}
	}

	return docs
}

func (a *QAAgent) generateAnswerWithSkill(question, ctxText, history string, intent *IntentResult, skill skills.Skill) (string, error) {
	var systemPrompt string
	if skill != nil {
		systemPrompt = skills.BuildSkillPrompt(skill, ctxText)
	} else {
		systemPrompt = a.buildSystemPrompt(ctxText)
	}

	messages := []*llm.ChatMessage{
		{Role: "system", Content: systemPrompt},
	}

	messages = append(messages, &llm.ChatMessage{
		Role:    "user",
		Content: a.buildUserPrompt(question, intent),
	})

	options := &llm.ChatOptions{
		Model:       a.model,
		Temperature: 0.3,
		MaxTokens:   2000,
	}

	resp, err := a.llmClient.Chat(context.Background(), messages, options)
	if err != nil {
		return "", err
	}

	return resp.Content, nil
}

func (a *QAAgent) buildSystemPrompt(analysisContext string) string {
	return fmt.Sprintf(`你是 TradingAgents 的专业投资顾问，基于多 Agent 分析结果为用户提供专业解答。

规则：
1. 回答必须基于提供的分析报告内容，不得编造数据
2. 引用来源必须标注具体报告名称
3. 涉及具体操作建议时，必须附加投资风险提示
4. 不确定的信息明确说明"根据现有数据无法确定"
5. 保持专业、客观、中立的语气
6. 使用中文回答

分析报告内容：
%s`, analysisContext)
}

func (a *QAAgent) buildUserPrompt(question string, intent *IntentResult) string {
	intentGuides := map[IntentType]string{
		IntentReportInterpretation: "请基于分析报告解释用户的疑问，引用报告中的具体内容。",
		IntentDataQuery:            "请提供具体的数据指标和数值，引用来源。",
		IntentComparison:           "请进行横向对比分析，列出关键差异。",
		IntentRiskInquiry:          "请详细说明风险因素，并给出风险等级评估。",
		IntentStrategyAdvice:       "请给出具体的操作建议，并附上风险提示。",
		IntentMarketInsight:        "请分析宏观市场环境和趋势。",
		IntentGeneral:              "请基于分析报告内容提供全面的回答。",
	}

	guide, ok := intentGuides[intent.Intent]
	if !ok {
		guide = intentGuides[IntentGeneral]
	}

	cleanContext := cleanAnalysisContext(a.getLatestContext())

	return fmt.Sprintf(`%s

用户问题：%s

请提供：
1. 直接回答
2. 引用的数据来源
3. 相关风险提示（如适用）`, guide, question) + "\n\n相关分析：" + truncateText(cleanContext, 3000)
}

func (a *QAAgent) getLatestContext() string {
	if len(a.retriever.sections) > 0 {
		var builder strings.Builder
		for _, section := range a.retriever.sections {
			builder.WriteString(section.content)
			builder.WriteString("\n")
		}
		return builder.String()
	}
	return ""
}

func cleanAnalysisContext(text string) string {
	lines := strings.Split(text, "\n")
	var cleaned []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") || strings.HasPrefix(trimmed, "# ") {
			continue
		}
		if trimmed == "---" {
			continue
		}
		cleaned = append(cleaned, line)
	}
	return strings.Join(cleaned, "\n")
}

func (a *QAAgent) handleSkillCommand(command string) (types.AgentRunResult, error) {
	command = strings.TrimSpace(command)

	if command == "/" {
		return a.listSkillsWithExamples(), nil
	}

	parts := strings.SplitN(command[1:], " ", 2)
	skillID := parts[0]

	skill, ok := a.skillRegistry.Get(skillID)
	if !ok {
		skill, ok = a.skillRegistry.Get("ai-berkshire:" + skillID)
	}

	if !ok {
		for _, s := range a.skillRegistry.List() {
			name := s.Name()
			if strings.HasSuffix(name, ":"+skillID) {
				skill = s
				ok = true
				break
			}
		}
	}

	if !ok {
		return a.listSkillsWithExamples(), nil
	}

	if len(parts) == 1 {
		return a.showSkillHelp(skill), nil
	}

	paramsStr := strings.TrimSpace(parts[1])
	params := a.parseSkillParams(skill, paramsStr)

	err := skill.ValidateParams(params)
	if err != nil {
		hint, _ := a.skillRegistry.GetSkillHint(skill.Name())
		return types.AgentRunResult{
			AgentName: "QA",
			Outputs: map[string]any{
				"error":         "参数错误",
				"error_message": err.Error(),
				"skill_hint":    hint,
			},
		}, nil
	}

	hint, _ := a.skillRegistry.GetSkillHint(skill.Name())
	return types.AgentRunResult{
		AgentName: "QA",
		Outputs: map[string]any{
			"action":     "skill_invoke",
			"skill_hint": hint,
			"params":     params,
		},
	}, nil
}

func (a *QAAgent) listSkillsWithExamples() types.AgentRunResult {
	hints := a.skillRegistry.GetSkillHints()

	return types.AgentRunResult{
		AgentName: "QA",
		Outputs: map[string]any{
			"skill_hints": hints,
			"count":       len(hints),
		},
	}
}

func (a *QAAgent) showSkillHelp(skill skills.Skill) types.AgentRunResult {
	hint, _ := a.skillRegistry.GetSkillHint(skill.Name())

	return types.AgentRunResult{
		AgentName: "QA",
		Outputs: map[string]any{
			"skill_hint":  hint,
			"focus_areas": skill.FocusAreas(),
		},
	}
}

func (a *QAAgent) parseSkillParams(skill skills.Skill, paramsStr string) map[string]interface{} {
	params := make(map[string]interface{})

	requiredCount := len(skill.RequiredParams())
	optionalCount := len(skill.OptionalParams())
	totalCount := requiredCount + optionalCount

	parts := strings.Fields(paramsStr)

	for i, part := range parts {
		if i >= totalCount {
			break
		}
		if i < requiredCount {
			params[skill.RequiredParams()[i].Name] = part
		} else {
			params[skill.OptionalParams()[i-requiredCount].Name] = part
		}
	}

	for i := len(parts); i < requiredCount; i++ {
		p := skill.RequiredParams()[i]
		if p.Default != nil {
			params[p.Name] = p.Default
		}
	}

	for i := max(len(parts), requiredCount); i < totalCount; i++ {
		p := skill.OptionalParams()[i-requiredCount]
		if p.Default != nil {
			params[p.Name] = p.Default
		}
	}

	return params
}

func (a *QAAgent) buildSkillListForOutput(skills []skills.Skill) []map[string]any {
	var result []map[string]any
	for _, s := range skills {
		result = append(result, map[string]any{
			"name":           s.Name(),
			"display_name":   s.DisplayName(),
			"description":    s.Description(),
			"invoke_example": s.InvokeExample(),
			"category":       s.Category(),
			"risk_tolerance": s.RiskTolerance(),
		})
	}
	return result
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
