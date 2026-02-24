package handlers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/Kxiandaoyan/Memoh-v2/internal/channel"
	"github.com/Kxiandaoyan/Memoh-v2/internal/db/sqlc"
	"github.com/Kxiandaoyan/Memoh-v2/internal/preauth"
)

// PendingReply represents a completed AI reply waiting to be polled by Sentinel.
type PendingReply struct {
	TaskID  string    `json:"task_id"`
	Reply   string    `json:"reply"`
	Sender  string    `json:"sender"`
	ChatID  string    `json:"chat_id"`
	Created time.Time `json:"-"`
}

var (
	pendingReplies sync.Map  // key=botID, value=[]PendingReply
	pendingMu      sync.Mutex
	cleanupOnce    sync.Once
)

func startCleanupLoop() {
	cleanupOnce.Do(func() {
		go func() {
			ticker := time.NewTicker(1 * time.Minute)
			defer ticker.Stop()
			for range ticker.C {
				cutoff := time.Now().Add(-5 * time.Minute)
				pendingReplies.Range(func(key, value any) bool {
					pendingMu.Lock()
					replies := value.([]PendingReply)
					filtered := replies[:0]
					for _, r := range replies {
						if r.Created.After(cutoff) {
							filtered = append(filtered, r)
						}
					}
					if len(filtered) == 0 {
						pendingReplies.Delete(key)
					} else {
						pendingReplies.Store(key, filtered)
					}
					pendingMu.Unlock()
					return true
				})
			}
		}()
	})
}

// WeChatWebhookHandler handles WeChat webhook endpoints for external bot integrations.
type WeChatWebhookHandler struct {
	processor      channel.InboundProcessor
	channelService *channel.Service
	preauthService *preauth.Service
	queries        *sqlc.Queries
}

// NewWeChatWebhookHandler creates a new WeChat webhook handler.
func NewWeChatWebhookHandler(
	processor channel.InboundProcessor,
	channelService *channel.Service,
	preauthService *preauth.Service,
	queries *sqlc.Queries,
) *WeChatWebhookHandler {
	return &WeChatWebhookHandler{
		processor:      processor,
		channelService: channelService,
		preauthService: preauthService,
		queries:        queries,
	}
}

// Register registers the WeChat webhook routes.
func (h *WeChatWebhookHandler) Register(e *echo.Echo) {
	group := e.Group("/channels/wechat/webhook")
	group.POST("/:botID", h.HandleWebhook)
	group.GET("/:botID/poll", h.PollReplies)
}

// WeChatWebhookRequest represents the incoming webhook payload from WeChat bot.
type WeChatWebhookRequest struct {
	APIKey      string `json:"api_key"`
	Message     string `json:"message"`
	Sender      string `json:"sender"`
	SenderName  string `json:"sender_name"`
	ChatID      string `json:"chat_id"`
	ChatType    string `json:"chat_type"`
	MessageID   string `json:"message_id"`
}

// WeChatWebhookResponse represents the response sent back to the WeChat bot.
type WeChatWebhookResponse struct {
	Success bool   `json:"success"`
	Reply   string `json:"reply,omitempty"`
	Error   string `json:"error,omitempty"`
}

