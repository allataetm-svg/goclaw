package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	"github.com/allataetm-svg/goclaw/internal/config"
)

type TaskType string

const (
	TaskTypeCron     TaskType = "cron"
	TaskTypeAt       TaskType = "at_seconds"
	TaskTypeInterval TaskType = "every_seconds"
)

type Task struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Type        TaskType          `json:"type"`
	Enabled     bool              `json:"enabled"`
	AgentID     string            `json:"agent_id"`
	Command     string            `json:"command"`
	IntervalSec int               `json:"interval_sec,omitempty"`
	CronExpr    string            `json:"cron_expr,omitempty"`
	AtSeconds   int               `json:"at_seconds,omitempty"`
	ChannelID   string            `json:"channel_id,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	LastRun     *time.Time        `json:"last_run,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

var (
	tasks      = make(map[string]*Task)
	tasksMu    sync.RWMutex
	stopChan   = make(chan struct{})
	runningCtx context.Context
	cancelFunc context.CancelFunc
)

func GetTasksDir() string {
	return filepath.Join(config.GetConfigDir(), "scheduler")
}

func LoadTasks() error {
	dir := GetTasksDir()
	data, err := os.ReadFile(filepath.Join(dir, "tasks.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var loaded []Task
	if err := json.Unmarshal(data, &loaded); err != nil {
		return err
	}
	tasksMu.Lock()
	defer tasksMu.Unlock()
	for i := range loaded {
		tasks[loaded[i].ID] = &loaded[i]
	}
	return nil
}

func SaveTasks() error {
	dir := GetTasksDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	tasksMu.RLock()
	defer tasksMu.RUnlock()
	all := make([]Task, 0, len(tasks))
	for _, t := range tasks {
		all = append(all, *t)
	}
	data, err := json.MarshalIndent(all, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "tasks.json"), data, 0644)
}

func ListTasks() []Task {
	tasksMu.RLock()
	defer tasksMu.RUnlock()
	all := make([]Task, 0, len(tasks))
	for _, t := range tasks {
		all = append(all, *t)
	}
	return all
}

func AddTask(t Task) error {
	tasksMu.Lock()
	defer tasksMu.Unlock()
	if _, exists := tasks[t.ID]; exists {
		return fmt.Errorf("task with id %s already exists", t.ID)
	}
	t.CreatedAt = time.Now()
	tasks[t.ID] = &t
	return SaveTasks()
}

func RemoveTask(id string) error {
	tasksMu.Lock()
	defer tasksMu.Unlock()
	if _, exists := tasks[id]; !exists {
		return fmt.Errorf("task not found: %s", id)
	}
	delete(tasks, id)
	return SaveTasks()
}

func EnableTask(id string) error {
	tasksMu.Lock()
	defer tasksMu.Unlock()
	if t, exists := tasks[id]; exists {
		t.Enabled = true
		return SaveTasks()
	}
	return fmt.Errorf("task not found: %s", id)
}

func DisableTask(id string) error {
	tasksMu.Lock()
	defer tasksMu.Unlock()
	if t, exists := tasks[id]; exists {
		t.Enabled = false
		return SaveTasks()
	}
	return fmt.Errorf("task not found: %s", id)
}

func Start(ctx context.Context) error {
	if cancelFunc != nil {
		return nil
	}
	runningCtx, cancelFunc = context.WithCancel(ctx)
	go runScheduler(runningCtx)
	return nil
}

func Stop() {
	if cancelFunc != nil {
		cancelFunc()
		cancelFunc = nil
	}
}

func runScheduler(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			runDueTasks(ctx)
		}
	}
}

func runDueTasks(ctx context.Context) {
	tasksMu.RLock()
	defer tasksMu.RUnlock()
	now := time.Now()

	for _, t := range tasks {
		if !t.Enabled {
			continue
		}

		shouldRun := false

		switch t.Type {
		case TaskTypeInterval:
			if t.LastRun == nil {
				shouldRun = true
			} else {
				nextRun := t.LastRun.Add(time.Duration(t.IntervalSec) * time.Second)
				if now.After(nextRun) || now.Equal(nextRun) {
					shouldRun = true
				}
			}
		case TaskTypeAt:
			if t.LastRun == nil && now.Unix() >= int64(t.AtSeconds) {
				shouldRun = true
			}
		case TaskTypeCron:
			if t.CronExpr != "" && matchesCron(now, t.CronExpr) {
				if t.LastRun == nil || !isSameCronPeriod(now, *t.LastRun, t.CronExpr) {
					shouldRun = true
				}
			}
		}

		if shouldRun {
			executeTask(ctx, t)
		}
	}
}

func executeTask(ctx context.Context, t *Task) {
	now := time.Now()
	t.LastRun = &now
	SaveTasks()

	fmt.Printf("[Scheduler] Running task: %s (%s)\n", t.Name, t.Command)
}

func matchesCron(t time.Time, expr string) bool {
	fields := parseCronExpr(expr)
	if len(fields) != 5 {
		return false
	}

	minute := fmt.Sprintf("%02d", t.Minute())
	hour := fmt.Sprintf("%02d", t.Hour())
	day := fmt.Sprintf("%02d", t.Day())
	month := fmt.Sprintf("%02d", int(t.Month()))
	weekday := fmt.Sprintf("%d", int(t.Weekday()))

	patterns := []string{fields[0], fields[1], fields[2], fields[3], fields[4]}
	values := []string{minute, hour, day, month, weekday}

	for i, pattern := range patterns {
		if pattern == "*" {
			continue
		}
		if !matchesCronField(pattern, values[i]) {
			return false
		}
	}
	return true
}

func parseCronExpr(expr string) []string {
	re := regexp.MustCompile(`\s+`)
	return re.Split(expr, -1)
}

func matchesCronField(pattern, value string) bool {
	if pattern == value {
		return true
	}

	stepMatch := regexp.MustCompile(`^(\*|[\d]+)-([\d]+)/(\d+)$`)
	if sm := stepMatch.FindStringSubmatch(pattern); sm != nil {
		start, end := 0, 59
		if sm[1] != "*" {
			fmt.Sscanf(sm[1], "%d", &start)
		}
		fmt.Sscanf(sm[2], "%d", &end)
		step := 1
		fmt.Sscanf(sm[3], "%d", &step)

		var v int
		fmt.Sscanf(value, "%d", &v)
		if v >= start && v <= end && (v-start)%step == 0 {
			return true
		}
	}

	rangeMatch := regexp.MustCompile(`^(\d+)-(\d+)$`)
	if rm := rangeMatch.FindStringSubmatch(pattern); rm != nil {
		start, end := 0, 0
		fmt.Sscanf(rm[1], "%d", &start)
		fmt.Sscanf(rm[2], "%d", &end)
		var v int
		fmt.Sscanf(value, "%d", &v)
		return v >= start && v <= end
	}

	return false
}

func isSameCronPeriod(t1, t2 time.Time, expr string) bool {
	fields := parseCronExpr(expr)
	if len(fields) != 5 {
		return false
	}

	minuteField := fields[0]
	hourField := fields[1]

	t1Str := fmt.Sprintf("%s%s", hourField, minuteField)
	t2Str := fmt.Sprintf("%02d%02d", t2.Hour(), t2.Minute())

	if minuteField == "*" && hourField == "*" {
		return t1.Hour() == t2.Hour() && t1.Minute() == t2.Minute()
	}

	if minuteField == "*" {
		return t1.Hour() == t2.Hour()
	}

	if hourField == "*" {
		return t1.Minute() == t2.Minute()
	}

	return t1Str == t2Str
}
