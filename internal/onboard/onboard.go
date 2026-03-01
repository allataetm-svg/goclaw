package onboard

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/huh"

	"github.com/openclaw-alternative/goclaw/internal/config"
	"github.com/openclaw-alternative/goclaw/internal/provider"
)

// Run starts the onboarding wizard
func Run() {
	fmt.Println("Welcome to the Lobster Wizard! 🦞")

	var conf config.Config

	var selectedIDs []string
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Which Providers would you like to add? (Multiple can be selected)").
				Options(
					huh.NewOption("Ollama (Local)", "ollama").Selected(true),
					huh.NewOption("Custom (LiteLLM/vLLM/Local)", "custom_openai"),
					huh.NewOption("Opencode Zen (Enterprise)", "opencode_zen"),
					huh.NewOption("OpenRouter (Cloud)", "openrouter"),
					huh.NewOption("OpenAI", "openai"),
					huh.NewOption("Anthropic (Claude)", "anthropic"),
					huh.NewOption("Google Gemini", "gemini"),
					huh.NewOption("Mistral AI", "mistral"),
				).
				Height(12).
				Value(&selectedIDs),
		),
	).Run()

	if err != nil {
		fmt.Println("Wizard cancelled.")
		return
	}

	for _, id := range selectedIDs {
		apiKey := ""
		baseURL := ""

		if id == "custom_openai" || id == "ollama" {
			if err := huh.NewForm(
				huh.NewGroup(
					huh.NewInput().
						Title(fmt.Sprintf("Base URL for %s (optional, leave empty for default):", id)).
						Value(&baseURL),
				),
			).Run(); err != nil {
				fmt.Printf("⚠️ Skipped Base URL for %s\n", id)
			}
		}

		if id != "ollama" {
			if err := huh.NewForm(
				huh.NewGroup(
					huh.NewInput().
						Title(fmt.Sprintf("Enter API Key for %s:", id)).
						EchoMode(huh.EchoModePassword).
						Value(&apiKey),
				),
			).Run(); err != nil {
				fmt.Printf("⚠️ Skipped API Key for %s\n", id)
			}
		}

		conf.Providers = append(conf.Providers, config.ProviderConfig{
			ID:      id,
			APIKey:  strings.TrimSpace(apiKey),
			BaseURL: strings.TrimSpace(baseURL),
		})
	}

	fmt.Println("\nFetching model list, please wait...")
	var allModels []string

	for _, pc := range conf.Providers {
		prov := provider.MakeProvider(pc)
		models, err := prov.FetchModels()
		if err != nil {
			fmt.Printf("⚠️ Error fetching models from %s: %v\n", prov.Name(), err)
			if pc.ID == "custom_openai" {
				allModels = append(allModels, fmt.Sprintf("%s:gpt-3.5-turbo", pc.ID))
			}
			continue
		}
		for _, m := range models {
			allModels = append(allModels, fmt.Sprintf("%s:%s", pc.ID, m))
		}
	}

	if len(allModels) == 0 {
		fmt.Println("❌ No models found. Please check your settings and API keys.")
		return
	}
	sort.Strings(allModels)

	fmt.Println("\nNow let's create your default First Agent!")

	var agentName string
	var agentPrompt string
	var selectedModel string

	opts := make([]huh.Option[string], 0, len(allModels))
	for _, m := range allModels {
		opts = append(opts, huh.NewOption(m, m))
	}

	err = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Agent Name (e.g., Coder, Helper, Translator):").
				Value(&agentName),

			huh.NewText().
				Title("System Prompt (Instructions):").
				Description("How should the agent behave?").
				Value(&agentPrompt),

			huh.NewSelect[string]().
				Title("LLM Model for the Agent:").
				Options(opts...).
				Height(10).
				Value(&selectedModel),
		),
	).Run()

	if err != nil {
		fmt.Println("Wizard cancelled.")
		return
	}

	if agentName == "" {
		agentName = "GoClaw Assistant"
	}
	if agentPrompt == "" {
		agentPrompt = "You are a helpful and intelligent AI assistant."
	}

	firstAgent := config.AgentConfig{
		ID:           strings.ToLower(strings.ReplaceAll(agentName, " ", "_")),
		Name:         agentName,
		SystemPrompt: agentPrompt,
		Model:        selectedModel,
	}

	conf.Agents = append(conf.Agents, firstAgent)
	conf.DefaultAgent = firstAgent.ID
	conf.MaxTokens = 8000

	if err := config.Save(conf); err != nil {
		fmt.Printf("❌ Failed to save config: %v\n", err)
		return
	}

	fmt.Println("\n✅ Setup complete!")
	fmt.Printf("Agent Name: %s\nID: %s\nSelected LLM: %s\n", firstAgent.Name, firstAgent.ID, firstAgent.Model)
	fmt.Println("\nTo start chatting: goclaw tui")
}
