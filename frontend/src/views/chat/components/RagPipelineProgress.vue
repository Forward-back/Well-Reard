<template>
  <div v-if="visible" ref="rootElement" class="rag-pipeline-progress">
    <!-- Announcements need a region that outlives each wait row, otherwise screen
         readers miss a live region that appears together with its own text. -->
    <div class="sr-only" role="status" aria-live="polite">{{ liveStatusText }}</div>

    <!-- A turn that only recalled memory has no pipeline to draw. It borrows
         this component for the memory row alone, so anything derived from the
         agent event stream must stay out of the way — the agent timeline is
         rendering those same steps itself. -->
    <ChatMemoryStep
      v-if="memoryOnly"
      variant="root"
      :memories="memoryItems"
      :expanded="memoryExpanded"
      :forgetting-id="forgettingId"
      @toggle="toggleMemory"
      @forget="forgetMemory"
    />

    <div v-else-if="showPrePipelineWait" class="tree-children">
      <div class="tree-child tree-child-last streaming-loading-node">
        <div class="tree-branch" />
        <div class="tree-child-content">
          <div class="tool-event">
            <div class="action-card action-pending">
              <div class="action-header no-results">
                <div class="action-title">
                  <t-icon class="action-title-icon" name="lightbulb" />
                  <span class="action-name">{{ t('chat.preparingAnswer') }}</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 检索过程（YLSW stage 时间线）：答案开始输出后折叠展示，点击展开为 waterfall trace -->
    <div v-else-if="showRagTimeline" class="tree-children">
      <div class="tree-child tree-child-last">
        <div class="tree-branch" />
        <div class="tree-child-content">
          <div class="tool-event">
            <div
              class="action-card rag-trace-card"
              role="button"
              tabindex="0"
              :aria-expanded="ragTimelineExpanded"
              @click="ragTimelineExpanded = !ragTimelineExpanded"
              @keydown.enter="ragTimelineExpanded = !ragTimelineExpanded"
              @keydown.space.prevent="ragTimelineExpanded = !ragTimelineExpanded"
            >
              <div class="action-header no-results">
                <div class="action-title">
                  <TraceSourceIcon
                    class="action-title-icon"
                    name="search-doc"
                    :size="18"
                    :active="ragTimelineExpanded"
                  />
                  <span class="action-name">{{ t('chat.ragProcessTitle') }}</span>
                  <span
                    v-if="referenceSummaryText"
                    class="tree-root-reference"
                    role="button"
                    tabindex="0"
                    @click.stop="toggleReferencesDrawer"
                    @keydown.enter.stop="toggleReferencesDrawer"
                  >
                    {{ referenceSummaryText }}
                  </span>
                  <span class="rag-trace-summary">{{ traceSummary }}</span>
                  <span v-if="traceMiniSegments.length" class="rag-trace-mini" aria-hidden="true">
                    <span
                      v-for="seg in traceMiniSegments"
                      :key="seg.key"
                      class="rag-trace-mini-seg"
                      :class="`is-${seg.key}`"
                      :style="{ width: seg.width }"
                    />
                  </span>
                  <button
                    v-if="traceWarningCount"
                    type="button"
                    class="rag-trace-warn"
                    :title="t('chat.traceWarnJump')"
                    @click.stop="jumpToWarnRow"
                  >⚠️ {{ traceWarningCount }}</button>
                  <button
                    type="button"
                    class="rag-trace-copy"
                    :aria-label="tt('chat.traceCopyDiagnostics', '复制诊断信息')"
                    :title="tt('chat.traceCopyDiagnostics', '复制诊断信息')"
                    @click.stop="copyDiagnostics"
                  >
                    <t-icon name="file-copy" size="14px" />
                  </button>
                  <t-icon
                    class="rag-stage-chevron"
                    :name="ragTimelineExpanded ? 'chevron-up' : 'chevron-down'"
                  />
                </div>
              </div>
              <div v-if="ragTimelineExpanded" class="rag-trace-body">
                <template v-for="row in traceRows" :key="row.key">
                  <div v-if="row.group" class="rag-trace-group">{{ row.group }}</div>
                  <div
                    class="rag-trace-row"
                    :class="{
                      'is-warn': row.warn,
                      'is-flash': flashTraceRow === row.key,
                      'is-expandable': row.details?.length,
                      'is-expanded': expandedTraceRows.has(row.key),
                    }"
                    :role="row.details?.length ? 'button' : undefined"
                    :tabindex="row.details?.length ? 0 : undefined"
                    :aria-expanded="row.details?.length ? expandedTraceRows.has(row.key) : undefined"
                    @click.stop="toggleTraceRow(row)"
                    @keydown.enter.stop="toggleTraceRow(row)"
                    @keydown.space.prevent.stop="toggleTraceRow(row)"
                  >
                    <span class="rag-trace-name">
                      <span class="rag-trace-icon-box">
                        <t-icon
                          v-if="row.pending"
                          name="loading"
                          class="rag-trace-icon rag-trace-spin"
                          size="14px"
                        />
                        <TraceSourceIcon
                          v-else-if="row.sourceIcon"
                          class="rag-trace-icon"
                          :name="row.sourceIcon"
                          :size="14"
                        />
                        <t-icon v-else :name="row.icon" class="rag-trace-icon" size="14px" />
                      </span>
                      <span>{{ row.name }}</span>
                    </span>
                    <span class="rag-trace-data">
                      <template v-if="row.chips?.length">
                        <span
                          v-for="chip in row.chips"
                          :key="chip.label"
                          class="rag-trace-chip"
                          :class="`is-${chip.tone || 'plain'}`"
                        >{{ chip.text }}</span>
                      </template>
                      <span v-else class="rag-trace-text">{{ row.text }}</span>
                      <span v-if="row.progress !== undefined" class="rag-trace-usage">
                        <span
                          class="rag-trace-usage-fill"
                          :class="{ 'is-hot': row.progress > 0.85 }"
                          :style="{ width: `${Math.max(4, Math.round(row.progress * 100))}%` }"
                        />
                      </span>
                    </span>
                    <span class="rag-trace-dur">{{ row.durationLabel }}</span>
                    <span class="rag-trace-bar">
                      <span
                        class="rag-trace-bar-fill"
                        :class="{ 'is-max': row.isMax }"
                        :style="{ width: row.barWidth }"
                      />
                    </span>
                  </div>
                  <div
                    v-if="row.details?.length && expandedTraceRows.has(row.key)"
                    class="rag-trace-detail"
                  >
                    <template v-for="(item, itemIdx) in row.details" :key="itemIdx">
                      <div v-if="item.kind === 'kv'" class="rag-trace-detail-kv">
                        <span class="k">{{ item.label }}</span>
                        <span class="v">{{ item.value }}</span>
                      </div>
                      <div v-else class="rag-trace-detail-top">
                        <span class="rank">{{ item.rank }}</span>
                        <span class="score">
                          <span class="score-num">{{ item.scoreLabel }}</span>
                          <span class="score-bar">
                            <span class="score-bar-fill" :style="{ width: `${item.scorePct}%` }" />
                          </span>
                        </span>
                        <span class="snippet">{{ item.text }}</span>
                        <span class="doc">
                          <span class="doc-title">{{ item.title }}</span>
                          <span v-if="item.sub" class="doc-sub">{{ item.sub }}</span>
                        </span>
                      </div>
                    </template>
                  </div>
                </template>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div v-else-if="!showCollapsedRoot" class="tree-children">
      <ChatMemoryStep
        v-if="hasMemory"
        :memories="memoryItems"
        :expanded="memoryExpanded"
        :is-last="memoryIsLast"
        :forgetting-id="forgettingId"
        @toggle="toggleMemory"
        @forget="forgetMemory"
      />

      <div v-for="(step, index) in steps" :key="step.id" class="tree-child" :class="{
        'tree-child-last':
          !showDoneRow
          && !showWaitStep
          && !showThinkingStep
          && index === steps.length - 1,
      }">
        <div class="tree-branch" />
        <div class="tree-child-content">
          <div class="tool-event">
            <div
              class="action-card"
              :class="{ 'has-reference-trigger': step.canOpenReferences }"
              :role="step.canOpenReferences ? 'button' : undefined"
              :tabindex="step.canOpenReferences ? 0 : undefined"
              @click="handleStepClick(step)"
              @keydown.enter="handleStepClick(step)"
              @keydown.space.prevent="handleStepClick(step)"
            >
              <div
                class="action-header"
                :class="{ 'no-results': !step.canOpenReferences }"
              >
                <div class="action-title">
                  <t-icon class="action-title-icon" :name="step.iconName" />
                  <span class="action-name" :class="{ 'is-running': step.pending }">{{ step.title }}</span>
                </div>
              </div>
              <div v-if="step.summaryHtml" class="search-results-summary-fixed">
                <div class="results-summary-text" v-html="step.summaryHtml" />
              </div>
            </div>
          </div>
        </div>
      </div>

      <div
        v-if="showWaitStep"
        class="tree-child tree-child-last streaming-loading-node rag-model-wait-step"
      >
        <div class="tree-branch" />
        <div class="tree-child-content">
          <div class="tool-event">
            <div class="action-card" :class="{ 'action-pending': !waitStepStalled }">
              <div class="action-header no-results">
                <div class="action-title">
                  <t-icon class="action-title-icon" name="lightbulb" />
                  <span class="action-name">{{ waitStepText }}</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div v-if="showThinkingStep" class="tree-child rag-thinking-step"
        :class="{ 'tree-child-last': !showDoneRow }">
        <div class="tree-branch" />
        <div class="tree-child-content">
          <div class="tool-event">
            <div class="action-card" :class="{ 'action-pending': thinkingPending }">
              <div class="action-header" :class="{ 'no-results': !thinkingContent }" @click="toggleThinking">
                <div class="action-title">
                  <t-icon class="action-title-icon" name="lightbulb" />
                  <span class="action-name">{{ t('agent.think') }}</span>
                </div>
              </div>
              <div v-if="thinkingContent && thinkingExpanded" class="thinking-detail-content">
                {{ thinkingContent }}
              </div>
            </div>
          </div>
        </div>
      </div>

      <div v-if="showDoneRow" class="tree-child agent-step-done tree-child-last">
        <div class="tree-branch" />
        <div class="tree-child-content">
          <div class="tool-event">
            <div class="action-card">
              <div class="action-header no-results">
                <div class="action-title">
                  <t-icon class="action-title-icon" name="check-circle" />
                  <span class="action-name">{{ t('common.finish') }}</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div v-else class="tree-container">
      <div class="tool-event">
        <div class="action-card tree-root">
          <div class="tree-root-toolbar">
            <button
              type="button"
              class="tree-root-expand"
              :aria-expanded="showExpandedTimeline"
              :aria-label="collapsedStatusText"
              @click="toggleExpanded"
            >
              <span class="tree-root-status">{{ collapsedStatusText }}</span>
              <span
                v-if="referenceSummaryText"
                class="tree-root-reference"
              >
                {{ referenceSummaryText }}
              </span>
              <t-icon
                class="tree-root-expand__icon"
                :name="showExpandedTimeline ? 'chevron-down' : 'chevron-right'"
              />
            </button>
          </div>
        </div>
      </div>

      <div v-if="showExpandedTimeline" class="tree-children tree-children-expanded">
        <ChatMemoryStep
          v-if="hasMemory"
          :memories="memoryItems"
          :expanded="memoryExpanded"
          :is-last="memoryIsLast"
          :forgetting-id="forgettingId"
          @toggle="toggleMemory"
          @forget="forgetMemory"
        />

        <div v-for="(step, index) in steps" :key="step.id" class="tree-child"
          :class="{ 'tree-child-last': index === steps.length - 1 && !showDoneRow && !showThinkingStep }">
          <div class="tree-branch" />
          <div class="tree-child-content">
            <div class="tool-event">
              <div
                class="action-card"
                :class="{ 'has-reference-trigger': step.canOpenReferences }"
                :role="step.canOpenReferences ? 'button' : undefined"
                :tabindex="step.canOpenReferences ? 0 : undefined"
                @click="handleStepClick(step)"
                @keydown.enter="handleStepClick(step)"
                @keydown.space.prevent="handleStepClick(step)"
              >
                <div
                  class="action-header"
                  :class="{ 'no-results': !step.canOpenReferences }"
                >
                  <div class="action-title">
                    <t-icon class="action-title-icon" :name="step.iconName" />
                    <span class="action-name" :class="{ 'is-running': step.pending }">{{ step.title }}</span>
                  </div>
                </div>
                <div v-if="step.summaryHtml" class="search-results-summary-fixed">
                  <div class="results-summary-text" v-html="step.summaryHtml" />
                </div>
              </div>
            </div>
          </div>
        </div>

        <div v-if="showThinkingStep" class="tree-child rag-thinking-step" :class="{ 'tree-child-last': !showDoneRow }">
          <div class="tree-branch" />
          <div class="tree-child-content">
            <div class="tool-event">
              <div class="action-card" :class="{ 'action-pending': thinkingPending }">
                <div class="action-header" :class="{ 'no-results': !thinkingContent }" @click="toggleThinking">
                  <div class="action-title">
                    <t-icon class="action-title-icon" name="lightbulb" />
                    <span class="action-name">{{ t('agent.think') }}</span>
                  </div>
                </div>
                <div v-if="thinkingContent && thinkingExpanded" class="thinking-detail-content">
                  {{ thinkingContent }}
                </div>
              </div>
            </div>
          </div>
        </div>

        <div v-if="showDoneRow" class="tree-child agent-step-done tree-child-last">
          <div class="tree-branch" />
          <div class="tree-child-content">
            <div class="tool-event">
              <div class="action-card">
                <div class="action-header no-results">
                  <div class="action-title">
                    <t-icon class="action-title-icon" name="check-circle" />
                    <span class="action-name">{{ t('common.finish') }}</span>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import ChatMemoryStep from './ChatMemoryStep.vue'
