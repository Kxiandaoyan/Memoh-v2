package discord

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/bwmarrin/discordgo"

	"github.com/Kxiandaoyan/Memoh-v2/internal/channel"
	"github.com/Kxiandaoyan/Memoh-v2/internal/channel/adapters/common"
)

// DiscordAdapter implements channel adapter interfaces for Discord.
type DiscordAdapter struct {
	logger   *slog.Logger
	mu       sync.RWMutex
	sessions map[string]*discordgo.Session
}

// NewDiscordAdapter creates a DiscordAdapter with the given logger.
func NewDiscordAdapter(log *slog.Logger) *DiscordAdapter {
	if log == nil {
		log = slog.Default()
	}
	return &DiscordAdapter{
		logger:   log.With(slog.String("adapter", "discord")),
		sessions: make(map[string]*discordgo.Session),
	}
}

func (a *DiscordAdapter) getOrCreateSession(token, configID string) (*discordgo.Session, error) {
	a.mu.RLock()
	s, ok := a.sessions[token]
	a.mu.RUnlock()
	if ok {
		return s, nil
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if s, ok := a.sessions[token]; ok {
		return s, nil
	}
	s, err := discordgo.New("Bot " + token)
	if err != nil {
		a.logger.Error("create session failed", slog.String("config_id", configID), slog.Any("error", err))
		return nil, err
	}
	a.sessions[token] = s
	return s, nil
}

// Type returns the Discord channel type.
func (a *DiscordAdapter) Type() channel.ChannelType { return Type }

// Descriptor returns the Discord channel metadata.
func (a *DiscordAdapter) Descriptor() channel.Descriptor {
	return channel.Descriptor{
		Type:        Type,
		DisplayName: "Discord",
		Capabilities: channel.ChannelCapabilities{
			Text:           true,
			Markdown:       true,
			Reply:          true,
			Streaming:      true,
			BlockStreaming: true,
		},
		ConfigSchema: channel.ConfigSchema{
			Version: 1,
			Fields: map[string]channel.FieldSchema{
				"botToken": {Type: channel.FieldSecret, Required: true, Title: "Bot Token"},
				"guildId":  {Type: channel.FieldString, Title: "Guild ID (optional)"},
			},
		},
		UserConfigSchema: channel.ConfigSchema{
			Version: 1,
			Fields: map[string]channel.FieldSchema{
				"user_id":    {Type: channel.FieldString},
				"username":   {Type: channel.FieldString},
				"channel_id": {Type: channel.FieldString},
			},
		},
		TargetSpec: channel.TargetSpec{
			Format: "channel_id | user:user_id",
			Hints: []channel.TargetHint{
				{Label: "Channel ID", Example: "1234567890"},
				{Label: "User DM", Example: "user:9876543210"},
			},
		},
	}
}

// NormalizeConfig validates and normalizes a Discord channel configuration.
func (a *DiscordAdapter) NormalizeConfig(raw map[string]any) (map[string]any, error) {
	cfg, err := parseConfig(raw)
	if err != nil {
		return nil, err
	}
	result := map[string]any{"botToken": cfg.BotToken}
	if cfg.GuildID != "" {
		result["guildId"] = cfg.GuildID
	}
	return result, nil
}

// NormalizeUserConfig validates and normalizes a Discord user-binding configuration.
func (a *DiscordAdapter) NormalizeUserConfig(raw map[string]any) (map[string]any, error) {
	cfg, err := parseUserConfig(raw)
	if err != nil {
		return nil, err
	}
	result := map[string]any{}
	if cfg.UserID != "" {
		result["user_id"] = cfg.UserID
	}
	if cfg.Username != "" {
		result["username"] = cfg.Username
	}
	if cfg.ChannelID != "" {
		result["channel_id"] = cfg.ChannelID
	}
	return result, nil
}

// NormalizeTarget normalizes a Discord delivery target string.
func (a *DiscordAdapter) NormalizeTarget(raw string) string {
	return strings.TrimSpace(raw)
}

// ResolveTarget derives a delivery target from a Discord user-binding configuration.
func (a *DiscordAdapter) ResolveTarget(userConfig map[string]any) (string, error) {
	cfg, err := parseUserConfig(userConfig)
	if err != nil {
		return "", err
	}
	if cfg.ChannelID != "" {
		return cfg.ChannelID, nil
	}
	if cfg.UserID != "" {
		return "user:" + cfg.UserID, nil
	}
	return "", fmt.Errorf("discord binding requires channel_id or user_id")
}

// MatchBinding reports whether a Discord user binding matches the given criteria.
func (a *DiscordAdapter) MatchBinding(config map[string]any, criteria channel.BindingCriteria) bool {
	cfg, err := parseUserConfig(config)
	if err != nil {
		return false
	}
	if v := strings.TrimSpace(criteria.Attribute("user_id")); v != "" && v == cfg.UserID {
		return true
	}
	if v := strings.TrimSpace(criteria.Attribute("username")); v != "" && strings.EqualFold(v, cfg.Username) {
		return true
	}
	if v := strings.TrimSpace(criteria.Attribute("channel_id")); v != "" && v == cfg.ChannelID {
		return true
	}
	if criteria.SubjectID != "" {
		if criteria.SubjectID == cfg.UserID || strings.EqualFold(criteria.SubjectID, cfg.Username) {
			return true
		}
	}
	return false
}

// BuildUserConfig constructs a Discord user-binding config from an Identity.
func (a *DiscordAdapter) BuildUserConfig(identity channel.Identity) map[string]any {
	result := map[string]any{}
	if v := strings.TrimSpace(identity.Attribute("user_id")); v != "" {
		result["user_id"] = v
	}
	if v := strings.TrimSpace(identity.Attribute("username")); v != "" {
		result["username"] = v
	}
	if v := strings.TrimSpace(identity.Attribute("channel_id")); v != "" {
		result["channel_id"] = v
	}
	return result
}

// DiscoverSelf retrieves the bot's own identity from Discord.
func (a *DiscordAdapter) DiscoverSelf(ctx context.Context, credentials map[string]any) (map[string]any, string, error) {
	cfg, err := parseConfig(credentials)
	if err != nil {
		return nil, "", err
	}
	s, err := discordgo.New("Bot " + cfg.BotToken)
	if err != nil {
		return nil, "", err
	}
	user, err := s.User("@me")
	if err != nil {
		return nil, "", fmt.Errorf("discord discover self: %w", err)
	}
	identity := map[string]any{
		"user_id":  user.ID,
		"username": user.Username,
	}
	return identity, user.ID, nil
}

// Connect starts a Discord WebSocket gateway connection and forwards messages to the handler.
func (a *DiscordAdapter) Connect(ctx context.Context, cfg channel.ChannelConfig, handler channel.InboundHandler) (channel.Connection, error) {
	a.logger.Info("start", slog.String("config_id", cfg.ID))
	discordCfg, err := parseConfig(cfg.Credentials)
	if err != nil {
		a.logger.Error("decode config failed", slog.String("config_id", cfg.ID), slog.Any("error", err))
		return nil, err
	}
	s, err := discordgo.New("Bot " + discordCfg.BotToken)
	if err != nil {
		a.logger.Error("create session failed", slog.String("config_id", cfg.ID), slog.Any("error", err))
		return nil, err
	}
	s.Identify.Intents = discordgo.IntentsGuildMessages |
		discordgo.IntentsDirectMessages |
		discordgo.IntentMessageContent |
		discordgo.IntentsGuildMessageReactions

	connCtx, cancel := context.WithCancel(ctx)

	s.AddHandler(func(_ *discordgo.Session, m *discordgo.MessageCreate) {
		if m.Author == nil || m.Author.Bot {
			return
		}
		if discordCfg.GuildID != "" && m.GuildID != "" && m.GuildID != discordCfg.GuildID {
			return
		}
		text := strings.TrimSpace(m.Content)
		if text == "" {
			return
		}
		isDM := m.GuildID == ""
		chatType := "channel"
		if isDM {
			chatType = "dm"
		}
		isMentioned := false
		for _, u := range m.Mentions {
			if u.ID == s.State.User.ID {
				isMentioned = true
				break
			}
		}
		var replyRef *channel.ReplyRef
		if m.MessageReference != nil && m.MessageReference.MessageID != "" {
			replyRef = &channel.ReplyRef{
				MessageID: m.MessageReference.MessageID,
				Target:    m.ChannelID,
			}
		}
		isReplyToBot := false
		if m.ReferencedMessage != nil && m.ReferencedMessage.Author != nil {
			isReplyToBot = m.ReferencedMessage.Author.ID == s.State.User.ID
		}

		msg := channel.InboundMessage{
			Channel: Type,
			Message: channel.Message{
				ID:     m.ID,
				Format: channel.MessageFormatMarkdown,
				Text:   text,
				Reply:  replyRef,
			},
			BotID:       cfg.BotID,
			ReplyTarget: m.ChannelID,
			Sender: channel.Identity{
				SubjectID:   m.Author.ID,
				DisplayName: m.Author.Username,
				Attributes: map[string]string{
					"user_id":    m.Author.ID,
					"username":   m.Author.Username,
					"channel_id": m.ChannelID,
				},
			},
			Conversation: channel.Conversation{
				ID:   m.ChannelID,
				Type: chatType,
			},
			ReceivedAt: m.Timestamp,
			Source:     "discord",
			Metadata: map[string]any{
				"is_mentioned":    isMentioned,
				"is_reply_to_bot": isReplyToBot,
				"is_dm":           isDM,
			},
		}
		a.logger.Info("inbound received",
			slog.String("config_id", cfg.ID),
			slog.String("chat_type", chatType),
			slog.String("channel_id", m.ChannelID),
			slog.String("user_id", m.Author.ID),
			slog.String("username", m.Author.Username),
			slog.String("text", common.SummarizeText(text)),
		)
		go func() {
			if err := handler(connCtx, cfg, msg); err != nil {
				a.logger.Error("handle inbound failed", slog.String("config_id", cfg.ID), slog.Any("error", err))
			}
		}()
	})

	if err := s.Open(); err != nil {
		cancel()
		a.logger.Error("open session failed", slog.String("config_id", cfg.ID), slog.Any("error", err))
		return nil, err
	}

	stop := func(_ context.Context) error {
		a.logger.Info("stop", slog.String("config_id", cfg.ID))
		cancel()
		return s.Close()
	}
	return channel.NewConnection(cfg, stop), nil
}

// resolveChannelID resolves a target to a Discord channel ID.
// Targets prefixed with "user:" create a DM channel first.
func (a *DiscordAdapter) resolveChannelID(s *discordgo.Session, target string) (string, error) {
	if strings.HasPrefix(target, "user:") {
		userID := strings.TrimPrefix(target, "user:")
		ch, err := s.UserChannelCreate(userID)
		if err != nil {
			return "", fmt.Errorf("discord create DM channel: %w", err)
		}
		return ch.ID, nil
	}
	return target, nil
}

// Send delivers an outbound message to Discord.
func (a *DiscordAdapter) Send(ctx context.Context, cfg channel.ChannelConfig, msg channel.OutboundMessage) error {
	discordCfg, err := parseConfig(cfg.Credentials)
	if err != nil {
		return err
	}
	to := strings.TrimSpace(msg.Target)
	if to == "" {
		return fmt.Errorf("discord target is required")
	}
	s, err := a.getOrCreateSession(discordCfg.BotToken, cfg.ID)
	if err != nil {
		return err
	}
	if msg.Message.IsEmpty() {
		return fmt.Errorf("message is required")
	}
	channelID, err := a.resolveChannelID(s, to)
	if err != nil {
		return err
	}
	text := formatDiscordOutput(msg.Message.PlainText())
	msgSend := &discordgo.MessageSend{Content: text}
	if msg.Message.Reply != nil && msg.Message.Reply.MessageID != "" {
		msgSend.Reference = &discordgo.MessageReference{MessageID: msg.Message.Reply.MessageID}
	}
	_, err = s.ChannelMessageSendComplex(channelID, msgSend)
	return err
}

// OpenStream opens a Discord streaming session.
func (a *DiscordAdapter) OpenStream(ctx context.Context, cfg channel.ChannelConfig, target string, opts channel.StreamOptions) (channel.OutboundStream, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return nil, fmt.Errorf("discord target is required")
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	return &discordOutboundStream{
		adapter: a,
		cfg:     cfg,
		target:  target,
		reply:   opts.Reply,
	}, nil
}

// React adds an emoji reaction to a Discord message.
func (a *DiscordAdapter) React(ctx context.Context, cfg channel.ChannelConfig, target string, messageID string, emoji string) error {
	discordCfg, err := parseConfig(cfg.Credentials)
	if err != nil {
		return err
	}
	s, err := a.getOrCreateSession(discordCfg.BotToken, cfg.ID)
	if err != nil {
		return err
	}
	return s.MessageReactionAdd(target, messageID, emoji)
}

// Unreact removes the bot's emoji reaction from a Discord message.
func (a *DiscordAdapter) Unreact(ctx context.Context, cfg channel.ChannelConfig, target string, messageID string, emoji string) error {
	discordCfg, err := parseConfig(cfg.Credentials)
	if err != nil {
		return err
	}
	s, err := a.getOrCreateSession(discordCfg.BotToken, cfg.ID)
	if err != nil {
		return err
	}
	return s.MessageReactionRemove(target, messageID, emoji, "@me")
}

// ProcessingStarted sends a typing indicator to Discord.
func (a *DiscordAdapter) ProcessingStarted(ctx context.Context, cfg channel.ChannelConfig, msg channel.InboundMessage, info channel.ProcessingStatusInfo) (channel.ProcessingStatusHandle, error) {
	channelID := strings.TrimSpace(info.ReplyTarget)
	if channelID == "" {
		return channel.ProcessingStatusHandle{}, nil
	}
	discordCfg, err := parseConfig(cfg.Credentials)
	if err != nil {
		return channel.ProcessingStatusHandle{}, err
	}
	s, err := a.getOrCreateSession(discordCfg.BotToken, cfg.ID)
	if err != nil {
		return channel.ProcessingStatusHandle{}, err
	}
	if err := s.ChannelTyping(channelID); err != nil {
		a.logger.Warn("send typing failed", slog.String("config_id", cfg.ID), slog.Any("error", err))
	}
	return channel.ProcessingStatusHandle{}, nil
}

// ProcessingCompleted is a no-op for Discord.
func (a *DiscordAdapter) ProcessingCompleted(_ context.Context, _ channel.ChannelConfig, _ channel.InboundMessage, _ channel.ProcessingStatusInfo, _ channel.ProcessingStatusHandle) error {
	return nil
}

// ProcessingFailed is a no-op for Discord.
func (a *DiscordAdapter) ProcessingFailed(_ context.Context, _ channel.ChannelConfig, _ channel.InboundMessage, _ channel.ProcessingStatusInfo, _ channel.ProcessingStatusHandle, _ error) error {
	return nil
}