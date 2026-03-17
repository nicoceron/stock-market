<template>
  <div class="space-y-6">
    <!-- Page header -->
    <div class="mb-8">
      <h1 class="text-[24px] font-bold text-gray-900 mb-2">Stock Prices by Analyst Rating</h1>
      <div class="flex items-center text-[13px] text-gray-600">
        <span>The global stock market cap today is $108.4 Trillion, a</span>
        <span class="text-gecko-green-500 font-bold flex items-center mx-1">
          <ChevronUpIcon class="w-3.5 h-3.5 mr-0.5" />
          0.4%
        </span>
        <span>change in the last 24 hours.</span>
      </div>
    </div>

    <!-- Top Cards Section -->
    <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
      <!-- Market Cap Card -->
      <div class="lg:col-span-1 bg-white border border-gray-100 rounded-xl p-5 shadow-sm hover:shadow-md transition-shadow">
        <div class="space-y-6">
          <div>
            <div class="text-[18px] font-bold text-gray-900 mb-1">$108,412,941,021,162</div>
            <div class="flex items-center text-[12px] text-gray-500">
              <span>Market Cap</span>
              <span class="text-gecko-green-500 font-bold flex items-center ml-1">
                <ChevronUpIcon class="w-3 h-3 mr-0.5" />
                0.4%
              </span>
            </div>
            <div class="mt-2 h-10 w-full opacity-30">
              <svg viewBox="0 0 100 20" class="w-full h-full text-gecko-green-500 fill-current">
                <path d="M0 20 L0 10 L10 12 L20 8 L30 15 L40 5 L50 10 L60 2 L70 8 L80 4 L90 12 L100 0 L100 20 Z" />
              </svg>
            </div>
          </div>
          <div>
            <div class="text-[18px] font-bold text-gray-900 mb-1">$452,155,969,196,671</div>
            <div class="text-[12px] text-gray-500">24h Trading Volume</div>
            <div class="mt-2 h-10 w-full opacity-30">
              <svg viewBox="0 0 100 20" class="w-full h-full text-gecko-green-500 fill-current">
                <path d="M0 20 L0 15 L10 18 L20 12 L30 14 L40 8 L50 10 L60 5 L70 7 L80 2 L90 5 L100 0 L100 20 Z" />
              </svg>
            </div>
          </div>
        </div>
      </div>

      <!-- Trending Card -->
      <TrendingRecommendations
        :recommendations="stocksStore.recommendations"
        :is-loading="stocksStore.isLoading"
      />

      <!-- Upgrades Card -->
      <RecentUpgrades :ratings="stocksStore.ratings" />
    </div>

    <!-- Main table -->
    <div class="bg-white">
      <SearchAndFilters />
      
      <StockRatingsTable
        :ratings="stocksStore.ratings"
        :sort-by="sortBy"
        :sort-order="sortOrder"
        :current-page="stocksStore.currentPage"
        :page-size="pageSize"
        @sort="handleSort"
      />

      <div class="mt-6 flex justify-between items-center">
        <div class="text-[13px] text-gray-600">
          Showing {{ (stocksStore.currentPage - 1) * pageSize + 1 }} - {{ Math.min(stocksStore.currentPage * pageSize, stocksStore.totalRatings) }} of {{ stocksStore.totalRatings }} stocks
        </div>
        <MarketsPagination
          :current-page="stocksStore.currentPage"
          :total-pages="stocksStore.totalPages"
          :page-limit="stocksStore.pageLimit"
          :ratings-count="stocksStore.ratings.length"
          :total-ratings="stocksStore.totalRatings"
          @change-page="stocksStore.changePage"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ChevronUpIcon, AdjustmentsHorizontalIcon } from '@heroicons/vue/20/solid'
import { useStocksStore } from '@/stores/stocks'
import type { RatingsFilters } from '@/types'
import TrendingRecommendations from '@/components/Markets/TrendingRecommendations.vue'
import RecentUpgrades from '@/components/Markets/RecentUpgrades.vue'
import StockRatingsTable from '@/components/Markets/StockRatingsTable.vue'
import MarketsPagination from '@/components/Markets/MarketsPagination.vue'

const stocksStore = useStocksStore()

const sortBy = ref<RatingsFilters['sort_by']>('time')
const sortOrder = ref<RatingsFilters['order']>('desc')
const pageSize = ref(20)

const handleSort = (column: string) => {
  if (sortBy.value === column) {
    sortOrder.value = sortOrder.value === 'desc' ? 'asc' : 'desc'
  } else {
    sortBy.value = column as RatingsFilters['sort_by']
    sortOrder.value = 'desc'
  }
  stocksStore.sortRatings(sortBy.value, sortOrder.value)
}

onMounted(async () => {
  try {
    await stocksStore.priorityLoadTrendingData()
    await stocksStore.fetchRatings()
  } catch (error) {
    console.error('Failed to load data:', error)
  }
})
</script>

