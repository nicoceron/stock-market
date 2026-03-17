// Package main provides the AWS Lambda entry point for the Stock Analyzer backend.
//
// This Lambda function serves multiple purposes based on the FUNCTION_TYPE environment variable:
//   - "api" (default): Handles HTTP requests via API Gateway using Gin router
//   - "ingestion": Performs scheduled data ingestion from external APIs
//   - "scheduler": Executes maintenance and cleanup tasks
//
// The function is designed to be deployed as a single Lambda with different configurations
// for different use cases, allowing for cost optimization and simplified deployment.
//
// Environment Variables Required:
//   - DATABASE_URL: PostgreSQL/CockroachDB connection string
//   - ALPACA_API_KEY: Alpaca API key for market data
//   - ALPACA_API_SECRET: Alpaca API secret
//   - STOCK_API_TOKEN: External stock ratings API token
//   - FUNCTION_TYPE: Optional, defaults to "api"
//
// AWS Lambda Configuration:
//   - Runtime: Go 1.x
//   - Memory: 512MB (API), 1024MB (ingestion), 256MB (scheduler)
//   - Timeout: 30s (API), 15min (ingestion), 5min (scheduler)
//   - Environment: Set via Terraform or AWS Console
package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/awslabs/aws-lambda-go-api-proxy/gin"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	_ "modernc.org/sqlite"

	"stock-analyzer/internal/alpaca"

	"stock-analyzer/internal/api"
	"stock-analyzer/internal/domain"
	"stock-analyzer/internal/ingestion"
	"stock-analyzer/internal/recommendation"
	"stock-analyzer/internal/storage"
	"stock-analyzer/pkg/config"

	"github.com/joho/godotenv"
)

var (
	// ginLambda is the Gin adapter for AWS Lambda, initialized once during cold start
	ginLambda *ginadapter.GinLambda

	// db is the database connection pool, shared across Lambda invocations
	db *sql.DB

	// Services initialized once during cold start
	stockRepo         domain.StockRepository
	ingestionSvc      domain.IngestionService
	recommendationSvc domain.RecommendationService
	alpacaSvc         domain.AlpacaService
)

// init performs one-time initialization during Lambda cold start.
// This includes database connection setup, service initialization,
// and router configuration. The initialization is expensive but only
// happens once per Lambda container lifecycle.
func init() {
	// Set Gin to release mode in Lambda to reduce log verbosity
	gin.SetMode(gin.ReleaseMode)

	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// Load configuration from environment variables
	cfg := config.Load()

	// Initialize database connection pool
	// The connection will be reused across Lambda invocations
	var err error
	driverName := "postgres"
	dataSourceName := cfg.DatabaseURL

	if strings.HasPrefix(cfg.DatabaseURL, "sqlite") {
		driverName = "sqlite"
		dataSourceName = strings.TrimPrefix(cfg.DatabaseURL, "sqlite://")
	}

	db, err = sql.Open(driverName, dataSourceName)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Test database connectivity during initialization
	// This ensures we fail fast if database is unreachable
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	// Initialize business services with their dependencies
	alpacaSvc = alpaca.NewAdapter(cfg.AlpacaAPIKey, cfg.AlpacaAPISecret, cfg.AlpacaBaseURL)
	stockRepo = storage.NewPostgresRepository(db)
	ingestionSvc = ingestion.NewService(stockRepo, alpacaSvc, cfg.StockAPIURL, cfg.StockAPIToken)
	recommendationSvc = recommendation.NewService(stockRepo, alpacaSvc)

	// Setup HTTP router with all handlers and middleware
	router := api.SetupRouter(stockRepo, ingestionSvc, recommendationSvc, alpacaSvc)

	// Create Lambda adapter for Gin router
	// This allows the Gin application to handle Lambda events
	ginLambda = ginadapter.New(router)
}

