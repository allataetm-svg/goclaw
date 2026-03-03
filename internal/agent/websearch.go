package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/allataetm-svg/goclaw/internal/config"
)

type WebSearchTool struct{}

func (t *WebSearchTool) Name() string { return "web_search" }

func (t *WebSearchTool) Description() string {
	return `Searches the web for information. Args: { "query": "string", "provider": "string (optional: duckduckgo, tavily, perplexity)", "max_results": "number (optional, default 5) }`
}

type SearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

type SearchProvider interface {
	Search(ctx context.Context, query string, maxResults int) ([]SearchResult, error)
	Name() string
}

func getSearchProvider(providerName string, conf config.Config) SearchProvider {
	switch strings.ToLower(providerName) {
	case "tavily":
		return &TavilyProvider{Config: conf}
	case "perplexity":
		return &PerplexityProvider{Config: conf}
	case "duckduckgo", "":
		return &DuckDuckGoProvider{}
	default:
		return &DuckDuckGoProvider{}
	}
}

type DuckDuckGoProvider struct{}

func (p *DuckDuckGoProvider) Name() string { return "duckduckgo" }

func (p *DuckDuckGoProvider) Search(ctx context.Context, query string, maxResults int) ([]SearchResult, error) {
	if maxResults == 0 {
		maxResults = 5
	}

	apiURL := fmt.Sprintf("https://html.duckduckgo.com/html/?q=%s", url.QueryEscape(query))

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	results, err := parseDuckDuckGoHTML(string(body), maxResults)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		results, err = parseDuckDuckGoLite(query, maxResults)
		if err != nil {
			return nil, err
		}
	}

	return results, nil
}

func parseDuckDuckGoHTML(html string, maxResults int) ([]SearchResult, error) {
	results := []SearchResult{}

	resultPattern := regexp.MustCompile(`<a rel="nofollow" class="result__a" href="([^"]+)"[^>]*>([^<]+)</a>`)
	snippetPattern := regexp.MustCompile(`<a class="result__snippet"[^>]*>([^<]+)</a>`)

	resultMatches := resultPattern.FindAllStringSubmatch(html, -1)
	snippetMatches := snippetPattern.FindAllStringSubmatch(html, -1)

	for i := 0; i < len(resultMatches) && i < maxResults; i++ {
		result := SearchResult{}
		result.URL = resultMatches[i][1]
		result.Title = strings.TrimSpace(htmlUnescape(resultMatches[i][2]))

		if i < len(snippetMatches) {
			result.Snippet = strings.TrimSpace(htmlUnescape(snippetMatches[i][1]))
		}

		if result.URL != "" && result.Title != "" {
			results = append(results, result)
		}
	}

	return results, nil
}

