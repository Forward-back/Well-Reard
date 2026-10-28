import assert from 'node:assert/strict'
import test from 'node:test'
import { applyQueryRewrite } from './queryRewrite.ts'

test('attaches the rewrite to the trailing user row when the live row has no id', () => {
  const messages: Array<Record<string, unknown>> = [
    { role: 'user', content: '2024年经济热词有哪些？' },
    { role: 'assistant', content: '……' },
    { role: 'user', content: '它是什么？' },
  ]
  const target = applyQueryRewrite(messages, {
    original_query: '它是什么？',
    rewritten_query: '2024年经济热词是什么？',
    user_message_id: 'server-user-id',
    rewrite_meta: {
      resolutions: [{ source_text: '它', target: '2024年经济热词' }],
      elapsed_ms: 420,
    },
  })
  assert.equal(target, messages[2])
  assert.equal(messages[2].query_rewritten, '2024年经济热词是什么？')
  assert.equal(messages[2].query_original, '它是什么？')
  assert.deepEqual(messages[2].query_rewrite_meta, {
    resolutions: [{ source_text: '它', target: '2024年经济热词' }],
    elapsed_ms: 420,
  })
})

test('prefers an exact server user id match', () => {
  const messages: Array<Record<string, unknown>> = [
    { role: 'user', id: 'old', content: '上一条' },
    { role: 'user', id: 'new', content: '它呢' },
  ]
  const target = applyQueryRewrite(messages, {
    original_query: '它呢',
    rewritten_query: '云闪付的费率是多少？',
    user_message_id: 'old',
  })
  assert.equal(target, messages[0])
  assert.equal(messages[0].query_rewritten, '云闪付的费率是多少？')
  assert.equal(messages[1].query_rewritten, undefined)
})

test('ignores empty or unchanged rewrites', () => {
  const messages: Array<Record<string, unknown>> = [{ role: 'user', content: '它是什么？' }]
  assert.equal(applyQueryRewrite(messages, { rewritten_query: '' }), null)
  assert.equal(
    applyQueryRewrite(messages, { original_query: '它是什么？', rewritten_query: '它是什么？' }),
    null,
  )
  assert.equal(messages[0].query_rewritten, undefined)
})
