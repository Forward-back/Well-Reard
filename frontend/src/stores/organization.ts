import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { SharedKnowledgeBase, SharedAgentInfo } from '@/api/organization'
import { listSharedKnowledgeBases } from '@/api/organization'

// Shared-space state used by KB list/detail/input views. Organization
// management was removed with its backend; this store now keeps only the
// department-shared KB projection plus permission helpers.
export const useOrganizationStore = defineStore('organization', () => {
  const sharedKnowledgeBases = ref<SharedKnowledgeBase[]>([])
  const sharedAgents = ref<SharedAgentInfo[]>([])
  // Kept as an empty list for callers that still render the legacy space
  // switcher; the backend organization routes are gone.
  const organizations = ref<Array<{ id: string; name: string }>>([])
  const totalPendingJoinRequestCount = ref(0)

  let sharedKbLoadedAt = 0
  const SHARED_RESOURCE_TTL_MS = 60_000

  async function fetchSharedKnowledgeBases(options?: { force?: boolean }) {
    const force = options?.force ?? false
    if (!force && sharedKbLoadedAt > 0 && Date.now() - sharedKbLoadedAt < SHARED_RESOURCE_TTL_MS) {
      return sharedKnowledgeBases.value
    }
    const response = await listSharedKnowledgeBases()
    if (response.success && response.data) {
      sharedKnowledgeBases.value = response.data.filter(s => s.knowledge_base != null)
      sharedKbLoadedAt = Date.now()
    }
    return sharedKnowledgeBases.value
  }

  async function fetchSharedAgents(_options?: { force?: boolean }) {
    sharedAgents.value = []
    return sharedAgents.value
  }

  async function fetchOrganizations(_options?: { force?: boolean }) {
    organizations.value = []
    totalPendingJoinRequestCount.value = 0
    return organizations.value
  }

  function getKBPermission(kbId: string): 'owner' | 'admin' | 'editor' | 'viewer' | null {
    const shared = sharedKnowledgeBases.value.find(s => s.knowledge_base?.id === kbId)
    return shared?.permission || null
  }

  function canEditKB(kbId: string, isOwner: boolean): boolean {
    if (isOwner) return true
    const permission = getKBPermission(kbId)
    return permission === 'admin' || permission === 'editor'
  }

  function canManageKB(kbId: string, isOwner: boolean): boolean {
    if (isOwner) return true
    return getKBPermission(kbId) === 'admin'
  }

  function clearState() {
    sharedKnowledgeBases.value = []
    sharedAgents.value = []
    organizations.value = []
    totalPendingJoinRequestCount.value = 0
    sharedKbLoadedAt = 0
  }

  return {
    sharedKnowledgeBases,
    sharedAgents,
    organizations,
    totalPendingJoinRequestCount,
    fetchSharedKnowledgeBases,
    fetchSharedAgents,
    fetchOrganizations,
    getKBPermission,
    canEditKB,
    canManageKB,
    clearState,
  }
})
