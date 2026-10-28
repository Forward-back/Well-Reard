export type ReferenceItemKind = 'web' | 'document' | 'tool'

import {
  buildCitationNumberMap,
  citationDocDisplayName,
  citationDocKey,
  selectCitedRefs,
} from './citationMarkdown.ts'

export type KnowledgeReferenceLike = {
  id?: string
  chunk_ids?: string[]
  knowledge_id?: string
  knowledge_title?: string
  knowledge_filename?: string
  knowledge_base_id?: string
  chunk_index?: number
  chunk_type?: string
  content?: string
  version_id?: string
  version_label?: string
  metadata?: Record<string, string>
}

export type ReferenceMetaChip = {
  text: string
  danger?: boolean
}

export type ReferenceListItem = {
  key: string
  kind: ReferenceItemKind
  index: number
  title: string
  url?: string
  domain?: string
  faviconUrl?: string
  snippet?: string
  chunkId?: string
  chunkIds?: string[]
  chunkIndex?: number
  knowledgeId?: string
  knowledgeBaseId?: string
  content?: string
  /** 与正文内联 [n] 上标一致的文档编号（仅文档类，来自合并后的文档顺序） */
  number?: number
  /** 机构名副标题 */
  subtitle?: string
  /** 元数据 chips（类型/日期/产品/版本/已废止） */
  metaChips?: ReferenceMetaChip[]
  filename?: string
}

export type ReferenceDrawerSection = {
  id: 'web' | 'documents' | 'tools'
  items: ReferenceListItem[]
}

export function normalizeReferenceUrl(url: string): string {
  const raw = String(url || '').trim()
  if (!raw) return ''
  try {
    const parsed = new URL(raw)
    parsed.hash = ''
    let pathname = parsed.pathname
    if (pathname.length > 1 && pathname.endsWith('/')) {
      pathname = pathname.slice(0, -1)
    }
    parsed.pathname = pathname
    return parsed.toString()
  } catch {
    return raw.replace(/\/$/, '')
  }
}

export function getWebSearchUrl(item: KnowledgeReferenceLike): string {
  if (item.metadata?.url) return item.metadata.url
  if (item.id && (item.id.startsWith('http://') || item.id.startsWith('https://'))) {
    return item.id
  }
  return ''
}

export function getDomainFromUrl(url: string): string {
  if (!url) return ''
  try {
    return new URL(url).hostname.replace(/^www\./i, '')
  } catch {
    return url
  }
}

export function getFaviconUrl(urlOrDomain: string): string {
  const domain = urlOrDomain.includes('://')
    ? getDomainFromUrl(urlOrDomain)
    : urlOrDomain.replace(/^www\./i, '')
  if (!domain) return ''
  return `https://www.google.com/s2/favicons?domain=${encodeURIComponent(domain)}&sz=32`
}

function truncateText(text: string, maxLen: number): string {
  const normalized = formatReferenceSnippet(text)
  if (normalized.length <= maxLen) return normalized
  return `${normalized.slice(0, maxLen)}…`
}

