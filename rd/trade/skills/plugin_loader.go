package skills

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/huzilin/trade/llm"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type PluginConfig struct {
	Name           string          `yaml:"name"`
	Version        string          `yaml:"version"`
	RepoURL        string          `yaml:"repo_url"`
	LocalPath      string          `yaml:"local_path"`
	Branch         string          `yaml:"branch,omitempty"`
	CommitHash     string          `yaml:"commit_hash,omitempty"`
	Tag            string          `yaml:"tag,omitempty"`
	Skills         []PluginSkill   `yaml:"skills"`
	CategoryGroups []CategoryGroup `yaml:"category_groups,omitempty"`
}

type CategoryGroup struct {
	Name   string   `yaml:"name"`
	Skills []string `yaml:"skills"`
}

type PluginSkill struct {
	ID             string       `yaml:"id"`
	DisplayName    string       `yaml:"display_name"`
	Description    string       `yaml:"description"`
	SkillFile      string       `yaml:"skill_file"`
	RequiredParams []SkillParam `yaml:"required_params,omitempty"`
	OptionalParams []SkillParam `yaml:"optional_params,omitempty"`
	Category       string       `yaml:"category,omitempty"`
	FocusAreas     []string     `yaml:"focus_areas,omitempty"`
	RiskTolerance  string       `yaml:"risk_tolerance,omitempty"`
}

type PluginLoader struct {
	registry    *SkillRegistry
	pluginsDir  string
	registryDir string
	mdParser    *MarkdownParser
}

func NewPluginLoader(registry *SkillRegistry) *PluginLoader {
	return NewPluginLoaderWithPaths(registry, "./skills/plugin", "./skills/registry")
}

func NewPluginLoaderWithPaths(registry *SkillRegistry, pluginsDir, registryDir string) *PluginLoader {
	return &PluginLoader{
		registry:    registry,
		pluginsDir:  pluginsDir,
		registryDir: registryDir,
		mdParser:    NewMarkdownParser(),
	}
}

func (pl *PluginLoader) LoadPlugins() error {
	logger := llm.Logger()
	if logger != nil {
		logger.Info("Loading plugins...")
	}

	err := pl.ensureDirectories()
	if err != nil {
		return fmt.Errorf("failed to ensure directories: %w", err)
	}

	configFiles, err := filepath.Glob(filepath.Join(pl.registryDir, "*.yaml"))
	if err != nil {
		return fmt.Errorf("failed to glob config files: %w", err)
	}

	if len(configFiles) == 0 {
		if logger != nil {
			logger.Warn("No plugin config files found in registry directory")
		}
		return nil
	}

	for _, configFile := range configFiles {
		err := pl.loadPlugin(configFile)
		if err != nil {
			if logger != nil {
				logger.Error("Failed to load plugin",
					zap.String("config_file", configFile),
					zap.Error(err))
			}
			continue
		}
	}

	if logger != nil {
		logger.Info("Plugins loaded successfully",
			zap.Int("skill_count", len(pl.registry.Names())))
	}

	return nil
}

func (pl *PluginLoader) ensureDirectories() error {
	for _, dir := range []string{pl.pluginsDir, pl.registryDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	return nil
}

func (pl *PluginLoader) loadPlugin(configFile string) error {
	content, err := ioutil.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var config PluginConfig
	if err := yaml.Unmarshal(content, &config); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	logger := llm.Logger()
	if logger != nil {
		logger.Info("Loading plugin",
			zap.String("name", config.Name),
			zap.String("version", config.Version))
	}

	err = pl.ensurePluginRepo(&config)
	if err != nil {
		return fmt.Errorf("failed to ensure plugin repo: %w", err)
	}

	for _, skillDef := range config.Skills {
		err := pl.registerPluginSkill(&config, &skillDef)
		if err != nil {
			if logger != nil {
				logger.Error("Failed to register skill",
					zap.String("skill_id", skillDef.ID),
					zap.Error(err))
			}
			continue
		}
	}

	return nil
}

func (pl *PluginLoader) ensurePluginRepo(config *PluginConfig) error {
	repoPath := filepath.Join(pl.pluginsDir, config.Name)
	logger := llm.Logger()

	_, err := os.Stat(repoPath)
	if os.IsNotExist(err) {
		if logger != nil {
			logger.Info("Cloning plugin repo",
				zap.String("repo_url", config.RepoURL),
				zap.String("path", repoPath),
				zap.String("branch", config.Branch))
		}

		var args []string
		if config.Branch != "" {
			args = []string{"clone", "-b", config.Branch, config.RepoURL, repoPath}
		} else {
			args = []string{"clone", config.RepoURL, repoPath}
		}

		cmd := exec.Command("git", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("git clone failed: %w", err)
		}
	} else if err == nil {
		if logger != nil {
			logger.Info("Plugin repo already exists, updating",
				zap.String("path", repoPath),
				zap.String("branch", config.Branch))
		}

		if config.Tag != "" {
			cmd := exec.Command("git", "-C", repoPath, "checkout", config.Tag)
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("git checkout tag failed: %w", err)
			}
		} else if config.CommitHash != "" {
			cmd := exec.Command("git", "-C", repoPath, "checkout", config.CommitHash)
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("git checkout commit failed: %w", err)
			}
		} else if config.Branch != "" {
			cmd := exec.Command("git", "-C", repoPath, "checkout", config.Branch)
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("git checkout branch failed: %w", err)
			}

			cmd = exec.Command("git", "-C", repoPath, "pull", "origin", config.Branch)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				if logger != nil {
					logger.Warn("Git pull failed, using existing version",
						zap.String("path", repoPath),
						zap.Error(err))
				}
			}
		} else {
			cmd := exec.Command("git", "-C", repoPath, "pull")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				if logger != nil {
					logger.Warn("Git pull failed, using existing version",
						zap.String("path", repoPath),
						zap.Error(err))
				}
			}
		}
	} else {
		return fmt.Errorf("failed to check repo path: %w", err)
	}

	return nil
}

