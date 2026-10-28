<template>
  <svg
    class="trace-source-icon"
    :class="{ 'is-active': active }"
    :width="size"
    :height="size"
    viewBox="0 0 50 50"
    aria-hidden="true"
    focusable="false"
  >
    <path class="trace-source-icon__light" fill-rule="evenodd" :d="icon.light" />
    <path class="trace-source-icon__dark" fill-rule="evenodd" :d="icon.dark" />
  </svg>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { TRACE_SOURCE_ICONS, type TraceSourceIconName } from './traceSourceIcons'

const props = withDefaults(
  defineProps<{
    name: TraceSourceIconName
    size?: number | string
    active?: boolean
  }>(),
  { size: 14, active: false },
)

const icon = computed(() => TRACE_SOURCE_ICONS[props.name])
</script>

<style scoped>
.trace-source-icon {
  display: inline-block;
  flex-shrink: 0;
  vertical-align: middle;
}

.trace-source-icon__dark {
  fill: #575e66;
}

.trace-source-icon__light {
  fill: #b3bac2;
}

.trace-source-icon.is-active .trace-source-icon__dark {
  fill: var(--td-success-color);
}

.trace-source-icon.is-active .trace-source-icon__light {
  fill: var(--td-success-color);
  opacity: 0.55;
}
</style>
