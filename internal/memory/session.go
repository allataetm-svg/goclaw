package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/allataetm-svg/goclaw/internal/config"
)

type SessionID string

type SessionState struct {
	ID          SessionID         `json:"id"`
	Task        string            `json:"task"`
	Goal        string            `json:"goal,omitempty"`
	Entities    map[string]string `json:"entities,omitempty"`
	Steps       []SessionStep     `json:"steps,omitempty"`
	StartedAt   time.Time         `json:"started_at"`
	LastUpdated time.Time         `json:"last_updated"`
	Completed   bool              `json:"completed"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type SessionStep struct {
	Timestamp time.Time `json:"timestamp"`
	Action    string    `json:"action"`
	Result    string    `json:"result,omitempty"`
	Success   bool      `json:"success"`
}

type SessionManager struct {
	activeSession *SessionState
	sessionsDir   string
}

func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessionsDir: filepath.Join(config.GetConfigDir(), "memory", "sessions"),
	}
}

func (sm *SessionManager) StartSession(task string) SessionID {
	id := SessionID(fmt.Sprintf("session_%d", time.Now().UnixNano()))
	sm.activeSession = &SessionState{
		ID:          id,
		Task:        task,
		Entities:    make(map[string]string),
		Steps:       []SessionStep{},
		StartedAt:   time.Now(),
		LastUpdated: time.Now(),
		Metadata:    make(map[string]string),
	}
	return id
}

func (sm *SessionManager) GetActiveSession() *SessionState {
	return sm.activeSession
}

func (sm *SessionManager) SetGoal(goal string) {
	if sm.activeSession != nil {
		sm.activeSession.Goal = goal
		sm.activeSession.LastUpdated = time.Now()
	}
}

func (sm *SessionManager) AddEntity(key, value string) {
	if sm.activeSession != nil {
		sm.activeSession.Entities[key] = value
		sm.activeSession.LastUpdated = time.Now()
	}
}

func (sm *SessionManager) AddStep(action, result string, success bool) {
	if sm.activeSession != nil {
		sm.activeSession.Steps = append(sm.activeSession.Steps, SessionStep{
			Timestamp: time.Now(),
			Action:    action,
			Result:    result,
			Success:   success,
		})
		sm.activeSession.LastUpdated = time.Now()
	}
}

func (sm *SessionManager) CompleteSession() error {
	if sm.activeSession == nil {
		return fmt.Errorf("no active session")
	}
	sm.activeSession.Completed = true
	sm.activeSession.LastUpdated = time.Now()
	return sm.saveSession(sm.activeSession)
}

func (sm *SessionManager) EndSession() {
	sm.activeSession = nil
}

func (sm *SessionManager) saveSession(s *SessionState) error {
	if err := os.MkdirAll(sm.sessionsDir, 0755); err != nil {
		return fmt.Errorf("failed to create sessions dir: %w", err)
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}
	filename := fmt.Sprintf("%s.json", s.ID)
	if err := os.WriteFile(filepath.Join(sm.sessionsDir, filename), data, 0644); err != nil {
		return fmt.Errorf("failed to write session: %w", err)
	}
	return nil
}

func (sm *SessionManager) LoadSession(id SessionID) (*SessionState, error) {
	data, err := os.ReadFile(filepath.Join(sm.sessionsDir, fmt.Sprintf("%s.json", id)))
	if err != nil {
		return nil, fmt.Errorf("failed to read session: %w", err)
	}
	var s SessionState
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("failed to parse session: %w", err)
	}
	return &s, nil
}

func (sm *SessionManager) ListSessions() ([]SessionState, error) {
	entries, err := os.ReadDir(sm.sessionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []SessionState{}, nil
		}
		return nil, err
	}
	var sessions []SessionState
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(sm.sessionsDir, e.Name()))
		if err != nil {
			continue
		}
		var s SessionState
		if err := json.Unmarshal(data, &s); err != nil {
			continue
		}
		sessions = append(sessions, s)
	}
	return sessions, nil
}

func (sm *SessionManager) GetLastSession() (*SessionState, error) {
	sessions, err := sm.ListSessions()
	if err != nil {
		return nil, err
	}
	if len(sessions) == 0 {
		return nil, fmt.Errorf("no sessions found")
	}
	// Sort by LastUpdated descending
	for i := 0; i < len(sessions)-1; i++ {
		for j := i + 1; j < len(sessions); j++ {
			if sessions[j].LastUpdated.After(sessions[i].LastUpdated) {
				sessions[i], sessions[j] = sessions[j], sessions[i]
			}
		}
	}
	return &sessions[0], nil
}
