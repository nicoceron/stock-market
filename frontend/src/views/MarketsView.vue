<template>
  <div class="space-y-4">
    <!-- Page header -->
    <div class="bg-white p-4">
      <div>
        <h1 class="text-3xl font-bold text-gray-900">Stock Markets</h1>
      </div>
    </div>

    <!-- Trending and Top Gainers section -->
    <div class="grid grid-cols-1 gap-4">
      <TrendingRecommendations
        :recommendations="stocksStore.recommendations"
        :is-loading="stocksStore.isLoading"
      />
    </div>

    <!-- Main table -->
    <div class="bg-white">
      <SearchAndFilters
        v-model:search-query="searchQuery"
        v-model:page-size="pageSize"
        @search="handleSearch"
        @page-size-change="handlePageSizeChange"
      />

      <StockRatingsTable
        :ratings="stocksStore.ratings"
        :sort-by="sortBy"
        :sort-order="sortOrder"
        :current-page="stocksStore.currentPage"
        :page-size="pageSize"
        @sort="handleSort"
      />

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
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useStocksStore } from '@/stores/stocks'
import type { RatingsFilters } from '@/types'
import TrendingRecommendations from '@/components/Markets/TrendingRecommendations.vue'
import RecentUpgrades from '@/components/Markets/RecentUpgrades.vue'
import SearchAndFilters from '@/components/Markets/SearchAndFilters.vue'
import StockRatingsTable from '@/components/Markets/StockRatingsTable.vue'
import MarketsPagination from '@/components/Markets/MarketsPagination.vue'

const stocksStore = useStocksStore()

const searchQuery = ref('')
const sortBy = ref<RatingsFilters['sort_by']>('time')
const sortOrder = ref<RatingsFilters['order']>('desc')
const pageSize = ref(20)

const handleSearch = (query: string) => {
  stocksStore.searchRatings(query)
}

const handlePageSizeChange = (size: number) => {
  stocksStore.changePageSize(size)
}

const handleSort = (column: string) => {
  if (sortBy.value === column) {
    sortOrder.value = sortOrder.value === 'desc' ? 'asc' : 'desc'
  } else {
    sortBy.value = column as RatingsFilters['sort_by']
    sortOrder.value = 'desc'
  }
}

onMounted(async () => {
  console.log('🔄 MarketsView mounted, loading data...')

  try {
    console.log('🔥 Loading trending recommendations with priority...')
    await stocksStore.priorityLoadTrendingData()

    console.log('📊 Loading remaining data...')
    await stocksStore.fetchRatings()
  } catch (error) {
    console.error('Failed to load data:', error)
  }
})
</script>
