<template>
  <div class="min-h-screen bg-white font-sans text-gray-900">
    <!-- Global Stats Bar (Top) -->
    <div class="bg-white border-b border-gray-100 py-1.5 text-[11px] font-medium text-gray-500 overflow-x-auto whitespace-nowrap hidden md:block">
      <div class="mx-auto px-4 lg:px-8 max-w-[1344px] flex items-center space-x-6">
        <div class="flex items-center">
          <span>Stocks:</span>
          <span class="text-gecko-green-500 ml-1">{{ stocksStore.totalRatings.toLocaleString() }}</span>
        </div>
        <div class="flex items-center">
          <span>Recommendations:</span>
          <span class="text-gecko-green-500 ml-1">{{ stocksStore.recommendations.length }}</span>
        </div>
        <div class="flex items-center">
          <span>Market Cap:</span>
          <span class="text-gecko-green-500 ml-1">$108.4T</span>
          <span class="text-gecko-green-500 ml-1 flex items-center">
            <ChevronUpIcon class="w-3 h-3" />
            0.4%
          </span>
        </div>
        <div class="flex items-center">
          <span>24h Vol:</span>
          <span class="text-gecko-green-500 ml-1">$452.1B</span>
        </div>
        <div class="flex items-center">
          <span>Dominance:</span>
          <span class="text-gecko-green-500 ml-1">AAPL 6.4% MSFT 5.8%</span>
        </div>
        <div class="flex-grow"></div>
      </div>
    </div>

    <!-- Navigation / Header -->
    <header class="bg-white border-b border-gray-100 sticky top-0 z-50">
      <div class="mx-auto px-4 lg:px-8 max-w-[1344px]">
        <div class="flex justify-between items-center h-16 lg:h-[72px]">
          <!-- Logo and main navigation -->
          <div class="flex items-center lg:space-x-8">
            <router-link to="/" class="flex items-center space-x-2 mr-6">
              <div class="w-8 h-8 bg-gecko-green-500 rounded-full flex items-center justify-center">
                <ChartBarIcon class="w-5 h-5 text-white" />
              </div>
              <span class="text-[22px] font-bold tracking-tight text-gray-900">StockAnalyzer</span>
            </router-link>

            <!-- Desktop navigation -->
            <nav class="hidden lg:flex items-center space-x-6">
              <router-link
                v-for="item in navigation"
                :key="item.name"
                :to="item.href"
                :class="[
                  item.current ? 'text-gray-900' : 'text-gray-700 hover:text-gecko-green-500',
                  'text-[15px] font-semibold transition-colors duration-200',
                ]"
              >
                {{ item.name }}
              </router-link>
            </nav>
          </div>

          <!-- Right side items: Search -->
          <div class="flex items-center space-x-4 flex-1 justify-end max-w-xl">
            <!-- Search Bar -->
            <div class="relative w-full max-w-[280px] hidden md:block search-container">
              <div class="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                <MagnifyingGlassIcon class="h-4 w-4 text-gray-400" />
              </div>
              <input
                type="text"
                v-model="searchQuery"
                @input="handleSearch"
                @focus="showSuggestions = suggestions.length > 0"
                placeholder="Search"
                class="block w-full pl-10 pr-3 py-1.5 bg-gray-100 border-none rounded-lg text-sm focus:ring-1 focus:ring-gecko-green-500 focus:bg-white transition-all duration-200"
              />

              <!-- Search Suggestions Dropdown -->
              <transition
                enter-active-class="transition duration-100 ease-out"
                enter-from-class="transform scale-95 opacity-0"
                enter-to-class="transform scale-100 opacity-100"
                leave-active-class="transition duration-75 ease-in"
                leave-from-class="transform scale-100 opacity-100"
                leave-to-class="transform scale-95 opacity-0"
              >
                <div v-if="showSuggestions" class="absolute left-0 right-0 mt-2 bg-white rounded-xl shadow-xl border border-gray-100 overflow-hidden z-50">
                  <div class="max-h-[400px] overflow-y-auto">
                    <div v-for="suggestion in suggestions" :key="suggestion.ticker" 
                      @click="selectSuggestion(suggestion.ticker)"
                      class="px-4 py-3 hover:bg-gray-50 flex items-center justify-between cursor-pointer border-b border-gray-50 last:border-0 group"
                    >
                      <div class="flex items-center space-x-3">
                        <StockLogo :symbol="suggestion.ticker" size="xs" />
                        <div class="flex flex-col">
                          <span class="text-sm font-bold text-gray-900 group-hover:text-gecko-green-600 transition-colors">{{ suggestion.ticker }}</span>
                          <span class="text-xs text-gray-500 truncate max-w-[120px]">{{ suggestion.company }}</span>
                        </div>
                      </div>
                      <div class="text-right">
                        <div class="text-[13px] font-bold text-gray-900">${{ (suggestion.target_to || 0).toFixed(2) }}</div>
                      </div>
                    </div>
                  </div>
                </div>
              </transition>
            </div>

            <div class="flex items-center">
              <!-- Mobile menu button -->
              <button
                @click="mobileMenuOpen = !mobileMenuOpen"
                class="lg:hidden p-2 rounded-md text-gray-500 hover:bg-gray-100 focus:outline-none"
              >
                <Bars3Icon v-if="!mobileMenuOpen" class="h-6 w-6" />
                <XMarkIcon v-else class="h-6 w-6" />
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- Mobile menu -->
      <transition
        enter-active-class="transition duration-150 ease-out"
        enter-from-class="transform -translate-y-4 opacity-0"
        enter-to-class="transform translate-y-0 opacity-100"
        leave-active-class="transition duration-100 ease-in"
        leave-from-class="transform translate-y-0 opacity-100"
        leave-to-class="transform -translate-y-4 opacity-0"
      >
        <div v-show="mobileMenuOpen" class="lg:hidden bg-white border-t border-gray-100 shadow-lg">
          <div class="px-4 pt-2 pb-6 space-y-1">
            <router-link
              v-for="item in navigation"
              :key="item.name"
              :to="item.href"
              @click="mobileMenuOpen = false"
              :class="[
                item.current ? 'bg-gray-50 text-gecko-green-600' : 'text-gray-700 hover:bg-gray-50',
                'block px-3 py-3 text-base font-semibold rounded-lg',
              ]"
            >
              {{ item.name }}
            </router-link>

            <div class="pt-4 pb-2">
              <div class="relative px-3 search-container">
                <input
                  type="text"
                  v-model="searchQuery"
                  @input="handleSearch"
                  placeholder="Search"
                  class="block w-full pl-10 pr-3 py-2 bg-gray-100 border-none rounded-lg text-sm focus:ring-1 focus:ring-gecko-green-500"
                />
                <div class="absolute inset-y-0 left-6 flex items-center">
                  <MagnifyingGlassIcon class="h-4 w-4 text-gray-400" />
                </div>

                <!-- Mobile Suggestions -->
                <div v-if="showSuggestions" class="absolute left-3 right-3 mt-1 bg-white rounded-lg shadow-lg border border-gray-100 overflow-hidden z-50">
                  <div v-for="suggestion in suggestions" :key="suggestion.ticker" 
                    @click="selectSuggestion(suggestion.ticker)"
                    class="px-4 py-3 hover:bg-gray-50 flex items-center justify-between border-b border-gray-50 last:border-0"
                  >
                    <div class="flex items-center space-x-3">
                      <StockLogo :symbol="suggestion.ticker" size="xs" />
                      <div class="flex flex-col">
                        <span class="text-sm font-bold text-gray-900">{{ suggestion.ticker }}</span>
                        <span class="text-xs text-gray-500">{{ suggestion.company }}</span>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </transition>
    </header>

    <!-- Error alert -->
    <div v-if="stocksStore.hasError" class="mx-auto mt-4 px-4 lg:px-8 max-w-[1344px]">
      <div class="bg-red-50 border border-red-100 rounded-xl p-4 flex items-center justify-between">
        <div class="flex items-center">
          <ExclamationTriangleIcon class="h-5 w-5 text-red-500 mr-3" />
          <p class="text-sm font-medium text-red-800">{{ stocksStore.error }}</p>
        </div>
        <button @click="stocksStore.clearError" class="text-red-400 hover:text-red-600 transition-colors">
          <XMarkIcon class="h-5 w-5" />
        </button>
      </div>
    </div>

    <!-- Loading indicator (Subtle) -->
    <div v-if="stocksStore.isLoading" class="fixed top-0 left-0 w-full h-1 z-[60]">
      <div class="h-full bg-gecko-green-500 animate-pulse w-full"></div>
    </div>

    <!-- Main content -->
    <main class="mx-auto py-6 px-4 lg:px-8 max-w-[1344px]">
      <router-view />
    </main>

    <!-- Footer (Basic) -->
    <footer class="bg-white border-t border-gray-100 py-12 mt-12">
      <div class="mx-auto px-4 lg:px-8 max-w-[1344px]">
        <div class="flex flex-col md:flex-row justify-between items-center text-sm text-gray-500">
          <div class="flex items-center space-x-2 mb-4 md:mb-0">
            <div class="w-6 h-6 bg-gecko-green-500 rounded-full flex items-center justify-center">
              <ChartBarIcon class="w-4 h-4 text-white" />
            </div>
            <span class="font-bold text-gray-900">StockAnalyzer</span>
            <span>&copy; 2026. All rights reserved.</span>
          </div>
        </div>
      </div>
    </footer>


  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  ChartBarIcon,
  Bars3Icon,
  XMarkIcon,
  ExclamationTriangleIcon,
  MagnifyingGlassIcon,
} from '@heroicons/vue/24/outline'
import { ChevronUpIcon } from '@heroicons/vue/20/solid'
import { useStocksStore } from '@/stores/stocks'
import { apiService } from '@/services/api'
import StockLogo from '@/components/StockLogo.vue'
import type { NavItem, StockRating } from '@/types'

