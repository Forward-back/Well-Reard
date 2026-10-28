/** Shared citation tag preprocessing for chat markdown (QA + agent). */

/** Self-closing or unclosed `<kb/>` / `<web/>` tags from model output. */
export const KB_WEB_TAG_RE = /<(?:kb|web)\b[^>]*?\s*\/?>/g
const KB_TAG_ATTR_RE = /<kb\b([^>]*?)\s*\/?>/g
const WEB_TAG_ATTR_RE = /<web\b([^>]*?)\s*\/?>/g

const ATTRIBUTE_REGEX = /([\w-]+)\s*=\s*"([^"]*)"/g
const UUID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i

/**
 * Hide a citation tag while the typewriter has only emitted part of it.
 *
 * Without this guard, Markdown renders the leading `<` as ordinary text until
 * the closing `>` arrives. Only the unfinished tail is removed; a complete tag
 * continues through the normal citation pipeline.
 */
export function stripIncompleteCitationTag(content: string): string {
  if (!content) return content

  const start = content.lastIndexOf('<')
  if (start < 0) return content

  const tail = content.slice(start)
  if (tail.includes('>')) return content

  const isCitationPrefix = tail === '<'
    || /^<k(?:b(?:\s[\s\S]*)?)?$/i.test(tail)
    || /^<w(?:e(?:b(?:\s[\s\S]*)?)?)?$/i.test(tail)

  return isCitationPrefix ? content.slice(0, start) : content
}

export type CitationKnowledgeRef = {
  id?: string
  knowledge_id?: string
  knowledge_title?: string
  knowledge_filename?: string
  chunk_index?: number
  chunk_type?: string
  knowledge_base_id?: string
  /** chunk 原文摘要（YLSW 透传，悬浮引用卡直接展示） */
  content?: string
  /** 答案文本内对齐位置（YLSW 后处理回填），供内联 [n] 上标注入 */
  start_at?: number
  end_at?: number
  /** 回答生成时绑定的文档版本（落库随引用保存） */
  version_id?: string
  version_label?: string
  metadata?: Record<string, string>
}

function parseTagAttributes(attrString: string): Record<string, string> {
  const attributes: Record<string, string> = {}
  if (!attrString) return attributes
  ATTRIBUTE_REGEX.lastIndex = 0
  let match: RegExpExecArray | null
  while ((match = ATTRIBUTE_REGEX.exec(attrString)) !== null) {
    attributes[match[1]] = match[2]
  }
  return attributes
}

function escapeHtml(text: string): string {
  return String(text)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
}

function truncateMiddle(text: string, maxLength = 13): string {
  if (!text) return ''
  if (text.length <= maxLength) return text
  const half = Math.floor((maxLength - 3) / 2)
  const start = text.slice(0, half + ((maxLength - 3) % 2))
  const end = text.slice(-half)
  return `${start}...${end}`
}

function normalizeDocTitle(title: string): string {
  return title.trim().toLowerCase()
}

function docTitlesMatch(a: string, b: string): boolean {
  if (!a || !b) return false
  const na = normalizeDocTitle(a)
  const nb = normalizeDocTitle(b)
  return na === nb || na.includes(nb) || nb.includes(na)
}

