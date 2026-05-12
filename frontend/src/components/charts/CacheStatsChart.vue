<template>
  <div class="card p-4">
    <div class="mb-4 flex flex-wrap items-center gap-3">
      <div>
        <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
          {{ t('admin.usage.cacheStatsTitle') }}
        </h3>
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.usage.cacheStatsSubtitle') }}
        </p>
      </div>
      <div class="ml-auto flex flex-wrap items-center gap-2">
        <div class="w-32">
          <Select v-model="dimensionModel" :options="dimensionOptions" />
        </div>
        <div v-if="dimensionModel === 'model'" class="w-32">
          <Select v-model="modelSourceModel" :options="modelSourceOptions" />
        </div>
        <div v-if="dimensionModel === 'endpoint'" class="w-32">
          <Select v-model="endpointSourceModel" :options="endpointSourceOptions" />
        </div>
      </div>
    </div>

    <div class="mb-4 grid grid-cols-2 gap-3 lg:grid-cols-4">
      <div class="rounded-md border border-gray-200 p-3 dark:border-dark-600">
        <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.usage.cacheReadRate') }}</p>
        <p class="mt-1 text-lg font-semibold text-sky-600 dark:text-sky-400">
          {{ formatPercent(summary.cache_read_rate) }}
        </p>
      </div>
      <div class="rounded-md border border-gray-200 p-3 dark:border-dark-600">
        <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.usage.cacheTokenRate') }}</p>
        <p class="mt-1 text-lg font-semibold text-violet-600 dark:text-violet-400">
          {{ formatPercent(summary.cache_token_rate) }}
        </p>
      </div>
      <div class="rounded-md border border-gray-200 p-3 dark:border-dark-600">
        <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.usage.cacheReadTokens') }}</p>
        <p class="mt-1 text-lg font-semibold text-gray-900 dark:text-white">
          {{ formatTokens(summary.cache_read_tokens) }}
        </p>
      </div>
      <div class="rounded-md border border-gray-200 p-3 dark:border-dark-600">
        <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.usage.cacheCreationTokens') }}</p>
        <p class="mt-1 text-lg font-semibold text-gray-900 dark:text-white">
          {{ formatTokens(summary.cache_creation_tokens) }}
        </p>
      </div>
    </div>

    <div v-if="loading" class="flex h-56 items-center justify-center">
      <LoadingSpinner />
    </div>
    <div v-else-if="items.length > 0 && chartData" class="h-56">
      <Bar :data="chartData" :options="chartOptions" />
    </div>
    <div v-else class="flex h-56 items-center justify-center text-sm text-gray-500 dark:text-gray-400">
      {{ t('admin.dashboard.noDataAvailable') }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  BarElement,
  CategoryScale,
  Chart as ChartJS,
  Legend,
  LinearScale,
  Tooltip
} from 'chart.js'
import { Bar } from 'vue-chartjs'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import Select from '@/components/common/Select.vue'
import type {
  CacheStatsDimension,
  CacheStatsEndpointSource,
  CacheStatsItem,
  CacheStatsModelSource
} from '@/types/cacheStats'

ChartJS.register(CategoryScale, LinearScale, BarElement, Tooltip, Legend)

const { t } = useI18n()

const props = withDefaults(defineProps<{
  items: CacheStatsItem[]
  summary: CacheStatsItem | null
  dimension: CacheStatsDimension
  modelSource: CacheStatsModelSource
  endpointSource: CacheStatsEndpointSource
  loading?: boolean
}>(), {
  loading: false
})

const emit = defineEmits<{
  (e: 'update:dimension', value: CacheStatsDimension): void
  (e: 'update:modelSource', value: CacheStatsModelSource): void
  (e: 'update:endpointSource', value: CacheStatsEndpointSource): void
}>()

const dimensionModel = computed({
  get: () => props.dimension,
  set: (value) => emit('update:dimension', value as CacheStatsDimension)
})

const modelSourceModel = computed({
  get: () => props.modelSource,
  set: (value) => emit('update:modelSource', value as CacheStatsModelSource)
})

const endpointSourceModel = computed({
  get: () => props.endpointSource,
  set: (value) => emit('update:endpointSource', value as CacheStatsEndpointSource)
})