import TraceSourceIcon from './TraceSourceIcon.vue'
import type { TraceSourceIconName } from './traceSourceIcons'
import { copyWithToast } from '@/utils/clipboard'
import { useChatMemoryRow } from '@/composables/useChatMemoryRow'
import { getAgentToolIconName } from '@/utils/agent-tool-icons'
import {
  getKnowledgeSearchSummaryHtml,
  getRagPipelineStepTitle,
  getRetrievalSearchSource,
} from '@/utils/agent-tool-display'
import { getAttachmentParsingSummaryHtml } from '@/utils/attachmentParsingDisplay'
import { RAG_RETRIEVAL_TOOL_NAMES, RAG_TIMELINE_TOOL_NAMES } from '@/utils/rag-pipeline-history'
import { useChatReferencesDrawer } from '@/composables/useChatReferencesDrawer'
import { buildReferenceSections } from '@/utils/referenceSources'
import {
  createRagWaitController,
  getRagPipelineWaitKind,
  type RagWaitView,
} from '@/utils/rag-pipeline-state'

const props = defineProps<{
  userQuery?: string
  session?: {
    id?: string | number
    agentEventStream?: Array<Record<string, unknown>>
    content?: string
    knowledge_references?: Array<{ chunk_type?: string; knowledge_id?: string; knowledge_title?: string }>
    used_memories?: Array<{ id: string; kind: string; content: string }>
    is_completed?: boolean
    /** YLSW 检索阶段状态（stage SSE 事件，流式期等待行如实展示） */
    ragStage?: { stage?: string; counts?: Record<string, number> }
    /** 本轮 stage 时间线（答案开始后折叠为「检索过程」） */
    ragStageTimeline?: Array<{
      stage: string
      counts?: Record<string, number>
      detail?: Record<string, unknown>
      ts?: number
      t_ms?: number
    }>
  }
  // Set by hosts that already render their own timeline (the agent stream) and
  // only need the memory row from here.
  memoryOnly?: boolean
}>()