// Handler is the main AWS Lambda function handler.
// It routes requests to different handlers based on the FUNCTION_TYPE environment variable.
// This allows a single Lambda deployment to serve multiple purposes with different configurations.
//
// Function Types:
//   - "api": Handles HTTP API requests via API Gateway (default)
//   - "ingestion": Performs data ingestion from external APIs
//   - "scheduler": Executes scheduled maintenance tasks
//
// The handler implements the standard AWS Lambda signature and returns
// API Gateway-compatible responses for HTTP functions.
func Handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Determine function type from environment variable
	functionType := os.Getenv("FUNCTION_TYPE")
	log.Printf("Executing function type: %s", functionType)

	switch functionType {
	case "ingestion":
		return handleIngestion(ctx)
	case "scheduler":
		return handleScheduler(ctx)
	default:
		// Default to API handler for HTTP requests
		return ginLambda.ProxyWithContext(ctx, req)
	}
}

// handleIngestion processes background data ingestion tasks.
// This function is triggered by EventBridge on a schedule (typically every 4 hours)
// to fetch fresh stock ratings data from external APIs.
//
// The function performs the following operations:
//  1. Initializes ingestion service with current configuration
//  2. Executes complete data ingestion cycle with error handling
//  3. Returns success/failure status for monitoring
//
// Expected Trigger: EventBridge scheduled event
// Timeout: 15 minutes (configurable via Lambda settings)
// Memory: 1024MB (higher memory for batch processing)
func handleIngestion(ctx context.Context) (events.APIGatewayProxyResponse, error) {
	log.Println("Starting data ingestion...")

	// Perform complete data ingestion cycle
	// This includes fetching, transforming, and storing data
	if err := ingestionSvc.IngestAllData(ctx); err != nil {
		log.Printf("Ingestion failed: %v", err)
		return api.NewErrorResponse(500, "Ingestion failed"), nil
	}

	log.Println("Data ingestion completed successfully")
	return api.NewSuccessResponse(200, map[string]string{"message": "Ingestion completed successfully"}), nil
}

// handleScheduler processes scheduled maintenance and cleanup tasks.
// This function runs daily to perform housekeeping operations that
// keep the system running efficiently.
//
// Potential tasks include:
//   - Cleaning up old data beyond retention period
//   - Generating daily reports and analytics
//   - Updating cached recommendation data
//   - Database maintenance and optimization
//
// Expected Trigger: EventBridge scheduled event (daily)
// Timeout: 5 minutes
// Memory: 256MB (lightweight maintenance tasks)
func handleScheduler(ctx context.Context) (events.APIGatewayProxyResponse, error) {
	log.Println("Running scheduled tasks...")

	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)

	deletedCount, err := stockRepo.DeleteOldEnrichedData(ctx, thirtyDaysAgo)
	if err != nil {
		log.Printf("Scheduler failed to clean up old enriched data: %v", err)
		return api.NewErrorResponse(500, "Scheduler task failed during data cleanup"), nil
	}

	log.Printf("Scheduler successfully cleaned up %d old enriched data records.", deletedCount)
	response := map[string]interface{}{
		"message":         "Scheduled tasks completed successfully",
		"cleaned_records": deletedCount,
	}

	return api.NewSuccessResponse(200, response), nil
}

// main is the Lambda entry point that starts the AWS Lambda runtime.
func main() {
	// Check if running in Lambda
	if os.Getenv("LAMBDA_TASK_ROOT") == "" && os.Getenv("AWS_LAMBDA_FUNCTION_NAME") == "" {
		// Local execution
		log.Println("🚀 Starting local server...")
		
		// Load config to get port
		cfg := config.Load()
		
		// Setup router (re-initializing because init() might have failed if env vars were missing)
		router := api.SetupRouter(stockRepo, ingestionSvc, recommendationSvc, alpacaSvc)
		
		log.Printf("📡 Listening on port %s", cfg.Port)
		if err := router.Run(":" + cfg.Port); err != nil {
			log.Fatalf("Failed to start local server: %v", err)
		}
	} else {
		// Lambda execution
		lambda.Start(Handler)
	}
}
