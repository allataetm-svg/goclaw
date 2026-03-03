package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/allataetm-svg/goclaw/internal/config"
	"github.com/allataetm-svg/goclaw/internal/secrets"
)

type SecretsTool struct{}

func (t *SecretsTool) Name() string { return "secrets" }

func (t *SecretsTool) Description() string {
	return `Manages workspace secrets for agent. Args: { "action": "string (add|verify|approve|revoke|check)", "agent_id": "string", "path": "string", "secret_key": "string", "user_id": "string", "duration_min": "number" }`
}

func (t *SecretsTool) Execute(ctx context.Context, args map[string]interface{}, conf config.Config) (string, error) {
	action, _ := args["action"].(string)
	if action == "" {
		return "", fmt.Errorf("missing action parameter")
	}

	switch action {
	case "add":
		return addSecret(args)
	case "verify":
		return verifySecret(args)
	case "approve":
		return approveAccess(args)
	case "revoke":
		return revokeAccess(args)
	case "check":
		return checkApproval(args)
	default:
		return "", fmt.Errorf("unknown action: %s", action)
	}
}

func addSecret(args map[string]interface{}) (string, error) {
	agentID, _ := args["agent_id"].(string)
	secretKey, _ := args["secret_key"].(string)
	path, _ := args["path"].(string)

	if agentID == "" || secretKey == "" || path == "" {
		return "", fmt.Errorf("missing required parameters: agent_id, secret_key, path")
	}

	maxAttempts := 3
	if ma, ok := args["max_attempts"].(float64); ok {
		maxAttempts = int(ma)
	}

	if err := secrets.AddSecret(agentID, secretKey, path, maxAttempts); err != nil {
		return "", err
	}

	return fmt.Sprintf("Secret added for agent %s at path %s.", agentID, path), nil
}

func verifySecret(args map[string]interface{}) (string, error) {
	agentID, _ := args["agent_id"].(string)
	secretKey, _ := args["secret_key"].(string)
	path, _ := args["path"].(string)

	if agentID == "" || secretKey == "" || path == "" {
		return "", fmt.Errorf("missing required parameters: agent_id, secret_key, path")
	}

	valid, err := secrets.VerifySecret(agentID, secretKey, path)
	if err != nil {
		return "", err
	}

	if valid {
		return "Secret verified successfully.", nil
	}
	return "", fmt.Errorf("invalid secret key")
}

func approveAccess(args map[string]interface{}) (string, error) {
	agentID, _ := args["agent_id"].(string)
	userID, _ := args["user_id"].(string)
	path, _ := args["path"].(string)

	if agentID == "" || userID == "" || path == "" {
		return "", fmt.Errorf("missing required parameters: agent_id, user_id, path")
	}

	duration := 30 * time.Minute
	if d, ok := args["duration_min"].(float64); ok {
		duration = time.Duration(int(d)) * time.Minute
	}

	if err := secrets.ApproveAccess(agentID, userID, path, duration); err != nil {
		return "", err
	}

	return fmt.Sprintf("Access approved for %s to %s (expires in %v).", userID, path, duration), nil
}

func revokeAccess(args map[string]interface{}) (string, error) {
	agentID, _ := args["agent_id"].(string)
	userID, _ := args["user_id"].(string)
	path, _ := args["path"].(string)

	if agentID == "" || userID == "" || path == "" {
		return "", fmt.Errorf("missing required parameters: agent_id, user_id, path")
	}

	if err := secrets.RevokeApproval(agentID, userID, path); err != nil {
		return "", err
	}

	return fmt.Sprintf("Access revoked for %s to %s.", userID, path), nil
}

func checkApproval(args map[string]interface{}) (string, error) {
	agentID, _ := args["agent_id"].(string)
	userID, _ := args["user_id"].(string)
	path, _ := args["path"].(string)

	if agentID == "" || userID == "" || path == "" {
		return "", fmt.Errorf("missing required parameters: agent_id, user_id, path")
	}

	approved := secrets.CheckApproval(agentID, userID, path)

	if approved {
		return "Access is approved.", nil
	}
	return "No active approval for this access.", nil
}