func parseDuckDuckGoLite(query string, maxResults int) ([]SearchResult, error) {
	apiURL := fmt.Sprintf("https://lite.duckduckgo.com/50x/?q=%s&format=json", url.QueryEscape(query))

	req, err := http.NewRequestWithContext(context.Background(), "GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data struct {
		Results []struct {
			URL   string `json:"u"`
			Title string `json:"t"`
			Desc  string `json:"d"`
		} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	results := []SearchResult{}
	for i, r := range data.Results {
		if i >= maxResults {
			break
		}
		results = append(results, SearchResult{
			Title:   r.Title,
			URL:     r.URL,
			Snippet: r.Desc,
		})
	}

	return results, nil
}

type TavilyProvider struct {
	Config config.Config
}

func (p *TavilyProvider) Name() string { return "tavily" }

func (p *TavilyProvider) Search(ctx context.Context, query string, maxResults int) ([]SearchResult, error) {
	if maxResults == 0 {
		maxResults = 5
	}

	apiKey := ""
	for _, prov := range p.Config.Providers {
		if strings.ToLower(prov.ID) == "tavily" {
			apiKey = prov.APIKey
			break
		}
	}

	if apiKey == "" {
		return nil, fmt.Errorf("Tavily API key not configured. Add a provider with id 'tavily' in config")
	}

	type TavilyResponse struct {
		Results []struct {
			Title   string  `json:"title"`
			URL     string  `json:"url"`
			Content string  `json:"content"`
			Score   float64 `json:"score"`
		} `json:"results"`
	}

	body, _ := json.Marshal(map[string]interface{}{
		"query":               query,
		"max_results":         maxResults,
		"include_answer":      false,
		"include_raw_content": false,
	})

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.tavily.com/search", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var tavilyResp TavilyResponse
	if err := json.NewDecoder(resp.Body).Decode(&tavilyResp); err != nil {
		return nil, err
	}

	results := []SearchResult{}
	for _, r := range tavilyResp.Results {
		snippet := r.Content
		if len(snippet) > 300 {
			snippet = snippet[:300] + "..."
		}
		results = append(results, SearchResult{
			Title:   r.Title,
			URL:     r.URL,
			Snippet: snippet,
		})
	}

	return results, nil
}

type PerplexityProvider struct {
	Config config.Config
}

func (p *PerplexityProvider) Name() string { return "perplexity" }

func (p *PerplexityProvider) Search(ctx context.Context, query string, maxResults int) ([]SearchResult, error) {
	if maxResults == 0 {
		maxResults = 5
	}

	apiKey := ""
	baseURL := "https://api.perplexity.ai"
	for _, prov := range p.Config.Providers {
		if strings.ToLower(prov.ID) == "perplexity" {
			apiKey = prov.APIKey
			if prov.BaseURL != "" {
				baseURL = prov.BaseURL
			}
			break
		}
	}

	if apiKey == "" {
		return nil, fmt.Errorf("Perplexity API key not configured. Add a provider with id 'perplexity' in config")
	}

	type PerplexityResponse struct {
		Results []struct {
			URL     string `json:"url"`
			Title   string `json:"title"`
			Snippet string `json:"snippet"`
		} `json:"results"`
	}

	type RequestBody struct {
		Query      string `json:"query"`
		MaxResults int    `json:"max_results"`
	}

	body, _ := json.Marshal(RequestBody{
		Query:      query,
		MaxResults: maxResults,
	})

	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/search", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var perpResp PerplexityResponse
	if err := json.NewDecoder(resp.Body).Decode(&perpResp); err != nil {
		return nil, err
	}

	results := []SearchResult{}
	for _, r := range perpResp.Results {
		results = append(results, SearchResult{
			Title:   r.Title,
			URL:     r.URL,
			Snippet: r.Snippet,
		})
	}

	return results, nil
}

func htmlUnescape(s string) string {
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = strings.ReplaceAll(s, "&#39;", "'")
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	return s
}

func (t *WebSearchTool) Execute(ctx context.Context, args map[string]interface{}, conf config.Config) (string, error) {
	query, ok := args["query"].(string)
	if !ok || query == "" {
		return "", fmt.Errorf("missing query parameter")
	}

	providerName := ""
	if p, ok := args["provider"].(string); ok {
		providerName = p
	}

	maxResults := 5
	if mr, ok := args["max_results"].(float64); ok {
		maxResults = int(mr)
	} else if mr, ok := args["max_results"].(string); ok {
		if parsed, err := strconv.Atoi(mr); err == nil {
			maxResults = parsed
		}
	}

	provider := getSearchProvider(providerName, conf)
	results, err := provider.Search(ctx, query, maxResults)
	if err != nil {
		return "", fmt.Errorf("search failed: %w", err)
	}

	if len(results) == 0 {
		return "No results found.", nil
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("## Web Search Results (%s)\n\n", provider.Name()))

	for i, r := range results {
		output.WriteString(fmt.Sprintf("### %d. %s\n", i+1, r.Title))
		output.WriteString(fmt.Sprintf("URL: %s\n\n", r.URL))
		if r.Snippet != "" {
			output.WriteString(fmt.Sprintf("%s\n\n", r.Snippet))
		}
	}

	return output.String(), nil
}
