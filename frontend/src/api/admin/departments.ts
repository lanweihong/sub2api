/**
 * Admin Departments API endpoints
 * Handles department management for administrators
 */

import { apiClient } from '../client'
import type {
  Department,
  CreateDepartmentRequest,
  UpdateDepartmentRequest
} from '@/types'

/**
 * List all departments
 * @returns List of all departments
 */
export async function list(): Promise<Department[]> {
  const { data } = await apiClient.get<Department[]>('/admin/departments')
  return data
}

/**
 * Get a single department by ID
 * @param id - Department ID
 * @returns Department details
 */
export async function get(id: number): Promise<Department> {
  const { data } = await apiClient.get<Department>(`/admin/departments/${id}`)
  return data
}

/**
 * Create a new department
 * @param request - Department creation data
 * @returns Created department
 */
export async function create(request: CreateDepartmentRequest): Promise<Department> {
  const { data } = await apiClient.post<Department>('/admin/departments', request)
  return data
}

/**
 * Update an existing department
 * @param id - Department ID
 * @param request - Department update data
 * @returns Updated department
 */
export async function update(id: number, request: UpdateDepartmentRequest): Promise<Department> {
  const { data } = await apiClient.put<Department>(`/admin/departments/${id}`, request)
  return data
}

/**
 * Delete a department
 * @param id - Department ID
 * @param force - Force delete even if department has children
 */
export async function remove(id: number, force: boolean = false): Promise<void> {
  await apiClient.delete(`/admin/departments/${id}`, {
    params: force ? { force: 'true' } : undefined
  })
}

const departmentsAPI = {
  list,
  get,
  create,
  update,
  remove
}

export default departmentsAPI
