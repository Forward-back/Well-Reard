<template>
  <div
    class="query-rewrite-card"
    :class="{ 'is-expanded': expanded }"
    data-testid="query-rewrite-card"
  >
    <div
      class="query-rewrite-header"
      role="button"
      tabindex="0"
      :aria-expanded="expanded"
      @click="expanded = !expanded"
      @keydown.enter="expanded = !expanded"
      @keydown.space.prevent="expanded = !expanded"
    >
      <TraceSourceIcon class="query-rewrite-icon" name="brush" :size="18" :active="true" />
      <span class="query-rewrite-title">{{ expanded ? t('chat.queryRewriteDone') : t('chat.queryRewriteCollapsedTitle') }}</span>
      <span v-if="summary && !expanded" class="query-rewrite-sep">·</span>
      <span v-if="summary && !expanded" class="query-rewrite-summary">{{ summary }}</span>
      <span class="query-rewrite-header-right">
        <span v-if="expanded" class="query-rewrite-subtitle">{{ t('chat.queryRewriteSubtitle') }}</span>
        <t-icon class="query-rewrite-chevron" :name="expanded ? 'chevron-up' : 'chevron-down'" />
      </span>
    </div>

    <div v-if="expanded" class="query-rewrite-body" @click.stop>
      <div class="query-rewrite-label">{{ t('chat.queryRewriteOriginalLabel') }}</div>
      <div class="query-rewrite-original">
        <template v-for="(part, index) in originalParts" :key="`orig-${index}`">
          <span v-if="part.hit" class="query-rewrite-ref">{{ part.text }}</span>
          <template v-else>{{ part.text }}</template>
        </template>
      </div>

      <div class="query-rewrite-divider">
        <span>{{ t('chat.queryRewriteDivider') }}</span>
      </div>

      <div class="query-rewrite-rewritten">
        <template v-for="(part, index) in rewrittenParts" :key="`new-${index}`">
          <span v-if="part.hit" class="query-rewrite-target">{{ part.text }}</span>
          <template v-else>{{ part.text }}</template>
        </template>
      </div>

      <div v-if="resolutions.length" class="query-rewrite-resolutions">
        <div v-for="(res, index) in resolutions" :key="`res-${index}`" class="query-rewrite-resolution">
          <t-icon class="query-rewrite-link-icon" name="link" size="14px" />
          <span class="query-rewrite-resolution-text">
            <b>{{ res.source_text }}</b>
            <span class="query-rewrite-arrow">→</span>
            <b>{{ res.target }}</b>
            <span v-if="res.source?.message_id" class="query-rewrite-anchor">
              · {{ sourceLabel(res) }}
            </span>
          </span>
          <button
            v-if="res.source?.message_id"
            type="button"
            class="query-rewrite-jump"
            @click="jumpToSource(res)"
          >
            {{ t('chat.queryRewriteJump') }}
            <t-icon name="arrow-up-right" size="13px" />
          </button>
        </div>
      </div>

      <div v-if="meta" class="query-rewrite-chips">
        <span
          class="query-rewrite-chip"
          :class="mustKeep.length ? 'is-success' : 'is-muted'"
        >{{ mustKeep.length ? t('chat.queryRewriteMustKeepPassed') : t('chat.queryRewriteNoMustKeep') }}</span>
        <span v-if="elapsedLabel" class="query-rewrite-chip">{{ elapsedLabel }}</span>
        <span v-if="slotLabel" class="query-rewrite-chip">{{ slotLabel }}</span>
      </div>

      <div class="query-rewrite-actions">
        <button type="button" class="query-rewrite-btn is-primary" @click="emit('edit-rewrite')">
          <t-icon name="edit" size="14px" />
          {{ t('chat.queryRewriteEdit') }}
        </button>
        <button
          type="button"
          class="query-rewrite-btn"
          :disabled="!canUseOriginal"
          @click="useOriginal"
        >
          <t-icon name="refresh" size="14px" />
          {{ t('chat.useOriginalQuery') }}
        </button>
        <span v-if="isReplying" class="query-rewrite-hint">{{ t('chat.queryRewriteBusyHint') }}</span>
        <span v-else-if="rewriteRetried" class="query-rewrite-hint">{{ t('chat.queryRewriteRetried') }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import TraceSourceIcon from './TraceSourceIcon.vue'
import type {
  QueryRewriteMeta,
  QueryRewriteResolution,
  QueryRewriteSource,
} from '@/utils/queryRewrite'

const props = withDefaults(
  defineProps<{
    original?: string
    rewritten: string
    meta?: QueryRewriteMeta | null
    isReplying?: boolean
    rewriteRetried?: boolean
  }>(),
  {
    original: '',
    meta: null,
    isReplying: false,
    rewriteRetried: false,
  },
)

const emit = defineEmits<{
  (e: 'use-original'): void
  (e: 'edit-rewrite'): void
  (e: 'jump-to-source', payload: {
    source: QueryRewriteSource
    target: string
    candidates: string[]
  }): void
}>()

const { t } = useI18n()
const expanded = ref(false)

const resolutions = computed<QueryRewriteResolution[]>(() =>
  (props.meta?.resolutions || []).filter((res) => res?.source_text && res?.target),
)

const summary = computed(() => {
  const first = resolutions.value[0]
  if (!first) return ''
  const base = `${first.source_text} → ${first.target}`
  return resolutions.value.length > 1
    ? t('chat.queryRewriteSummaryMore', { first: base, count: resolutions.value.length })
    : base
})

const mustKeep = computed(() => props.meta?.must_keep || [])
const canUseOriginal = computed(() => !props.isReplying && !props.rewriteRetried)
const elapsedLabel = computed(() => {
  const ms = props.meta?.elapsed_ms
  if (!ms || ms <= 0) return ''
  return t('chat.queryRewriteLlmTime', { time: (ms / 1000).toFixed(2) })
})
const slotLabel = computed(() => {
  const meta = props.meta
  if (!meta) return ''
  const slots = meta.slot_count || 0
  const turns = meta.recent_turns || 0
  if (!slots && !turns) return ''
  return t('chat.queryRewriteSlots', { slots, turns })
})

interface Segment {
  text: string
  hit: boolean
}

function splitBySpans(text: string, spans: string[]): Segment[] {
  if (!text) return []
  const ranges: Array<[number, number]> = []
  for (const span of spans) {
    if (!span) continue
    const index = text.indexOf(span)
    if (index < 0) continue
    ranges.push([index, index + span.length])
  }
  if (!ranges.length) return [{ text, hit: false }]
  ranges.sort((a, b) => a[0] - b[0])
  const merged: Array<[number, number]> = []
  for (const range of ranges) {
    const last = merged[merged.length - 1]
    if (last && range[0] <= last[1]) {
      last[1] = Math.max(last[1], range[1])
    } else {
      merged.push([...range])
    }
  }
  const segments: Segment[] = []
  let cursor = 0
  for (const [start, end] of merged) {
    if (start > cursor) segments.push({ text: text.slice(cursor, start), hit: false })
    segments.push({ text: text.slice(start, end), hit: true })
    cursor = end
  }
  if (cursor < text.length) segments.push({ text: text.slice(cursor), hit: false })
  return segments
}

const originalParts = computed(() =>
  splitBySpans(
    props.original || '',
    resolutions.value.map((res) => res.source_text),
  ),
)
const rewrittenParts = computed(() =>
  splitBySpans(
    props.rewritten || '',
    resolutions.value.map((res) => {
      const anchor = res.source?.text
      return anchor && props.rewritten.includes(anchor) ? anchor : res.target
    }),
  ),
)

function sourceLabel(res: QueryRewriteResolution): string {
  const source = res.source
  if (!source) return ''
  const turn = source.turn_offset || 1
  const anchor =
    turn <= 1
      ? t('chat.queryRewriteAnchorLast')
      : t('chat.queryRewriteAnchorEarlier', { turn })
  const item = source.item_index || 0
  if (item <= 0) return anchor
  return source.item_kind === 'list'
    ? t('chat.queryRewriteAnchorItem', { anchor, n: item })
    : t('chat.queryRewriteAnchorParagraph', { anchor, n: item })
}

function anchorCandidates(res: QueryRewriteResolution): string[] {
  const out: string[] = []
  const seen = new Set<string>()
  const add = (value?: string) => {
    const text = (value || '').trim()
    if (text.length < 2 || seen.has(text)) return
    seen.add(text)
    out.push(text)
  }
  const anchor = res.source?.text || ''
  const target = res.target || ''
  // 后端锚点若只是改写短语的前缀（如「2024年经济热词」），先尝试去掉
  // 「中/的」等连接词后的实体部分（「一揽子增量政策」）。
  if (anchor && target.startsWith(anchor)) {
    const rest = target
      .slice(anchor.length)
      .replace(/^(其中|关于|针对|对于|以及|[中里的内])/, '')
      .trim()
    add(rest)
  }
  add(anchor)
  const parts = target.split(/其中|关于|针对|对于|以及|[中里的内]/)
  for (let index = parts.length - 1; index >= 0; index--) add(parts[index])
  add(target)
  return out
}

function jumpToSource(res: QueryRewriteResolution) {
  if (!res.source?.message_id) return
  emit('jump-to-source', {
    source: res.source,
    target: res.source.text || res.target,
    candidates: anchorCandidates(res),
  })
}

function useOriginal() {
  if (!canUseOriginal.value) return
  emit('use-original')
}
</script>

<style scoped>
.query-rewrite-card {
  width: fit-content;
  max-width: min(720px, 100%);
  text-align: left;
}

.query-rewrite-card.is-expanded {
  width: 100%;
  max-width: 100%;
}

.query-rewrite-header {
  display: flex;
  align-items: center;
  gap: 6px;
  min-height: 28px;
  padding: 2px 0;
  cursor: pointer;
  user-select: none;
}

.query-rewrite-header:hover .query-rewrite-title {
  color: var(--td-text-color-secondary);
}

.query-rewrite-icon {
  flex-shrink: 0;
}

.query-rewrite-title {
  flex-shrink: 0;
  font-size: 13px;
  font-weight: 400;
  color: var(--td-text-color-placeholder);
  transition: color 0.15s ease;
}

.query-rewrite-sep {
  flex-shrink: 0;
  color: var(--td-text-color-placeholder);
}

.query-rewrite-summary {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 13px;
  font-weight: 500;
  color: var(--td-text-color-primary);
}

.query-rewrite-header-right {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-left: auto;
  flex-shrink: 0;
}

.query-rewrite-subtitle {
  font-size: 11px;
  color: var(--td-text-color-placeholder);
  white-space: nowrap;
}

.query-rewrite-chevron {
  font-size: 14px;
  color: var(--td-text-color-placeholder);
}

.query-rewrite-body {
  margin-top: 6px;
  padding-top: 8px;
}

.query-rewrite-label {
  margin-bottom: 4px;
  font-size: 11px;
  color: var(--td-text-color-placeholder);
}

.query-rewrite-original {
  font-size: 14px;
  line-height: 1.8;
  color: var(--td-text-color-primary);
  word-break: break-word;
}

.query-rewrite-ref {
  text-decoration: underline;
  text-decoration-color: var(--td-warning-color);
  text-decoration-thickness: 2px;
  text-underline-offset: 4px;
}

.query-rewrite-divider {
  display: flex;
  align-items: center;
  gap: 10px;
  margin: 10px 0 10px;
  font-size: 11px;
  color: var(--td-text-color-placeholder);
}

.query-rewrite-divider::after {
  content: '';
  flex: 1;
  height: 1px;
  background: var(--td-component-stroke);
}

.query-rewrite-rewritten {
  font-size: 15px;
  line-height: 1.8;
  color: var(--td-text-color-primary);
  word-break: break-word;
}

.query-rewrite-target {
  padding: 1px 4px;
  border-radius: 4px;
  background: color-mix(in srgb, var(--td-success-color) 12%, transparent);
  color: var(--td-success-color);
}

.query-rewrite-resolutions {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-top: 10px;
}

.query-rewrite-resolution {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 7px 10px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
  background: var(--td-bg-color-secondarycontainer);
  font-size: 12px;
  color: var(--td-text-color-secondary);
}

.query-rewrite-link-icon {
  flex-shrink: 0;
  color: var(--td-text-color-placeholder);
}

.query-rewrite-resolution-text {
  min-width: 0;
  line-height: 1.6;
  word-break: break-word;
}

.query-rewrite-resolution-text b {
  font-weight: 500;
  color: var(--td-text-color-primary);
}

.query-rewrite-arrow {
  margin: 0 4px;
  color: var(--td-text-color-placeholder);
}

.query-rewrite-anchor {
  color: var(--td-text-color-placeholder);
}

.query-rewrite-jump {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  flex-shrink: 0;
  margin-left: auto;
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--td-success-color);
  font-size: 12px;
  cursor: pointer;
}

