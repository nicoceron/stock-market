<template>
  <div class="bg-white shadow rounded-lg p-3 flex flex-col">
    <div class="flex items-center justify-between mb-2">
      <h3 class="text-lg font-semibold text-gray-900 flex items-center">🚀 Recent Upgrades</h3>
    </div>

    <div v-if="recentUpgrades && recentUpgrades.length > 0" class="space-y-2 flex-1">
      <div
        v-for="(rating, index) in recentUpgrades.slice(0, 3)"
        :key="`upgrade-${rating.ticker}-${index}`"
        class="flex items-center justify-between p-3 hover:bg-gray-50 rounded-lg cursor-pointer border border-gray-100"
        @click="$router.push(`/stock/${rating.ticker}`)"
      >
        <div class="flex items-center space-x-3">
          <StockLogo :symbol="rating.ticker" size="sm" />
          <div>
            <div class="text-sm font-medium text-gray-900">{{ rating.ticker }}</div>
            <div class="text-xs text-gray-500">{{ rating.brokerage }}</div>
          </div>
        </div>
        <div class="flex items-center space-x-4">
          <div class="text-right">
            <div class="text-lg font-bold text-gray-900">
              ${{ (rating.target_to || 0).toFixed(2) }}
            </div>
            <div class="text-xs text-gray-500">Price Target</div>
          </div>

          <div class="flex items-center space-x-2">
            <div
              v-if="rating.rating_from && rating.rating_from !== rating.rating_to"
              class="flex items-center space-x-2"
            >
              <span
                class="inline-flex px-2 py-1 text-xs font-medium rounded-lg bg-gray-100 text-gray-500 border line-through"
              >
                {{ rating.rating_from }}
              </span>
              <div class="text-blue-500 text-sm font-bold">→</div>
            </div>
            <span
              class="inline-flex px-3 py-1.5 text-xs font-bold rounded-lg bg-green-100 text-green-800 border border-green-200"
            >
              {{ rating.rating_to }}
            </span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import StockLogo from '@/components/StockLogo.vue'
import type { StockRating } from '@/types'

interface Props {
  ratings?: StockRating[]
}

const props = withDefaults(defineProps<Props>(), {
  ratings: () => [],
})

const recentUpgrades = computed(() => {
  if (!Array.isArray(props.ratings)) return []

  const buyRatings = props.ratings
    .filter((rating) => rating.rating_to && rating.rating_to.toLowerCase().includes('buy'))
    .sort((a, b) => new Date(b.time).getTime() - new Date(a.time).getTime())

  // Deduplicate by ticker
  const seen = new Set()
  return buyRatings.filter((rating) => {
    if (seen.has(rating.ticker)) return false
    seen.add(rating.ticker)
    return true
  })
})
</script>
