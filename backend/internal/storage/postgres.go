package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"stock-analyzer/internal/domain"
	apperrors "stock-analyzer/pkg/errors"
	"strings"
	"time"
)

// PostgresRepository implements the StockRepository interface for PostgreSQL/CockroachDB
type PostgresRepository struct {
	db      *sql.DB
	dialect string
}

// NewPostgresRepository creates a new PostgresRepository instance
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	dialect := "postgres"
	// Simple check for SQLite
	err := db.Ping()
	if err == nil {
		// This is a bit hacky but works for local testing
		var dummy int
		err = db.QueryRow("SELECT 1").Scan(&dummy)
		if err == nil {
			// Check if we are using modernc.org/sqlite
			// Usually we can just check the connection type but here we'll use an env var hint or similar
			// For now, we'll try to detect by executing a postgres-specific query
			_, err = db.Exec("SET TIME ZONE 'UTC'")
			if err != nil {
				dialect = "sqlite"
			}
		}
	}

	return &PostgresRepository{
		db:      db,
		dialect: dialect,
	}
}

func (r *PostgresRepository) placeholder(n int) string {
	if r.dialect == "sqlite" {
		return "?"
	}
	return fmt.Sprintf("$%d", n)
}

func (r *PostgresRepository) ilike() string {
	if r.dialect == "sqlite" {
		return "LIKE" // SQLite LIKE is case-insensitive by default for ASCII
	}
	return "ILIKE"
}

// CreateStockRating stores a new stock rating
func (r *PostgresRepository) CreateStockRating(ctx context.Context, rating *domain.StockRating) error {
	query := fmt.Sprintf(`
		INSERT INTO stock_ratings (
			rating_id, ticker, company, brokerage, action, 
			rating_from, rating_to, target_from, target_to, time
		) VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s)`,
		r.placeholder(1), r.placeholder(2), r.placeholder(3), r.placeholder(4), r.placeholder(5),
		r.placeholder(6), r.placeholder(7), r.placeholder(8), r.placeholder(9), r.placeholder(10))

	_, err := r.db.ExecContext(ctx, query,
		rating.RatingID, rating.Ticker, rating.Company, rating.Brokerage,
		rating.Action, rating.RatingFrom, rating.RatingTo, rating.TargetFrom,
		rating.TargetTo, rating.Time)

	if err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeDatabase, "failed to create stock rating")
	}

	return nil
}

// CreateStockRatingsBatch stores multiple stock ratings in a single transaction
func (r *PostgresRepository) CreateStockRatingsBatch(ctx context.Context, ratings []*domain.StockRating) (int, error) {
	if len(ratings) == 0 {
		return 0, nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, apperrors.Wrap(err, apperrors.ErrCodeDatabase, "failed to begin transaction")
	}
	defer tx.Rollback()

	query := fmt.Sprintf(`
		INSERT INTO stock_ratings (
			rating_id, ticker, company, brokerage, action, 
			rating_from, rating_to, target_from, target_to, time
		) VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
		ON CONFLICT (ticker, brokerage, rating_to, time) DO NOTHING`,
		r.placeholder(1), r.placeholder(2), r.placeholder(3), r.placeholder(4), r.placeholder(5),
		r.placeholder(6), r.placeholder(7), r.placeholder(8), r.placeholder(9), r.placeholder(10))

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return 0, apperrors.Wrap(err, apperrors.ErrCodeDatabase, "failed to prepare statement")
	}
	defer stmt.Close()

	insertedCount := 0
	for _, rating := range ratings {
		result, err := stmt.ExecContext(ctx,
			rating.RatingID, rating.Ticker, rating.Company, rating.Brokerage,
			rating.Action, rating.RatingFrom, rating.RatingTo, rating.TargetFrom,
			rating.TargetTo, rating.Time)
		if err != nil {
			return 0, apperrors.Wrap(err, apperrors.ErrCodeDatabase, "failed to insert rating")
		}

		if rowsAffected, err := result.RowsAffected(); err == nil && rowsAffected > 0 {
			insertedCount++
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, apperrors.Wrap(err, apperrors.ErrCodeDatabase, "failed to commit transaction")
	}

	fmt.Printf("Database batch: %d attempted → %d inserted\n", len(ratings), insertedCount)
	return insertedCount, nil
}

