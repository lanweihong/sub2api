<template>
  <BaseDialog
    :show="show"
    :title="t('admin.accounts.zhipuUsage.detail')"
    width="extra-wide"
    @close="emit('close')"
  >
    <div v-if="usage?.zhipu_detail" class="space-y-5">
      <!-- Header: Platform badge -->
      <div class="flex items-center gap-3">
        <span
          :class="[
            'inline-block rounded px-2 py-0.5 text-xs font-semibold',
            usage.zhipu_detail.platform === 'zai'
              ? 'bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-300'
              : 'bg-purple-100 text-purple-700 dark:bg-purple-900/40 dark:text-purple-300'
          ]"
        >
          {{
            usage.zhipu_detail.platform === 'zai'
              ? t('admin.accounts.zhipuUsage.platform.zai')
              : t('admin.accounts.zhipuUsage.platform.zhipu')
          }}
        </span>
      </div>

      <!-- Quota Cards: 5h / 7d / MCP -->
      <div class="grid grid-cols-3 gap-3">
        <!-- 5h Quota Card -->
        <div class="rounded-lg border border-gray-200 bg-white p-3 dark:border-gray-700 dark:bg-gray-800">
          <div class="mb-1 flex items-center gap-1 text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.accounts.zhipuUsage.fiveHourQuota') }}
          </div>
          <div v-if="usage.five_hour" class="space-y-2">
            <div class="text-xl font-bold text-gray-900 dark:text-white">
              {{ Math.round(usage.five_hour.utilization) }}%
              <span class="text-xs font-normal text-gray-400">{{ t('admin.accounts.zhipuUsage.used') }}</span>
            </div>
            <div class="h-1.5 w-full overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700">
              <div
                class="h-full rounded-full transition-all"
                :class="barColor(usage.five_hour.utilization)"
                :style="{ width: `${Math.min(usage.five_hour.utilization, 100)}%` }"
              />
            </div>
            <div v-if="usage.five_hour.resets_at" class="text-[10px] text-gray-400 dark:text-gray-500">
              {{ t('admin.accounts.zhipuUsage.resetTime') }}: {{ formatResetTime(usage.five_hour.resets_at) }}
            </div>
          </div>
          <div v-else class="py-2 text-center text-xs text-gray-400">-</div>
        </div>

        <!-- 7d Quota Card -->
        <div class="rounded-lg border border-gray-200 bg-white p-3 dark:border-gray-700 dark:bg-gray-800">
          <div class="mb-1 flex items-center gap-1 text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.accounts.zhipuUsage.weeklyQuota') }}
          </div>
          <div v-if="usage.seven_day" class="space-y-2">
            <div class="text-xl font-bold text-gray-900 dark:text-white">
              {{ Math.round(usage.seven_day.utilization) }}%
              <span class="text-xs font-normal text-gray-400">{{ t('admin.accounts.zhipuUsage.used') }}</span>
            </div>
            <div class="h-1.5 w-full overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700">
              <div
                class="h-full rounded-full transition-all"
                :class="barColor(usage.seven_day.utilization)"
                :style="{ width: `${Math.min(usage.seven_day.utilization, 100)}%` }"
              />
            </div>
            <div v-if="usage.seven_day.resets_at" class="text-[10px] text-gray-400 dark:text-gray-500">
              {{ t('admin.accounts.zhipuUsage.resetTime') }}: {{ formatResetTime(usage.seven_day.resets_at) }}
            </div>
          </div>
          <div v-else class="py-2 text-center text-xs text-gray-400">-</div>
        </div>

        <!-- MCP Monthly Quota Card -->
        <div class="rounded-lg border border-gray-200 bg-white p-3 dark:border-gray-700 dark:bg-gray-800">
          <div class="mb-1 flex items-center gap-1 text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.accounts.zhipuUsage.monthlyQuota') }}
          </div>
          <div v-if="usage.zhipu_detail.monthly_mcp" class="space-y-2">
            <div class="text-xl font-bold text-gray-900 dark:text-white">
              {{ Math.round(usage.zhipu_detail.monthly_mcp.percentage) }}%
              <span class="text-xs font-normal text-gray-400">{{ t('admin.accounts.zhipuUsage.used') }}</span>
            </div>
            <div class="h-1.5 w-full overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700">
              <div
                class="h-full rounded-full transition-all"
                :class="barColor(usage.zhipu_detail.monthly_mcp.percentage)"
                :style="{ width: `${Math.min(usage.zhipu_detail.monthly_mcp.percentage, 100)}%` }"
              />
            </div>
            <div v-if="usage.zhipu_detail.monthly_mcp.next_reset_time" class="text-[10px] text-gray-400 dark:text-gray-500">
              {{ t('admin.accounts.zhipuUsage.resetTime') }}: {{ formatResetTime(usage.zhipu_detail.monthly_mcp.next_reset_time) }}
            </div>
          </div>
          <div v-else class="py-2 text-center text-xs text-gray-400">-</div>
        </div>
      </div>

      <!-- Tabs + Period Selector -->
      <div class="flex items-center justify-between border-b border-gray-200 dark:border-gray-700">
        <nav class="-mb-px flex gap-4">
          <button
            v-for="tab in tabs"
            :key="tab.key"
            type="button"
            :class="[
              'whitespace-nowrap border-b-2 px-1 py-2 text-sm font-medium transition-colors',
              activeTab === tab.key
                ? 'border-primary-500 text-primary-600 dark:border-primary-400 dark:text-primary-400'
                : 'border-transparent text-gray-500 hover:border-gray-300 hover:text-gray-700 dark:text-gray-400 dark:hover:border-gray-600 dark:hover:text-gray-200'
            ]"
            @click="activeTab = tab.key"
          >
            {{ tab.label }}
          </button>
        </nav>
        <div class="mb-1 flex gap-1">
          <button
            v-for="p in periods"
            :key="p.key"
            type="button"
            :class="[
              'rounded-md px-2.5 py-1 text-xs font-medium transition-colors',
              activePeriod === p.key
                ? 'bg-primary-100 text-primary-700 dark:bg-primary-900/40 dark:text-primary-300'
                : 'text-gray-500 hover:bg-gray-100 dark:text-gray-400 dark:hover:bg-gray-800'
            ]"
            @click="activePeriod = p.key"
          >
            {{ p.label }}
          </button>
        </div>
      </div>

      <!-- Usage Detail Section -->
      <div class="space-y-4">
        <!-- Loading state -->
        <div v-if="loading" class="flex items-center justify-center py-12">
          <LoadingSpinner />
        </div>

        <!-- Error state -->
        <div v-else-if="fetchError" class="py-8 text-center text-sm text-red-500 dark:text-red-400">
          {{ fetchError }}
        </div>

        <!-- Content -->
        <template v-else-if="hasData">
          <!-- Summary cards -->
          <div class="flex gap-3 overflow-x-auto pb-1">
            <!-- Grand Total card -->
            <div class="flex min-w-[120px] flex-shrink-0 flex-col rounded-lg border border-gray-100 bg-gray-50 p-2.5 dark:border-gray-700 dark:bg-gray-800/50">
              <div class="flex items-center gap-1.5 text-[10px] text-gray-500 dark:text-gray-400">
                <span class="inline-block h-2 w-2 rounded-full bg-gray-400" />
                {{ activeTab === 'model' ? t('admin.accounts.zhipuUsage.totalTokenConsumption') : t('admin.accounts.zhipuUsage.totalToolCalls') }}
              </div>
              <div class="mt-1 text-lg font-bold text-gray-900 dark:text-white">
                {{ formatCompact(grandTotal) }}
              </div>
            </div>
            <!-- Per-item cards -->
            <div
              v-for="(item, idx) in summaryItems"
              :key="item.name"
              class="flex min-w-[110px] flex-shrink-0 flex-col rounded-lg border border-gray-100 bg-gray-50 p-2.5 dark:border-gray-700 dark:bg-gray-800/50"
            >
              <div class="flex items-center gap-1.5 text-[10px] text-gray-500 dark:text-gray-400">
                <span
                  class="inline-block h-2 w-2 rounded-full"
                  :style="{ backgroundColor: CHART_COLORS[idx % CHART_COLORS.length] }"
                />
                {{ item.name }} {{ activeTab === 'model' ? t('admin.accounts.zhipuUsage.consumption') : t('admin.accounts.zhipuUsage.calls') }}
              </div>
              <div class="mt-1 text-lg font-bold text-gray-900 dark:text-white">
                {{ formatCompact(item.value) }}
              </div>
            </div>
          </div>

          <!-- Chart -->
          <div class="h-64">
            <Line v-if="chartData" :data="chartData" :options="chartOptions" />
          </div>
        </template>

        <!-- Empty state -->
        <div v-else class="py-8 text-center text-sm text-gray-400 dark:text-gray-500">
          {{ t('admin.accounts.zhipuUsage.noData') }}
        </div>
      </div>
    </div>

    <!-- No zhipu_detail fallback -->
    <div v-else class="py-8 text-center text-sm text-gray-400 dark:text-gray-500">
      {{ t('admin.accounts.zhipuUsage.noData') }}
    </div>
  </BaseDialog>
