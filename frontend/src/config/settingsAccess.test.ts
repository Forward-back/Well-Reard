import assert from 'node:assert/strict'
import test from 'node:test'

import {
  SETTINGS_SECTION_CAPS,
  SYSTEM_ADMIN_SETTINGS_SECTIONS,
} from './settingsAccess'

test('management settings sections require governance capabilities', () => {
  assert.deepEqual(SETTINGS_SECTION_CAPS.members, ['member:manage'])
  assert.deepEqual(SETTINGS_SECTION_CAPS.departments, ['department:manage'])
  assert.deepEqual(SETTINGS_SECTION_CAPS.models, ['settings:models_manage'])
  assert.deepEqual(SETTINGS_SECTION_CAPS.integrations, ['settings:integrations_manage'])
})

test('skills page requires both manage and view capability', () => {
  assert.deepEqual(SETTINGS_SECTION_CAPS.skills, ['skills:view'])
})

test('removed agent surfaces no longer have settings entries', () => {
  for (const key of ['sandbox', 'envvars', 'websearch']) {
    assert.equal(
      Object.prototype.hasOwnProperty.call(SETTINGS_SECTION_CAPS, key),
      false,
      `${key} should not be a settings section`,
    )
  }
})

test('system administration settings stay explicitly system-admin-only', () => {
  assert.deepEqual(
    [...SYSTEM_ADMIN_SETTINGS_SECTIONS],
    ['system-global', 'runtime-queues', 'platform-api-keys', 'system-audit-log'],
  )
})
