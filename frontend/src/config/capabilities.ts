import contract from '../../../app/contracts/capabilities.json'

export type CapabilityKey = string

export const CAPABILITY_KEYS: readonly CapabilityKey[] = contract.keys

export const CAPABILITY_ROLES: Record<string, readonly CapabilityKey[]> = contract.roles

export const CAPABILITY_LABELS: Record<CapabilityKey, string> = {
  'kb:read': '阅读知识库',
  'kb:create': '创建知识库',
  'kb:content_edit': '编辑知识库内容',
  'kb:settings_manage': '管理知识库设置',
  'kb:share_manage': '管理知识库共享',
  'kb:acl_manage': '管理知识库内权限',
  'folder:manage': '管理文件夹',
  'shared_space:view': '查看共享空间',
  'member:view': '查看成员',
  'member:manage': '管理成员',
  'department:manage': '管理部门',
  'settings:self_manage': '管理个人设置',
  'settings:models_manage': '管理模型',
  'settings:infrastructure_manage': '管理基础设施',
  'settings:integrations_manage': '管理集成',
  'settings:skills_manage': '管理技能',
  'settings:workspace_manage': '管理空间运行设置',
  'settings:system_manage': '系统管理',
  'skills:view': '查看技能',
  'cr:audit_manage': '管理 CR 审计员',
  'stats:dept_read': '查看部门统计',
  'stats:company_read': '查看公司统计',
  'audit:dept_read': '查看部门审计',
  'audit:company_read': '查看公司审计',
}

const KEY_SET = new Set<string>(CAPABILITY_KEYS)

export function isKnownCapability(key: string): key is CapabilityKey {
  return KEY_SET.has(key)
}

function normalizeCompanyRole(role: string): string {
  if (role === 'company_admin' || role === 'owner' || role === 'admin') return 'company_admin'
  if (role === 'member' || role === 'contributor' || role === 'viewer') return 'member'
  return ''
}

/** Mirrors the server-side union of company-role and department-role caps. */
export function capsForRoles(
  companyRole: string,
  departmentRole: string,
  systemAdmin = false,
): CapabilityKey[] {
  const roles: string[] = []
  const company = normalizeCompanyRole(companyRole)
  if (company) roles.push(company)
  if (company !== 'company_admin' && ['manager', 'knowledge_admin', 'editor', 'viewer'].includes(departmentRole)) {
    roles.push(departmentRole)
  }
  if (systemAdmin) roles.push('system_admin')
  const granted = new Set<string>()
  for (const role of roles) {
    for (const cap of CAPABILITY_ROLES[role] ?? []) granted.add(cap)
  }
  return CAPABILITY_KEYS.filter((key) => granted.has(key))
}

export function hasCapability(caps: readonly string[], key: CapabilityKey): boolean {
  return caps.includes(key)
}

export function hasCapabilities(caps: readonly string[], required: readonly CapabilityKey[]): boolean {
  return required.every((key) => caps.includes(key))
}