const { t } = useI18n()

// i18n 缺失兜底：任何标签先过 tt()，避免 undefined 上屏
// （键缺失时 vue-i18n 在不同版本/热更新态下可能返回空串或 undefined）
const tt = (key: string, fallback: string, params?: Record<string, string | number>) => {
  const msg = params ? (t as (k: string, p: Record<string, string | number>) => unknown)(key, params) : t(key)
  return typeof msg === 'string' && msg && !msg.startsWith('chat.') ? msg : fallback
}
const referencesDrawer = useChatReferencesDrawer()
const userExpanded = ref(false)
const thinkingExpanded = ref(true)
const rootElement = ref<HTMLElement | null>(null)
const waitView = ref<RagWaitView>({ kind: 'none', stalled: false })
const waitController = createRagWaitController((view) => {
  waitView.value = view
})

// Long-term memory is shown as a timeline row rather than as a card of its
// own: it is one more thing the turn did before answering, and giving it a
// separate visual language would make it read as unrelated to the pipeline.
const {
  memoryItems,
  hasMemory,
  expanded: memoryExpanded,
  forgettingId,
  toggle: toggleMemory,
  forget: forgetMemory,
} = useChatMemoryRow(() => props.session?.used_memories)

const thinkingContent = computed(() => {
  const stream = props.session?.agentEventStream
  if (!Array.isArray(stream)) return ''
  return stream
    .filter((event) => event.type === 'thinking')
    .map((event) => String(event.content || ''))
    .join('')
})

const hasThinking = computed(() => thinkingContent.value.trim().length > 0)

const hasThinkingEvent = computed(() => {
  const stream = props.session?.agentEventStream
  if (!Array.isArray(stream)) return false
  return stream.some((event) => event.type === 'thinking')
})

const hasAnswer = computed(() => {
  const sessionContent = props.session?.content
  if (typeof sessionContent === 'string' && sessionContent.trim().length > 0) return true

  const stream = props.session?.agentEventStream
  if (!stream?.length) return false
  return stream.some((event) => {
    if (event.type !== 'answer' || event.superseded) return false
    const content = event.content
    return typeof content === 'string' && content.trim().length > 0
  })
})

const hasReferences = computed(
  () => (props.session?.knowledge_references?.length ?? 0) > 0,
)

const referenceSections = computed(() => buildReferenceSections(props.session?.knowledge_references))

const steps = computed(() => {
  const stream = props.session?.agentEventStream
  if (!stream?.length) return []

  return stream
    .filter((event) => {
      return (
        event.type === 'tool_call' &&
        typeof event.tool_name === 'string' &&
        RAG_TIMELINE_TOOL_NAMES.has(event.tool_name)
      )
    })
    .map((event) => {
      const toolName = String(event.tool_name)
      const pending = event.pending === true
      const toolData =
        event.tool_data && typeof event.tool_data === 'object'
          ? (event.tool_data as Record<string, unknown>)
          : null

      const isSearchTool = RAG_RETRIEVAL_TOOL_NAMES.has(toolName)
      const isAttachmentTool = toolName === 'attachment_parsing' || toolName === 'image_analysis'
      const searchSource = isSearchTool
        ? getRetrievalSearchSource(event.arguments, toolData)
        : undefined
      let summaryHtml = ''
      if (!pending && isSearchTool && toolData) {
        summaryHtml = getKnowledgeSearchSummaryHtml(t, toolData)
      } else if (!pending && isAttachmentTool) {
        summaryHtml = getAttachmentParsingSummaryHtml(t, event)
      }
      const canOpenReferences = !pending && isSearchTool && hasReferences.value

      return {
        id: String(event.tool_call_id || `${toolName}-${event.timestamp || 0}`),
        toolName,
        pending,
        iconName: getAgentToolIconName(toolName, searchSource),
        title: getRagPipelineStepTitle(t, {
          tool_name: toolName,
          pending,
          success: event.success as boolean | undefined,
          arguments: event.arguments,
          tool_data: toolData,
        }),
        summaryHtml,
        canOpenReferences,
      }
    })
})

const allStepsDone = computed(
  () => steps.value.length > 0 && steps.value.every((step) => !step.pending),
)

const hasCompletedRetrievalStep = computed(() => steps.value.some(
  (step) => RAG_RETRIEVAL_TOOL_NAMES.has(step.toolName) && !step.pending,
))

const waitKind = computed(() => getRagPipelineWaitKind({
  isCompleted: Boolean(props.session?.is_completed),
  hasAnswer: hasAnswer.value,
  hasThinkingEvent: hasThinkingEvent.value,
  stepCount: steps.value.length,
  allStepsDone: allStepsDone.value,
  hasCompletedRetrievalStep: hasCompletedRetrievalStep.value,
}))

const showWaitStep = computed(() => waitView.value.kind !== 'none')

const waitStepStalled = computed(() => waitView.value.stalled)

/** 召回明细：四路全列出（含 0 值 + SQL 命中/未命中），日志要如实 */
const retrievalRouteDetail = (counts: Record<string, number>) => {
  const routes = [
    `faiss ${tt('chat.stageRouteCount', '{count} 条', { count: counts.faiss || 0 })}`,
    `bm25 ${tt('chat.stageRouteCount', '{count} 条', { count: counts.bm25 || 0 })}`,
    tt('chat.stageRouteMeta', '元数据 {count} 条', { count: counts.meta || 0 }),
  ]
  routes.push(counts.sql ? tt('chat.stageRouteSqlHit', 'SQL 命中') : tt('chat.stageRouteSqlMiss', 'SQL 未命中'))
  return routes.join(' / ')
}

