package sessions

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/allataetm-svg/goclaw/internal/config"
)

type Session struct {
	ID          string            `json:"id"`
	ChannelID   string            `json:"channel_id"`
	UserID      string            `json:"user_id"`
	AgentID     string            `json:"agent_id"`
	StartedAt   time.Time         `json:"started_at"`
	LastActive  time.Time         `json:"last_active"`
	LastMessage string            `json:"last_message,omitempty"`
	Status      string            `json:"status"` // active, paused, ended
	Metadata    map[string]string `json:"metadata,omitempty"`
}

var (
	activeSessions = make(map[string]*Session)
	sessionsMu     sync.RWMutex
)

func GetSessionsDir() string {
	return filepath.Join(config.GetConfigDir(), "sessions")
}

func LoadSessions() error {
	dir := GetSessionsDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := os.ReadFile(filepath.Join(dir, "sessions.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var loaded []Session
	if err := json.Unmarshal(data, &loaded); err != nil {
		return err
	}

	sessionsMu.Lock()
	defer sessionsMu.Unlock()
	for i := range loaded {
		activeSessions[loaded[i].ID] = &loaded[i]
	}

	return nil
}

func SaveSessions() error {
	dir := GetSessionsDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	sessionsMu.RLock()
	defer sessionsMu.RUnlock()

	all := make([]Session, 0, len(activeSessions))
	for _, s := range activeSessions {
		all = append(all, *s)
	}

	data, err := json.MarshalIndent(all, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(dir, "sessions.json"), data, 0644)
}

func ListActiveSessions() []Session {
	sessionsMu.RLock()
	defer sessionsMu.RUnlock()

	all := make([]Session, 0, len(activeSessions))
	for _, s := range activeSessions {
		if s.Status == "active" {
			all = append(all, *s)
		}
	}
	return all
}

func ListAllSessions() []Session {
	sessionsMu.RLock()
	defer sessionsMu.RUnlock()

	all := make([]Session, 0, len(activeSessions))
	for _, s := range activeSessions {
		all = append(all, *s)
	}
	return all
}

func GetSession(id string) (*Session, bool) {
	sessionsMu.RLock()
	defer sessionsMu.RUnlock()
	s, ok := activeSessions[id]
	return s, ok
}

func CreateSession(channelID, userID, agentID string) (*Session, error) {
	sessionsMu.Lock()
	defer sessionsMu.Unlock()

	session := &Session{
		ID:         fmt.Sprintf("sess_%d", time.Now().Unix()),
		ChannelID:  channelID,
		UserID:     userID,
		AgentID:    agentID,
		StartedAt:  time.Now(),
		LastActive: time.Now(),
		Status:     "active",
		Metadata:   make(map[string]string),
	}

	activeSessions[session.ID] = session
	SaveSessions()

	return session, nil
}

func UpdateSessionActivity(id string) error {
	sessionsMu.Lock()
	defer sessionsMu.Unlock()

	if s, ok := activeSessions[id]; ok {
		s.LastActive = time.Now()
		return SaveSessions()
	}
	return fmt.Errorf("session not found: %s", id)
}

func EndSession(id string) error {
	sessionsMu.Lock()
	defer sessionsMu.Unlock()

	if s, ok := activeSessions[id]; ok {
		s.Status = "ended"
		return SaveSessions()
	}
	return fmt.Errorf("session not found: %s", id)
}

func SendMessageToSession(sessionID, message string) error {
	sessionsMu.RLock()
	defer sessionsMu.RUnlock()

	if s, ok := activeSessions[sessionID]; ok {
		if s.Metadata == nil {
			s.Metadata = make(map[string]string)
		}
		s.LastMessage = message
		s.LastActive = time.Now()
		return SaveSessions()
	}
	return fmt.Errorf("session not found: %s", sessionID)
}

func GetChannelSessions(channelID string) []Session {
	sessionsMu.RLock()
	defer sessionsMu.RUnlock()

	var result []Session
	for _, s := range activeSessions {
		if s.ChannelID == channelID && s.Status == "active" {
			result = append(result, *s)
		}
	}
	return result
}

func GetUserSessions(userID string) []Session {
	sessionsMu.RLock()
	defer sessionsMu.RUnlock()

	var result []Session
	for _, s := range activeSessions {
		if s.UserID == userID {
			result = append(result, *s)
		}
	}
	return result
}
