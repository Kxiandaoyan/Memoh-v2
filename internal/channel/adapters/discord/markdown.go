package discord

import (
	"strings"
	"unicode/utf8"
)

const discordMaxMessageLength = 2000

// formatDiscordOutput returns text as-is since Discord natively supports markdown.
func formatDiscordOutput(text string) string {
	return truncateDiscordText(strings.ToValidUTF8(text, ""))
}

// truncateDiscordText truncates text to 2000 chars on a rune boundary.
func truncateDiscordText(text string) string {
	if len(text) <= discordMaxMessageLength {
		return text
	}
	const suffix = "..."
	limit := discordMaxMessageLength - len(suffix)
	for limit > 0 && !utf8.RuneStart(text[limit]) {
		limit--
	}
	return text[:limit] + suffix
}
