package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/allataetm-svg/goclaw/internal/config"
)

type MemoryType string

const (
	MemoryTypePreference MemoryType = "preference"
	MemoryTypeFact       MemoryType = "fact"
	MemoryTypeContext    MemoryType = "context"
	MemoryTypeCustom     MemoryType = "custom"
)

type UserMemory struct {
	ID        string                 `json:"id"`
	Type      MemoryType             `json:"type"`
	Key       string                 `json:"key"`
	Value     string                 `json:"value"`
	Source    string                 `json:"source,omitempty"`
	Tags      []string               `json:"tags,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

type UserMemoryStore struct {
	memories []UserMemory
	filePath string
}

func NewUserMemoryStore(agentID string) *UserMemoryStore {
	dir := filepath.Join(config.GetConfigDir(), "memory", "longterm")
	return &UserMemoryStore{
		filePath: filepath.Join(dir, agentID+".json"),
	}
}

func (ums *UserMemoryStore) Load() error {
	data, err := os.ReadFile(ums.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			ums.memories = []UserMemory{}
			return nil
		}
		return fmt.Errorf("failed to read memory store: %w", err)
	}
	return json.Unmarshal(data, &ums.memories)
}

func (ums *UserMemoryStore) Save() error {
	dir := filepath.Dir(ums.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create memory dir: %w", err)
	}
	data, err := json.MarshalIndent(ums.memories, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal memories: %w", err)
	}
	if err := os.WriteFile(ums.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write memory store: %w", err)
	}
	return nil
}

func (ums *UserMemoryStore) Store(mem UserMemory) error {
	mem.CreatedAt = time.Now()
	mem.UpdatedAt = time.Now()
	if mem.ID == "" {
		mem.ID = fmt.Sprintf("mem_%d", time.Now().UnixNano())
	}
	ums.memories = append(ums.memories, mem)
	return ums.Save()
}

func (ums *UserMemoryStore) Update(id string, value string) error {
	for i := range ums.memories {
		if ums.memories[i].ID == id {
			ums.memories[i].Value = value
			ums.memories[i].UpdatedAt = time.Now()
			return ums.Save()
		}
	}
	return fmt.Errorf("memory not found: %s", id)
}

func (ums *UserMemoryStore) Delete(id string) error {
	for i := range ums.memories {
		if ums.memories[i].ID == id {
			ums.memories = append(ums.memories[:i], ums.memories[i+1:]...)
			return ums.Save()
		}
	}
	return fmt.Errorf("memory not found: %s", id)
}

func (ums *UserMemoryStore) Get(id string) (*UserMemory, error) {
	for i := range ums.memories {
		if ums.memories[i].ID == id {
			return &ums.memories[i], nil
		}
	}
	return nil, fmt.Errorf("memory not found: %s", id)
}

func (ums *UserMemoryStore) GetByKey(key string) (*UserMemory, error) {
	for i := range ums.memories {
		if ums.memories[i].Key == key {
			return &ums.memories[i], nil
		}
	}
	return nil, fmt.Errorf("memory not found for key: %s", key)
}

func (ums *UserMemoryStore) List() []UserMemory {
	sort.Slice(ums.memories, func(i, j int) bool {
		return ums.memories[i].UpdatedAt.After(ums.memories[j].UpdatedAt)
	})
	result := make([]UserMemory, len(ums.memories))
	copy(result, ums.memories)
	return result
}

func (ums *UserMemoryStore) ListByType(memType MemoryType) []UserMemory {
	var result []UserMemory
	for _, mem := range ums.memories {
		if mem.Type == memType {
			result = append(result, mem)
		}
	}
	return result
}

func (ums *UserMemoryStore) Search(query string) []UserMemory {
	query = toLower(query)
	var result []UserMemory
	for _, mem := range ums.memories {
		if contains(toLower(mem.Key), query) || contains(toLower(mem.Value), query) {
			result = append(result, mem)
		}
	}
	return result
}

func (ums *UserMemoryStore) Count() int {
	return len(ums.memories)
}

func (ums *UserMemoryStore) Clear() error {
	ums.memories = []UserMemory{}
	return ums.Save()
}

func toLower(s string) string {
	if s == "" {
		return ""
	}
	// Simple lowercase - avoiding unicode complications
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		result[i] = c
	}
	return string(result)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
