package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type OllamaProvider struct {
	URL string
}

func (p *OllamaProvider) ID() string   { return "ollama" }
func (p *OllamaProvider) Name() string { return "Ollama" }

func (p *OllamaProvider) FetchModels() ([]string, error) {
	resp, err := http.Get(p.URL + "/api/tags")
	if err != nil {
		return nil, fmt.Errorf("could not connect to Ollama at %s: %w", p.URL, err)
	}
	defer resp.Body.Close()

	var data struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode Ollama models: %w", err)
	}
	var models []string
	for _, m := range data.Models {
		models = append(models, m.Name)
	}
	return models, nil
}

func (p *OllamaProvider) Query(ctx context.Context, model string, messages []ChatMessage) (string, error) {
	type reqBody struct {
		Model    string        `json:"model"`
		Messages []ChatMessage `json:"messages"`
		Stream   bool          `json:"stream"`
	}
	data, err := json.Marshal(reqBody{Model: model, Messages: messages, Stream: false})
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.URL+"/api/chat", bytes.NewBuffer(data))
	if err != nil {
		return "", fmt.Errorf("failed to create Ollama request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("Ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Ollama error: HTTP %d", resp.StatusCode)
	}

	var chatResp struct {
		Message ChatMessage `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("failed to decode Ollama response: %w", err)
	}
	return strings.TrimSpace(chatResp.Message.Content), nil
}

func (p *OllamaProvider) QueryStream(ctx context.Context, model string, messages []ChatMessage, ch chan<- StreamChunk) {
	defer close(ch)

	type reqBody struct {
		Model    string        `json:"model"`
		Messages []ChatMessage `json:"messages"`
		Stream   bool          `json:"stream"`
	}
	data, err := json.Marshal(reqBody{Model: model, Messages: messages, Stream: true})
	if err != nil {
		ch <- StreamChunk{Error: fmt.Errorf("failed to marshal request: %w", err)}
		return
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.URL+"/api/chat", bytes.NewBuffer(data))
	if err != nil {
		ch <- StreamChunk{Error: fmt.Errorf("failed to create Ollama request: %w", err)}
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		ch <- StreamChunk{Error: fmt.Errorf("Ollama request failed: %w", err)}
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		ch <- StreamChunk{Error: fmt.Errorf("Ollama error: HTTP %d", resp.StatusCode)}
		return
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		var chunk struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
			Done bool `json:"done"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &chunk); err != nil {
			continue
		}
		if chunk.Done {
			ch <- StreamChunk{Done: true}
			return
		}
		ch <- StreamChunk{Text: chunk.Message.Content}
	}
	if err := scanner.Err(); err != nil {
		ch <- StreamChunk{Error: fmt.Errorf("stream read error: %w", err)}
	}
}
