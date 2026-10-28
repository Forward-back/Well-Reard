import { hasCapabilities } from './capabilities'

export type ManagementTarget =
  | { kind: 'governance' }
  | { kind: 'audit' }
  | { kind: 'auditors' }
  | { kind: 'kbSettings' }
  | { kind: 'settings'; section: string }

export interface ManagementCardDefinition {
  key: string
  icon: string
  titleKey: string
  descriptionKey: string
  requiredCaps?: readonly string[]
  requiredAnyCaps?: readonly string[]
  target: ManagementTarget
}

export const MANAGEMENT_CARDS: readonly ManagementCardDefinition[] = [
  {
    key: 'members',
    icon: 'usergroup',
    titleKey: 'management.cards.members.title',
    descriptionKey: 'management.cards.members.description',
    requiredCaps: ['member:manage'],
    target: { kind: 'settings', section: 'members' },
  },
  {
    key: 'departments',
    icon: 'system-sum',
    titleKey: 'management.cards.departments.title',
    descriptionKey: 'management.cards.departments.description',
    requiredCaps: ['department:manage'],
    target: { kind: 'settings', section: 'departments' },
  },
  {
    key: 'kbSettings',
    icon: 'setting',
    titleKey: 'management.cards.kbSettings.title',
    descriptionKey: 'management.cards.kbSettings.description',
    requiredCaps: ['kb:settings_manage'],
    target: { kind: 'kbSettings' },
  },
  {
    key: 'auditors',
    icon: 'lock-on',
    titleKey: 'management.cards.auditors.title',
    descriptionKey: 'management.cards.auditors.description',
    requiredCaps: ['cr:audit_manage'],
    target: { kind: 'auditors' },
  },
  {
    key: 'skills',
    icon: 'system-code',
    titleKey: 'management.cards.skills.title',
    descriptionKey: 'management.cards.skills.description',
    requiredCaps: ['settings:skills_manage'],
    target: { kind: 'settings', section: 'skills' },
  },
]

export const MANAGEMENT_ENTRY_CAPS: readonly string[] = Array.from(
  new Set(
    MANAGEMENT_CARDS.flatMap((card) => [
      ...(card.requiredCaps ?? []),
      ...(card.requiredAnyCaps ?? []),
    ]),
  ),
)

export function canSeeManagement(caps: readonly string[]): boolean {
  return MANAGEMENT_CARDS.some((card) => canSeeManagementCard(card, caps))
}

export function visibleManagementCards(caps: readonly string[]): ManagementCardDefinition[] {
  return MANAGEMENT_CARDS.filter((card) => canSeeManagementCard(card, caps))
}

function canSeeManagementCard(card: ManagementCardDefinition, caps: readonly string[]): boolean {
  if (card.requiredCaps && !hasCapabilities(caps, card.requiredCaps)) return false
  if (
    card.requiredAnyCaps &&
    card.requiredAnyCaps.length > 0 &&
    !card.requiredAnyCaps.some((cap) => caps.includes(cap))
  ) {
    return false
  }
  return true
}