</template>

<script setup lang="ts">
import { adminAPI } from '@/api/admin'
import BaseDialog from '@/components/common/BaseDialog.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import type { Account, AccountUsageInfo, ZhipuModelUsageResponse, ZhipuToolUsageResponse } from '@/types'
import {
    CategoryScale,
    Chart as ChartJS,
    Filler,
    Legend,
    LinearScale,
    LineElement,
    PointElement,
    Title,
    Tooltip
} from 'chart.js'
import { computed, ref, watch } from 'vue'
import { Line } from 'vue-chartjs'
import { useI18n } from 'vue-i18n'

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Title, Tooltip, Legend, Filler)

const CHART_COLORS = [
  '#3b82f6', '#8b5cf6', '#10b981', '#06b6d4', '#f59e0b',
  '#ef4444', '#ec4899', '#14b8a6', '#f97316', '#6366f1'
]

const props = defineProps<{
  show: boolean
  account: Account | null
  usage: AccountUsageInfo | null
}>()

const emit = defineEmits<{
  (e: 'close'): void
}>()

const { t } = useI18n()

type TabKey = 'model' | 'tool'
type PeriodKey = 'today' | '7d' | '30d'

const activeTab = ref<TabKey>('model')
const activePeriod = ref<PeriodKey>('30d')
const loading = ref(false)
const fetchError = ref('')

