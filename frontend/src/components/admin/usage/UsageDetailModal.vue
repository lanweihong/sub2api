<template>
  <BaseDialog :show="show" :title="t('admin.usage.detailTitle')" width="extra-wide" @close="emit('close')">
    <div v-if="loading" class="flex items-center justify-center py-12">
      <div class="h-8 w-8 animate-spin rounded-full border-2 border-primary-600 border-t-transparent" />
    </div>
    <div v-else class="space-y-6">
      <!-- Basic info -->
      <section>
        <h3 class="mb-3 text-sm font-semibold text-gray-700 dark:text-gray-300">
          {{ t('admin.usage.basicInfo') }}
        </h3>
        <div class="grid grid-cols-2 gap-3 text-sm lg:grid-cols-4">
          <div class="rounded-lg bg-gray-50 px-3 py-2 dark:bg-dark-800">
            <span class="text-xs text-gray-500 dark:text-gray-400">ID</span>
            <div class="font-medium text-gray-900 dark:text-white">{{ record?.id }}</div>
          </div>
          <div class="rounded-lg bg-gray-50 px-3 py-2 dark:bg-dark-800">
            <span class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.usage.model') }}</span>
            <div class="font-medium text-gray-900 dark:text-white">{{ record?.model }}</div>
            <div v-if="record?.upstream_model && record.upstream_model !== record.model" class="text-xs text-gray-400">
              ↳ {{ record.upstream_model }}
            </div>
          </div>
          <div class="rounded-lg bg-gray-50 px-3 py-2 dark:bg-dark-800">
            <span class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.usage.inputTokens') }}</span>
            <div class="font-medium text-gray-900 dark:text-white">{{ record?.input_tokens?.toLocaleString() || 0 }}</div>
          </div>
          <div class="rounded-lg bg-gray-50 px-3 py-2 dark:bg-dark-800">
            <span class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.usage.outputTokens') }}</span>
            <div class="font-medium text-gray-900 dark:text-white">{{ record?.output_tokens?.toLocaleString() || 0 }}</div>
          </div>
          <div class="rounded-lg bg-gray-50 px-3 py-2 dark:bg-dark-800">
            <span class="text-xs text-gray-500 dark:text-gray-400">{{ t('usage.userBilled') }}</span>
            <div class="font-medium text-green-600 dark:text-green-400">${{ record?.actual_cost?.toFixed(6) || '0.000000' }}</div>
          </div>
          <div class="rounded-lg bg-gray-50 px-3 py-2 dark:bg-dark-800">
            <span class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.usage.duration') }}</span>
            <div class="font-medium text-gray-900 dark:text-white">{{ formatDuration(record?.duration_ms) }}</div>
          </div>
          <div class="rounded-lg bg-gray-50 px-3 py-2 dark:bg-dark-800">
            <span class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.usage.user') }}</span>
            <div class="font-medium text-gray-900 dark:text-white">{{ record?.user?.email || '-' }}</div>
          </div>
          <div class="rounded-lg bg-gray-50 px-3 py-2 dark:bg-dark-800">
            <span class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.usage.createdAt') }}</span>
            <div class="font-medium text-gray-900 dark:text-white">{{ record?.created_at ? formatDateTime(record.created_at) : '-' }}</div>
          </div>
        </div>
      </section>

      <!-- Payload error -->
      <p v-if="payloadError" class="text-sm text-red-500 dark:text-red-400">
        {{ t('admin.usage.payloadLoadError') }}
      </p>

      <!-- Request body -->
      <section>
        <div class="flex items-center justify-between">
          <h3 class="text-sm font-semibold text-gray-700 dark:text-gray-300">
            {{ t('admin.usage.requestBody') }}
            <span v-if="payload?.request_truncated" class="ml-2 text-xs text-amber-500">
              {{ t('admin.usage.truncated') }}
            </span>
          </h3>
          <button
            v-if="payload?.request_body"
            class="text-xs font-medium text-primary-600 hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300"
            @click="copyToClipboard(payload!.request_body!, t('admin.usage.requestBody'))"
          >
            {{ t('common.copy') }}
          </button>
        </div>
        <pre
          v-if="payload?.request_body"
          class="mt-2 max-h-80 overflow-auto rounded-lg bg-gray-50 p-4 text-xs leading-relaxed text-gray-800 dark:bg-dark-800 dark:text-gray-200"
        >{{ formatJSON(payload.request_body) }}</pre>
        <p v-else class="mt-2 text-sm text-gray-400 dark:text-gray-500">
          {{ t('admin.usage.noPayload') }}
        </p>
      </section>

      <!-- Response body -->
      <section>
        <div class="flex items-center justify-between">
          <h3 class="text-sm font-semibold text-gray-700 dark:text-gray-300">
            {{ t('admin.usage.responseBody') }}
            <span v-if="payload?.response_truncated" class="ml-2 text-xs text-amber-500">
              {{ t('admin.usage.truncated') }}
            </span>
          </h3>
          <button
            v-if="payload?.response_body"
            class="text-xs font-medium text-primary-600 hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300"
            @click="copyToClipboard(payload!.response_body!, t('admin.usage.responseBody'))"
          >
            {{ t('common.copy') }}
          </button>
        </div>
        <pre
          v-if="payload?.response_body"
          class="mt-2 max-h-80 overflow-auto rounded-lg bg-gray-50 p-4 text-xs leading-relaxed text-gray-800 dark:bg-dark-800 dark:text-gray-200"
        >{{ formatJSON(payload.response_body) }}</pre>
        <p v-else class="mt-2 text-sm text-gray-400 dark:text-gray-500">
          {{ t('admin.usage.noPayload') }}
        </p>
      </section>
    </div>
  </BaseDialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { adminAPI } from '@/api'
import { useClipboard } from '@/composables/useClipboard'
import { formatDateTime } from '@/utils/format'
import type { AdminUsageLog } from '@/types'
import type { UsageLogPayload } from '@/api/admin/usage'

const { t } = useI18n()
const { copyToClipboard } = useClipboard()

const props = defineProps<{
  show: boolean
  record: AdminUsageLog | null
}>()

const emit = defineEmits<{
  (e: 'close'): void
}>()

const payload = ref<UsageLogPayload | null>(null)
const payloadError = ref(false)
const loading = ref(false)

watch(
  () => props.show,
  async (visible) => {
    if (visible && props.record) {
      loading.value = true
      payload.value = null
      payloadError.value = false
      try {
        payload.value = await adminAPI.usage.getPayload(props.record.id)
      } catch {
        payloadError.value = true
      } finally {
        loading.value = false
      }
    }
  }
)

function formatDuration(ms: number | null | undefined): string {
  if (ms == null) return '-'
  if (ms < 1000) return `${ms}ms`
  return `${(ms / 1000).toFixed(2)}s`
}

function formatJSON(text: string): string {
  try {
    return JSON.stringify(JSON.parse(text), null, 2)
  } catch {
    return text
  }
}
</script>
