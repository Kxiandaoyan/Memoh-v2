package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/Kxiandaoyan/Memoh-v2/internal/globalsettings"
)

// GlobalSettingsHandler exposes read/write access to global server settings.
type GlobalSettingsHandler struct {
	svc *globalsettings.Service
}

// NewGlobalSettingsHandler creates a new handler for global settings.
func NewGlobalSettingsHandler(svc *globalsettings.Service) *GlobalSettingsHandler {
	return &GlobalSettingsHandler{svc: svc}
}

// Register mounts the handler routes.
func (h *GlobalSettingsHandler) Register(e *echo.Echo) {
	e.GET("/settings/global", h.GetGlobalSettings)
	e.PUT("/settings/global", h.UpdateGlobalSettings)
}

// GlobalSettingsResponse is the response body for GET /settings/global.
type GlobalSettingsResponse struct {
	Timezone string `json:"timezone"`
}

// GetGlobalSettings returns global server settings.
// @Summary      Get global settings
// @Description  Returns global server configuration such as timezone.
// @Tags         settings
// @Produce      json
// @Success      200  {object}  GlobalSettingsResponse
// @Router       /settings/global [get]
func (h *GlobalSettingsHandler) GetGlobalSettings(c echo.Context) error {
	tz, _ := h.svc.GetTimezone()
	return c.JSON(http.StatusOK, GlobalSettingsResponse{
		Timezone: tz,
	})
}

// UpdateGlobalSettingsRequest is the request body for PUT /settings/global.
type UpdateGlobalSettingsRequest struct {
	Timezone string `json:"timezone"`
}

// UpdateGlobalSettings updates global server settings.
// @Summary      Update global settings
// @Description  Updates global server configuration such as timezone.
// @Tags         settings
// @Accept       json
// @Produce      json
// @Param        body body UpdateGlobalSettingsRequest true "Settings to update"
// @Success      200  {object}  GlobalSettingsResponse
// @Failure      400  {object}  map[string]string
// @Router       /settings/global [put]
func (h *GlobalSettingsHandler) UpdateGlobalSettings(c echo.Context) error {
	var req UpdateGlobalSettingsRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if req.Timezone == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "timezone is required"})
	}

	if err := h.svc.SetTimezone(c.Request().Context(), req.Timezone); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid timezone: " + err.Error()})
	}

	tz, _ := h.svc.GetTimezone()
	return c.JSON(http.StatusOK, GlobalSettingsResponse{
		Timezone: tz,
	})
}
