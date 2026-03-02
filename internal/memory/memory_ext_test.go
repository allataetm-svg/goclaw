package memory

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/allataetm-svg/goclaw/internal/config"
)

func TestUserMemoryStore(t *testing.T) {
	agentID := "test_agent_user"
	store := NewUserMemoryStore(agentID)

	err := store.Store(UserMemory{
		Type:  MemoryTypePreference,
		Key:   "prefers_dark_mode",
		Value: "true",
	})
	if err != nil {
		t.Errorf("Failed to store memory: %v", err)
	}

	err = store.Store(UserMemory{
		Type:  MemoryTypeFact,
		Key:   "favorite_language",
		Value: "Python",
	})
	if err != nil {
		t.Errorf("Failed to store memory: %v", err)
	}

	mems := store.List()
	if len(mems) != 2 {
		t.Errorf("Expected 2 memories, got %d", len(mems))
	}

	results := store.Search("dark")
	if len(results) != 1 {
		t.Errorf("Expected 1 result for 'dark', got %d", len(results))
	}

	err = store.Delete(mems[0].ID)
	if err != nil {
		t.Errorf("Failed to delete memory: %v", err)
	}

	mems = store.List()
	if len(mems) != 1 {
		t.Errorf("Expected 1 memory after delete, got %d", len(mems))
	}

	os.RemoveAll(filepath.Join(config.GetConfigDir(), "memory", "longterm"))
}

func TestKnowledgeStore(t *testing.T) {
	agentID := "test_agent_knowledge"
	store := NewKnowledgeStore(agentID)

	err := store.AddDocument(Document{
		Content: "Python best practice: Use virtual environments",
		Source:  "manual",
		Tags:    []string{"python", "best-practice"},
	})
	if err != nil {
		t.Errorf("Failed to add document: %v", err)
	}

	err = store.AddDocument(Document{
		Content: "Go best practice: Use go modules",
		Source:  "manual",
		Tags:    []string{"go", "best-practice"},
	})
	if err != nil {
		t.Errorf("Failed to add document: %v", err)
	}

	docs := store.Search("python", 5)
	if len(docs) != 1 {
		t.Errorf("Expected 1 document for 'python', got %d", len(docs))
	}

	allDocs := store.List()
	if len(allDocs) != 2 {
		t.Errorf("Expected 2 documents, got %d", len(allDocs))
	}

	err = store.Delete(allDocs[0].ID)
	if err != nil {
		t.Errorf("Failed to delete document: %v", err)
	}

	docs = store.List()
	if len(docs) != 1 {
		t.Errorf("Expected 1 document after delete, got %d", len(docs))
	}

	os.RemoveAll(filepath.Join(config.GetConfigDir(), "memory", "longterm", "knowledge"))
}

func TestSessionManager(t *testing.T) {
	sm := NewSessionManager()

	id := sm.StartSession("Test task")
	if id == "" {
		t.Error("Expected non-empty session ID")
	}

	sm.SetGoal("Complete testing")
	sm.AddEntity("test_key", "test_value")
	sm.AddStep("step1", "Testing", true)

	state := sm.GetActiveSession()
	if state == nil {
		t.Error("Expected active session")
	}

	if state.Goal != "Complete testing" {
		t.Errorf("Expected goal 'Complete testing', got '%s'", state.Goal)
	}

	if len(state.Entities) != 1 {
		t.Errorf("Expected 1 entity, got %d", len(state.Entities))
	}

	if len(state.Steps) != 1 {
		t.Errorf("Expected 1 step, got %d", len(state.Steps))
	}

	err := sm.CompleteSession()
	if err != nil {
		t.Errorf("Failed to complete session: %v", err)
	}

	sessions, err := sm.ListSessions()
	if err != nil {
		t.Errorf("Failed to list sessions: %v", err)
	}

	if len(sessions) != 1 {
		t.Errorf("Expected 1 session, got %d", len(sessions))
	}

	os.RemoveAll(filepath.Join(config.GetConfigDir(), "memory", "sessions"))
}
