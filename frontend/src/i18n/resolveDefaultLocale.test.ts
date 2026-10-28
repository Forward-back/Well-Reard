import assert from 'node:assert/strict'
import { test } from 'node:test'

import { BUILT_IN_DEFAULT, resolveDefaultLocale } from './resolveDefaultLocale.ts'

test('resolveDefaultLocale only accepts zh-CN', () => {
  assert.equal(resolveDefaultLocale('zh-CN', undefined), 'zh-CN')
  assert.equal(resolveDefaultLocale('', 'zh-CN'), 'zh-CN')
  assert.equal(resolveDefaultLocale(undefined, undefined), BUILT_IN_DEFAULT)
})

test('resolveDefaultLocale rejects other and unknown values', () => {
  assert.equal(resolveDefaultLocale('en-US', 'ru-RU'), BUILT_IN_DEFAULT)
  assert.equal(resolveDefaultLocale('fr-FR'), BUILT_IN_DEFAULT)
  assert.equal(resolveDefaultLocale('en-US"};alert(1);//'), BUILT_IN_DEFAULT)
  assert.equal(resolveDefaultLocale('   '), BUILT_IN_DEFAULT)
})
