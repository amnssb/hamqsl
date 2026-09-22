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
  <!-- 桌面端与平板端：包裹水平平滑滚动容器，避免撑开视口 -->
  <div v-else class="rtable-desktop-wrap">
    <slot />
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'

const props = defineProps({
  data: { type: Array, default: () => [] },
  fields: { type: Array, required: true },
  rowKey: { type: String, default: 'id' },
})

const isMobile = ref(false)
const sync = () => { isMobile.value = window.innerWidth <= 720 }
onMounted(() => { sync(); window.addEventListener('resize', sync) })
onBeforeUnmount(() => window.removeEventListener('resize', sync))

const cardFields = computed(() => props.fields.slice(1))
</script>

<style scoped>
.rtable-desktop-wrap {
  width: 100%;
  overflow-x: auto;
  -webkit-overflow-scrolling: touch;
}

.rcard-list { display: flex; flex-direction: column; gap: 12px; }
.rcard {
  border: 1px solid #e2e8f0;
  background: #ffffff;
  border-radius: 12px;
  padding: 16px;
  box-shadow: 0 1px 3px 0 rgba(0, 0, 0, 0.04);
  transition: all 0.2s ease;
}
.rcard-head {
  display: flex; align-items: center; justify-content: space-between; gap: 10px;
  padding-bottom: 10px; margin-bottom: 8px;
  border-bottom: 1px solid #f1f5f9;
  color: #0f172a; font-weight: 750; font-size: 14.5px;
  word-break: break-all;
}
.rcard-row { display: flex; gap: 12px; padding: 4px 0; font-size: 13px; line-height: 1.6; }
.rcard-label { flex: none; width: 78px; color: #64748b; font-weight: 500; }
.rcard-value { flex: 1; min-width: 0; word-break: break-all; color: #0f172a; }
.rcard-actions {
  display: flex; flex-wrap: wrap; gap: 6px 10px;
  margin-top: 12px; padding-top: 12px;
  border-top: 1px dashed #e2e8f0;
}
</style>
