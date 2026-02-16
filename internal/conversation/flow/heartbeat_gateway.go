package flow

import (
	"context"
	"fmt"

	"github.com/Kxiandaoyan/Memoh-v2/internal/heartbeat"
)

// HeartbeatGateway adapts heartbeat trigger calls to the chat Resolver.
type HeartbeatGateway struct {
	resolver *Resolver
}

// NewHeartbeatGateway creates a HeartbeatGateway backed by the given Resolver.
func NewHeartbeatGateway(resolver *Resolver) *HeartbeatGateway {
	return &HeartbeatGateway{resolver: resolver}
}

// TriggerHeartbeat delegates a heartbeat trigger to the chat Resolver.
func (g *HeartbeatGateway) TriggerHeartbeat(ctx context.Context, botID string, payload heartbeat.TriggerPayload, token string) error {
	if g == nil || g.resolver == nil {
		return fmt.Errorf("chat resolver not configured")
	}
	return g.resolver.TriggerHeartbeat(ctx, botID, payload, token)
}
