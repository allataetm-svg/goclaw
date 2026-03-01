package provider

import (
	"encoding/json"
	"testing"

	"github.com/openclaw-alternative/goclaw/internal/config"
)

func TestMakeProvider_Ollama(t *testing.T) {
	pc := config.ProviderConfig{ID: "ollama"}
	p := MakeProvider(pc)
	if p.ID() != "ollama" {
		t.Errorf("Expected ID 'ollama', got '%s'", p.ID())
	}
	if p.Name() != "Ollama" {
		t.Errorf("Expected Name 'Ollama', got '%s'", p.Name())
	}
}

func TestMakeProvider_OllamaDefaultURL(t *testing.T) {
	pc := config.ProviderConfig{ID: "ollama"}
	p := MakeProvider(pc)
	op, ok := p.(*OllamaProvider)
	if !ok {
		t.Fatal("Expected OllamaProvider type")
	}
	if op.URL != "http://127.0.0.1:11434" {
		t.Errorf("Expected default URL, got '%s'", op.URL)
	}
}

func TestMakeProvider_OllamaCustomURL(t *testing.T) {
	pc := config.ProviderConfig{ID: "ollama", BaseURL: "http://myhost:11434"}
	p := MakeProvider(pc)
	op, ok := p.(*OllamaProvider)
	if !ok {
		t.Fatal("Expected OllamaProvider type")
	}
	if op.URL != "http://myhost:11434" {
		t.Errorf("Expected custom URL, got '%s'", op.URL)
	}
}

func TestMakeProvider_OpenRouter(t *testing.T) {
	pc := config.ProviderConfig{ID: "openrouter", APIKey: "test-key"}
	p := MakeProvider(pc)
	if p.ID() != "openrouter" {
		t.Errorf("Expected ID 'openrouter', got '%s'", p.ID())
	}
	op, ok := p.(*OpenAICompatProvider)
	if !ok {
		t.Fatal("Expected OpenAICompatProvider type")
	}
	if op.BaseURL != "https://openrouter.ai/api/v1" {
		t.Errorf("Expected OpenRouter URL, got '%s'", op.BaseURL)
	}
	if op.APIKey != "test-key" {
		t.Errorf("Expected API key 'test-key', got '%s'", op.APIKey)
	}
}

func TestMakeProvider_OpenAI(t *testing.T) {
	pc := config.ProviderConfig{ID: "openai", APIKey: "sk-123"}
	p := MakeProvider(pc)
	if p.ID() != "openai" {
		t.Errorf("Expected ID 'openai', got '%s'", p.ID())
	}
	if p.Name() != "OpenAI" {
		t.Errorf("Expected Name 'OpenAI', got '%s'", p.Name())
	}
}

func TestMakeProvider_Anthropic(t *testing.T) {
	pc := config.ProviderConfig{ID: "anthropic", APIKey: "sk-ant-123"}
	p := MakeProvider(pc)
	if p.ID() != "anthropic" {
		t.Errorf("Expected ID 'anthropic', got '%s'", p.ID())
	}
	_, ok := p.(*AnthropicProvider)
	if !ok {
		t.Fatal("Expected AnthropicProvider type")
	}
}

func TestMakeProvider_Gemini(t *testing.T) {
	pc := config.ProviderConfig{ID: "gemini", APIKey: "aig-123"}
	p := MakeProvider(pc)
	if p.ID() != "gemini" {
		t.Errorf("Expected ID 'gemini', got '%s'", p.ID())
	}
	_, ok := p.(*GeminiProvider)
	if !ok {
		t.Fatal("Expected GeminiProvider type")
	}
}

func TestMakeProvider_CustomOpenAI(t *testing.T) {
	pc := config.ProviderConfig{ID: "custom_openai", BaseURL: "http://localhost:8000/v1", APIKey: "test"}
	p := MakeProvider(pc)
	op, ok := p.(*OpenAICompatProvider)
	if !ok {
		t.Fatal("Expected OpenAICompatProvider type")
	}
	if op.BaseURL != "http://localhost:8000/v1" {
		t.Errorf("Expected custom URL, got '%s'", op.BaseURL)
	}
}

func TestMakeProvider_CustomOpenAIDefaultURL(t *testing.T) {
	pc := config.ProviderConfig{ID: "custom_openai"}
	p := MakeProvider(pc)
	op, ok := p.(*OpenAICompatProvider)
	if !ok {
		t.Fatal("Expected OpenAICompatProvider type")
	}
	if op.BaseURL != "http://localhost:4000/v1" {
		t.Errorf("Expected default custom URL, got '%s'", op.BaseURL)
	}
}

func TestAnthropicFetchModels(t *testing.T) {
	p := &AnthropicProvider{APIKey: "test"}
	models, err := p.FetchModels()
	if err != nil {
		t.Fatalf("FetchModels should not error: %v", err)
	}
	if len(models) == 0 {
		t.Error("Expected some hardcoded models")
	}
}

func TestChatMessageSerialization(t *testing.T) {
	msg := ChatMessage{Role: "user", Content: "Hello"}
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var loaded ChatMessage
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if loaded.Role != "user" || loaded.Content != "Hello" {
		t.Errorf("Unexpected: role=%s, content=%s", loaded.Role, loaded.Content)
	}
}
