package provider

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type GeminiProvider struct {
	APIKey string
}

func (p *GeminiProvider) ID() string   { return "gemini" }
func (p *GeminiProvider) Name() string { return "Google Gemini" }

func (p *GeminiProvider) FetchModels() ([]string, error) {
	req, err := http.NewRequest("GET", "https://generativelanguage.googleapis.com/v1beta/models?key="+p.APIKey, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Gemini models: %w", err)
	}
	defer resp.Body.Close()

	var data struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode Gemini models: %w", err)
	}

	var models []string
	for _, m := range data.Models {
		models = append(models, strings.TrimPrefix(m.Name, "models/"))
	}
	return models, nil
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiContent struct {
	Role  string       `json:"role"`
	Parts []geminiPart `json:"parts"`
}

func (p *GeminiProvider) buildContents(messages []ChatMessage) ([]geminiContent, *geminiContent) {
	var contents []geminiContent
	var systemInstruction *geminiContent

	for _, m := range messages {
		if m.Role == "system" {
			if systemInstruction == nil {
				systemInstruction = &geminiContent{Role: "user", Parts: []geminiPart{{Text: m.Content}}}
			} else {
				systemInstruction.Parts[0].Text += "\n" + m.Content
			}
			continue
		}
		role := m.Role
		if role == "assistant" {
			role = "model"
		}
		contents = append(contents, geminiContent{
			Role:  role,
			Parts: []geminiPart{{Text: m.Content}},
		})
	}
	return contents, systemInstruction
}

func (p *GeminiProvider) Query(model string, messages []ChatMessage) (string, error) {
	contents, systemInstruction := p.buildContents(messages)

	type reqBody struct {
		SystemInstruction *geminiContent  `json:"system_instruction,omitempty"`
		Contents          []geminiContent `json:"contents"`
	}
	data, err := json.Marshal(reqBody{SystemInstruction: systemInstruction, Contents: contents})
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", model, p.APIKey)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("Gemini request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Gemini error (%d): %s", resp.StatusCode, string(body))
	}

	var chatResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("failed to decode Gemini response: %w", err)
	}
	if len(chatResp.Candidates) > 0 && len(chatResp.Candidates[0].Content.Parts) > 0 {
		return strings.TrimSpace(chatResp.Candidates[0].Content.Parts[0].Text), nil
	}
	return "", fmt.Errorf("empty response from Gemini")
}

func (p *GeminiProvider) QueryStream(model string, messages []ChatMessage, ch chan<- StreamChunk) {
	defer close(ch)

	contents, systemInstruction := p.buildContents(messages)

	type reqBody struct {
		SystemInstruction *geminiContent  `json:"system_instruction,omitempty"`
		Contents          []geminiContent `json:"contents"`
	}
	data, err := json.Marshal(reqBody{SystemInstruction: systemInstruction, Contents: contents})
	if err != nil {
		ch <- StreamChunk{Error: fmt.Errorf("failed to marshal request: %w", err)}
		return
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:streamGenerateContent?alt=sse&key=%s", model, p.APIKey)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		ch <- StreamChunk{Error: fmt.Errorf("failed to create request: %w", err)}
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		ch <- StreamChunk{Error: fmt.Errorf("Gemini request failed: %w", err)}
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		ch <- StreamChunk{Error: fmt.Errorf("Gemini error (%d): %s", resp.StatusCode, string(body))}
		return
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		jsonData := strings.TrimPrefix(line, "data: ")

		var chunk struct {
			Candidates []struct {
				Content struct {
					Parts []struct {
						Text string `json:"text"`
					} `json:"parts"`
				} `json:"content"`
			} `json:"candidates"`
		}
		if err := json.Unmarshal([]byte(jsonData), &chunk); err != nil {
			continue
		}
		if len(chunk.Candidates) > 0 && len(chunk.Candidates[0].Content.Parts) > 0 {
			ch <- StreamChunk{Text: chunk.Candidates[0].Content.Parts[0].Text}
		}
	}
	if err := scanner.Err(); err != nil {
		ch <- StreamChunk{Error: fmt.Errorf("stream read error: %w", err)}
		return
	}
	ch <- StreamChunk{Done: true}
}
