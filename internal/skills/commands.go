package skills

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/allataetm-svg/goclaw/internal/config"
)

func ListSkillsCLI() error {
	allSkills, err := LoadAllGlobalSkills()
	if err != nil {
		return fmt.Errorf("failed to load skills: %w", err)
	}

	if len(allSkills) == 0 {
		fmt.Println("No skills found.")
		return nil
	}

	fmt.Printf("Found %d skills:\n\n", len(allSkills))
	for _, s := range allSkills {
		emoji := ""
		if s.Metadata.Metadata.Emoji != "" {
			emoji = s.Metadata.Metadata.Emoji + " "
		}
		fmt.Printf("%s**%s** - %s\n", emoji, s.Metadata.Name, s.Metadata.Description)
		if s.Metadata.Version != "" {
			fmt.Printf("   Version: %s\n", s.Metadata.Version)
		}
		if len(s.Metadata.Triggers) > 0 {
			fmt.Printf("   Triggers: %s\n", joinStrings(s.Metadata.Triggers, ", "))
		}
		fmt.Println()
	}

	return nil
}

func ListAgentSkillsCLI(agentID string) error {
	agentSkills, err := LoadSkillsForAgent(agentID)
	if err != nil {
		return fmt.Errorf("failed to load skills: %w", err)
	}

	if len(agentSkills) == 0 {
		fmt.Printf("No skills found for agent %s.\n", agentID)
		return nil
	}

	fmt.Printf("Found %d skills for agent %s:\n\n", len(agentSkills), agentID)
	for _, s := range agentSkills {
		emoji := ""
		if s.Metadata.Metadata.Emoji != "" {
			emoji = s.Metadata.Metadata.Emoji + " "
		}
		fmt.Printf("%s**%s** - %s\n", emoji, s.Metadata.Name, s.Metadata.Description)
		fmt.Printf("   Location: %s\n", s.Dir)
		fmt.Println()
	}

	return nil
}

func CreateSkill(name, description string) error {
	if name == "" {
		return fmt.Errorf("skill name is required")
	}

	skillsDir := GetSkillsDir()
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		return fmt.Errorf("failed to create skills directory: %w", err)
	}

	skillDir := filepath.Join(skillsDir, name)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return fmt.Errorf("failed to create skill directory: %w", err)
	}

	skillPath := filepath.Join(skillDir, "SKILL.md")
	content := fmt.Sprintf(`---
name: %s
description: %s
version: 1.0.0
---

# %s

Describe what this skill does and when to use it.

## Instructions

1. First step
2. Second step
3. Third step

## Examples

- Example 1
- Example 2

## Notes

Any additional notes or considerations.
`, name, description, name)

	if err := os.WriteFile(skillPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write SKILL.md: %w", err)
	}

	fmt.Printf("Skill '%s' created at %s\n", name, skillPath)
	return nil
}

func ShowSkill(name string) error {
	allSkills, err := LoadAllGlobalSkills()
	if err != nil {
		return fmt.Errorf("failed to load skills: %w", err)
	}

	for _, s := range allSkills {
		if s.Metadata.Name == name {
			fmt.Printf("# Skill: %s\n\n", s.Metadata.Name)
			fmt.Printf("**Description:** %s\n\n", s.Metadata.Description)
			if s.Metadata.Version != "" {
				fmt.Printf("**Version:** %s\n\n", s.Metadata.Version)
			}
			if len(s.Metadata.Triggers) > 0 {
				fmt.Printf("**Triggers:** %s\n\n", joinStrings(s.Metadata.Triggers, ", "))
			}
			fmt.Println("---")
			fmt.Println(s.GetInstructions())
			return nil
		}
	}

	return fmt.Errorf("skill '%s' not found", name)
}

func EnableSkill(name string) error {
	conf, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if conf.Skills.Entries == nil {
		conf.Skills.Entries = make(map[string]config.SkillEntry)
	}

	if entry, exists := conf.Skills.Entries[name]; exists {
		entry.Enabled = true
		conf.Skills.Entries[name] = entry
	} else {
		conf.Skills.Entries[name] = config.SkillEntry{Enabled: true}
	}

	if err := config.Save(conf); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Skill '%s' enabled.\n", name)
	return nil
}

func DisableSkill(name string) error {
	conf, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if conf.Skills.Entries == nil {
		fmt.Printf("Skill '%s' is not configured.\n", name)
		return nil
	}

	if entry, exists := conf.Skills.Entries[name]; exists {
		entry.Enabled = false
		conf.Skills.Entries[name] = entry
	} else {
		conf.Skills.Entries[name] = config.SkillEntry{Enabled: false}
	}

	if err := config.Save(conf); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Skill '%s' disabled.\n", name)
	return nil
}

func GetSkillsDirPath() string {
	return GetSkillsDir()
}

func joinStrings(slice []string, sep string) string {
	result := ""
	for i, s := range slice {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}
