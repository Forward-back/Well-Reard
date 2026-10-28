<template>
  <div class="cv-split-diff-panes">
    <div
      ref="leftPane"
      class="cv-split-pane"
      @scroll.passive="syncFromLeft"
    >
      <div
        v-for="(row, index) in displayRows"
        :key="index"
        class="cv-split-row"
        :class="rowClass(row.left?.type)"
      >
        <span class="cv-split-no" :class="{ 'is-del-no': row.left?.type === 'del' }">{{ row.left?.no ?? '' }}</span>
        <span class="cv-split-text">{{ row.left?.text ?? ' ' }}</span>
      </div>
    </div>
    <div
      ref="rightPane"
      class="cv-split-pane"
      @scroll.passive="syncFromRight"
    >
      <div
        v-for="(row, index) in displayRows"
        :key="index"
        class="cv-split-row"
        :class="rowClass(row.right?.type)"
      >
        <span class="cv-split-no" :class="{ 'is-add-no': row.right?.type === 'add' }">{{ row.right?.no ?? '' }}</span>
        <span class="cv-split-text">{{ row.right?.text ?? ' ' }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { diffLines, type DiffRow } from './diff'

const props = withDefaults(defineProps<{
  before: string
  after: string
}>(), {})

const leftPane = ref<HTMLElement | null>(null)
const rightPane = ref<HTMLElement | null>(null)
let syncing = false

const result = computed(() => diffLines(props.before, props.after))

interface SplitCell {
  type: 'ctx' | 'add' | 'del'
  no: string
  text: string
}

interface SplitRow {
  left: SplitCell | null
  right: SplitCell | null
}

// A replacement is emitted by the LCS as a delete row followed by an add row.
// Pairing them onto one display row keeps old and new lines side by side, so
// a modified line reads as one aligned change instead of two staggered rows.
const displayRows = computed<SplitRow[]>(() => {
  const rows = result.value.rows
  const out: SplitRow[] = []
  let index = 0
  while (index < rows.length) {
    const row = rows[index]
    if (row.type === 'ctx') {
      out.push({
        left: { type: 'ctx', no: String(row.oldNo ?? ''), text: row.text || ' ' },
        right: { type: 'ctx', no: String(row.newNo ?? ''), text: row.text || ' ' },
      })
      index++
      continue
    }
    const deletions: DiffRow[] = []
    const additions: DiffRow[] = []
    while (index < rows.length && rows[index].type !== 'ctx') {
      if (rows[index].type === 'del') deletions.push(rows[index])
      else additions.push(rows[index])
      index++
    }
    const count = Math.max(deletions.length, additions.length)
    for (let offset = 0; offset < count; offset++) {
      const deletion = deletions[offset]
      const addition = additions[offset]
      out.push({
        left: deletion ? { type: 'del', no: String(deletion.oldNo ?? ''), text: deletion.text || ' ' } : null,
        right: addition ? { type: 'add', no: `+${addition.newNo ?? ''}`, text: addition.text || ' ' } : null,
      })
    }
  }
  return out
})

function syncFromLeft() {
  syncVertical(leftPane.value, rightPane.value)
}

function syncFromRight() {
  syncVertical(rightPane.value, leftPane.value)
}

function syncVertical(source: HTMLElement | null, target: HTMLElement | null) {
  if (!source || !target || syncing) return
  const next = source.scrollTop
  if (Math.abs(target.scrollTop - next) < 1) return
  syncing = true
  target.scrollTop = next
  requestAnimationFrame(() => { syncing = false })
}

function rowClass(type?: 'ctx' | 'add' | 'del') {
  if (type === 'add') return 'is-add'
  if (type === 'del') return 'is-del'
  return 'is-ctx'
}
</script>

<style scoped lang="less">
.cv-split-diff-panes {
  display: flex;
  min-height: 180px;
  max-height: 360px;
  border-top: 1px solid var(--cv-line);
  background: #fff;
}

.cv-split-pane {
  flex: 1;
  min-width: 0;
  overflow: auto;
  scrollbar-width: none;
  scrollbar-color: #cbd5e1 transparent;
}

.cv-split-pane + .cv-split-pane {
  border-left: 1px solid var(--cv-line);
  scrollbar-width: thin;
}

.cv-split-pane::-webkit-scrollbar {
  width: 0;
  height: 0;
  display: none;
}

.cv-split-pane + .cv-split-pane::-webkit-scrollbar {
  width: 8px;
  height: 0;
}

.cv-split-pane + .cv-split-pane::-webkit-scrollbar:horizontal {
  display: none;
}

.cv-split-pane::-webkit-scrollbar-track {
  background: #f6f8fa;
}

.cv-split-pane::-webkit-scrollbar-thumb {
  border-radius: 4px;
  background: #cbd5e1;
}

.cv-split-pane::-webkit-scrollbar-thumb:hover {
  background: #9ca3af;
}

.cv-split-row {
  display: flex;
  min-width: max-content;
  height: 20px;
  line-height: 20px;
  font: 12px/20px 'Cascadia Code', 'SFMono-Regular', Consolas, 'Liberation Mono', Menlo, monospace;
}

.cv-split-no {
  position: sticky;
  left: 0;
  z-index: 1;
  flex: none;
  width: 42px;
  padding-right: 8px;
  text-align: right;
  color: #9ca3af;
  background: #f6f8fa;
  user-select: none;
}

.cv-split-text {
  flex: none;
  min-width: max-content;
  padding: 0 12px;
  color: #4b5563;
  white-space: pre;
  word-break: normal;
}

.cv-split-row.is-del .cv-split-text {
  background: #ffebe9;
}

.cv-split-row.is-del .cv-split-no,
.cv-split-no.is-del-no {
  color: #cf222e;
}

.cv-split-row.is-add .cv-split-text {
  background: #e6ffec;
}

.cv-split-row.is-add .cv-split-no {
  color: #116329;
  font-weight: 600;
}

.cv-split-row:hover .cv-split-text {
  background-image: linear-gradient(rgba(16, 185, 129, 0.035), rgba(16, 185, 129, 0.035));
}
</style>
