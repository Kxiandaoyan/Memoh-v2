package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	mcpgw "github.com/Kxiandaoyan/Memoh-v2/internal/mcp"
	"github.com/Kxiandaoyan/Memoh-v2/internal/searchproviders"
	"github.com/Kxiandaoyan/Memoh-v2/internal/settings"
)

const (
	toolWebSearch = "web_search"

	// SearXNG local instance (deployed alongside via docker-compose).
	searxngBaseURL = "http://searxng:8080"
)

type Executor struct {
	logger          *slog.Logger
	settings        *settings.Service
	searchProviders *searchproviders.Service
}

func NewExecutor(log *slog.Logger, settingsSvc *settings.Service, searchSvc *searchproviders.Service) *Executor {
	if log == nil {
		log = slog.Default()
	}
	return &Executor{
		logger:          log.With(slog.String("provider", "web_tool")),
		settings:        settingsSvc,
		searchProviders: searchSvc,
	}
}

func (p *Executor) ListTools(ctx context.Context, session mcpgw.ToolSessionContext) ([]mcpgw.ToolDescriptor, error) {
	return []mcpgw.ToolDescriptor{
		{
			Name:        toolWebSearch,
			Description: "Search web results via configured search provider.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{"type": "string", "description": "Search query"},
					"count": map[string]any{"type": "integer", "description": "Number of results, default 5"},
				},
				"required": []string{"query"},
			},
		},
	}, nil
}

func (p *Executor) CallTool(ctx context.Context, session mcpgw.ToolSessionContext, toolName string, arguments map[string]any) (map[string]any, error) {
	switch toolName {
	case toolWebSearch:
		return p.callWebSearchWithFallback(ctx, session, arguments)
	default:
		return nil, mcpgw.ErrToolNotFound
	}
}

// callWebSearchWithFallback tries SearXNG first, then falls back to the
// bot's configured commercial search provider.
func (p *Executor) callWebSearchWithFallback(ctx context.Context, session mcpgw.ToolSessionContext, arguments map[string]any) (map[string]any, error) {
	query := strings.TrimSpace(mcpgw.StringArg(arguments, "query"))
	if query == "" {
		return mcpgw.BuildToolErrorResult("query is required"), nil
	}
	count := 5
	if value, ok, err := mcpgw.IntArg(arguments, "count"); err != nil {
		return mcpgw.BuildToolErrorResult(err.Error()), nil
	} else if ok && value > 0 {
		count = value
	}
	if count > 20 {
		count = 20
	}

	// 1. Try SearXNG (local, free, no API key needed).
	result, err := p.callSearXNGSearch(ctx, query, count)
	if err == nil {
		return result, nil
	}
	p.logger.Warn("searxng search failed, falling back to configured provider",
		slog.String("query", query),
		slog.String("error", err.Error()),
	)

	// 2. Fallback to configured commercial provider.
	if p.settings == nil || p.searchProviders == nil {
		return mcpgw.BuildToolErrorResult("searxng unavailable and no fallback search provider configured"), nil
	}
	botID := strings.TrimSpace(session.BotID)
	if botID == "" {
		return mcpgw.BuildToolErrorResult("bot_id is required"), nil
	}
	botSettings, err := p.settings.GetBot(ctx, botID)
	if err != nil {
		return mcpgw.BuildToolErrorResult(err.Error()), nil
	}
	searchProviderID := strings.TrimSpace(botSettings.SearchProviderID)
	if searchProviderID == "" {
		return mcpgw.BuildToolErrorResult("searxng unavailable and no fallback search provider configured"), nil
	}
	provider, err := p.searchProviders.GetRawByID(ctx, searchProviderID)
	if err != nil {
		return mcpgw.BuildToolErrorResult(err.Error()), nil
	}
	return p.callWebSearch(ctx, provider.Provider, provider.Config, arguments)
}

func (p *Executor) callWebSearch(ctx context.Context, providerName string, configJSON []byte, arguments map[string]any) (map[string]any, error) {
	query := strings.TrimSpace(mcpgw.StringArg(arguments, "query"))
	if query == "" {
		return mcpgw.BuildToolErrorResult("query is required"), nil
	}
	count := 5
	if value, ok, err := mcpgw.IntArg(arguments, "count"); err != nil {
		return mcpgw.BuildToolErrorResult(err.Error()), nil
	} else if ok && value > 0 {
		count = value
	}
	if count > 20 {
		count = 20
	}

	switch strings.TrimSpace(providerName) {
	case string(searchproviders.ProviderBrave):
		return p.callBraveSearch(ctx, configJSON, query, count)
	case string(searchproviders.ProviderSerpApi):
		return p.callSerpApiSearch(ctx, configJSON, query, count)
	default:
		return mcpgw.BuildToolErrorResult("unsupported search provider"), nil
	}
}