.query-rewrite-jump:hover {
  text-decoration: underline;
}

.query-rewrite-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 8px;
}

.query-rewrite-chip {
  padding: 2px 6px;
  border-radius: 4px;
  background: var(--td-bg-color-secondarycontainer);
  font-size: 11px;
  line-height: 1.6;
  color: var(--td-text-color-placeholder);
}

.query-rewrite-chip.is-success {
  background: color-mix(in srgb, var(--td-success-color) 10%, transparent);
  color: var(--td-success-color);
}

.query-rewrite-chip.is-muted {
  color: var(--td-text-color-placeholder);
}

.query-rewrite-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 12px;
}

.query-rewrite-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  height: 30px;
  padding: 0 12px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 6px;
  background: var(--td-bg-color-container);
  color: var(--td-text-color-primary);
  font-size: 12px;
  cursor: pointer;
}

.query-rewrite-btn:hover:not(:disabled) {
  border-color: var(--td-success-color);
}

.query-rewrite-btn.is-primary {
  border-color: var(--td-success-color);
  color: var(--td-success-color);
}

.query-rewrite-btn:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}

.query-rewrite-hint {
  margin-left: auto;
  font-size: 11px;
  color: var(--td-text-color-placeholder);
}
</style>

<style>
mark.query-rewrite-source-flash {
  padding: 1px 2px;
  border-radius: 3px;
  background: color-mix(in srgb, var(--td-success-color) 26%, transparent);
  color: inherit;
  animation: query-rewrite-source-flash 1.2s ease;
}

@keyframes query-rewrite-source-flash {
  0%,
  100% {
    background: color-mix(in srgb, var(--td-success-color) 8%, transparent);
  }
  20% {
    background: color-mix(in srgb, var(--td-success-color) 32%, transparent);
  }
}
</style>
