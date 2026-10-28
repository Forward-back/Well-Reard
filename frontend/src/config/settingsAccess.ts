/**
 * Settings section → capability contract. This replaces the old role map:
 * page entry visibility is a projection of the shared capability contract,
 * while backend resource guards remain authoritative.
 */
export const SETTINGS_SECTION_CAPS: Record<string, string[]> = {
  general: ['settings:self_manage'],
  userprofile: ['settings:self_manage'],
  mymemory: ['settings:self_manage'],
  tenant: ['settings:self_manage'],
  system: ['settings:self_manage'],
  members: ['member:manage'],
  departments: ['department:manage'],
  models: ['settings:models_manage'],
  ollama: ['settings:infrastructure_manage'],
  parser: ['settings:infrastructure_manage'],
  storage: ['settings:infrastructure_manage'],
  chathistory: ['settings:workspace_manage'],
  memory: ['settings:workspace_manage'],
  docsync: ['settings:workspace_manage'],
  skills: ['skills:view'],
  integrations: ['settings:integrations_manage'],
}

export const SYSTEM_ADMIN_SETTINGS_SECTIONS = new Set([
  'system-global',
  'runtime-queues',
  'platform-api-keys',
  'system-audit-log',
])
