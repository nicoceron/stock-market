package ingestion

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"stock-analyzer/internal/domain"
	apperrors "stock-analyzer/pkg/errors"

	"github.com/google/uuid"
)

// Service implements the IngestionService interface
type Service struct {
	stockRepo domain.StockRepository
	alpacaSvc domain.AlpacaService
	apiURL    string
	apiToken  string
	client    *http.Client
}

// NewService creates a new ingestion service
func NewService(stockRepo domain.StockRepository, alpacaSvc domain.AlpacaService, apiURL, apiToken string) *Service {
	return &Service{
		stockRepo: stockRepo,
		alpacaSvc: alpacaSvc,
		apiURL:    apiURL,
		apiToken:  apiToken,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// IngestAllData fetches and stores all data from the external API
func (s *Service) IngestAllData(ctx context.Context) error {
	fmt.Println("🚀 Starting data ingestion...")

	// Attempt to fetch from the legacy API first (though we know it's likely down)
	err := s.ingestFromLegacyAPI(ctx)
	if err != nil {
		fmt.Printf("⚠️ Legacy API ingestion failed: %v. Falling back to Alpaca News...\n", err)
		return s.ingestFromAlpacaNews(ctx)
	}

	return nil
}

func (s *Service) ingestFromLegacyAPI(ctx context.Context) error {
	var nextPage *string
	totalIngested := 0

	for {
		// Fetch data from API
		apiResponse, err := s.fetchDataFromAPI(ctx, nextPage)
		if err != nil {
			return err
		}

		if len(apiResponse.Items) == 0 {
			break
		}

		// Transform API response to domain models
		ratings, err := s.transformAPIRatings(apiResponse.Items)
		if err != nil {
			return err
		}

		// Convert to pointers for the repository call
		ratingPointers := make([]*domain.StockRating, len(ratings))
		for i := range ratings {
			ratingPointers[i] = &ratings[i]
		}

		// Store ratings in batches
		insertedCount, err := s.stockRepo.CreateStockRatingsBatch(ctx, ratingPointers)
		if err != nil {
			return err
		}

		totalIngested += insertedCount
		fmt.Printf("Ingested batch of %d ratings from legacy API (total: %d)\n", insertedCount, totalIngested)

		// Check if there's more data
		if apiResponse.NextPage == nil || *apiResponse.NextPage == "" {
			break
		}

		nextPage = apiResponse.NextPage
	}
	return nil
}

func (s *Service) ingestFromAlpacaNews(ctx context.Context) error {
	tickers := []string{
		"AAPL", "MSFT", "GOOGL", "AMZN", "TSLA", "META", "NVDA", "BRK.B", "JPM", "V",
		"UNH", "LLY", "XOM", "AVGO", "HD", "PG", "MA", "COST", "ABBV", "JNJ",
		"MRK", "CRM", "BAC", "CVX", "ADBE", "NFLX", "AMD", "PEP", "TMO", "WMT",
		"WFC", "KO", "DIS", "CSCO", "ACN", "ABT", "LIN", "MCD", "INTU", "ORCL",
	}
	end := time.Now()
	start := end.AddDate(0, 0, -7) // Last 7 days

	totalIngested := 0
	for _, ticker := range tickers {
		fmt.Printf("Fetching news for %s...\n", ticker)
		
		// Fetch current price for realistic ratings
		currentPrice := 150.0 // Default fallback
		snapshot, err := s.alpacaSvc.GetSnapshot(ctx, ticker)
		if err == nil && snapshot != nil {
			if snapshot.LatestTrade != nil {
				currentPrice = snapshot.LatestTrade.Price
			} else if snapshot.MinuteBar != nil {
				currentPrice = snapshot.MinuteBar.Close
			}
		}

		articles, err := s.alpacaSvc.GetNews(ctx, ticker, start, end)
		if err != nil {
			fmt.Printf("Failed to fetch news for %s: %v\n", ticker, err)
			continue
		}

		if len(articles) == 0 {
			continue
		}

		ratings := make([]*domain.StockRating, 0)
		for _, article := range articles {
			rating := s.transformNewsToRating(ticker, article, currentPrice)
			ratings = append(ratings, rating)
		}

		count, err := s.stockRepo.CreateStockRatingsBatch(ctx, ratings)
		if err != nil {
			fmt.Printf("Failed to store news-based ratings for %s: %v\n", ticker, err)
			continue
		}
		totalIngested += count
	}

	fmt.Printf("Alpaca News ingestion completed. Total simulated ratings ingested: %d\n", totalIngested)
	return nil
}

func (s *Service) transformNewsToRating(ticker string, article domain.NewsArticle, currentPrice float64) *domain.StockRating {
	// Basic sentiment analysis based on keywords in headline
	headline := strings.ToLower(article.Headline)
	summary := strings.ToLower(article.Summary)

	action := "Reiterated by"
	ratingTo := "Hold"
	targetTo := currentPrice * 1.05 // Default 5% upside
	targetFrom := currentPrice
	
	positiveKeywords := []string{"surge", "gain", "buy", "growth", "positive", "upgrade", "beat", "strong", "high", "rise"}
	negativeKeywords := []string{"drop", "fall", "sell", "negative", "downgrade", "miss", "weak", "low", "slump"}

	posCount := 0
	for _, kw := range positiveKeywords {
		if strings.Contains(headline, kw) || strings.Contains(summary, kw) {
			posCount++
		}
	}

	negCount := 0
	for _, kw := range negativeKeywords {
		if strings.Contains(headline, kw) || strings.Contains(summary, kw) {
			negCount++
		}
	}

	if posCount > negCount {
		action = "Upgraded by"
		ratingTo = "Buy"
		targetTo = currentPrice * 1.15 // 15% upside
		if posCount > 2 {
			ratingTo = "Strong Buy"
			targetTo = currentPrice * 1.25 // 25% upside
		}
	} else if negCount > posCount {
		action = "Downgraded by"
		ratingTo = "Sell"
		targetTo = currentPrice * 0.85 // 15% downside
		if negCount > 2 {
			ratingTo = "Strong Sell"
			targetTo = currentPrice * 0.75 // 25% downside
		}
	}

	return &domain.StockRating{
		RatingID:   uuid.New(),
		Ticker:     ticker,
		Company:    ticker + " Corp", // Simplified
		Brokerage:  "Alpaca News",
		Action:     action,
		RatingTo:   ratingTo,
		TargetFrom: &targetFrom,
		TargetTo:   &targetTo,
		Time:       article.Timestamp,
		CreatedAt:  time.Now(),
	}
}

// fetchDataFromAPI makes HTTP request to the external API
func (s *Service) fetchDataFromAPI(ctx context.Context, nextPage *string) (*domain.APIResponse, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", s.apiURL, nil)
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeUpstreamAPI, "failed to create API request")
	}

	// Add authorization header
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.apiToken))
	req.Header.Set("Content-Type", "application/json")

	// Add next_page parameter if provided
	if nextPage != nil && *nextPage != "" {
		q := req.URL.Query()
		q.Add("next_page", *nextPage)
		req.URL.RawQuery = q.Encode()
	}

	// Make the request with retry logic
	resp, err := s.makeRequestWithRetry(ctx, req, 3)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, apperrors.New(apperrors.ErrCodeUpstreamAPI,
			fmt.Sprintf("API request failed with status %d: %s", resp.StatusCode, string(body)))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeUpstreamAPI, "failed to read API response body")
	}

	var apiResponse domain.APIResponse
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeUpstreamAPI, "failed to unmarshal API response")
	}

	return &apiResponse, nil
}

