package templates

import (
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
)

// Handler serves bot template endpoints.
type Handler struct {
	logger *slog.Logger
}

// NewHandler creates a new template handler.
func NewHandler(log *slog.Logger) *Handler {
	if log == nil {
		log = slog.Default()
	}
	return &Handler{
		logger: log.With(slog.String("handler", "templates")),
	}
}

// Register registers template routes.
func (h *Handler) Register(e *echo.Echo) {
	g := e.Group("/templates")
	g.GET("", h.ListTemplates)
	g.GET("/:id", h.GetTemplate)
}

// ListTemplates godoc
// @Summary List bot templates
// @Description List all available predefined bot templates
// @Tags templates
// @Success 200 {object} ListTemplatesResponse
// @Failure 500 {object} map[string]string
// @Router /templates [get]
func (h *Handler) ListTemplates(c echo.Context) error {
	items, err := List()
	if err != nil {
		h.logger.Error("failed to list templates", slog.Any("error", err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to load templates")
	}
	return c.JSON(http.StatusOK, ListTemplatesResponse{Items: items})
}

// GetTemplate godoc
// @Summary Get bot template
// @Description Get a specific bot template by ID
// @Tags templates
// @Param id path string true "Template ID"
// @Success 200 {object} Template
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /templates/{id} [get]
func (h *Handler) GetTemplate(c echo.Context) error {
	id := c.Param("id")
	tmpl, err := Get(id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, err.Error())
	}
	return c.JSON(http.StatusOK, tmpl)
}