/** 单条 stage → 日志式文案（完成态；进行中的措辞由等待行单独处理） */
const stageTextFor = (stage: string, counts: Record<string, number>) => {
  switch (stage) {
    case 'retrieval_start':
      return tt('chat.stageRetrievalStarted', '开始检索')
    case 'retrieval_done':
      return tt('chat.stageRetrievalDoneAll', '召回完成：{detail}', { detail: retrievalRouteDetail(counts) })
    case 'rrf_done':
      return tt('chat.stageRrfDone', '融合完成，{count} 条候选', { count: counts.merged || 0 })
    case 'rerank_start':
      return tt('chat.stageRerankStarted', '开始重排')
    case 'rerank_done': {
      const n = counts.reranked || 0
      return n > 0
        ? tt('chat.stageRerankedCount', '重排完成，{count} 条入选', { count: n })
        : tt('chat.stageReranked', '重排完成')
    }
    case 'context_done':
      return tt('chat.stageContextDone', '上下文组装，{contexts} 段 / {chars} 字', {
        contexts: counts.contexts || 0,
        chars: counts.chars || 0,
      })
    case 'generate_start':
      return tt('chat.stageGenerateStarted', '开始生成回答')
    case 'generate_done':
      return tt('chat.stageGenerateDone', '生成完成，{count} 字', { count: counts.chars || 0 })
    default:
      return ''
  }
}

/** 进行中的等待行文案（仅 *_start / rerank 等待阶段） */
const stageLiveTextFor = (stage: string, counts: Record<string, number>) => {
  switch (stage) {
    case 'retrieval_start':
      return tt('chat.stageRetrieving', '正在检索…')
    case 'rerank_start':
      return tt('chat.stageReranking', '重排中…')
    case 'generate_start':
      return tt('chat.stageGenerating', '生成中…')
    default:
      return stageTextFor(stage, counts)
  }
}

const ragTimeline = computed(() =>
  (props.session?.ragStageTimeline || []).filter((entry) => entry?.stage),
)

const ragTimelineExpanded = ref(false)

const showRagTimeline = computed(
  () =>
    ragTimeline.value.length > 0 &&
    (hasAnswer.value || Boolean(props.session?.is_completed)) &&
    steps.value.length === 0 &&
    !hasThinking.value,
)

type TraceChip = { label: string; text: string; tone?: 'warn' | 'muted' | 'ok' | undefined }
type TraceDetailItem =
  | { kind: 'kv'; label: string; value: string }
  | {
      kind: 'top'
      rank: number
      scoreLabel: string
      scorePct: number
      text: string
      title: string
      sub: string
    }

type TraceRow = {
  key: string
  group?: string
  icon: string
  sourceIcon?: TraceSourceIconName
  name: string
  text?: string
  chips?: TraceChip[]
  details?: TraceDetailItem[]
  /** 0~1，主行内联用量条（组装上下文占用） */
  progress?: number
  durationMs?: number
  durationLabel: string
  barWidth: string
  isMax?: boolean
  warn?: boolean
  pending?: boolean
}

const formatCount = (n: number) => n.toLocaleString('en-US')

/** 毫秒 → 秒（两位小数；不足 10ms 统一显示 +0ms 表示可忽略） */
const formatDuration = (ms?: number) => {
  if (typeof ms !== 'number') return '…'
  if (ms < 10) return '+0ms'
  return `${(ms / 1000).toFixed(2)}s`
}

/** 文件名 → 标题/副标题（机构 · 日期），raw 文件名不上屏 */
const parseDocName = (raw: string) => {
  const base = raw.replace(/\.[a-z0-9]{1,6}$/i, '')
  if (!base.includes('_')) return { title: base, sub: '' }
  const parts = base.split('_').filter(Boolean)
  const datePart = parts[1] && /^\d{8}$/.test(parts[1])
    ? `${parts[1].slice(0, 4)}-${parts[1].slice(4, 6)}-${parts[1].slice(6, 8)}`
    : ''
  const vIdx = parts.findIndex((part) => /^V\d/i.test(part))
  const end = vIdx > 4 ? vIdx : Math.max(4, parts.length - 2)
  const title = parts.slice(4, end).join('_') || base
  const org = parts[2] || ''
  return { title, sub: [org, datePart].filter(Boolean).join(' · ') }
}

