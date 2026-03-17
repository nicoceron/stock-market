import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type {
  StockRating,
  StockRecommendation,
  PaginatedResponse,
  RatingsFilters,
  LoadingState,
  ApiError,
} from '@/types'
import { apiService } from '@/services/api'

// Interface for logo caching only (no price caching for fresh chart data)
interface StockLogoData {
  symbol: string
  logoUrl: string
  lastUpdated: number
}

// Interface for price bar data
interface PriceBar {
  timestamp: string
  close: number
}

// Interface for price data store (no caching, just shared data)
interface StockPriceData {
  symbol: string
  bars: PriceBar[]
  lastUpdated: number
}

// Rate limiting configuration
const RATE_LIMIT_CONFIG = {
  LOGO_BATCH_SIZE: 5,
  PRICE_BATCH_SIZE: 5,
  BATCH_DELAY: 250, // Delay between batches
  REQUEST_TIMEOUT: 8000, // 8 second timeout
  MAX_CONCURRENT_REQUESTS: 6,
}

export const useStocksStore = defineStore('stocks', () => {
  // State
  const ratings = ref<StockRating[]>([])
  const recommendations = ref<StockRecommendation[]>([])
  const pagination = ref({
    page: 1,
    limit: 20,
    total_items: 0,
    total_pages: 0,
  })

  const loadingState = ref<LoadingState>({
    isLoading: false,
    error: null,
    lastUpdated: null,
  })

  const filters = ref<RatingsFilters>({
    page: 1,
    limit: 20,
    sort_by: 'time',
    order: 'desc',
    search: '',
  })

  // Add call tracking to prevent duplicates
  const callTracker = ref({
    fetchRatingsInProgress: false,
    fetchRecommendationsInProgress: false,
    lastRatingsCall: 0,
    lastRecommendationsCall: 0,
  })

  // Logo caching only (no price caching for fresh chart data)
  const logoCache = ref<Record<string, StockLogoData>>({})

  // Price data store (no caching, just shared data for current session)
  const priceDataStore = ref<Record<string, StockPriceData>>({})

  // Rate limiting state
  const rateLimitState = ref({
    activeRequests: 0,
    logoBatchInProgress: false,
    priceBatchInProgress: false,
    pendingLogoRequests: new Set<string>(),
    pendingPriceRequests: new Set<string>(),
  })

  // Computed
  const isLoading = computed(() => loadingState.value.isLoading)
  const error = computed(() => loadingState.value.error)
  const hasError = computed(() => !!loadingState.value.error)
  const lastUpdated = computed(() => loadingState.value.lastUpdated)

  const totalRatings = computed(() => pagination.value.total_items || 0)
  const currentPage = computed(() => pagination.value.page || 1)
  const totalPages = computed(() => pagination.value.total_pages || 0)
  const pageLimit = computed(() => filters.value.limit || 20)

  const topRecommendations = computed(() => {
    const recs = recommendations.value
    if (!Array.isArray(recs)) return []
    return recs.slice(0, 10).sort((a, b) => (b.score || 0) - (a.score || 0))
  })

  // Utility functions
  function setLoading(loading: boolean) {
    loadingState.value.isLoading = loading
  }

  function setError(error: string | null) {
    loadingState.value.error = error
  }

  function clearError() {
    loadingState.value.error = null
  }

  function updateLastUpdated() {
    loadingState.value.lastUpdated = new Date()
  }

  // Logo cache utility functions only
  function getCachedLogoData(symbol: string): StockLogoData | null {
    const cached = logoCache.value[symbol]
    if (!cached) return null

    // Cache valid for 1 hour
    const now = Date.now()
    if (now - cached.lastUpdated > 60 * 60 * 1000) {
      delete logoCache.value[symbol]
      return null
    }

    return cached
  }

  function setCachedLogoData(symbol: string, logoUrl: string, alternatives?: string[]) {
    logoCache.value[symbol] = {
      symbol,
      logoUrl,
      alternatives,
      lastUpdated: Date.now(),
    } as any
  }

  // Price data utility functions (no caching, just shared store)
  function getPriceData(symbol: string): StockPriceData | null {
    return priceDataStore.value[symbol] || null
  }

  function setPriceData(symbol: string, bars: PriceBar[]) {
    priceDataStore.value[symbol] = {
      symbol,
      bars,
      lastUpdated: Date.now(),
    }
  }

  function isPriceDataLoading(symbol: string): boolean {
    return rateLimitState.value.pendingPriceRequests.has(symbol)
  }

  // Priority loading - Load trending recommendations first
  async function priorityLoadTrendingData() {
    console.log('🔥 Priority loading trending recommendations...')

    // First, get recommendations quickly
    await fetchRecommendations()

    // Get top 2 trending symbols
    const trendingSymbols = topRecommendations.value.slice(0, 2).map((r) => r.ticker)

    if (trendingSymbols.length > 0) {
      console.log('⚡ Loading priority data for trending stocks:', trendingSymbols)

      // Load logos and price data for trending stocks with highest priority
      await Promise.all([
        batchLoadLogoData(trendingSymbols, true), // true = priority
        batchLoadPriceData(trendingSymbols, true), // true = priority
      ])
    }
  }

  // Rate-limited batch loading functions
  async function batchLoadLogoData(symbols: string[], isPriority: boolean = false) {
    if (rateLimitState.value.logoBatchInProgress && !isPriority) {
      // Add new symbols to the queue if a batch is already running
      const newSymbols = symbols.filter((s) => !rateLimitState.value.pendingLogoRequests.has(s))
      newSymbols.forEach((s) => rateLimitState.value.pendingLogoRequests.add(s))
      return
    }

    if (isPriority) {
      // For priority loading, process immediately
      await processPriorityLogoBatch(symbols)
      return
    }

    rateLimitState.value.logoBatchInProgress = true
    symbols.forEach((s) => rateLimitState.value.pendingLogoRequests.add(s))

    const processQueue = async () => {
      const symbolsToLoad = Array.from(rateLimitState.value.pendingLogoRequests).filter(
        (symbol) => {
          return !getCachedLogoData(symbol)
        },
      )

      if (symbolsToLoad.length === 0) {
        rateLimitState.value.logoBatchInProgress = false
        return
      }

      console.log(`📦 Loading logos for ${symbolsToLoad.length} symbols:`, symbolsToLoad)

      for (let i = 0; i < symbolsToLoad.length; i += RATE_LIMIT_CONFIG.LOGO_BATCH_SIZE) {
        const batch = symbolsToLoad.slice(i, i + RATE_LIMIT_CONFIG.LOGO_BATCH_SIZE)

        while (rateLimitState.value.activeRequests >= RATE_LIMIT_CONFIG.MAX_CONCURRENT_REQUESTS) {
          await new Promise((resolve) => setTimeout(resolve, 100))
        }

        const promises = batch.map(async (symbol) => {
          rateLimitState.value.activeRequests++
          try {
            const logoData = await apiService.getStockLogo(symbol)
            if (logoData?.logo_url) {
              setCachedLogoData(symbol, logoData.logo_url, (logoData as any).alternatives)
            } else {
              setCachedLogoData(symbol, `https://logo.clearbit.com/${symbol.toLowerCase()}.com`)
            }
          } catch {
            setCachedLogoData(symbol, `https://logo.clearbit.com/${symbol.toLowerCase()}.com`)
          } finally {
            rateLimitState.value.activeRequests--
            rateLimitState.value.pendingLogoRequests.delete(symbol)
          }
        })
        await Promise.all(promises)
        await new Promise((resolve) => setTimeout(resolve, RATE_LIMIT_CONFIG.BATCH_DELAY))
      }

      if (rateLimitState.value.pendingLogoRequests.size > 0) {
        await processQueue()
      } else {
        rateLimitState.value.logoBatchInProgress = false
      }
    }

    await processQueue()
  }

  // Priority processing functions
  async function processPriorityLogoBatch(symbols: string[]) {
    console.log('🔥 Priority loading logos:', symbols)
    const promises = symbols.map(async (symbol) => {
      try {
        const cached = getCachedLogoData(symbol)
        if (cached) return // Already cached

        const logoData = await apiService.getStockLogo(symbol)
        if (logoData?.logo_url) {
          setCachedLogoData(symbol, logoData.logo_url, (logoData as any).alternatives)
        } else {
          setCachedLogoData(symbol, `https://logo.clearbit.com/${symbol.toLowerCase()}.com`)
        }
      } catch (error) {
        console.warn('Priority logo loading failed for', symbol, error)
        setCachedLogoData(symbol, `https://logo.clearbit.com/${symbol.toLowerCase()}.com`)
      }
    })

    await Promise.all(promises)
  }

  async function processPriorityPriceBatch(symbols: string[]) {
    console.log('🔥 Priority loading price data:', symbols)
    const promises = symbols.map(async (symbol) => {
      try {
        const priceData = await apiService.getStockPrice(symbol, '1W')
        if (priceData?.bars) {
          setPriceData(symbol, priceData.bars)
        }
      } catch (error) {
        console.warn('Priority price loading failed for', symbol, error)
      }
    })

    await Promise.all(promises)
  }

  async function batchLoadPriceData(symbols: string[], isPriority: boolean = false) {
    if (rateLimitState.value.priceBatchInProgress && !isPriority) {
      const newSymbols = symbols.filter((s) => !rateLimitState.value.pendingPriceRequests.has(s))
      newSymbols.forEach((s) => rateLimitState.value.pendingPriceRequests.add(s))
      return
    }

    if (isPriority) {
      // For priority loading, process immediately
      await processPriorityPriceBatch(symbols)
      return
    }

    rateLimitState.value.priceBatchInProgress = true
    symbols.forEach((s) => rateLimitState.value.pendingPriceRequests.add(s))

    const processQueue = async () => {
      const symbolsToLoad = Array.from(rateLimitState.value.pendingPriceRequests)

      if (symbolsToLoad.length === 0) {
        rateLimitState.value.priceBatchInProgress = false
        return
      }

      console.log(`📊 Loading price data for ${symbolsToLoad.length} symbols:`, symbolsToLoad)

      for (let i = 0; i < symbolsToLoad.length; i += RATE_LIMIT_CONFIG.PRICE_BATCH_SIZE) {
        const batch = symbolsToLoad.slice(i, i + RATE_LIMIT_CONFIG.PRICE_BATCH_SIZE)

        while (rateLimitState.value.activeRequests >= RATE_LIMIT_CONFIG.MAX_CONCURRENT_REQUESTS) {
          await new Promise((resolve) => setTimeout(resolve, 100))
        }

        const promises = batch.map(async (symbol) => {
          rateLimitState.value.activeRequests++
          try {
            const priceData = await apiService.getStockPrice(symbol, '1W')
            if (priceData?.bars) {
              setPriceData(symbol, priceData.bars)
            } else {
              setPriceData(symbol, [])
            }
          } catch {
            setPriceData(symbol, [])
          } finally {
            rateLimitState.value.activeRequests--
            rateLimitState.value.pendingPriceRequests.delete(symbol)
          }
        })
        await Promise.all(promises)
        await new Promise((resolve) => setTimeout(resolve, RATE_LIMIT_CONFIG.BATCH_DELAY))
      }

      if (rateLimitState.value.pendingPriceRequests.size > 0) {
        await processQueue()
      } else {
        rateLimitState.value.priceBatchInProgress = false
      }
    }

    await processQueue()
  }

  // Actions
  async function fetchRatings(newFilters?: Partial<RatingsFilters>, forceRefresh = false) {
    const now = Date.now()

    // If we have recent data and no new filters, skip the call unless forced
    if (!forceRefresh && !newFilters && ratings.value.length > 0 && lastUpdated.value) {
      const timeSinceLastUpdate = now - lastUpdated.value.getTime()
      if (timeSinceLastUpdate < 30000) {
        // 30 seconds cache
        console.log('🔄 Using cached ratings data (less than 30s old)')
        const symbols = ratings.value.map((r) => r.ticker).filter(Boolean)
        if (symbols.length > 0) {
          Promise.all([batchLoadLogoData(symbols), batchLoadPriceData(symbols)])
        }
        return
      }
    }

    // Prevent duplicate calls with multiple checks
    if (callTracker.value.fetchRatingsInProgress) {
      console.log('🔄 fetchRatings already in progress, waiting for completion...')

      // Wait for the current call to complete instead of just skipping
      return new Promise((resolve) => {
        const checkInterval = setInterval(() => {
          if (!callTracker.value.fetchRatingsInProgress) {
            clearInterval(checkInterval)
            resolve(undefined)
          }
        }, 50)

        // Timeout after 10 seconds to prevent infinite waiting
        setTimeout(() => {
          clearInterval(checkInterval)
          resolve(undefined)
        }, 10000)
      })
    }

    // Prevent rapid successive calls (within 500ms for better throttling)
    if (now - callTracker.value.lastRatingsCall < 500) {
      console.log('🔄 fetchRatings called too quickly, skipping (throttled)')
      return
    }

    try {
      console.log('🚀 fetchRatings STARTING with filters:', newFilters)
      // Removed stack trace logging to reduce console noise

      callTracker.value.fetchRatingsInProgress = true
      callTracker.value.lastRatingsCall = now
      setLoading(true)
      clearError()

      if (newFilters) {
        filters.value = { ...filters.value, ...newFilters }
      }

      const response: PaginatedResponse<StockRating> = await apiService.getRatings(filters.value)

      // Defensive checks for response structure
      if (!response) {
        throw new Error('Invalid API response: no response received')
      }

      // Handle case where response.data might be undefined but response itself is valid
      if (!response.data) {
        console.warn('⚠️ API response missing data field, using empty array:', response)
        response.data = []
      }

      if (!Array.isArray(response.data)) {
        console.warn(
          '⚠️ API response data is not an array, converting:',
          typeof response.data,
          response.data,
        )
        response.data = []
      }

      if (!response.pagination) {
        console.warn('⚠️ API response missing pagination, using defaults')
        response.pagination = {
          page: 1,
          limit: 20,
          total_items: response.data?.length || 0,
          total_pages: 1,
        }
      }

      // Deduplicate ratings at the store level to prevent duplicate data from backend
      const uniqueRatings = response.data.reduce((acc, rating) => {
        // Create a unique key using multiple fields
        const uniqueKey = `${rating.ticker}-${rating.brokerage}-${rating.rating_to}-${rating.target_to}-${rating.time}`

        // Only add if this combination doesn't exist yet
        if (!acc.has(uniqueKey)) {
          acc.set(uniqueKey, rating)
        }
        // Don't log every single duplicate - it's too noisy

        return acc
      }, new Map())

      const deduplicatedRatings = Array.from(uniqueRatings.values())

      // Only log if there were significant duplicates
      if (deduplicatedRatings.length !== response.data.length) {
        const duplicatesCount = response.data.length - deduplicatedRatings.length
        console.log(
          `📊 Filtered ${duplicatesCount} duplicate ratings from backend: ${response.data.length} → ${deduplicatedRatings.length}`,
        )
      }

      ratings.value = deduplicatedRatings

      // Update pagination to reflect deduplicated data
      pagination.value = {
        ...response.pagination,
        // Update the limit to show actual items on this page
        limit: deduplicatedRatings.length,
        // Keep total_items as is since that represents the total in the database
        // but note that individual pages will have fewer items due to deduplication
      }

      updateLastUpdated()

      // No batch loading needed - each component loads individually
      console.log(
        `📊 Loaded ${deduplicatedRatings.length} ratings (page ${response.pagination.page}/${response.pagination.total_pages})`,
      )

      // Trigger batch loading for logos and price data
      const symbols = ratings.value.map((r) => r.ticker).filter(Boolean)
      if (symbols.length > 0) {
        Promise.all([batchLoadLogoData(symbols), batchLoadPriceData(symbols)])
      }
    } catch (err) {
      const error = err as ApiError
      console.error('❌ Failed to fetch ratings:', error)
      setError(error.error || 'Failed to load ratings')

      // Reset to empty state on error to prevent undefined access
      ratings.value = []
      pagination.value = {
        page: 1,
        limit: 20,
        total_items: 0,
        total_pages: 0,
      }
    } finally {
      callTracker.value.fetchRatingsInProgress = false
      setLoading(false)
    }
  }

  async function fetchRatingsByTicker(ticker: string) {
    try {
      setLoading(true)
      clearError()

      const data = await apiService.getRatingsByTicker(ticker)

      console.log(`📊 Loaded ${data.length} ratings for ${ticker}`)
      return data
    } catch (err) {
      const error = err as ApiError
      console.error(`❌ Failed to fetch ratings for ${ticker}:`, error)
      setError(error.error || `Failed to load ratings for ${ticker}`)
      return []
    } finally {
      setLoading(false)
    }
  }

  async function fetchRecommendations(forceRefresh = false) {
    const now = Date.now()

    // If we have recent data, skip the call unless forced
    if (!forceRefresh && recommendations.value.length > 0 && lastUpdated.value) {
      const timeSinceLastUpdate = now - lastUpdated.value.getTime()
      if (timeSinceLastUpdate < 30000) {
        // 30 seconds cache
        console.log('🔄 Using cached recommendations data (less than 30s old)')
        const symbols = recommendations.value.map((r) => r.ticker).filter(Boolean)
        if (symbols.length > 0) {
          Promise.all([batchLoadLogoData(symbols), batchLoadPriceData(symbols)])
        }
        return
      }
    }

    // Prevent duplicate calls with multiple checks
    if (callTracker.value.fetchRecommendationsInProgress) {
      console.log('🔄 fetchRecommendations already in progress, waiting for completion...')

      // Wait for the current call to complete instead of just skipping
      return new Promise((resolve) => {
        const checkInterval = setInterval(() => {
          if (!callTracker.value.fetchRecommendationsInProgress) {
            clearInterval(checkInterval)
            resolve(undefined)
          }
        }, 50)

        // Timeout after 10 seconds to prevent infinite waiting
        setTimeout(() => {
          clearInterval(checkInterval)
          resolve(undefined)
        }, 10000)
      })
    }

    // Prevent rapid successive calls (within 500ms for better throttling)
    if (now - callTracker.value.lastRecommendationsCall < 500) {
      console.log('🔄 fetchRecommendations called too quickly, skipping (throttled)')
      return
    }

    try {
      console.log('🚀 fetchRecommendations STARTING')
      // Removed stack trace logging to reduce console noise

      callTracker.value.fetchRecommendationsInProgress = true
      callTracker.value.lastRecommendationsCall = now
      setLoading(true)
      clearError()

      let data = await apiService.getRecommendations()

      // Defensive check for recommendations response
      if (!Array.isArray(data)) {
        console.warn('⚠️ Recommendations response is not an array, converting:', typeof data, data)
        data = []
      }

      recommendations.value = data
      updateLastUpdated()

      console.log(`🎯 Loaded ${data.length} recommendations`)

      // Trigger batch loading for logos and price data
      const symbols = data.map((r) => r.ticker).filter(Boolean)
      if (symbols.length > 0) {
        Promise.all([batchLoadLogoData(symbols), batchLoadPriceData(symbols)])
      }
    } catch (err) {
      const error = err as ApiError
      console.error('❌ Failed to fetch recommendations:', error)
      setError(error.error || 'Failed to load recommendations')

      // Reset to empty state on error
      recommendations.value = []
    } finally {
      callTracker.value.fetchRecommendationsInProgress = false
      setLoading(false)
    }
  }

  // Logo cache functions only (no batch loading - components load individually)

  // Public function to get cached logo data
  function getLogoUrl(symbol: string): string {
    const cached = getCachedLogoData(symbol)
    if (cached?.logoUrl) {
      return cached.logoUrl
    }
    // Return fallback logo URL
    return `https://logo.clearbit.com/${symbol.toLowerCase()}.com`
  }

  async function searchRatings(searchQuery: string) {
    await fetchRatings({
      search: searchQuery,
      page: 1, // Reset to first page when searching
    })
  }

  async function sortRatings(
    sortBy: RatingsFilters['sort_by'],
    order: RatingsFilters['order'] = 'desc',
  ) {
    await fetchRatings({
      sort_by: sortBy,
      order,
      page: 1, // Reset to first page when sorting
    })
  }

  async function changePage(page: number) {
    await fetchRatings({ page })
  }

  async function changePageSize(limit: number) {
    await fetchRatings({
      limit,
      page: 1, // Reset to first page when changing page size
    })
  }

  async function triggerDataIngestion() {
    try {
      setLoading(true)
      clearError()

      const result = await apiService.triggerIngestion()
      console.log('🔄 Data ingestion triggered:', result)

      // Refresh data after ingestion
      await Promise.all([fetchRatings(), fetchRecommendations()])

      return result
    } catch (err) {
      const error = err as ApiError
      console.error('❌ Failed to trigger ingestion:', error)
      setError(error.error || 'Failed to trigger data ingestion')
      throw error
    } finally {
      setLoading(false)
    }
  }

  function resetFilters() {
    filters.value = {
      page: 1,
      limit: 20,
      sort_by: 'time',
      order: 'desc',
      search: '',
    }
  }

  function reset() {
    ratings.value = []
    recommendations.value = []
    pagination.value = {
      page: 1,
      limit: 20,
      total_items: 0,
      total_pages: 0,
    }
    loadingState.value = {
      isLoading: false,
      error: null,
      lastUpdated: null,
    }
    resetFilters()
  }

  // No price batch loading - components load individually with fresh data

  return {
    // State
    ratings,
    recommendations,
    pagination,
    filters,
    loadingState,
    logoCache,
    priceDataStore,

    // Computed
    isLoading,
    error,
    hasError,
    lastUpdated,
    totalRatings,
    currentPage,
    totalPages,
    pageLimit,
    topRecommendations,

    // Actions
    fetchRatings,
    fetchRatingsByTicker,
    fetchRecommendations,
    searchRatings,
    sortRatings,
    changePage,
    changePageSize,
    triggerDataIngestion,
    resetFilters,
    reset,
    clearError,

    // Cache functions
    getLogoUrl,
    getCachedLogoData,
    getPriceData,
    isPriceDataLoading,

    // Batch loading functions
    batchLoadLogoData,
    batchLoadPriceData,

    // Priority loading
    priorityLoadTrendingData,
  }
})
