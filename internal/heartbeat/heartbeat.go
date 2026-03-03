package heartbeat

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/allataetm-svg/goclaw/internal/config"
	"github.com/allataetm-svg/goclaw/internal/provider"
)

type HeartbeatConfig struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	AgentID     string            `json:"agent_id"`
	ChannelID   string            `json:"channel_id"`
	IntervalMin int               `json:"interval_min"`
	Prompt      string            `json:"prompt"`
	Enabled     bool              `json:"enabled"`
	LastRun     *time.Time        `json:"last_run,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

var (
	heartbeats     = make(map[string]*HeartbeatConfig)
	heartbeatsMu   sync.RWMutex
	runningCtx     context.Context
	cancelFunc     context.CancelFunc
	heartbeatCbs   = make(map[string]func(ctx context.Context, result HeartbeatResult))
	heartbeatCbsMu sync.RWMutex
)

type HeartbeatResult struct {
	Output   string        `json:"output"`
	Duration time.Duration `json:"duration"`
	Error    string        `json:"error,omitempty"`
}

type HeartbeatCallback func(ctx context.Context, result HeartbeatResult)

func GetHeartbeatsDir() string {
	return filepath.Join(config.GetConfigDir(), "heartbeat")
}

func LoadHeartbeats() error {
	dir := GetHeartbeatsDir()
	data, err := os.ReadFile(filepath.Join(dir, "heartbeats.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var loaded []HeartbeatConfig
	if err := json.Unmarshal(data, &loaded); err != nil {
		return err
	}
	heartbeatsMu.Lock()
	defer heartbeatsMu.Unlock()
	for i := range loaded {
		heartbeats[loaded[i].ID] = &loaded[i]
	}
	return nil
}

func SaveHeartbeats() error {
	dir := GetHeartbeatsDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	heartbeatsMu.RLock()
	defer heartbeatsMu.RUnlock()
	all := make([]HeartbeatConfig, 0, len(heartbeats))
	for _, h := range heartbeats {
		all = append(all, *h)
	}
	data, err := json.MarshalIndent(all, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "heartbeats.json"), data, 0644)
}

func ListHeartbeats() []HeartbeatConfig {
	heartbeatsMu.RLock()
	defer heartbeatsMu.RUnlock()
	all := make([]HeartbeatConfig, 0, len(heartbeats))
	for _, h := range heartbeats {
		all = append(all, *h)
	}
	return all
}

func AddHeartbeat(h HeartbeatConfig) error {
	if h.IntervalMin < 5 {
		return fmt.Errorf("minimum interval is 5 minutes")
	}
	heartbeatsMu.Lock()
	defer heartbeatsMu.Unlock()
	if _, exists := heartbeats[h.ID]; exists {
		return fmt.Errorf("heartbeat with id %s already exists", h.ID)
	}
	h.Enabled = true
	heartbeats[h.ID] = &h
	return SaveHeartbeats()
}

func RemoveHeartbeat(id string) error {
	heartbeatsMu.Lock()
	defer heartbeatsMu.Unlock()
	if _, exists := heartbeats[id]; !exists {
		return fmt.Errorf("heartbeat not found: %s", id)
	}
	delete(heartbeats, id)
	return SaveHeartbeats()
}

func EnableHeartbeat(id string) error {
	heartbeatsMu.Lock()
	defer heartbeatsMu.Unlock()
	if h, exists := heartbeats[id]; exists {
		h.Enabled = true
		return SaveHeartbeats()
	}
	return fmt.Errorf("heartbeat not found: %s", id)
}

func DisableHeartbeat(id string) error {
	heartbeatsMu.Lock()
	defer heartbeatsMu.Unlock()
	if h, exists := heartbeats[id]; exists {
		h.Enabled = false
		return SaveHeartbeats()
	}
	return fmt.Errorf("heartbeat not found: %s", id)
}

func RegisterCallback(id string, cb HeartbeatCallback) {
	heartbeatCbsMu.Lock()
	defer heartbeatCbsMu.Unlock()
	heartbeatCbs[id] = cb
}

func Start(ctx context.Context) error {
	if cancelFunc != nil {
		return nil
	}
	runningCtx, cancelFunc = context.WithCancel(ctx)
	go runHeartbeats(runningCtx)
	return nil
}

func Stop() {
	if cancelFunc != nil {
		cancelFunc()
		cancelFunc = nil
	}
}

func runHeartbeats(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			checkAndRunHeartbeats(ctx)
		}
	}
}

func checkAndRunHeartbeats(ctx context.Context) {
	heartbeatsMu.RLock()
	defer heartbeatsMu.RUnlock()
	now := time.Now()

	for _, h := range heartbeats {
		if !h.Enabled {
			continue
		}

		shouldRun := false
		if h.LastRun == nil {
			shouldRun = true
		} else {
			nextRun := h.LastRun.Add(time.Duration(h.IntervalMin) * time.Minute)
			if now.After(nextRun) || now.Equal(nextRun) {
				shouldRun = true
			}
		}

		if shouldRun {
			go runHeartbeat(ctx, h)
		}
	}
}

func runHeartbeat(ctx context.Context, h *HeartbeatConfig) {
	startTime := time.Now()

	result := executeHeartbeatPrompt(ctx, h)

	duration := time.Since(startTime)
	h.LastRun = &startTime
	SaveHeartbeats()

	if result.Error != "" {
		fmt.Printf("[Heartbeat] Error running %s: %s\n", h.Name, result.Error)
		return
	}

	fmt.Printf("[Heartbeat] Completed %s in %v\n", h.Name, duration)

	heartbeatCbsMu.RLock()
	if cb, ok := heartbeatCbs[h.ID]; ok {
		go cb(ctx, result)
	}
	heartbeatCbsMu.RUnlock()
}

func executeHeartbeatPrompt(ctx context.Context, h *HeartbeatConfig) HeartbeatResult {
	conf, err := config.Load()
	if err != nil {
		return HeartbeatResult{Error: fmt.Sprintf("failed to load config: %v", err)}
	}

	ws, prov, modName, err := loadAgentForHeartbeat(conf, h.AgentID)
	if err != nil {
		return HeartbeatResult{Error: fmt.Sprintf("failed to load agent: %v", err)}
	}

	messages := []provider.ChatMessage{
		{Role: "system", Content: ws.systemPrompt},
		{Role: "user", Content: h.Prompt},
	}

	resp, err := prov.Query(ctx, modName, messages)
	if err != nil {
		return HeartbeatResult{Error: err.Error()}
	}

	return HeartbeatResult{Output: resp, Duration: time.Since(time.Now())}
}

type wsConfig struct {
	systemPrompt string
}

func loadAgentForHeartbeat(conf config.Config, agentID string) (wsConfig, provider.LLMProvider, string, error) {
	dir := filepath.Join(config.GetConfigDir(), "agents", agentID)
	configData, err := os.ReadFile(filepath.Join(dir, "config.json"))
	if err != nil {
		return wsConfig{}, nil, "", err
	}

	var agentCfg struct {
		Model string `json:"model"`
	}
	json.Unmarshal(configData, &agentCfg)

	provID, modName := parseModel(agentCfg.Model)

	var pc config.ProviderConfig
	for _, p := range conf.Providers {
		if p.ID == provID {
			pc = p
			break
		}
	}

	prov := provider.MakeProvider(pc)

	soulData, _ := os.ReadFile(filepath.Join(dir, "SOUL.md"))
	agentData, _ := os.ReadFile(filepath.Join(dir, "AGENT.md"))
	instrData, _ := os.ReadFile(filepath.Join(dir, "INSTRUCTIONS.md"))

	ws := wsConfig{
		systemPrompt: fmt.Sprintf("You are %s.\n\n%s\n\n%s", soulData, agentData, instrData),
	}

	return ws, prov, modName, nil
}

func parseModel(model string) (string, string) {
	for i := 0; i < len(model); i++ {
		if model[i] == ':' {
			return model[:i], model[i+1:]
		}
	}
	return "openai", model
}