// makeRequestWithRetry implements exponential backoff retry logic
func (s *Service) makeRequestWithRetry(ctx context.Context, req *http.Request, maxRetries int) (*http.Response, error) {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 1s, 2s, 4s, etc.
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}

		resp, err := s.client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		// Success or non-retryable error
		if resp.StatusCode < 500 {
			return resp, nil
		}

		resp.Body.Close()
		lastErr = fmt.Errorf("server error: %d", resp.StatusCode)
	}

	return nil, apperrors.Wrap(lastErr, apperrors.ErrCodeUpstreamAPI, "API request failed after retries")
}

// transformAPIRatings converts API response items to domain models
func (s *Service) transformAPIRatings(apiRatings []domain.APIStockRating) ([]domain.StockRating, error) {
	ratings := make([]domain.StockRating, 0, len(apiRatings))

	// Use a map to track unique ratings and prevent duplicates
	uniqueRatings := make(map[string]domain.StockRating)

	for _, apiRating := range apiRatings {
		// Parse time
		parsedTime, err := time.Parse(time.RFC3339, apiRating.Time)
		if err != nil {
			return nil, apperrors.Wrap(err, apperrors.ErrCodeValidation,
				fmt.Sprintf("failed to parse time for ticker %s", apiRating.Ticker))
		}

		// Parse target prices
		var targetFrom, targetTo *float64
		if apiRating.TargetFrom != "" {
			if val, err := s.parsePrice(apiRating.TargetFrom); err == nil {
				targetFrom = &val
			}
		}
		if apiRating.TargetTo != "" {
			if val, err := s.parsePrice(apiRating.TargetTo); err == nil {
				targetTo = &val
			}
		}

		// Handle optional rating_from
		var ratingFrom *string
		if apiRating.RatingFrom != "" {
			ratingFrom = &apiRating.RatingFrom
		}

		rating := domain.StockRating{
			RatingID:   uuid.New(),
			Ticker:     strings.ToUpper(apiRating.Ticker),
			Company:    apiRating.Company,
			Brokerage:  apiRating.Brokerage,
			Action:     apiRating.Action,
			RatingFrom: ratingFrom,
			RatingTo:   apiRating.RatingTo,
			TargetFrom: targetFrom,
			TargetTo:   targetTo,
			Time:       parsedTime,
			CreatedAt:  time.Now(),
		}

		// Create unique key to prevent duplicates
		uniqueKey := fmt.Sprintf("%s-%s-%s-%s-%s",
			rating.Ticker,
			rating.Brokerage,
			rating.RatingTo,
			rating.Time.Format(time.RFC3339),
			rating.Action)

		// Only add if this combination doesn't exist yet
		if _, exists := uniqueRatings[uniqueKey]; !exists {
			uniqueRatings[uniqueKey] = rating
		} else {
			fmt.Printf("🔄 Skipping duplicate rating: %s - %s - %s\n",
				rating.Ticker, rating.Brokerage, rating.RatingTo)
		}
	}

	// Convert map back to slice
	for _, rating := range uniqueRatings {
		ratings = append(ratings, rating)
	}

	fmt.Printf("Filtered ratings: %d → %d\n", len(apiRatings), len(ratings))
	return ratings, nil
}

// parsePrice extracts numeric value from price string
func (s *Service) parsePrice(priceStr string) (float64, error) {
	cleaned := strings.TrimSpace(priceStr)
	cleaned = strings.TrimPrefix(cleaned, "$")
	cleaned = strings.TrimPrefix(cleaned, "€")
	cleaned = strings.TrimPrefix(cleaned, "£")

	return strconv.ParseFloat(cleaned, 64)
}

// EnrichStockData fetches additional data for stocks from external sources
func (s *Service) EnrichStockData(ctx context.Context, tickers []string) error {
	fmt.Printf("Enriching data for %d tickers (not implemented)\n", len(tickers))
	return nil
}