// Cache: key → data (avoid redundant API calls)
const cache = ref(new Map<string, ZhipuModelUsageResponse | ZhipuToolUsageResponse>())

const tabs = computed(() => [
  { key: 'model' as const, label: t('admin.accounts.zhipuUsage.modelUsage') },
  { key: 'tool' as const, label: t('admin.accounts.zhipuUsage.toolUsage') }
])

const periods = computed(() => [
  { key: 'today' as const, label: t('admin.accounts.zhipuUsage.today') },
  { key: '7d' as const, label: t('admin.accounts.zhipuUsage.last7Days') },
  { key: '30d' as const, label: t('admin.accounts.zhipuUsage.last30Days') }
])

// Current data for active tab+period
const currentData = ref<ZhipuModelUsageResponse | ZhipuToolUsageResponse | null>(null)

async function fetchData() {
  if (!props.account) return
  const cacheKey = `${activeTab.value}-${activePeriod.value}`
  if (cache.value.has(cacheKey)) {
    currentData.value = cache.value.get(cacheKey)!
    return
  }
  loading.value = true
  fetchError.value = ''
  try {
    const data = await adminAPI.accounts.getZhipuUsage(props.account.id, activeTab.value, activePeriod.value)
    cache.value.set(cacheKey, data)
    currentData.value = data
  } catch (err: any) {
    fetchError.value = err?.message || 'Failed to fetch usage data'
    currentData.value = null
  } finally {
    loading.value = false
  }
}

