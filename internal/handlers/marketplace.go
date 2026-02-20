package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/Kxiandaoyan/Memoh-v2/internal/config"
)

var marketplaceHTTPClient = &http.Client{Timeout: 15 * time.Second}

const smitheryAPIBase = "https://api.smithery.ai"

type MarketplaceHandler struct {
	smitheryKey string
	logger      *slog.Logger
}

func NewMarketplaceHandler(log *slog.Logger, cfg config.SmitheryConfig) *MarketplaceHandler {
	return &MarketplaceHandler{
		smitheryKey: cfg.APIKey,
		logger:      log.With(slog.String("handler", "marketplace")),
	}
}

func (h *MarketplaceHandler) Register(e *echo.Echo) {
	g := e.Group("/mcp-marketplace")
	g.GET("/search", h.Search)
	g.GET("/detail", h.Detail)
	g.GET("/skills", h.SearchSkills)
	g.GET("/skills/detail", h.SkillDetail)
}

// Search proxies a search request to the Smithery registry.
// @Summary Search MCP marketplace
// @Tags marketplace
// @Param q query string false "Search query"
// @Param page query int false "Page number (1-indexed)"
// @Param pageSize query int false "Results per page"
// @Success 200 {object} map[string]any
// @Router /mcp-marketplace/search [get]
func (h *MarketplaceHandler) Search(c echo.Context) error {
	q := c.QueryParam("q")
	page := c.QueryParam("page")
	pageSize := c.QueryParam("pageSize")

	params := url.Values{}
	if q != "" {
		params.Set("q", q)
	}
	if page != "" {
		params.Set("page", page)
	}
	if pageSize == "" {
		pageSize = "20"
	}
	params.Set("pageSize", pageSize)

	apiURL := smitheryAPIBase + "/servers?" + params.Encode()
	body, status, err := h.doSmitheryRequest(apiURL)
	if err != nil {
		h.logger.Error("smithery search failed", "error", err)
		return echo.NewHTTPError(http.StatusBadGateway, "marketplace search failed")
	}

	return c.JSONBlob(status, body)
}

// Detail fetches full server details from Smithery, including connection config and tools.
// @Summary Get MCP server detail from marketplace
// @Tags marketplace
// @Param name query string true "Qualified name (e.g. namespace/server)"
// @Success 200 {object} map[string]any
// @Router /mcp-marketplace/detail [get]
func (h *MarketplaceHandler) Detail(c echo.Context) error {
	name := strings.TrimSpace(c.QueryParam("name"))
	if name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "name is required")
	}

	encoded := url.PathEscape(name)
	apiURL := fmt.Sprintf("%s/servers/%s", smitheryAPIBase, encoded)

	body, status, err := h.doSmitheryRequest(apiURL)
	if err != nil {
		h.logger.Error("smithery detail failed", "name", name, "error", err)
		return echo.NewHTTPError(http.StatusBadGateway, "marketplace detail failed")
	}

	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return c.JSONBlob(status, body)
	}

	result := map[string]any{
		"qualifiedName": raw["qualifiedName"],
		"displayName":   raw["displayName"],
		"description":   raw["description"],
		"iconUrl":       raw["iconUrl"],
		"remote":        raw["remote"],
		"deploymentUrl": raw["deploymentUrl"],
		"tools":         raw["tools"],
		"security":      raw["security"],
	}

	conns, ok := raw["connections"].([]any)
	if ok && len(conns) > 0 {
		result["connections"] = conns
	}

	return c.JSON(http.StatusOK, result)
}

// SearchSkills proxies a skills search request to Smithery.
// @Summary Search Smithery skills marketplace
// @Tags marketplace
// @Param q query string false "Search query"
// @Param page query int false "Page number (1-indexed)"
// @Param pageSize query int false "Results per page"
// @Success 200 {object} map[string]any
// @Router /mcp-marketplace/skills [get]
func (h *MarketplaceHandler) SearchSkills(c echo.Context) error {
	q := c.QueryParam("q")
	page := c.QueryParam("page")
	pageSize := c.QueryParam("pageSize")

	params := url.Values{}
	if q != "" {
		params.Set("q", q)
	}
	if page != "" {
		params.Set("page", page)
	}
	if pageSize == "" {
		pageSize = "20"
	}
	params.Set("pageSize", pageSize)

	apiURL := smitheryAPIBase + "/skills?" + params.Encode()
	body, status, err := h.doSmitheryRequest(apiURL)
	if err != nil {
		h.logger.Error("smithery skills search failed", "error", err)
		return echo.NewHTTPError(http.StatusBadGateway, "skills search failed")
	}

	return c.JSONBlob(status, body)
}

// SkillDetail fetches a single skill from Smithery including its prompt content.
// @Summary Get skill detail from Smithery
// @Tags marketplace
// @Param namespace query string true "Skill namespace"
// @Param slug query string true "Skill slug"
// @Success 200 {object} map[string]any
// @Router /mcp-marketplace/skills/detail [get]
func (h *MarketplaceHandler) SkillDetail(c echo.Context) error {
	ns := strings.TrimSpace(c.QueryParam("namespace"))
	slug := strings.TrimSpace(c.QueryParam("slug"))
	if ns == "" || slug == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "namespace and slug are required")
	}

	apiURL := fmt.Sprintf("%s/skills/%s/%s", smitheryAPIBase, url.PathEscape(ns), url.PathEscape(slug))
	body, status, err := h.doSmitheryRequest(apiURL)
	if err != nil {
		h.logger.Error("smithery skill detail failed", "namespace", ns, "slug", slug, "error", err)
		return echo.NewHTTPError(http.StatusBadGateway, "skill detail failed")
	}

	return c.JSONBlob(status, body)
}

func (h *MarketplaceHandler) doSmitheryRequest(apiURL string) ([]byte, int, error) {
	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Accept", "application/json")
	if h.smitheryKey != "" {
		req.Header.Set("Authorization", "Bearer "+h.smitheryKey)
	}

	resp, err := marketplaceHTTPClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, 0, err
	}

	return body, resp.StatusCode, nil
}
