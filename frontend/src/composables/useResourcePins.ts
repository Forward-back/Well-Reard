// Per-user recents for KB and agent list views. Favorites were removed with
// the backend subsystem; recents stay local because they are a navigation aid.

import { ref, computed, watch, type ComputedRef } from 'vue'
import { safeGetItem, safeSetItem, readUserId } from './preferenceStorage'
import { useAuthStore } from '@/stores/auth'

export type ResourceType = 'kb' | 'agent'

export interface PinEntry {
  type: ResourceType
  id: string
  /** Unix ms when the entry was last accessed. */
  ts: number
}

const RECENTS_SUFFIX = 'resource_recents'
const RECENTS_CAP = 30

function tenantSegmentForKey(): string {
  try {
    const authStore = useAuthStore()
    const tid = authStore.effectiveTenantId
    return tid ? `t${tid}_` : ''
  } catch {
    return ''
  }
}

function recentsKey(): string {
  return `Gaze_${readUserId()}_${tenantSegmentForKey()}${RECENTS_SUFFIX}`
}

function readRecents(): PinEntry[] {
  const raw = safeGetItem(recentsKey())
  if (!raw) return []
  try {
    const parsed = JSON.parse(raw)
    if (!Array.isArray(parsed)) return []
    return parsed.filter(
      (e: unknown): e is PinEntry =>
        !!e &&
        typeof (e as PinEntry).type === 'string' &&
        ((e as PinEntry).type === 'kb' || (e as PinEntry).type === 'agent') &&
        typeof (e as PinEntry).id === 'string' &&
        typeof (e as PinEntry).ts === 'number'
    )
  } catch {
    return []
  }
}

function writeRecents(list: PinEntry[]): void {
  safeSetItem(recentsKey(), JSON.stringify(list))
}

const recentsRevision = ref(0)
function bumpRecents() {
  recentsRevision.value++
}

if (typeof window !== 'undefined') {
  window.addEventListener('storage', (e) => {
    if (e.key?.endsWith(`_${RECENTS_SUFFIX}`)) bumpRecents()
  })
}

let tenantWatcherInstalled = false
function installTenantWatcher(): void {
  if (tenantWatcherInstalled) return
  tenantWatcherInstalled = true
  const authStore = useAuthStore()
  watch(
    () => authStore.effectiveTenantId,
    () => bumpRecents()
  )
}

export interface UseResourcePinsResult {
  recents: ComputedRef<PinEntry[]>
  touchRecent: (type: ResourceType, id: string) => void
  removeRecent: (type: ResourceType, id: string) => void
}

export function useResourcePins(): UseResourcePinsResult {
  installTenantWatcher()

  const recents = computed<PinEntry[]>(() => {
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    const _ = recentsRevision.value
    return readRecents().sort((a, b) => b.ts - a.ts)
  })

  const touchRecent = (type: ResourceType, id: string): void => {
    const list = readRecents()
    const idx = list.findIndex((e) => e.type === type && e.id === id)
    if (idx >= 0) list.splice(idx, 1)
    list.unshift({ type, id, ts: Date.now() })
    if (list.length > RECENTS_CAP) list.length = RECENTS_CAP
    writeRecents(list)
    bumpRecents()
  }

  const removeRecent = (type: ResourceType, id: string): void => {
    const list = readRecents()
    const next = list.filter((e) => !(e.type === type && e.id === id))
    if (next.length !== list.length) {
      writeRecents(next)
      bumpRecents()
    }
  }

  return { recents, touchRecent, removeRecent }
}