// Watch show → load on open, clear cache on close
watch(() => props.show, (val) => {
  if (val) {
    cache.value.clear()
    fetchData()
  } else {
    currentData.value = null
    fetchError.value = ''
  }
})

// Watch tab/period change → refetch
watch([activeTab, activePeriod], () => {
  if (props.show) fetchData()
})

// === Data processing ===

// Whether the current data contains any displayable content
const hasData = computed(() => {
  const d = currentData.value
  if (!d) return false
  if (activeTab.value === 'model') {
    const m = d as ZhipuModelUsageResponse
    return (m.modelDataList?.length > 0) || (m.x_time?.length > 0)
  }
  const t = d as ZhipuToolUsageResponse
  return (t.toolDataList?.length > 0) || (t.x_time?.length > 0)
})

// Summary items: from summaryList or totalUsage, sorted by value desc
const summaryItems = computed(() => {
  const d = currentData.value
  if (!d) return []
  if (activeTab.value === 'model') {
    const m = d as ZhipuModelUsageResponse
    const list = m.modelSummaryList || m.totalUsage?.modelSummaryList || []
    return list
      .map((s) => ({ name: s.modelName, value: s.totalTokens }))
      .sort((a, b) => b.value - a.value)
  }
  const t = d as ZhipuToolUsageResponse
  const list = t.toolSummaryList || t.totalUsage?.toolSummaryList || []
  if (list.length > 0) {
    return list
      .map((s) => ({ name: s.toolName, value: s.totalCallCount }))
      .sort((a, b) => b.value - a.value)
  }
  // Fallback: builtin tool counts from totalUsage
  const items: { name: string; value: number }[] = []
  if (t.totalUsage) {
    if (t.totalUsage.totalNetworkSearchCount) items.push({ name: 'NetworkSearch', value: t.totalUsage.totalNetworkSearchCount })
    if (t.totalUsage.totalWebReadMcpCount) items.push({ name: 'WebReadMCP', value: t.totalUsage.totalWebReadMcpCount })
    if (t.totalUsage.totalZreadMcpCount) items.push({ name: 'ZreadMCP', value: t.totalUsage.totalZreadMcpCount })
    if (t.totalUsage.totalSearchMcpCount) items.push({ name: 'SearchMCP', value: t.totalUsage.totalSearchMcpCount })
  }
  return items.sort((a, b) => b.value - a.value)
})

const grandTotal = computed(() => {
  const d = currentData.value
  if (!d) return 0
  if (activeTab.value === 'model') {
    const m = d as ZhipuModelUsageResponse
    return m.totalUsage?.totalTokensUsage ?? summaryItems.value.reduce((acc, i) => acc + i.value, 0)
  }
  const t = d as ZhipuToolUsageResponse
  const tu = t.totalUsage
  if (tu) {
    return (tu.totalNetworkSearchCount || 0) + (tu.totalWebReadMcpCount || 0) + (tu.totalZreadMcpCount || 0) + (tu.totalSearchMcpCount || 0)
  }
  return summaryItems.value.reduce((acc, i) => acc + i.value, 0)
})

