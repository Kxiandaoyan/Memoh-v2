package discord

import (
	"fmt"
	"log/slog"
)

// slogDiscordLogger adapts slog.Logger to discordgo's logging interface.
type slogDiscordLogger struct {
	log *slog.Logger
}

func (s *slogDiscordLogger) Println(v ...interface{}) {
	s.log.Warn(fmt.Sprint(v...))
}

func (s *slogDiscordLogger) Printf(format string, v ...interface{}) {
	s.log.Warn(fmt.Sprintf(format, v...))
}
