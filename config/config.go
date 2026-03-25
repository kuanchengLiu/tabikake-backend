package config

import (
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
	NotionRootPageID        string // parent of all trip pages
	JWTSecret               string
	Port                    string
	NotionOAuthRedirectURI  string
	FrontendURL             string
	SQLitePath              string
}

// Load reads configuration from environment variables and returns a Config.
func Load() (*Config, error) {
	env := func(key string) string {
		return strings.TrimSpace(os.Getenv(key))
	}

	cfg := &Config{
		AnthropicAPIKey:         env("ANTHROPIC_API_KEY"),
		NotionIntegrationToken:  env("NOTION_INTEGRATION_TOKEN"),
		NotionOAuthClientID:     env("NOTION_OAUTH_CLIENT_ID"),
		NotionOAuthClientSecret: env("NOTION_OAUTH_CLIENT_SECRET"),
		NotionRootPageID:        env("NOTION_ROOT_PAGE_ID"),
		JWTSecret:               env("JWT_SECRET"),
		Port:                    env("PORT"),
		NotionOAuthRedirectURI:  env("NOTION_OAUTH_REDIRECT_URI"),
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

	return cfg, nil
}

func (c *Config) validate() error {
	required := map[string]string{
		"ANTHROPIC_API_KEY":         c.AnthropicAPIKey,
		"NOTION_INTEGRATION_TOKEN":  c.NotionIntegrationToken,
		"NOTION_OAUTH_CLIENT_ID":    c.NotionOAuthClientID,
		"NOTION_OAUTH_CLIENT_SECRET": c.NotionOAuthClientSecret,
		"NOTION_ROOT_PAGE_ID":       c.NotionRootPageID,
		"JWT_SECRET":                c.JWTSecret,
	}

	for key, val := range required {
		if val == "" {
			return fmt.Errorf("missing required environment variable: %s", key)
		}
	}
	return nil
}