// HandleWebhook processes incoming WeChat webhook messages.
//
// @Summary WeChat Webhook Endpoint
// @Description Receives messages from WeChat bot and returns AI-generated replies
// @Tags webhook
// @Accept json
// @Produce json
// @Param botID path string true "Bot ID"
// @Param payload body WeChatWebhookRequest true "WeChat webhook payload"
// @Success 200 {object} WeChatWebhookResponse
// @Failure 400 {object} WeChatWebhookResponse
// @Failure 401 {object} WeChatWebhookResponse
// @Failure 500 {object} WeChatWebhookResponse
// @Router /channels/wechat/webhook/{botID} [post]
func (h *WeChatWebhookHandler) HandleWebhook(c echo.Context) error {
	botID := strings.TrimSpace(c.Param("botID"))
	if botID == "" {
		return c.JSON(http.StatusBadRequest, WeChatWebhookResponse{
			Success: false,
			Error:   "bot_id is required",
		})
	}

	var req WeChatWebhookRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, WeChatWebhookResponse{
			Success: false,
			Error:   "invalid request payload",
		})
	}

	// Validate required fields
	if strings.TrimSpace(req.APIKey) == "" {
		return c.JSON(http.StatusBadRequest, WeChatWebhookResponse{
			Success: false,
			Error:   "api_key is required",
		})
	}
	if strings.TrimSpace(req.Message) == "" {
		return c.JSON(http.StatusBadRequest, WeChatWebhookResponse{
			Success: false,
			Error:   "message is required",
		})
	}

	// Verify API key and get bot
	ctx := c.Request().Context()
	key, err := h.preauthService.Get(ctx, req.APIKey)
	if err != nil {
		if errors.Is(err, preauth.ErrKeyNotFound) {
			return c.JSON(http.StatusUnauthorized, WeChatWebhookResponse{
				Success: false,
				Error:   "invalid api_key",
			})
		}
		return c.JSON(http.StatusInternalServerError, WeChatWebhookResponse{
			Success: false,
			Error:   "failed to verify api_key",
		})
	}

	// Verify bot ID matches
	if key.BotID != botID {
		return c.JSON(http.StatusUnauthorized, WeChatWebhookResponse{
			Success: false,
			Error:   "api_key does not match bot_id",
		})
	}

	// Get channel configuration for this bot
	cfg, err := h.channelService.ResolveEffectiveConfig(ctx, botID, channel.ChannelType("wechat"))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, WeChatWebhookResponse{
			Success: false,
			Error:   "failed to resolve channel configuration",
		})
	}

	// Construct routing key for this conversation
	routeKey := generateWeChatRouteKey(botID, req.ChatID, req.ChatType, req.Sender)

	// Build InboundMessage
	msg := channel.InboundMessage{
		Channel: channel.ChannelType("wechat"),
		Message: channel.Message{
			ID:   req.MessageID,
			Text: req.Message,
		},
		BotID:       botID,
		ReplyTarget: routeKey,
		RouteKey:    routeKey,
		Sender: channel.Identity{
			SubjectID:   req.Sender,
			DisplayName: req.SenderName,
			Attributes: map[string]string{
				"sender_id":   req.Sender,
				"sender_name": req.SenderName,
			},
		},
		Conversation: channel.Conversation{
			ID:   req.ChatID,
			Type: normalizeWeChatChatType(req.ChatType),
			Metadata: map[string]any{
				"chat_type": req.ChatType,
			},
		},
		ReceivedAt: time.Now().UTC(),
		Source:     "wechat_webhook",
	}

	// Async mode: accept message, process in background, return task_id immediately
	startCleanupLoop()

	taskID := uuid.New().String()

	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		reply, err := h.processMessageSync(bgCtx, cfg, msg)
		if err != nil {
			slog.Error("wechat async processing failed",
				slog.String("bot_id", botID),
				slog.String("task_id", taskID),
				slog.Any("error", err),
			)
			return
		}

		pr := PendingReply{
			TaskID:  taskID,
			Reply:   reply,
			Sender:  req.Sender,
			ChatID:  req.ChatID,
			Created: time.Now(),
		}
		pendingMu.Lock()
		existing, _ := pendingReplies.Load(botID)
		var list []PendingReply
		if existing != nil {
			list = existing.([]PendingReply)
		}
		pendingReplies.Store(botID, append(list, pr))
		pendingMu.Unlock()
	}()

	return c.JSON(http.StatusOK, map[string]any{
		"success": true,
		"task_id": taskID,
	})
}

