<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="flex flex-wrap items-center gap-3">
          <div class="flex flex-1 flex-wrap items-center gap-3">
            <div class="relative w-full md:w-64">
              <Icon name="search" size="md" class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
              <input
                v-model="searchQuery"
                type="text"
                :placeholder="t('admin.departments.searchDepartments')"
                class="input pl-10"
              />
            </div>
          </div>
          <div class="flex items-center gap-2">
            <button @click="loadDepartments" :disabled="loading" class="btn btn-secondary px-2 md:px-3" :title="t('common.refresh')">
              <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
            </button>
            <button @click="openCreateModal" class="btn btn-primary">
              <Icon name="plus" size="sm" class="mr-1.5" />
              {{ t('admin.departments.createDepartment') }}
            </button>
          </div>
        </div>
      </template>

      <template #table>
        <div class="table-wrapper">
          <table v-if="filteredDepartments.length > 0">
            <thead>
              <tr>
                <th>{{ t('admin.departments.columns.name') }}</th>
                <th>{{ t('admin.departments.columns.code') }}</th>
                <th>{{ t('admin.departments.parentDepartment') }}</th>
                <th>{{ t('admin.departments.columns.sortOrder') }}</th>
                <th>{{ t('admin.departments.columns.status') }}</th>
                <th class="text-right">{{ t('admin.departments.columns.actions') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="dept in filteredDepartments" :key="dept.id">
                <td>
                  <div class="flex items-center gap-2">
                    <span class="font-medium">{{ dept.name }}</span>
                    <span v-if="dept.is_default" class="badge badge-primary text-xs">{{ t('admin.departments.defaultBadge') }}</span>
                  </div>
                </td>
                <td><code class="text-xs">{{ dept.code }}</code></td>
                <td>{{ getParentName(dept.parent_id) }}</td>
                <td>{{ dept.sort_order }}</td>
                <td>
                  <span :class="dept.status === 'active' ? 'badge-green' : 'badge-gray'" class="badge">
                    {{ dept.status === 'active' ? t('common.active') : t('admin.users.disabled') }}
                  </span>
                </td>
                <td class="text-right">
                  <div class="flex items-center justify-end gap-2">
                    <button @click="openEditModal(dept)" class="btn btn-secondary btn-sm">
                      <Icon name="edit" size="sm" />
                    </button>
                    <button
                      v-if="!dept.is_default"
                      @click="handleDelete(dept)"
                      class="btn btn-sm text-red-600 hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-900/20"
                    >
                      <Icon name="trash" size="sm" />
                    </button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
          <EmptyState v-else :message="t('admin.departments.noDepartments')" />
        </div>
      </template>
    </TablePageLayout>

    <!-- Create/Edit Modal -->
    <div v-if="showModal" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50" @click.self="closeModal">
      <div class="w-full max-w-lg rounded-2xl bg-white p-6 shadow-xl dark:bg-dark-800">
        <h3 class="mb-4 text-lg font-semibold text-gray-900 dark:text-white">
          {{ editingDept ? t('admin.departments.editDepartment') : t('admin.departments.createDepartment') }}
        </h3>
        <form @submit.prevent="handleSubmit" class="space-y-4">
          <div>
            <label class="label">{{ t('admin.departments.name') }}</label>
            <input v-model="form.name" type="text" class="input" :placeholder="t('admin.departments.enterName')" required />
          </div>
          <div>
            <label class="label">{{ t('admin.departments.enterCodeOptional') }}</label>
            <input v-model="form.code" type="text" class="input" :placeholder="t('admin.departments.enterCode')" maxlength="50" />
          </div>
          <div>
            <label class="label">{{ t('admin.departments.description_field') }}</label>
            <input v-model="form.description" type="text" class="input" :placeholder="t('admin.departments.enterDescription')" />
          </div>
          <div>
            <label class="label">{{ t('admin.departments.parentDepartment') }}</label>
            <select v-model="form.parent_id" class="input">
              <option :value="null">{{ t('admin.departments.noParent') }}</option>
              <option v-for="dept in availableParents" :key="dept.id" :value="dept.id">{{ dept.name }}</option>
            </select>
          </div>
          <div class="grid grid-cols-2 gap-4">
            <div>
              <label class="label">{{ t('admin.departments.sortOrder') }}</label>
              <input v-model.number="form.sort_order" type="number" class="input" />
            </div>
            <div>
              <label class="label">{{ t('admin.departments.status') }}</label>
              <select v-model="form.status" class="input">
                <option value="active">{{ t('common.active') }}</option>
                <option value="disabled">{{ t('admin.users.disabled') }}</option>
              </select>
            </div>
          </div>
          <div class="flex justify-end gap-3 pt-2">
            <button type="button" @click="closeModal" class="btn btn-secondary">{{ t('common.cancel') }}</button>
            <button type="submit" :disabled="submitting" class="btn btn-primary">
              {{ submitting ? (editingDept ? t('admin.departments.updating') : t('admin.departments.creating')) : (editingDept ? t('common.save') : t('common.create')) }}
            </button>
          </div>
        </form>
      </div>
    </div>

    <ConfirmDialog
      :show="showDeleteDialog"
      :title="t('admin.departments.deleteDepartment')"
      :message="t('admin.departments.deleteConfirm', { name: deletingDept?.name })"
      :danger="true"
      @confirm="confirmDelete"
      @cancel="showDeleteDialog = false"
    />

    <ConfirmDialog
      :show="showForceDeleteDialog"
      :title="t('admin.departments.forceDeleteTitle')"
      :message="t('admin.departments.forceDeleteConfirm', { name: deletingDept?.name })"
      :danger="true"
      @confirm="confirmForceDelete"
      @cancel="showForceDeleteDialog = false"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import type { Department } from '@/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores/app'

const { t } = useI18n()
const appStore = useAppStore()

const departments = ref<Department[]>([])
const loading = ref(false)
const searchQuery = ref('')

// Modal state
const showModal = ref(false)
const editingDept = ref<Department | null>(null)
const submitting = ref(false)
const form = ref({
  name: '',
  code: '',
  description: '',
  parent_id: null as number | null,
  sort_order: 0,
  status: 'active' as 'active' | 'disabled'
})

// Delete state
const showDeleteDialog = ref(false)
const showForceDeleteDialog = ref(false)
const deletingDept = ref<Department | null>(null)

const filteredDepartments = computed(() => {
  if (!searchQuery.value) return departments.value
  const q = searchQuery.value.toLowerCase()
  return departments.value.filter(
    d => d.name.toLowerCase().includes(q) || d.code.toLowerCase().includes(q)
  )
})

const availableParents = computed(() => {
  if (!editingDept.value) return departments.value
  return departments.value.filter(d => d.id !== editingDept.value!.id)
})

const getParentName = (parentId: number | null) => {
  if (!parentId) return '-'
  const parent = departments.value.find(d => d.id === parentId)
  return parent?.name || '-'
}

const loadDepartments = async () => {
  loading.value = true
  try {
    departments.value = await adminAPI.departments.list()
  } catch (e) {
    appStore.showError(t('admin.departments.failedToLoad'))
    console.error('Failed to load departments:', e)
  } finally {
    loading.value = false
  }
}

const openCreateModal = () => {
  editingDept.value = null
  form.value = { name: '', code: '', description: '', parent_id: null, sort_order: 0, status: 'active' }
  showModal.value = true
}

const openEditModal = (dept: Department) => {
  editingDept.value = dept
  form.value = {
    name: dept.name,
    code: dept.code,
    description: dept.description,
    parent_id: dept.parent_id,
    sort_order: dept.sort_order,
    status: dept.status
  }
  showModal.value = true
}

const closeModal = () => {
  showModal.value = false
  editingDept.value = null
}

const handleSubmit = async () => {
  submitting.value = true
  form.value.code = form.value.code.trim()
  try {
    if (editingDept.value) {
      await adminAPI.departments.update(editingDept.value.id, {
        name: form.value.name,
        code: form.value.code,
        description: form.value.description,
        parent_id: form.value.parent_id,
        sort_order: form.value.sort_order,
        status: form.value.status
      })
      appStore.showSuccess(t('admin.departments.updatedSuccess'))
    } else {
      await adminAPI.departments.create({
        name: form.value.name,
        code: form.value.code,
        description: form.value.description,
        parent_id: form.value.parent_id,
        sort_order: form.value.sort_order,
        status: form.value.status
      })
      appStore.showSuccess(t('admin.departments.createdSuccess'))
    }
    closeModal()
    await loadDepartments()
  } catch (e: any) {
    const msg = e?.message || (editingDept.value ? t('admin.departments.failedToUpdate') : t('admin.departments.failedToCreate'))
    appStore.showError(msg)
    console.error('Failed to save department:', e)
  } finally {
    submitting.value = false
  }
}

const handleDelete = (dept: Department) => {
  deletingDept.value = dept
  showDeleteDialog.value = true
}

const confirmDelete = async () => {
  if (!deletingDept.value) return
  try {
    await adminAPI.departments.remove(deletingDept.value.id)
    appStore.showSuccess(t('admin.departments.deletedSuccess'))
    deletingDept.value = null
    await loadDepartments()
  } catch (e: any) {
    const reason = e?.reason
    if (reason === 'DEPARTMENT_HAS_USERS') {
      // 有用户 → 弹出 force 二次确认
      showForceDeleteDialog.value = true
    } else if (reason === 'DEPARTMENT_HAS_CHILDREN') {
      appStore.showError(t('admin.departments.deleteChildrenFirst'))
    } else {
      appStore.showError(t('admin.departments.failedToDelete'))
    }
    console.error('Failed to delete department:', e)
  } finally {
    showDeleteDialog.value = false
  }
}

const confirmForceDelete = async () => {
  if (!deletingDept.value) return
  try {
    await adminAPI.departments.remove(deletingDept.value.id, true)
    appStore.showSuccess(t('admin.departments.deletedSuccess'))
    await loadDepartments()
  } catch (e: any) {
    appStore.showError(t('admin.departments.failedToDelete'))
    console.error('Failed to force delete department:', e)
  } finally {
    showForceDeleteDialog.value = false
    deletingDept.value = null
  }
}

onMounted(() => {
  loadDepartments()
})
</script>
