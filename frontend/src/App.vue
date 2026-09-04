<template>
  <el-config-provider :locale="elLocale">
    <LangSwitch v-if="isPublicPage" />
    <router-view />
  </el-config-provider>
</template>

<script setup>
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { ElConfigProvider } from 'element-plus'
import zhCn from 'element-plus/es/locale/lang/zh-cn'
import en from 'element-plus/es/locale/lang/en'
import { locale } from './i18n'
import LangSwitch from './components/LangSwitch.vue'

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
</script>

<style>
body {
  margin: 0;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
}
</style>