// PollReplies returns and consumes all pending replies for a bot.
func (h *WeChatWebhookHandler) PollReplies(c echo.Context) error {
	botID := strings.TrimSpace(c.Param("botID"))
	if botID == "" {
		return c.JSON(http.StatusBadRequest, map[string]any{"success": false, "error": "bot_id is required"})
	}

	apiKey := strings.TrimSpace(c.QueryParam("api_key"))
	if apiKey == "" {
		return c.JSON(http.StatusBadRequest, map[string]any{"success": false, "error": "api_key is required"})
	}

	ctx := c.Request().Context()
	key, err := h.preauthService.Get(ctx, apiKey)
	if err != nil {
		if errors.Is(err, preauth.ErrKeyNotFound) {
			return c.JSON(http.StatusUnauthorized, map[string]any{"success": false, "error": "invalid api_key"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]any{"success": false, "error": "failed to verify api_key"})
	}
	if key.BotID != botID {
		return c.JSON(http.StatusUnauthorized, map[string]any{"success": false, "error": "api_key does not match bot_id"})
	}

	// Drain all pending replies (pull = consume)
	pendingMu.Lock()
	var messages []PendingReply
	if val, ok := pendingReplies.LoadAndDelete(botID); ok {
		messages = val.([]PendingReply)
	}
	pendingMu.Unlock()

	if messages == nil {
		messages = []PendingReply{}
	}

	return c.JSON(http.StatusOK, map[string]any{
		"success":  true,
		"messages": messages,
	})
}

// processMessageSync processes the inbound message and waits for the complete AI reply.
func (h *WeChatWebhookHandler) processMessageSync(ctx context.Context, cfg channel.ChannelConfig, msg channel.InboundMessage) (string, error) {
	if h.processor == nil {
		return "", fmt.Errorf("processor not configured")
	}

	// Create a reply collector
	collector := &replyCollector{
		done:  make(chan struct{}),
		mutex: &sync.Mutex{},
	}

	// Create a custom sender that captures the final reply
	sender := &wechatSyncReplySender{
		collector: collector,
	}

	// Process the message synchronously with our custom sender
	errCh := make(chan error, 1)
	go func() {
		if err := h.processor.HandleInbound(ctx, cfg, msg, sender); err != nil {
			collector.mutex.Lock()
			collector.err = err
			collector.mutex.Unlock()
			select {
			case <-collector.done:
				// Already closed
			default:
				close(collector.done)
			}
			errCh <- err
			return
		}
		// Success: ensure done is closed even if Send() wasn't called
		select {
		case <-collector.done:
			// Already closed by Send()
		default:
			close(collector.done)
		}
		errCh <- nil
	}()

	// Wait for either completion, error, or timeout
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-collector.done:
		collector.mutex.Lock()
		defer collector.mutex.Unlock()
		if collector.err != nil {
			return "", collector.err
		}
		return collector.reply, nil
	}
}

// replyCollector accumulates the final reply message.
type replyCollector struct {
	reply string
	err   error
	done  chan struct{}
	mutex *sync.Mutex
}

// wechatSyncReplySender is a custom reply sender that captures the final message.
type wechatSyncReplySender struct {
	collector *replyCollector
}

func (s *wechatSyncReplySender) Send(ctx context.Context, msg channel.OutboundMessage) error {
	s.collector.mutex.Lock()
	defer s.collector.mutex.Unlock()

	// Extract text from the message
	s.collector.reply = msg.Message.PlainText()

	// Safe close: check if already closed
	select {
	case <-s.collector.done:
		// Already closed
	default:
		close(s.collector.done)
	}

	return nil
}

func (s *wechatSyncReplySender) OpenStream(ctx context.Context, target string, opts channel.StreamOptions) (channel.OutboundStream, error) {
	return &wechatSyncOutboundStream{
		collector: s.collector,
	}, nil
}

// wechatSyncOutboundStream captures streaming events and accumulates the final reply.
type wechatSyncOutboundStream struct {
	collector *replyCollector
}

func (s *wechatSyncOutboundStream) Push(ctx context.Context, event channel.StreamEvent) error {
	s.collector.mutex.Lock()
	defer s.collector.mutex.Unlock()

	switch event.Type {
	case channel.StreamEventDelta:
		// Accumulate deltas
		s.collector.reply += event.Delta
	case channel.StreamEventFinal:
		// Use final message if available
		if event.Final != nil && !event.Final.Message.IsEmpty() {
			s.collector.reply = event.Final.Message.PlainText()
		}
	case channel.StreamEventError:
		s.collector.err = fmt.Errorf("stream error: %s", event.Error)
	}

	return nil
}

func (s *wechatSyncOutboundStream) Close(ctx context.Context) error {
	s.collector.mutex.Lock()
	defer s.collector.mutex.Unlock()

	select {
	case <-s.collector.done:
		// Already closed
	default:
		close(s.collector.done)
	}
	return nil
}

// generateWeChatRouteKey creates a routing key for WeChat conversations.
// Format: wechat:bot_id:chat_id[:sender_id] (sender appended for group chats)
func generateWeChatRouteKey(botID, chatID, chatType, senderID string) string {
	normalized := normalizeWeChatChatType(chatType)
	if normalized == "group" {
		return fmt.Sprintf("wechat:%s:%s:%s", botID, chatID, senderID)
	}
	return fmt.Sprintf("wechat:%s:%s", botID, chatID)
}

// normalizeWeChatChatType converts WeChat chat type to standard format.
func normalizeWeChatChatType(chatType string) string {
	ct := strings.ToLower(strings.TrimSpace(chatType))
	switch ct {
	case "private", "p2p":
		return "private"
	case "group":
		return "group"
	default:
		return "private"
	}
}
