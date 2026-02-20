package skillstore

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	mcpgw "github.com/Kxiandaoyan/Memoh-v2/internal/mcp"
	mcpcontainer "github.com/Kxiandaoyan/Memoh-v2/internal/mcp/providers/container"
	"github.com/Kxiandaoyan/Memoh-v2/internal/searchproviders"
	"github.com/Kxiandaoyan/Memoh-v2/internal/settings"
	"gopkg.in/yaml.v3"
)

const (
	toolDiscoverSkills = "discover_skills"
	toolForkSkill      = "fork_skill"

	execWorkDir = "/data"
)

type Executor struct {
	logger          *slog.Logger
	execRunner      mcpcontainer.ExecRunner
	settings        *settings.Service
	searchProviders *searchproviders.Service
	dataRoot        string
}

func NewExecutor(
	log *slog.Logger,
	execRunner mcpcontainer.ExecRunner,
	settingsSvc *settings.Service,
	searchSvc *searchproviders.Service,
	dataRoot string,
) *Executor {
	if log == nil {
		log = slog.Default()
	}
	return &Executor{
		logger:          log.With(slog.String("provider", "skillstore")),
		execRunner:      execRunner,
		settings:        settingsSvc,
		searchProviders: searchSvc,
		dataRoot:        dataRoot,
	}
}

func (e *Executor) ListTools(_ context.Context, _ mcpgw.ToolSessionContext) ([]mcpgw.ToolDescriptor, error) {
	return []mcpgw.ToolDescriptor{
		{
			Name:        toolDiscoverSkills,
			Description: "Search for AI agent skills from ClawHub marketplace, the web, or the shared workspace between bots. Returns a list of skill summaries.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "Search keywords for discovering skills",
					},
					"source": map[string]any{
						"type":        "string",
						"enum":        []string{"clawhub", "web", "shared", "all"},
						"description": "Where to search: clawhub (marketplace), web (internet), shared (other bots), all (every source). Default: all",
					},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        toolForkSkill,
			Description: "Fetch a skill from a source (ClawHub, web URL, or shared workspace) and save it into your skills directory. Returns the original skill content so you can adapt it with the write tool.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"source": map[string]any{
						"type":        "string",
						"enum":        []string{"clawhub", "web", "shared"},
						"description": "Where to fetch from: clawhub, web, or shared",
					},
					"slug": map[string]any{
						"type":        "string",
						"description": "ClawHub skill slug (required when source=clawhub)",
					},
					"url": map[string]any{
						"type":        "string",
						"description": "URL to fetch SKILL.md content (required when source=web)",
					},
					"skill_name": map[string]any{
						"type":        "string",
						"description": "Skill directory name in shared workspace (required when source=shared)",
					},
					"save_as": map[string]any{
						"type":        "string",
						"description": "Name for the new skill directory under /data/.skills/",
					},
				},
				"required": []string{"source", "save_as"},
			},
		},
	}, nil
}

func (e *Executor) CallTool(ctx context.Context, session mcpgw.ToolSessionContext, toolName string, arguments map[string]any) (map[string]any, error) {
	botID := strings.TrimSpace(session.BotID)
	if botID == "" {
		return mcpgw.BuildToolErrorResult("bot_id is required"), nil
	}
	switch toolName {
	case toolDiscoverSkills:
		return e.discoverSkills(ctx, session, arguments)
	case toolForkSkill:
		return e.forkSkill(ctx, session, arguments)
	default:
		return nil, mcpgw.ErrToolNotFound
	}
}

// ─── discover_skills ────────────────────────────────────────────────────────

type skillResult struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Source      string `json:"source"`
	Slug        string `json:"slug,omitempty"`
	URL         string `json:"url,omitempty"`
	Author      string `json:"author,omitempty"`
	Version     string `json:"version,omitempty"`
}

