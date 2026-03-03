package skills

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/allataetm-svg/goclaw/internal/config"
)

type Skill struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Version     string            `json:"version"`
	Source      string            `json:"source"` // workspace, global, builtin
	Tools       []string          `json:"tools"`
	Prompts     []SkillPrompt     `json:"prompts"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type SkillPrompt struct {
	Name    string `json:"name"`
	Prompt  string `json:"prompt"`
	Pattern string `json:"pattern,omitempty"`
}

var (
	skills   = make(map[string]*Skill)
	skillsMu sync.RWMutex
	loaded   bool
)

func GetSkillsDir() string {
	return filepath.Join(config.GetConfigDir(), "skills")
}

func LoadSkills() error {
	if loaded {
		return nil
	}

	skillsMu.Lock()
	defer skillsMu.Unlock()

	if err := loadBuiltinSkills(); err != nil {
		return err
	}

	if err := loadGlobalSkills(); err != nil {
		return err
	}

	loaded = true
	return nil
}

func loadBuiltinSkills() error {
	builtinDir := filepath.Join(GetSkillsDir(), "builtin")
	if err := os.MkdirAll(builtinDir, 0755); err != nil {
		return err
	}
	return loadSkillsFromDir(builtinDir, "builtin")
}

func loadGlobalSkills() error {
	globalDir := filepath.Join(GetSkillsDir(), "global")
	if err := os.MkdirAll(globalDir, 0755); err != nil {
		return err
	}
	return loadSkillsFromDir(globalDir, "global")
}

func loadSkillsFromDir(dir, source string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			skillDir := filepath.Join(dir, entry.Name())
			skill, err := loadSkillFromDir(skillDir, source)
			if err != nil {
				continue
			}
			skills[skill.ID] = skill
		}
	}
	return nil
}

func loadSkillFromDir(dir, source string) (*Skill, error) {
	data, err := os.ReadFile(filepath.Join(dir, "skill.json"))
	if err != nil {
		return nil, err
	}

	var skill Skill
	if err := json.Unmarshal(data, &skill); err != nil {
		return nil, err
	}

	skill.Source = source

	return &skill, nil
}

func LoadAgentSkills(agentID string) ([]Skill, error) {
	skillsMu.RLock()
	defer skillsMu.RUnlock()

	agentSkillsDir := filepath.Join(config.GetConfigDir(), "agents", agentID, "skills")
	data, err := os.ReadFile(filepath.Join(agentSkillsDir, "skills.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return []Skill{}, nil
		}
		return nil, err
	}

	var skillIDs []string
	if err := json.Unmarshal(data, &skillIDs); err != nil {
		return nil, err
	}

	var result []Skill
	for _, id := range skillIDs {
		if s, ok := skills[id]; ok {
			result = append(result, *s)
		}
	}

	return result, nil
}

func SaveAgentSkills(agentID string, skillIDs []string) error {
	agentSkillsDir := filepath.Join(config.GetConfigDir(), "agents", agentID, "skills")
	if err := os.MkdirAll(agentSkillsDir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(skillIDs, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(agentSkillsDir, "skills.json"), data, 0644)
}

func ListSkills() []Skill {
	skillsMu.RLock()
	defer skillsMu.RUnlock()

	all := make([]Skill, 0, len(skills))
	for _, s := range skills {
		all = append(all, *s)
	}
	return all
}

func GetSkill(id string) (*Skill, bool) {
	skillsMu.RLock()
	defer skillsMu.RUnlock()
	s, ok := skills[id]
	return s, ok
}

func AddSkill(s Skill) error {
	skillsMu.Lock()
	defer skillsMu.Unlock()

	if _, exists := skills[s.ID]; exists {
		return fmt.Errorf("skill with id %s already exists", s.ID)
	}

	dir := filepath.Join(GetSkillsDir(), s.Source, s.ID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(dir, "skill.json"), data, 0644); err != nil {
		return err
	}

	skills[s.ID] = &s
	return nil
}

func RemoveSkill(id string) error {
	skillsMu.Lock()
	defer skillsMu.Unlock()

	s, exists := skills[id]
	if !exists {
		return fmt.Errorf("skill not found: %s", id)
	}

	dir := filepath.Join(GetSkillsDir(), s.Source, s.ID)
	if err := os.RemoveAll(dir); err != nil {
		return err
	}

	delete(skills, id)
	return nil
}

func SearchSkills(query string) []Skill {
	skillsMu.RLock()
	defer skillsMu.RUnlock()

	var result []Skill
	query = strings.ToLower(query)

	for _, s := range skills {
		if strings.Contains(strings.ToLower(s.Name), query) ||
			strings.Contains(strings.ToLower(s.Description), query) {
			result = append(result, *s)
		}
	}

	return result
}
