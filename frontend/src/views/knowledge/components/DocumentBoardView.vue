<script setup lang="ts">
// 看板视图:每 UI 状态桶一栏,精简行卡,栏内滚动加载更多。
// 数据由父组件(KnowledgeBase.vue)按桶独立分页拉取,本组件纯展示。
interface BoardItem {
  id: string;
  parse_status: string;
  title?: string;
  file_name?: string;
  display_name?: string;
  original_file_name?: string;
  type?: string;
  updated_at?: string;
}
interface BoardColumnState {
  items: BoardItem[];
  page: number;
  total: number;
  loading: boolean;
  loadingMore: boolean;
}
const props = defineProps<{
  columns: string[];
  state: Record<string, BoardColumnState>;
  statusLabel: (key: string) => string;
  detailLabel: (status: string) => string;
  emptyText: string;
  loadMoreText: string;
}>();
const emit = defineEmits<{
  open: [item: BoardItem];
  loadMore: [key: string];
}>();

const col = (key: string): BoardColumnState =>
  props.state[key] || { items: [], page: 1, total: 0, loading: false, loadingMore: false };

const itemTitle = (item: BoardItem) =>
  item.title || item.file_name || item.display_name || item.original_file_name || '';

const iconOf = (item: BoardItem): string => {
  if (item.type === 'manual') return 'edit';
  if (item.type === 'url') return 'link';
  return 'file';
};

const formatTime = (iso?: string): string => {
  if (!iso) return '';
  return iso.slice(0, 16).replace('T', ' ');
};

const onColumnScroll = (key: string, event: Event) => {
  const el = event.currentTarget as HTMLElement;
  const c = col(key);
  if (c.loading || c.loadingMore || c.items.length >= c.total) return;
  if (el.scrollTop + el.clientHeight >= el.scrollHeight - 10) {
    emit('loadMore', key);
  }
};
</script>

<template>
  <div class="doc-board">
    <div v-for="key in columns" :key="key" class="doc-board-column">
      <div class="doc-board-column__header">
        <span class="doc-board-column__title">{{ statusLabel(key) }}</span>
        <span class="doc-board-column__count">{{ col(key).total }}</span>
      </div>
      <div class="doc-board-column__body" @scroll="onColumnScroll(key, $event)">
        <div v-if="col(key).loading" class="doc-board-skeleton">
          <div v-for="n in 5" :key="'skel-' + n" class="doc-board-row doc-board-row--skeleton">
            <t-skeleton animation="gradient" :row-col="[{ width: '70%', height: '16px' }]" />
          </div>
        </div>
        <template v-else>
          <button
            v-for="item in col(key).items"
            :key="item.id"
            type="button"
            class="doc-board-row"
            :title="itemTitle(item)"
            @click="emit('open', item)"
          >
            <span class="doc-board-row__icon"><t-icon :name="iconOf(item)" size="16px" /></span>
            <span class="doc-board-row__main">
              <span class="doc-board-row__title">{{ itemTitle(item) }}</span>
              <span class="doc-board-row__meta">{{ formatTime(item.updated_at) }}</span>
            </span>
            <t-tooltip
              v-if="detailLabel(item.parse_status)"
              :content="detailLabel(item.parse_status)"
              placement="left"
            >
              <span class="doc-board-row__badge" aria-hidden="true">···</span>
            </t-tooltip>
          </button>
          <div v-if="!col(key).items.length" class="doc-board-empty">{{ emptyText }}</div>
          <div v-if="col(key).items.length < col(key).total" class="doc-board-loadmore">
            <t-button
              variant="text"
              size="small"
              :loading="col(key).loadingMore"
              @click="emit('loadMore', key)"
            >
              {{ loadMoreText }}
            </t-button>
          </div>
        </template>
      </div>
    </div>
  </div>
</template>

<style scoped>
.doc-board {
  display: flex;
  gap: 12px;
  height: 100%;
  min-height: 0;
  padding: 4px 2px;
  overflow-x: auto;
}
.doc-board-column {
  flex: 1 1 0;
  min-width: 220px;
  display: flex;
  flex-direction: column;
  min-height: 0;
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-border);
  border-radius: 6px;
  overflow: hidden;
}
.doc-board-column__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  border-bottom: 1px solid var(--td-component-border);
  background: var(--td-bg-color-secondarycontainer);
}
.doc-board-column__title {
  font-size: 13px;
  font-weight: 600;
}
.doc-board-column__count {
  font-size: 12px;
  color: var(--td-text-color-secondary);
  background: var(--td-bg-color-container);
  border-radius: 10px;
  padding: 0 8px;
  line-height: 18px;
}
.doc-board-column__body {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 6px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.doc-board-row {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  text-align: left;
  padding: 8px 10px;
  border: none;
  border-radius: 4px;
  background: transparent;
  cursor: pointer;
  transition: background 0.15s;
}
.doc-board-row:hover {
  background: var(--td-bg-color-container-hover);
}
.doc-board-row__icon {
  flex: none;
  color: var(--td-text-color-placeholder);
  display: inline-flex;
}
.doc-board-row__main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.doc-board-row__title {
  font-size: 13px;
  color: var(--td-text-color-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.doc-board-row__meta {
  font-size: 12px;
  color: var(--td-text-color-placeholder);
}
.doc-board-row__badge {
  flex: none;
  font-size: 12px;
  color: var(--td-brand-color);
  cursor: default;
}
.doc-board-row--skeleton {
  cursor: default;
  padding: 12px 10px;
}
.doc-board-empty {
  padding: 24px 8px;
  text-align: center;
  font-size: 12px;
  color: var(--td-text-color-placeholder);
}
.doc-board-loadmore {
  display: flex;
  justify-content: center;
  padding: 4px 0;
}
</style>
