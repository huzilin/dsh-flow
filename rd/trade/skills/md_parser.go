package skills

import (
	"strings"
)

type SkillMD struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Sections    map[string]string `json:"sections"`
	Parameters  []SkillParam      `json:"parameters"`
	Workflow    []WorkflowStep    `json:"workflow"`
	SystemPrompt string           `json:"system_prompt"`
}

type WorkflowStep struct {
	Step       int    `json:"step"`
	Name       string `json:"name"`
	Action     string `json:"action"`
	Target     string `json:"target"`
	OutputType string `json:"output_type"`
}

type MarkdownParser struct{}

func NewMarkdownParser() *MarkdownParser {
	return &MarkdownParser{}
}

func (mp *MarkdownParser) Parse(content string) *SkillMD {
	lines := strings.Split(content, "\n")
	result := &SkillMD{
		Sections: make(map[string]string),
	}

	var currentSection string
	var currentSectionContent strings.Builder
	inCodeBlock := false

	for _, line := range lines {
		if strings.HasPrefix(line, "# ") {
			if currentSection != "" {
				result.Sections[currentSection] = strings.TrimSpace(currentSectionContent.String())
			}
			result.Title = strings.TrimPrefix(line, "# ")
			currentSection = "title"
			currentSectionContent.Reset()
			continue
		}

		if strings.HasPrefix(line, "## ") {
			if currentSection != "" {
				result.Sections[currentSection] = strings.TrimSpace(currentSectionContent.String())
			}
			currentSection = strings.TrimPrefix(line, "## ")
			currentSectionContent.Reset()
			continue
		}

		if strings.HasPrefix(line, "### ") {
			if currentSection != "" {
				result.Sections[currentSection] = strings.TrimSpace(currentSectionContent.String())
			}
			currentSection = strings.TrimPrefix(line, "### ")
			currentSectionContent.Reset()
			continue
		}

		if strings.HasPrefix(line, "```") {
			inCodeBlock = !inCodeBlock
		}

		if !inCodeBlock {
			currentSectionContent.WriteString(line)
			currentSectionContent.WriteString("\n")
		}
	}

	if currentSection != "" {
		result.Sections[currentSection] = strings.TrimSpace(currentSectionContent.String())
	}

	result.SystemPrompt = mp.extractSystemPrompt(result)
	result.Description = mp.extractDescription(result)
	result.Workflow = mp.extractWorkflow(result)

	return result
}

func (mp *MarkdownParser) extractSystemPrompt(md *SkillMD) string {
	prioritySections := []string{
		"系统提示词",
		"System Prompt",
		"system prompt",
		"分析框架",
		"分析方法论",
		"分析方法",
		"工作流程",
		"工作流",
		"流程",
	}

	for _, section := range prioritySections {
		if content, ok := md.Sections[section]; ok && content != "" {
			return content
		}
	}

	for name, content := range md.Sections {
		if strings.Contains(strings.ToLower(name), "prompt") ||
			strings.Contains(strings.ToLower(name), "framework") ||
			strings.Contains(strings.ToLower(name), "analysis") ||
			strings.Contains(strings.ToLower(name), "方法") {
			return content
		}
	}

	var builder strings.Builder
	for name, content := range md.Sections {
		if name != "title" && name != "description" && content != "" {
			builder.WriteString("## " + name + "\n")
			builder.WriteString(content + "\n\n")
		}
	}

	return strings.TrimSpace(builder.String())
}

func (mp *MarkdownParser) extractDescription(md *SkillMD) string {
	for _, key := range []string{"description", "Description", "描述", "简介"} {
		if content, ok := md.Sections[key]; ok && content != "" {
			return content
		}
	}

	if md.Title != "" {
		return md.Title
	}

	return ""
}

func (mp *MarkdownParser) extractWorkflow(md *SkillMD) []WorkflowStep {
	var steps []WorkflowStep

	for name, content := range md.Sections {
		if strings.Contains(strings.ToLower(name), "step") ||
			strings.Contains(strings.ToLower(name), "流程") ||
			strings.Contains(strings.ToLower(name), "阶段") {

			lines := strings.Split(content, "\n")
			stepNum := 0
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}

				if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "1.") ||
					strings.HasPrefix(line, "2.") || strings.HasPrefix(line, "3.") ||
					strings.HasPrefix(line, "4.") || strings.HasPrefix(line, "5.") {

					stepNum++
					stepName := strings.TrimPrefix(line, "- ")
					stepName = strings.TrimPrefix(stepName, "1.")
					stepName = strings.TrimPrefix(stepName, "2.")
					stepName = strings.TrimPrefix(stepName, "3.")
					stepName = strings.TrimPrefix(stepName, "4.")
					stepName = strings.TrimPrefix(stepName, "5.")
					stepName = strings.TrimSpace(stepName)

					steps = append(steps, WorkflowStep{
						Step: stepNum,
						Name: stepName,
					})
				}
			}
		}
	}

	return steps
}

func (mp *MarkdownParser) ExtractParameters(content string) []SkillParam {
	var params []SkillParam

	if content == "" {
		return params
	}

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
			line = strings.TrimPrefix(line, "- ")
			line = strings.TrimPrefix(line, "* ")

			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				name := strings.TrimSpace(parts[0])
				desc := strings.TrimSpace(parts[1])

				isRequired := strings.Contains(strings.ToLower(line), "required") ||
					strings.Contains(strings.ToLower(line), "必填")

				param := SkillParam{
					Name:        name,
					Description: desc,
					Required:    isRequired,
				}

				params = append(params, param)
			}
		}
	}

	return params
}

func (mp *MarkdownParser) GetSection(md *SkillMD, sectionName string) string {
	if content, ok := md.Sections[sectionName]; ok {
		return content
	}

	for name, content := range md.Sections {
		if strings.EqualFold(name, sectionName) {
			return content
		}
	}

	return ""
}

func (mp *MarkdownParser) HasSection(md *SkillMD, sectionName string) bool {
	if _, ok := md.Sections[sectionName]; ok {
		return true
	}

	for name := range md.Sections {
		if strings.EqualFold(name, sectionName) {
			return true
		}
	}

	return false
}