// GetStockRatings retrieves paginated stock ratings with optional filtering
func (r *PostgresRepository) GetStockRatings(ctx context.Context, filters domain.FilterOptions) (*domain.PaginatedResponse[domain.StockRating], error) {
	page := filters.Page
	if page < 1 {
		page = 1
	}
	limit := filters.Limit
	if limit < 1 || limit > 100 {
		limit = 20
	}
	sortBy := filters.SortBy
	if sortBy == "" {
		sortBy = "time"
	}
	search := filters.Search
	offset := (page - 1) * limit

	// Build WHERE clause for search
	whereClause := ""
	args := []interface{}{}
	argCount := 0

	if search != "" {
		// Use distinct placeholders for each column to ensure compatibility across dialects
		whereClause = fmt.Sprintf("WHERE (company %s %s OR ticker %s %s OR brokerage %s %s)",
			r.ilike(), r.placeholder(1), r.ilike(), r.placeholder(2), r.ilike(), r.placeholder(3))
		args = append(args, "%"+search+"%", "%"+search+"%", "%"+search+"%")
		argCount = 3
	}

	// Validate and build ORDER BY clause
	validSortFields := map[string]bool{
		"time":      true,
		"ticker":    true,
		"company":   true,
		"brokerage": true,
		"target_to": true,
		"rating_to": true,
	}

	if !validSortFields[sortBy] {
		sortBy = "time"
	}

	order := "desc"
	if !filters.SortDesc {
		order = "asc"
	}

	// Internal order for deduplication must be by ticker first
	// then we sort the final result set by user's preference
	
	// Get total count
	var countQuery string
	if r.dialect == "sqlite" {
		countQuery = fmt.Sprintf("SELECT COUNT(DISTINCT ticker) FROM stock_ratings %s", whereClause)
	} else {
		countQuery = fmt.Sprintf("SELECT COUNT(DISTINCT ticker) FROM stock_ratings %s", whereClause)
	}

	var totalCount int
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeDatabase, "failed to get total count")
	}

	// Build paginated results query with mandatory deduplication
	var query string
	if r.dialect == "sqlite" {
		// SQLite compatible deduplication
		query = fmt.Sprintf(`
			SELECT rating_id, ticker, company, brokerage, action, rating_from, 
				   rating_to, target_from, target_to, time, created_at
			FROM stock_ratings 
			WHERE rowid IN (
				SELECT id FROM (
					SELECT rowid as id, ticker, MAX(time) as max_time
					FROM stock_ratings 
					%s
					GROUP BY ticker
				)
			)
			ORDER BY %s %s
			LIMIT %s OFFSET %s`,
			whereClause, sortBy, strings.ToUpper(order), r.placeholder(argCount+1), r.placeholder(argCount+2))
	} else {
		// Postgres/CockroachDB optimized deduplication
		query = fmt.Sprintf(`
			SELECT * FROM (
				SELECT DISTINCT ON (ticker) rating_id, ticker, company, brokerage, action, rating_from, 
					   rating_to, target_from, target_to, time, created_at
				FROM stock_ratings 
				%s
				ORDER BY ticker, time DESC
			) sub
			ORDER BY %s %s
			LIMIT %s OFFSET %s`,
			whereClause, sortBy, strings.ToUpper(order), r.placeholder(argCount+1), r.placeholder(argCount+2))
	}

	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeDatabase, "failed to query stock ratings")
	}
	defer rows.Close()

	var ratings []domain.StockRating
	for rows.Next() {
		var rating domain.StockRating
		err := rows.Scan(
			&rating.RatingID, &rating.Ticker, &rating.Company, &rating.Brokerage,
			&rating.Action, &rating.RatingFrom, &rating.RatingTo, &rating.TargetFrom,
			&rating.TargetTo, &rating.Time, &rating.CreatedAt)
		if err != nil {
			return nil, apperrors.Wrap(err, apperrors.ErrCodeDatabase, "failed to scan rating")
		}
		ratings = append(ratings, rating)
	}

	if err := rows.Err(); err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeDatabase, "error iterating over ratings")
	}

	// Calculate pagination metadata
	totalPages := (totalCount + limit - 1) / limit

	response := &domain.PaginatedResponse[domain.StockRating]{
		Data: ratings,
		Pagination: domain.Pagination{
			Page:       page,
			Limit:      limit,
			TotalItems: totalCount,
			TotalPages: totalPages,
		},
	}

	return response, nil
}

// GetStockRatingsByTicker retrieves all ratings for a specific ticker
func (r *PostgresRepository) GetStockRatingsByTicker(ctx context.Context, ticker string) ([]domain.StockRating, error) {
	query := fmt.Sprintf(`
		SELECT rating_id, ticker, company, brokerage, action, rating_from, 
			   rating_to, target_from, target_to, time, created_at
		FROM stock_ratings 
		WHERE ticker = %s 
		ORDER BY time DESC`, r.placeholder(1))

	rows, err := r.db.QueryContext(ctx, query, ticker)
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeDatabase, "failed to query ratings by ticker")
	}
	defer rows.Close()

	var ratings []domain.StockRating
	for rows.Next() {
		var rating domain.StockRating
		err := rows.Scan(
			&rating.RatingID, &rating.Ticker, &rating.Company, &rating.Brokerage,
			&rating.Action, &rating.RatingFrom, &rating.RatingTo, &rating.TargetFrom,
			&rating.TargetTo, &rating.Time, &rating.CreatedAt)
		if err != nil {
			return nil, apperrors.Wrap(err, apperrors.ErrCodeDatabase, "failed to scan rating")
		}
		ratings = append(ratings, rating)
	}

	if err := rows.Err(); err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeDatabase, "error iterating over ratings")
	}

	return ratings, nil
}

