import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import test from 'node:test'
import { fileURLToPath } from 'node:url'

import {
  CAPABILITY_KEYS,
  CAPABILITY_LABELS,
  capsForRoles,
  hasCapabilities,
  isKnownCapability,
} from './capabilities'

const here = dirname(fileURLToPath(import.meta.url))

test('every contract key has a human-readable label', () => {
  for (const key of CAPABILITY_KEYS) {
    assert.ok(CAPABILITY_LABELS[key], `missing label for ${key}`)
  }
})

test('capability derivation mirrors the role contract', () => {
  const admin = capsForRoles('company_admin', 'viewer')
  assert.ok(admin.includes('member:manage'))
  assert.ok(!admin.includes('kb:create'))

  const manager = capsForRoles('member', 'manager')
  assert.ok(manager.includes('kb:create'))
  assert.ok(manager.includes('kb:share_manage'))
  assert.ok(hasCapabilities(manager, ['kb:acl_manage', 'folder:manage']))

  const knowledgeAdmin = capsForRoles('member', 'knowledge_admin')
  assert.ok(!knowledgeAdmin.includes('kb:share_manage'))
  assert.ok(!knowledgeAdmin.includes('kb:acl_manage'))
  assert.ok(knowledgeAdmin.includes('kb:settings_manage'))
})

test('route meta only references contract capability keys', () => {
  const routerSource = readFileSync(resolve(here, '../router/index.ts'), 'utf8')
  const matches = routerSource.matchAll(/required(?:Any)?Caps:\s*\[([^\]]*)\]/g)
  let checked = 0
  for (const match of matches) {
    for (const raw of match[1].split(',')) {
      const key = raw.trim().replace(/^['"]|['"]$/g, '')
      if (!key) continue
      checked += 1
      assert.ok(isKnownCapability(key), `unknown capability in router meta: ${key}`)
    }
  }
  assert.ok(checked > 0, 'expected at least one requiredCaps declaration')
})
