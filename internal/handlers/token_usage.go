package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"

	"github.com/Kxiandaoyan/Memoh-v2/internal/accounts"
	"github.com/Kxiandaoyan/Memoh-v2/internal/bots"
	"github.com/Kxiandaoyan/Memoh-v2/internal/db"
	dbsqlc "github.com/Kxiandaoyan/Memoh-v2/internal/db/sqlc"
)

// TokenUsageHandler handles token usage API endpoints.
type TokenUsageHandler struct {
	queries        *dbsqlc.Queries
	botService     *bots.Service
	accountService *accounts.Service
}

// NewTokenUsageHandler creates a new TokenUsageHandler.
func NewTokenUsageHandler(queries *dbsqlc.Queries, botService *bots.Service, accountService *accounts.Service) *TokenUsageHandler {
	return &TokenUsageHandler{
		queries:        queries,
		botService:     botService,
		accountService: accountService,
	}
}

// Register registers token usage routes.
func (h *TokenUsageHandler) Register(e *echo.Echo) {
	e.GET("/bots/:bot_id/token-usage/total", h.GetBotTotal)
	e.GET("/bots/:bot_id/token-usage/daily", h.GetBotDaily)
	e.GET("/token-usage/all", h.GetAllBotsTotals)
	e.GET("/token-usage/daily", h.GetAllBotsDaily)
	e.GET("/token-usage/by-model", h.GetByModel)
}

// GetBotTotal godoc
// @Summary Get total token usage for a bot
// @Tags token-usage
// @Param bot_id path string true "Bot ID"
// @Success 200 {object} map[string]int64
// @Router /bots/{bot_id}/token-usage/total [get]
func (h *TokenUsageHandler) GetBotTotal(c echo.Context) error {
	if _, err := h.requireAccess(c); err != nil {
		return err
	}
	botID := strings.TrimSpace(c.Param("bot_id"))
	botUUID, err := db.ParseUUID(botID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid bot id")
	}
	row, err := h.queries.GetBotTokenTotal(c.Request().Context(), botUUID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, map[string]int64{
		"prompt_tokens":     row.PromptTokens,
		"completion_tokens": row.CompletionTokens,
		"total_tokens":      row.TotalTokens,
	})
}

