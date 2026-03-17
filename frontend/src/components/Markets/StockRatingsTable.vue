<template>
  <div class="overflow-x-auto -mx-4 lg:-mx-8">
    <div class="inline-block min-w-full align-middle">
      <table class="min-w-full border-separate border-spacing-0">
        <thead>
          <tr class="border-t border-b border-gray-100">
            <th class="sticky top-0 z-10 border-b border-t border-gray-100 bg-white py-3 pl-4 pr-3 text-left text-[11px] font-bold text-gray-500 uppercase tracking-wider lg:pl-8 w-10">
              <div class="flex items-center space-x-1">
                <StarIcon class="w-3.5 h-3.5 text-gray-300" />
                <span>#</span>
              </div>
            </th>
            <th @click="handleHeaderClick('ticker')" class="sticky top-0 z-10 border-b border-t border-gray-100 bg-white py-3 px-3 text-left text-[11px] font-bold text-gray-500 uppercase tracking-wider cursor-pointer hover:bg-gray-50 transition-colors">
              <div class="flex items-center space-x-1">
                <span>Stock</span>
                <ChevronDownIcon v-if="sortBy === 'ticker'" class="w-3 h-3" :class="sortOrder === 'desc' ? '' : 'rotate-180'" />
              </div>
            </th>
            <th @click="handleHeaderClick('target_to')" class="sticky top-0 z-10 border-b border-t border-gray-100 bg-white py-3 px-3 text-right text-[11px] font-bold text-gray-500 uppercase tracking-wider cursor-pointer hover:bg-gray-50 transition-colors">
              <div class="flex items-center justify-end space-x-1">
                <span>Price Target</span>
                <ChevronDownIcon v-if="sortBy === 'target_to'" class="w-3 h-3" :class="sortOrder === 'desc' ? '' : 'rotate-180'" />
              </div>
            </th>
            <th @click="handleHeaderClick('rating_to')" class="sticky top-0 z-10 border-b border-t border-gray-100 bg-white py-3 px-3 text-right text-[11px] font-bold text-gray-500 uppercase tracking-wider cursor-pointer hover:bg-gray-50 transition-colors">
              <div class="flex items-center justify-end space-x-1">
                <span>Rating</span>
                <ChevronDownIcon v-if="sortBy === 'rating_to'" class="w-3 h-3" :class="sortOrder === 'desc' ? '' : 'rotate-180'" />
              </div>
            </th>
            <th @click="handleHeaderClick('brokerage')" class="sticky top-0 z-10 border-b border-t border-gray-100 bg-white py-3 px-3 text-right text-[11px] font-bold text-gray-500 uppercase tracking-wider cursor-pointer hover:bg-gray-50 transition-colors">
              <div class="flex items-center justify-end space-x-1">
                <span>Analyst</span>
                <ChevronDownIcon v-if="sortBy === 'brokerage'" class="w-3 h-3" :class="sortOrder === 'desc' ? '' : 'rotate-180'" />
              </div>
            </th>
            <th class="sticky top-0 z-10 border-b border-t border-gray-100 bg-white py-3 pl-3 pr-4 text-right text-[11px] font-bold text-gray-500 uppercase tracking-wider lg:pr-8">
              Last 7 Days
            </th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-100 bg-white">
          <tr
            v-for="(rating, index) in sortedRatings"
            :key="`rating-${rating.ticker}-${index}`"
            class="hover:bg-gray-50 transition-colors cursor-pointer group"
            @click="$router.push(`/stock/${rating.ticker}`)"
          >
            <td class="whitespace-nowrap py-4 pl-4 pr-3 text-[13px] text-gray-500 lg:pl-8">
              <div class="flex items-center space-x-2">
                <StarIcon class="w-4 h-4 text-gray-300 hover:text-yellow-400 transition-colors" />
                <span>{{ (currentPage - 1) * pageSize + index + 1 }}</span>
              </div>
            </td>

            <td class="whitespace-nowrap px-3 py-4">
              <div class="flex items-center">
                <StockLogo :symbol="rating.ticker" size="sm" class="w-6 h-6 mr-3" />
                <div class="flex flex-col">
                  <span class="text-[14px] font-bold text-gray-900 group-hover:text-gecko-green-600 transition-colors">{{ rating.ticker }}</span>
                  <span class="text-[12px] text-gray-500 truncate max-w-[120px]">{{ rating.company }}</span>
                </div>
              </div>
            </td>

            <td class="whitespace-nowrap px-3 py-4 text-right">
              <div class="flex flex-col items-end">
                <span class="text-[14px] font-bold text-gray-900">${{ (rating.target_to || 0).toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 }) }}</span>
                <div v-if="rating.target_from" class="flex items-center text-[11px] mt-0.5">
                  <span :class="getTargetTrendColor(rating)" class="flex items-center font-bold">
                    <component :is="getTargetTrendIcon(rating)" class="w-2.5 h-2.5 mr-0.5" />
                    {{ getTargetPercentChange(rating) }}%
                  </span>
                </div>
              </div>
            </td>

            <td class="whitespace-nowrap px-3 py-4 text-right">
              <span
                :class="getRatingColor(rating.rating_to)"
                class="inline-flex px-2 py-0.5 text-[11px] font-bold rounded-md uppercase tracking-tight"
              >
                {{ rating.rating_to }}
              </span>
            </td>

            <td class="whitespace-nowrap px-3 py-4 text-right">
              <div class="flex flex-col items-end">
                <span class="text-[13px] font-medium text-gray-900">{{ rating.brokerage }}</span>
                <span class="text-[11px] text-gray-400">{{ formatDate(rating.time) }}</span>
              </div>
            </td>

            <td class="whitespace-nowrap pl-3 pr-4 py-4 text-right lg:pr-8">
              <div class="w-24 h-10 ml-auto">
                <MiniChart :symbol="rating.ticker" :rating="rating" period="1W" />
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { StarIcon } from '@heroicons/vue/24/outline'
import { ChevronDownIcon, ChevronUpIcon, MinusIcon } from '@heroicons/vue/20/solid'
import StockLogo from '@/components/StockLogo.vue'
import MiniChart from '@/components/MiniChart.vue'
import type { StockRating, RatingsFilters } from '@/types'

