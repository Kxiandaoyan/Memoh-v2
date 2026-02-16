package local

import (
	"context"
	"fmt"
	"strings"

	"github.com/Kxiandaoyan/Memoh-v2/internal/channel"
)

// WebAdapter implements channel.Sender for the local Web channel.
type WebAdapter struct {
	hub *RouteHub
}

// NewWebAdapter creates a WebAdapter backed by the given route hub.
func NewWebAdapter(hub *RouteHub) *WebAdapter {
	return &WebAdapter{hub: hub}
}

// Type returns the Web channel type.
func (a *WebAdapter) Type() channel.ChannelType {
	return WebType
}

// Descriptor returns the Web channel metadata.
func (a *WebAdapter) Descriptor() channel.Descriptor {
	return channel.Descriptor{
		Type:        WebType,
		DisplayName: "Web",
		Configless:  true,
		Capabilities: channel.ChannelCapabilities{
			Text:           true,
			Reply:          true,
			Attachments:    true,
			Streaming:      true,
			BlockStreaming: true,
		},
		TargetSpec: channel.TargetSpec{
			Format: "bot_id",
			Hints: []channel.TargetHint{
				{Label: "Bot ID", Example: "bot_123"},
			},
		},
	}
}

// Send publishes an outbound message to the Web route hub.
func (a *WebAdapter) Send(ctx context.Context, cfg channel.ChannelConfig, msg channel.OutboundMessage) error {
	if a.hub == nil {
		return fmt.Errorf("web hub not configured")
	}
	target := strings.TrimSpace(msg.Target)
	if target == "" {
		return fmt.Errorf("web target is required")
	}
	if msg.Message.IsEmpty() {
		return fmt.Errorf("message is required")
	}
	a.hub.Publish(target, msg)
	return nil
}

// OpenStream opens a local stream session bound to the target route.
func (a *WebAdapter) OpenStream(ctx context.Context, cfg channel.ChannelConfig, target string, opts channel.StreamOptions) (channel.OutboundStream, error) {
	if a.hub == nil {
		return nil, fmt.Errorf("web hub not configured")
	}
	target = strings.TrimSpace(target)
	if target == "" {
		return nil, fmt.Errorf("web target is required")
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	return newLocalOutboundStream(a.hub, target), nil
}