// waterfall trace 行：start/done 合并为 span，耗时画比例条，数据用 chips/键值/证据条
const traceRows = computed<TraceRow[]>(() => {
  const timeline = ragTimeline.value
  const at = (stage: string, fromIdx = 0) => {
    for (let i = fromIdx; i < timeline.length; i++) {
      if (timeline[i].stage === stage) return i
    }
    return -1
  }
  const entry = (idx: number) => (idx >= 0 ? timeline[idx] : null)
  const tms = (idx: number) => entry(idx)?.t_ms
  const counts = (idx: number) => entry(idx)?.counts || {}
  const detail = (idx: number) => ((entry(idx)?.detail || {}) as Record<string, unknown>)
  const asStr = (v: unknown) => (typeof v === 'string' ? v : '')
  const asNum = (v: unknown) => (typeof v === 'number' ? v : 0)

  const lastT = tms(timeline.length - 1) || 0
  const iRetrievalStart = at('retrieval_start')
  const iRetrievalDone = at('retrieval_done')
  const iRrf = at('rrf_done')
  const iRerankStart = at('rerank_start')
  const iRerankDone = at('rerank_done')
  const iContext = at('context_done')
  const iGenStart = at('generate_start')
  const iGenDone = at('generate_done')
  const completed = Boolean(props.session?.is_completed)

  const retrievalCounts = counts(iRetrievalDone)
  const mergedCount = counts(iRrf).merged || 0
  const rerankedCount = counts(iRerankDone).reranked || 0
  const ctxCounts = counts(iContext)
  const genCounts = counts(iGenDone)

  const rows: TraceRow[] = []

  // 召回（span）
  if (iRetrievalStart >= 0) {
    const done = iRetrievalDone >= 0
    const pending = !done && !completed
    const startT = tms(iRetrievalStart) || 0
    const endT = done ? (tms(iRetrievalDone) || lastT) : lastT
    const faiss = retrievalCounts.faiss || 0
    const meta = retrievalCounts.meta || 0
    const chips: TraceChip[] = done
      ? [
          { label: 'faiss', text: tt('chat.traceChipFaiss', 'faiss {count}', { count: faiss }), tone: faiss === 0 ? 'warn' : undefined },
          { label: 'bm25', text: tt('chat.traceChipBm25', 'bm25 {count}', { count: retrievalCounts.bm25 || 0 }) },
          { label: 'meta', text: tt('chat.traceChipMeta', 'meta {count}', { count: meta }), tone: meta === 0 ? 'muted' : undefined },
          {
            label: 'sql',
            text: retrievalCounts.sql
              ? tt('chat.stageRouteSqlHit', 'SQL 命中')
              : tt('chat.stageRouteSqlMiss', 'SQL 未命中'),
            tone: retrievalCounts.sql ? 'ok' : 'muted',
          },
        ]
      : []
    rows.push({
      key: 'recall',
      group: tt('chat.traceGroupRetrieval', '检索'),
      icon: 'search',
      name: tt('chat.traceRecall', '召回'),
      chips,
      durationMs: endT - startT,
      durationLabel: pending ? '…' : formatDuration(endT - startT),
      barWidth: '0%',
      warn: done && faiss === 0,
      pending,
    })
  }

  // 融合（瞬时点）
  if (iRrf >= 0) {
    const base = iRetrievalDone >= 0 ? tms(iRetrievalDone) || 0 : tms(iRrf) || 0
    const strategy = asStr(detail(iRrf).strategy)
    const details: TraceDetailItem[] = strategy
      ? [{ kind: 'kv', label: tt('chat.traceDetailStrategy', '策略'), value: strategy }]
      : []
    rows.push({
      key: 'fusion',
      icon: 'swap',
      name: tt('chat.traceFusion', '融合'),
      text: tt('chat.traceFusionText', 'RRF → {count} 条候选', { count: mergedCount }),
      details: details.length ? details : undefined,
      durationMs: (tms(iRrf) || 0) - base,
      durationLabel: formatDuration((tms(iRrf) || 0) - base),
      barWidth: '2px',
    })
  }

  // 重排（span）：漏斗数字 + 后端/阈值 + Top 证据条
  if (iRerankStart >= 0) {
    const done = iRerankDone >= 0
    const pending = !done && !completed
    const startT = tms(iRerankStart) || 0
    const endT = done ? (tms(iRerankDone) || lastT) : lastT
    const rerankDetail = detail(iRerankDone)
    const rerankDetails: TraceDetailItem[] = []
    if (done) {
      const backend = asStr(rerankDetail.backend)
      const threshold = asNum(rerankDetail.threshold)
      if (backend) rerankDetails.push({ kind: 'kv', label: tt('chat.traceDetailBackend', '后端'), value: backend })
      if (threshold) rerankDetails.push({ kind: 'kv', label: tt('chat.traceDetailThreshold', '阈值'), value: threshold.toFixed(2) })
      if (mergedCount || rerankedCount) {
        rerankDetails.push({
          kind: 'kv',
          label: tt('chat.traceDetailFunnel', '漏斗'),
          value: `${formatCount(mergedCount)} → ${formatCount(rerankedCount)}`,
        })
      }
      const tops = Array.isArray(rerankDetail.top)
        ? (rerankDetail.top as Array<Record<string, unknown>>)
        : []
      const maxScore = Math.max(1e-6, ...tops.map((cand) => asNum(cand.score)))
      tops.forEach((cand, candIdx) => {
        const score = asNum(cand.score)
        const doc = parseDocName(asStr(cand.doc))
        rerankDetails.push({
          kind: 'top',
          rank: candIdx + 1,
          scoreLabel: score.toFixed(2),
          scorePct: Math.round((score / maxScore) * 100),
          text: asStr(cand.text),
          title: doc.title,
          sub: doc.sub,
        })
      })
    }
    rows.push({
      key: 'rerank',
      group: tt('chat.traceGroupRerank', '重排'),
      icon: 'filter',
      name: tt('chat.traceRerank', '重排'),
      text: done
        ? tt('chat.traceRerankText', '{from} → {to} 条入选', {
            from: formatCount(mergedCount),
            to: formatCount(rerankedCount),
          })
        : tt('chat.stageReranking', '重排中…'),
      details: rerankDetails.length ? rerankDetails : undefined,
      durationMs: endT - startT,
      durationLabel: pending ? '…' : formatDuration(endT - startT),
      barWidth: '0%',
      pending,
    })
  }

  // 组装（瞬时点）：实际 / 上限 + 用量条
  if (iContext >= 0) {
    const base = iRerankDone >= 0 ? tms(iRerankDone) || 0 : tms(iContext) || 0
    const maxChars = asNum(detail(iContext).max_chars)
    const chars = ctxCounts.chars || 0
    const assembleDetails: TraceDetailItem[] = maxChars
      ? [{ kind: 'kv', label: tt('chat.traceDetailMaxChars', '上下文中限'), value: `${formatCount(maxChars)} 字` }]
      : []
    rows.push({
      key: 'assemble',
      group: tt('chat.traceGroupAssemble', '组装 & 生成'),
      icon: 'chart-bubble',
      sourceIcon: 'assemble',
      name: tt('chat.traceAssemble', '组装'),
      text: maxChars
        ? tt('chat.traceAssembleUsage', '{contexts} 段 · {chars} / {max} 字', {
            contexts: ctxCounts.contexts || 0,
            chars: formatCount(chars),
            max: formatCount(maxChars),
          })
        : tt('chat.traceAssembleText', '{contexts} 段 · {chars} 字', {
            contexts: ctxCounts.contexts || 0,
            chars: formatCount(chars),
          }),
      details: assembleDetails.length ? assembleDetails : undefined,
      progress: maxChars ? Math.min(1, chars / maxChars) : undefined,
      durationMs: (tms(iContext) || 0) - base,
      durationLabel: formatDuration((tms(iContext) || 0) - base),
      barWidth: '2px',
    })
  }

  // 生成（span，附 TTFT / 速率）
  if (iGenStart >= 0) {
    const done = iGenDone >= 0
    const pending = !done && !completed
    const startT = tms(iGenStart) || 0
    const endT = done ? (tms(iGenDone) || lastT) : lastT
    const durMs = Math.max(1, endT - startT)
    const chars = genCounts.chars || 0
    const rate = done && chars ? ((chars / durMs) * 1000).toFixed(1) : ''
    const genDetail = detail(iGenDone)
    const genDetails: TraceDetailItem[] = []
    if (done) {
      const model = asStr(genDetail.model)
      const ttft = asNum(genDetail.ttft_ms)
      if (model) genDetails.push({ kind: 'kv', label: tt('chat.traceDetailModel', '模型'), value: model })
      if (ttft) genDetails.push({ kind: 'kv', label: tt('chat.traceDetailTtft', '首 token'), value: `${(ttft / 1000).toFixed(2)}s` })
      if (rate) genDetails.push({ kind: 'kv', label: tt('chat.traceDetailRate', '速率'), value: `${rate} 字/s` })
    }
    rows.push({
      key: 'generate',
      icon: 'chart-line',
      sourceIcon: 'done',
      name: tt('chat.traceGenerate', '生成'),
      text: done
        ? rate
          ? tt('chat.traceGenerateText', '{chars} 字 · {rate} 字/s', { chars: formatCount(chars), rate })
          : tt('chat.stageGenerateDone', '生成完成，{count} 字', { count: chars })
        : tt('chat.stageGenerating', '生成中…'),
      details: genDetails.length ? genDetails : undefined,
      durationMs: endT - startT,
      durationLabel: pending ? '…' : formatDuration(endT - startT),
      barWidth: '0%',
      pending,
    })
  }

  // 耗时条按最长 span 归一化
  const maxDur = Math.max(1, ...rows.map((row) => row.durationMs || 0))
  for (const row of rows) {
    if (row.barWidth === '0%') {
      const dur = Math.max(0, row.durationMs || 0)
      row.barWidth = `${Math.max(2, Math.round((dur / maxDur) * 100))}%`
      row.isMax = dur === maxDur && maxDur > 1
    }
  }
  return rows
})

