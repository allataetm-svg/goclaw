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

type AnthropicProvider struct {
	APIKey string
}

func (p *AnthropicProvider) ID() string   { return "anthropic" }
func (p *AnthropicProvider) Name() string { return "Anthropic (Claude)" }

func (p *AnthropicProvider) FetchModels() ([]string, error) {
	return []string{
		"claude-3-5-sonnet-20240620",
		"claude-3-opus-20240229",
		"claude-3-haiku-20240307",
	}, nil
}

func (p *AnthropicProvider) buildRequest(ctx context.Context, model string, messages []ChatMessage, stream bool) (*http.Request, error) {
	var systemPrompt string
	var anthropicMessages []ChatMessage

	for _, m := range messages {
		if m.Role == "system" {
			systemPrompt += m.Content + "\n"
		} else {
			anthropicMessages = append(anthropicMessages, m)
		}
	}

	type reqBody struct {
		Model     string        `json:"model"`
		MaxTokens int           `json:"max_tokens"`
		System    string        `json:"system,omitempty"`
		Messages  []ChatMessage `json:"messages"`
		Stream    bool          `json:"stream,omitempty"`
	}

	data, err := json.Marshal(reqBody{
		Model:     model,
		MaxTokens: 4096,
		System:    strings.TrimSpace(systemPrompt),
		Messages:  anthropicMessages,
		Stream:    stream,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	return req, nil
}

func (p *AnthropicProvider) Query(ctx context.Context, model string, messages []ChatMessage) (string, error) {
	req, err := p.buildRequest(ctx, model, messages, false)
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("Anthropic request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Anthropic error (%d): %s", resp.StatusCode, string(body))
	}

	var chatResp struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("failed to decode Anthropic response: %w", err)
	}
	if len(chatResp.Content) > 0 {
		return strings.TrimSpace(chatResp.Content[0].Text), nil
	}
	return "", fmt.Errorf("empty response from Anthropic")
}

func (p *AnthropicProvider) QueryStream(ctx context.Context, model string, messages []ChatMessage, ch chan<- StreamChunk) {
	defer close(ch)

	req, err := p.buildRequest(ctx, model, messages, true)
	if err != nil {
		ch <- StreamChunk{Error: err}
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		ch <- StreamChunk{Error: fmt.Errorf("Anthropic request failed: %w", err)}
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		ch <- StreamChunk{Error: fmt.Errorf("Anthropic error (%d): %s", resp.StatusCode, string(body))}
		return
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		jsonData := strings.TrimPrefix(line, "data: ")

		var event struct {
			Type  string `json:"type"`
			Delta struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"delta"`
		}
		if err := json.Unmarshal([]byte(jsonData), &event); err != nil {
			continue
		}

		switch event.Type {
		case "content_block_delta":
			if event.Delta.Text != "" {
				ch <- StreamChunk{Text: event.Delta.Text}
			}
		case "message_stop":
			ch <- StreamChunk{Done: true}
			return
		}
	}
	if err := scanner.Err(); err != nil {
		ch <- StreamChunk{Error: fmt.Errorf("stream read error: %w", err)}
	}
}