/** Map model context index (1, FAQ-1, DOC-2) to the real chunk UUID from retrieval results. */
export function resolveCitationChunkId(
  rawChunkId: string,
  attrs: { doc?: string; kbId?: string },
  refs?: CitationKnowledgeRef[] | null,
): string {
  const raw = String(rawChunkId || '').trim()
  if (!raw || UUID_RE.test(raw)) return raw

  const list = (refs || []).filter((r) => r && r.chunk_type !== 'web_search')
  if (!list.length) return raw

  const doc = (attrs.doc || '').trim()
  const kbId = (attrs.kbId || '').trim()

  if (doc) {
    const byDoc = list.find(
      (r) =>
        docTitlesMatch(doc, r.knowledge_title || '') ||
        docTitlesMatch(doc, r.knowledge_filename || ''),
    )
    if (byDoc?.id) return byDoc.id
  }

  const faqMatch = raw.match(/^FAQ-(\d+)$/i)
  if (faqMatch) {
    const faqRefs = list.filter((r) => r.chunk_type === 'faq')
    const hit = faqRefs[parseInt(faqMatch[1], 10) - 1]
    if (hit?.id) return hit.id
  }

  const docMatch = raw.match(/^DOC-(\d+)$/i)
  if (docMatch) {
    const docRefs = list.filter((r) => r.chunk_type !== 'faq')
    const hit = docRefs[parseInt(docMatch[1], 10) - 1]
    if (hit?.id) return hit.id
  }

  const num = parseInt(raw, 10)
  if (!Number.isNaN(num) && String(num) === raw) {
    const byPos = list[num - 1]
    if (byPos?.id) return byPos.id
    const byChunkIndex = list.find((r) => r.chunk_index === num || r.chunk_index === num - 1)
    if (byChunkIndex?.id) return byChunkIndex.id
  }

  if (kbId) {
    const scoped = list.filter((r) => r.knowledge_base_id === kbId)
    if (doc) {
      const byDoc = scoped.find(
        (r) =>
          docTitlesMatch(doc, r.knowledge_title || '') ||
          docTitlesMatch(doc, r.knowledge_filename || ''),
      )
      if (byDoc?.id) return byDoc.id
    }
    if (scoped.length === 1 && scoped[0].id) return scoped[0].id
  }

  return raw
}

/** Convert <web/> / <kb/> / [[gaze]] tags into inline citation HTML. */export function preprocessCitationTags(
  contentStr: string,
  refs?: CitationKnowledgeRef[] | null,
): string {
  if (!contentStr.trim()) return ''

  return contentStr
    .replace(WEB_TAG_ATTR_RE, (_m, attrString: string) => {
      const attrs = parseTagAttributes(attrString)
      const url = attrs.url || ''
      const title = attrs.title || ''
      if (!url) return ''

      let domain = url
      try {
        const u = new URL(url)
        const host = u.hostname || ''
        const parts = host.split('.')
        domain = parts.length >= 2 ? parts.slice(-2).join('.') : host || url
      } catch {
        // keep original
      }
      const safeTitle = escapeHtml(title)
      const safeUrl = escapeHtml(url)
      return `<a class="citation citation-web" data-url="${safeUrl}" href="${safeUrl}" target="_blank" rel="noopener noreferrer"><span class="citation-icon citation-icon--web" aria-hidden="true"></span><span class="citation-domain">${domain}</span><span class="citation-tip"><span class="tip-title">${safeTitle}</span><span class="tip-url">${safeUrl}</span></span></a>`
    })
    .replace(KB_TAG_ATTR_RE, (_m, attrString: string) => {
      const attrs = parseTagAttributes(attrString)
      const doc = attrs.doc || ''
      const rawChunkId = attrs.chunk_id || attrs.chunkId || ''
      const kbId = attrs.kb_id || attrs.kbId || ''
      const chunkId = resolveCitationChunkId(rawChunkId, { doc, kbId }, refs)
      if (!doc || !chunkId) return ''

      const safeDoc = escapeHtml(doc)
      const safeKbId = escapeHtml(kbId)
      const safeChunkId = escapeHtml(chunkId)
      const displayDoc = escapeHtml(truncateMiddle(doc))
      return `<span class="citation citation-kb" data-kb-id="${safeKbId}" data-chunk-id="${safeChunkId}" data-doc="${safeDoc}" role="button" tabindex="0"><span class="citation-icon citation-icon--book" aria-hidden="true"></span><span class="citation-text">${displayDoc}</span><span class="citation-tip"><span class="tip-loading">…</span></span></span>`
    })
    .replace(/\[\[([^\]]+)\]\]/g, (match, inner: string) => {
      const pipeIdx = inner.indexOf('|')
      const slug = pipeIdx > 0 ? inner.substring(0, pipeIdx).trim() : inner.trim()
      if (!slug) return match
      let display = slug
      if (pipeIdx > 0) {
        display = inner.substring(pipeIdx + 1).trim()
      } else {
        const parts = slug.split('/')
        display = parts.length > 1 ? parts.slice(1).join('/') : slug
      }
      return `<a href="#" class="gaze-content-link citation-gaze" data-slug="${escapeHtml(slug)}">${escapeHtml(display)}</a>`
    })
}

