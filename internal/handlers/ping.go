package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
)

type PingHandler struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

func NewPingHandler(log *slog.Logger, pool *pgxpool.Pool) *PingHandler {
	return &PingHandler{
		pool:   pool,
		logger: log.With(slog.String("handler", "ping")),
	}
}

func (h *PingHandler) Register(e *echo.Echo) {
	e.GET("/ping", h.Ping)
	e.GET("/health", h.Health)
	e.HEAD("/health", h.PingHead)
}

func (h *PingHandler) Ping(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"status": "ok",
	})
}

func (h *PingHandler) Health(c echo.Context) error {
	checks := map[string]string{}
	healthy := true

	if h.pool != nil {
		ctx, cancel := context.WithTimeout(c.Request().Context(), 3*time.Second)
		defer cancel()
		if err := h.pool.Ping(ctx); err != nil {
			h.logger.Warn("health check: database ping failed", slog.Any("error", err))
			checks["database"] = "unhealthy"
			healthy = false
		} else {
			checks["database"] = "ok"
		}
	} else {
		checks["database"] = "not configured"
		healthy = false
	}

	status := "ok"
	httpStatus := http.StatusOK
	if !healthy {
		status = "degraded"
		httpStatus = http.StatusServiceUnavailable
	}

	return c.JSON(httpStatus, map[string]any{
		"status": status,
		"checks": checks,
	})
}

func (h *PingHandler) PingHead(c echo.Context) error {
	return c.NoContent(http.StatusOK)
}
