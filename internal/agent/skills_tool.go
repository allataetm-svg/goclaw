package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/allataetm-svg/goclaw/internal/config"
	"github.com/allataetm-svg/goclaw/internal/skills"
)

type SkillsTool struct{}

func (t *SkillsTool) Name() string { return "skills" }

func (t *SkillsTool) Description() string {
	return `Manages skill packages. Args: { "action": "string (list|search|enable|disable|add)", "query": "string", "skill_id": "string", "agent_id": "string", "name": "string", "description": "string", "tools": "array", "source": "string" }`
}

func (t *SkillsTool) Execute(ctx context.Context, args map[string]interface{}, conf config.Config) (string, error) {
	action, _ := args["action"].(string)
	if action == "" {
		return "", fmt.Errorf("missing action parameter")
	}

	if err := skills.LoadSkills(); err != nil {
		return "", fmt.Errorf("failed to load skills: %w", err)
	}

	switch action {
	case "list":
		return listSkills(args)
	case "search":
		return searchSkills(args)
	case "enable":
		return manageAgentSkill(args, true)
	case "disable":
		return manageAgentSkill(args, false)
	default:
		return "", fmt.Errorf("unknown action: %s", action)
	}
}

func listSkills(args map[string]interface{}) (string, error) {
	skillList := skills.ListSkills()
	if len(skillList) == 0 {
		return "No skills available.", nil
	}

	agentID, _ := args["agent_id"].(string)
	var enabledSkillNames []string
	if agentID != "" {
		enabledSkills, err := skills.LoadSkillsForAgent(agentID)
		if err == nil {
			for _, s := range enabledSkills {
				enabledSkillNames = append(enabledSkillNames, s.Metadata.Name)
			}
		}
	}

	var sb strings.Builder
	sb.WriteString("## Available Skills\n\n")

	for _, s := range skillList {
		sb.WriteString(fmt.Sprintf("### %s\n", s.Name))
		sb.WriteString(fmt.Sprintf("- Description: %s\n", s.Description))
		sb.WriteString(fmt.Sprintf("- Version: %s\n", s.Version))

		if len(enabledSkillNames) > 0 {
			enabled := false
			for _, name := range enabledSkillNames {
				if name == s.Name {
					enabled = true
					break
				}
			}
			if enabled {
				sb.WriteString("- Status: **enabled**\n")
			}
		}

		if len(s.Triggers) > 0 {
			sb.WriteString(fmt.Sprintf("- Triggers: %s\n", strings.Join(s.Triggers, ", ")))
		}
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

func searchSkills(args map[string]interface{}) (string, error) {
	query, _ := args["query"].(string)
	if query == "" {
		return "", fmt.Errorf("missing query parameter")
	}

	results := skills.SearchSkills(query)
	if len(results) == 0 {
		return fmt.Sprintf("No skills found matching '%s'.", query), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## Search Results for '%s'\n\n", query))

	for _, s := range results {
		sb.WriteString(fmt.Sprintf("### %s\n", s.Name))
		sb.WriteString(fmt.Sprintf("- Description: %s\n", s.Description))
		sb.WriteString(fmt.Sprintf("- Version: %s\n", s.Version))
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

func manageAgentSkill(args map[string]interface{}, enable bool) (string, error) {
	name, _ := args["name"].(string)
	skillID, _ := args["skill_id"].(string)

	if name == "" && skillID == "" {
		return "", fmt.Errorf("missing name or skill_id parameter")
	}

	skillName := name
	if skillName == "" {
		skillName = skillID
	}

	if enable {
		if err := skills.EnableSkill(skillName); err != nil {
			return "", err
		}
		return fmt.Sprintf("Skill '%s' enabled.", skillName), nil
	} else {
		if err := skills.DisableSkill(skillName); err != nil {
			return "", err
		}
		return fmt.Sprintf("Skill '%s' disabled.", skillName), nil
	}
}