const HTML_PLACEHOLDER_RE = /@@GAZE_HTML_PLACEHOLDER_(\d+)@@/g
const SUP_PLACEHOLDER_RE = /@@GAZE_SUP_PLACEHOLDER_(\d+)@@/g

/** 引用文档的去重键（同一文档多个 chunk 共用编号）。 */
export function citationDocKey(ref: CitationKnowledgeRef | null | undefined): string {
  if (!ref) return ''
  return String(ref.knowledge_id || ref.knowledge_filename || ref.knowledge_title || ref.id || '')
}

/**
 * 文档显示名：优先 doc_title 元数据（但 YLSW 存量数据 doc_title 常为文件名，
 * 形如 xxx_xxx_V1.0_ZH.md，需识别并跳过）；否则从文件名
 * （TYPE_DATE_ORG_产品_标题_V1.0_ZH.md）解析出标题段；再退回文号/机构/原文件名。
 */
export function citationDocDisplayName(ref: CitationKnowledgeRef | null | undefined): string {
  if (!ref) return ''
  const md = ref.metadata || {}
  const name = ref.knowledge_filename || ref.knowledge_title || ''
  const looksLikeFilename = (s: string) =>
    /_/.test(s) && (/\.(md|markdown|pdf|docx?|txt)$/i.test(s) || s === name)
  if (md.doc_title && !looksLikeFilename(md.doc_title)) return md.doc_title
  const base = name.replace(/\.[a-z0-9]{1,6}$/i, '')
  if (base.includes('_')) {
    const parts = base.split('_').filter(Boolean)
    const vIdx = parts.findIndex((p) => /^V\d/i.test(p))
    const end = vIdx > 4 ? vIdx : Math.max(4, parts.length - 2)
    const title = parts.slice(4, end).join('_')
    if (title) return title
  }
  if (md.doc_title) return md.doc_title
  return md.doc_number || md.org_name || base || name
}

/** 引用是否携带答案内对齐位置。 */
export function isPositionedRef(ref: CitationKnowledgeRef | null | undefined): boolean {
  return Boolean(
    ref &&
    typeof ref.start_at === 'number' &&
    typeof ref.end_at === 'number' &&
    ref.start_at >= 0 &&
    (ref.end_at as number) > (ref.start_at as number),
  )
}

/**
 * 实际被答案引用的文档：有任意一条带位置时只保留带位置的引用
 * （未被对齐命中的检索结果不算“引用”，不进入面板/底部列表）；
 * 全部无位置时（Agent 工具引用/历史消息）回退全量。
 */
export function selectCitedRefs(
  refs?: CitationKnowledgeRef[] | null,
): CitationKnowledgeRef[] {
  const list = (refs || []).filter((r) => r && r.chunk_type !== 'web_search')
  const positioned = list.filter(isPositionedRef)
  return positioned.length ? positioned : list
}

/**
 * 按 refs 为每个文档分配 [n] 编号（1 起），同一文档的多个 chunk 共用编号。
 * 存在对位位置时按答案内出现顺序编号（首个引用位置最早 = [1]），
 * 否则按检索排序编号。内联上标与引用面板/底部列表共用本函数。
 */
export function buildCitationNumberMap(
  refs?: CitationKnowledgeRef[] | null,
): Map<string, number> {
  const cited = selectCitedRefs(refs)
  const items = cited.some(isPositionedRef)
    ? [...cited].sort((a, b) => (a.start_at as number) - (b.start_at as number))
    : cited
  const map = new Map<string, number>()
  for (const ref of items) {
    const key = citationDocKey(ref)
    if (!key || map.has(key)) continue
    map.set(key, map.size + 1)
  }
  return map
}

