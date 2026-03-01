package provider

import "github.com/allataetm-svg/goclaw/internal/config"

// MakeProvider creates the appropriate LLMProvider from a ProviderConfig
func MakeProvider(pc config.ProviderConfig) LLMProvider {
	switch pc.ID {
	case "openrouter":
		return &OpenAICompatProvider{ProviderID: "openrouter", ProviderName: "OpenRouter", BaseURL: "https://openrouter.ai/api/v1", APIKey: pc.APIKey}
	case "opencode_zen":
		return &OpenAICompatProvider{ProviderID: "opencode_zen", ProviderName: "Opencode Zen", BaseURL: "https://opencode.ai/zen/v1", APIKey: pc.APIKey}
	case "openai":
		return &OpenAICompatProvider{ProviderID: "openai", ProviderName: "OpenAI", BaseURL: "https://api.openai.com/v1", APIKey: pc.APIKey}
	case "mistral":
		return &OpenAICompatProvider{ProviderID: "mistral", ProviderName: "Mistral AI", BaseURL: "https://api.mistral.ai/v1", APIKey: pc.APIKey}
	case "anthropic":
		return &AnthropicProvider{APIKey: pc.APIKey}
	case "gemini":
		return &GeminiProvider{APIKey: pc.APIKey}
	case "custom_openai":
		baseURL := pc.BaseURL
		if baseURL == "" {
			baseURL = "http://localhost:4000/v1"
		}
		return &OpenAICompatProvider{ProviderID: "custom_openai", ProviderName: "Custom (OpenAI Compat)", BaseURL: baseURL, APIKey: pc.APIKey}
	default:
		baseURL := pc.BaseURL
		if baseURL == "" {
			baseURL = "http://127.0.0.1:11434"
		}
		return &OllamaProvider{URL: baseURL}
	}
}
