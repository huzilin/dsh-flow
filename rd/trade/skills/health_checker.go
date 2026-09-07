package skills

import (
	"fmt"
	"os"

	"github.com/huzilin/trade/llm"
	"go.uber.org/zap"
)

type HealthStatus string

const (
	StatusHealthy   HealthStatus = "healthy"
	StatusWarning   HealthStatus = "warning"
	StatusCritical  HealthStatus = "critical"
	StatusUnknown   HealthStatus = "unknown"
)

type HealthResult struct {
	SkillID   string       `json:"skill_id"`
	Status    HealthStatus `json:"status"`
	Message   string       `json:"message,omitempty"`
	SkillFile string       `json:"skill_file,omitempty"`
}

type HealthChecker struct {
	registry *SkillRegistry
	logger   *zap.Logger
}

func NewHealthChecker(registry *SkillRegistry) *HealthChecker {
	return &HealthChecker{
		registry: registry,
		logger:   llm.Logger(),
	}
}

func (hc *HealthChecker) Check() []HealthResult {
	results := make([]HealthResult, 0)
	skills := hc.registry.List()

	for _, skill := range skills {
		result := HealthResult{
			SkillID:   skill.Name(),
			Status:    StatusHealthy,
			SkillFile: skill.SkillFile(),
		}

		if err := hc.checkSkillFile(skill); err != nil {
			result.Status = StatusCritical
			result.Message = err.Error()
		} else if err := hc.checkParams(skill); err != nil {
			if result.Status == StatusHealthy {
				result.Status = StatusWarning
			}
			if result.Message != "" {
				result.Message += "; " + err.Error()
			} else {
				result.Message = err.Error()
			}
		}

		results = append(results, result)
	}

	if hc.logger != nil {
		hc.logger.Info("Health check completed",
			zap.Int("total", len(results)),
			zap.Int("healthy", hc.countStatus(results, StatusHealthy)),
			zap.Int("warning", hc.countStatus(results, StatusWarning)),
			zap.Int("critical", hc.countStatus(results, StatusCritical)))
	}

	return results
}

func (hc *HealthChecker) checkSkillFile(skill Skill) error {
	if skill.SkillFile() == "" {
		return nil
	}

	_, err := os.Stat(skill.SkillFile())
	if os.IsNotExist(err) {
		return fmt.Errorf("skill file not found: %s", skill.SkillFile())
	}
	if err != nil {
		return fmt.Errorf("failed to stat skill file: %w", err)
	}

	return nil
}

func (hc *HealthChecker) checkParams(skill Skill) error {
	if len(skill.RequiredParams()) == 0 && len(skill.OptionalParams()) == 0 {
		return fmt.Errorf("no parameters defined")
	}

	for _, p := range skill.RequiredParams() {
		if p.Name == "" {
			return fmt.Errorf("required param has empty name")
		}
	}

	return nil
}

func (hc *HealthChecker) countStatus(results []HealthResult, status HealthStatus) int {
	count := 0
	for _, r := range results {
		if r.Status == status {
			count++
		}
	}
	return count
}

func (hc *HealthChecker) GetCriticalSkills() []Skill {
	var criticalSkills []Skill
	results := hc.Check()

	for _, result := range results {
		if result.Status == StatusCritical {
			if skill, ok := hc.registry.Get(result.SkillID); ok {
				criticalSkills = append(criticalSkills, skill)
			}
		}
	}

	return criticalSkills
}

func (hc *HealthChecker) GetWarningSkills() []Skill {
	var warningSkills []Skill
	results := hc.Check()

	for _, result := range results {
		if result.Status == StatusWarning {
			if skill, ok := hc.registry.Get(result.SkillID); ok {
				warningSkills = append(warningSkills, skill)
			}
		}
	}

	return warningSkills
}

func (hc *HealthChecker) IsAllHealthy() bool {
	results := hc.Check()
	for _, result := range results {
		if result.Status == StatusCritical {
			return false
		}
	}
	return true
}