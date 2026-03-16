package domain

import (
	"context"
	"time"
)

// StockRepository defines the contract for stock data persistence.
type StockRepository interface {
	// CreateStockRating stores a single stock rating in the database.
	CreateStockRating(ctx context.Context, rating *StockRating) error

	// CreateStockRatingsBatch efficiently stores multiple stock ratings in a single transaction.
	CreateStockRatingsBatch(ctx context.Context, ratings []*StockRating) (int, error)

	// GetStockRatings retrieves paginated stock ratings with optional filtering and sorting.
	GetStockRatings(ctx context.Context, filters FilterOptions) (*PaginatedResponse[StockRating], error)

	// GetStockRatingsByTicker retrieves all ratings for a specific stock ticker.
	GetStockRatingsByTicker(ctx context.Context, ticker string) ([]StockRating, error)

	// GetUniqueTickers retrieves all unique stock tickers that have ratings.
	GetUniqueTickers(ctx context.Context) ([]string, error)

	// CreateEnrichedStockData stores additional analysis data for a stock.
	CreateEnrichedStockData(ctx context.Context, data *EnrichedStockData) error

	// GetEnrichedStockData retrieves additional analysis data for a stock ticker.
	GetEnrichedStockData(ctx context.Context, ticker string) (*EnrichedStockData, error)

	// GetLatestRatingsByTicker returns the most recent rating for each ticker.
	GetLatestRatingsByTicker(ctx context.Context) (map[string]*StockRating, error)

	// DeleteOldEnrichedData removes enriched stock data records older than a given time.
	DeleteOldEnrichedData(ctx context.Context, olderThan time.Time) (int64, error)
}

// IngestionService defines the contract for data ingestion from external APIs.
type IngestionService interface {
	// IngestAllData performs a complete data ingestion cycle.
	IngestAllData(ctx context.Context) error
}

// RecommendationService defines the contract for generating stock recommendations.
type RecommendationService interface {
	// GenerateRecommendations analyzes all available data and generates fresh stock recommendations.
	GenerateRecommendations(ctx context.Context) ([]StockRecommendation, error)

	// GetCachedRecommendations retrieves the latest generated recommendations from cache.
	GetCachedRecommendations(ctx context.Context) ([]StockRecommendation, error)
}

// PriceBar represents a single price bar/candle from market data.
type PriceBar struct {
	Timestamp string  `json:"timestamp"` // ISO 8601 timestamp in UTC
	Open      float64 `json:"open"`      // Opening price for the period
	High      float64 `json:"high"`      // Highest price during the period
	Low       float64 `json:"low"`       // Lowest price during the period
	Close     float64 `json:"close"`     // Closing price for the period
	Volume    int64   `json:"volume"`    // Number of shares traded during the period
}

// Snapshot represents current market snapshot data for real-time quotes.
type Snapshot struct {
	Symbol       string    `json:"symbol"`                   // Stock symbol
	LatestTrade  *Trade    `json:"latest_trade,omitempty"`   // Most recent trade
	LatestQuote  *Quote    `json:"latest_quote,omitempty"`   // Most recent bid/ask quote
	MinuteBar    *PriceBar `json:"minute_bar,omitempty"`     // Current minute bar
	DailyBar     *PriceBar `json:"daily_bar,omitempty"`      // Current day's bar
	PrevDailyBar *PriceBar `json:"prev_daily_bar,omitempty"` // Previous day's bar
}

// Trade represents a single trade execution.
type Trade struct {
	Timestamp string  `json:"timestamp"` // ISO 8601 timestamp of the trade
	Price     float64 `json:"price"`     // Execution price per share
	Size      int64   `json:"size"`      // Number of shares traded
}

// Quote represents the current bid/ask spread for a stock.
type Quote struct {
	Timestamp string  `json:"timestamp"` // ISO 8601 timestamp of the quote
	BidPrice  float64 `json:"bid_price"` // Highest price buyers are willing to pay
	AskPrice  float64 `json:"ask_price"` // Lowest price sellers are willing to accept
	BidSize   int64   `json:"bid_size"`  // Number of shares available at bid price
	AskSize   int64   `json:"ask_size"`  // Number of shares available at ask price
}

// AlpacaService defines the contract for Alpaca API interactions.
type AlpacaService interface {
	// GetHistoricalBars fetches historical price data for technical analysis.
	GetHistoricalBars(ctx context.Context, symbol string, timeframe string, start, end time.Time) ([]PriceBar, error)

	// GetSnapshot fetches current market snapshot for real-time data.
	GetSnapshot(ctx context.Context, symbol string) (*Snapshot, error)

	// GetRecentBars fetches the most recent bars for a symbol.
	GetRecentBars(ctx context.Context, symbol string) ([]PriceBar, error)

	// IsMarketHours checks if the US stock market is currently open.
	IsMarketHours() bool

	// GetNews fetches recent news articles for a ticker.
	GetNews(ctx context.Context, symbol string, start, end time.Time) ([]NewsArticle, error)
}

// NewsArticle represents a news article from Alpaca.
type NewsArticle struct {
	ID        int       `json:"id"`
	Headline  string    `json:"headline"`
	Summary   string    `json:"summary"`
	Content   string    `json:"content"`
	URL       string    `json:"url"`
	Symbols   []string  `json:"symbols"`
	Timestamp time.Time `json:"timestamp"`
}

// FilterOptions defines filtering and pagination options for data queries.
type FilterOptions struct {
	Page     int    `json:"page"`      // Page number (1-based)
	Limit    int    `json:"limit"`     // Items per page
	Search   string `json:"search"`    // Search term for full-text search
	SortBy   string `json:"sort_by"`   // Field to sort by
	SortDesc bool   `json:"sort_desc"` // Sort direction
}