const traceWarningCount = computed(() => traceRows.value.filter((row) => row.warn).length)

// dev 哨兵：任何 trace 文案含 'undefined' 立即 console.error，防字段契约失配静默上屏
watch(
  traceRows,
  (rows) => {
    for (const row of rows) {
      const texts = [
        row.name,
        row.text,
        ...(row.chips || []).map((chip) => chip.text),
        ...(row.details || []).flatMap((item) =>
          item.kind === 'kv'
            ? [item.label, item.value]
            : [item.scoreLabel, item.text, item.title, item.sub],
        ),
      ]
      if (texts.some((value) => typeof value === 'string' && value.includes('undefined'))) {
        console.error('[rag-trace] undefined 字段上屏:', row)
      }
    }
  },
  { immediate: true },
)

const expandedTraceRows = reactive(new Set<string>())
const flashTraceRow = ref('')

const toggleTraceRow = (row: TraceRow) => {
  if (!row.details?.length) return
  if (expandedTraceRows.has(row.key)) expandedTraceRows.delete(row.key)
  else expandedTraceRows.add(row.key)
}

/** 头部 ⚠️：展开并高亮异常行 */
const jumpToWarnRow = () => {
  const warnRow = traceRows.value.find((row) => row.warn)
  if (!warnRow) return
  ragTimelineExpanded.value = true
  expandedTraceRows.add(warnRow.key)
  flashTraceRow.value = warnRow.key
  window.setTimeout(() => {
    if (flashTraceRow.value === warnRow.key) flashTraceRow.value = ''
  }, 1600)
}

const copyDiagnostics = () => {
  const payload = {
    summary: traceSummary.value,
    stages: ragTimeline.value,
  }
  void copyWithToast(JSON.stringify(payload, null, 2), 'common.copied')
}

// 头部 mini 分布条：召回/重排/生成 耗时占比
const traceMiniSegments = computed(() => {
  const wanted = ['recall', 'rerank', 'generate']
  const rows = traceRows.value.filter(
    (row) => wanted.includes(row.key) && (row.durationMs || 0) > 0,
  )
  const total = rows.reduce((sum, row) => sum + (row.durationMs || 0), 0)
  if (!total) return []
  return rows.map((row) => ({
    key: row.key,
    width: `${(((row.durationMs || 0) / total) * 100).toFixed(1)}%`,
  }))
})

const traceSummary = computed(() => {
  const timeline = ragTimeline.value
  if (!timeline.length) return ''
  const total = timeline[timeline.length - 1].t_ms || 0
  const byStage = (stage: string) => timeline.find((e) => e.stage === stage)
  const merged = byStage('rrf_done')?.counts?.merged || 0
  const reranked = byStage('rerank_done')?.counts?.reranked || 0
  const totalLabel = `${(total / 1000).toFixed(2)}s`
  if (merged || reranked) {
    return tt('chat.traceSummary', '总耗时 {total} · 召回{from} → 入选{to}', {
      total: totalLabel,
      from: formatCount(merged),
      to: formatCount(reranked),
    })
  }
  return totalLabel
})

// YLSW stage 事件 → 等待行文案（流式期如实展示检索/重排/生成阶段）
const ragStageText = computed(() => {
  const latest = ragTimeline.value[ragTimeline.value.length - 1]
  const stage = latest?.stage || props.session?.ragStage?.stage || ''
  const counts = latest?.counts || props.session?.ragStage?.counts || {}
  return stage ? stageLiveTextFor(stage, counts) : ''
})

const waitStepText = computed(() => {
  if (waitView.value.stalled) return t('chat.modelStillResponding')
  if (waitView.value.kind === 'model') return t('chat.connectingModelAndGeneratingAnswer')
  return ragStageText.value || t('chat.preparingAnswer')
})

const showCollapsedRoot = computed(
  () =>
    (hasAnswer.value || Boolean(props.session?.is_completed)) &&
    (steps.value.length > 0 || hasThinking.value),
)

const showExpandedTimeline = computed(() => {
  if (!showCollapsedRoot.value) return true
  return userExpanded.value
})

const showDoneRow = computed(() => {
  const turnDone = hasAnswer.value || Boolean(props.session?.is_completed)
  if (!turnDone) return false
  // A timeline rendered only because memory was used has nothing to report as
  // finished; adding a "done" row there would put a step into plain chat that
  // never had one.
  if (steps.value.length === 0 && !hasThinking.value) return false
  if (steps.value.length > 0 && !allStepsDone.value) return false
  return true
})

const showPrePipelineWait = computed(() => {
  if (hasAnswer.value || props.session?.is_completed || steps.value.length > 0 || hasThinking.value) {
    return false
  }
  // Memory is recalled before the first token, so once it is on screen the
  // turn is visibly under way and the placeholder would be redundant.
  return !hasMemory.value
})

// Only show the thinking row once the backend actually streams thinking events.
// Do not pre-empt during the model phase — that flashes "思考" even when thinking is disabled.
const showThinkingStep = computed(() => hasThinkingEvent.value)

const thinkingPending = computed(
  () =>
    showThinkingStep.value &&
    !hasThinking.value &&
    !hasAnswer.value &&
    !props.session?.is_completed,
)

const isThinkingStreaming = computed(
  () =>
    showThinkingStep.value &&
    thinkingExpanded.value &&
    !hasAnswer.value &&
    !props.session?.is_completed,
)

// Memory alone is enough to render: a plain-chat answer that used memory still
// needs somewhere to say so.
const visible = computed(() => {
  if (props.memoryOnly) return hasMemory.value
  return (
    steps.value.length > 0 ||
    showPrePipelineWait.value ||
    showRagTimeline.value ||
    showThinkingStep.value ||
    hasMemory.value
  )
})

// The memory row leads the timeline, so it is only the last node when nothing
// else rendered.
const memoryIsLast = computed(
  () => steps.value.length === 0 && !showWaitStep.value && !showThinkingStep.value && !showDoneRow.value,
)

const liveStatusText = computed(() => {
  if (showPrePipelineWait.value) return ragStageText.value || t('chat.preparingAnswer')
  if (showWaitStep.value) return waitStepText.value
  return ''
})

const collapsedStatusText = computed(() => {
  if (steps.value.length === 0) {
    return hasThinking.value ? t('agentStream.toolStatus.thinkingDone') : ''
  }
  return t('agentStream.ragPipeline.searchDone')
})

const referenceSummaryText = computed(() => {
  const docCount = referenceSections.value.find((section) => section.id === 'documents')?.items.length ?? 0
  const webCount = referenceSections.value.find((section) => section.id === 'web')?.items.length ?? 0

  if (docCount > 0 && webCount > 0) {
    return t('chat.referencesDocAndWebCount', { docCount, webCount })
  }
  if (docCount > 0) {
    return t('chat.referencesDocCount', { count: docCount })
  }
  if (webCount > 0) {
    return t('chat.referencesWebCount', { count: webCount })
  }

  return ''
})

