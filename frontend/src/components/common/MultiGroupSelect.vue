<template>
  <div class="multi-group-select">
    <!-- Tag input area -->
    <div
      ref="containerRef"
      :class="[
        'mgs-trigger',
        isOpen && 'mgs-trigger-open',
        error && 'mgs-trigger-error'
      ]"
      @click="openDropdown"
    >
      <div class="mgs-tags-area">
        <!-- Selected group tags -->
        <span
          v-for="group in selectedGroups"
          :key="group.group_id"
          class="mgs-tag"
        >
          <GroupBadge
            :name="getGroupLabel(group.group_id)"
            :platform="getGroupPlatform(group.group_id)"
            :subscription-type="getGroupSubscriptionType(group.group_id)"
            :rate-multiplier="getGroupRate(group.group_id)"
            :user-rate-multiplier="getGroupUserRate(group.group_id)"
          />
          <button
            type="button"
            class="mgs-tag-remove"
            @click.stop="removeGroup(group.group_id)"
            :title="t('keys.multiGroup.removeGroup')"
          >
            <svg class="h-3 w-3" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2.5">
              <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </span>
        <!-- Placeholder when empty -->
        <span v-if="selectedGroups.length === 0" class="mgs-placeholder">
          {{ t('keys.selectGroup') }}
        </span>
      </div>
      <span class="select-icon">
        <Icon
          name="chevronDown"
          size="md"
          :class="['transition-transform duration-200', isOpen && 'rotate-180']"
        />
      </span>
    </div>

    <!-- Dropdown panel -->
    <Teleport to="body">
      <Transition name="select-dropdown">
        <div
          v-if="isOpen"
          ref="dropdownRef"
          class="select-dropdown-portal"
          :class="[instanceId]"
          :style="dropdownStyle"
          @click.stop
          @mousedown.stop
        >
          <!-- Search -->
          <div class="select-search">
            <Icon name="search" size="sm" class="text-gray-400" />
            <input
              ref="searchInputRef"
              v-model="searchQuery"
              type="text"
              :placeholder="t('keys.searchGroup')"
              class="select-search-input"
              @click.stop
            />
          </div>
          <!-- Options -->
          <div class="select-options">
            <div
              v-for="option in filteredOptions"
              :key="option.value"
              @click.stop="toggleOption(option)"
              :class="[
                'select-option',
                isOptionSelected(option) && 'select-option-selected'
              ]"
            >
              <slot name="option" :option="option" :selected="isOptionSelected(option)">
                <GroupOptionItem
                  :name="option.label"
                  :platform="option.platform!"
                  :subscription-type="option.subscriptionType"
                  :rate-multiplier="option.rate"
                  :user-rate-multiplier="option.userRate"
                  :description="option.description"
                  :selected="isOptionSelected(option)"
                />
              </slot>
            </div>
            <div v-if="filteredOptions.length === 0" class="select-empty">
              {{ t('common.noOptionsFound') }}
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>

    <!-- Multi-group hint -->
    <p v-if="selectedGroups.length > 1" class="input-hint mt-1.5">
      {{ t('keys.multiGroup.hint') }}
    </p>

    <!-- Model patterns config for multi-group (shown when ≥ 2 groups selected) -->
    <div v-if="selectedGroups.length > 1" class="mt-3 space-y-2">
      <div
        v-for="(group, idx) in selectedGroups"
        :key="group.group_id"
        :class="[
          'mgs-config-row',
          dragOverIdx === idx && dragFromIdx !== idx && 'mgs-config-row-dragover',
          dragFromIdx === idx && 'mgs-config-row-dragging'
        ]"
        draggable="true"
        @dragstart="onDragStart(idx, $event)"
        @dragover.prevent="onDragOver(idx)"
        @dragleave="onDragLeave(idx)"
        @drop.prevent="onDrop(idx)"
        @dragend="onDragEnd"
      >
        <!-- Drag handle -->
        <span class="mgs-drag-handle" :title="t('keys.multiGroup.priority')">
          <svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="1.5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M3.75 6.75h16.5M3.75 12h16.5M3.75 17.25h16.5" />
          </svg>
        </span>
        <span class="mgs-config-index">#{{ idx + 1 }}</span>
        <GroupBadge
          :name="getGroupLabel(group.group_id)"
          :platform="getGroupPlatform(group.group_id)"
          :subscription-type="getGroupSubscriptionType(group.group_id)"
          :show-rate="false"
          class="shrink-0"
        />
        <input
          v-model="group.model_patterns"
          type="text"
          class="input text-xs py-1 flex-1"
          :placeholder="t('keys.multiGroup.modelPatternsPlaceholder')"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import Icon from '@/components/icons/Icon.vue'