func (p *Executor) callBraveSearch(ctx context.Context, configJSON []byte, query string, count int) (map[string]any, error) {
	cfg := parseConfig(configJSON)
	endpoint := strings.TrimRight(firstNonEmpty(stringValue(cfg["base_url"]), "https://api.search.brave.com/res/v1/web/search"), "/")
	reqURL, err := url.Parse(endpoint)
	if err != nil {
		return mcpgw.BuildToolErrorResult("invalid search provider base_url"), nil
	}
	params := reqURL.Query()
	params.Set("q", query)
	params.Set("count", fmt.Sprintf("%d", count))
	reqURL.RawQuery = params.Encode()

	timeout := parseTimeout(configJSON, 15*time.Second)
	client := &http.Client{Timeout: timeout}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return mcpgw.BuildToolErrorResult(err.Error()), nil
	}
	req.Header.Set("Accept", "application/json")
	apiKey := stringValue(cfg["api_key"])
	if strings.TrimSpace(apiKey) != "" {
		req.Header.Set("X-Subscription-Token", strings.TrimSpace(apiKey))
	}
	resp, err := client.Do(req)
	if err != nil {
		return mcpgw.BuildToolErrorResult(err.Error()), nil
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return mcpgw.BuildToolErrorResult(err.Error()), nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return mcpgw.BuildToolErrorResult("search request failed"), nil
	}
	var raw struct {
		Web struct {
			Results []struct {
				Title       string `json:"title"`
				URL         string `json:"url"`
				Description string `json:"description"`
			} `json:"results"`
		} `json:"web"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return mcpgw.BuildToolErrorResult("invalid search response"), nil
	}
	results := make([]map[string]any, 0, len(raw.Web.Results))
	for _, item := range raw.Web.Results {
		results = append(results, map[string]any{
			"title":       item.Title,
			"url":         item.URL,
			"description": item.Description,
		})
	}
	return mcpgw.BuildToolSuccessResult(map[string]any{
		"query":   query,
		"results": results,
	}), nil
}

func (p *Executor) callSerpApiSearch(ctx context.Context, configJSON []byte, query string, count int) (map[string]any, error) {
	cfg := parseConfig(configJSON)
	apiKey := strings.TrimSpace(stringValue(cfg["api_key"]))
	if apiKey == "" {
		return mcpgw.BuildToolErrorResult("serpapi api_key is required"), nil
	}

	engine := firstNonEmpty(stringValue(cfg["engine"]), "google")
	endpoint := firstNonEmpty(stringValue(cfg["base_url"]), "https://serpapi.com/search.json")

	reqURL, err := url.Parse(endpoint)
	if err != nil {
		return mcpgw.BuildToolErrorResult("invalid serpapi base_url"), nil
	}
	params := reqURL.Query()
	params.Set("q", query)
	params.Set("api_key", apiKey)
	params.Set("engine", engine)
	params.Set("num", fmt.Sprintf("%d", count))
	reqURL.RawQuery = params.Encode()

	timeout := parseTimeout(configJSON, 15*time.Second)
	client := &http.Client{Timeout: timeout}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return mcpgw.BuildToolErrorResult(err.Error()), nil
	}
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return mcpgw.BuildToolErrorResult(err.Error()), nil
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return mcpgw.BuildToolErrorResult(err.Error()), nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return mcpgw.BuildToolErrorResult(fmt.Sprintf("serpapi request failed (HTTP %d)", resp.StatusCode)), nil
	}

	var raw struct {
		OrganicResults []struct {
			Title   string `json:"title"`
			Link    string `json:"link"`
			Snippet string `json:"snippet"`
		} `json:"organic_results"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return mcpgw.BuildToolErrorResult("invalid serpapi response"), nil
	}

	results := make([]map[string]any, 0, len(raw.OrganicResults))
	for _, item := range raw.OrganicResults {
		results = append(results, map[string]any{
			"title":       item.Title,
			"url":         item.Link,
			"description": item.Snippet,
		})
	}
	return mcpgw.BuildToolSuccessResult(map[string]any{
		"query":   query,
		"results": results,
	}), nil
}

// callSearXNGSearch queries the local SearXNG instance (docker-compose service).
// Returns an error (instead of a tool-error map) so the caller can fallback.
func (p *Executor) callSearXNGSearch(ctx context.Context, query string, count int) (map[string]any, error) {
	reqURL, _ := url.Parse(searxngBaseURL + "/search")
	params := reqURL.Query()
	params.Set("q", query)
	params.Set("format", "json")
	params.Set("categories", "general")
	reqURL.RawQuery = params.Encode()

	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("searxng: build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("searxng: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("searxng: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("searxng: read body: %w", err)
	}

	var raw struct {
		Results []struct {
			Title   string  `json:"title"`
			URL     string  `json:"url"`
			Content string  `json:"content"`
			Score   float64 `json:"score"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("searxng: invalid JSON: %w", err)
	}
	if len(raw.Results) == 0 {
		return nil, fmt.Errorf("searxng: no results")
	}

	// SearXNG doesn't support count param; truncate client-side.
	items := raw.Results
	if len(items) > count {
		items = items[:count]
	}

	results := make([]map[string]any, 0, len(items))
	for _, item := range items {
		results = append(results, map[string]any{
			"title":       item.Title,
			"url":         item.URL,
			"description": item.Content,
		})
	}
	return mcpgw.BuildToolSuccessResult(map[string]any{
		"query":   query,
		"results": results,
	}), nil
}

func parseTimeout(configJSON []byte, fallback time.Duration) time.Duration {
	cfg := parseConfig(configJSON)
	raw, ok := cfg["timeout_seconds"]
	if !ok {
		return fallback
	}
	switch value := raw.(type) {
	case float64:
		if value > 0 {
			return time.Duration(value * float64(time.Second))
		}
	case int:
		if value > 0 {
			return time.Duration(value) * time.Second
		}
	}
	return fallback
}

func parseConfig(configJSON []byte) map[string]any {
	if len(configJSON) == 0 {
		return map[string]any{}
	}
	var cfg map[string]any
	if err := json.Unmarshal(configJSON, &cfg); err != nil || cfg == nil {
		return map[string]any{}
	}
	return cfg
}

func stringValue(raw any) string {
	if value, ok := raw.(string); ok {
		return strings.TrimSpace(value)
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
