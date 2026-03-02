package memory

import (
	"github.com/allataetm-svg/goclaw/internal/provider"
)

type Memory struct {
	ephemeral *Ephemeral
	session   *SessionManager
	userStore *UserMemoryStore
	knowledge *KnowledgeStore
	agentID   string
}

func NewMemory(agentID, systemPrompt string, maxTokens int) *Memory {
	return &Memory{
		ephemeral: NewEphemeral(systemPrompt, maxTokens),
		session:   NewSessionManager(),
		userStore: NewUserMemoryStore(agentID),
		knowledge: NewKnowledgeStore(agentID),
		agentID:   agentID,
	}
}

func (m *Memory) Initialize() error {
	if err := m.userStore.Load(); err != nil {
		return err
	}
	if err := m.knowledge.Load(); err != nil {
		return err
	}
	return nil
}

func (m *Memory) SetSystemPrompt(prompt string) {
	m.ephemeral.SetSystemPrompt(prompt)
}

func (m *Memory) AddUserMessage(content string) {
	m.ephemeral.AddMessage(provider.ChatMessage{
		Role:    "user",
		Content: content,
	})
}

func (m *Memory) AddAssistantMessage(content string) {
	m.ephemeral.AddMessage(provider.ChatMessage{
		Role:    "assistant",
		Content: content,
	})
}

func (m *Memory) GetContext() []provider.ChatMessage {
	return m.ephemeral.GetTrimmedMessages()
}

func (m *Memory) GetFullContext() []provider.ChatMessage {
	return m.ephemeral.GetMessages()
}

func (m *Memory) ClearContext() {
	m.ephemeral.Clear()
}

func (m *Memory) StartSession(task string) {
	m.session.StartSession(task)
}

func (m *Memory) SetSessionGoal(goal string) {
	m.session.SetGoal(goal)
}

func (m *Memory) AddSessionEntity(key, value string) {
	m.session.AddEntity(key, value)
}

func (m *Memory) AddSessionStep(action, result string, success bool) {
	m.session.AddStep(action, result, success)
}

func (m *Memory) EndSession() error {
	if m.session.GetActiveSession() != nil {
		return m.session.CompleteSession()
	}
	m.session.EndSession()
	return nil
}

func (m *Memory) GetSessionState() interface{} {
	return m.session.GetActiveSession()
}

func (m *Memory) StoreUserMemory(memType MemoryType, key, value, source string, tags []string) error {
	mem := UserMemory{
		Type:   memType,
		Key:    key,
		Value:  value,
		Source: source,
		Tags:   tags,
	}
	return m.userStore.Store(mem)
}

func (m *Memory) GetUserMemory(id string) (*UserMemory, error) {
	return m.userStore.Get(id)
}

func (m *Memory) GetUserMemoryByKey(key string) (*UserMemory, error) {
	return m.userStore.GetByKey(key)
}

func (m *Memory) ListUserMemories() []UserMemory {
	return m.userStore.List()
}

func (m *Memory) SearchUserMemory(query string) []UserMemory {
	return m.userStore.Search(query)
}

func (m *Memory) DeleteUserMemory(id string) error {
	return m.userStore.Delete(id)
}

func (m *Memory) UpdateUserMemory(id, value string) error {
	return m.userStore.Update(id, value)
}

func (m *Memory) AddKnowledgeDocument(content, source, url string, tags []string) error {
	doc := Document{
		Content: content,
		Source:  source,
		URL:     url,
		Tags:    tags,
	}
	return m.knowledge.AddDocument(doc)
}

func (m *Memory) AddKnowledgeDocumentFromFile(path string) error {
	return m.knowledge.AddDocumentFromFile(path)
}

func (m *Memory) SearchKnowledge(query string, limit int) []Document {
	return m.knowledge.Search(query, limit)
}

func (m *Memory) ListKnowledgeDocuments() []Document {
	return m.knowledge.List()
}

func (m *Memory) GetKnowledgeDocument(id string) (*Document, error) {
	return m.knowledge.Get(id)
}

func (m *Memory) DeleteKnowledgeDocument(id string) error {
	return m.knowledge.Delete(id)
}

func (m *Memory) CountUserMemories() int {
	return m.userStore.Count()
}

func (m *Memory) CountKnowledgeDocuments() int {
	return m.knowledge.Count()
}

func (m *Memory) EstimateContextTokens() int {
	return m.ephemeral.EstimateTokens()
}

type MemoryStats struct {
	ContextMessages int
	ContextTokens   int
	UserMemories    int
	KnowledgeDocs   int
	ActiveSession   bool
}

func (m *Memory) GetStats() MemoryStats {
	return MemoryStats{
		ContextMessages: m.ephemeral.CountMessages(),
		ContextTokens:   m.ephemeral.EstimateTokens(),
		UserMemories:    m.userStore.Count(),
		KnowledgeDocs:   m.knowledge.Count(),
		ActiveSession:   m.session.GetActiveSession() != nil,
	}
}