const emptySummary: CacheStatsItem = {
  key: 'summary',
  label: 'Summary',
  requests: 0,
  input_tokens: 0,
  output_tokens: 0,
  cache_creation_tokens: 0,
  cache_read_tokens: 0,
  total_tokens: 0,
  cache_token_rate: 0,
  cache_read_rate: 0,
  cache_write_rate: 0,
  cost: 0,
  actual_cost: 0,
  account_cost: 0
}

const summary = computed(() => props.summary || emptySummary)

const dimensionOptions = computed(() => [
  { value: 'account', label: t('admin.usage.dimensionAccount') },
  { value: 'user', label: t('admin.usage.dimensionUser') },
  { value: 'api_key', label: t('admin.usage.dimensionApiKey') },
  { value: 'group', label: t('admin.usage.dimensionGroup') },
  { value: 'model', label: t('admin.usage.dimensionModel') },
  { value: 'endpoint', label: t('admin.usage.dimensionEndpoint') },
  { value: 'day', label: t('admin.usage.dimensionDay') },
  { value: 'hour', label: t('admin.usage.dimensionHour') }
])

const modelSourceOptions = computed(() => [
  { value: 'requested', label: t('usage.requestedModel') },
  { value: 'upstream', label: t('usage.upstreamModel') },
  { value: 'mapping', label: t('usage.mapping') }
])

const endpointSourceOptions = computed(() => [
  { value: 'inbound', label: t('usage.inboundEndpoint') },
  { value: 'upstream', label: t('usage.upstreamEndpoint') }
])

const isDarkMode = computed(() => document.documentElement.classList.contains('dark'))

const colors = computed(() => ({
  text: isDarkMode.value ? '#e5e7eb' : '#374151',
  grid: isDarkMode.value ? '#374151' : '#e5e7eb',
  readRate: '#0ea5e9',
  tokenRate: '#8b5cf6'
}))

const truncateLabel = (value: string) => {
  if (value.length <= 24) return value
  return `${value.slice(0, 21)}...`
}

const chartData = computed(() => {
  if (!props.items.length) return null
  return {
    labels: props.items.map((item) => truncateLabel(item.label || item.key)),
    datasets: [
      {
        label: t('admin.usage.cacheReadRate'),
        data: props.items.map((item) => item.cache_read_rate * 100),
        backgroundColor: colors.value.readRate
      },
      {
        label: t('admin.usage.cacheTokenRate'),
        data: props.items.map((item) => item.cache_token_rate * 100),
        backgroundColor: colors.value.tokenRate
      }
    ]
  }
})

const chartOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: {
      position: 'top' as const,
      labels: {
        color: colors.value.text,
        usePointStyle: true,
        pointStyle: 'rectRounded',
        font: { size: 11 }
      }
    },
    tooltip: {
      callbacks: {
        title: (items: any[]) => {
          const index = items[0]?.dataIndex
          return props.items[index]?.label || ''
        },
        label: (context: any) => `${context.dataset.label}: ${context.raw.toFixed(1)}%`,
        afterBody: (items: any[]) => {
          const index = items[0]?.dataIndex
          const item = props.items[index]
          if (!item) return []
          return [
            `${t('admin.usage.cacheReadTokens')}: ${formatTokens(item.cache_read_tokens)}`,
            `${t('admin.usage.cacheCreationTokens')}: ${formatTokens(item.cache_creation_tokens)}`,
            `${t('usage.totalRequests')}: ${item.requests.toLocaleString()}`
          ]
        }
      }
    }
  },
  scales: {
    x: {
      grid: { color: colors.value.grid },
      ticks: {
        color: colors.value.text,
        font: { size: 10 }
      }
    },
    y: {
      min: 0,
      max: 100,
      grid: { color: colors.value.grid },
      ticks: {
        color: colors.value.text,
        font: { size: 10 },
        callback: (value: string | number) => `${value}%`
      }
    }
  }
}))

const formatPercent = (value: number) => `${(value * 100).toFixed(1)}%`

const formatTokens = (value: number): string => {
  if (value >= 1_000_000_000) return `${(value / 1_000_000_000).toFixed(2)}B`
  if (value >= 1_000_000) return `${(value / 1_000_000).toFixed(2)}M`
  if (value >= 1_000) return `${(value / 1_000).toFixed(2)}K`
  return value.toLocaleString()
}
</script>
