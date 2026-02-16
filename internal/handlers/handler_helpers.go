package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/Kxiandaoyan/Memoh-v2/internal/accounts"
	"github.com/Kxiandaoyan/Memoh-v2/internal/auth"
	"github.com/Kxiandaoyan/Memoh-v2/internal/bots"
	"github.com/Kxiandaoyan/Memoh-v2/internal/identity"
)

// RequireChannelIdentityID extracts and validates the channel identity ID from the request context.
func RequireChannelIdentityID(c echo.Context) (string, error) {
	channelIdentityID, err := auth.UserIDFromContext(c)
	if err != nil {
		return "", err
	}
	if err := identity.ValidateChannelIdentityID(channelIdentityID); err != nil {
		return "", echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return channelIdentityID, nil
}

// RequireAdmin checks that the current user has admin role.
func RequireAdmin(c echo.Context, accountService *accounts.Service) error {
	channelIdentityID, err := RequireChannelIdentityID(c)
	if err != nil {
		return err
	}
	if accountService == nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "account service not configured")
	}
	isAdmin, err := accountService.IsAdmin(c.Request().Context(), channelIdentityID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to check admin status")
	}
	if !isAdmin {
		return echo.NewHTTPError(http.StatusForbidden, "admin role required")
	}
	return nil
}

// AuthorizeBotAccess validates that the given identity has access to the specified bot.
func AuthorizeBotAccess(ctx context.Context, botService *bots.Service, accountService *accounts.Service, channelIdentityID, botID string, policy bots.AccessPolicy) (bots.Bot, error) {
	if botService == nil || accountService == nil {
		return bots.Bot{}, echo.NewHTTPError(http.StatusInternalServerError, "bot services not configured")
	}
	isAdmin, err := accountService.IsAdmin(ctx, channelIdentityID)
	if err != nil {
		return bots.Bot{}, echo.NewHTTPError(http.StatusInternalServerError, "failed to check admin status")
	}
	bot, err := botService.AuthorizeAccess(ctx, channelIdentityID, botID, isAdmin, policy)
	if err != nil {
		if errors.Is(err, bots.ErrBotNotFound) {
			return bots.Bot{}, echo.NewHTTPError(http.StatusNotFound, "bot not found")
		}
		if errors.Is(err, bots.ErrBotAccessDenied) {
			return bots.Bot{}, echo.NewHTTPError(http.StatusForbidden, "bot access denied")
		}
		return bots.Bot{}, echo.NewHTTPError(http.StatusInternalServerError, "failed to authorize bot access")
	}
	return bot, nil
}
