package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/allataetm-svg/goclaw/internal/config"
	"github.com/allataetm-svg/goclaw/internal/heartbeat"
)

type HeartbeatTool struct{}

func (t *HeartbeatTool) Name() string { return "heartbeat" }

func (t *HeartbeatTool) Description() string {
	return `Manages heartbeat services. Args: { "action": "string (add|list|remove|enable|disable)", "name": "string", "agent_id": "string", "interval_min": "number", "prompt": "string", "heartbeat_id": "string" }`
}

func (t *HeartbeatTool) Execute(ctx context.Context, args map[string]interface{}, conf config.Config) (string, error) {
	action, _ := args["action"].(string)
	if action == "" {
		return "", fmt.Errorf("missing action parameter")
	}

	switch action {
	case "list":
		return listHeartbeats()
	case "add":
		return addHeartbeat(args)
	case "remove":
		hbID, _ := args["heartbeat_id"].(string)
		if hbID == "" {
			return "", fmt.Errorf("missing heartbeat_id")
		}
		if err := heartbeat.RemoveHeartbeat(hbID); err != nil {
			return "", err
		}
		return "Heartbeat removed successfully.", nil
	case "enable":
		hbID, _ := args["heartbeat_id"].(string)
		if hbID == "" {
			return "", fmt.Errorf("missing heartbeat_id")
		}
		if err := heartbeat.EnableHeartbeat(hbID); err != nil {
			return "", err
		}
		return "Heartbeat enabled successfully.", nil
	case "disable":
		hbID, _ := args["heartbeat_id"].(string)
		if hbID == "" {
			return "", fmt.Errorf("missing heartbeat_id")
		}
		if err := heartbeat.DisableHeartbeat(hbID); err != nil {
			return "", err
		}
		return "Heartbeat disabled successfully.", nil
	default:
		return "", fmt.Errorf("unknown action: %s", action)
	}
}

func listHeartbeats() (string, error) {
	heartbeats := heartbeat.ListHeartbeats()
	if len(heartbeats) == 0 {
		return "No heartbeat services configured.", nil
	}

	var sb strings.Builder
	sb.WriteString("## Heartbeat Services\n\n")

	for _, h := range heartbeats {
		status := "enabled"
		if !h.Enabled {
			status = "disabled"
		}

		sb.WriteString(fmt.Sprintf("### %s (%s)\n", h.Name, status))
		sb.WriteString(fmt.Sprintf("- ID: %s\n", h.ID))
		sb.WriteString(fmt.Sprintf("- Agent: %s\n", h.AgentID))
		sb.WriteString(fmt.Sprintf("- Interval: %d minutes\n", h.IntervalMin))
		sb.WriteString(fmt.Sprintf("- Prompt: %s\n", truncate(h.Prompt, 100)))

		if h.LastRun != nil {
			sb.WriteString(fmt.Sprintf("- Last Run: %s\n", h.LastRun.Format(time.RFC3339)))
		}
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

func addHeartbeat(args map[string]interface{}) (string, error) {
	name, _ := args["name"].(string)
	agentID, _ := args["agent_id"].(string)
	prompt, _ := args["prompt"].(string)
	intervalMin, _ := args["interval_min"].(float64)

	if name == "" || agentID == "" || prompt == "" || intervalMin == 0 {
		return "", fmt.Errorf("missing required parameters: name, agent_id, prompt, interval_min")
	}

	if intervalMin < 5 {
		return "", fmt.Errorf("minimum interval is 5 minutes")
	}

	hb := heartbeat.HeartbeatConfig{
		ID:          fmt.Sprintf("hb_%d", time.Now().Unix()),
		Name:        name,
		AgentID:     agentID,
		IntervalMin: int(intervalMin),
		Prompt:      prompt,
		Enabled:     true,
	}

	if err := heartbeat.AddHeartbeat(hb); err != nil {
		return "", err
	}

	return fmt.Sprintf("Heartbeat '%s' added successfully (min interval: %d minutes).", name, intervalMin), nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
