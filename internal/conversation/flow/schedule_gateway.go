package flow

import (
	"context"
	"fmt"

	"github.com/Kxiandaoyan/Memoh-v2/internal/schedule"
)

// ScheduleGateway adapts schedule trigger calls to the chat Resolver.
type ScheduleGateway struct {
	resolver *Resolver
}

// NewScheduleGateway creates a ScheduleGateway backed by the given Resolver.
func NewScheduleGateway(resolver *Resolver) *ScheduleGateway {
	return &ScheduleGateway{resolver: resolver}
}

// TriggerSchedule delegates a schedule trigger to the chat Resolver.
func (g *ScheduleGateway) TriggerSchedule(ctx context.Context, botID string, payload schedule.TriggerPayload, token string) error {
	if g == nil || g.resolver == nil {
		return fmt.Errorf("chat resolver not configured")
	}
	return g.resolver.TriggerSchedule(ctx, botID, payload, token)
}
