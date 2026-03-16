package main

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	"stock-analyzer/pkg/config"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	_ "modernc.org/sqlite"
)

func runMigrations(db *sql.DB, driverName string) error {
	var migrations []string

	if driverName == "sqlite" {
		migrations = []string{
			`CREATE TABLE IF NOT EXISTS stock_ratings (
				rating_id TEXT PRIMARY KEY,
				ticker VARCHAR(10) NOT NULL,
				company VARCHAR(255) NOT NULL,
				brokerage VARCHAR(255) NOT NULL,
				action VARCHAR(50) NOT NULL,
				rating_from VARCHAR(50),
				rating_to VARCHAR(50) NOT NULL,
				target_from DECIMAL(10, 2),
				target_to DECIMAL(10, 2),
				time DATETIME NOT NULL,
				created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
			)`,
			`CREATE TABLE IF NOT EXISTS enriched_stock_data (
				ticker VARCHAR(10) PRIMARY KEY,
				historical_prices TEXT,
				news_sentiment TEXT,
				updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
			)`,
			`CREATE INDEX IF NOT EXISTS idx_stock_ratings_ticker ON stock_ratings(ticker)`,
			`CREATE INDEX IF NOT EXISTS idx_stock_ratings_time ON stock_ratings(time)`,
			`CREATE INDEX IF NOT EXISTS idx_stock_ratings_ticker_time ON stock_ratings(ticker, time)`,
		}
	} else {
		migrations = []string{
			`CREATE TABLE IF NOT EXISTS stock_ratings (
				rating_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				ticker VARCHAR(10) NOT NULL,
				company VARCHAR(255) NOT NULL,
				brokerage VARCHAR(255) NOT NULL,
				action VARCHAR(50) NOT NULL,
				rating_from VARCHAR(50),
				rating_to VARCHAR(50) NOT NULL,
				target_from DECIMAL(10, 2),
				target_to DECIMAL(10, 2),
				time TIMESTAMPTZ NOT NULL,
				created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
			)`,
			`CREATE TABLE IF NOT EXISTS enriched_stock_data (
				ticker VARCHAR(10) PRIMARY KEY,
				historical_prices JSONB,
				news_sentiment JSONB,
				updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
			)`,
			`CREATE INDEX IF NOT EXISTS idx_stock_ratings_ticker ON stock_ratings(ticker)`,
			`CREATE INDEX IF NOT EXISTS idx_stock_ratings_time ON stock_ratings(time DESC)`,
			`CREATE INDEX IF NOT EXISTS idx_stock_ratings_ticker_time ON stock_ratings(ticker, time DESC)`,
		}
	}

	for i, migration := range migrations {
		log.Printf("Running migration %d...", i+1)
		if _, err := db.Exec(migration); err != nil {
			return fmt.Errorf("migration %d failed: %w", i+1, err)
		}
	}

	return nil
}

func main() {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// Load configuration
	cfg := config.Load()

	// Connect to database
	driverName := "postgres"
	dataSourceName := cfg.DatabaseURL

	if strings.HasPrefix(cfg.DatabaseURL, "sqlite") {
		driverName = "sqlite"
		dataSourceName = strings.TrimPrefix(cfg.DatabaseURL, "sqlite://")
	}

	db, err := sql.Open(driverName, dataSourceName)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	log.Println("Connected to database successfully!")

	// Run migrations
	if err := runMigrations(db, driverName); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	log.Println("Database setup completed successfully!")
}