import type { GroupPlatform, SubscriptionType } from '@/types'
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import GroupBadge from './GroupBadge.vue'
import GroupOptionItem from './GroupOptionItem.vue'

const { t } = useI18n()
const instanceId = `mgs-${Math.random().toString(36).substring(2, 9)}`

export interface GroupOption {
  value: number
  label: string
  description?: string | null
  rate?: number
  userRate?: number | null
  subscriptionType?: SubscriptionType
  platform?: GroupPlatform
}

export interface SelectedGroup {
  group_id: number
  priority: number
  model_patterns: string
}

interface Props {
  modelValue: SelectedGroup[]
  options: GroupOption[]
  error?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  error: false
})

const emit = defineEmits<{
  (e: 'update:modelValue', value: SelectedGroup[]): void
}>()

const isOpen = ref(false)
const searchQuery = ref('')
const containerRef = ref<HTMLElement | null>(null)
const dropdownRef = ref<HTMLElement | null>(null)
const searchInputRef = ref<HTMLInputElement | null>(null)
const dropdownPosition = ref<'bottom' | 'top'>('bottom')
const triggerRect = ref<DOMRect | null>(null)

const selectedGroups = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val)
})

const selectedIds = computed(() => new Set(props.modelValue.map(g => g.group_id)))

const filteredOptions = computed(() => {
  if (!searchQuery.value) return props.options
  const query = searchQuery.value.toLowerCase()
  return props.options.filter(opt =>
    opt.label.toLowerCase().includes(query) ||
    (opt.description && opt.description.toLowerCase().includes(query))
  )
})

const isOptionSelected = (option: GroupOption): boolean => {
  return selectedIds.value.has(option.value)
}

const toggleOption = (option: GroupOption) => {
  const current = [...props.modelValue]
  const idx = current.findIndex(g => g.group_id === option.value)
  if (idx >= 0) {
    // Deselect
    current.splice(idx, 1)
    // Re-index priorities
    current.forEach((g, i) => { g.priority = i })
  } else {
    // Select
    current.push({
      group_id: option.value,
      priority: current.length,
      model_patterns: ''
    })
  }
  emit('update:modelValue', current)
}

const removeGroup = (groupId: number) => {
  const current = props.modelValue.filter(g => g.group_id !== groupId)
  current.forEach((g, i) => { g.priority = i })
  emit('update:modelValue', current)
}

// Drag-and-drop reordering
const dragFromIdx = ref<number | null>(null)
const dragOverIdx = ref<number | null>(null)

const onDragStart = (idx: number, e: DragEvent) => {
  dragFromIdx.value = idx
  if (e.dataTransfer) {
    e.dataTransfer.effectAllowed = 'move'
    e.dataTransfer.setData('text/plain', String(idx))
  }
}

const onDragOver = (idx: number) => {
  dragOverIdx.value = idx
}

const onDragLeave = (idx: number) => {
  if (dragOverIdx.value === idx) {
    dragOverIdx.value = null
  }
}

const onDrop = (targetIdx: number) => {
  const fromIdx = dragFromIdx.value
  if (fromIdx === null || fromIdx === targetIdx) {
    dragFromIdx.value = null
    dragOverIdx.value = null
    return
  }
  const current = [...props.modelValue]
  const [moved] = current.splice(fromIdx, 1)
  current.splice(targetIdx, 0, moved)
  current.forEach((g, i) => { g.priority = i })
  emit('update:modelValue', current)
  dragFromIdx.value = null
  dragOverIdx.value = null
}

const onDragEnd = () => {
  dragFromIdx.value = null
  dragOverIdx.value = null
}

// Helper methods to look up group info from options
const findGroupOption = (groupId: number | null) =>
  props.options.find(o => o.value === groupId)

const getGroupLabel = (groupId: number | null) =>
  findGroupOption(groupId)?.label ?? ''

const getGroupPlatform = (groupId: number | null) =>
  findGroupOption(groupId)?.platform

const getGroupSubscriptionType = (groupId: number | null) =>
  findGroupOption(groupId)?.subscriptionType

const getGroupRate = (groupId: number | null) =>
  findGroupOption(groupId)?.rate

const getGroupUserRate = (groupId: number | null) =>
  findGroupOption(groupId)?.userRate

