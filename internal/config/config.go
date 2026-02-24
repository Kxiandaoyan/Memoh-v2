package config

import (
	"fmt"
	"os"
	"time"

	"github.com/BurntSushi/toml"
)

const (
	DefaultConfigPath       = "config.toml"
	DefaultHTTPAddr         = ":8080"
	DefaultNamespace        = "default"
	DefaultSocketPath       = "/run/containerd/containerd.sock"
	DefaultMCPImage         = "docker.io/library/memoh-mcp:latest"
	DefaultDataRoot         = "data"
	DefaultDataMount        = "/data"
	DefaultJWTExpiresIn     = "24h"
	DefaultPGHost           = "127.0.0.1"
	DefaultPGPort           = 5432
	DefaultPGUser           = "postgres"
	DefaultPGDatabase       = "memoh"
	DefaultPGSSLMode        = "disable"
	DefaultQdrantURL        = "http://127.0.0.1:6334"
	DefaultQdrantCollection = "memory"
)

type Config struct {
	Log          LogConfig          `toml:"log"`
	Server       ServerConfig       `toml:"server"`
	Admin        AdminConfig        `toml:"admin"`
	Auth         AuthConfig         `toml:"auth"`
	Containerd   ContainerdConfig   `toml:"containerd"`
	MCP          MCPConfig          `toml:"mcp"`
	Postgres     PostgresConfig     `toml:"postgres"`
	Qdrant       QdrantConfig       `toml:"qdrant"`
	AgentGateway AgentGatewayConfig `toml:"agent_gateway"`
	Smithery     SmitheryConfig     `toml:"smithery"`
}

type LogConfig struct {
	Level  string `toml:"level"`
	Format string `toml:"format"`
}

type ServerConfig struct {
	Addr     string `toml:"addr"`
	Timezone string `toml:"timezone"`
}

type AdminConfig struct {
	Username string `toml:"username"`
	Password string `toml:"password"`
	Email    string `toml:"email"`
}

type AuthConfig struct {
	JWTSecret    string `toml:"jwt_secret"`
	JWTExpiresIn string `toml:"jwt_expires_in"`
}

type ContainerdConfig struct {
	SocketPath string `toml:"socket_path"`
	Namespace  string `toml:"namespace"`
}

type MCPConfig struct {
	Image              string `toml:"image"`
	Snapshotter        string `toml:"snapshotter"`
	DataRoot           string `toml:"data_root"`
	DataMount          string `toml:"data_mount"`
	IdleTimeoutMinutes int    `toml:"idle_timeout_minutes"` // 0 = disabled
}

type PostgresConfig struct {
	Host     string `toml:"host"`
	Port     int    `toml:"port"`
	User     string `toml:"user"`
	Password string `toml:"password"`
	Database string `toml:"database"`
	SSLMode  string `toml:"sslmode"`
}

type QdrantConfig struct {
	BaseURL        string `toml:"base_url"`
	APIKey         string `toml:"api_key"`
	Collection     string `toml:"collection"`
	TimeoutSeconds int    `toml:"timeout_seconds"`
}

type AgentGatewayConfig struct {
	Host string `toml:"host"`
	Port int    `toml:"port"`
	// PublicHost is the hostname other services use to reach the Agent Gateway.
	// In Docker this is the service DNS name (e.g. "memoh-agent").
	// When empty, falls back to Host (with 0.0.0.0 replaced by 127.0.0.1).
	PublicHost string `toml:"public_host"`
	// TimeoutSeconds is the HTTP client timeout for gateway requests.
	// Defaults to 600 seconds when unset or zero.
	TimeoutSeconds int `toml:"timeout_seconds"`
}

type SmitheryConfig struct {
	APIKey string `toml:"api_key"`
}

// BaseURL returns the HTTP URL that the Server should use to connect to the Agent Gateway.
func (c AgentGatewayConfig) BaseURL() string {
	host := c.PublicHost
	if host == "" {
		host = c.Host
	}
	if host == "" || host == "0.0.0.0" {
		host = "127.0.0.1"
	}
	port := c.Port
	if port == 0 {
		port = 8081
	}
	return "http://" + host + ":" + fmt.Sprint(port)
}

// GatewayTimeout returns the HTTP client timeout for gateway requests.
func (c AgentGatewayConfig) GatewayTimeout() time.Duration {
	if c.TimeoutSeconds > 0 {
		return time.Duration(c.TimeoutSeconds) * time.Second
	}
	return 600 * time.Second
}

func Load(path string) (Config, error) {
	cfg := Config{
		Log: LogConfig{
			Level:  "info",
			Format: "text",
		},
		Server: ServerConfig{
			Addr: DefaultHTTPAddr,
		},
		Admin: AdminConfig{
			Username: "admin",
			Password: "change-your-password-here",
			Email:    "you@example.com",
		},
		Auth: AuthConfig{
			JWTExpiresIn: DefaultJWTExpiresIn,
		},
		Containerd: ContainerdConfig{
			SocketPath: DefaultSocketPath,
			Namespace:  DefaultNamespace,
		},
		MCP: MCPConfig{
			Image:     DefaultMCPImage,
			DataRoot:  DefaultDataRoot,
			DataMount: DefaultDataMount,
		},
		Postgres: PostgresConfig{
			Host:     DefaultPGHost,
			Port:     DefaultPGPort,
			User:     DefaultPGUser,
			Database: DefaultPGDatabase,
			SSLMode:  DefaultPGSSLMode,
		},
		Qdrant: QdrantConfig{
			BaseURL:    DefaultQdrantURL,
			Collection: DefaultQdrantCollection,
		},
		AgentGateway: AgentGatewayConfig{
			Host: "127.0.0.1",
			Port: 8081,
		},
	}

	if path == "" {
		path = DefaultConfigPath
	}

	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}

	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return cfg, err
	}

	return cfg, nil
}
