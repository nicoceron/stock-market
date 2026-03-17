package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	"stock-analyzer/internal/alpaca"
	"stock-analyzer/internal/ingestion"
	"stock-analyzer/internal/storage"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	_ "modernc.org/sqlite"
)

func main() {
	godotenv.Load("../.env")
	godotenv.Load()

	dbURL := os.Getenv("DATABASE_URL")
	fmt.Printf("Connecting to Database: %s\n", dbURL)

	driverName := "postgres"
	dataSourceName := dbURL
	if strings.HasPrefix(dbURL, "sqlite") {
		driverName = "sqlite"
		dataSourceName = strings.TrimPrefix(dbURL, "sqlite://")
	}

	db, err := sql.Open(driverName, dataSourceName)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create the required unique index for the ON CONFLICT clause
	fmt.Println("🔧 Ensuring unique constraint exists...")
	_, err = db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_ratings_unique ON stock_ratings(ticker, brokerage, rating_to, time)")
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		fmt.Printf("Warning: Failed to create unique index: %v\n", err)
	}

	// Clear old manual dummy data
	fmt.Println("🧹 Clearing old dummy ratings...")
	_, err = db.Exec("DELETE FROM stock_ratings")
	if err != nil {
		log.Fatalf("Failed to clear ratings: %v", err)
	}

	repo := storage.NewPostgresRepository(db)
	alpacaSvc := alpaca.NewAdapter(
		os.Getenv("ALPACA_API_KEY"),
		os.Getenv("ALPACA_API_SECRET"),
		os.Getenv("ALPACA_BASE_URL"),
	)
	
	ingestSvc := ingestion.NewService(
		repo,
		alpacaSvc,
		os.Getenv("STOCK_API_URL"),
		os.Getenv("STOCK_API_TOKEN"),
	)

	fmt.Println("🚀 Starting REAL data ingestion from Alpaca for 40+ tickers...")
	err = ingestSvc.IngestAllData(context.Background())
	if err != nil {
		log.Fatalf("Ingestion failed: %v", err)
	}

	fmt.Println("🎉 Real cloud ingestion complete!")
}
