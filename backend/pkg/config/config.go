package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds application configuration
type Config struct {
	Port        string
	Environment string
	DatabaseURL string
	LogLevel    string

	// External API configuration
	StockAPIURL     string
	StockAPIToken   string
	AlpacaAPIKey    string
	AlpacaAPISecret string
	AlpacaBaseURL   string

	// Application settings
	MaxWorkers     int
	RequestTimeout int
	CacheEnabled   bool
}

// Load reads configuration from environment variables
func Load() *Config {
	return &Config{
		Port:        getEnv("PORT", "8080"),
		Environment: getEnv("ENVIRONMENT", "development"),
		DatabaseURL: getEnv("DATABASE_URL", ""),
		LogLevel:    getEnv("LOG_LEVEL", "info"),

		StockAPIURL:     getEnv("STOCK_API_URL", ""),
		StockAPIToken:   getEnv("STOCK_API_TOKEN", ""),
		AlpacaAPIKey:    getEnv("ALPACA_API_KEY", ""),
		AlpacaAPISecret: getEnv("ALPACA_API_SECRET", ""),
		AlpacaBaseURL:   getEnv("ALPACA_BASE_URL", "https://data.alpaca.markets"),

		MaxWorkers:     getEnvInt("MAX_WORKERS", 10),
		RequestTimeout: getEnvInt("REQUEST_TIMEOUT_SECONDS", 30),
		CacheEnabled:   getEnvBool("CACHE_ENABLED", true),
	}
}

// Validate checks if required configuration is present
func (c *Config) Validate() error {
	var missing []string

	if c.DatabaseURL == "" {
		missing = append(missing, "DATABASE_URL")
	}
	if c.AlpacaAPIKey == "" {
		missing = append(missing, "ALPACA_API_KEY")
	}
	if c.AlpacaAPISecret == "" {
		missing = append(missing, "ALPACA_API_SECRET")
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	return nil
}

// IsProduction returns true if running in production environment
func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}

// IsDevelopment returns true if running in development environment
func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}

// Utility functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}
