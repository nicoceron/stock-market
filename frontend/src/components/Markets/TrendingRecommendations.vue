<template>
  <div class="bg-white border border-gray-100 rounded-xl p-4 h-full shadow-sm hover:shadow-md transition-shadow duration-200">
    <div class="flex items-center justify-between mb-4">
      <h3 class="text-[15px] font-bold text-gray-900 flex items-center">
        <span class="mr-1.5">🔥</span> Trending
      </h3>
      <router-link to="/recommendations" class="text-gecko-green-600 hover:underline text-[13px] font-semibold flex items-center">
        View more <ChevronRightIcon class="w-3.5 h-3.5 ml-0.5" />
      </router-link>
    </div>

    <div v-if="recommendations && recommendations.length > 0" class="space-y-3">
      <div
        v-for="(rec, index) in topRecommendations.slice(0, 3)"
        :key="rec.ticker"
        class="flex items-center justify-between group cursor-pointer"
        @click="$router.push(`/stock/${rec.ticker}`)"
      >
        <div class="flex items-center space-x-2.5">
          <span class="text-[12px] font-medium text-gray-400 w-3">{{ index + 1 }}</span>
          <StockLogo :symbol="rec.ticker" size="xs" class="w-5 h-5" />
          <div class="flex flex-col">
            <span class="text-[13px] font-bold text-gray-900 group-hover:text-gecko-green-600 transition-colors">{{ rec.ticker }}</span>
          </div>
        </div>
        <div class="flex items-center space-x-3">
          <span class="text-[13px] font-bold text-gray-900">${{ (rec.target_price || 0).toFixed(2) }}</span>
          <span :class="[ (rec.score || 0) > 0 ? 'text-gecko-green-500' : 'text-danger-500', 'text-[12px] font-bold flex items-center' ]">
            <ChevronUpIcon v-if="(rec.score || 0) > 0" class="w-3 h-3 mr-0.5" />
            <ChevronDownIcon v-else class="w-3 h-3 mr-0.5" />
            {{ (rec.score || 0).toFixed(1) }}%
          </span>
        </div>
      </div>
    </div>
    
    <div v-else-if="isLoading" class="space-y-4">
      <div v-for="i in 3" :key="i" class="flex items-center justify-between animate-pulse">
        <div class="flex items-center space-x-3">
          <div class="w-3 h-3 bg-gray-100 rounded"></div>
          <div class="w-6 h-6 bg-gray-100 rounded-full"></div>
          <div class="w-12 h-3 bg-gray-100 rounded"></div>
        </div>
        <div class="w-16 h-3 bg-gray-100 rounded"></div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { ChevronRightIcon, ChevronUpIcon, ChevronDownIcon } from '@heroicons/vue/20/solid'
import StockLogo from '@/components/StockLogo.vue'
import type { StockRecommendation } from '@/types'

interface Props {
  recommendations?: StockRecommendation[]
  isLoading?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  recommendations: () => [],
  isLoading: false,
})

const topRecommendations = computed(() => {
  if (!props.recommendations) return []
  return [...props.recommendations].sort((a, b) => (b.score || 0) - (a.score || 0))
})
</script>

