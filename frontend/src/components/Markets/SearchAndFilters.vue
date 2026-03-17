<template>
  <div class="p-4">
    <div class="grid grid-cols-1 gap-4">
      <div>
        <div class="relative">
          <input
            :value="searchQuery"
            type="text"
            placeholder="Search by ticker, company..."
            class="block w-full pr-10 border-gray-300 rounded-md focus:ring-blue-500 focus:border-blue-500 text-sm font-medium text-gray-900 placeholder-gray-500"
            @input="handleSearchInput"
          />
          <div class="absolute inset-y-0 right-0 pr-3 flex items-center">
            <MagnifyingGlassIcon class="h-5 w-5 text-gray-400" />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { MagnifyingGlassIcon } from '@heroicons/vue/24/outline'

interface Props {
  searchQuery: string
  pageSize: number
}

interface Emits {
  (e: 'update:searchQuery', value: string): void
  (e: 'update:pageSize', value: number): void
  (e: 'search', value: string): void
  (e: 'pageSizeChange', value: number): void
}

defineProps<Props>()
const emit = defineEmits<Emits>()

let searchTimeout: ReturnType<typeof setTimeout> | null = null

const handleSearchInput = (event: Event) => {
  const target = event.target as HTMLInputElement
  const value = target.value

  emit('update:searchQuery', value)

  if (searchTimeout) clearTimeout(searchTimeout)
  searchTimeout = setTimeout(() => {
    emit('search', value)
  }, 300)
}

const handlePageSizeChange = (event: Event) => {
  const target = event.target as HTMLSelectElement
  const value = parseInt(target.value)

  emit('update:pageSize', value)
  emit('pageSizeChange', value)
}
</script>
