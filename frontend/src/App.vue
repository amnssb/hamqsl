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

// 应用已启用的主题插件（公开端点，无需登录；失败静默不影响功能）
onMounted(async () => {
  try {
    const res = await api.get('/public/plugins')
    applyEnabledThemes((res?.items || []).map(i => i.name))
  } catch { /* 主题应用失败时保持默认外观 */ }
})
</script>

<style>
body {
  margin: 0;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
}
</style>
