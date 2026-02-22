package webread

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	mcpgw "github.com/Kxiandaoyan/Memoh-v2/internal/mcp"
	mcpcontainer "github.com/Kxiandaoyan/Memoh-v2/internal/mcp/providers/container"
)

// degradationMetadata tracks which strategies were attempted and performance metrics.
type degradationMetadata struct {
	Attempts   []string
	FinalMethod string
	StartTime  time.Time
	Duration   time.Duration
}

func newDegradationMetadata() *degradationMetadata {
	return &degradationMetadata{
		Attempts:  []string{},
		StartTime: time.Now(),
	}
}

func (m *degradationMetadata) AddAttempt(attempt string) {
	m.Attempts = append(m.Attempts, attempt)
}

func (m *degradationMetadata) Complete() {
	m.Duration = time.Since(m.StartTime)
}

// ─────────────────────────────────────────────────────────────────────────────
// Level 1: Markdown Header Strategy
// ─────────────────────────────────────────────────────────────────────────────

// tryMarkdownHeader attempts to fetch content using curl with Accept: text/markdown.
// This works well for sites that support markdown conversion (like some documentation sites).
func (e *Executor) tryMarkdownHeader(ctx context.Context, botID string, urlStr string) (string, error) {
	// Validate URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return "", fmt.Errorf("invalid url: %w", err)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return "", fmt.Errorf("url must be http or https")
	}

	// Execute curl with markdown header
	cmd := fmt.Sprintf("curl -L -s -H 'Accept: text/markdown' %s 2>&1", mcpcontainer.ShellQuote(urlStr))
	result, err := e.execRunner.ExecWithCapture(ctx, mcpgw.ExecRequest{
		BotID:   botID,
		Command: []string{"/bin/sh", "-c", cmd},
		WorkDir: execWorkDir,
	})
	if err != nil {
		return "", fmt.Errorf("curl exec failed: %w", err)
	}
	if result.ExitCode != 0 {
		return "", fmt.Errorf("curl failed (exit %d): %s", result.ExitCode, result.Stderr)
	}

	content := strings.TrimSpace(result.Stdout)
	if content == "" {
		return "", fmt.Errorf("empty response from server")
	}

	// Validate that response is actually markdown (not HTML)
	if !isValidMarkdown(content) {
		return "", fmt.Errorf("response is not valid markdown (appears to be HTML or plain text)")
	}

	return content, nil
}

// isValidMarkdown checks if content appears to be markdown rather than HTML.
// This is a heuristic check to filter out HTML responses.
func isValidMarkdown(content string) bool {
	if len(content) < 50 {
		// Too short to be meaningful content
		return false
	}

	lower := strings.ToLower(content)

	// Reject HTML
	if strings.Contains(lower, "<html") ||
		strings.Contains(lower, "<!doctype") ||
		strings.Contains(lower, "<body") ||
		strings.Contains(lower, "<head") {
		return false
	}

	// Check for markdown indicators
	hasMarkdownHeading := strings.Contains(content, "# ") ||
		strings.Contains(content, "## ") ||
		strings.Contains(content, "### ")
	hasMarkdownLink := strings.Contains(content, "](")
	hasMarkdownList := strings.Contains(content, "\n- ") ||
		strings.Contains(content, "\n* ") ||
		strings.Contains(content, "\n1. ")
	hasMarkdownCode := strings.Contains(content, "```")

	// Content is likely markdown if it has at least one markdown indicator
	// and doesn't look like HTML
	markdownScore := 0
	if hasMarkdownHeading {
		markdownScore++
	}
	if hasMarkdownLink {
		markdownScore++
	}
	if hasMarkdownList {
		markdownScore++
	}
	if hasMarkdownCode {
		markdownScore++
	}

	return markdownScore >= 1
}

// ─────────────────────────────────────────────────────────────────────────────
// Level 2: Web Search Strategy
// ─────────────────────────────────────────────────────────────────────────────

