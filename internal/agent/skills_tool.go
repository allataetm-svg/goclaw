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
	case "add":
		return addSkill(args)
	case "remove":
		skillID, _ := args["skill_id"].(string)
		if skillID == "" {
			return "", fmt.Errorf("missing skill_id")
		}
		if err := skills.RemoveSkill(skillID); err != nil {
			return "", err
		}
		return "Skill removed successfully.", nil
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
	var enabledSkillIDs []string
	if agentID != "" {
		enabledSkills, err := skills.LoadAgentSkills(agentID)
		if err == nil {
			for _, s := range enabledSkills {
				enabledSkillIDs = append(enabledSkillIDs, s.ID)
			}
		}
	}

	var sb strings.Builder
	sb.WriteString("## Available Skills\n\n")

	for _, s := range skillList {
		sb.WriteString(fmt.Sprintf("### %s\n", s.Name))
		sb.WriteString(fmt.Sprintf("- ID: %s\n", s.ID))
		sb.WriteString(fmt.Sprintf("- Description: %s\n", s.Description))
		sb.WriteString(fmt.Sprintf("- Version: %s\n", s.Version))
		sb.WriteString(fmt.Sprintf("- Source: %s\n", s.Source))

		if len(enabledSkillIDs) > 0 {
			enabled := false
			for _, id := range enabledSkillIDs {
				if id == s.ID {
					enabled = true
					break
				}
			}
			if enabled {
				sb.WriteString("- Status: **enabled**\n")
			}
		}

		if len(s.Tools) > 0 {
			sb.WriteString(fmt.Sprintf("- Tools: %s\n", strings.Join(s.Tools, ", ")))
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
		sb.WriteString(fmt.Sprintf("- ID: %s\n", s.ID))
		sb.WriteString(fmt.Sprintf("- Description: %s\n", s.Description))
		sb.WriteString(fmt.Sprintf("- Source: %s\n", s.Source))
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

func addSkill(args map[string]interface{}) (string, error) {
	name, _ := args["name"].(string)
	description, _ := args["description"].(string)
	source, _ := args["source"].(string)

	if name == "" {
		return "", fmt.Errorf("missing name parameter")
	}

	if source == "" {
		source = "global"
	}

	skill := skills.Skill{
		ID:          fmt.Sprintf("skill_%s", strings.ToLower(strings.ReplaceAll(name, " ", "_"))),
		Name:        name,
		Description: description,
		Version:     "1.0.0",
		Source:      source,
		Tools:       []string{},
	}

	if tools, ok := args["tools"].([]interface{}); ok {
		for _, t := range tools {
			if tStr, ok := t.(string); ok {
				skill.Tools = append(skill.Tools, tStr)
			}
		}
	}

	if err := skills.AddSkill(skill); err != nil {
		return "", err
	}

	return fmt.Sprintf("Skill '%s' added successfully.", name), nil
}

func manageAgentSkill(args map[string]interface{}, enable bool) (string, error) {
	agentID, _ := args["agent_id"].(string)
	skillID, _ := args["skill_id"].(string)

	if agentID == "" || skillID == "" {
		return "", fmt.Errorf("missing agent_id or skill_id")
	}

	currentSkills, err := skills.LoadAgentSkills(agentID)
	if err != nil {
		return "", err
	}

	var currentIDs []string
	for _, s := range currentSkills {
		currentIDs = append(currentIDs, s.ID)
	}

	exists := false
	for _, id := range currentIDs {
		if id == skillID {
			exists = true
			break
		}
	}

	if enable && !exists {
		currentIDs = append(currentIDs, skillID)
	} else if !enable && exists {
		var newIDs []string
		for _, id := range currentIDs {
			if id != skillID {
				newIDs = append(newIDs, id)
			}
		}
		currentIDs = newIDs
	}

	if err := skills.SaveAgentSkills(agentID, currentIDs); err != nil {
		return "", err
	}

	action := "enabled"
	if !enable {
		action = "disabled"
	}
	return fmt.Sprintf("Skill %s for agent %s.", skillID, action), nil
}
