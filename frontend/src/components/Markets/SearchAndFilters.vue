<template>
  <div class="p-4">
    <div class="grid grid-cols-1 gap-4">
      <div class="relative" ref="searchContainer">
        <div class="relative">
          <input
            v-model="searchQuery"
            type="text"
            placeholder="Search by ticker, company..."
            class="block w-full pr-10 border-gray-300 rounded-md focus:ring-blue-500 focus:border-blue-500 text-sm font-medium text-gray-900 placeholder-gray-500"
            @input="handleInput"
            @focus="showDropdown = true"
          />
          <div class="absolute inset-y-0 right-0 pr-3 flex items-center">
            <MagnifyingGlassIcon v-if="!isSearching" class="h-5 w-5 text-gray-400" />
            <div v-else class="animate-spin h-4 w-4 border-2 border-blue-500 border-t-transparent rounded-full"></div>
          </div>
        </div>

        <!-- Search Results Dropdown -->
        <div
          v-if="showDropdown && (searchResults.length > 0 || searchQuery)"
          class="absolute z-50 mt-1 w-full bg-white shadow-lg max-h-60 rounded-md py-1 text-base ring-1 ring-black ring-opacity-5 overflow-auto focus:outline-none sm:text-sm"
        >
          <div v-if="isSearching" class="px-4 py-2 text-gray-500">Searching...</div>
          <template v-else-if="searchResults.length > 0">
            <div
              v-for="result in searchResults"
              :key="result.rating_id"
              class="cursor-pointer select-none relative py-2 pl-3 pr-9 hover:bg-blue-50 transition-colors"
              @click="handleSelect(result)"
            >
              <div class="flex items-center">
                <StockLogo :symbol="result.ticker" size="xs" class="mr-2" />
                <span class="font-bold text-gray-900 mr-2">{{ result.ticker }}</span>
                <span class="text-gray-500 truncate">{{ result.company }}</span>
              </div>
            </div>
          </template>
          <div v-else-if="searchQuery" class="px-4 py-2 text-gray-500">No results found for "{{ searchQuery }}"</div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { MagnifyingGlassIcon } from '@heroicons/vue/20/solid'
import { useStocksStore } from '@/stores/stocks'
import StockLogo from '@/components/StockLogo.vue'
import type { StockRating } from '@/types'

const router = useRouter()
const stocksStore = useStocksStore()

const searchQuery = ref('')
const searchResults = ref<StockRating[]>([])
const isSearching = ref(false)
const showDropdown = ref(false)
const searchContainer = ref<HTMLElement | null>(null)

let debounceTimeout: ReturnType<typeof setTimeout>

const handleInput = () => {
  clearTimeout(debounceTimeout)
  showDropdown.value = true

  if (!searchQuery.value) {

    searchResults.value = []
    return
  }

  isSearching.value = true
  debounceTimeout = setTimeout(async () => {
    searchResults.value = await stocksStore.searchRatings(searchQuery.value)
    isSearching.value = false
  }, 300)
}

const handleSelect = (result: StockRating) => {
  searchQuery.value = ''
  searchResults.value = []
  showDropdown.value = false
  router.push(`/stock/${result.ticker}`)
}

// Close dropdown when clicking outside
const handleClickOutside = (event: MouseEvent) => {
  if (searchContainer.value && !searchContainer.value.contains(event.target as Node)) {
    showDropdown.value = false
  }
}

onMounted(() => {
  document.addEventListener('click', handleClickOutside)
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
  clearTimeout(debounceTimeout)
})
</script>

</script>
