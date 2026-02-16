package telegram

import (
	"fmt"
	"log/slog"
)

// slogBotLogger adapts slog.Logger to tgbotapi.BotLogger so library logs go through slog.
type slogBotLogger struct {
	log *slog.Logger
}

func (s *slogBotLogger) Println(v ...interface{}) {
	s.log.Warn(fmt.Sprint(v...))
}

func (s *slogBotLogger) Printf(format string, v ...interface{}) {
	s.log.Warn(fmt.Sprintf(format, v...))
}
