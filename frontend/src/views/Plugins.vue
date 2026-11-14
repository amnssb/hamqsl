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
            <!-- 插件自己的界面：pages/<插件名>.vue 存在且已启用才挂载。文件删掉按钮就消失，这里不用改 -->
            <component :is="pluginPages[p.name]" v-if="p.enabled" :plugin="p" />
          </div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup>
/*
插件页。列表、开关来自 /api/plugins；每个插件的按钮、弹窗是它自己的文件：
frontend/src/plugins/pages/<插件名>.vue（没有就没有界面，加文件即出现，不用改这里）。
主题的配色/纹理逻辑在 ../plugins/registry.js，本页只在启停后触发它。
*/
import { ref, onMounted, defineAsyncComponent } from 'vue'
import api from '../api'
import { applyEnabledThemes, reloadThemeCSS } from '../plugins/registry'
import { ElMessage } from 'element-plus'

// 按文件名把插件界面扫进来，包成异步组件并缓存：包装只做一次，
// 渲染期反复新建组件会把打开中的弹窗状态丢掉。
// 键是相对本文件的路径，取文件段就是插件名。
const pageFiles = import.meta.glob('../plugins/pages/*.vue')
const pluginPages = {}
for (const [path, load] of Object.entries(pageFiles)) {
  const name = path.split('/').pop().replace(/\.vue$/, '')
  pluginPages[name] = defineAsyncComponent(load)
}

const items = ref([])

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
  return p.swatches && p.swatches.length ? p.swatches : null // 色点随插件清单一起来，不再在前端单独记一份
}

async function toggle(p, on) {
  p._busy = true
  try {
    await api.post(`/plugins/${p.name}/${on ? 'enable' : 'disable'}`)
    ElMessage.success(on ? `「${p.title}」已启用，即时生效` : `「${p.title}」已禁用`)
    await load()
  } catch {
    // 拦截器已统一提示错误；启用失败（Init 出错）后端已回退开关，这里刷新即可
    await load()
  } finally {
    p._busy = false
  }
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
</style>
