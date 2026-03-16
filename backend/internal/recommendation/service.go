package recommendation

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"stock-analyzer/internal/domain"
	apperrors "stock-analyzer/pkg/errors"
)

// Service implements the RecommendationService interface
type Service struct {
	stockRepo domain.StockRepository
	cache     *recommendationCache
}

// recommendationCache provides in-memory caching for recommendations
type recommendationCache struct {
	recommendations []domain.StockRecommendation
	lastUpdated     time.Time
	mutex           sync.RWMutex
	ttl             time.Duration
}

// NewService creates a new recommendation service
func NewService(stockRepo domain.StockRepository) *Service {
	return &Service{
		stockRepo: stockRepo,
		cache: &recommendationCache{
			ttl: 5 * time.Minute, // Cache for 5 minutes
		},
	}
}

// GenerateRecommendations analyzes data and generates stock recommendations
func (s *Service) GenerateRecommendations(ctx context.Context) ([]domain.StockRecommendation, error) {
	// Step 1: Get the latest ratings for all tickers
	latestRatings, err := s.stockRepo.GetLatestRatingsByTicker(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeDatabase, "failed to get latest ratings")
	}

	// Step 2: Filter stocks with positive analyst ratings
	candidates := s.filterPositiveRatings(latestRatings)
	if len(candidates) == 0 {
		return []domain.StockRecommendation{}, nil
	}

	// Step 3: Generate recommendations (using basic analysis to avoid slowdowns)
	var recommendations []domain.StockRecommendation
	for _, rating := range candidates {
		// Skip enriched data lookup for now to avoid timeouts
		recommendation := s.createBasicRecommendation(rating)
		if recommendation != nil {
			recommendations = append(recommendations, *recommendation)
		}
	}

	// Step 4: Sort recommendations by score (descending)
	sort.Slice(recommendations, func(i, j int) bool {
		return recommendations[i].Score > recommendations[j].Score
	})

	// Step 5: Return top 10 recommendations
	if len(recommendations) > 10 {
		recommendations = recommendations[:10]
	}

	return recommendations, nil
}

// filterPositiveRatings filters stocks with positive analyst ratings
func (s *Service) filterPositiveRatings(latestRatings map[string]*domain.StockRating) []*domain.StockRating {
	var candidates []*domain.StockRating

	positiveActions := map[string]bool{
		"upgraded by":   true,
		"initiated by":  true,
		"reiterated by": true,
	}

	positiveRatings := map[string]bool{
		"Buy":        true,
		"Strong Buy": true,
		"Outperform": true,
		"Overweight": true,
	}

	for _, rating := range latestRatings {
		// Check if action indicates positive movement
		actionPositive := positiveActions[strings.ToLower(rating.Action)]

		// Check if rating is positive
		ratingPositive := positiveRatings[rating.RatingTo]

		// Check if this was an upgrade (rating_from was worse than rating_to)
		wasUpgraded := s.isUpgrade(rating.RatingFrom, &rating.RatingTo)

		// Include if any condition is met
		if actionPositive || ratingPositive || wasUpgraded {
			candidates = append(candidates, rating)
		}
	}

	return candidates
}

// isUpgrade determines if the rating change represents an upgrade
func (s *Service) isUpgrade(from *string, to *string) bool {
	if from == nil || to == nil {
		return false
	}

	ratingScore := map[string]int{
		"Sell":           1,
		"Underperform":   2,
		"Hold":           3,
		"Market Perform": 3,
		"Neutral":        3,
		"Buy":            4,
		"Outperform":     4,
		"Overweight":     4,
		"Strong Buy":     5,
	}

	fromScore, fromExists := ratingScore[*from]
	toScore, toExists := ratingScore[*to]

	return fromExists && toExists && toScore > fromScore
}