/**
 * 按引用携带的答案内位置（start_at/end_at，YLSW 后处理对齐回填）注入
 * `[n]` 上标 HTML 占位符（n 为文档编号）。无位置则原样返回。
 *
 * - 同文档多个引用共用编号；web_search 引用不参与
 * - 位置落在代码块/表格片段内时跳过，避免破坏 markdown 结构
 * - 按结束偏移降序插入，保证前序偏移不被插入内容挪动
 */
export function injectPositionedCitations(
  contentStr: string,
  refs?: CitationKnowledgeRef[] | null,
): { content: string; snippets: string[] } {
  if (!contentStr || !refs?.length) return { content: contentStr, snippets: [] }

  const numberMap = buildCitationNumberMap(refs)
  const seen = new Set<string>()
  const candidates: CitationKnowledgeRef[] = []
  for (const r of refs) {
    if (!r || !r.id || r.chunk_type === 'web_search') continue
    if (typeof r.start_at !== 'number' || typeof r.end_at !== 'number') continue
    if (r.start_at < 0 || r.end_at <= r.start_at) continue
    const key = citationDocKey(r)
    const number = numberMap.get(key)
    if (!number || seen.has(r.id)) continue
    seen.add(r.id)
    candidates.push(r)
  }
  if (!candidates.length) return { content: contentStr, snippets: [] }

  const snippets: string[] = []
  const htmlPlaceholder = (ref: CitationKnowledgeRef, number: number, hitText: string): string => {
    const idx = snippets.length
    const doc = escapeHtml(citationDocDisplayName(ref))
    const chunkId = escapeHtml(ref.id || '')
    const citeText = escapeHtml(String(hitText || '').replace(/\s+/g, ' ').trim())
    snippets.push(
      `<sup class="cite-num" data-chunk-id="${chunkId}" data-doc="${doc}"${citeText ? ` data-cite-text="${citeText}"` : ''} role="button" tabindex="0">[${number}]</sup>`,
    )
    return `@@GAZE_SUP_PLACEHOLDER_${idx}@@`
  }

  const ordered = [...candidates].sort((a, b) => (b.end_at as number) - (a.end_at as number))
  let result = contentStr
  for (const r of ordered) {
    const start = Math.max(0, r.start_at as number)
    const end = Math.min(result.length, r.end_at as number)
    if (start >= end) continue
    const frag = result.slice(start, end)
    if (frag.includes('`') || frag.includes('|')) continue
    const key = citationDocKey(r)
    const number = numberMap.get(key) || 0
    result = result.slice(0, end) + htmlPlaceholder(r, number, frag) + result.slice(end)
  }
  return { content: result, snippets }
}

/** 还原 [n] 上标占位符。 */
export function restoreSupPlaceholders(html: string, snippets: string[]): string {
  if (!snippets.length || !html.includes('@@GAZE_SUP_PLACEHOLDER_')) return html
  return html.replace(SUP_PLACEHOLDER_RE, (_m, idx: string) => snippets[Number(idx)] || '')
}

/** Protect citation HTML from markdown parser; restore after marked.parse. */
export function extractCitationHtmlPlaceholders(
  contentStr: string,
  refs?: CitationKnowledgeRef[] | null,
): { content: string; htmlSnippets: string[] } {
  const htmlSnippets: string[] = []
  const storeHtml = (html: string): string => {
    const idx = htmlSnippets.length
    htmlSnippets.push(html)
    return `@@GAZE_HTML_PLACEHOLDER_${idx}@@`
  }

  const content = contentStr
    .replace(KB_WEB_TAG_RE, (match) => storeHtml(preprocessCitationTags(match, refs)))
    .replace(/\[\[([^\]]+)\]\]/g, (match) => storeHtml(preprocessCitationTags(match, refs)))

  return { content, htmlSnippets }
}

