package discord

import (
	"fmt"
	"strings"

	"github.com/Kxiandaoyan/Memoh-v2/internal/channel"
)

// Config holds Discord bot credentials.
type Config struct {
	BotToken string
	GuildID  string // optional, restrict to single guild
}

// UserConfig holds identifiers for targeting a Discord user or channel.
type UserConfig struct {
	UserID    string
	Username  string
	ChannelID string
}

func parseConfig(raw map[string]any) (Config, error) {
	token := strings.TrimSpace(channel.ReadString(raw, "botToken", "bot_token"))
	if token == "" {
		return Config{}, fmt.Errorf("discord botToken is required")
	}
	guildID := strings.TrimSpace(channel.ReadString(raw, "guildId", "guild_id"))
	return Config{BotToken: token, GuildID: guildID}, nil
}

func parseUserConfig(raw map[string]any) (UserConfig, error) {
	userID := strings.TrimSpace(channel.ReadString(raw, "userId", "user_id"))
	username := strings.TrimSpace(channel.ReadString(raw, "username"))
	channelID := strings.TrimSpace(channel.ReadString(raw, "channelId", "channel_id"))
	if userID == "" && username == "" && channelID == "" {
		return UserConfig{}, fmt.Errorf("discord user config requires user_id, username, or channel_id")
	}
	return UserConfig{UserID: userID, Username: username, ChannelID: channelID}, nil
}