// createBasicRecommendation creates a recommendation based on an advanced scoring algorithm
func (s *Service) createBasicRecommendation(rating *domain.StockRating) *domain.StockRecommendation {
	baseScore := 0.5 // Start neutral

	// 1. Rating Strength Bonus
	ratingBonus := map[string]float64{
		"Strong Buy": 0.25,
		"Buy":        0.15,
		"Outperform": 0.10,
		"Overweight": 0.10,
		"Hold":       0.0,
		"Sell":       -0.2,
		"Strong Sell": -0.3,
	}
	if bonus, exists := ratingBonus[rating.RatingTo]; exists {
		baseScore += bonus
	}

	// 2. Upside Potential Calculation (if we have target prices)
	upsideBonus := 0.0
	upsidePercentage := 0.0
	if rating.TargetTo != nil && rating.TargetFrom != nil && *rating.TargetFrom > 0 {
		currentPrice := *rating.TargetFrom // We store current price in TargetFrom during ingestion
		targetPrice := *rating.TargetTo
		upsidePercentage = ((targetPrice - currentPrice) / currentPrice) * 100
		
		// Add up to 0.20 score based on upside (e.g., 20% upside = max bonus)
		if upsidePercentage > 0 {
			upsideBonus = math.Min(0.20, (upsidePercentage/100.0))
		}
	} else if rating.TargetTo != nil {
		// Fallback if we only have the target (give a flat optimistic bonus)
		upsideBonus = 0.05
	}
	baseScore += upsideBonus

	// 3. Time Decay Penalty (Ratings get stale)
	daysSince := time.Since(rating.Time).Hours() / 24.0
	timePenalty := 0.0
	if daysSince > 0 {
		// Lose 0.01 score for every day old, max penalty 0.15
		timePenalty = math.Min(0.15, daysSince*0.01)
	}
	baseScore -= timePenalty

	// 4. Action Momentum
	if strings.Contains(strings.ToLower(rating.Action), "upgrad") {
		baseScore += 0.05
	} else if strings.Contains(strings.ToLower(rating.Action), "downgrad") {
		baseScore -= 0.10
	}

	// Calculate final score bounded between 0.1 and 0.99
	finalScore := math.Max(0.1, math.Min(0.99, baseScore))

	// Generate smart rationale
	rationale := s.generateSmartRationale(rating, upsidePercentage, daysSince)

	return &domain.StockRecommendation{
		Ticker:          rating.Ticker,
		Company:         rating.Company,
		Score:           finalScore,
		Rationale:       rationale,
		LatestRating:    rating.RatingTo,
		TargetPrice:     rating.TargetTo,
		TechnicalSignal: "Pending Analysis",
		SentimentScore:  nil,
		GeneratedAt:     time.Now(),
	}
}

// generateSmartRationale creates a dynamic string explaining the score
func (s *Service) generateSmartRationale(rating *domain.StockRating, upside float64, daysOld float64) string {
	var parts []string

	if strings.Contains(strings.ToLower(rating.Action), "upgrad") {
		parts = append(parts, fmt.Sprintf("Upgraded to %s", rating.RatingTo))
	} else {
		parts = append(parts, fmt.Sprintf("Rated %s", rating.RatingTo))
	}

	if upside > 0 {
		parts = append(parts, fmt.Sprintf("with a %.1f%% upside potential", upside))
	} else if rating.TargetTo != nil {
		parts = append(parts, fmt.Sprintf("targeting $%.2f", *rating.TargetTo))
	}

	if daysOld < 1 {
		parts = append(parts, "(Issued today)")
	} else {
		parts = append(parts, fmt.Sprintf("(%d days ago)", int(daysOld)))
	}

	return strings.Join(parts, " ")
}

// GetCachedRecommendations retrieves cached recommendations or generates new ones if cache is stale
func (s *Service) GetCachedRecommendations(ctx context.Context) ([]domain.StockRecommendation, error) {
	s.cache.mutex.RLock()

	if time.Since(s.cache.lastUpdated) < s.cache.ttl && len(s.cache.recommendations) > 0 {
		recommendations := make([]domain.StockRecommendation, len(s.cache.recommendations))
		copy(recommendations, s.cache.recommendations)
		s.cache.mutex.RUnlock()

		return recommendations, nil
	}

	s.cache.mutex.RUnlock()

	recommendations, err := s.GenerateRecommendations(ctx)
	if err != nil {
		return nil, err
	}

	s.cache.mutex.Lock()
	s.cache.recommendations = recommendations
	s.cache.lastUpdated = time.Now()
	s.cache.mutex.Unlock()

	return recommendations, nil
}

// analyzeTechnical analyzes historical data and returns technical signal and score
func (s *Service) analyzeTechnical(historicalData map[string]interface{}) (string, float64) {
	data, exists := historicalData["data"]
	if !exists {
		return "Insufficient Data", 0.0
	}

	dataSlice, ok := data.([]map[string]interface{})
	if !ok || len(dataSlice) < 2 {
		return "Insufficient Data", 0.0
	}

	firstClose, ok1 := dataSlice[0]["close"].(float64)
	lastClose, ok2 := dataSlice[len(dataSlice)-1]["close"].(float64)

	if !ok1 || !ok2 {
		return "Insufficient Data", 0.0
	}

	percentChange := (lastClose - firstClose) / firstClose * 100

	if percentChange > 2.0 {
		return "Golden Cross", 0.8
	} else if percentChange < -2.0 {
		return "Death Cross", 0.2
	} else {
		return "Sideways", 0.5
	}
}

// analyzeSentiment analyzes sentiment data and returns normalized score
func (s *Service) analyzeSentiment(sentimentData map[string]interface{}) *float64 {
	score, exists := sentimentData["sentiment_score"]
	if !exists {
		return nil
	}

	sentimentScore, ok := score.(float64)
	if !ok {
		return nil
	}

	// Normalize sentiment score from [-1, 1] to [0, 1]
	normalizedScore := (sentimentScore + 1) / 2
	return &normalizedScore
}