export function restoreCitationHtmlPlaceholders(html: string, htmlSnippets: string[]): string {
  if (!htmlSnippets.length) return html
  return html.replace(HTML_PLACEHOLDER_RE, (_match, idx) => htmlSnippets[Number(idx)] || '')
}

/** Opening/closing fence for GFM fenced code blocks (up to 3 spaces indent). */
const FENCED_CODE_DELIMITER_RE = /^ {0,3}(`{3,}|~{3,})(\s*\S.*)?\s*$/

function isFencedCodeDelimiterLine(line: string): boolean {
  return FENCED_CODE_DELIMITER_RE.test(line)
}

/** Collapse newlines around <kb/> / <web/> so marked keeps citations inline. */
export function joinCitationTagsToPreviousLine(content: string): string {
  if (!content) return content

  let result = content

  // Newlines between consecutive citation tags
  let prev = ''
  while (result !== prev) {
    prev = result
    result = result.replace(
      /(<(?:kb|web)\b[^>]*?\s*\/?>)\s*\n+\s*(<(?:kb|web)\b)/gi,
      '$1 $2',
    )
  }

  // Blank lines before citations: join to the previous content. Fenced-code
  // delimiters are the only exception because ``` / ~~~ must stay on their own line.
  result = result.replace(/\n[ \t]*\n+([ \t]*<(?:kb|web)\b)/gi, (match, kbStart, offset, full) => {
    const before = full.slice(0, offset)
    const lastLine = before.split('\n').filter((line: string) => line.trim()).pop() || ''
    if (isFencedCodeDelimiterLine(lastLine)) {
      return `\n\n${kbStart}`
    }
    return ` ${kbStart.trimStart()}`
  })

  // Single newline before citation when it follows text or another citation (not after a blank line)
  result = result.replace(
    /(?<!\n)(<(?:kb|web)\b[^>]*?\s*\/?>|[ \t]*\S[^\n]*?)\n([ \t]*<(?:kb|web)\b)/g,
    (match, beforePart: string, kbStart: string, offset: number, full: string) => {
      // Resolve the full preceding line: lazy capture + lookbehind can grab only a
      // partial line (e.g. ``` captured as ``), which would skip the fence check.
      const lineStart = full.lastIndexOf('\n', offset - 1) + 1
      const fullPrevLine = full.slice(lineStart, offset + beforePart.length)
      if (isFencedCodeDelimiterLine(fullPrevLine)) {
        return match
      }
      return `${beforePart} ${kbStart.trimStart()}`
    },
  )

  return result
}

const CITATION_HTML_FRAGMENT =
  '(?:<span class="citation\\b[^]*?</span>|<a class="citation\\b[^]*?</a>)'

/** Merge citation-only <p> blocks into the preceding paragraph (marked splits on newlines). */
export function collapseStandaloneCitationParagraphs(html: string): string {
  if (!html || !html.includes('citation')) return html

  const mergePattern = new RegExp(
    `(<\\/(?:p|li)>)\\s*(?:<p>\\s*<\\/p>\\s*)*<p>\\s*(${CITATION_HTML_FRAGMENT})\\s*<\\/p>`,
    'g',
  )

  let result = html
  let prev = ''
  while (result !== prev) {
    prev = result
    result = result.replace(mergePattern, (_match, closeTag: string, citation: string) => {
      return ` ${citation}${closeTag}`
    })
  }

  return result
}

/** Preserve raw <kb>/<web> tags before sanitizers that would strip them. */
export function preserveCitationTags(contentStr: string): { text: string; tags: string[] } {
  const tags: string[] = []
  const text = contentStr.replace(KB_WEB_TAG_RE, (match) => {
    const idx = tags.length
    tags.push(match)
    return `\x00TAG${idx}\x00`
  })
  return { text, tags }
}

export function restoreCitationTags(text: string, tags: string[]): string {
  return text.replace(/\x00TAG(\d+)\x00/g, (_, idx) => tags[Number(idx)] || '')
}
