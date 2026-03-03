package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/allataetm-svg/goclaw/internal/config"
	"github.com/allataetm-svg/goclaw/internal/scheduler"
)

type SchedulerTool struct{}

func (t *SchedulerTool) Name() string { return "scheduler" }

func (t *SchedulerTool) Description() string {
	return `Manages scheduled tasks. Args: { "action": "string (add|list|remove|enable|disable)", "name": "string", "type": "string (cron|at_seconds|every_seconds)", "interval_sec": "number", "cron_expr": "string", "at_seconds": "number", "command": "string", "task_id": "string" }`
}

func (t *SchedulerTool) Execute(ctx context.Context, args map[string]interface{}, conf config.Config) (string, error) {
	action, _ := args["action"].(string)
	if action == "" {
		return "", fmt.Errorf("missing action parameter")
	}

	switch action {
	case "list":
		return listScheduledTasks()
	case "add":
		return addScheduledTask(args)
	case "remove":
		taskID, _ := args["task_id"].(string)
		if taskID == "" {
			return "", fmt.Errorf("missing task_id")
		}
		if err := scheduler.RemoveTask(taskID); err != nil {
			return "", err
		}
		return "Task removed successfully.", nil
	case "enable":
		taskID, _ := args["task_id"].(string)
		if taskID == "" {
			return "", fmt.Errorf("missing task_id")
		}
		if err := scheduler.EnableTask(taskID); err != nil {
			return "", err
		}
		return "Task enabled successfully.", nil
	case "disable":
		taskID, _ := args["task_id"].(string)
		if taskID == "" {
			return "", fmt.Errorf("missing task_id")
		}
		if err := scheduler.DisableTask(taskID); err != nil {
			return "", err
		}
		return "Task disabled successfully.", nil
	default:
		return "", fmt.Errorf("unknown action: %s", action)
	}
}

func listScheduledTasks() (string, error) {
	tasks := scheduler.ListTasks()
	if len(tasks) == 0 {
		return "No scheduled tasks.", nil
	}

	var sb strings.Builder
	sb.WriteString("## Scheduled Tasks\n\n")

	for _, t := range tasks {
		status := "enabled"
		if !t.Enabled {
			status = "disabled"
		}

		sb.WriteString(fmt.Sprintf("### %s (%s)\n", t.Name, status))
		sb.WriteString(fmt.Sprintf("- ID: %s\n", t.ID))
		sb.WriteString(fmt.Sprintf("- Type: %s\n", t.Type))
		sb.WriteString(fmt.Sprintf("- Command: %s\n", t.Command))

		if t.Type == "every_seconds" {
			sb.WriteString(fmt.Sprintf("- Interval: %d seconds\n", t.IntervalSec))
		} else if t.Type == "cron" {
			sb.WriteString(fmt.Sprintf("- Cron: %s\n", t.CronExpr))
		} else if t.Type == "at_seconds" {
			sb.WriteString(fmt.Sprintf("- At: %d seconds from now\n", t.AtSeconds))
		}

		if t.LastRun != nil {
			sb.WriteString(fmt.Sprintf("- Last Run: %s\n", t.LastRun.Format(time.RFC3339)))
		}
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

func addScheduledTask(args map[string]interface{}) (string, error) {
	name, _ := args["name"].(string)
	taskType, _ := args["type"].(string)
	command, _ := args["command"].(string)

	if name == "" || taskType == "" || command == "" {
		return "", fmt.Errorf("missing required parameters: name, type, command")
	}

	task := scheduler.Task{
		ID:      fmt.Sprintf("task_%d", time.Now().Unix()),
		Name:    name,
		Type:    scheduler.TaskType(taskType),
		Enabled: true,
		Command: command,
	}

	switch taskType {
	case "every_seconds":
		if interval, ok := args["interval_sec"].(float64); ok {
			task.IntervalSec = int(interval)
		}
		if task.IntervalSec == 0 {
			return "", fmt.Errorf("interval_sec is required for every_seconds type")
		}
	case "cron":
		if cron, ok := args["cron_expr"].(string); ok {
			task.CronExpr = cron
		}
		if task.CronExpr == "" {
			return "", fmt.Errorf("cron_expr is required for cron type")
		}
	case "at_seconds":
		if at, ok := args["at_seconds"].(float64); ok {
			task.AtSeconds = int(at)
		}
		if task.AtSeconds == 0 {
			return "", fmt.Errorf("at_seconds is required for at_seconds type")
		}
	}

	if err := scheduler.AddTask(task); err != nil {
		return "", err
	}

	return fmt.Sprintf("Task '%s' added successfully.", name), nil
}