interface Props {
  ratings: StockRating[]
  sortBy: RatingsFilters['sort_by']
  sortOrder: RatingsFilters['order']
  currentPage: number
  pageSize: number
}

interface Emits {
  (e: 'sort', column: string): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const getRatingPriority = (rating: string): number => {
  const r = rating.toLowerCase()
  if (r.includes('strong buy') || r.includes('outperform')) return 5
  if (r.includes('buy')) return 4
  if (r.includes('hold') || r.includes('neutral')) return 3
  if (r.includes('underweight') || r.includes('underperform')) return 2
  if (r.includes('sell') || r.includes('strong sell')) return 1
  return 0
}

const sortedRatings = computed(() => {
  if (!Array.isArray(props.ratings)) return []
  return [...props.ratings]
})

const getRatingColor = (rating: string) => {
  const r = rating.toLowerCase()
  if (r.includes('buy') || r.includes('strong')) return 'bg-gecko-green-50 text-gecko-green-600'
  if (r.includes('sell')) return 'bg-red-50 text-red-600'
  if (r.includes('hold')) return 'bg-yellow-50 text-yellow-600'
  return 'bg-gray-50 text-gray-500'
}

const getTargetPercentChange = (rating: StockRating) => {
  if (!rating.target_from || !rating.target_to) return '0.0'
  const change = ((rating.target_to - rating.target_from) / rating.target_from) * 100
  return Math.abs(change).toFixed(1)
}

const getTargetTrendIcon = (rating: StockRating) => {
  const targetTo = rating.target_to || 0
  const targetFrom = rating.target_from || 0
  if (targetFrom && targetTo > targetFrom) return ChevronUpIcon
  if (targetFrom && targetTo < targetFrom) return ChevronDownIcon
  return MinusIcon
}

const getTargetTrendColor = (rating: StockRating) => {
  const targetTo = rating.target_to || 0
  const targetFrom = rating.target_from || 0
  if (targetFrom && targetTo > targetFrom) return 'text-gecko-green-500'
  if (targetFrom && targetTo < targetFrom) return 'text-red-500'
  return 'text-gray-400'
}

const formatDate = (dateString: string) => {
  return new Intl.DateTimeFormat('en-US', {
    month: 'short',
    day: 'numeric',
  }).format(new Date(dateString))
}

const handleHeaderClick = (column: string) => {
  emit('sort', column)
}
</script>