// GetBotDaily godoc
// @Summary Get daily token usage series for a bot
// @Tags token-usage
// @Param bot_id path string true "Bot ID"
// @Param days query int false "Number of days (default 30)"
// @Success 200 {object} map[string]any
// @Router /bots/{bot_id}/token-usage/daily [get]
func (h *TokenUsageHandler) GetBotDaily(c echo.Context) error {
	if _, err := h.requireAccess(c); err != nil {
		return err
	}
	botID := strings.TrimSpace(c.Param("bot_id"))
	botUUID, err := db.ParseUUID(botID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid bot id")
	}
	since, until := parseDateRange(c)
	rows, err := h.queries.GetBotTokenDailySeries(c.Request().Context(), dbsqlc.GetBotTokenDailySeriesParams{
		BotID: botUUID,
		Since: toPgTimestamptz(since),
		Until: toPgTimestamptz(until),
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	type dailyEntry struct {
		Day              string `json:"day"`
		TotalTokens      int64  `json:"total_tokens"`
		PromptTokens     int64  `json:"prompt_tokens"`
		CompletionTokens int64  `json:"completion_tokens"`
	}
	items := make([]dailyEntry, 0, len(rows))
	for _, r := range rows {
		items = append(items, dailyEntry{
			Day:              r.Day.Time.Format("2006-01-02"),
			TotalTokens:      r.TotalTokens,
			PromptTokens:     r.PromptTokens,
			CompletionTokens: r.CompletionTokens,
		})
	}
	return c.JSON(http.StatusOK, map[string]any{"items": items})
}

// GetAllBotsTotals godoc
// @Summary Get token usage totals for all bots
// @Tags token-usage
// @Success 200 {object} map[string]any
// @Router /token-usage/all [get]
func (h *TokenUsageHandler) GetAllBotsTotals(c echo.Context) error {
	if _, err := h.requireAccess(c); err != nil {
		return err
	}
	rows, err := h.queries.GetAllBotsTokenTotals(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	type botTotal struct {
		BotID            string `json:"bot_id"`
		PromptTokens     int64  `json:"prompt_tokens"`
		CompletionTokens int64  `json:"completion_tokens"`
		TotalTokens      int64  `json:"total_tokens"`
	}
	items := make([]botTotal, 0, len(rows))
	for _, r := range rows {
		items = append(items, botTotal{
			BotID:            db.UUIDToString(r.BotID),
			PromptTokens:     r.PromptTokens,
			CompletionTokens: r.CompletionTokens,
			TotalTokens:      r.TotalTokens,
		})
	}
	return c.JSON(http.StatusOK, map[string]any{"items": items})
}

// GetAllBotsDaily godoc
// @Summary Get daily token usage series for all bots
// @Tags token-usage
// @Param days query int false "Number of days (default 30)"
// @Success 200 {object} map[string]any
// @Router /token-usage/daily [get]
func (h *TokenUsageHandler) GetAllBotsDaily(c echo.Context) error {
	if _, err := h.requireAccess(c); err != nil {
		return err
	}
	since, until := parseDateRange(c)
	rows, err := h.queries.GetAllBotsTokenDailySeries(c.Request().Context(), dbsqlc.GetAllBotsTokenDailySeriesParams{
		Since: toPgTimestamptz(since),
		Until: toPgTimestamptz(until),
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	type dailyEntry struct {
		BotID            string `json:"bot_id"`
		Day              string `json:"day"`
		TotalTokens      int64  `json:"total_tokens"`
		PromptTokens     int64  `json:"prompt_tokens"`
		CompletionTokens int64  `json:"completion_tokens"`
	}
	items := make([]dailyEntry, 0, len(rows))
	for _, r := range rows {
		items = append(items, dailyEntry{
			BotID:            db.UUIDToString(r.BotID),
			Day:              r.Day.Time.Format("2006-01-02"),
			TotalTokens:      r.TotalTokens,
			PromptTokens:     r.PromptTokens,
			CompletionTokens: r.CompletionTokens,
		})
	}
	return c.JSON(http.StatusOK, map[string]any{"items": items})
}

// GetByModel godoc
// @Summary Get token usage totals grouped by model
// @Tags token-usage
// @Success 200 {object} map[string]any
// @Router /token-usage/by-model [get]
func (h *TokenUsageHandler) GetByModel(c echo.Context) error {
	if _, err := h.requireAccess(c); err != nil {
		return err
	}
	rows, err := h.queries.GetTokenTotalsByModel(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	type modelTotal struct {
		Model            string `json:"model"`
		PromptTokens     int64  `json:"prompt_tokens"`
		CompletionTokens int64  `json:"completion_tokens"`
		TotalTokens      int64  `json:"total_tokens"`
	}
	items := make([]modelTotal, 0, len(rows))
	for _, r := range rows {
		items = append(items, modelTotal{
			Model:            r.Model,
			PromptTokens:     r.PromptTokens,
			CompletionTokens: r.CompletionTokens,
			TotalTokens:      r.TotalTokens,
		})
	}
	return c.JSON(http.StatusOK, map[string]any{"items": items})
}

func (h *TokenUsageHandler) requireAccess(c echo.Context) (string, error) {
	return RequireChannelIdentityID(c)
}

func parseDateRange(c echo.Context) (time.Time, time.Time) {
	days := 30
	if s := strings.TrimSpace(c.QueryParam("days")); s != "" {
		if n, err := time.ParseDuration(s + "h"); err == nil {
			days = int(n.Hours())
		}
	}
	if d := c.QueryParam("days"); d != "" {
		if n := parseInt(d); n > 0 && n <= 365 {
			days = n
		}
	}
	until := time.Now().UTC().Truncate(24 * time.Hour).Add(24 * time.Hour)
	since := until.Add(-time.Duration(days) * 24 * time.Hour)
	return since, until
}

func parseInt(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return n
		}
		n = n*10 + int(c-'0')
	}
	return n
}

func toPgTimestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}
