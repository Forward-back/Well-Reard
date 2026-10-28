import { onBeforeUnmount, onMounted, ref, watch, type Ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { getChunkByIdOnly } from '@/api/knowledge-base'
import { resolveCitationChunkId, type CitationKnowledgeRef } from '@/utils/citationMarkdown'
import {
  getCitationChunkCache,
  setCitationChunkCache,
} from '@/utils/citationChunkCache'
import { useChatReferencesDrawer } from '@/composables/useChatReferencesDrawer'

export { clearCitationChunkCache } from '@/utils/citationChunkCache'

type FloatState = {
  visible: boolean
  type: 'kb' | 'web'
  top: number
  left: number
  title: string
  content: string
  url: string
  loading: boolean
  error: string
  meta?: {
    doc_number?: string
    org_name?: string
    lifecycle_status?: string
    version_label?: string
  }
  /** 该 [n] 上标对应的答案命中句（悬浮卡内高亮） */
  citeText?: string
}

export type CitationFloatState = FloatState

export type ChatCitationPopoverOptions = {
  getKnowledgeReferences?: () => CitationKnowledgeRef[] | null | undefined
  sessionId?: () => string | undefined
  getFeedbackContext?: () => { userQuery?: string; answer?: string }
}

export function useChatCitationPopover(
  rootRef: Ref<HTMLElement | null>,
  options?: ChatCitationPopoverOptions,
) {
  const { t } = useI18n()
  const referencesDrawer = useChatReferencesDrawer()

  const getCacheScope = () => options?.sessionId?.() || 'default'

  const float = ref<FloatState>({
    visible: false,
    type: 'kb',
    top: 0,
    left: 0,
    title: '',
    content: '',
    url: '',
    loading: false,
    error: '',
    citeText: '',
  })

  let hoverTimer: number | null = null
  let closeTimer: number | null = null

  const positionFor = (el: HTMLElement, offsetY = 0) => {
    const rect = el.getBoundingClientRect()
    float.value.top = rect.bottom + window.scrollY + 6 + offsetY
    float.value.left = Math.min(rect.left + window.scrollX, window.innerWidth - 320)
  }

  const openWeb = (el: HTMLElement) => {
    const url = el.getAttribute('data-url') || ''
    float.value.type = 'web'
    float.value.url = url
    float.value.title = el.querySelector('.tip-title')?.textContent || ''
    float.value.content = ''
    float.value.loading = false
    float.value.error = ''
    float.value.citeText = ''
    float.value.visible = true
    positionFor(el)
  }

  const fetchChunkContent = async (chunkId: string) => getChunkByIdOnly(chunkId)

  const openKb = async (el: HTMLElement) => {
    const rawChunkId = el.getAttribute('data-chunk-id') || ''
    const title = el.getAttribute('data-doc') || ''
    const kbId = el.getAttribute('data-kb-id') || ''
    const refs = options?.getKnowledgeReferences?.() || []
    // 先按引用 id 精确匹配：YLSW chunk id 是纯数字，若先走
    // resolveCitationChunkId 会被误当列表下标解析到别的引用。
    const exactRef = refs.find((r) => r?.id === rawChunkId)
    const chunkId = exactRef
      ? rawChunkId
      : resolveCitationChunkId(rawChunkId, { doc: title, kbId }, refs) || rawChunkId
    if (!chunkId) return
    float.value.type = 'kb'
    float.value.url = ''
    float.value.visible = true
    const ref = exactRef || refs.find((r) => r?.id === chunkId)
    float.value.title = title || ref?.knowledge_title || ref?.knowledge_filename || ''
    const md = ref?.metadata
    const versionLabel = ref?.version_label
    float.value.meta = (md && (md.doc_number || md.org_name || md.lifecycle_status)) || versionLabel
      ? {
          doc_number: md?.doc_number,
          org_name: md?.org_name,
          lifecycle_status: md?.lifecycle_status,
          version_label: versionLabel,
        }
      : undefined
    float.value.citeText = el.getAttribute('data-cite-text') || ''
    positionFor(el, 4)

    // YLSW 引用自带原文摘要：直接展示，不回查 chunk 接口
    // （YLSW chunk id 与 app 本地 chunk 表不互通，回查必然 404）。
    const inlineContent = String(ref?.content || '').trim()
    if (inlineContent) {
      float.value.content = inlineContent
      float.value.error = ''
      float.value.loading = false
      return
    }

    const scope = getCacheScope()
    const cached = getCitationChunkCache(scope, chunkId)
    if (cached) {
      float.value.content = cached.content
      float.value.error = cached.error || ''
      float.value.loading = false
      return
    }

    float.value.loading = true
    float.value.error = ''
    float.value.content = ''
    try {
      const res = await fetchChunkContent(chunkId)
      const content = String(res?.data?.content || '').trim()
      if (!content) {
        const msg = t('agentStream.citation.notFound')
        setCitationChunkCache(scope, chunkId, { content: '', error: msg })
        float.value.error = msg
        return
      }
      setCitationChunkCache(scope, chunkId, { content })
      float.value.content = content
    } catch {
      const msg = t('agentStream.citation.loadFailed')
      setCitationChunkCache(scope, chunkId, { content: '', error: msg })
      float.value.error = msg
    } finally {
      float.value.loading = false
    }
  }

  const scheduleClose = () => {
    if (closeTimer) window.clearTimeout(closeTimer)
    closeTimer = window.setTimeout(() => {
      const hoveredCitation = document.querySelector('.citation-kb:hover, .citation-web:hover, .cite-num:hover')
      const hoveredPopup = document.querySelector('.chat-citation-float:hover')
      if (!hoveredCitation && !hoveredPopup) {
        float.value.visible = false
      }
    }, 120)
  }

  const cancelClose = () => {
    if (closeTimer) {
      window.clearTimeout(closeTimer)
      closeTimer = null
    }
  }

  const onMouseOver = (e: Event) => {
    const target = e.target as HTMLElement
    const kbEl = target.closest?.('.citation-kb, .cite-num') as HTMLElement | null
    const webEl = target.closest?.('.citation-web') as HTMLElement | null
    if (!kbEl && !webEl) return
    cancelClose()
    if (hoverTimer) window.clearTimeout(hoverTimer)
    hoverTimer = window.setTimeout(() => {
      if (kbEl) void openKb(kbEl)
      else if (webEl) openWeb(webEl)
    }, kbEl ? 80 : 40)
  }

  const onMouseOut = (e: Event) => {
    const rt = (e as MouseEvent).relatedTarget as HTMLElement | null
    if (rt?.closest?.('.citation-kb, .citation-web, .cite-num, .chat-citation-float')) return
    if (hoverTimer) {
      window.clearTimeout(hoverTimer)
      hoverTimer = null
    }
    scheduleClose()
  }

  const openDrawerForCitation = (payload: { url?: string; chunkId?: string }) => {
    const refs = options?.getKnowledgeReferences?.() || []
    if (!referencesDrawer || !refs.length) return false
    referencesDrawer.open({
      references: refs,
      highlight: payload,
      feedbackContext: options?.getFeedbackContext?.(),
    })
    return true
  }

  const onClick = (e: Event) => {
    const target = e.target as HTMLElement
    const webEl = target.closest?.('.citation-web') as HTMLElement | null
    if (webEl) {
      e.preventDefault()
      e.stopPropagation()
      const url = webEl.getAttribute('data-url') || ''
      if (openDrawerForCitation({ url })) return
      openWeb(webEl)
      return
    }

    const kbEl = target.closest?.('.citation-kb, .cite-num') as HTMLElement | null
    if (kbEl) {
      e.preventDefault()
      e.stopPropagation()
      const rawChunkId = kbEl.getAttribute('data-chunk-id') || ''
      const title = kbEl.getAttribute('data-doc') || ''
      const kbId = kbEl.getAttribute('data-kb-id') || ''
      const refs = options?.getKnowledgeReferences?.() || []
      // 与 openKb 一致：先按 id 精确匹配，避免数字 id 被当列表下标
      const chunkId = refs.some((r) => r?.id === rawChunkId)
        ? rawChunkId
        : resolveCitationChunkId(rawChunkId, { doc: title, kbId }, refs) || rawChunkId
      if (openDrawerForCitation({ chunkId })) return
      void openKb(kbEl)
    }
  }

  const onViewportChange = () => {
    if (float.value.visible) scheduleClose()
  }

  const bind = () => {
    const root = rootRef.value
    if (!root) return
    root.addEventListener('mouseover', onMouseOver, true)
    root.addEventListener('mouseout', onMouseOut, true)
    root.addEventListener('click', onClick, true)
    window.addEventListener('scroll', onViewportChange, true)
    window.addEventListener('resize', onViewportChange, true)
  }

  const unbind = () => {
    const root = rootRef.value
    if (root) {
      root.removeEventListener('mouseover', onMouseOver, true)
      root.removeEventListener('mouseout', onMouseOut, true)
      root.removeEventListener('click', onClick, true)
    }
    window.removeEventListener('scroll', onViewportChange, true)
    window.removeEventListener('resize', onViewportChange, true)
  }

  watch(rootRef, () => {
    unbind()
    bind()
  }, { flush: 'post' })

  onMounted(() => {
    bind()
  })

  onBeforeUnmount(() => {
    unbind()
    if (hoverTimer) window.clearTimeout(hoverTimer)
    if (closeTimer) window.clearTimeout(closeTimer)
  })

  return { float, rebind: () => { unbind(); bind() }, cancelClose, scheduleClose }
}