func (e *Executor) discoverSkills(ctx context.Context, session mcpgw.ToolSessionContext, args map[string]any) (map[string]any, error) {
	query := mcpgw.StringArg(args, "query")
	if query == "" {
		return mcpgw.BuildToolErrorResult("query is required"), nil
	}
	source := mcpgw.StringArg(args, "source")
	if source == "" {
		source = "all"
	}

	type sourceResult struct {
		results []skillResult
		err     error
		name    string
	}

	var sources []func() sourceResult

	switch source {
	case "clawhub":
		sources = append(sources, func() sourceResult {
			r, err := e.searchClawHub(ctx, session.BotID, query)
			return sourceResult{results: r, err: err, name: "clawhub"}
		})
	case "web":
		sources = append(sources, func() sourceResult {
			r, err := e.searchWeb(ctx, session.BotID, query)
			return sourceResult{results: r, err: err, name: "web"}
		})
	case "shared":
		sources = append(sources, func() sourceResult {
			r, err := e.searchShared(query)
			return sourceResult{results: r, err: err, name: "shared"}
		})
	case "all":
		sources = append(sources,
			func() sourceResult {
				r, err := e.searchClawHub(ctx, session.BotID, query)
				return sourceResult{results: r, err: err, name: "clawhub"}
			},
			func() sourceResult {
				r, err := e.searchWeb(ctx, session.BotID, query)
				return sourceResult{results: r, err: err, name: "web"}
			},
			func() sourceResult {
				r, err := e.searchShared(query)
				return sourceResult{results: r, err: err, name: "shared"}
			},
		)
	default:
		return mcpgw.BuildToolErrorResult("invalid source: must be clawhub, web, shared, or all"), nil
	}

	var wg sync.WaitGroup
	ch := make(chan sourceResult, len(sources))
	for _, fn := range sources {
		wg.Add(1)
		go func(f func() sourceResult) {
			defer wg.Done()
			ch <- f()
		}(fn)
	}
	wg.Wait()
	close(ch)

	var all []skillResult
	var errors []string
	for sr := range ch {
		if sr.err != nil {
			e.logger.Warn("discover_skills: source failed",
				slog.String("source", sr.name), slog.Any("error", sr.err))
			errors = append(errors, sr.name+": "+sr.err.Error())
			continue
		}
		all = append(all, sr.results...)
	}

	seen := map[string]bool{}
	deduped := make([]skillResult, 0, len(all))
	for _, r := range all {
		key := strings.ToLower(r.Name + "|" + r.Source)
		if seen[key] {
			continue
		}
		seen[key] = true
		deduped = append(deduped, r)
	}

	result := map[string]any{
		"query":   query,
		"source":  source,
		"results": deduped,
		"count":   len(deduped),
	}
	if len(errors) > 0 {
		result["warnings"] = errors
	}
	return mcpgw.BuildToolSuccessResult(result), nil
}

func (e *Executor) searchClawHub(ctx context.Context, botID, query string) ([]skillResult, error) {
	if e.execRunner == nil {
		return nil, fmt.Errorf("container runtime not available")
	}
	containerID := mcpgw.ContainerPrefix + botID
	script := fmt.Sprintf("clawhub search %s --json 2>/dev/null || echo '[]'",
		mcpcontainer.ShellQuote(query))

	result, err := e.execRunner.ExecWithCapture(ctx, mcpgw.ExecRequest{
		BotID:   botID,
		Command: []string{"/bin/sh", "-c", script},
		WorkDir: execWorkDir,
	})
	if err != nil {
		return nil, fmt.Errorf("clawhub exec failed (container %s): %w", containerID, err)
	}

	stdout := strings.TrimSpace(result.Stdout)
	if stdout == "" || stdout == "[]" {
		return nil, nil
	}

	var raw []struct {
		Name        string `json:"name"`
		Slug        string `json:"slug"`
		Description string `json:"description"`
		Author      string `json:"author"`
		Version     string `json:"version"`
	}
	if err := json.Unmarshal([]byte(stdout), &raw); err != nil {
		return nil, fmt.Errorf("clawhub response parse error: %w", err)
	}

	results := make([]skillResult, 0, len(raw))
	for _, item := range raw {
		results = append(results, skillResult{
			Name:        item.Name,
			Description: item.Description,
			Source:      "clawhub",
			Slug:        item.Slug,
			Author:      item.Author,
			Version:     item.Version,
		})
	}
	return results, nil
}

