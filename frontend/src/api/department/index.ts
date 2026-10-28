import { del, get, post, put } from '@/utils/request'
import type { DepartmentRole, TenantMember } from '@/api/tenant/members'

export interface Department {
  id: string
  tenant_id: number
  name: string
  description: string
  is_default: boolean
  member_count?: number
  knowledge_base_count?: number
  created_at?: string
  updated_at?: string
}

export interface ListDepartmentsResponse {
  success: boolean
  data?: Department[]
  message?: string
}

export interface DepartmentResponse {
  success: boolean
  data?: Department
  message?: string
}

export interface DepartmentMembersResponse {
  success: boolean
  data?: TenantMember[]
  message?: string
}

export interface SimpleResponse {
  success: boolean
  message?: string
}

export async function listDepartments(tenantId: number): Promise<ListDepartmentsResponse> {
  return (await get(`/api/v1/tenants/${tenantId}/departments`)) as unknown as ListDepartmentsResponse
}

export async function createDepartment(
  tenantId: number,
  body: { name: string; description?: string },
): Promise<DepartmentResponse> {
  return (await post(`/api/v1/tenants/${tenantId}/departments`, body)) as unknown as DepartmentResponse
}

export async function updateDepartment(
  tenantId: number,
  departmentId: string,
  body: { name: string; description?: string },
): Promise<DepartmentResponse> {
  return (await put(
    `/api/v1/tenants/${tenantId}/departments/${departmentId}`,
    body,
  )) as unknown as DepartmentResponse
}

export async function deleteDepartment(
  tenantId: number,
  departmentId: string,
): Promise<SimpleResponse> {
  return (await del(
    `/api/v1/tenants/${tenantId}/departments/${departmentId}`,
  )) as unknown as SimpleResponse
}

export async function listDepartmentMembers(
  tenantId: number,
  departmentId: string,
): Promise<DepartmentMembersResponse> {
  return (await get(
    `/api/v1/tenants/${tenantId}/departments/${departmentId}/members`,
  )) as unknown as DepartmentMembersResponse
}

export async function assignDepartmentMember(
  tenantId: number,
  departmentId: string,
  userId: string,
  departmentRole: DepartmentRole,
): Promise<SimpleResponse> {
  return (await put(
    `/api/v1/tenants/${tenantId}/departments/${departmentId}/members/${userId}`,
    { department_role: departmentRole },
  )) as unknown as SimpleResponse
}

export async function removeDepartmentMember(
  tenantId: number,
  departmentId: string,
  userId: string,
): Promise<SimpleResponse> {
  return (await del(
    `/api/v1/tenants/${tenantId}/departments/${departmentId}/members/${userId}`,
  )) as unknown as SimpleResponse
}

export interface DepartmentCRAuditorsResponse {
  success: boolean
  data?: { user_ids: string[] }
  message?: string
}

export async function listDepartmentCRAuditors(
  tenantId: number,
  departmentId: string,
): Promise<DepartmentCRAuditorsResponse> {
  return (await get(
    `/api/v1/tenants/${tenantId}/departments/${departmentId}/cr-auditors`,
  )) as unknown as DepartmentCRAuditorsResponse
}

export async function addDepartmentCRAuditor(
  tenantId: number,
  departmentId: string,
  userId: string,
): Promise<SimpleResponse> {
  return (await post(
    `/api/v1/tenants/${tenantId}/departments/${departmentId}/cr-auditors`,
    { user_id: userId },
  )) as unknown as SimpleResponse
}

export async function removeDepartmentCRAuditor(
  tenantId: number,
  departmentId: string,
  userId: string,
): Promise<SimpleResponse> {
  return (await del(
    `/api/v1/tenants/${tenantId}/departments/${departmentId}/cr-auditors/${userId}`,
  )) as unknown as SimpleResponse
}