func (pl *PluginLoader) registerPluginSkill(config *PluginConfig, skillDef *PluginSkill) error {
	skillFilePath := filepath.Join(pl.pluginsDir, config.Name, skillDef.SkillFile)

	if _, err := os.Stat(skillFilePath); os.IsNotExist(err) {
		return fmt.Errorf("skill file not found: %s", skillFilePath)
	}

	var systemPrompt string
	content, err := ioutil.ReadFile(skillFilePath)
	logger := llm.Logger()
	if err != nil {
		if logger != nil {
			logger.Warn("Failed to read skill file, using description as prompt",
				zap.String("file", skillFilePath),
				zap.Error(err))
		}
		systemPrompt = skillDef.Description
	} else {
		parsed := pl.mdParser.Parse(string(content))
		if parsed.SystemPrompt != "" {
			systemPrompt = parsed.SystemPrompt
		} else {
			systemPrompt = string(content)
		}
	}

	skill := &BaseSkill{
		NameVal:           fmt.Sprintf("%s:%s", config.Name, skillDef.ID),
		DisplayNameVal:    skillDef.DisplayName,
		DescriptionVal:    skillDef.Description,
		SystemPromptVal:   systemPrompt,
		FocusAreasVal:     skillDef.FocusAreas,
		RiskToleranceVal:  skillDef.RiskTolerance,
		RequiredParamsVal: skillDef.RequiredParams,
		OptionalParamsVal: skillDef.OptionalParams,
		SkillFileVal:      skillFilePath,
		CategoryVal:       skillDef.Category,
	}

	pl.registry.Register(skill)
	if logger != nil {
		logger.Info("Registered plugin skill",
			zap.String("name", skill.Name()),
			zap.String("display_name", skill.DisplayName()))
	}

	return nil
}

func ValidateSkillID(id string) error {
	if !strings.Contains(id, ":") {
		return fmt.Errorf("skill ID must contain namespace prefix (format: namespace:name)")
	}
	parts := strings.SplitN(id, ":", 2)
	if len(parts[0]) == 0 || len(parts[1]) == 0 {
		return fmt.Errorf("invalid skill ID format")
	}
	return nil
}

func (pl *PluginLoader) ListPlugins() ([]string, error) {
	files, err := filepath.Glob(filepath.Join(pl.registryDir, "*.yaml"))
	if err != nil {
		return nil, err
	}

	var plugins []string
	for _, file := range files {
		name := strings.TrimSuffix(filepath.Base(file), ".yaml")
		plugins = append(plugins, name)
	}

	return plugins, nil
}

func (pl *PluginLoader) GetPluginInfo(pluginName string) (*PluginConfig, error) {
	configFile := filepath.Join(pl.registryDir, pluginName+".yaml")
	content, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, err
	}

	var config PluginConfig
	if err := yaml.Unmarshal(content, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func (pl *PluginLoader) GetSkillDetail(skillID string) (string, error) {
	skill, ok := pl.registry.Get(skillID)
	if !ok {
		return "", fmt.Errorf("skill not found: %s", skillID)
	}

	detail := map[string]interface{}{
		"name":            skill.Name(),
		"display_name":    skill.DisplayName(),
		"description":     skill.Description(),
		"category":        skill.Category(),
		"risk_tolerance":  skill.RiskTolerance(),
		"focus_areas":     skill.FocusAreas(),
		"required_params": skill.RequiredParams(),
		"optional_params": skill.OptionalParams(),
		"has_skill_file":  skill.SkillFile() != "",
	}

	data, err := json.MarshalIndent(detail, "", "  ")
	if err != nil {
		return "", err
	}

	return string(data), nil
}
