<template>
  <el-config-provider :locale="elLocale">
    <LangSwitch v-if="isPublicPage" />
    <router-view />
  </el-config-provider>
</template>

<script setup>
import { computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { ElConfigProvider } from 'element-plus'
import zhCn from 'element-plus/es/locale/lang/zh-cn'
import en from 'element-plus/es/locale/lang/en'
import { locale } from './i18n'
import LangSwitch from './components/LangSwitch.vue'
import api from './api'
import { applyEnabledThemes } from './plugins/registry'

const route = useRoute()

// 后台页面始终中文组件文案；公开页跟随 i18n 语言（日期选择器/分页/弹窗按钮等内置文案）
const elLocale = computed(() => {
  const p = route.path || ''
  if (p.startsWith('/admin') || p.startsWith('/login')) return zhCn
  return locale.current === 'en' ? en : zhCn
})

// 公开门户页显示语言切换器（后台不显示）
const isPublicPage = computed(() => {
  const p = route.path || ''
  return !(p.startsWith('/admin') || p.startsWith('/login'))
})

// 主题
//   · 配色（--p-* / --el-*）已经由 index.html 里的
//     <link rel="stylesheet" href="/api/public/theme.css"> 在首帧应用，这里不再碰颜色。
//   · 这里只做两件事：把 <html data-theme> 对齐到服务端当前主题，
//     并按需加载该主题的装饰层（Layer 3）。
//   · 拉取失败时什么都不做 —— 默认主题本来就已经生效。
onMounted(async () => {
  try {
    const res = await api.get('/public/plugins')
    await applyEnabledThemes((res?.items || []).map(i => i.name))
  } catch { /* 保持服务端下发的配色 */ }
})
</script>

<style>
body {
  margin: 0;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
}
</style>
