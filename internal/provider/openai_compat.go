package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type OpenAICompatProvider struct {
	ProviderID   string
	ProviderName string
	BaseURL      string
	APIKey       string
}

func (p *OpenAICompatProvider) ID() string   { return p.ProviderID }
func (p *OpenAICompatProvider) Name() string { return p.ProviderName }

func (p *OpenAICompatProvider) FetchModels() ([]string, error) {
	req, err := http.NewRequest("GET", p.BaseURL+"/models", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	if p.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.APIKey)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch models from %s: %w", p.ProviderName, err)
	}
	defer resp.Body.Close()

	var data struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode models response: %w", err)
	}

	var models []string
	for _, m := range data.Data {
		models = append(models, m.ID)
	}
	return models, nil
}

func (p *OpenAICompatProvider) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	if p.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.APIKey)
	}
	if p.ProviderID == "openrouter" {
		req.Header.Set("HTTP-Referer", "http://localhost")
		req.Header.Set("X-Title", "GoClaw")
	}
}

func (p *OpenAICompatProvider) Query(ctx context.Context, model string, messages []ChatMessage) (string, error) {
	type reqBody struct {
		Model    string        `json:"model"`
		Messages []ChatMessage `json:"messages"`
	}
	data, err := json.Marshal(reqBody{Model: model, Messages: messages})
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.BaseURL+"/chat/completions", bytes.NewBuffer(data))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	p.setHeaders(req)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("%s request failed: %w", p.ProviderName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("%s API error (%d): %s", p.ProviderName, resp.StatusCode, string(body))
	}

	var chatResp struct {
		Choices []struct {
			Message ChatMessage `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}
	if len(chatResp.Choices) > 0 {
		return strings.TrimSpace(chatResp.Choices[0].Message.Content), nil
	}
	return "", fmt.Errorf("empty response from %s", p.ProviderName)
}

func (p *OpenAICompatProvider) QueryStream(ctx context.Context, model string, messages []ChatMessage, ch chan<- StreamChunk) {
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

	req, err := http.NewRequestWithContext(ctx, "POST", p.BaseURL+"/chat/completions", bytes.NewBuffer(data))
	if err != nil {
		ch <- StreamChunk{Error: fmt.Errorf("failed to create request: %w", err)}
		return
	}
	p.setHeaders(req)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		ch <- StreamChunk{Error: fmt.Errorf("%s request failed: %w", p.ProviderName, err)}
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		ch <- StreamChunk{Error: fmt.Errorf("%s API error (%d): %s", p.ProviderName, resp.StatusCode, string(body))}
		return
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		jsonData := strings.TrimPrefix(line, "data: ")
		if jsonData == "[DONE]" {
			ch <- StreamChunk{Done: true}
			return
		}
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(jsonData), &chunk); err != nil {
			continue
		}
		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			ch <- StreamChunk{Text: chunk.Choices[0].Delta.Content}
		}
	}
	if err := scanner.Err(); err != nil {
		ch <- StreamChunk{Error: fmt.Errorf("stream read error: %w", err)}
	}
}