export function formatReferenceSnippet(text: string | undefined): string {
  let value = String(text || '').replace(/\s+/g, ' ').trim()
  if (!value) return ''
  value = value.replace(/^(\.{3}|…+)\s*/, '')
  value = value.replace(/!\[[^\]]*]\([^)]*\)/g, ' ')
  value = value.replace(/\[([^\]]+)]\([^)]*\)/g, '$1')
  value = value.replace(/`([^`]+)`/g, '$1')
  value = value.replace(/\*\*([^*]+)\*\*/g, '$1')
  value = value.replace(/\*([^*]+)\*/g, '$1')
  value = value.replace(/(^|\s)#+\s*/g, '$1')
  value = value.replace(/\s+/g, ' ').trim()
  return value
}

function isWebReference(item: KnowledgeReferenceLike): boolean {
  return item.chunk_type === 'web_search'
}

function isToolReference(item: KnowledgeReferenceLike): boolean {
  return item.chunk_type === 'tool_result'
}

function isLikelyUrl(text: string): boolean {
  const trimmed = String(text || '').trim()
  if (!trimmed) return false
  return /^https?:\/\//i.test(trimmed) || /^www\./i.test(trimmed)
}

function isSameReferenceUrl(a: string, b: string): boolean {
  const left = String(a || '').trim()
  const right = String(b || '').trim()
  if (!left || !right) return false
  if (left === right) return true
  return normalizeReferenceUrl(left) === normalizeReferenceUrl(right)
}

function resolveWebTitle(item: KnowledgeReferenceLike, url: string, domain: string): string {
  const metaTitle = item.metadata?.title?.trim()
  if (metaTitle && !isLikelyUrl(metaTitle)) return metaTitle

  const knowledgeTitle = item.knowledge_title?.trim()
  if (
    knowledgeTitle &&
    !isLikelyUrl(knowledgeTitle) &&
    !isSameReferenceUrl(knowledgeTitle, url) &&
    knowledgeTitle !== domain
  ) {
    return knowledgeTitle
  }

  return domain || 'Web page'
}

function buildWebItem(item: KnowledgeReferenceLike, index: number): ReferenceListItem | null {
  const url = getWebSearchUrl(item)
  if (!url) return null
  const domain = getDomainFromUrl(url)
  const title = resolveWebTitle(item, url, domain)
  const snippet =
    item.metadata?.snippet ||
    truncateText(item.content || '', 220)
  const normalizedUrl = normalizeReferenceUrl(url)
  return {
    key: `web:${normalizedUrl}`,
    kind: 'web',
    index,
    title,
    url,
    domain,
    faviconUrl: getFaviconUrl(url),
    snippet: snippet || undefined,
    chunkId: item.id,
    content: item.content,
  }
}

/** 文件名中的版本段（V1.0 等）。 */
function extractVersion(filename: string): string {
  const hit = String(filename || '').match(/(V\d+(?:\.\d+)*)/i)
  return hit ? hit[1].toUpperCase() : ''
}

// 引用卡标签（用户定稿）：只放 版本 / 创建者 / 所属知识库
function buildDocumentMetaChips(item: KnowledgeReferenceLike): ReferenceMetaChip[] {
  const chips: ReferenceMetaChip[] = []
  const md = item.metadata || {}
  const filename = item.knowledge_filename || item.knowledge_title || ''
  const version = item.version_label || extractVersion(filename)
  if (version) chips.push({ text: version })
  if (md.org_name) chips.push({ text: md.org_name })
  if (md.kb_name) chips.push({ text: md.kb_name })
  if (md.lifecycle_status === 'expired') chips.push({ text: '已废止', danger: true })
  return chips
}

function buildDocumentItem(
  item: KnowledgeReferenceLike,
  index: number,
  numberMap: Map<string, number>,
): ReferenceListItem {
  const chunkId = item.id || `${item.knowledge_id || 'doc'}-${item.chunk_index ?? index}`
  const filename = item.knowledge_title || item.knowledge_filename || ''
  const title = citationDocDisplayName(item) || filename || item.knowledge_id || 'Document'
  const documentKey =
    item.knowledge_id ||
    [item.knowledge_base_id, item.knowledge_title || item.knowledge_filename].filter(Boolean).join(':') ||
    chunkId
  return {
    key: `doc:${documentKey}`,
    kind: 'document',
    index,
    number: numberMap.get(citationDocKey(item)) || index,
    title,
    subtitle: item.metadata?.org_name || undefined,
    metaChips: buildDocumentMetaChips(item),
    filename,
    chunkId,
    chunkIds: item.chunk_ids,
    chunkIndex: typeof item.chunk_index === 'number' ? item.chunk_index : undefined,
    content: item.content,
    knowledgeId: item.knowledge_id,
    knowledgeBaseId: item.knowledge_base_id,
  }
}

function buildToolItem(item: KnowledgeReferenceLike, index: number): ReferenceListItem {
  const id = item.id || `tool-${index}`
  return {
    key: `tool:${id}`,
    kind: 'tool',
    index,
    title: item.knowledge_title || item.metadata?.title || 'Tool result',
    domain: item.metadata?.source || item.metadata?.tool || undefined,
    snippet: truncateText(item.content || '', 220) || undefined,
    chunkId: id,
    content: item.content,
  }
}

function getDocumentGroupKey(item: KnowledgeReferenceLike, index: number): string {
  if (item.knowledge_id) return item.knowledge_id
  const title = item.knowledge_title || item.knowledge_filename
  if (title) return [item.knowledge_base_id, title].filter(Boolean).join(':')
  return (
    item.id ||
    `doc-${index}`
  )
}

function mergeDocumentReferences(refs: KnowledgeReferenceLike[]): KnowledgeReferenceLike[] {
  const groups = new Map<string, KnowledgeReferenceLike & { content_parts?: string[] }>()

  refs.forEach((item, index) => {
    const key = getDocumentGroupKey(item, index)
    const content = String(item.content || '').trim()
    const chunkIds = Array.from(new Set([...(item.chunk_ids || []), ...(item.id ? [item.id] : [])]))
    const existing = groups.get(key)

    if (!existing) {
      groups.set(key, {
        ...item,
        id: item.id || key,
        content_parts: content ? [content] : [],
        chunk_ids: chunkIds,
      })
      return
    }

    if (!existing.knowledge_id && item.knowledge_id) existing.knowledge_id = item.knowledge_id
    if (!existing.knowledge_title && item.knowledge_title) existing.knowledge_title = item.knowledge_title
    if (!existing.knowledge_filename && item.knowledge_filename) existing.knowledge_filename = item.knowledge_filename
    if (!existing.knowledge_base_id && item.knowledge_base_id) existing.knowledge_base_id = item.knowledge_base_id
    for (const chunkId of chunkIds) {
      if (!existing.chunk_ids?.includes(chunkId)) {
        existing.chunk_ids = [...(existing.chunk_ids || []), chunkId]
      }
    }
    if (content && !existing.content_parts?.includes(content)) {
      existing.content_parts = [...(existing.content_parts || []), content]
    }
  })

  return Array.from(groups.values()).map((item) => {
    const { content_parts: contentParts, ...rest } = item
    return {
      ...rest,
      content: contentParts?.slice(0, 3).join('\n\n') || rest.content,
    }
  })
}

function mergeWebReferences(refs: KnowledgeReferenceLike[]): KnowledgeReferenceLike[] {
  const groups = new Map<string, KnowledgeReferenceLike>()

  refs.forEach((item, index) => {
    const url = getWebSearchUrl(item)
    const key = url ? normalizeReferenceUrl(url) : item.id || `web-${index}`
    if (!groups.has(key)) {
      groups.set(key, item)
    }
  })

  return Array.from(groups.values())
}

export function buildReferenceSections(
  refs: KnowledgeReferenceLike[] | null | undefined,
): ReferenceDrawerSection[] {
  const list = Array.isArray(refs) ? refs.filter(Boolean) : []
  if (!list.length) return []

  const webReferences: KnowledgeReferenceLike[] = []
  const documentReferences: KnowledgeReferenceLike[] = []
  const toolReferences: KnowledgeReferenceLike[] = []
  const webItems: ReferenceListItem[] = []
  const docItems: ReferenceListItem[] = []
  const toolItems: ReferenceListItem[] = []
  let webIndex = 0
  let docIndex = 0
  let toolIndex = 0

  for (const item of list) {
    if (isWebReference(item)) {
      webReferences.push(item)
      continue
    }
    if (isToolReference(item)) {
      toolReferences.push(item)
      continue
    }
    documentReferences.push(item)
  }

  for (const item of mergeWebReferences(webReferences)) {
    const built = buildWebItem(item, ++webIndex)
    if (built) webItems.push(built)
  }

  // 只展示实际被答案引用的文档（有对齐位置时过滤掉未被命中的检索结果）
  const citedDocs = selectCitedRefs(documentReferences) as KnowledgeReferenceLike[]
  const documentNumberMap = buildCitationNumberMap(citedDocs)
  for (const item of mergeDocumentReferences(citedDocs)) {
    docItems.push(buildDocumentItem(item, ++docIndex, documentNumberMap))
  }
  docItems.sort((a, b) => (a.number || a.index) - (b.number || b.index))

  for (const item of toolReferences) {
    toolItems.push(buildToolItem(item, ++toolIndex))
  }

  const sections: ReferenceDrawerSection[] = []
  if (webItems.length) sections.push({ id: 'web', items: webItems })
  if (docItems.length) sections.push({ id: 'documents', items: docItems })
  if (toolItems.length) sections.push({ id: 'tools', items: toolItems })
  return sections
}

export function buildReferenceList(
  refs: KnowledgeReferenceLike[] | null | undefined,
): ReferenceListItem[] {
  return buildReferenceSections(refs).flatMap((section) => section.items)
}

export type ReferenceHighlightTarget = {
  url?: string
  chunkId?: string
  documentTitle?: string
  knowledgeBaseId?: string
  key?: string
}

export function resolveReferenceHighlightKey(
  refs: KnowledgeReferenceLike[] | null | undefined,
  target: ReferenceHighlightTarget | null | undefined,
): string | null {
  if (!target) return null
  if (target.key) return target.key

  const items = buildReferenceList(refs)
  if (!items.length) return null

  if (target.url) {
    const normalized = normalizeReferenceUrl(target.url)
    const hit = items.find(
      (item) => item.kind === 'web' && item.url && normalizeReferenceUrl(item.url) === normalized,
    )
    if (hit) return hit.key
  }

  if (target.chunkId) {
    const raw = String(target.chunkId).trim()
    const hit = items.find(
      (item) =>
        item.chunkId === raw ||
        item.chunkIds?.includes(raw) ||
        item.key === `doc:${raw}` ||
        (item.kind === 'web' && (item.chunkId === raw || item.url === raw)),
    )
    if (hit) return hit.key
  }

  // Defensive fallback for historical/partially streamed Agent messages whose
  // tool replay payload no longer contains the exact cited chunk. The public
  // citation still carries the full document title and KB id, which identify
  // the same document-level drawer card.
  if (target.documentTitle) {
    const title = target.documentTitle.trim().toLowerCase()
    const candidates = items.filter(
      (item) =>
        item.kind === 'document' &&
        (item.title.trim().toLowerCase() === title ||
          String(item.filename || '').trim().toLowerCase() === title),
    )
    const scoped = target.knowledgeBaseId
      ? candidates.find((item) => item.knowledgeBaseId === target.knowledgeBaseId)
      : undefined
    if (scoped) return scoped.key
    if (candidates.length === 1) return candidates[0].key
  }

  return null
}
