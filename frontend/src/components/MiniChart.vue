<template>
  <div class="w-full h-full">
    <svg :width="width" :height="height" viewBox="0 0 80 32" class="w-full h-full">
      <polyline
        v-if="dataPoints.length > 0"
        :points="trendPoints"
        fill="none"
        :stroke="trendColor"
        stroke-width="1.5"
        stroke-linecap="round"
        stroke-linejoin="round"
      />
      <!-- Loading indicator -->
      <text v-else-if="isLoading" x="40" y="16" text-anchor="middle" font-size="8" fill="#666">
        ...
      </text>
      <!-- Fallback to rating-based display if no price data -->
      <polyline
        v-else-if="fallbackPoints.length > 0"
        :points="fallbackTrendPoints"
        fill="none"
        :stroke="fallbackTrendColor"
        stroke-width="1.5"
        stroke-linecap="round"
        stroke-linejoin="round"
        opacity="0.6"
      />
      <!-- No data indicator -->
      <text v-else x="40" y="16" text-anchor="middle" font-size="6" fill="#999">--</text>
    </svg>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useStocksStore } from '@/stores/stocks'

interface Rating {
  target_to?: number
  target_from?: number
  rating_to?: string
}

interface PriceBar {
  timestamp: string
  close: number
}

interface Props {
  symbol: string
  rating?: Rating
  width?: number
  height?: number
  period?: string
}

const props = withDefaults(defineProps<Props>(), {
  width: 80,
  height: 32,
  period: '1W',
})

// Store
const stocksStore = useStocksStore()

// Get price data from store (uses data from batch loading)
const priceData = computed(() => {
  if (!props.symbol) return []

  const storeData = stocksStore.priceDataStore[props.symbol]
  if (!storeData?.bars) return []

  return storeData.bars.map((bar: PriceBar) => ({
    timestamp: bar.timestamp,
    close: bar.close,
  }))
})

// Check if data is loading
const isLoading = computed(() => {
  return stocksStore.isPriceDataLoading(props.symbol)
})

// Calculate trend based on actual price data
const trend = computed(() => {
  if (priceData.value.length < 2) return 'up'
  const firstPrice = priceData.value[0].close
  const lastPrice = priceData.value[priceData.value.length - 1].close
  return lastPrice >= firstPrice ? 'up' : 'down'
})

const trendColor = computed(() => {
  return trend.value === 'up' ? '#8cc351' : '#ea3943'
})

// Generate data points from real price data
const dataPoints = computed(() => {
  if (priceData.value.length === 0) return []

  const prices = priceData.value.map((bar) => bar.close)
  const minPrice = Math.min(...prices)
  const maxPrice = Math.max(...prices)
  const priceRange = maxPrice - minPrice || maxPrice * 0.05 // 5% buffer for flat prices

  // Chart area with padding
  const chartWidth = 80
  const chartHeight = 32
  const paddingX = 4
  const paddingY = 6
  const usableWidth = chartWidth - paddingX * 2
  const usableHeight = chartHeight - paddingY * 2

  return priceData.value.map((bar, index) => {
    // Horizontal distribution
    const x = paddingX + (index / Math.max(1, priceData.value.length - 1)) * usableWidth
    
    // Vertical distribution (normalized 0 to 1, then flipped)
    const normalizedPrice = (bar.close - minPrice) / priceRange
    const y = paddingY + usableHeight - normalizedPrice * usableHeight
    
    return { x, y: Math.max(paddingY, Math.min(chartHeight - paddingY, y)) }
  })
})

const trendPoints = computed(() => {
  return dataPoints.value.map((point) => `${point.x},${point.y}`).join(' ')
})

// Fallback: Generate simple trend based on rating data
const fallbackTrend = computed(() => {
  if (!props.rating) return 'up'

  const targetTo = props.rating.target_to || 0
  const targetFrom = props.rating.target_from || 0
  const ratingTo = props.rating.rating_to?.toLowerCase() || ''

  if (targetTo > targetFrom) return 'up'
  if (targetTo < targetFrom) return 'down'
  if (ratingTo.includes('buy') || ratingTo.includes('strong')) return 'up'
  if (ratingTo.includes('sell')) return 'down'

  return 'up'
})

const fallbackTrendColor = computed(() => {
  return fallbackTrend.value === 'up' ? '#8cc351' : '#ea3943'
})

const fallbackPoints = computed(() => {
  if (!props.rating) return []

  const isUp = fallbackTrend.value === 'up'
  const startY = isUp ? 25 : 7
  const endY = isUp ? 7 : 25

  return [
    { x: 5, y: startY },
    { x: 75, y: endY },
  ]
})

const fallbackTrendPoints = computed(() => {
  return fallbackPoints.value.map((point) => `${point.x},${point.y}`).join(' ')
})
</script>

<style scoped>
svg {
  display: block;
}
</style>