// Chart data: x_time as labels, dataList items as datasets
const chartData = computed(() => {
  const d = currentData.value
  if (!d || !d.x_time?.length) return null

  // Format date labels (strip year for readability)
  const labels = d.x_time.map((dt) => {
    const parts = dt.split('-')
    return parts.length === 3 ? `${parts[1]}-${parts[2]}` : dt
  })

  if (activeTab.value === 'model') {
    const m = d as ZhipuModelUsageResponse
    const sorted = [...(m.modelDataList || [])].sort((a, b) => a.sortOrder - b.sortOrder)
    const datasets = sorted.map((item, idx) => ({
      label: item.modelName,
      data: item.tokensUsage || [],
      borderColor: CHART_COLORS[idx % CHART_COLORS.length],
      backgroundColor: `${CHART_COLORS[idx % CHART_COLORS.length]}20`,
      fill: false,
      tension: 0.3,
      borderWidth: 2,
      pointRadius: labels.length > 15 ? 0 : 3
    }))
    return { labels, datasets }
  }

  // Tool usage
  const t = d as ZhipuToolUsageResponse
  const datasets: any[] = []
  let idx = 0

  // Custom tool datasets from toolDataList
  if (t.toolDataList?.length) {
    const sorted = [...t.toolDataList].sort((a, b) => a.sortOrder - b.sortOrder)
    for (const item of sorted) {
      datasets.push({
        label: item.toolName,
        data: item.callCount || [],
        borderColor: CHART_COLORS[idx % CHART_COLORS.length],
        backgroundColor: `${CHART_COLORS[idx % CHART_COLORS.length]}20`,
        fill: false,
        tension: 0.3,
        borderWidth: 2,
        pointRadius: labels.length > 15 ? 0 : 3
      })
      idx++
    }
  }

  // Builtin tool time-series arrays
  const builtinSeries: { name: string; data: number[] }[] = []
  if (t.networkSearchCount?.some((v) => v > 0)) builtinSeries.push({ name: 'NetworkSearch', data: t.networkSearchCount })
  if (t.webReadMcpCount?.some((v) => v > 0)) builtinSeries.push({ name: 'WebReadMCP', data: t.webReadMcpCount })
  if (t.zreadMcpCount?.some((v) => v > 0)) builtinSeries.push({ name: 'ZreadMCP', data: t.zreadMcpCount })

  for (const s of builtinSeries) {
    datasets.push({
      label: s.name,
      data: s.data,
      borderColor: CHART_COLORS[idx % CHART_COLORS.length],
      backgroundColor: `${CHART_COLORS[idx % CHART_COLORS.length]}20`,
      fill: false,
      tension: 0.3,
      borderWidth: 2,
      pointRadius: labels.length > 15 ? 0 : 3
    })
    idx++
  }

  return datasets.length ? { labels, datasets } : null
})

const isDarkMode = computed(() => document.documentElement.classList.contains('dark'))

const chartOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  interaction: { intersect: false, mode: 'index' as const },
  plugins: {
    legend: {
      position: 'top' as const,
      labels: {
        color: isDarkMode.value ? '#e5e7eb' : '#374151',
        usePointStyle: true,
        pointStyle: 'circle',
        padding: 12,
        font: { size: 11 }
      }
    },
    tooltip: {
      callbacks: {
        label: (ctx: any) => `${ctx.dataset.label}: ${formatCompact(ctx.raw)}`
      }
    }
  },
  scales: {
    x: {
      grid: { color: isDarkMode.value ? '#374151' : '#e5e7eb' },
      ticks: { color: isDarkMode.value ? '#e5e7eb' : '#374151', font: { size: 10 } }
    },
    y: {
      grid: { color: isDarkMode.value ? '#374151' : '#e5e7eb' },
      ticks: {
        color: isDarkMode.value ? '#e5e7eb' : '#374151',
        font: { size: 10 },
        callback: (value: string | number) => formatCompact(Number(value))
      }
    }
  }
}))

// === Helpers ===

function barColor(pct: number): string {
  if (pct >= 90) return 'bg-red-500'
  if (pct >= 70) return 'bg-amber-500'
  return 'bg-blue-500'
}

function formatResetTime(dateStr: string): string {
  try {
    const d = new Date(dateStr)
    if (isNaN(d.getTime())) return dateStr
    const now = new Date()
    // Same day → show time only
    if (d.toDateString() === now.toDateString()) {
      return d.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' })
    }
    return d.toLocaleDateString(undefined, { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
  } catch {
    return dateStr
  }
}

function formatCompact(num: number): string {
  if (num == null || isNaN(num)) return '-'
  if (num >= 1e9) return (num / 1e9).toFixed(2) + 'B'
  if (num >= 1e6) return (num / 1e6).toFixed(2) + 'M'
  if (num >= 1e3) return (num / 1e3).toFixed(2) + 'K'
  return num.toLocaleString()
}
</script>
