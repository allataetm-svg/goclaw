package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/allataetm-svg/goclaw/internal/config"
	"github.com/allataetm-svg/goclaw/internal/sessions"
)

type SessionsTool struct{}

func (t *SessionsTool) Name() string { return "sessions" }

func (t *SessionsTool) Description() string {
	return `Manages agent sessions. Args: { "action": "string (list|history|send|spawn)", "session_id": "string", "channel_id": "string", "user_id": "string", "agent_id": "string", "message": "string" }`
}

func (t *SessionsTool) Execute(ctx context.Context, args map[string]interface{}, conf config.Config) (string, error) {
	action, _ := args["action"].(string)
	if action == "" {
		return "", fmt.Errorf("missing action parameter")
	}

	switch action {
	case "list":
		return listSessions(args)
	case "history":
		return getSessionHistory(args)
	case "send":
		return sendToSession(args)
	case "spawn":
		return spawnSession(ctx, args, conf)
	default:
		return "", fmt.Errorf("unknown action: %s", action)
	}
}

func listSessions(args map[string]interface{}) (string, error) {
	channelID, _ := args["channel_id"].(string)
	userID, _ := args["user_id"].(string)

	var sessionList []sessions.Session

	if channelID != "" {
		sessionList = sessions.GetChannelSessions(channelID)
	} else if userID != "" {
		sessionList = sessions.GetUserSessions(userID)
	} else {
		sessionList = sessions.ListActiveSessions()
	}

	if len(sessionList) == 0 {
		return "No active sessions found.", nil
	}

	var sb strings.Builder
	sb.WriteString("## Active Sessions\n\n")

	for _, s := range sessionList {
		sb.WriteString(fmt.Sprintf("### Session: %s\n", s.ID))
		sb.WriteString(fmt.Sprintf("- Channel: %s\n", s.ChannelID))
		sb.WriteString(fmt.Sprintf("- User: %s\n", s.UserID))
		sb.WriteString(fmt.Sprintf("- Agent: %s\n", s.AgentID))
		sb.WriteString(fmt.Sprintf("- Started: %s\n", s.StartedAt.Format(time.RFC3339)))
		sb.WriteString(fmt.Sprintf("- Last Active: %s\n", s.LastActive.Format(time.RFC3339)))
		sb.WriteString(fmt.Sprintf("- Status: %s\n", s.Status))
		if s.LastMessage != "" {
			sb.WriteString(fmt.Sprintf("- Last Message: %s\n", truncate(s.LastMessage, 50)))
		}
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

func getSessionHistory(args map[string]interface{}) (string, error) {
	sessionID, _ := args["session_id"].(string)
	if sessionID == "" {
		return "", fmt.Errorf("missing session_id parameter")
	}

	session, ok := sessions.GetSession(sessionID)
	if !ok {
		return "", fmt.Errorf("session not found: %s", sessionID)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## Session: %s\n\n", session.ID))
	sb.WriteString(fmt.Sprintf("- Channel: %s\n", session.ChannelID))
	sb.WriteString(fmt.Sprintf("- User: %s\n", session.UserID))
	sb.WriteString(fmt.Sprintf("- Agent: %s\n", session.AgentID))
	sb.WriteString(fmt.Sprintf("- Started: %s\n", session.StartedAt.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("- Last Active: %s\n", session.LastActive.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("- Status: %s\n", session.Status))

	if len(session.Metadata) > 0 {
		sb.WriteString("\n### Metadata:\n")
		for k, v := range session.Metadata {
			sb.WriteString(fmt.Sprintf("- %s: %s\n", k, v))
		}
	}

	return sb.String(), nil
}

func sendToSession(args map[string]interface{}) (string, error) {
	sessionID, _ := args["session_id"].(string)
	message, _ := args["message"].(string)

	if sessionID == "" || message == "" {
		return "", fmt.Errorf("missing session_id or message parameter")
	}

	if err := sessions.SendMessageToSession(sessionID, message); err != nil {
		return "", err
	}

	return fmt.Sprintf("Message sent to session %s.", sessionID), nil
}

func spawnSession(ctx context.Context, args map[string]interface{}, conf config.Config) (string, error) {
	channelID, _ := args["channel_id"].(string)
	userID, _ := args["user_id"].(string)
	agentID, _ := args["agent_id"].(string)

	if channelID == "" || userID == "" || agentID == "" {
		return "", fmt.Errorf("missing required parameters: channel_id, user_id, agent_id")
	}

	session, err := sessions.CreateSession(channelID, userID, agentID)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("New session created: %s", session.ID), nil
}
