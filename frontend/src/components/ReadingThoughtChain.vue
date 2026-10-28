<template>
  <div v-if="advancing" class="reading-thought-card reading-thought-loading">
    <span class="reading-thought-title">{{ t('reading.thinkingNow') }}</span>
    <t-loading size="small" />
  </div>
  <div v-else class="reading-thought-card">
    <div class="reading-thought-head">
      <button type="button" class="reading-thought-toggle" @click="expanded = !expanded">
        <span class="reading-thought-title">{{ t('reading.thoughtChain') }}</span>
        <span class="reading-thought-meta">
          {{ t('reading.thoughtMeta', {
            searches: searches.length,
            questions: questions.length,
            duration: durationLabel || '—',
          }) }}
        </span>
        <span class="reading-thought-caret">{{ expanded ? '▾' : '▸' }}</span>
      </button>
      <t-select v-if="roundOptions.length > 1" v-model="selectedRoundValue"
        :options="roundOptions" size="small" class="reading-thought-round" />
    </div>
    <div v-if="expanded" class="reading-thought-body">
      <div class="reading-thought-phase">
        <span class="reading-thought-label">{{ t('reading.phaseWindow') }}</span>
        <p>
          {{ sectionTitle || '—' }}
          <template v-if="sectionHeading"> · {{ sectionHeading }}</template>
        </p>
      </div>
      <div class="reading-thought-phase">
        <span class="reading-thought-label">{{ t('reading.phaseReasoning') }}</span>
        <pre v-if="reasoning" class="reading-thought-reasoning">{{ reasoning }}</pre>
        <p v-else class="reading-empty">{{ t('reading.noReasoning') }}</p>
      </div>
      <div v-if="understanding" class="reading-thought-phase">
        <span class="reading-thought-label">{{ t('reading.phaseUnderstanding') }}</span>
        <p>{{ understanding }}</p>
      </div>
      <div v-if="questions.length" class="reading-thought-phase">
        <span class="reading-thought-label">{{ t('reading.phaseQuestions') }}</span>
        <ul>
          <li v-for="question in questions" :key="question.id">
            {{ question.text }}<span v-if="question.why" class="reading-thought-why">（{{ question.why }}）</span>
          </li>
        </ul>
      </div>
      <div v-if="searches.length" class="reading-thought-phase">
        <span class="reading-thought-label">{{ t('reading.phaseSearch') }}</span>
        <ul>
          <li v-for="(item, index) in searches" :key="index">
            {{ item.query }} · {{ t('reading.searchHits', { count: item.hit_count }) }}
          </li>
        </ul>
      </div>
      <div class="reading-thought-phase">
        <span class="reading-thought-label">{{ t('reading.phaseDeposit') }}</span>
        <p>{{ t('reading.traceCounts', counts) }}</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ReadingNotes, ReadingQuestionNode } from '@/api/reading'

const props = defineProps<{
  trace: ReadingNotes
  questions: ReadingQuestionNode[]
  sectionTitle?: string
  sectionHeading?: string
  durationLabel?: string
  roundOptions: Array<{ value: number; label: string }>
  selectedRound: number
  advancing?: boolean
}>()

const emit = defineEmits<{
  (event: 'update:selectedRound', value: number): void
}>()

const { t } = useI18n()
const expanded = ref(true)
const reasoning = computed(() =>
  typeof props.trace.reasoning === 'string' ? props.trace.reasoning : '',
)
const understanding = computed(() =>
  typeof props.trace.understanding === 'string' ? props.trace.understanding : '',
)
const searches = computed(() => props.trace.search_trace || [])
const counts = computed(() => ({
  conclusions: Number(props.trace.conclusion_count || 0),
  terms: Number(props.trace.term_count || 0),
  contradictions: Number(props.trace.contradiction_count || 0),
}))
const selectedRoundValue = computed({
  get: () => props.selectedRound,
  set: (value: number) => emit('update:selectedRound', Number(value)),
})
</script>

<style scoped lang="less">
.reading-thought-card {
  padding: 10px 12px;
  margin-bottom: 14px;
  background: var(--td-bg-color-secondarycontainer);
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
}

.reading-thought-loading {
  display: flex;
  gap: 10px;
  align-items: center;
  font-size: 12px;
  color: var(--td-text-color-secondary);
}

.reading-thought-head {
  display: flex;
  gap: 8px;
  align-items: center;
}

.reading-thought-toggle {
  display: flex;
  flex: 1;
  gap: 8px;
  align-items: center;
  min-width: 0;
  padding: 0;
  font-family: inherit;
  color: var(--td-text-color-primary);
  text-align: left;
  cursor: pointer;
  background: transparent;
  border: 0;
}

.reading-thought-title {
  flex-shrink: 0;
  font-size: 13px;
  font-weight: 600;
}

.reading-thought-meta {
  margin-left: auto;
  overflow: hidden;
  font-size: 11px;
  color: var(--td-text-color-secondary);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.reading-thought-caret {
  flex-shrink: 0;
  font-size: 12px;
  color: var(--td-text-color-secondary);
}

.reading-thought-round {
  flex-shrink: 0;
  width: 112px;
}

.reading-thought-body {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin-top: 10px;
}

.reading-thought-phase {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.reading-thought-label {
  font-size: 11px;
  font-weight: 600;
  color: var(--td-brand-color);
}

.reading-thought-phase p,
.reading-thought-phase ul {
  margin: 0;
  font-size: 12px;
  line-height: 1.6;
  color: var(--td-text-color-primary);
}

.reading-thought-phase ul {
  padding-left: 16px;
}

.reading-thought-reasoning {
  max-height: 280px;
  padding: 8px;
  margin: 0;
  overflow: auto;
  font-family: inherit;
  font-size: 12px;
  line-height: 1.7;
  color: var(--td-text-color-primary);
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  word-break: break-word;
  background: var(--td-bg-color-container);
  border-radius: 6px;
  scrollbar-width: none;
}

.reading-thought-reasoning::-webkit-scrollbar {
  display: none;
}

.reading-thought-why {
  color: var(--td-text-color-secondary);
}

.reading-empty {
  margin: 0;
  font-size: 12px;
  color: var(--td-text-color-placeholder);
}
</style>