func (e *Executor) searchWeb(ctx context.Context, botID, query string) ([]skillResult, error) {
	if e.settings == nil || e.searchProviders == nil {
		return nil, fmt.Errorf("search provider not configured")
	}
	botSettings, err := e.settings.GetBot(ctx, botID)
	if err != nil {
		return nil, fmt.Errorf("failed to load bot settings: %w", err)
	}
	searchProviderID := strings.TrimSpace(botSettings.SearchProviderID)
	if searchProviderID == "" {
		return nil, fmt.Errorf("no search provider configured for this bot")
	}
	provider, err := e.searchProviders.GetRawByID(ctx, searchProviderID)
	if err != nil {
		return nil, fmt.Errorf("search provider lookup failed: %w", err)
	}

	enrichedQuery := "AI agent skill SKILL.md " + query
	searchResults, err := e.callSearch(ctx, provider.Provider, provider.Config, enrichedQuery, 5)
	if err != nil {
		return nil, err
	}

	results := make([]skillResult, 0, len(searchResults))
	for _, item := range searchResults {
		results = append(results, skillResult{
			Name:        item.Title,
			Description: item.Description,
			Source:      "web",
			URL:         item.URL,
		})
	}
	return results, nil
}

type webSearchItem struct {
	Title       string
	URL         string
	Description string
}

func (e *Executor) callSearch(ctx context.Context, providerName string, configJSON []byte, query string, count int) ([]webSearchItem, error) {
	cfg := parseConfig(configJSON)

	switch strings.TrimSpace(providerName) {
	case string(searchproviders.ProviderBrave):
		return e.callBraveSearch(ctx, cfg, query, count)
	case string(searchproviders.ProviderSerpApi):
		return e.callSerpApiSearch(ctx, cfg, query, count)
	default:
		return nil, fmt.Errorf("unsupported search provider: %s", providerName)
	}
}

func (e *Executor) callBraveSearch(ctx context.Context, cfg map[string]any, query string, count int) ([]webSearchItem, error) {
	endpoint := firstNonEmpty(stringValue(cfg["base_url"]), "https://api.search.brave.com/res/v1/web/search")
	apiKey := stringValue(cfg["api_key"])

	reqURL := fmt.Sprintf("%s?q=%s&count=%d", strings.TrimRight(endpoint, "/"),
		urlEncode(query), count)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if apiKey != "" {
		req.Header.Set("X-Subscription-Token", apiKey)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("brave search HTTP %d", resp.StatusCode)
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
		return nil, fmt.Errorf("brave response parse error: %w", err)
	}

	items := make([]webSearchItem, 0, len(raw.Web.Results))
	for _, r := range raw.Web.Results {
		items = append(items, webSearchItem{Title: r.Title, URL: r.URL, Description: r.Description})
	}
	return items, nil
}

