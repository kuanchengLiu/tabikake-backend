package config

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	AnthropicAPIKey         string
	NotionIntegrationToken  string
	NotionOAuthClientID     string
	NotionOAuthClientSecret string
	NotionOAuthRedirectURI  string
	NotionRootPageID        string
	JWTSecret               string
	TokenEncryptKey         []byte // 32-byte key for AES-256-GCM
	Port                    string
	FrontendURL             string
	SQLitePath              string
}

// Load reads configuration from environment variables and returns a Config.
func Load() (*Config, error) {
	env := func(key string) string {
		return strings.TrimSpace(os.Getenv(key))
	}

	encKeyRaw := env("TOKEN_ENCRYPT_KEY")
	var encKey []byte
	if encKeyRaw != "" {
		var err error
		encKey, err = base64.StdEncoding.DecodeString(encKeyRaw)
		if err != nil {
			return nil, fmt.Errorf("TOKEN_ENCRYPT_KEY must be base64-encoded: %w", err)
		}
		if len(encKey) != 32 {
			return nil, fmt.Errorf("TOKEN_ENCRYPT_KEY must decode to exactly 32 bytes (got %d)", len(encKey))
		}
	}

	cfg := &Config{
		AnthropicAPIKey:         env("ANTHROPIC_API_KEY"),
		NotionIntegrationToken:  env("NOTION_INTEGRATION_TOKEN"),
		NotionOAuthClientID:     env("NOTION_OAUTH_CLIENT_ID"),
		NotionOAuthClientSecret: env("NOTION_OAUTH_CLIENT_SECRET"),
		NotionOAuthRedirectURI:  env("NOTION_OAUTH_REDIRECT_URI"),
		NotionRootPageID:        env("NOTION_ROOT_PAGE_ID"),
		JWTSecret:               env("JWT_SECRET"),
		TokenEncryptKey:         encKey,
		Port:                    env("PORT"),
		FrontendURL:             env("FRONTEND_URL"),
		SQLitePath:              env("SQLITE_PATH"),
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	if cfg.Port == "" {
		cfg.Port = "8080"
	}
	if cfg.SQLitePath == "" {
		cfg.SQLitePath = "tabikake.db"
	}
	if cfg.FrontendURL == "" {
		cfg.FrontendURL = "http://localhost:3000"
	}

	return cfg, nil
}

func (c *Config) validate() error {
	required := map[string]string{
		"ANTHROPIC_API_KEY":          c.AnthropicAPIKey,
		"NOTION_INTEGRATION_TOKEN":   c.NotionIntegrationToken,
		"NOTION_OAUTH_CLIENT_ID":     c.NotionOAuthClientID,
		"NOTION_OAUTH_CLIENT_SECRET": c.NotionOAuthClientSecret,
		"NOTION_OAUTH_REDIRECT_URI":  c.NotionOAuthRedirectURI,
		"NOTION_ROOT_PAGE_ID":        c.NotionRootPageID,
		"JWT_SECRET":                 c.JWTSecret,
		"TOKEN_ENCRYPT_KEY":          string(c.TokenEncryptKey),
	}
	for key, val := range required {
		if val == "" {
			return fmt.Errorf("missing required environment variable: %s", key)
		}
	}
	return nil
}
