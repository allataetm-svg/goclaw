package agent

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/allataetm-svg/goclaw/internal/config"
)

type SecurityLevel string

const (
	SecurityStrict   SecurityLevel = "strict"
	SecurityModerate SecurityLevel = "moderate"
	SecurityOff      SecurityLevel = "off"
)

var (
	blockedPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)ignore[_ ]previous[_ ]instructions`),
		regexp.MustCompile(`(?i)system[:\s]`),
		regexp.MustCompile(`(?i)you are now`),
		regexp.MustCompile(`(?i)\bDAN\b`),
		regexp.MustCompile(`(?i)do anything now`),
		regexp.MustCompile(`(?i)jailbreak`),
		regexp.MustCompile(`(?i)prompt injection`),
		regexp.MustCompile(`(?i)override your instructions`),
		regexp.MustCompile(`(?i)forget all previous`),
		regexp.MustCompile(`(?i)new instructions`),
		regexp.MustCompile(`(?i)developer mode`),
		regexp.MustCompile(`(?i)pretend to be`),
		regexp.MustCompile(`(?i)roleplay as`),
		regexp.MustCompile(`(?i)ignore all rules`),
		regexp.MustCompile(`(?i)bypass.*(safety|filter|restriction)`),
		regexp.MustCompile(`(?i)CALL:[\s]*`),
		regexp.MustCompile(`(?i)function call`),
		regexp.MustCompile(`\{\{.*\}\}`),
		regexp.MustCompile(`<script[^>]*>`),
		regexp.MustCompile(`javascript:`),
		regexp.MustCompile(`on\w+\s*=`),
	}
)

type WebFetchTool struct{}

func (t *WebFetchTool) Name() string { return "web_fetch" }

func (t *WebFetchTool) Description() string {
	return `Fetches content from a URL. Args: { "url": "string", "security_level": "string (optional: strict, moderate, off)", "max_length": "number (optional)" }`
}

type FetchResult struct {
	Content    string
	Title      string
	URL        string
	WasCleaned bool
	Warnings   []string
}

func (t *WebFetchTool) Execute(ctx context.Context, args map[string]interface{}, conf config.Config) (string, error) {
	url, ok := args["url"].(string)
	if !ok || url == "" {
		return "", fmt.Errorf("missing url parameter")
	}

	securityLevel := SecurityModerate
	if sl, ok := args["security_level"].(string); ok {
		securityLevel = SecurityLevel(strings.ToLower(sl))
	}

	maxLength := 10000
	if ml, ok := args["max_length"].(float64); ok {
		maxLength = int(ml)
	}

	result, err := fetchURL(ctx, url, securityLevel, maxLength)
	if err != nil {
		return "", fmt.Errorf("failed to fetch URL: %w", err)
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("## Fetched: %s\n\n", result.Title))
	output.WriteString(fmt.Sprintf("URL: %s\n\n", result.URL))

	if result.WasCleaned {
		output.WriteString("⚠️ **Warning**: Content was cleaned due to potential security concerns.\n\n")
		for _, w := range result.Warnings {
			output.WriteString(fmt.Sprintf("- %s\n", w))
		}
		output.WriteString("\n")
	}

	output.WriteString("### Content:\n")
	output.WriteString(result.Content)

	return output.String(), nil
}

func fetchURL(ctx context.Context, url string, securityLevel SecurityLevel, maxLength int) (*FetchResult, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, int64(2*maxLength)))
	if err != nil {
		return nil, err
	}

	content := string(body)

	content = extractTextFromHTML(content)

	if len(content) > maxLength {
		content = content[:maxLength] + "\n\n... (truncated)"
	}

	title := extractTitle(content)

	result := &FetchResult{
		Content: content,
		Title:   title,
		URL:     url,
	}

	if securityLevel != SecurityOff {
		cleaned, wasCleaned, warnings := sanitizeContent(content, securityLevel)
		result.Content = cleaned
		result.WasCleaned = wasCleaned
		result.Warnings = warnings
	}

	return result, nil
}

func extractTextFromHTML(html string) string {
	scriptRe := regexp.MustCompile(`<script[^>]*>.*?</script>`)
	html = scriptRe.ReplaceAllString(html, "")

	styleRe := regexp.MustCompile(`<style[^>]*>.*?</style>`)
	html = styleRe.ReplaceAllString(html, "")

	html = regexp.MustCompile(`<[^>]+>`).ReplaceAllString(html, "\n")

	html = regexp.MustCompile(`[\t ]+`).ReplaceAllString(html, " ")

	html = regexp.MustCompile(`\n\s*\n`).ReplaceAllString(html, "\n\n")

	lines := strings.Split(html, "\n")
	var cleaned []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			cleaned = append(cleaned, line)
		}
	}

	return strings.Join(cleaned, "\n")
}

func extractTitle(content string) string {
	lines := strings.Split(content, "\n")
	if len(lines) > 0 {
		firstLine := strings.TrimSpace(lines[0])
		if len(firstLine) > 100 {
			firstLine = firstLine[:100]
		}
		return firstLine
	}
	return "Untitled"
}

func sanitizeContent(content string, level SecurityLevel) (string, bool, []string) {
	if level == SecurityOff {
		return content, false, nil
	}

	var warnings []string
	cleaned := content

	for _, pattern := range blockedPatterns {
		matches := pattern.FindAllString(cleaned, -1)
		if len(matches) > 0 {
			warnings = append(warnings, fmt.Sprintf("Found blocked pattern: %s", truncate(matches[0], 50)))

			if level == SecurityStrict {
				cleaned = pattern.ReplaceAllString(cleaned, "[BLOCKED]")
			}
		}
	}

	cleaned = strings.ReplaceAll(cleaned, "{{", "")
	cleaned = strings.ReplaceAll(cleaned, "}}", "")

	wasCleaned := len(warnings) > 0

	return cleaned, wasCleaned, warnings
}