// GetUniqueTickers retrieves all unique ticker symbols
func (r *PostgresRepository) GetUniqueTickers(ctx context.Context) ([]string, error) {
	query := "SELECT DISTINCT ticker FROM stock_ratings ORDER BY ticker"

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeDatabase, "failed to query unique tickers")
	}
	defer rows.Close()

	var tickers []string
	for rows.Next() {
		var ticker string
		if err := rows.Scan(&ticker); err != nil {
			return nil, apperrors.Wrap(err, apperrors.ErrCodeDatabase, "failed to scan ticker")
		}
		tickers = append(tickers, ticker)
	}

	if err := rows.Err(); err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeDatabase, "error iterating over unique tickers")
	}

	return tickers, nil
}

// CreateEnrichedStockData stores enriched stock data
func (r *PostgresRepository) CreateEnrichedStockData(ctx context.Context, data *domain.EnrichedStockData) error {
	histPricesJSON, err := json.Marshal(data.HistoricalPrices)
	if err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeValidation, "failed to marshal historical prices")
	}

	sentimentJSON, err := json.Marshal(data.NewsSentiment)
	if err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeValidation, "failed to marshal news sentiment")
	}

	query := fmt.Sprintf(`
		INSERT INTO enriched_stock_data (ticker, historical_prices, news_sentiment, updated_at)
		VALUES (%s, %s, %s, CURRENT_TIMESTAMP)
		ON CONFLICT (ticker) DO UPDATE SET
			historical_prices = EXCLUDED.historical_prices,
			news_sentiment = EXCLUDED.news_sentiment,
			updated_at = CURRENT_TIMESTAMP`,
		r.placeholder(1), r.placeholder(2), r.placeholder(3))

	_, err = r.db.ExecContext(ctx, query, data.Ticker, histPricesJSON, sentimentJSON)
	if err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeDatabase, "failed to create enriched stock data")
	}

	return nil
}

// GetEnrichedStockData retrieves enriched data for a ticker
func (r *PostgresRepository) GetEnrichedStockData(ctx context.Context, ticker string) (*domain.EnrichedStockData, error) {
	query := fmt.Sprintf(`
		SELECT ticker, historical_prices, news_sentiment, updated_at
		FROM enriched_stock_data 
		WHERE ticker = %s`, r.placeholder(1))

	var data domain.EnrichedStockData
	var histPricesJSON, sentimentJSON []byte

	err := r.db.QueryRowContext(ctx, query, ticker).Scan(
		&data.Ticker, &histPricesJSON, &sentimentJSON, &data.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, apperrors.ErrNotFound.WithDetails(fmt.Sprintf("enriched data for ticker %s not found", ticker))
	}
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeDatabase, "failed to get enriched stock data")
	}

	if err := json.Unmarshal(histPricesJSON, &data.HistoricalPrices); err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeDatabase, "failed to unmarshal historical prices")
	}

	if err := json.Unmarshal(sentimentJSON, &data.NewsSentiment); err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeDatabase, "failed to unmarshal news sentiment")
	}

	return &data, nil
}

// GetLatestRatingsByTicker gets the most recent rating for each ticker
func (r *PostgresRepository) GetLatestRatingsByTicker(ctx context.Context) (map[string]*domain.StockRating, error) {
	var query string
	if r.dialect == "sqlite" {
		query = `
			SELECT ticker, rating_id, company, brokerage, action, 
				   rating_from, rating_to, target_from, target_to, time, created_at
			FROM stock_ratings 
			WHERE (ticker, time) IN (
				SELECT ticker, MAX(time)
				FROM stock_ratings
				GROUP BY ticker
			)`
	} else {
		query = `
			SELECT DISTINCT ON (ticker) ticker, rating_id, company, brokerage, action, 
				   rating_from, rating_to, target_from, target_to, time, created_at
			FROM stock_ratings 
			ORDER BY ticker, time DESC`
	}

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeDatabase, "failed to query latest ratings")
	}
	defer rows.Close()

	result := make(map[string]*domain.StockRating)
	for rows.Next() {
		var rating domain.StockRating
		err := rows.Scan(
			&rating.Ticker, &rating.RatingID, &rating.Company, &rating.Brokerage,
			&rating.Action, &rating.RatingFrom, &rating.RatingTo, &rating.TargetFrom,
			&rating.TargetTo, &rating.Time, &rating.CreatedAt)
		if err != nil {
			return nil, apperrors.Wrap(err, apperrors.ErrCodeDatabase, "failed to scan latest rating")
		}
		result[rating.Ticker] = &rating
	}

	if err := rows.Err(); err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeDatabase, "error iterating over latest ratings")
	}

	return result, nil
}

// DeleteOldEnrichedData removes enriched stock data records older than a given time
func (r *PostgresRepository) DeleteOldEnrichedData(ctx context.Context, olderThan time.Time) (int64, error) {
	query := fmt.Sprintf(`DELETE FROM enriched_stock_data WHERE updated_at < %s`, r.placeholder(1))

	result, err := r.db.ExecContext(ctx, query, olderThan)
	if err != nil {
		return 0, apperrors.Wrap(err, apperrors.ErrCodeDatabase, "failed to delete old enriched data")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, apperrors.Wrap(err, apperrors.ErrCodeDatabase, "failed to get affected rows after deletion")
	}

	return rowsAffected, nil
}
