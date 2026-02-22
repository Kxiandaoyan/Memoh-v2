package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/Kxiandaoyan/Memoh-v2/internal/db"
	"github.com/Kxiandaoyan/Memoh-v2/internal/mcp"
)

// UnifiedToolsHandler handles unified tool management APIs (builtin + MCP).
type UnifiedToolsHandler struct {
	logger               *slog.Logger
	builtinConfigService *mcp.BuiltinToolConfigService
	mcpConnectionService *mcp.ConnectionService
}

// NewUnifiedToolsHandler creates a new UnifiedToolsHandler instance.
func NewUnifiedToolsHandler(
	logger *slog.Logger,
	builtinConfigService *mcp.BuiltinToolConfigService,
	mcpConnectionService *mcp.ConnectionService,
) *UnifiedToolsHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &UnifiedToolsHandler{
		logger:               logger.With(slog.String("handler", "unified_tools")),
		builtinConfigService: builtinConfigService,
		mcpConnectionService: mcpConnectionService,
	}
}

// Register registers the unified tools routes.
func (h *UnifiedToolsHandler) Register(e *echo.Echo) {
	// Tools routes under /bots/:bot_id/tools
	toolsGroup := e.Group("/bots/:bot_id/tools")
	toolsGroup.GET("", h.ListAllTools)
	toolsGroup.PUT("/builtin", h.UpdateBuiltinTools)
	toolsGroup.POST("/reset", h.ResetBuiltinTools)
}

// UnifiedToolItem is the merged format the frontend expects.
type UnifiedToolItem struct {
	Name              string `json:"name"`
	Category          string `json:"category"`
	Type              string `json:"type"` // "builtin" or "mcp"
	Enabled           bool   `json:"enabled"`
	Order             int    `json:"order"`
	MCPConnectionName string `json:"mcpConnectionName,omitempty"`
}

// ListAllToolsResponse contains a unified tools array.
type ListAllToolsResponse struct {
	Tools []UnifiedToolItem `json:"tools"`
}

// UpdateBuiltinToolsRequest contains the array of builtin tool configs to update.
type UpdateBuiltinToolsRequest struct {
	Tools []mcp.BuiltinToolConfig `json:"tools"`
}

// UpdateBuiltinToolsResponse confirms the update operation.
type UpdateBuiltinToolsResponse struct {
	Updated bool `json:"updated"`
}

// ResetBuiltinToolsResponse confirms the reset operation.
type ResetBuiltinToolsResponse struct {
	Reset bool `json:"reset"`
}

// ListAllTools godoc
// @Summary List all tools (builtin + MCP)
// @Description Returns a unified list of builtin tool configurations and MCP connections for a bot
// @Tags tools
// @Param bot_id path string true "Bot ID"
// @Success 200 {object} ListAllToolsResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /bots/{bot_id}/tools [get]
func (h *UnifiedToolsHandler) ListAllTools(c echo.Context) error {
	botID := strings.TrimSpace(c.Param("bot_id"))
	if botID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "bot_id is required")
	}

	ctx := c.Request().Context()

	// 1. Get builtin tool configurations
	builtinConfigs, err := h.builtinConfigService.GetByBot(ctx, botID)
	if err != nil {
		h.logger.Error("failed to get builtin tool configs",
			slog.String("bot_id", botID),
			slog.Any("error", err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to retrieve builtin tools")
	}

	// 2. Get MCP connections
	mcpConnections, err := h.mcpConnectionService.ListByBot(ctx, botID)
	if err != nil {
		h.logger.Error("failed to get MCP connections",
			slog.String("bot_id", botID),
			slog.Any("error", err))
		mcpConnections = []mcp.Connection{}
	}

	// 3. Merge into unified tools array
	tools := make([]UnifiedToolItem, 0, len(builtinConfigs)+len(mcpConnections))
	for i, cfg := range builtinConfigs {
		tools = append(tools, UnifiedToolItem{
			Name:     cfg.ToolName,
			Category: cfg.Category,
			Type:     "builtin",
			Enabled:  cfg.Enabled,
			Order:    i,
		})
	}
	for i, conn := range mcpConnections {
		tools = append(tools, UnifiedToolItem{
			Name:              conn.Name,
			Category:          "mcp",
			Type:              "mcp",
			Enabled:           conn.Active,
			Order:             len(builtinConfigs) + i,
			MCPConnectionName: conn.Name,
		})
	}

	return c.JSON(http.StatusOK, ListAllToolsResponse{Tools: tools})
}

// UpdateBuiltinTools godoc
// @Summary Update builtin tool configurations
// @Description Updates the enabled/disabled state and priority for builtin tools
// @Tags tools
// @Param bot_id path string true "Bot ID"
// @Param payload body UpdateBuiltinToolsRequest true "Builtin tool configurations"
// @Success 200 {object} UpdateBuiltinToolsResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /bots/{bot_id}/tools/builtin [put]
func (h *UnifiedToolsHandler) UpdateBuiltinTools(c echo.Context) error {
	botID := strings.TrimSpace(c.Param("bot_id"))
	if botID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "bot_id is required")
	}

	var req UpdateBuiltinToolsRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("invalid request: %v", err))
	}

	if len(req.Tools) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "tools array cannot be empty")
	}

	ctx := c.Request().Context()

	// Call BuiltinToolConfigService.UpsertBatch()
	err := h.builtinConfigService.UpsertBatch(ctx, botID, req.Tools)
	if err != nil {
		h.logger.Error("failed to update builtin tool configs",
			slog.String("bot_id", botID),
			slog.Any("error", err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update builtin tools")
	}

	h.logger.Info("updated builtin tool configs",
		slog.String("bot_id", botID),
		slog.Int("count", len(req.Tools)))

	return c.JSON(http.StatusOK, UpdateBuiltinToolsResponse{
		Updated: true,
	})
}

// ResetBuiltinTools godoc
// @Summary Reset builtin tools to defaults
// @Description Deletes existing builtin tool configurations and re-initializes with defaults
// @Tags tools
// @Param bot_id path string true "Bot ID"
// @Success 200 {object} ResetBuiltinToolsResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /bots/{bot_id}/tools/reset [post]
func (h *UnifiedToolsHandler) ResetBuiltinTools(c echo.Context) error {
	botID := strings.TrimSpace(c.Param("bot_id"))
	if botID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "bot_id is required")
	}

	// Validate bot_id is a valid UUID
	_, err := db.ParseUUID(botID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid bot_id format")
	}

	ctx := c.Request().Context()

	// Delete all existing builtin tool configs for this bot
	err = h.builtinConfigService.DeleteByBot(ctx, botID)
	if err != nil {
		h.logger.Error("failed to delete existing builtin tool configs",
			slog.String("bot_id", botID),
			slog.Any("error", err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to reset builtin tools")
	}

	// Reinitialize with defaults
	err = h.builtinConfigService.InitializeDefaults(ctx, botID)
	if err != nil {
		h.logger.Error("failed to initialize default builtin tool configs",
			slog.String("bot_id", botID),
			slog.Any("error", err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to initialize default tools")
	}

	h.logger.Info("reset builtin tool configs to defaults",
		slog.String("bot_id", botID))

	return c.JSON(http.StatusOK, ResetBuiltinToolsResponse{
		Reset: true,
	})
}
