<template>
  <BaseDialog
    :show="show"
    :title="t('admin.users.batchCreate.title')"
    width="full"
    :close-on-click-outside="!isCompleted"
    @close="handleClose"
  >
    <div
      v-if="isCompleted"
      class="rounded-2xl border border-emerald-200 bg-emerald-50 p-8 dark:border-emerald-900/30 dark:bg-emerald-900/10"
    >
      <div class="mx-auto flex max-w-xl flex-col items-center text-center">
        <div class="flex h-14 w-14 items-center justify-center rounded-full bg-emerald-100 text-emerald-600 dark:bg-emerald-900/30 dark:text-emerald-300">
          <Icon name="check" size="lg" :stroke-width="2.5" />
        </div>
        <h3 class="mt-4 text-xl font-semibold text-gray-900 dark:text-white">
          {{ t('admin.users.batchCreate.completedTitle') }}
        </h3>
        <p class="mt-2 text-sm leading-6 text-gray-600 dark:text-dark-300">
          {{ t('admin.users.batchCreate.completedDescription', { count: completedCount }) }}
        </p>
        <div class="mt-6 grid w-full grid-cols-1 gap-3 rounded-xl border border-emerald-200/70 bg-white/80 p-4 text-left dark:border-emerald-900/30 dark:bg-dark-900/70 sm:grid-cols-2">
          <div>
            <div class="text-xs uppercase tracking-wider text-gray-500 dark:text-dark-400">
              {{ t('admin.users.batchCreate.createdCount') }}
            </div>
            <div class="mt-1 text-lg font-semibold text-gray-900 dark:text-white">{{ completedCount }}</div>
          </div>
          <div>
            <div class="text-xs uppercase tracking-wider text-gray-500 dark:text-dark-400">
              {{ t('admin.users.batchCreate.downloadIncludes') }}
            </div>
            <div class="mt-1 text-sm font-medium text-gray-900 dark:text-white">
              {{ t('admin.users.username') }} / {{ t('admin.users.email') }} / {{ t('admin.users.password') }}
            </div>
          </div>
        </div>
      </div>
    </div>

    <div v-else class="space-y-4">
      <p class="text-sm text-gray-600 dark:text-dark-300">
        {{ t('admin.users.batchCreate.hint') }}
      </p>

      <div
        class="flex flex-col gap-3 rounded-xl border border-dashed border-gray-300 bg-gray-50 p-4 dark:border-dark-600 dark:bg-dark-800"
      >
        <div class="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
          <div class="min-w-0">
            <div class="truncate text-sm font-medium text-gray-900 dark:text-white">
              {{ fileName || t('admin.users.batchCreate.selectFile') }}
            </div>
            <div class="text-xs text-gray-500 dark:text-dark-400">{{ t('admin.users.batchCreate.fileFormat') }}</div>
          </div>
          <button type="button" class="btn btn-secondary shrink-0" :disabled="previewing || submitting" @click="openFilePicker">
            {{ previewing ? t('admin.users.batchCreate.previewing') : t('common.chooseFile') }}
          </button>
        </div>
        <input
          ref="fileInput"
          type="file"
          class="hidden"
          accept=".txt,text/plain"
          @change="handleFileChange"
        />
      </div>

      <div
        v-if="rows.length > 0"
        class="grid grid-cols-1 gap-3 rounded-xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-900 lg:grid-cols-4"
      >
        <div>
          <div class="text-xs uppercase tracking-wider text-gray-500 dark:text-dark-400">
            {{ t('admin.users.batchCreate.totalRows') }}
          </div>
          <div class="mt-1 text-lg font-semibold text-gray-900 dark:text-white">{{ rows.length }}</div>
        </div>
        <div>
          <div class="text-xs uppercase tracking-wider text-gray-500 dark:text-dark-400">
            {{ t('admin.users.batchCreate.validRows') }}
          </div>
          <div class="mt-1 text-lg font-semibold text-emerald-600 dark:text-emerald-400">{{ validRowCount }}</div>
        </div>
        <div>
          <div class="text-xs uppercase tracking-wider text-gray-500 dark:text-dark-400">
            {{ t('admin.users.batchCreate.errorRows') }}
          </div>
          <div class="mt-1 text-lg font-semibold text-red-600 dark:text-red-400">{{ errorRowCount }}</div>
        </div>
        <div class="flex items-end justify-start lg:justify-end">
          <button type="button" class="btn btn-secondary" :disabled="previewing || submitting" @click="regenerateAllPasswords">
            {{ t('admin.users.batchCreate.regeneratePasswords') }}
          </button>
        </div>
      </div>

      <div
        v-if="rows.length > 0"
        class="flex flex-col gap-3 rounded-xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-900 lg:flex-row lg:items-end"
      >
        <div class="w-full lg:max-w-[160px]">
          <label class="input-label">{{ t('admin.users.batchCreate.bulkBalance') }}</label>
          <input v-model.number="bulkBalance" type="number" step="any" class="input" />
        </div>
        <button type="button" class="btn btn-secondary" :disabled="previewing || submitting" @click="applyBulkBalance">
          {{ t('admin.users.batchCreate.applyBalance') }}
        </button>

        <div class="w-full lg:max-w-[160px]">
          <label class="input-label">{{ t('admin.users.batchCreate.bulkConcurrency') }}</label>
          <input v-model.number="bulkConcurrency" type="number" step="1" min="1" class="input" />
        </div>
        <button type="button" class="btn btn-secondary" :disabled="previewing || submitting" @click="applyBulkConcurrency">
          {{ t('admin.users.batchCreate.applyConcurrency') }}
        </button>
      </div>

      <div
        v-if="previewing"
        class="rounded-xl border border-primary-200 bg-primary-50 p-4 text-sm text-primary-700 dark:border-primary-900/40 dark:bg-primary-900/20 dark:text-primary-300"
      >
        {{ t('admin.users.batchCreate.previewing') }}
      </div>

      <div
        v-else-if="rows.length === 0"
        class="rounded-xl border border-gray-200 bg-white p-8 text-center text-sm text-gray-500 dark:border-dark-700 dark:bg-dark-900 dark:text-dark-400"
      >
        {{ t('admin.users.batchCreate.emptyState') }}
      </div>

      <div
        v-else
        class="overflow-hidden rounded-xl border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-900"
      >
        <div class="max-h-[58vh] overflow-auto">
          <table class="min-w-full divide-y divide-gray-200 dark:divide-dark-700">
            <thead class="bg-gray-50 dark:bg-dark-800">
              <tr>
                <th class="px-3 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">#</th>
                <th class="px-3 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">{{ t('admin.users.batchCreate.sourceName') }}</th>
                <th class="px-3 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">{{ t('admin.users.username') }}</th>
                <th class="px-3 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">{{ t('admin.users.email') }}</th>
                <th class="px-3 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">{{ t('admin.users.password') }}</th>
                <th class="px-3 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">{{ t('admin.users.columns.balance') }}</th>
                <th class="px-3 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">{{ t('admin.users.columns.concurrency') }}</th>
                <th class="px-3 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">{{ t('admin.users.notes') }}</th>
                <th class="px-3 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">{{ t('common.status') }}</th>
                <th class="px-3 py-3 text-right text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">{{ t('common.actions') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-200 dark:divide-dark-700">
              <tr v-for="row in rows" :key="row.row_no" :class="rowErrorMap[row.row_no]?.length ? 'bg-red-50/60 dark:bg-red-900/10' : ''">
                <td class="px-3 py-3 text-sm text-gray-500 dark:text-dark-400">{{ row.row_no }}</td>
                <td class="px-3 py-3 text-sm font-medium text-gray-900 dark:text-white">{{ row.source_name }}</td>
                <td class="px-3 py-3 align-top">
                  <input
                    :value="row.username"
                    type="text"
                    class="input min-w-[140px]"
                    @input="setTextField(row, 'username', ($event.target as HTMLInputElement).value)"
                  />
                </td>
                <td class="px-3 py-3 align-top">
                  <input
                    :value="row.email"
                    type="email"
                    class="input min-w-[220px]"
                    @input="setTextField(row, 'email', ($event.target as HTMLInputElement).value)"
                  />
                </td>
                <td class="px-3 py-3 align-top">
                  <div class="flex min-w-[180px] gap-2">
                    <input
                      :value="row.password"
                      type="text"
                      class="input flex-1"
                      @input="setTextField(row, 'password', ($event.target as HTMLInputElement).value)"
                    />
                    <button type="button" class="btn btn-secondary px-3" :disabled="previewing || submitting" @click="regeneratePassword(row)">
                      <Icon name="refresh" size="sm" />
                    </button>
                  </div>
                </td>
                <td class="px-3 py-3 align-top">
                  <input
                    :value="row.balance"
                    type="number"
                    step="any"
                    class="input min-w-[120px]"
                    @input="setNumberField(row, 'balance', Number(($event.target as HTMLInputElement).value))"
                  />
                </td>
                <td class="px-3 py-3 align-top">
                  <input
                    :value="row.concurrency"
                    type="number"
                    step="1"
                    min="1"
                    class="input min-w-[100px]"
                    @input="setNumberField(row, 'concurrency', Number(($event.target as HTMLInputElement).value))"
                  />
                </td>
                <td class="px-3 py-3 align-top">
                  <input
                    :value="row.notes"
                    type="text"
                    class="input min-w-[160px]"
                    @input="setTextField(row, 'notes', ($event.target as HTMLInputElement).value)"
                  />
                </td>
                <td class="px-3 py-3 align-top">
                  <div v-if="rowErrorMap[row.row_no]?.length" class="space-y-1">
                    <div
                      v-for="(err, index) in rowErrorMap[row.row_no]"
                      :key="`${row.row_no}-${err.field}-${err.code}-${index}`"
                      class="text-xs text-red-600 dark:text-red-400"
                    >
                      {{ err.message }}
                    </div>
                  </div>
                  <span v-else class="inline-flex rounded-full bg-emerald-50 px-2 py-1 text-xs font-medium text-emerald-600 dark:bg-emerald-900/20 dark:text-emerald-400">
                    {{ t('common.valid') }}
                  </span>
                </td>
                <td class="px-3 py-3 text-right align-top">
                  <button type="button" class="btn btn-secondary px-3" :disabled="previewing || submitting" @click="removeRow(row.row_no)">
                    {{ t('common.delete') }}
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <template #footer>
      <div class="flex justify-end gap-3">
        <template v-if="isCompleted">
          <button type="button" class="btn btn-secondary" @click="handleClose">
            {{ t('common.close') }}
          </button>
          <button type="button" class="btn btn-primary" @click="handleDownloadCredentials">
            <Icon name="download" size="sm" class="mr-2" />
            {{ t('admin.users.batchCreate.downloadCredentials') }}
          </button>
        </template>
        <template v-else>
          <button type="button" class="btn btn-secondary" :disabled="previewing || submitting" @click="handleClose">
            {{ t('common.cancel') }}
          </button>
          <button type="button" class="btn btn-primary" :disabled="!canSubmit" @click="handleSubmit">
            {{ submitting ? t('admin.users.batchCreate.submitting') : t('common.confirm') }}
          </button>
        </template>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import { adminAPI } from '@/api/admin'
import type { BatchCreateUsersResponse, BatchUserFieldError, BatchUserPreviewItem, BatchUserRowError } from '@/types'
import { useAppStore } from '@/stores/app'

interface Props {
  show: boolean
}

interface Emits {
  (e: 'close'): void
  (e: 'success'): void
}

type EditableBatchRow = BatchUserPreviewItem & {
  serverErrors: BatchUserFieldError[]
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const { t } = useI18n()
const appStore = useAppStore()

const previewing = ref(false)
const submitting = ref(false)
const file = ref<File | null>(null)
const rows = ref<EditableBatchRow[]>([])
const createdUsers = ref<NonNullable<BatchCreateUsersResponse['users']>>([])
const completedCount = ref(0)
const bulkBalance = ref<number>(9999)
const bulkConcurrency = ref<number>(3)
const fileInput = ref<HTMLInputElement | null>(null)

const fileName = computed(() => file.value?.name || '')
const isCompleted = computed(() => completedCount.value > 0)

watch(
  () => props.show,
  (open) => {
    if (open) {
      resetState()
    }
  }
)

const rowErrorMap = computed<Record<number, BatchUserFieldError[]>>(() => {
  const errorsByRow: Record<number, BatchUserFieldError[]> = {}
  const emailRows = new Map<string, number[]>()
  const usernameRows = new Map<string, number[]>()

  for (const row of rows.value) {
    const rowErrors: BatchUserFieldError[] = []
    const username = row.username.trim()
    const email = row.email.trim().toLowerCase()

    if (!username) {
      rowErrors.push(fieldError('username', 'USERNAME_REQUIRED', t('admin.users.batchCreate.validation.usernameRequired')))
    } else if (username.length > 100) {
      rowErrors.push(fieldError('username', 'USERNAME_TOO_LONG', t('admin.users.batchCreate.validation.usernameTooLong')))
    }

    if (!email) {
      rowErrors.push(fieldError('email', 'EMAIL_REQUIRED', t('admin.users.batchCreate.validation.emailRequired')))
    } else if (!isValidEmail(email)) {
      rowErrors.push(fieldError('email', 'INVALID_EMAIL', t('admin.users.batchCreate.validation.emailInvalid')))
    }

    if (row.password.length < 6) {
      rowErrors.push(fieldError('password', 'PASSWORD_TOO_SHORT', t('admin.users.batchCreate.validation.passwordTooShort')))
    }

    if (!Number.isFinite(row.balance) || row.balance < 0) {
      rowErrors.push(fieldError('balance', 'INVALID_BALANCE', t('admin.users.batchCreate.validation.balanceInvalid')))
    }

    if (!Number.isInteger(row.concurrency) || row.concurrency < 1) {
      rowErrors.push(fieldError('concurrency', 'INVALID_CONCURRENCY', t('admin.users.batchCreate.validation.concurrencyInvalid')))
    }

    if (email) {
      emailRows.set(email, [...(emailRows.get(email) || []), row.row_no])
    }
    if (username) {
      usernameRows.set(username, [...(usernameRows.get(username) || []), row.row_no])
    }

    errorsByRow[row.row_no] = rowErrors
  }

  for (const rowNos of emailRows.values()) {
    if (rowNos.length > 1) {
      for (const rowNo of rowNos) {
        errorsByRow[rowNo] = [
          ...(errorsByRow[rowNo] || []),
          fieldError('email', 'DUPLICATE_EMAIL', t('admin.users.batchCreate.validation.duplicateEmail'))
        ]
      }
    }
  }

  for (const rowNos of usernameRows.values()) {
    if (rowNos.length > 1) {
      for (const rowNo of rowNos) {
        errorsByRow[rowNo] = [
          ...(errorsByRow[rowNo] || []),
          fieldError('username', 'DUPLICATE_USERNAME', t('admin.users.batchCreate.validation.duplicateUsername'))
        ]
      }
    }
  }

  for (const row of rows.value) {
    const merged = [...(errorsByRow[row.row_no] || [])]
    for (const err of row.serverErrors) {
      if (!merged.some(item => item.field === err.field && item.code === err.code && item.message === err.message)) {
        merged.push(err)
      }
    }
    if (merged.length > 0) {
      errorsByRow[row.row_no] = merged
    } else {
      delete errorsByRow[row.row_no]
    }
  }

  return errorsByRow
})

const errorRowCount = computed(() => Object.keys(rowErrorMap.value).length)
const validRowCount = computed(() => rows.value.length - errorRowCount.value)
const canSubmit = computed(() => rows.value.length > 0 && !previewing.value && !submitting.value && errorRowCount.value === 0)

function resetState() {
  previewing.value = false
  submitting.value = false
  file.value = null
  rows.value = []
  createdUsers.value = []
  completedCount.value = 0
  bulkBalance.value = 9999
  bulkConcurrency.value = 3
  if (fileInput.value) {
    fileInput.value.value = ''
  }
}

function handleClose() {
  if (previewing.value || submitting.value) return
  emit('close')
}

function openFilePicker() {
  fileInput.value?.click()
}

async function handleFileChange(event: Event) {
  const target = event.target as HTMLInputElement
  const selectedFile = target.files?.[0] || null
  file.value = selectedFile
  if (selectedFile) {
    await previewFile(selectedFile)
  }
}

async function previewFile(selectedFile: File) {
  previewing.value = true
  rows.value = []
  try {
    const text = await readFileAsText(selectedFile)
    const names = text
      .replace(/^\uFEFF/, '')
      .split(/\r?\n/)
      .map(line => line.trim())
      .filter(Boolean)

    if (names.length === 0) {
      appStore.showError(t('admin.users.batchCreate.emptyFile'))
      return
    }

    const result = await adminAPI.users.previewBatch(names)
    rows.value = result.items.map(item => ({
      ...item,
      notes: item.notes || '',
      serverErrors: [...(item.errors || [])]
    }))
  } catch (error: any) {
    rows.value = []
    appStore.showError(error?.message || t('admin.users.batchCreate.previewFailed'))
  } finally {
    previewing.value = false
  }
}

function setTextField(row: EditableBatchRow, field: 'username' | 'email' | 'password' | 'notes', value: string) {
  row[field] = value
  if (field !== 'notes') {
    clearRowServerErrors(row, field)
  }
}

function setNumberField(row: EditableBatchRow, field: 'balance' | 'concurrency', value: number) {
  row[field] = Number.isNaN(value) ? 0 : value
  clearRowServerErrors(row, field)
}

function clearRowServerErrors(row: EditableBatchRow, field?: string) {
  if (!field) {
    row.serverErrors = []
    return
  }
  row.serverErrors = row.serverErrors.filter(err => err.field !== field)
}

function removeRow(rowNo: number) {
  rows.value = rows.value.filter(row => row.row_no !== rowNo)
}

function applyBulkBalance() {
  for (const row of rows.value) {
    setNumberField(row, 'balance', bulkBalance.value)
  }
}

function applyBulkConcurrency() {
  for (const row of rows.value) {
    setNumberField(row, 'concurrency', bulkConcurrency.value)
  }
}

function regeneratePassword(row: EditableBatchRow) {
  row.password = generateRandomPassword()
  clearRowServerErrors(row, 'password')
}

function regenerateAllPasswords() {
  for (const row of rows.value) {
    regeneratePassword(row)
  }
}

async function handleSubmit() {
  if (!canSubmit.value) {
    appStore.showError(t('admin.users.batchCreate.fixErrors'))
    return
  }

  submitting.value = true
  try {
    const result = await adminAPI.users.createBatch(rows.value.map(row => ({
      row_no: row.row_no,
      source_name: row.source_name,
      email: row.email.trim(),
      password: row.password,
      username: row.username.trim(),
      notes: row.notes.trim(),
      balance: row.balance,
      concurrency: row.concurrency
    })))

    if (result.errors?.length) {
      applyServerErrors(result.errors)
      appStore.showError(t('admin.users.batchCreate.fixErrors'))
      return
    }

    createdUsers.value = result.users || []
    completedCount.value = result.created_count
    appStore.showSuccess(t('admin.users.batchCreate.success', { count: result.created_count }))
    emit('success')
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.users.batchCreate.failed'))
  } finally {
    submitting.value = false
  }
}

function applyServerErrors(errors: BatchUserRowError[]) {
  const grouped = new Map<number, BatchUserFieldError[]>()
  for (const error of errors) {
    grouped.set(error.row_no, [...(grouped.get(error.row_no) || []), {
      field: error.field,
      code: error.code,
      message: error.message
    }])
  }
  for (const row of rows.value) {
    row.serverErrors = grouped.get(row.row_no) || []
  }
}

function fieldError(field: string, code: string, message: string): BatchUserFieldError {
  return { field, code, message }
}

function isValidEmail(email: string): boolean {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)
}

async function readFileAsText(sourceFile: File): Promise<string> {
  if (typeof sourceFile.text === 'function') {
    return sourceFile.text()
  }
  if (typeof sourceFile.arrayBuffer === 'function') {
    const buffer = await sourceFile.arrayBuffer()
    return new TextDecoder().decode(buffer)
  }
  return await new Promise<string>((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(String(reader.result ?? ''))
    reader.onerror = () => reject(reader.error || new Error('Failed to read file'))
    reader.readAsText(sourceFile)
  })
}

function generateRandomPassword(): string {
  const chars = 'ABCDEFGHJKLMNPQRSTUVWXYZabcdefghjkmnpqrstuvwxyz23456789'
  let result = ''
  for (let i = 0; i < 16; i += 1) {
    result += chars.charAt(Math.floor(Math.random() * chars.length))
  }
  return result
}

function handleDownloadCredentials() {
  downloadCreatedUsersCredentials(createdUsers.value)
}

function downloadCreatedUsersCredentials(createdUsers: NonNullable<BatchCreateUsersResponse['users']>) {
  if (createdUsers.length === 0) {
    return
  }

  const rowMap = new Map(rows.value.map(row => [row.row_no, row]))
  const lines = [
    [t('admin.users.username'), t('admin.users.email'), t('admin.users.password')].join('\t')
  ]

  for (const user of createdUsers) {
    const row = rowMap.get(user.row_no)
    if (!row) {
      continue
    }
    lines.push([
      row.username.trim(),
      row.email.trim(),
      row.password
    ].join('\t'))
  }

  if (lines.length === 1) {
    return
  }

  const blob = new Blob([`\uFEFF${lines.join('\r\n')}`], { type: 'text/plain;charset=utf-8' })
  const url = window.URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = `batch-users-credentials-${formatLocalDate(new Date())}.txt`
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  window.URL.revokeObjectURL(url)
}

function formatLocalDate(date: Date): string {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}
</script>
