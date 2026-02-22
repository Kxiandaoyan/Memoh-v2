package wechat

import (
	"fmt"
	"strings"

	"github.com/Kxiandaoyan/Memoh-v2/internal/channel"
)

// Config holds the WeChat credentials extracted from a channel configuration.
// Note: WeChat uses webhook mode, so the API key is system-generated.
type Config struct {
	// No user-provided credentials needed for webhook mode
}

// UserConfig holds the identifiers used to target a WeChat user.
type UserConfig struct {
	OpenID string
}

func normalizeConfig(raw map[string]any) (map[string]any, error) {
	// WeChat uses webhook mode with system-generated credentials
	// No user configuration needed
	return map[string]any{}, nil
}

func normalizeUserConfig(raw map[string]any) (map[string]any, error) {
	cfg, err := parseUserConfig(raw)
	if err != nil {
		return nil, err
	}
	result := map[string]any{}
	if cfg.OpenID != "" {
		result["openid"] = cfg.OpenID
	}
	return result, nil
}

func resolveTarget(raw map[string]any) (string, error) {
	cfg, err := parseUserConfig(raw)
	if err != nil {
		return "", err
	}
	if cfg.OpenID != "" {
		return cfg.OpenID, nil
	}
	return "", fmt.Errorf("wechat binding is incomplete: openid is required in the channel binding configuration")
}

func matchBinding(raw map[string]any, criteria channel.BindingCriteria) bool {
	cfg, err := parseUserConfig(raw)
	if err != nil {
		return false
	}
	if value := strings.TrimSpace(criteria.Attribute("openid")); value != "" && value == cfg.OpenID {
		return true
	}
	if criteria.SubjectID != "" && criteria.SubjectID == cfg.OpenID {
		return true
	}
	return false
}

func buildUserConfig(identity channel.Identity) map[string]any {
	result := map[string]any{}
	if value := strings.TrimSpace(identity.Attribute("openid")); value != "" {
		result["openid"] = value
	}
	return result
}

func parseConfig(raw map[string]any) (Config, error) {
	// No credentials needed for webhook mode
	return Config{}, nil
}

func parseUserConfig(raw map[string]any) (UserConfig, error) {
	openID := strings.TrimSpace(channel.ReadString(raw, "openid", "openId", "open_id"))
	if openID == "" {
		return UserConfig{}, fmt.Errorf("wechat user config requires openid")
	}
	return UserConfig{OpenID: openID}, nil
}

func normalizeTarget(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}
	// Remove common WeChat URL prefixes if present
	value = strings.TrimPrefix(value, "wx:")
	value = strings.TrimPrefix(value, "wechat:")
	value = strings.TrimSpace(value)
	return value
}
