package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"stock-analyzer/internal/domain"
	apperrors "stock-analyzer/pkg/errors"

	"github.com/gin-gonic/gin"
)

// StockPriceResponse represents the price data response
type StockPriceResponse struct {
	Symbol string            `json:"symbol"`
	Bars   []domain.PriceBar `json:"bars"`
}

// StockLogoResponse represents the logo response
type StockLogoResponse struct {
	Symbol  string `json:"symbol"`
	LogoURL string `json:"logo_url"`
}

// Handlers contains all the HTTP handlers
type Handlers struct {
	stockRepo         domain.StockRepository
	ingestionSvc      domain.IngestionService
	recommendationSvc domain.RecommendationService
	alpacaSvc         domain.AlpacaService
}

// NewHandlers creates a new handlers instance
func NewHandlers(stockRepo domain.StockRepository, ingestionSvc domain.IngestionService, recommendationSvc domain.RecommendationService, alpacaSvc domain.AlpacaService) *Handlers {
	return &Handlers{
		stockRepo:         stockRepo,
		ingestionSvc:      ingestionSvc,
		recommendationSvc: recommendationSvc,
		alpacaSvc:         alpacaSvc,
	}
}

// GetStockPrice retrieves historical price data for a stock using Alpaca API
func (h *Handlers) GetStockPrice(c *gin.Context) {
	symbol := c.Param("symbol")
	if symbol == "" {
		HandleError(c, apperrors.ErrValidationFailure.WithDetails("symbol parameter is required"))
		return
	}

	symbol = strings.ToUpper(symbol)
	period := c.DefaultQuery("period", "1M")

	var timeframe string
	var start time.Time
	end := time.Now()

	switch period {
	case "1W":
		start = end.AddDate(0, 0, -7)
		timeframe = "1Hour"
	case "1M":
		start = end.AddDate(0, -1, 0)
		timeframe = "1Hour"
	case "3M":
		start = end.AddDate(0, -3, 0)
		timeframe = "1Day"
	case "6M":
		start = end.AddDate(0, -6, 0)
		timeframe = "1Day"
	case "1Y":
		start = end.AddDate(-1, 0, 0)
		timeframe = "1Day"
	case "2Y":
		start = end.AddDate(-2, 0, 0)
		timeframe = "1Day"
	default:
		start = end.AddDate(0, -1, 0)
		timeframe = "1Hour"
	}

	alpacaBars, err := h.alpacaSvc.GetHistoricalBars(c.Request.Context(), symbol, timeframe, start, end)
	if err != nil {
		HandleError(c, err)
		return
	}

	bars := make([]domain.PriceBar, len(alpacaBars))
	for i, alpacaBar := range alpacaBars {
		bars[i] = domain.PriceBar{
			Timestamp: alpacaBar.Timestamp,
			Open:      alpacaBar.Open,
			High:      alpacaBar.High,
			Low:       alpacaBar.Low,
			Close:     alpacaBar.Close,
			Volume:    alpacaBar.Volume,
		}
	}

	if len(bars) == 0 {
		HandleError(c, apperrors.ErrNotFound.WithDetails(fmt.Sprintf("No price data available for %s", symbol)))
		return
	}

	response := StockPriceResponse{
		Symbol: symbol,
		Bars:   bars,
	}

	c.JSON(http.StatusOK, response)
}

// GetStockLogo retrieves the logo URL for a stock
func (h *Handlers) GetStockLogo(c *gin.Context) {
	symbol := c.Param("symbol")
	if symbol == "" {
		HandleError(c, apperrors.ErrValidationFailure.WithDetails("symbol parameter is required"))
		return
	}

	symbol = strings.ToUpper(symbol)
	logoURL := fmt.Sprintf("https://logo.clearbit.com/%s.com", strings.ToLower(symbol))

	response := StockLogoResponse{
		Symbol:  symbol,
		LogoURL: logoURL,
	}

	c.Header("Cache-Control", "public, max-age=3600")
	c.Header("ETag", fmt.Sprintf(`"%s"`, symbol))

	c.JSON(http.StatusOK, response)
}

// GetStockRatings retrieves paginated stock ratings with optional filtering
func (h *Handlers) GetStockRatings(c *gin.Context) {
	page, err := parseIntQuery(c, "page", 1)
	if err != nil {
		HandleError(c, apperrors.ErrValidationFailure.WithDetails("invalid page parameter"))
		return
	}

	limit, err := parseIntQuery(c, "limit", 20)
	if err != nil {
		HandleError(c, apperrors.ErrValidationFailure.WithDetails("invalid limit parameter"))
		return
	}

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	sortBy := c.DefaultQuery("sort_by", "time")
	order := c.DefaultQuery("order", "desc")
	search := c.Query("search")

	filters := domain.FilterOptions{
		Page:     page,
		Limit:    limit,
		Search:   search,
		SortBy:   sortBy,
		SortDesc: order == "desc",
	}

	response, err := h.stockRepo.GetStockRatings(c.Request.Context(), filters)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetStockRatingsByTicker retrieves all ratings for a specific ticker
func (h *Handlers) GetStockRatingsByTicker(c *gin.Context) {
	ticker := c.Param("ticker")
	if ticker == "" {
		HandleError(c, apperrors.ErrValidationFailure.WithDetails("ticker parameter is required"))
		return
	}

	ratings, err := h.stockRepo.GetStockRatingsByTicker(c.Request.Context(), ticker)
	if err != nil {
		HandleError(c, err)
		return
	}

	if len(ratings) == 0 {
		HandleError(c, apperrors.ErrNotFound.WithDetails("no ratings found for ticker "+ticker))
		return
	}

	c.JSON(http.StatusOK, ratings)
}

// GetRecommendations retrieves stock recommendations
func (h *Handlers) GetRecommendations(c *gin.Context) {
	recommendations, err := h.recommendationSvc.GetCachedRecommendations(c.Request.Context())
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, recommendations)
}

// TriggerIngestion manually triggers a full data ingestion process
func (h *Handlers) TriggerIngestion(c *gin.Context) {
	// Create a long-lived context for background processing
	ctx := context.Background()
	
	go func() {
		if err := h.ingestionSvc.IngestAllData(ctx); err != nil {
			println("Ingestion error:", err.Error())
		}
	}()

	c.JSON(http.StatusAccepted, gin.H{
		"message": "Data ingestion started",
		"status":  "accepted",
	})
}

// HealthCheck returns the health status of the service
func (h *Handlers) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"service":   "stock-analyzer",
		"timestamp": gin.H{"time": "now"},
	})
}

// parseIntQuery parses an integer query parameter with a default value
func parseIntQuery(c *gin.Context, key string, defaultValue int) (int, error) {
	str := c.Query(key)
	if str == "" {
		return defaultValue, nil
	}

	value, err := strconv.Atoi(str)
	if err != nil {
		return 0, err
	}

	return value, nil
}
