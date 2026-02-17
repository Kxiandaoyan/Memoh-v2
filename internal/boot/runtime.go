package boot

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Kxiandaoyan/Memoh-v2/internal/config"
)

type RuntimeConfig struct {
	JwtSecret            string
	JwtExpiresIn         time.Duration
	ServerAddr           string
	ContainerdSocketPath string
	Timezone             *time.Location
	TimezoneName         string
}

func ProvideRuntimeConfig(cfg config.Config) (*RuntimeConfig, error) {
	if strings.TrimSpace(cfg.Auth.JWTSecret) == "" {
		return nil, errors.New("jwt secret is required")
	}

	jwtExpiresIn, err := time.ParseDuration(cfg.Auth.JWTExpiresIn)
	if err != nil {
		return nil, fmt.Errorf("invalid jwt expires in: %w", err)
	}

	tzName := strings.TrimSpace(cfg.Server.Timezone)
	if value := os.Getenv("TZ"); value != "" {
		tzName = value
	}
	if tzName == "" {
		tzName = "UTC"
	}
	loc, err := time.LoadLocation(tzName)
	if err != nil {
		return nil, fmt.Errorf("invalid timezone %q: %w", tzName, err)
	}

	ret := &RuntimeConfig{
		JwtSecret:            cfg.Auth.JWTSecret,
		JwtExpiresIn:         jwtExpiresIn,
		ServerAddr:           cfg.Server.Addr,
		ContainerdSocketPath: cfg.Containerd.SocketPath,
		Timezone:             loc,
		TimezoneName:         tzName,
	}

	if value := os.Getenv("HTTP_ADDR"); value != "" {
		ret.ServerAddr = value
	}

	if value := os.Getenv("CONTAINERD_SOCKET"); value != "" {
		ret.ContainerdSocketPath = value
	}
	return ret, nil
}
