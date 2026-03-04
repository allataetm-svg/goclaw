package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type SkillLoader struct {
	globalSkillsDir  string
	agentSkillsDirs  []string
	bundledSkillsDir string
	extraDirs        []string
}

func NewSkillLoader() *SkillLoader {
	return &SkillLoader{
		globalSkillsDir:  GetSkillsDir(),
		bundledSkillsDir: GetBundledSkillsDir(),
	}
}

func (l *SkillLoader) SetAgentSkillsDir(agentID string) {
	agentDir := GetSkillsDirForAgent(agentID)
	l.agentSkillsDirs = append(l.agentSkillsDirs, agentDir)
}

func (l *SkillLoader) AddExtraDir(dir string) {
	l.extraDirs = append(l.extraDirs, dir)
}

func (l *SkillLoader) LoadAll() ([]*LoadedSkill, error) {
	var allSkills []*LoadedSkill
	seen := make(map[string]bool)

	dirs := []string{}
	dirs = append(dirs, l.extraDirs...)
	dirs = append(dirs, l.agentSkillsDirs...)
	dirs = append(dirs, l.globalSkillsDir)
	dirs = append(dirs, l.bundledSkillsDir)

	for _, dir := range dirs {
		if dir == "" {
			continue
		}
		skills, err := l.loadFromDir(dir)
		if err != nil {
			continue
		}
		for _, s := range skills {
			if !seen[s.Metadata.Name] {
				seen[s.Metadata.Name] = true
				allSkills = append(allSkills, s)
			}
		}
	}

	return allSkills, nil
}

func (l *SkillLoader) LoadForAgent(agentID string) ([]*LoadedSkill, error) {
	allSkills, err := l.LoadAll()
	if err != nil {
		return nil, err
	}

	var agentSkills []*LoadedSkill
	agentDir := GetSkillsDirForAgent(agentID)

	for _, s := range allSkills {
		if s.Dir == agentDir || strings.HasPrefix(s.Dir, agentDir) {
			agentSkills = append(agentSkills, s)
		}
	}

	return agentSkills, nil
}

func (l *SkillLoader) loadFromDir(dir string) ([]*LoadedSkill, error) {
	var skills []*LoadedSkill

	entries, err := os.ReadDir(dir)
	if err != nil {
		return skills, nil
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillPath := filepath.Join(dir, entry.Name(), "SKILL.md")
		if _, err := os.Stat(skillPath); os.IsNotExist(err) {
			continue
		}

		skill, err := ParseSkillFromFile(skillPath)
		if err != nil {
			continue
		}

		skills = append(skills, skill)
	}

	return skills, nil
}

func LoadAllGlobalSkills() ([]*LoadedSkill, error) {
	loader := NewSkillLoader()
	return loader.LoadAll()
}

func LoadSkillsForAgent(agentID string) ([]*LoadedSkill, error) {
	loader := NewSkillLoader()
	loader.SetAgentSkillsDir(agentID)
	return loader.LoadForAgent(agentID)
}

func GetSkillSummary(skills []*LoadedSkill) string {
	var lines []string
	for _, s := range skills {
		lines = append(lines, fmt.Sprintf("- **%s**: %s", s.Metadata.Name, s.Metadata.Description))
	}
	return strings.Join(lines, "\n")
}

func GetSkillPrompts(skills []*LoadedSkill) []string {
	var prompts []string
	for _, s := range skills {
		instructions := s.GetInstructions()
		if instructions != "" {
			prompts = append(prompts, instructions)
		}
	}
	return prompts
}

func LoadSkills() error {
	_, err := LoadAllGlobalSkills()
	return err
}

func ListSkills() []Skill {
	loaded, err := LoadAllGlobalSkills()
	if err != nil {
		return []Skill{}
	}
	var result []Skill
	for _, ls := range loaded {
		result = append(result, ls.Metadata)
	}
	return result
}

func SearchSkills(query string) []Skill {
	allSkills := ListSkills()
	query = strings.ToLower(query)
	var results []Skill
	for _, s := range allSkills {
		if strings.Contains(strings.ToLower(s.Name), query) ||
			strings.Contains(strings.ToLower(s.Description), query) {
			results = append(results, s)
		}
	}
	return results
}
