package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/Kxiandaoyan/Memoh-v2/internal/bots"
	"github.com/Kxiandaoyan/Memoh-v2/internal/conversation"
	"github.com/Kxiandaoyan/Memoh-v2/internal/conversation/flow"
)

const maxCallDepth = 3

// AgentCallHandler handles cross-bot call_agent requests.
type AgentCallHandler struct {
	botService *bots.Service
	resolver   *flow.Resolver
	logger     *slog.Logger
}

// NewAgentCallHandler creates a new AgentCallHandler.
func NewAgentCallHandler(log *slog.Logger, botService *bots.Service, resolver *flow.Resolver) *AgentCallHandler {
	if log == nil {
		log = slog.Default()
	}
	return &AgentCallHandler{
		botService: botService,
		resolver:   resolver,
		logger:     log.With(slog.String("handler", "agent_call")),
	}
}

// Register mounts the agent-call route.
func (h *AgentCallHandler) Register(e *echo.Echo) {
	e.POST("/bots/:bot_id/agent-call", h.CallAgent)
}

// AgentCallRequest is the body for POST /bots/:bot_id/agent-call.
type AgentCallRequest struct {
	CallerBotID string `json:"caller_bot_id"`
	Message     string `json:"message"`
	Async       bool   `json:"async"`
	CallDepth   int    `json:"call_depth"`
}

// AgentCallResponse is the body returned for synchronous calls.
type AgentCallResponse struct {
	Result string `json:"result"`
	Status string `json:"status"`
}

func (h *AgentCallHandler) CallAgent(c echo.Context) error {
	_, err := RequireChannelIdentityID(c)
	if err != nil {
		return err
	}

	targetBotID := strings.TrimSpace(c.Param("bot_id"))
	if targetBotID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "bot_id (target) is required")
	}

	var req AgentCallRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if strings.TrimSpace(req.Message) == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "message is required")
	}
	if req.CallDepth >= maxCallDepth {
		return echo.NewHTTPError(http.StatusTooManyRequests, "call_depth limit exceeded")
	}

	ctx := c.Request().Context()

	// Validate target bot exists.
	targetBot, err := h.botService.Get(ctx, targetBotID)
	if err != nil {
		return echo.NewHTTPError(http.StatusForbidden, "target bot not found")
	}

	if req.Async {
		go func() {
			bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()
			if _, callErr := h.triggerTargetBot(bgCtx, targetBot, req.Message); callErr != nil {
				h.logger.Warn("async agent call failed", slog.String("target", targetBotID), slog.Any("error", callErr))
			}
		}()
		return c.JSON(http.StatusAccepted, AgentCallResponse{Status: "accepted"})
	}

	result, callErr := h.triggerTargetBot(ctx, targetBot, req.Message)
	if callErr != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, callErr.Error())
	}
	return c.JSON(http.StatusOK, AgentCallResponse{Result: result, Status: "completed"})
}

func (h *AgentCallHandler) triggerTargetBot(ctx context.Context, targetBot bots.Bot, message string) (string, error) {
	if h.resolver == nil {
		return "", echo.NewHTTPError(http.StatusInternalServerError, "resolver not configured")
	}

	chatID := targetBot.ID + "-agent-call-" + uuid.NewString()[:8]
	req := conversation.ChatRequest{
		BotID:          targetBot.ID,
		ChatID:         chatID,
		Query:          message,
		TaskType:       "agent_call",
		CurrentChannel: "web",
		Channels:       []string{"web"},
	}

	resp, err := h.resolver.Chat(ctx, req)
	if err != nil {
		return "", err
	}

	var parts []string
	for _, msg := range resp.Messages {
		if msg.Role == "assistant" {
			if t := msg.TextContent(); t != "" {
				parts = append(parts, t)
			}
		}
	}
	result := strings.TrimSpace(strings.Join(parts, "\n"))
	if result != "" && targetBot.DisplayName != "" {
		result = "【" + targetBot.DisplayName + "】" + result
	}
	return result, nil
}
