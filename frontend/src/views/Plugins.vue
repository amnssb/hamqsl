<template>
  <div>
    <div class="page-heading">
      <div>
        <h1>插件</h1>
        <p>功能与主题皆以插件形式扩展，开关即时生效，无需重启服务。</p>
      </div>
      <el-button type="primary" plain @click="$router.push('/admin/theme-lab')">
        主题巡检
      </el-button>
    </div>

    <el-row :gutter="16">
      <el-col v-for="p in items" :key="p.name" :xs="24" :md="12" class="plugin-col">
        <el-card class="plugin-card">
          <div class="plugin-head">
            <div>
              <div class="plugin-title">{{ p.title }} <span class="plugin-ver">v{{ p.version }}</span></div>
              <div class="plugin-name mono-val">{{ p.name }}</div>
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
            <el-button v-if="p.name === 'showcase' && p.enabled" size="small" text type="primary" @click="openShowcase">取展示代码</el-button>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-dialog v-model="showcaseVisible" title="对外展示" width="640px">
      <el-alert
        v-if="!embed.site_url"
        type="warning"
        :closable="false"
        show-icon
        title="还没有可用的站点地址"
        description="到设置里把站点地址写成 https://你的域名。不填的话，贴出去的图片指不回这台服务器。"
      />
      <el-alert
        v-else-if="!embed.https"
        type="warning"
        :closable="false"
        show-icon
        title="当前站点地址是 http"
        description="QRZ 简介是 https 页面，http 图片会被浏览器丢掉。博客如果也是 https，同样显示不出来。"
      />
      <template v-if="embed.html">
        <p class="embed-note">QRZ 简介和博客都会丢掉脚本和内嵌页，图片和链接留得住。下面这一段两边都能贴。数字大约 60 秒更新一次，不含地址、邮箱和备注。</p>
        <el-input :model-value="embed.html" type="textarea" readonly :rows="4" />
        <p class="embed-note">自己排版时直接 GET 下面的地址，不要加自定义请求头。</p>
        <el-input :model-value="embed.feed_url" readonly />
        <div class="embed-actions">
          <el-button size="small" @click="copyText(embed.html)">复制 HTML</el-button>
          <el-button size="small" @click="copyText(embed.feed_url)">复制 JSON 地址</el-button>
          <el-button v-if="!embed.https" size="small" @click="showcaseVisible = false; $router.push('/admin/settings')">去改成 https</el-button>
        </div>
      </template>
      <div v-else class="embed-actions">
        <el-button type="primary" @click="showcaseVisible = false; $router.push('/admin/settings')">去填写站点地址</el-button>
      </div>
    </el-dialog>

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
import { themeRegistry, applyEnabledThemes, reloadThemeCSS } from '../plugins/registry'
import { ElMessage } from 'element-plus'

const items = ref([])
const statsVisible = ref(false)
const statsRows = ref([])
const showcaseVisible = ref(false)
const embed = ref({ site_url: '', html: '', feed_url: '', https: false })

async function load() {
  const res = await api.get('/plugins')
  items.value = (res?.items || []).map(p => ({ ...p, _busy: false }))
  // 先把配色样式表拉新（新启用的主题此前可能没有对应的 [data-theme] 块），
  // 再把 <html data-theme> 对齐并按需加载装饰层。
  await reloadThemeCSS()
  await applyEnabledThemes(
    items.value.filter(p => p.kind === 'theme' && p.enabled).map(p => p.name)
  )
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

async function openShowcase() {
  const res = await api.get('/ext/showcase/embed')
  embed.value = res || { site_url: '', html: '', feed_url: '', https: false }
  showcaseVisible.value = true
}

async function copyText(text) {
  if (!text) return
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success('已复制')
  } catch {
    ElMessage.warning('复制失败，请手动选择文本')
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
.plugin-card { transition: all 0.22s cubic-bezier(0.4, 0, 0.2, 1); }
.plugin-card:hover { border-color: var(--el-color-primary); box-shadow: var(--qsl-shadow-2); }
.plugin-head { display: flex; align-items: flex-start; justify-content: space-between; gap: 12px; }
.plugin-title { color: var(--qsl-navy); font-size: 16px; font-weight: 750; }
.plugin-ver { color: var(--qsl-muted); font-size: 11px; font-weight: 500; }
.plugin-name { margin-top: 2px; color: var(--qsl-muted); font-size: 11px; letter-spacing: .04em; }
.plugin-desc { margin: 10px 0 14px; color: var(--qsl-muted); font-size: 13px; line-height: 1.7; min-height: 44px; }
.plugin-foot { display: flex; align-items: center; gap: 10px; }
.swatches { display: inline-flex; gap: 6px; }
.swatch { width: 18px; height: 18px; border-radius: 6px; border: 1px solid rgba(0,0,0,.08); box-shadow: 0 1px 2px rgba(0,0,0,0.06); }
.chart { display: flex; flex-direction: column; gap: 8px; }
.chart-row { display: flex; align-items: flex-end; gap: 10px; }
.chart-day { width: 44px; flex: none; color: var(--qsl-muted); font-size: 11px; text-align: right; }
.chart-bar-wrap { flex: 1; display: flex; align-items: flex-end; height: 70px; border-bottom: 1px solid var(--qsl-line); }
.chart-bar { width: 100%; background: var(--el-color-primary); border-radius: 4px 4px 0 0; min-height: 2px; }
.chart-val { flex: none; width: 170px; color: var(--qsl-muted); font-size: 11.5px; }
.embed-note { margin: 12px 0 8px; color: var(--qsl-muted); font-size: 13px; line-height: 1.7; }
.embed-actions { display: flex; gap: 8px; margin-top: 12px; }
</style>