// Store and Router
const stocksStore = useStocksStore()
const route = useRoute()
const router = useRouter()

// Local state
const mobileMenuOpen = ref(false)
const searchQuery = ref('')
const suggestions = ref<StockRating[]>([])
const showSuggestions = ref(false)
let searchTimeout: ReturnType<typeof setTimeout> | null = null

// Navigation items
const navigation = ref<NavItem[]>([
  { name: 'Markets', href: '/', current: false },
  { name: 'Recommendations', href: '/recommendations', current: false },
])

// Update navigation current state based on route
const updateNavigationState = () => {
  navigation.value.forEach((item) => {
    item.current =
      route.path === item.href || (route.path.startsWith('/stock/') && item.href === '/')
  })
}

// Search handler
const handleSearch = async () => {
  if (searchTimeout) clearTimeout(searchTimeout)
  
  if (!searchQuery.value.trim()) {
    suggestions.value = []
    showSuggestions.value = false
    stocksStore.searchRatings('')
    return
  }

  searchTimeout = setTimeout(async () => {
    // 1. Update the table results (current behavior)
    stocksStore.searchRatings(searchQuery.value)
    
    // 2. Fetch suggestions for the dropdown
    try {
      const response = await apiService.getRatings({ 
        search: searchQuery.value, 
        limit: 8 
      })
      suggestions.value = response.data
      showSuggestions.value = suggestions.value.length > 0
    } catch (error) {
      console.error('Failed to fetch suggestions:', error)
      suggestions.value = []
    }
  }, 300)
}

const selectSuggestion = (ticker: string) => {
  router.push(`/stock/${ticker}`)
  searchQuery.value = ''
  showSuggestions.value = false
  suggestions.value = []
}

// Close suggestions on click outside
onMounted(() => {
  updateNavigationState()
  document.addEventListener('click', (e) => {
    const target = e.target as HTMLElement
    if (!target.closest('.search-container')) {
      showSuggestions.value = false
    }
  })
})

// Watch route changes to update navigation
watch(() => route.path, updateNavigationState)
</script>

<style scoped>
.font-sans {
  font-family: 'Inter', sans-serif;
}
</style>