func (e *Executor) callSerpApiSearch(ctx context.Context, cfg map[string]any, query string, count int) ([]webSearchItem, error) {
	apiKey := stringValue(cfg["api_key"])
	if apiKey == "" {
		return nil, fmt.Errorf("serpapi api_key is required")
	}
	engine := firstNonEmpty(stringValue(cfg["engine"]), "google")
	endpoint := firstNonEmpty(stringValue(cfg["base_url"]), "https://serpapi.com/search.json")

	reqURL := fmt.Sprintf("%s?q=%s&api_key=%s&engine=%s&num=%d",
		strings.TrimRight(endpoint, "/"),
		urlEncode(query), urlEncode(apiKey), urlEncode(engine), count)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("serpapi HTTP %d", resp.StatusCode)
	}

	var raw struct {
		OrganicResults []struct {
			Title   string `json:"title"`
			Link    string `json:"link"`
			Snippet string `json:"snippet"`
		} `json:"organic_results"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("serpapi response parse error: %w", err)
	}

	items := make([]webSearchItem, 0, len(raw.OrganicResults))
	for _, r := range raw.OrganicResults {
		items = append(items, webSearchItem{Title: r.Title, URL: r.Link, Description: r.Snippet})
	}
	return items, nil
}

func (e *Executor) searchShared(query string) ([]skillResult, error) {
	sharedSkillsDir := filepath.Join(e.dataRoot, "shared", ".skills")
	entries, err := os.ReadDir(sharedSkillsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read shared skills dir: %w", err)
	}

	keyword := strings.ToLower(query)
	var results []skillResult
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillPath := filepath.Join(sharedSkillsDir, entry.Name(), "SKILL.md")
		data, err := os.ReadFile(skillPath)
		if err != nil {
			continue
		}
		parsed := parseSkillFrontmatter(string(data), entry.Name())
		nameLC := strings.ToLower(parsed.Name)
		descLC := strings.ToLower(parsed.Description)
		if strings.Contains(nameLC, keyword) || strings.Contains(descLC, keyword) || keyword == "" {
			results = append(results, skillResult{
				Name:        parsed.Name,
				Description: parsed.Description,
				Source:      "shared",
			})
		}
	}
	return results, nil
}

// ─── fork_skill ─────────────────────────────────────────────────────────────

func (e *Executor) forkSkill(ctx context.Context, session mcpgw.ToolSessionContext, args map[string]any) (map[string]any, error) {
	source := mcpgw.StringArg(args, "source")
	saveAs := mcpgw.StringArg(args, "save_as")
	if source == "" {
		return mcpgw.BuildToolErrorResult("source is required"), nil
	}
	if saveAs == "" {
		return mcpgw.BuildToolErrorResult("save_as is required"), nil
	}
	if !isValidSkillName(saveAs) {
		return mcpgw.BuildToolErrorResult("invalid save_as: must not contain '..' or path separators"), nil
	}

	var content string
	var err error

	switch source {
	case "clawhub":
		slug := mcpgw.StringArg(args, "slug")
		if slug == "" {
			return mcpgw.BuildToolErrorResult("slug is required for source=clawhub"), nil
		}
		content, err = e.fetchClawHub(ctx, session.BotID, slug)
	case "web":
		rawURL := mcpgw.StringArg(args, "url")
		if rawURL == "" {
			return mcpgw.BuildToolErrorResult("url is required for source=web"), nil
		}
		content, err = e.fetchWeb(ctx, rawURL)
	case "shared":
		skillName := mcpgw.StringArg(args, "skill_name")
		if skillName == "" {
			return mcpgw.BuildToolErrorResult("skill_name is required for source=shared"), nil
		}
		content, err = e.fetchShared(skillName)
	default:
		return mcpgw.BuildToolErrorResult("invalid source: must be clawhub, web, or shared"), nil
	}
	if err != nil {
		return mcpgw.BuildToolErrorResult(fmt.Sprintf("failed to fetch skill: %s", err)), nil
	}
	if strings.TrimSpace(content) == "" {
		return mcpgw.BuildToolErrorResult("fetched skill content is empty"), nil
	}

	skillFilePath := fmt.Sprintf(".skills/%s/SKILL.md", saveAs)
	if err := mcpcontainer.ExecWrite(ctx, e.execRunner, session.BotID, execWorkDir, skillFilePath, content); err != nil {
		return mcpgw.BuildToolErrorResult(fmt.Sprintf("failed to write skill: %s", err)), nil
	}

	preview := content
	if len(preview) > 500 {
		preview = preview[:500] + "\n... (truncated)"
	}

	e.logger.Info("fork_skill: saved",
		slog.String("bot_id", session.BotID),
		slog.String("source", source),
		slog.String("save_as", saveAs))

	return mcpgw.BuildToolSuccessResult(map[string]any{
		"success":        true,
		"skill_name":     saveAs,
		"path":           "/data/" + skillFilePath,
		"source":         source,
		"content_preview": preview,
	}), nil
}

func (e *Executor) fetchClawHub(ctx context.Context, botID, slug string) (string, error) {
	if e.execRunner == nil {
		return "", fmt.Errorf("container runtime not available")
	}
	if strings.ContainsAny(slug, ";|&$`") {
		return "", fmt.Errorf("invalid slug")
	}

	// Try clawhub show first
	script := fmt.Sprintf("clawhub show %s --raw 2>/dev/null", mcpcontainer.ShellQuote(slug))
	result, err := e.execRunner.ExecWithCapture(ctx, mcpgw.ExecRequest{
		BotID:   botID,
		Command: []string{"/bin/sh", "-c", script},
		WorkDir: execWorkDir,
	})
	if err == nil && result.ExitCode == 0 && strings.TrimSpace(result.Stdout) != "" {
		return strings.TrimSpace(result.Stdout), nil
	}

	// Fallback: install to temp dir, read, then clean up
	tmpDir := fmt.Sprintf("/tmp/.skill-fork-%d", time.Now().UnixNano())
	installScript := fmt.Sprintf(
		"mkdir -p %s && clawhub install %s --dir %s 2>/dev/null && cat %s/*/SKILL.md 2>/dev/null || cat %s/SKILL.md 2>/dev/null; rm -rf %s",
		mcpcontainer.ShellQuote(tmpDir),
		mcpcontainer.ShellQuote(slug),
		mcpcontainer.ShellQuote(tmpDir),
		mcpcontainer.ShellQuote(tmpDir),
		mcpcontainer.ShellQuote(tmpDir),
		mcpcontainer.ShellQuote(tmpDir),
	)
	result, err = e.execRunner.ExecWithCapture(ctx, mcpgw.ExecRequest{
		BotID:   botID,
		Command: []string{"/bin/sh", "-c", installScript},
		WorkDir: execWorkDir,
	})
	if err != nil {
		return "", fmt.Errorf("clawhub install failed: %w", err)
	}
	content := strings.TrimSpace(result.Stdout)
	if content == "" {
		return "", fmt.Errorf("clawhub install returned empty content for slug: %s", slug)
	}
	return content, nil
}

