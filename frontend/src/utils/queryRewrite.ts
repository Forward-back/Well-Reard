export interface QueryRewriteSource {
  message_id?: string
  turn_offset?: number
  item_index?: number
  item_kind?: string
  text?: string
  match_index?: number
}

export interface QueryRewriteResolution {
  source_text: string
  target: string
  source?: QueryRewriteSource
}

export interface QueryRewriteMeta {
  resolutions?: QueryRewriteResolution[]
  must_keep?: string[]
  must_keep_passed?: boolean
  elapsed_ms?: number
  slot_count?: number
  recent_turns?: number
}

export interface QueryRewritePayload {
  original_query?: unknown
  rewritten_query?: unknown
  user_message_id?: unknown
  rewrite_meta?: unknown
}

type RewriteTarget = Record<string, unknown> & { role?: unknown; content?: unknown; id?: unknown }

function asRewriteMeta(value: unknown): QueryRewriteMeta | null {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return null
  return value as QueryRewriteMeta
}

/**
 * Attach the backend's coreference rewrite to the user row of the current
 * turn. The live row has no server id yet, so fall back to the last user row
 * (quick-answer turns are strictly serialized).
 */
export function applyQueryRewrite<T extends RewriteTarget>(
  messages: T[],
  payload: QueryRewritePayload | null | undefined,
): T | null {
  if (!Array.isArray(messages) || !payload) return null
  const rewritten = typeof payload.rewritten_query === 'string' ? payload.rewritten_query.trim() : ''
  if (!rewritten) return null
  const original = typeof payload.original_query === 'string' ? payload.original_query : ''
  const targetId = typeof payload.user_message_id === 'string' ? payload.user_message_id : ''

  let target: T | undefined
  if (targetId) {
    target = [...messages].reverse().find((item) => item?.role === 'user' && item?.id === targetId)
  }
  if (!target) {
    target = [...messages].reverse().find((item) => item?.role === 'user')
  }
  if (!target) return null
  if (rewritten === original || rewritten === String(target.content ?? '')) return null

  const mutable = target as Record<string, unknown>
  mutable.query_rewritten = rewritten
  mutable.query_original = original || String(target.content ?? '')
  const meta = asRewriteMeta(payload.rewrite_meta)
  if (meta) mutable.query_rewrite_meta = meta
  return target
}
