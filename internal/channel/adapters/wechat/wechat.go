package wechat

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/Kxiandaoyan/Memoh-v2/internal/channel"
)

// WeChatAdapter implements the channel.Adapter and channel.Sender interfaces for WeChat.
// WeChat uses webhook mode for receiving messages (handled by inbound service),
// so this adapter only needs to implement Send for outbound messages.
type WeChatAdapter struct {
	logger *slog.Logger
}

// NewWeChatAdapter creates a WeChatAdapter with the given logger.
func NewWeChatAdapter(log *slog.Logger) *WeChatAdapter {
	if log == nil {
		log = slog.Default()
	}
	return &WeChatAdapter{
		logger: log.With(slog.String("adapter", "wechat")),
	}
}

// Type returns the WeChat channel type.
func (a *WeChatAdapter) Type() channel.ChannelType {
	return Type
}

// Descriptor returns the WeChat channel metadata.
func (a *WeChatAdapter) Descriptor() channel.Descriptor {
	return channel.Descriptor{
		Type:        Type,
		DisplayName: "WeChat",
		Capabilities: channel.ChannelCapabilities{
			Text:        true,
			Markdown:    false, // WeChat Official Account supports limited formatting
			Reply:       false, // WeChat doesn't support quoting original messages
			Attachments: true,  // WeChat supports images, voice, video, etc.
			Media:       true,
			Streaming:   false, // WeChat doesn't support streaming updates
		},
		ConfigSchema: channel.ConfigSchema{
			Version: 1,
			Fields:  map[string]channel.FieldSchema{
				// Empty - API credentials are system-generated for webhook mode
			},
		},
		UserConfigSchema: channel.ConfigSchema{
			Version: 1,
			Fields: map[string]channel.FieldSchema{
				"openid": {
					Type:     channel.FieldString,
					Required: true,
					Title:    "OpenID",
				},
			},
		},
		TargetSpec: channel.TargetSpec{
			Format: "openid",
			Hints: []channel.TargetHint{
				{Label: "OpenID", Example: "oLVPpjqs9BhvzwPj5A-vTYAX3GLc"},
			},
		},
	}
}

// NormalizeConfig validates and normalizes a WeChat channel configuration map.
func (a *WeChatAdapter) NormalizeConfig(raw map[string]any) (map[string]any, error) {
	return normalizeConfig(raw)
}

// NormalizeUserConfig validates and normalizes a WeChat user-binding configuration map.
func (a *WeChatAdapter) NormalizeUserConfig(raw map[string]any) (map[string]any, error) {
	return normalizeUserConfig(raw)
}

// NormalizeTarget normalizes a WeChat delivery target string.
func (a *WeChatAdapter) NormalizeTarget(raw string) string {
	return normalizeTarget(raw)
}

// ResolveTarget derives a delivery target from a WeChat user-binding configuration.
func (a *WeChatAdapter) ResolveTarget(userConfig map[string]any) (string, error) {
	return resolveTarget(userConfig)
}

// MatchBinding reports whether a WeChat user binding matches the given criteria.
func (a *WeChatAdapter) MatchBinding(config map[string]any, criteria channel.BindingCriteria) bool {
	return matchBinding(config, criteria)
}

// BuildUserConfig constructs a WeChat user-binding config from an Identity.
func (a *WeChatAdapter) BuildUserConfig(identity channel.Identity) map[string]any {
	return buildUserConfig(identity)
}

// Send delivers an outbound message to WeChat.
// This will call WeChat's customer service message API to send messages to users.
func (a *WeChatAdapter) Send(ctx context.Context, cfg channel.ChannelConfig, msg channel.OutboundMessage) error {
	_, err := parseConfig(cfg.Credentials)
	if err != nil {
		if a.logger != nil {
			a.logger.Error("decode config failed", slog.String("config_id", cfg.ID), slog.Any("error", err))
		}
		return err
	}

	to := strings.TrimSpace(msg.Target)
	if to == "" {
		return fmt.Errorf("wechat target (openid) is required")
	}

	if msg.Message.IsEmpty() {
		return fmt.Errorf("message is required")
	}

	text := strings.TrimSpace(msg.Message.PlainText())
	if text == "" && len(msg.Message.Attachments) == 0 {
		return fmt.Errorf("message text or attachments are required")
	}

	// WeChat webhook mode uses synchronous reply (reply is returned in HTTP response).
	// Outbound Send is a no-op in webhook mode — the reply is captured by wechatSyncReplySender.
	if a.logger != nil {
		a.logger.Info("wechat send (webhook mode, no-op)",
			slog.String("config_id", cfg.ID),
			slog.String("target", to),
			slog.Int("text_len", len(text)),
		)
	}

	return nil
}
