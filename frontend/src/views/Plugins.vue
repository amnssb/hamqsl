<template>
  <div>
    <div class="page-heading">
      <div>
        <h1>插件</h1>
        <p>功能与主题皆以插件形式扩展，开关即时生效，无需重启服务。</p>
      </div>
    </div>

    <el-row :gutter="16">
      <el-col v-for="p in items" :key="p.name" :xs="24" :md="12" class="plugin-col">
        <el-card class="plugin-card">
          <div class="plugin-head">
            <div>
              <div class="plugin-title">{{ p.title }} <span class="plugin-ver">v{{ p.version }}</span></div>
              <div class="plugin-name">{{ p.name }}</div>
            </div>
            <el-switch :model-value="p.enabled" :loading="p._busy" @change="v => toggle(p, v)" />
          </div>
          <p class="plugin-desc">{{ p.description }}</p>
          <div class="plugin-foot">
            <el-tag size="small" :type="p.kind === 'theme' ? 'warning' : 'success'">{{ p.kind === 'theme' ? '主题' : '功能' }}</el-tag>
            <div v-if="swatchesOf(p)" class="swatches">
              <span v-for="c in swatchesOf(p)" :key="c" class="swatch" :style="{ background: c }"></span>
            </div>
            <el-button v-if="p.name === 'stats_daily' && p.enabled" size="small" text type="primary" @click="openStats">查看数据</el-button>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-dialog v-model="statsVisible" title="每日统计（最近 14 天，东八区）" width="640px">
      <div v-if="statsRows.length" class="chart">
        <div v-for="r in statsRows" :key="r.day" class="chart-row">
          <span class="chart-day">{{ r.day.slice(5) }}</span>
          <div class="chart-bar-wrap">
            <div class="chart-bar" :style="{ height: barH(r.created) + 'px' }"></div>
          </div>
          <span class="chart-val">建 {{ r.created }} · 发 {{ r.sent }} · 签 {{ r.signed }}</span>
        </div>
      </div>
      <el-empty v-else description="暂无数据" :image-size="80" />
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import api from '../api'
import { themeRegistry, applyEnabledThemes } from '../plugins/registry'
import { ElMessage } from 'element-plus'

const items = ref([])
const statsVisible = ref(false)
const statsRows = ref([])

async function load() {
  const res = await api.get('/plugins')
  items.value = (res?.items || []).map(p => ({ ...p, _busy: false }))
  // 主题变化即时应用到当前页面（公开页与下次进入时由 App.vue 统一应用）
  applyEnabledThemes(items.value.filter(p => p.kind === 'theme' && p.enabled).map(p => p.name))
}

function swatchesOf(p) {
  return (themeRegistry[p.name] || {}).swatches || null
}

async function toggle(p, on) {
  p._busy = true
  try {
    await api.post(`/plugins/${p.name}/${on ? 'enable' : 'disable'}`)
    ElMessage.success(on ? `「${p.title}」已启用，即时生效` : `「${p.title}」已禁用`)
    await load()
  } catch {
    // 拦截器已统一提示错误
  } finally {
    p._busy = false
  }
}

async function openStats() {
  const res = await api.get('/ext/stats_daily/summary', { params: { days: 14 } })
  statsRows.value = res?.items || []
  statsVisible.value = true
}

const maxCreated = () => Math.max(1, ...statsRows.value.map(r => r.created))
function barH(n) {
  return Math.round((n / maxCreated()) * 64) + 4
}

onMounted(load)
</script>

<style scoped>
.plugin-col { margin-bottom: 16px; }
.plugin-head { display: flex; align-items: flex-start; justify-content: space-between; gap: 12px; }
.plugin-title { color: var(--qsl-navy); font-size: 16px; font-weight: 750; }
.plugin-ver { color: var(--qsl-muted); font-size: 11px; font-weight: 500; }
.plugin-name { margin-top: 2px; color: var(--qsl-muted); font-size: 11px; letter-spacing: .04em; }
.plugin-desc { margin: 10px 0 14px; color: var(--qsl-muted); font-size: 13px; line-height: 1.7; min-height: 44px; }
.plugin-foot { display: flex; align-items: center; gap: 10px; }
.swatches { display: inline-flex; gap: 4px; }
.swatch { width: 16px; height: 16px; border-radius: 3px; border: 1px solid rgba(0,0,0,.12); }
.chart { display: flex; flex-direction: column; gap: 6px; }
.chart-row { display: flex; align-items: flex-end; gap: 10px; }
.chart-day { width: 40px; flex: none; color: var(--qsl-muted); font-size: 11px; text-align: right; }
.chart-bar-wrap { flex: 1; display: flex; align-items: flex-end; height: 70px; border-bottom: 1px solid var(--qsl-line); }
.chart-bar { width: 100%; background: var(--qsl-orange); border-radius: 2px 2px 0 0; opacity: .85; min-height: 2px; }
.chart-val { flex: none; width: 170px; color: var(--qsl-muted); font-size: 11px; }
</style>
