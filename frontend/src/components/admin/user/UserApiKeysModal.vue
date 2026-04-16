<template>
  <BaseDialog :show="show" :title="t('admin.users.userApiKeys')" width="wide" @close="handleClose">
    <div v-if="user" class="space-y-4">
      <div class="flex items-center gap-3 rounded-xl bg-gray-50 p-4 dark:bg-dark-700">
        <div class="flex h-10 w-10 items-center justify-center rounded-full bg-primary-100 dark:bg-primary-900/30">
          <span class="text-lg font-medium text-primary-700 dark:text-primary-300">{{ user.email.charAt(0).toUpperCase() }}</span>
        </div>
        <div>
          <p class="font-medium text-gray-900 dark:text-white">{{ user.email }}</p>
          <p class="text-sm text-gray-500 dark:text-dark-400">{{ user.username }}</p>
        </div>
      </div>

      <div v-if="loading" class="flex justify-center py-8">
        <svg class="h-8 w-8 animate-spin text-primary-500" fill="none" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
        </svg>
      </div>

      <div v-else-if="apiKeys.length === 0" class="py-8 text-center">
        <p class="text-sm text-gray-500">{{ t('admin.users.noApiKeys') }}</p>
      </div>

      <div v-else class="max-h-96 space-y-3 overflow-y-auto">
        <div v-for="key in apiKeys" :key="key.id" class="rounded-xl border border-gray-200 bg-white p-4 dark:border-dark-600 dark:bg-dark-800">
          <div class="flex items-start justify-between gap-4">
            <div class="min-w-0 flex-1">
              <div class="mb-1 flex items-center gap-2">
                <span class="font-medium text-gray-900 dark:text-white">{{ key.name }}</span>
                <span :class="['badge text-xs', key.status === 'active' ? 'badge-success' : 'badge-danger']">{{ key.status }}</span>
              </div>
              <p class="truncate font-mono text-sm text-gray-500">{{ maskKey(key.key) }}</p>
            </div>

            <button
              type="button"
              class="btn btn-secondary shrink-0 text-sm"
              :disabled="updatingKeyIds.has(key.id)"
              @click="openGroupEditor(key)"
            >
              <svg v-if="updatingKeyIds.has(key.id)" class="mr-2 h-4 w-4 animate-spin" fill="none" viewBox="0 0 24 24">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
              </svg>
              {{ t('common.edit') }}
            </button>
          </div>

          <div class="mt-3 flex flex-wrap gap-4 text-xs text-gray-500">
            <div class="flex items-start gap-1">
              <span class="pt-1">{{ t('admin.users.group') }}:</span>
              <div v-if="key.bound_groups?.length" class="flex flex-wrap gap-1.5">
                <GroupBadge
                  v-for="binding in key.bound_groups"
                  :key="binding.group_id"
                  :name="binding.group?.name || `#${binding.group_id}`"
                  :platform="binding.group?.platform"
                  :subscription-type="binding.group?.subscription_type"
                  :rate-multiplier="binding.group?.rate_multiplier"
                />
              </div>
              <GroupBadge
                v-else-if="key.group_id && key.group"
                :name="key.group.name"
                :platform="key.group.platform"
                :subscription-type="key.group.subscription_type"
                :rate-multiplier="key.group.rate_multiplier"
              />
              <span v-else class="italic text-gray-400">{{ t('admin.users.none') }}</span>
            </div>
            <div class="flex items-center gap-1">
              <span>{{ t('admin.users.columns.created') }}: {{ formatDateTime(key.created_at) }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </BaseDialog>

  <BaseDialog
    :show="showGroupEditor"
    :title="t('admin.users.editApiKeyGroups')"
    width="normal"
    @close="closeGroupEditor"
  >
    <div v-if="editingKey" class="space-y-5">
      <div class="rounded-xl bg-gray-50 p-4 dark:bg-dark-700">
        <p class="font-medium text-gray-900 dark:text-white">{{ editingKey.name }}</p>
        <p class="mt-1 font-mono text-xs text-gray-500">{{ maskKey(editingKey.key) }}</p>
      </div>

      <div v-if="groupsLoading" class="flex justify-center py-8">
        <svg class="h-8 w-8 animate-spin text-primary-500" fill="none" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
        </svg>
      </div>
      <div v-else>
        <label class="input-label">{{ t('admin.users.group') }}</label>
        <MultiGroupSelect v-model="editForm.selectedGroups" :options="groupOptions" />
        <p class="input-hint mt-2">{{ t('admin.users.editApiKeyGroupsHint') }}</p>
      </div>
    </div>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button type="button" class="btn btn-secondary" @click="closeGroupEditor">{{ t('common.cancel') }}</button>
        <button type="button" class="btn btn-primary" :disabled="groupEditorSubmitting || groupsLoading" @click="saveGroupBindings">
          <svg v-if="groupEditorSubmitting" class="mr-2 h-4 w-4 animate-spin" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
          </svg>
          {{ groupEditorSubmitting ? t('common.saving') : t('common.save') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { adminAPI } from '@/api/admin'
import { formatDateTime } from '@/utils/format'
import type { AdminUser, ApiKey, Group, GroupPlatform, SubscriptionType, UpdateApiKeyRequest } from '@/types'
import BaseDialog from '@/components/common/BaseDialog.vue'
import GroupBadge from '@/components/common/GroupBadge.vue'
import MultiGroupSelect from '@/components/common/MultiGroupSelect.vue'

interface GroupOption {
  value: number
  label: string
  description?: string | null
  rate?: number
  subscriptionType?: SubscriptionType
  platform?: GroupPlatform
}

interface SelectedGroup {
  group_id: number
  priority: number
  model_patterns: string
}

const props = defineProps<{ show: boolean; user: AdminUser | null }>()
const emit = defineEmits(['close'])
const { t } = useI18n()
const appStore = useAppStore()

const apiKeys = ref<ApiKey[]>([])
const availableGroups = ref<Group[]>([])
const loading = ref(false)
const groupsLoading = ref(false)
const updatingKeyIds = ref(new Set<number>())
const showGroupEditor = ref(false)
const groupEditorSubmitting = ref(false)
const editingKey = ref<ApiKey | null>(null)
const editForm = ref({
  selectedGroups: [] as SelectedGroup[],
  hadBoundGroups: false
})

watch(() => props.show, (visible) => {
  if (visible && props.user) {
    load()
    loadAvailableGroups()
    return
  }
  closeGroupEditor()
})

const groupOptions = computed<GroupOption[]>(() => {
  const optionMap = new Map<number, GroupOption>()
  for (const group of availableGroups.value) {
    optionMap.set(group.id, mapGroupToOption(group))
  }

  const key = editingKey.value
  if (key?.group_id && key.group && !optionMap.has(key.group_id)) {
    optionMap.set(key.group_id, mapGroupToOption(key.group))
  }
  for (const binding of key?.bound_groups || []) {
    if (!optionMap.has(binding.group_id)) {
      optionMap.set(binding.group_id, {
        value: binding.group_id,
        label: binding.group?.name || `#${binding.group_id}`,
        description: binding.group?.description,
        rate: binding.group?.rate_multiplier,
        subscriptionType: binding.group?.subscription_type,
        platform: binding.group?.platform
      })
    }
  }

  return Array.from(optionMap.values())
})

const load = async () => {
  if (!props.user) return
  loading.value = true
  try {
    const res = await adminAPI.users.getUserApiKeys(props.user.id)
    apiKeys.value = res.items || []
  } catch (error) {
    console.error('Failed to load API keys:', error)
    appStore.showError(t('admin.users.failedToLoadApiKeys'))
  } finally {
    loading.value = false
  }
}

const loadAvailableGroups = async () => {
  if (!props.user) return
  groupsLoading.value = true
  try {
    availableGroups.value = await adminAPI.users.getUserAvailableGroups(props.user.id)
  } catch (error) {
    console.error('Failed to load available groups:', error)
    appStore.showError(t('admin.users.failedToLoadGroups'))
  } finally {
    groupsLoading.value = false
  }
}

const openGroupEditor = (key: ApiKey) => {
  editingKey.value = key
  const hasBoundGroups = (key.bound_groups?.length ?? 0) > 0
  const selectedGroups: SelectedGroup[] = hasBoundGroups
    ? key.bound_groups!.map((binding) => ({
        group_id: binding.group_id,
        priority: binding.priority,
        model_patterns: (binding.model_patterns || []).join(', ')
      }))
    : (key.group_id !== null && key.group_id !== undefined
        ? [{ group_id: key.group_id, priority: 0, model_patterns: '' }]
        : [])

  editForm.value = {
    selectedGroups,
    hadBoundGroups: hasBoundGroups
  }
  showGroupEditor.value = true
}

const closeGroupEditor = () => {
  showGroupEditor.value = false
  groupEditorSubmitting.value = false
  editingKey.value = null
  editForm.value = {
    selectedGroups: [],
    hadBoundGroups: false
  }
}

const saveGroupBindings = async () => {
  if (!editingKey.value) return

  const groups = editForm.value.selectedGroups
  if (groups.length === 0) {
    appStore.showError(t('keys.groupRequired'))
    return
  }

  const useMultiGroupMode = groups.length > 1 || (groups.length === 1 && editForm.value.hadBoundGroups)
  const payload: UpdateApiKeyRequest = {}

  if (useMultiGroupMode) {
    payload.clear_group_id = true
    payload.bound_groups = groups.map((group, idx) => ({
      group_id: group.group_id,
      priority: group.priority ?? idx,
      model_patterns: group.model_patterns
        ? group.model_patterns.split(',').map((item) => item.trim()).filter(Boolean)
        : undefined
    }))
  } else {
    payload.group_id = groups[0].group_id
    if (editingKey.value.bound_groups?.length) {
      payload.bound_groups = []
    }
  }

  const keyID = editingKey.value.id
  updatingKeyIds.value.add(keyID)
  groupEditorSubmitting.value = true
  try {
    const result = await adminAPI.apiKeys.updateApiKey(keyID, payload)
    replaceKey(result.api_key)
    await load()
    if (result.auto_granted_group_access) {
      const grantedName = result.granted_group_names?.join(', ') || result.granted_group_name || ''
      appStore.showSuccess(t('admin.users.groupChangedWithGrant', { group: grantedName }))
    } else {
      appStore.showSuccess(t('admin.users.groupChangedSuccess'))
    }
    closeGroupEditor()
  } catch (error: any) {
    console.error('Failed to update API key groups:', error)
    appStore.showError(error?.response?.data?.detail || t('admin.users.groupChangeFailed'))
  } finally {
    updatingKeyIds.value.delete(keyID)
    groupEditorSubmitting.value = false
  }
}

const replaceKey = (updatedKey: ApiKey) => {
  const index = apiKeys.value.findIndex((key) => key.id === updatedKey.id)
  if (index !== -1) {
    apiKeys.value[index] = updatedKey
  }
}

const mapGroupToOption = (group: Group): GroupOption => ({
  value: group.id,
  label: group.name,
  description: group.description,
  rate: group.rate_multiplier,
  subscriptionType: group.subscription_type,
  platform: group.platform
})

const maskKey = (key: string) => `${key.substring(0, 20)}...${key.substring(key.length - 8)}`

const handleClose = () => {
  closeGroupEditor()
  emit('close')
}
</script>
