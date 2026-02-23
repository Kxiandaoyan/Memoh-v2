package discord

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/Kxiandaoyan/Memoh-v2/internal/channel"
	"github.com/Kxiandaoyan/Memoh-v2/internal/channel/adapters/common"
)

const discordStreamEditThrottle = 1000 * time.Millisecond

type discordOutboundStream struct {
	adapter     *DiscordAdapter
	cfg         channel.ChannelConfig
	target      string
	reply       *channel.ReplyRef
	closed      atomic.Bool
	mu          sync.Mutex
	buf         strings.Builder
	reasoning   bool
	channelID   string
	streamMsgID string
	lastEdited  string
	lastEditAt  time.Time
}

func (s *discordOutboundStream) getSession() (*discordgo.Session, error) {
	cfg, err := parseConfig(s.cfg.Credentials)
	if err != nil {
		return nil, err
	}
	return s.adapter.getOrCreateSession(cfg.BotToken, s.cfg.ID)
}

func (s *discordOutboundStream) ensureStreamMessage(ctx context.Context, text string) error {
	s.mu.Lock()
	if s.streamMsgID != "" {
		s.mu.Unlock()
		return nil
	}
	sess, err := s.getSession()
	if err != nil {
		s.mu.Unlock()
		return err
	}
	channelID, err := s.adapter.resolveChannelID(sess, s.target)
	if err != nil {
		s.mu.Unlock()
		return err
	}
	if strings.TrimSpace(text) == "" {
		text = "..."
	}
	text = formatDiscordOutput(text)
	msg, err := sess.ChannelMessageSend(channelID, text)
	if err != nil {
		s.mu.Unlock()
		return err
	}
	s.channelID = channelID
	s.streamMsgID = msg.ID
	s.lastEdited = text
	s.lastEditAt = time.Now()
	s.mu.Unlock()
	return nil
}

func (s *discordOutboundStream) editStreamMessage(ctx context.Context, text string) error {
	s.mu.Lock()
	chID := s.channelID
	msgID := s.streamMsgID
	last := s.lastEdited
	lastAt := s.lastEditAt
	s.mu.Unlock()
	if msgID == "" {
		return nil
	}
	rendered := formatDiscordOutput(text)
	if rendered == last {
		return nil
	}
	if time.Since(lastAt) < discordStreamEditThrottle {
		return nil
	}
	sess, err := s.getSession()
	if err != nil {
		return err
	}
	_, err = sess.ChannelMessageEdit(chID, msgID, rendered)
	if err != nil {
		return nil // best-effort throttled edit
	}
	s.mu.Lock()
	s.lastEdited = rendered
	s.lastEditAt = time.Now()
	s.mu.Unlock()
	return nil
}

func (s *discordOutboundStream) editStreamMessageFinal(ctx context.Context, text string) error {
	s.mu.Lock()
	chID := s.channelID
	msgID := s.streamMsgID
	last := s.lastEdited
	s.mu.Unlock()
	if msgID == "" {
		return nil
	}
	rendered := formatDiscordOutput(text)
	if rendered == last {
		return nil
	}
	sess, err := s.getSession()
	if err != nil {
		return err
	}
	for attempt := range 3 {
		_, editErr := sess.ChannelMessageEdit(chID, msgID, rendered)
		if editErr == nil {
			s.mu.Lock()
			s.lastEdited = rendered
			s.lastEditAt = time.Now()
			s.mu.Unlock()
			return nil
		}
		d := time.Duration(attempt+1) * time.Second
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(d):
		}
	}
	return nil
}

func (s *discordOutboundStream) Push(ctx context.Context, event channel.StreamEvent) error {
	if s == nil || s.adapter == nil {
		return fmt.Errorf("discord stream not configured")
	}
	if s.closed.Load() {
		return fmt.Errorf("discord stream is closed")
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	switch event.Type {
	case channel.StreamEventStatus:
		return nil
	case channel.StreamEventDelta:
		return s.handleDelta(ctx, event)
	case channel.StreamEventFinal:
		return s.handleFinal(ctx, event)
	case channel.StreamEventError:
		return s.handleError(ctx, event)
	default:
		return fmt.Errorf("unsupported stream event type: %s", event.Type)
	}
}

func (s *discordOutboundStream) handleDelta(ctx context.Context, event channel.StreamEvent) error {
	if event.Delta == "" {
		return nil
	}
	isReasoning := false
	if phase, ok := event.Metadata["phase"].(string); ok && phase == "reasoning" {
		isReasoning = true
	}
	s.mu.Lock()
	if isReasoning {
		s.reasoning = true
		s.mu.Unlock()
		return nil
	}
	if s.reasoning {
		s.reasoning = false
	}
	s.buf.WriteString(event.Delta)
	content := common.StripReasoningTagsStreaming(s.buf.String())
	s.mu.Unlock()
	if content == "" {
		return nil
	}
	if err := s.ensureStreamMessage(ctx, content); err != nil {
		return err
	}
	return s.editStreamMessage(ctx, content)
}

func (s *discordOutboundStream) handleFinal(ctx context.Context, event channel.StreamEvent) error {
	if event.Final == nil || event.Final.Message.IsEmpty() {
		s.mu.Lock()
		finalText := common.StripReasoningTags(s.buf.String())
		s.mu.Unlock()
		if finalText != "" {
			if err := s.ensureStreamMessage(ctx, finalText); err != nil {
				slog.Warn("discord: ensure stream message failed", slog.Any("error", err))
			}
			if err := s.editStreamMessageFinal(ctx, finalText); err != nil {
				slog.Warn("discord: edit stream message failed", slog.Any("error", err))
			}
		}
		return nil
	}
	msg := event.Final.Message
	finalText := common.StripReasoningTags(msg.PlainText())
	s.mu.Lock()
	if finalText == "" {
		finalText = common.StripReasoningTags(s.buf.String())
	}
	s.mu.Unlock()
	if err := s.ensureStreamMessage(ctx, finalText); err != nil {
		return err
	}
	return s.editStreamMessageFinal(ctx, finalText)
}

func (s *discordOutboundStream) handleError(ctx context.Context, event channel.StreamEvent) error {
	errText := strings.TrimSpace(event.Error)
	if errText == "" {
		return nil
	}
	display := "Error: " + errText
	if err := s.ensureStreamMessage(ctx, display); err != nil {
		return err
	}
	return s.editStreamMessage(ctx, display)
}

func (s *discordOutboundStream) Close(ctx context.Context) error {
	if s == nil {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	s.closed.Store(true)
	return nil
}