func (e *Executor) fetchWeb(ctx context.Context, rawURL string) (string, error) {
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		return "", fmt.Errorf("url must start with http:// or https://")
	}

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Memoh-SkillFetcher/1.0")
	req.Header.Set("Accept", "text/markdown, text/plain, */*")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("HTTP %d fetching %s", resp.StatusCode, rawURL)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
	if err != nil {
		return "", err
	}

	content := string(body)
	// If the response looks like raw markdown (has frontmatter), use as-is
	if strings.HasPrefix(strings.TrimSpace(content), "---") {
		return strings.TrimSpace(content), nil
	}
	// Try to extract SKILL.md content from HTML if it contains fenced frontmatter
	if idx := strings.Index(content, "---\n"); idx >= 0 {
		extracted := content[idx:]
		if endIdx := strings.Index(extracted[4:], "\n---"); endIdx >= 0 {
			return strings.TrimSpace(extracted), nil
		}
	}
	// Return raw content as-is; the bot can process it
	return strings.TrimSpace(content), nil
}

func (e *Executor) fetchShared(skillName string) (string, error) {
	if !isValidSkillName(skillName) {
		return "", fmt.Errorf("invalid skill name")
	}
	skillPath := filepath.Join(e.dataRoot, "shared", ".skills", skillName, "SKILL.md")
	data, err := os.ReadFile(skillPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("skill %q not found in shared workspace", skillName)
		}
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// ─── helpers ────────────────────────────────────────────────────────────────

type parsedFrontmatter struct {
	Name        string
	Description string
}

func parseSkillFrontmatter(raw, fallbackName string) parsedFrontmatter {
	result := parsedFrontmatter{Name: fallbackName}
	trimmed := strings.TrimSpace(raw)
	if !strings.HasPrefix(trimmed, "---") {
		return result
	}
	rest := trimmed[3:]
	rest = strings.TrimLeft(rest, " \t")
	if len(rest) > 0 && rest[0] == '\n' {
		rest = rest[1:]
	} else if len(rest) > 1 && rest[0] == '\r' && rest[1] == '\n' {
		rest = rest[2:]
	}
	closingIdx := strings.Index(rest, "\n---")
	if closingIdx < 0 {
		return result
	}
	var fm struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
	}
	if err := yaml.Unmarshal([]byte(rest[:closingIdx]), &fm); err != nil {
		return result
	}
	if strings.TrimSpace(fm.Name) != "" {
		result.Name = strings.TrimSpace(fm.Name)
	}
	result.Description = strings.TrimSpace(fm.Description)
	return result
}

func isValidSkillName(name string) bool {
	if name == "" {
		return false
	}
	if strings.Contains(name, "..") || strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return false
	}
	return true
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
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func urlEncode(s string) string {
	var b strings.Builder
	for _, c := range []byte(s) {
		if isUnreserved(c) {
			b.WriteByte(c)
		} else {
			fmt.Fprintf(&b, "%%%02X", c)
		}
	}
	return b.String()
}

func isUnreserved(c byte) bool {
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') ||
		(c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.' || c == '~'
}
