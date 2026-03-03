package agent

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/allataetm-svg/goclaw/internal/config"
	"github.com/allataetm-svg/goclaw/internal/provider"
)

func init() {
	RegisterTool(&DelegateTaskTool{})
	RegisterTool(&ReadFileTool{})
	RegisterTool(&WriteFileTool{})
	RegisterTool(&ShellTool{})
	RegisterTool(&WebSearchTool{})
	RegisterTool(&SchedulerTool{})
	RegisterTool(&HeartbeatTool{})
	RegisterTool(&SkillsTool{})
	RegisterTool(&WebFetchTool{})
	RegisterTool(&SessionsTool{})
	RegisterTool(&SecretsTool{})
}

// Tool defines the interface for agent capabilities
type Tool interface {
	Name() string
	Description() string
	Execute(ctx context.Context, args map[string]interface{}, conf config.Config) (string, error)
}

// DelegateTaskTool allows an agent to delegate a task to a subagent
type DelegateTaskTool struct{}

func (t *DelegateTaskTool) Name() string { return "delegate_task" }
func (t *DelegateTaskTool) Description() string {
	return "Delegates a specific task to a subagent. Args: { \"subagent_id\": \"string\", \"task\": \"string\" }"
}

func (t *DelegateTaskTool) Execute(ctx context.Context, args map[string]interface{}, conf config.Config) (string, error) {
	subagentID, ok := args["subagent_id"].(string)
	if !ok {
		return "", fmt.Errorf("missing subagent_id")
	}

	task, ok := args["task"].(string)
	if !ok {
		// Fallback to "prompt" in case LLM gets it slightly wrong
		task, ok = args["prompt"].(string)
		if !ok {
			return "", fmt.Errorf("missing task description")
		}
	}

	// 1. Load subagent
	ws, prov, mod, err := LoadAgent(conf, subagentID)
	if err != nil {
		return "", fmt.Errorf("failed to load subagent %s: %w", subagentID, err)
	}

	// 2. Prepare subagent prompt
	history := []provider.ChatMessage{
		{Role: "system", Content: BuildSystemPrompt(ws)},
		{Role: "user", Content: task},
	}

	// 3. Query subagent (stateless)
	resp, err := prov.Query(ctx, mod, history)
	if err != nil {
		return "", fmt.Errorf("subagent query failed: %w", err)
	}

	return fmt.Sprintf("Report from subagent [%s]:\n\n%s", subagentID, resp), nil
}

// ReadFileTool reads a file from the workspace
type ReadFileTool struct{}

func (t *ReadFileTool) Name() string { return "read_file" }
func (t *ReadFileTool) Description() string {
	return "Reads content from a file. Args: { \"path\": \"string\" }"
}
func (t *ReadFileTool) Execute(ctx context.Context, args map[string]interface{}, _ config.Config) (string, error) {
	path, _ := args["path"].(string)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// WriteFileTool writes a file to the workspace
type WriteFileTool struct{}

func (t *WriteFileTool) Name() string { return "write_file" }
func (t *WriteFileTool) Description() string {
	return "Writes content to a file. Args: { \"path\": \"string\", \"content\": \"string\" }"
}
func (t *WriteFileTool) Execute(ctx context.Context, args map[string]interface{}, _ config.Config) (string, error) {
	path, _ := args["path"].(string)
	content, _ := args["content"].(string)
	err := os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		return "", err
	}
	return "File written successfully.", nil
}

// ShellTool executes a shell command
type ShellTool struct{}

func (t *ShellTool) Name() string { return "shell" }
func (t *ShellTool) Description() string {
	return "Executes a shell command. Args: { \"command\": \"string\" }"
}
func (t *ShellTool) Execute(ctx context.Context, args map[string]interface{}, _ config.Config) (string, error) {
	command, _ := args["command"].(string)
	if command == "" {
		return "", fmt.Errorf("missing command")
	}

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "powershell", "-Command", command)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", command)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("command failed: %w", err)
	}
	return string(output), nil
}
