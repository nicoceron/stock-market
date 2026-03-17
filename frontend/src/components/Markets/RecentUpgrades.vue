<template>
  <div class="bg-white border border-gray-100 rounded-xl p-4 h-full shadow-sm hover:shadow-md transition-shadow duration-200">
    <div class="flex items-center justify-between mb-4">
      <h3 class="text-[15px] font-bold text-gray-900 flex items-center">
        <span class="mr-1.5">🚀</span> Upgrades
      </h3>
    </div>

    <div v-if="recentUpgrades && recentUpgrades.length > 0" class="space-y-3">
      <div
        v-for="(rating, index) in recentUpgrades.slice(0, 3)"
        :key="`upgrade-${rating.ticker}-${index}`"
        class="flex items-center justify-between group cursor-pointer"
        @click="$router.push(`/stock/${rating.ticker}`)"
      >
        <div class="flex items-center space-x-2.5">
          <span class="text-[12px] font-medium text-gray-400 w-3">{{ index + 1 }}</span>
          <StockLogo :symbol="rating.ticker" size="xs" class="w-5 h-5" />
          <div class="flex flex-col">
            <span class="text-[13px] font-bold text-gray-900 group-hover:text-gecko-green-600 transition-colors">{{ rating.ticker }}</span>
          </div>
        </div>
        <div class="flex items-center space-x-3">
          <span class="text-[13px] font-bold text-gray-900">${{ (rating.target_to || 0).toFixed(2) }}</span>
          <span class="text-[12px] font-bold text-gecko-green-500 flex items-center bg-gecko-green-50 px-1.5 py-0.5 rounded">
            {{ rating.rating_to }}
          </span>
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

