import assert from 'node:assert/strict'
import test from 'node:test'

import { capsForRoles, isKnownCapability } from './capabilities'
import {
  MANAGEMENT_CARDS,
  MANAGEMENT_ENTRY_CAPS,
  canSeeManagement,
  visibleManagementCards,
} from './managementCards'

const keysFor = (companyRole: string, departmentRole: string): string[] =>
  visibleManagementCards(capsForRoles(companyRole, departmentRole)).map((card) => card.key)

test('company admin sees the management action set', () => {
  assert.deepEqual(keysFor('company_admin', ''), [
    'members',
    'departments',
  ])
})

test('department manager sees members, auditors and skills', () => {
  assert.deepEqual(keysFor('member', 'manager'), ['members', 'auditors', 'skills'])
})

test('knowledge admin sees KB settings and skills', () => {
  assert.deepEqual(keysFor('member', 'knowledge_admin'), ['kbSettings', 'skills'])
})

test('editor and viewer do not get a management entry', () => {
  assert.equal(canSeeManagement(capsForRoles('member', 'editor')), false)
  assert.equal(canSeeManagement(capsForRoles('member', 'viewer')), false)
})

test('system admin identity alone does not grant company management', () => {
  const caps = capsForRoles('member', '', true)
  assert.equal(canSeeManagement(caps), false)
})

test('every management cap is declared in the shared contract', () => {
  assert.ok(MANAGEMENT_ENTRY_CAPS.length > 0)
  for (const cap of MANAGEMENT_ENTRY_CAPS) {
    assert.ok(isKnownCapability(cap), `unknown management capability: ${cap}`)
  }
})

test('management cards have unique keys and a cap declaration', () => {
  const keys = MANAGEMENT_CARDS.map((card) => card.key)
  assert.equal(new Set(keys).size, keys.length)
  for (const card of MANAGEMENT_CARDS) {
    assert.ok(
      (card.requiredCaps?.length ?? 0) + (card.requiredAnyCaps?.length ?? 0) > 0,
      `management card ${card.key} has no capability requirement`,
    )
  }
})
