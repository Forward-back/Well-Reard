<template>
  <SplitDiff v-if="mode === 'split'" :before="before" :after="after" />
  <table v-else class="cv-diff">
    <tbody>
      <tr v-for="(row, index) in result.rows" :key="index" :class="rowClass(row.type)">
        <td class="cv-diff-no">{{ row.oldNo || '' }}</td>
        <td class="cv-diff-no">{{ row.newNo || '' }}</td>
        <td class="cv-diff-sign">{{ row.type === 'add' ? '+' : row.type === 'del' ? '−' : '' }}</td>
        <td>{{ row.text || ' ' }}</td>
      </tr>
    </tbody>
  </table>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { diffLines, type DiffRowType } from './diff'
import SplitDiff from './SplitDiff.vue'

const props = withDefaults(defineProps<{
  before: string
  after: string
  mode?: 'unified' | 'split'
}>(), {
  mode: 'unified',
})

const result = computed(() => diffLines(props.before, props.after))

function rowClass(type: DiffRowType) {
  if (type === 'add') return 'is-add'
  if (type === 'del') return 'is-del'
  return 'is-ctx'
}
</script>

<style scoped lang="less">
@import '@/styles/collaboration.css';
</style>
