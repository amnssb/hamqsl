<template>
  <!-- 手机端：字段卡片流（每行数据一张卡，label:value 垂直排，操作按钮独立一排） -->
  <div v-if="isMobile" class="rcard-list">
    <div v-for="(row, i) in data" :key="rowKey ? row[rowKey] : i" class="rcard">
      <div class="rcard-head">
        <slot name="card-title" :row="row">{{ row[fields[0]?.prop] ?? '' }}</slot>
      </div>
      <div v-for="f in cardFields" :key="f.prop" class="rcard-row">
        <span class="rcard-label">{{ f.label }}</span>
        <span class="rcard-value">
          <slot :name="'card-' + f.prop" :row="row">{{ row[f.prop] ?? '-' }}</slot>
        </span>
      </div>
      <div v-if="$slots.actions" class="rcard-actions">
        <slot name="actions" :row="row" />
      </div>
    </div>
    <el-empty v-if="!data || !data.length" description="暂无数据" :image-size="72" />
  </div>
  <!-- 桌面端：原 el-table 原样渲染 -->
  <slot v-else />
</template>

<script setup>
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'

const props = defineProps({
  data: { type: Array, default: () => [] },
  // 手机卡片展示的字段：[{ prop, label }]；第一个字段同时作为卡片标题
  fields: { type: Array, required: true },
  rowKey: { type: String, default: 'id' },
})

const isMobile = ref(false)
const sync = () => { isMobile.value = window.innerWidth <= 640 }
onMounted(() => { sync(); window.addEventListener('resize', sync) })
onBeforeUnmount(() => window.removeEventListener('resize', sync))

const cardFields = computed(() => props.fields.slice(1))
</script>

<style scoped>
.rcard-list { display: flex; flex-direction: column; gap: 10px; }
.rcard {
  border: 1px solid var(--qsl-line);
  background: var(--qsl-panel);
  border-radius: 4px;
  padding: 12px 14px;
  box-shadow: 2px 2px 0 rgba(24, 45, 61, .06);
}
.rcard-head {
  display: flex; align-items: center; justify-content: space-between; gap: 10px;
  padding-bottom: 8px; margin-bottom: 6px;
  border-bottom: 1px solid var(--qsl-line);
  color: var(--qsl-navy); font-weight: 750; font-size: 14px;
  word-break: break-all;
}
.rcard-row { display: flex; gap: 10px; padding: 3px 0; font-size: 13px; line-height: 1.55; }
.rcard-label { flex: none; width: 74px; color: var(--qsl-muted); }
.rcard-value { flex: 1; min-width: 0; word-break: break-all; color: var(--qsl-ink); }
.rcard-actions {
  display: flex; flex-wrap: wrap; gap: 4px 10px;
  margin-top: 10px; padding-top: 10px;
  border-top: 1px dashed var(--qsl-line);
}
</style>