// tryWebSearch attempts to get content by searching for the URL and extracting
// relevant snippets from search results. This is useful when direct access fails
// but the content is indexed by search engines.
func (e *Executor) tryWebSearch(ctx context.Context, session mcpgw.ToolSessionContext, urlStr string) (string, error) {
	if e.webExec == nil {
		return "", fmt.Errorf("web executor not available")
	}

	// Create a search query based on the URL
	// For now, just use the full URL as the query
	// Future: could extract domain/path for better queries
	query := urlStr

	// Call web_search tool
	searchArgs := map[string]any{
		"query": query,
		"count": 5, // Get top 5 results
	}

	result, err := e.webExec.CallTool(ctx, session, "web_search", searchArgs)
	if err != nil {
		return "", fmt.Errorf("web_search failed: %w", err)
	}

	// Check if search returned an error
	if isError, ok := result["isError"].(bool); ok && isError {
		return "", fmt.Errorf("web_search returned error")
	}

	// Extract structured content
	structured, ok := result["structuredContent"].(map[string]any)
	if !ok {
		return "", fmt.Errorf("unexpected web_search response format")
	}

	// Extract results array
	resultsRaw, ok := structured["results"]
	if !ok {
		return "", fmt.Errorf("no results in web_search response")
	}

	results, ok := resultsRaw.([]any)
	if !ok {
		return "", fmt.Errorf("results is not an array")
	}

	if len(results) == 0 {
		return "", fmt.Errorf("no search results found")
	}

	// Build content from search results
	var contentBuilder strings.Builder
	contentBuilder.WriteString(fmt.Sprintf("# Search Results for %s\n\n", urlStr))

	for i, resultRaw := range results {
		resultMap, ok := resultRaw.(map[string]any)
		if !ok {
			continue
		}

		title, _ := resultMap["title"].(string)
		description, _ := resultMap["description"].(string)
		resultURL, _ := resultMap["url"].(string)

		if title != "" || description != "" {
			contentBuilder.WriteString(fmt.Sprintf("## Result %d\n", i+1))
			if title != "" {
				contentBuilder.WriteString(fmt.Sprintf("**Title:** %s\n\n", title))
			}
			if resultURL != "" {
				contentBuilder.WriteString(fmt.Sprintf("**URL:** %s\n\n", resultURL))
			}
			if description != "" {
				contentBuilder.WriteString(fmt.Sprintf("%s\n\n", description))
			}
			contentBuilder.WriteString("---\n\n")
		}
	}

	content := strings.TrimSpace(contentBuilder.String())
	if content == "" {
		return "", fmt.Errorf("no usable content in search results")
	}

	return content, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Level 3: Actionbook Strategy
// ─────────────────────────────────────────────────────────────────────────────

// tryActionbook searches for and retrieves pre-defined browser automation
// sequences (actionbooks) for specific domains. This strategy is useful when
// direct access and search fail, but an actionbook exists for the target domain.
func (e *Executor) tryActionbook(ctx context.Context, session mcpgw.ToolSessionContext, urlStr string) (string, error) {
	if e.browserExec == nil {
		return "", fmt.Errorf("browser executor not available")
	}

	// Step 1: Search for actionbook using URL as query
	searchArgs := map[string]any{
		"query": urlStr,
		"limit": 1, // We only need the first match
	}

	searchResult, err := e.browserExec.CallTool(ctx, session, "actionbook_search", searchArgs)
	if err != nil {
		return "", fmt.Errorf("actionbook_search failed: %w", err)
	}

	// Check if search returned an error
	if isError, ok := searchResult["isError"].(bool); ok && isError {
		return "", fmt.Errorf("actionbook_search returned error")
	}

	// Extract structured content
	structured, ok := searchResult["structuredContent"].(map[string]any)
	if !ok {
		return "", fmt.Errorf("unexpected actionbook_search response format")
	}

	// Extract results array
	resultsRaw, ok := structured["results"]
	if !ok {
		return "", fmt.Errorf("no results in actionbook_search response")
	}

	results, ok := resultsRaw.([]any)
	if !ok {
		return "", fmt.Errorf("results is not an array")
	}

	if len(results) == 0 {
		return "", fmt.Errorf("no actionbooks found for URL")
	}

	// Get first result
	firstResult, ok := results[0].(map[string]any)
	if !ok {
		return "", fmt.Errorf("invalid result format")
	}

	// Extract actionbook ID
	actionbookID, ok := firstResult["id"].(string)
	if !ok || actionbookID == "" {
		return "", fmt.Errorf("actionbook result missing id")
	}

	// Step 2: Get actionbook content
	getArgs := map[string]any{
		"id": actionbookID,
	}

	getResult, err := e.browserExec.CallTool(ctx, session, "actionbook_get", getArgs)
	if err != nil {
		return "", fmt.Errorf("actionbook_get failed: %w", err)
	}

	// Check for get errors
	if isError, ok := getResult["isError"].(bool); ok && isError {
		return "", fmt.Errorf("actionbook_get returned error")
	}

	// Extract content from actionbook
	content, err := extractContentFromActionbook(getResult)
	if err != nil {
		return "", fmt.Errorf("failed to extract content from actionbook: %w", err)
	}

	if strings.TrimSpace(content) == "" {
		return "", fmt.Errorf("actionbook returned empty content")
	}

	return content, nil
}

// extractContentFromActionbook parses the actionbook_get response and extracts content.
func extractContentFromActionbook(result map[string]any) (string, error) {
	// Try structured content first
	if structured, ok := result["structuredContent"].(map[string]any); ok {
		// Try content field
		if content, ok := structured["content"].(string); ok && content != "" {
			return content, nil
		}
		// Try text field
		if text, ok := structured["text"].(string); ok && text != "" {
			return text, nil
		}
		// Try description field
		if desc, ok := structured["description"].(string); ok && desc != "" {
			return desc, nil
		}
	}

	// Fallback: try to extract from content array (similar to browser result)
	if content, ok := result["content"].([]any); ok && len(content) > 0 {
		if textBlock, ok := content[0].(map[string]any); ok {
			if text, ok := textBlock["text"].(string); ok {
				return text, nil
			}
		}
	}

	// Last resort: JSON encode the whole result
	if jsonBytes, err := json.Marshal(result); err == nil {
		return string(jsonBytes), nil
	}

	return "", fmt.Errorf("unable to extract content from actionbook result")
}

// ─────────────────────────────────────────────────────────────────────────────
// Level 4: Browser Automation Strategy
// ─────────────────────────────────────────────────────────────────────────────

// tryAgentBrowser uses full browser automation to navigate and extract content.
// This is the most powerful but also slowest strategy, used as last resort.
func (e *Executor) tryAgentBrowser(ctx context.Context, session mcpgw.ToolSessionContext, urlStr string) (string, error) {
	if e.browserExec == nil {
		return "", fmt.Errorf("browser executor not available")
	}

	// Step 1: Navigate to URL
	navigateArgs := map[string]any{
		"url": urlStr,
	}
	navResult, err := e.browserExec.CallTool(ctx, session, "browser_navigate", navigateArgs)
	if err != nil {
		return "", fmt.Errorf("browser_navigate failed: %w", err)
	}

	// Check for navigation errors
	if isError, ok := navResult["isError"].(bool); ok && isError {
		// Extract error message from content
		if content, ok := navResult["content"].([]any); ok && len(content) > 0 {
			if textBlock, ok := content[0].(map[string]any); ok {
				if errMsg, ok := textBlock["text"].(string); ok {
					return "", fmt.Errorf("browser navigation failed: %s", errMsg)
				}
			}
		}
		return "", fmt.Errorf("browser navigation failed")
	}

	// Step 2: Wait a bit for page to load (optional, can be made configurable)
	waitArgs := map[string]any{
		"timeout": 3000, // 3 seconds
	}
	_, _ = e.browserExec.CallTool(ctx, session, "browser_wait", waitArgs)

	// Step 3: Extract page text
	getTextArgs := map[string]any{} // Empty args = get full page text
	textResult, err := e.browserExec.CallTool(ctx, session, "browser_get_text", getTextArgs)
	if err != nil {
		return "", fmt.Errorf("browser_get_text failed: %w", err)
	}

	// Check for get_text errors
	if isError, ok := textResult["isError"].(bool); ok && isError {
		return "", fmt.Errorf("browser text extraction failed")
	}

	// Extract text content from structured result
	content, err := extractTextFromBrowserResult(textResult)
	if err != nil {
		return "", fmt.Errorf("failed to extract text from browser result: %w", err)
	}

	if strings.TrimSpace(content) == "" {
		return "", fmt.Errorf("browser extracted empty content")
	}

	return content, nil
}

// extractTextFromBrowserResult parses the browser_get_text response and extracts text.
func extractTextFromBrowserResult(result map[string]any) (string, error) {
	// Try structured content first
	if structured, ok := result["structuredContent"].(map[string]any); ok {
		if text, ok := structured["text"].(string); ok {
			return text, nil
		}
	}

	// Fallback: try to extract from content array
	if content, ok := result["content"].([]any); ok && len(content) > 0 {
		if textBlock, ok := content[0].(map[string]any); ok {
			if text, ok := textBlock["text"].(string); ok {
				return text, nil
			}
		}
	}

	// Last resort: try to JSON encode the whole result
	if jsonBytes, err := json.Marshal(result); err == nil {
		return string(jsonBytes), nil
	}

	return "", fmt.Errorf("unable to extract text from browser result")
}
