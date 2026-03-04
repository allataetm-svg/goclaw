package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/allataetm-svg/goclaw/internal/config"
	"gopkg.in/yaml.v3"
)

type Skill struct {
	Name          string        `yaml:"name"`
	Description   string        `yaml:"description"`
	Version       string        `yaml:"version,omitempty"`
	Author        string        `yaml:"author,omitempty"`
	Triggers      []string      `yaml:"triggers,omitempty"`
	Tools         []string      `yaml:"tools,omitempty"`
	Env           []string      `yaml:"env,omitempty"`
	Bins          []string      `yaml:"bins,omitempty"`
	UserInvocable bool          `yaml:"user_invocable,omitempty"`
	Metadata      SkillMetadata `yaml:"metadata,omitempty"`
}

type SkillMetadata struct {
	Emoji                  string            `yaml:"emoji,omitempty"`
	Requires               SkillRequirements `yaml:"requires,omitempty"`
	PrimaryEnv             string            `yaml:"primaryEnv,omitempty"`
	DisableModelInvocation bool              `yaml:"disable_model_invocation,omitempty"`
}

type SkillRequirements struct {
	Env  []string `yaml:"env,omitempty"`
	Bins []string `yaml:"bins,omitempty"`
}

type SkillEntry struct {
	Enabled bool              `yaml:"enabled"`
	APIKey  string            `yaml:"api_key,omitempty"`
	Env     map[string]string `yaml:"env,omitempty"`
}

type SkillsConfig struct {
	AllowBundled []string              `yaml:"allow_bundled,omitempty"`
	Load         LoadConfig            `yaml:"load,omitempty"`
	Install      InstallConfig         `yaml:"install,omitempty"`
	Entries      map[string]SkillEntry `yaml:"entries,omitempty"`
}

type LoadConfig struct {
	ExtraDirs       []string `yaml:"extra_dirs,omitempty"`
	Watch           bool     `yaml:"watch,omitempty"`
	WatchDebounceMs int      `yaml:"watch_debounce_ms,omitempty"`
}

type InstallConfig struct {
	PreferBrew  bool   `yaml:"prefer_brew,omitempty"`
	NodeManager string `yaml:"node_manager,omitempty"`
}

type LoadedSkill struct {
	Metadata Skill
	Content  string
	Dir      string
	FilePath string
}

func GetSkillsDir() string {
	return filepath.Join(config.GetConfigDir(), "skills")
}

func GetSkillsDirForAgent(agentID string) string {
	return filepath.Join(config.GetConfigDir(), "agents", agentID, "skills")
}

func GetBundledSkillsDir() string {
	execPath, err := os.Executable()
	if err != nil {
		return ""
	}
	return filepath.Join(filepath.Dir(execPath), "skills")
}

func (s *LoadedSkill) GetInstructions() string {
	lines := strings.Split(s.Content, "\n")
	inContent := false
	var resultLines []string
	for _, line := range lines {
		if len(line) >= 3 && line[:3] == "---" {
			if !inContent {
				inContent = true
				continue
			}
			break
		}
		if inContent {
			resultLines = append(resultLines, line)
		}
	}
	return strings.Join(resultLines, "\n")
}

func (s *LoadedSkill) GetBaseDir() string {
	return s.Dir
}

func ParseSkillFromFile(filePath string) (*LoadedSkill, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read skill file: %w", err)
	}

	content := string(data)

	var skill Skill
	if err := yaml.Unmarshal([]byte(content), &skill); err != nil {
		return nil, fmt.Errorf("failed to parse skill metadata: %w", err)
	}

	if skill.Name == "" {
		return nil, fmt.Errorf("skill name is required")
	}

	dir := filepath.Dir(filePath)

	return &LoadedSkill{
		Metadata: skill,
		Content:  content,
		Dir:      dir,
		FilePath: filePath,
	}, nil
}
