/**
 * Admin API Keys API endpoints
 * Handles API key management for administrators
 */

import { apiClient } from '../client'
import type { ApiKey, UpdateApiKeyRequest } from '@/types'

export interface UpdateApiKeyGroupResult {
  api_key: ApiKey
  auto_granted_group_access: boolean
  granted_group_id?: number
  granted_group_name?: string
  granted_group_ids?: number[]
  granted_group_names?: string[]
}

/**
 * Update an API key's bindings
 * @param id - API Key ID
 * @param updates - Group / bound group updates
 * @returns Updated API key with auto-grant info
 */
export async function updateApiKey(id: number, updates: UpdateApiKeyRequest): Promise<UpdateApiKeyGroupResult> {
  const { data } = await apiClient.put<UpdateApiKeyGroupResult>(`/admin/api-keys/${id}`, updates)
  return data
}

/**
 * Update an API key's single-group binding
 * @param id - API Key ID
 * @param groupId - Group ID (0 to unbind, positive to bind, null/undefined to skip)
 * @returns Updated API key with auto-grant info
 */
export async function updateApiKeyGroup(id: number, groupId: number | null): Promise<UpdateApiKeyGroupResult> {
  return updateApiKey(id, {
    group_id: groupId === null ? 0 : groupId
  })
}

export const apiKeysAPI = {
  updateApiKey,
  updateApiKeyGroup
}

export default apiKeysAPI