function toggleReferencesDrawer() {
  const refs = props.session?.knowledge_references
  if (!referencesDrawer || !refs?.length) return
  referencesDrawer.toggle({
    references: refs,
    highlight: null,
    messageId: props.session?.id ? String(props.session.id) : '',
    sourceKey: `rag:${props.session?.id || refs.map((item) => item.knowledge_id || item.knowledge_title).join('|')}`,
    feedbackContext: {
      userQuery: props.userQuery || '',
      answer: props.session?.content || '',
    },
  })
}

function handleStepClick(step: { canOpenReferences?: boolean }) {
  if (!step.canOpenReferences) return
  toggleReferencesDrawer()
}

function toggleExpanded() {
  userExpanded.value = !userExpanded.value
}

function toggleThinking() {
  if (!showThinkingStep.value || !thinkingContent.value) return
  thinkingExpanded.value = !thinkingExpanded.value
}

function scrollThinkingDetailToBottom() {
  nextTick(() => {
    if (!rootElement.value) return
    rootElement.value.querySelectorAll('.thinking-detail-content').forEach((el) => {
      const htmlEl = el as HTMLElement
      htmlEl.scrollTop = htmlEl.scrollHeight
    })
  })
}

watch(thinkingPending, (pending) => {
  if (pending) {
    thinkingExpanded.value = true
  }
})

watch(waitKind, (kind) => waitController.update(kind), { immediate: true })

watch(hasAnswer, (answered) => {
  if (answered && hasThinking.value) {
    thinkingExpanded.value = false
  }
})

watch(thinkingContent, () => {
  if (!isThinkingStreaming.value) return
  scrollThinkingDetailToBottom()
})

watch(thinkingExpanded, (expanded) => {
  if (!expanded || !isThinkingStreaming.value) return
  scrollThinkingDetailToBottom()
})

onBeforeUnmount(() => {
  waitController.dispose()
})
</script>

<style scoped lang="less">
@import '@/components/css/chat-timeline-loading.less';

.rag-pipeline-progress {
  --agent-step-text-size: 14px;
  --agent-step-summary-size: 13px;
  --agent-step-line-color: color-mix(in srgb, var(--td-text-color-primary) 16%, transparent);
  --agent-step-icon-color: var(--td-text-color-placeholder);

  margin: 0;
}

.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border: 0;
}

.tree-container {
  margin: 0 0 8px;
  position: relative;
}

.tree-root {
  margin-bottom: 0;

  .tree-root-toolbar {
    display: flex;
    align-items: center;
    justify-content: flex-start;
    width: 100%;
    min-width: 0;
  }

  .tree-root-expand {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    margin: 0;
    padding: 0;
    border: 0;
    border-radius: 4px;
    background: transparent;
    color: var(--td-text-color-secondary);
    font-size: 14px;
    line-height: 22px;
    cursor: pointer;
    flex: 0 1 auto;
    min-width: 0;
    max-width: 100%;

    &:hover {
      background: transparent;
      color: var(--td-text-color-primary);
    }
  }

  .tree-root-status,
  .tree-root-reference {
    flex: 0 1 auto;
    min-width: 0;
    white-space: nowrap;
  }

  .tree-root-reference {
    display: inline-flex;
    align-items: center;
    gap: 6px;

    &::before {
      content: '';
      width: 3px;
      height: 3px;
      border-radius: 50%;
      background: currentColor;
      opacity: 0.65;
      flex-shrink: 0;
    }
  }

  .tree-root-expand__icon {
    flex-shrink: 0;
    font-size: 14px;
    color: currentColor;
  }

}

.tree-children {
  position: relative;
  padding-left: 0;
  margin-top: 0;
  margin-left: 10px;
}

.tree-children-expanded {
  margin-top: 14px;
}

.tree-child {
  position: relative;
  padding-left: 42px;
  padding-bottom: 0;
  margin-bottom: 18px;

  &::before {
    content: '';
    position: absolute;
    left: 9px;
    top: 22px;
    bottom: -18px;
    width: 0;
    border-left: 1px solid var(--agent-step-line-color);
  }

  .tree-branch {
    display: none;
  }

  &.tree-child-last {
    margin-bottom: 0;

    &::before {
      content: none;
    }
  }
}

.tool-event {
  .action-card {
    position: relative;
    background: transparent;
    border: 0;
    box-shadow: none;

    &.has-reference-trigger {
      cursor: pointer;

      &:hover {
        .action-name,
        .results-summary-text {
          color: var(--td-text-color-primary);
        }
      }
    }
  }

  .action-header {
    display: flex;
    align-items: center;
    min-height: 24px;
    padding: 0;
    cursor: pointer;
    user-select: none;

    &.no-results {
      cursor: default;
    }
  }

  .action-title {
    display: flex;
    align-items: center;
    gap: 12px;
    position: relative;
    flex: 0 1 auto;
    min-width: 0;

    .action-show-icon {
      flex-shrink: 0;
      margin-left: 2px;
    }
  }

  .action-title-icon {
    position: absolute;
    left: -42px;
    top: 3px;
    width: 18px;
    height: 18px;
    flex-shrink: 0;
    color: var(--agent-step-icon-color);
  }

  .action-name {
    font-size: var(--agent-step-text-size);
    line-height: 1.55;
    font-weight: 400;
    color: var(--td-text-color-secondary);
    word-break: break-word;
    max-width: min(820px, 100%);
  }
}