// Dropdown positioning (reused pattern from Select.vue)
const dropdownStyle = computed(() => {
  if (!triggerRect.value) return {}
  const rect = triggerRect.value
  const style: Record<string, string> = {
    position: 'fixed',
    left: `${rect.left}px`,
    minWidth: `${rect.width}px`,
    zIndex: '100000020'
  }
  if (dropdownPosition.value === 'top') {
    style.bottom = `${window.innerHeight - rect.top + 4}px`
  } else {
    style.top = `${rect.bottom + 4}px`
  }
  return style
})

const updateTriggerRect = () => {
  if (containerRef.value) {
    triggerRect.value = containerRef.value.getBoundingClientRect()
  }
}

const calculateDropdownPosition = () => {
  if (!containerRef.value) return
  updateTriggerRect()
  nextTick(() => {
    if (!dropdownRef.value || !triggerRect.value) return
    const dropdownHeight = dropdownRef.value.offsetHeight || 240
    const spaceBelow = window.innerHeight - triggerRect.value.bottom
    const spaceAbove = triggerRect.value.top
    if (spaceBelow < dropdownHeight && spaceAbove > dropdownHeight) {
      dropdownPosition.value = 'top'
    } else {
      dropdownPosition.value = 'bottom'
    }
  })
}

const openDropdown = () => {
  isOpen.value = !isOpen.value
}

watch(isOpen, (open) => {
  if (open) {
    calculateDropdownPosition()
    nextTick(() => searchInputRef.value?.focus())
    window.addEventListener('scroll', updateTriggerRect, { capture: true, passive: true })
    window.addEventListener('resize', calculateDropdownPosition)
  } else {
    searchQuery.value = ''
    window.removeEventListener('scroll', updateTriggerRect, { capture: true })
    window.removeEventListener('resize', calculateDropdownPosition)
  }
})

const handleClickOutside = (event: MouseEvent) => {
  const target = event.target as HTMLElement
  const isInDropdown = !!target.closest(`.${instanceId}`)
  const isInTrigger = containerRef.value?.contains(target)
  if (!isInDropdown && !isInTrigger && isOpen.value) {
    isOpen.value = false
  }
}

onMounted(() => {
  document.addEventListener('click', handleClickOutside)
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
  window.removeEventListener('scroll', updateTriggerRect, { capture: true })
  window.removeEventListener('resize', calculateDropdownPosition)
})
</script>

<style scoped>
.mgs-trigger {
  @apply flex w-full items-center justify-between gap-2;
  @apply rounded-xl px-3 py-2 text-sm;
  @apply bg-white dark:bg-dark-800;
  @apply border border-gray-200 dark:border-dark-600;
  @apply text-gray-900 dark:text-gray-100;
  @apply transition-all duration-200;
  @apply focus-within:border-primary-500 focus-within:outline-none focus-within:ring-2 focus-within:ring-primary-500/30;
  @apply hover:border-gray-300 dark:hover:border-dark-500;
  @apply cursor-pointer;
  min-height: 42px;
}

.mgs-trigger-open {
  @apply border-primary-500 ring-2 ring-primary-500/30;
}

.mgs-trigger-error {
  @apply border-red-500 focus-within:border-red-500 focus-within:ring-red-500/30;
}

.mgs-tags-area {
  @apply flex flex-1 flex-wrap items-center gap-1.5;
  min-height: 24px;
}

.mgs-tag {
  @apply inline-flex items-center gap-0.5 rounded-md;
  @apply transition-all duration-150;
}

.mgs-tag-remove {
  @apply rounded p-0.5 text-gray-400;
  @apply hover:bg-red-50 hover:text-red-500;
  @apply dark:hover:bg-red-900/20;
  @apply transition-colors duration-150;
}

.mgs-placeholder {
  @apply text-gray-400 dark:text-dark-400;
}

.mgs-config-row {
  @apply flex items-center gap-2 rounded-lg border border-gray-100 px-3 py-2;
  @apply dark:border-dark-600;
  @apply transition-all duration-150;
}

.mgs-config-row-dragging {
  @apply opacity-40;
}

.mgs-config-row-dragover {
  @apply border-primary-400 bg-primary-50/50;
  @apply dark:border-primary-500 dark:bg-primary-900/20;
}

.mgs-drag-handle {
  @apply cursor-grab text-gray-300 hover:text-gray-500 shrink-0;
  @apply dark:text-gray-600 dark:hover:text-gray-400;
  @apply transition-colors duration-150;
}

.mgs-config-row-dragging .mgs-drag-handle {
  @apply cursor-grabbing;
}

.mgs-config-index {
  @apply text-xs font-medium text-gray-400 dark:text-gray-500 shrink-0;
}
</style>
