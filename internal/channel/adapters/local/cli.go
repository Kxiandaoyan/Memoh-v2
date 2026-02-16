package local

import (
	"context"
	"fmt"
	"strings"

	"github.com/Kxiandaoyan/Memoh-v2/internal/channel"
)

// CLIAdapter implements channel.Sender for the local CLI channel.
type CLIAdapter struct {
	hub *RouteHub
}

// NewCLIAdapter creates a CLIAdapter backed by the given route hub.
func NewCLIAdapter(hub *RouteHub) *CLIAdapter {
	return &CLIAdapter{hub: hub}
}

// Type returns the CLI channel type.
func (a *CLIAdapter) Type() channel.ChannelType {
	return CLIType
}

// Descriptor returns the CLI channel metadata.
func (a *CLIAdapter) Descriptor() channel.Descriptor {
	return channel.Descriptor{
		Type:        CLIType,
		DisplayName: "CLI",
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

// Send publishes an outbound message to the CLI route hub.
func (a *CLIAdapter) Send(ctx context.Context, cfg channel.ChannelConfig, msg channel.OutboundMessage) error {
	if a.hub == nil {
		return fmt.Errorf("cli hub not configured")
	}
	target := strings.TrimSpace(msg.Target)
	if target == "" {
		return fmt.Errorf("cli target is required")
	}
	if msg.Message.IsEmpty() {
		return fmt.Errorf("message is required")
	}
	a.hub.Publish(target, msg)
	return nil
}

// OpenStream opens a local stream session bound to the target route.
func (a *CLIAdapter) OpenStream(ctx context.Context, cfg channel.ChannelConfig, target string, opts channel.StreamOptions) (channel.OutboundStream, error) {
	if a.hub == nil {
		return nil, fmt.Errorf("cli hub not configured")
	}
	target = strings.TrimSpace(target)
	if target == "" {
		return nil, fmt.Errorf("cli target is required")
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	return newLocalOutboundStream(a.hub, target), nil
}