.rag-trace-card {
  cursor: pointer;

  .rag-trace-summary {
    margin-left: 8px;
    font-size: 12px;
    color: var(--td-text-color-placeholder);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    min-width: 0;
  }

  .rag-trace-warn {
    flex-shrink: 0;
    margin-left: 6px;
    padding: 0 2px;
    border: 0;
    background: transparent;
    cursor: pointer;
    font-size: 11px;
    font-weight: 600;
    color: var(--td-warning-color);
    border-radius: 4px;

    &:hover {
      background: color-mix(in srgb, var(--td-warning-color) 12%, transparent);
    }
  }

  .rag-trace-mini {
    flex-shrink: 0;
    display: inline-flex;
    width: 56px;
    height: 6px;
    margin-left: 8px;
    border-radius: 3px;
    overflow: hidden;
    background: color-mix(in srgb, var(--td-text-color-primary) 6%, transparent);
  }

  .rag-trace-mini-seg {
    display: block;
    height: 100%;

    &.is-recall {
      background: color-mix(in srgb, var(--td-brand-color) 38%, transparent);
    }

    &.is-rerank {
      background: color-mix(in srgb, var(--td-brand-color) 60%, transparent);
    }

    &.is-generate {
      background: color-mix(in srgb, var(--td-brand-color) 84%, transparent);
    }
  }

  .rag-trace-copy {
    flex-shrink: 0;
    margin-left: 6px;
    padding: 3px;
    border: 0;
    border-radius: 6px;
    background: transparent;
    color: var(--td-text-color-placeholder);
    cursor: pointer;
    line-height: 0;
    opacity: 0;
    transition: opacity 0.15s ease, color 0.15s ease, background 0.15s ease;

    &:hover {
      background: var(--td-bg-color-secondarycontainer);
      color: var(--td-text-color-primary);
    }
  }

  &:hover .rag-trace-copy {
    opacity: 1;
  }

  .rag-stage-chevron {
    flex-shrink: 0;
    margin-left: 4px;
    font-size: 14px;
    color: var(--td-text-color-placeholder);
  }

  .rag-trace-body {
    margin: 6px 0 2px;
    padding-left: 2px;
  }

  .rag-trace-group {
    margin: 10px 0 4px;
    padding-top: 8px;
    border-top: 1px solid var(--td-component-stroke);
    font-size: 11px;
    font-weight: 600;
    letter-spacing: 0.5px;
    color: var(--td-text-color-placeholder);

    &:first-child {
      margin-top: 2px;
      padding-top: 0;
      border-top: 0;
    }
  }

  .rag-trace-row {
    display: grid;
    grid-template-columns: 7em minmax(0, 1fr) 52px 120px;
    align-items: center;
    gap: 10px;
    padding: 3px 4px;
    border-radius: 4px;
    font-size: 12px;
    line-height: 1.5;

    &:hover {
      background: color-mix(in srgb, var(--td-text-color-primary) 4%, transparent);
    }

    &.is-expandable {
      cursor: pointer;
    }

    &.is-expanded {
      background: color-mix(in srgb, var(--td-text-color-primary) 4%, transparent);
    }

    &.is-flash {
      animation: rag-trace-flash 1.6s ease;
    }
  }

  .rag-trace-icon-box {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 24px;
    height: 24px;
    flex-shrink: 0;
    border-radius: 6px;
    background: var(--td-bg-color-secondarycontainer);
  }

  .rag-trace-usage {
    display: inline-flex;
    width: 64px;
    height: 4px;
    margin-left: 2px;
    border-radius: 2px;
    overflow: hidden;
    background: color-mix(in srgb, var(--td-text-color-primary) 8%, transparent);
  }

  .rag-trace-usage-fill {
    display: block;
    height: 100%;
    border-radius: 2px;
    background: color-mix(in srgb, var(--td-brand-color) 55%, transparent);

    &.is-hot {
      background: var(--td-warning-color);
    }
  }

  .rag-trace-detail {
    margin: 0 4px 4px 6.5em;
    padding: 4px 8px;
    border-left: 1px solid color-mix(in srgb, var(--td-brand-color) 22%, transparent);
    display: flex;
    flex-direction: column;
    gap: 3px;
  }

  .rag-trace-detail-kv {
    display: inline-flex;
    align-items: baseline;
    gap: 6px;
    font-size: 12px;
    line-height: 1.6;
    font-variant-numeric: tabular-nums;

    .k {
      color: var(--td-text-color-placeholder);
      flex-shrink: 0;
    }

    .v {
      color: var(--td-text-color-secondary);
      word-break: break-word;
    }
  }

  .rag-trace-detail-top {
    display: grid;
    grid-template-columns: 16px 82px minmax(0, 1fr) minmax(80px, 34%);
    align-items: center;
    gap: 8px;
    min-height: 20px;
    font-size: 12px;
    line-height: 1.5;

    .rank {
      color: var(--td-text-color-placeholder);
      font-size: 11px;
      text-align: center;
    }

    .score {
      display: inline-flex;
      align-items: center;
      gap: 5px;
      font-variant-numeric: tabular-nums;
      color: var(--td-text-color-secondary);
      font-size: 11px;
    }

    .score-bar {
      display: inline-flex;
      width: 44px;
      height: 4px;
      border-radius: 2px;
      overflow: hidden;
      background: color-mix(in srgb, var(--td-text-color-primary) 8%, transparent);
    }

    .score-bar-fill {
      display: block;
      height: 100%;
      background: color-mix(in srgb, var(--td-brand-color) 55%, transparent);
    }

    .snippet {
      min-width: 0;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
      color: var(--td-text-color-secondary);
    }

    .doc {
      display: flex;
      flex-direction: column;
      min-width: 0;
      line-height: 1.3;
    }

    .doc-title {
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
      color: var(--td-text-color-secondary);
    }

    .doc-sub {
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
      color: var(--td-text-color-placeholder);
      font-size: 11px;
    }
  }

  .rag-trace-name {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    font-weight: 500;
    color: var(--td-text-color-primary);
    white-space: nowrap;
  }

  .rag-trace-icon {
    flex-shrink: 0;
    color: var(--td-text-color-secondary);
  }

  .rag-trace-spin {
    animation: rag-trace-rotate 1s linear infinite;
  }

  .rag-trace-data {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: 4px;
    min-width: 0;
  }

  .rag-trace-text {
    color: var(--td-text-color-secondary);
    font-variant-numeric: tabular-nums;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .rag-trace-chip {
    display: inline-flex;
    align-items: center;
    padding: 0 6px;
    border-radius: 6px;
    background: var(--td-bg-color-secondarycontainer);
    font-size: 11px;
    font-weight: 500;
    line-height: 1.7;
    color: var(--td-text-color-secondary);
    font-variant-numeric: tabular-nums;
    white-space: nowrap;

    &.is-warn {
      background: color-mix(in srgb, var(--td-warning-color) 14%, transparent);
      color: var(--td-warning-color);
    }

    &.is-muted {
      color: var(--td-text-color-placeholder);
    }

    &.is-ok {
      background: var(--td-brand-color-light);
      color: var(--td-brand-color);
    }
  }

  .rag-trace-dur {
    text-align: right;
    color: var(--td-text-color-secondary);
    font-size: 11px;
    font-variant-numeric: tabular-nums;
    white-space: nowrap;
  }

  .rag-trace-bar {
    display: block;
    height: 6px;
    border-radius: 3px;
    background: color-mix(in srgb, var(--td-text-color-primary) 6%, transparent);
    overflow: hidden;
  }

  .rag-trace-bar-fill {
    display: block;
    height: 100%;
    border-radius: 3px;
    background: color-mix(in srgb, var(--td-brand-color) 40%, transparent);
    transition: width 0.3s ease;

    &.is-max {
      background: var(--td-brand-color);
    }
  }
}

@keyframes rag-trace-rotate {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

@keyframes rag-trace-flash {
  0%,
  60% {
    background: color-mix(in srgb, var(--td-warning-color) 18%, transparent);
  }
  100% {
    background: transparent;
  }
}

.search-results-summary-fixed {
  padding: 2px 0 0 0;

  .results-summary-text {
    font-size: var(--agent-step-summary-size);
    font-weight: 400;
    color: var(--td-text-color-secondary);
    line-height: 1.5;

    :deep(strong) {
      color: var(--td-text-color-secondary);
      font-weight: 500;
    }
  }
}

.rag-thinking-step {
  .thinking-detail-content {
    margin-top: 4px;
    padding: 0;
    font-size: var(--agent-step-summary-size);
    font-weight: 400;
    color: var(--td-text-color-placeholder);
    line-height: 1.55;
    white-space: pre-wrap;
    word-break: break-word;
    max-height: 200px;
    overflow-y: auto;
  }

  .action-pending .action-name {
    color: var(--td-text-color-secondary);
  }
}

@media (max-width: 640px) {
  .tree-root {
    .tree-root-toolbar {
      gap: 8px;
    }

    .tree-root-expand {
      max-width: 100%;
    }
  }
}
</style>
