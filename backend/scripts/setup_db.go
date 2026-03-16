package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	_ "modernc.org/sqlite"
)

func main() {
	// Load .env file
	godotenv.Load("../.env")
	godotenv.Load()

	dbURL := os.Getenv("DATABASE_URL")
	fmt.Printf("Using Database: %s\n", dbURL)

	driverName := "postgres"
	dataSourceName := dbURL

	if strings.HasPrefix(dbURL, "sqlite") {
		driverName = "sqlite"
		dataSourceName = strings.TrimPrefix(dbURL, "sqlite://")
	}

	db, err := sql.Open(driverName, dataSourceName)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// 1. Create Sample Ratings
	tickers := []string{"AAPL", "MSFT", "GOOGL", "AMZN", "TSLA", "META", "NVDA", "BRK.B", "JPM", "V"}
	
	fmt.Println("🚀 Injecting sample data into cloud...")

	for _, ticker := range tickers {
		ratingID := uuid.New().String()
		now := time.Now()
		
		query := `INSERT INTO stock_ratings (rating_id, ticker, company, brokerage, action, rating_to, target_to, time, created_at) 
				  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
		
		if driverName == "sqlite" {
			query = strings.ReplaceAll(query, "$", "?")
		}

		_, err := db.Exec(query, 
			ratingID, 
			ticker, 
			ticker + " Corp", 
			"Manual Injection", 
			"Upgraded by", 
			"Buy", 
			250.50, 
			now, 
			now)
		
		if err != nil {
			fmt.Printf("Error inserting %s: %v\n", ticker, err)
		} else {
			fmt.Printf("✅ Injected %s\n", ticker)
		}
	}

	fmt.Println("🎉 Cloud Database injection complete!")
}
