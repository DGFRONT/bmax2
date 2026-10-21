// Package config loads runtime settings from environment variables.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	// Addr is the listen address of the HTTP server, e.g. ":8080".
	Addr string
	// DatabasePath is the SQLite database file.
	DatabasePath string
	// BotToken is the token of the MAX bot the mini app is attached to.
	// It is the key for initData signature validation.
	BotToken string
	// InitDataTTL limits how old a signed initData may be.
	InitDataTTL time.Duration
	// AdminToken enables the admin API when non-empty.
	AdminToken string
	// SeedDir, when set, is imported into an empty database on startup.
	SeedDir string
	// LogLevel is one of debug, info, warn, error.
	LogLevel string
	// BotPolling makes the server answer the bot chat with a welcome message.
	BotPolling bool
	// BotAPIURL is the MAX Bot API endpoint.
	BotAPIURL string
}

// Load reads the configuration. Variables used by every command are required
// here; the serve command checks the rest with RequireServe.
func Load() (Config, error) {
	c := Config{
		Addr:         env("ADDR", ":8080"),
		DatabasePath: env("DATABASE_PATH", "data/bmax.db"),
		BotToken:     os.Getenv("BOT_TOKEN"),
		AdminToken:   os.Getenv("ADMIN_TOKEN"),
		SeedDir:      os.Getenv("SEED_DIR"),
		LogLevel:     env("LOG_LEVEL", "info"),
		BotAPIURL:    env("BOT_API_URL", "https://platform-api.max.ru"),
	}

	polling, err := strconv.ParseBool(env("BOT_POLLING", "true"))
	if err != nil {
		return Config{}, fmt.Errorf("BOT_POLLING: invalid boolean %q", os.Getenv("BOT_POLLING"))
	}
	c.BotPolling = polling

	ttl, err := time.ParseDuration(env("INIT_DATA_TTL", "1h"))
	if err != nil || ttl <= 0 {
		return Config{}, fmt.Errorf("INIT_DATA_TTL: invalid duration %q", os.Getenv("INIT_DATA_TTL"))
	}
	c.InitDataTTL = ttl

	if c.AdminToken != "" && len(c.AdminToken) < 32 {
		return Config{}, errors.New("ADMIN_TOKEN must be at least 32 characters")
	}
	return c, nil
}

// RequireServe validates settings needed only by the HTTP server.
func (c Config) RequireServe() error {
	if c.BotToken == "" {
		return errors.New("BOT_TOKEN is required")
	}
	return nil
}

// Port returns the numeric port of Addr for local health checks.
func (c Config) Port() (int, error) {
	for i := len(c.Addr) - 1; i >= 0; i-- {
		if c.Addr[i] == ':' {
			return strconv.Atoi(c.Addr[i+1:])
		}
	}
	return 0, fmt.Errorf("ADDR %q has no port", c.Addr)
}

func env(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}
