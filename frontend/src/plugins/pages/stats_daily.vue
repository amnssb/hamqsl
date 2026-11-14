<!--
每日统计插件的自有界面。
插件页上这个插件启用时多一个「查看数据」按钮，点开是最近 14 天柱状图。
-->
<template>
  <el-button size="small" text type="primary" @click="open">查看数据</el-button>
  <el-dialog v-model="visible" title="每日统计（最近 14 天，东八区）" width="640px">
    <div v-if="rows.length" class="chart">
      <div v-for="r in rows" :key="r.day" class="chart-row">
        <span class="chart-day">{{ r.day.slice(5) }}</span>
        <div class="chart-bar-wrap">
          <div class="chart-bar" :style="{ height: barH(r.created) + 'px' }"></div>
        </div>
        <span class="chart-val">建 {{ r.created }} · 发 {{ r.sent }} · 签 {{ r.signed }}</span>
      </div>
    </div>
    <el-empty v-else description="暂无数据" :image-size="80" />
  </el-dialog>
</template>

<script setup>
// --- 打开弹窗并取数 ---
import { ref } from 'vue'
import api from '../../api'

const visible = ref(false)
const rows = ref([])

async function open() {
  const res = await api.get('/ext/stats_daily/summary', { params: { days: 14 } }) // 后端按东八区分好桶
  rows.value = res?.items || []
  visible.value = true
}

// --- 柱高：以最高一天为满格 ---
const maxCreated = () => Math.max(1, ...rows.value.map(r => r.created)) // 至少 1，避免除以 0
function barH(n) {
  return Math.round((n / maxCreated()) * 64) + 4 // 底量 4px，0 也有个点
}
</script>

<style scoped>
.chart { display: flex; flex-direction: column; gap: 8px; }
.chart-row { display: flex; align-items: flex-end; gap: 10px; }
.chart-day { width: 44px; flex: none; color: var(--qsl-muted); font-size: 11px; text-align: right; }
.chart-bar-wrap { flex: 1; display: flex; align-items: flex-end; height: 70px; border-bottom: 1px solid var(--qsl-line); }
.chart-bar { width: 100%; background: var(--el-color-primary); border-radius: 4px 4px 0 0; min-height: 2px; }
.chart-val { flex: none; width: 170px; color: var(--qsl-muted); font-size: 11.5px; }
</style>
