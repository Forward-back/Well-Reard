<template>
  <Teleport to="body">
    <div v-if="float.visible" class="chat-citation-float" :style="{ top: `${float.top}px`, left: `${float.left}px` }"
      @mouseenter="onEnter?.()" @mouseleave="onLeave?.()">
      <template v-if="float.type === 'web'">
        <div class="chat-citation-float__title">{{ float.title || float.url }}</div>
        <a v-if="float.url" class="chat-citation-float__link" :href="float.url" target="_blank"
          rel="noopener noreferrer">{{ float.url }}</a>
      </template>
      <template v-else>
        <div class="chat-citation-float__title">{{ float.title }}</div>
        <div v-if="float.meta" class="chat-citation-float__meta">
          <span v-if="float.meta.org_name" class="citation-meta-badge">{{ float.meta.org_name }}</span>
          <span v-if="float.meta.doc_number" class="citation-meta-badge">{{ float.meta.doc_number }}</span>
          <span v-if="float.meta.lifecycle_status === 'expired'" class="citation-meta-badge citation-meta-badge--danger">{{ $t('chat.referenceExpired') }}</span>
        </div>
        <div v-if="float.loading" class="chat-citation-float__muted">{{ loadingText }}</div>
        <div v-else-if="float.error" class="chat-citation-float__error">{{ float.error }}</div>
        <div v-else class="chat-citation-float__body">
          <template v-for="(part, partIdx) in bodyParts" :key="partIdx">
            <mark v-if="part.hit" class="chat-citation-float__hit">{{ part.text }}</mark>
            <template v-else>{{ part.text }}</template>
          </template>
        </div>
      </template>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { CitationFloatState } from '@/composables/useChatCitationPopover'

const props = defineProps<{
  float: CitationFloatState
  onEnter?: () => void
  onLeave?: () => void
}>()

const { t } = useI18n()
const loadingText = t('common.loading')

type BodyPart = { text: string; hit: boolean }

// 命中句高亮：citeText 为上标对应的答案片段，在 chunk 摘要里定位后包 mark。
const bodyParts = computed<BodyPart[]>(() => {
  const content = String(props.float.content || '')
  const hit = String(props.float.citeText || '').trim()
  if (!content || !hit) return [{ text: content, hit: false }]
  const idx = content.indexOf(hit)
  if (idx < 0) return [{ text: content, hit: false }]
  const parts: BodyPart[] = []
  if (idx > 0) parts.push({ text: content.slice(0, idx), hit: false })
  parts.push({ text: hit, hit: true })
  const rest = content.slice(idx + hit.length)
  if (rest) parts.push({ text: rest, hit: false })
  return parts
})
</script>

<style lang="less">
@import './css/chat-citations.less';
</style>
