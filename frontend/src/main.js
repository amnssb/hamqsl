import { createApp } from 'vue'
import { createPinia } from 'pinia'
import ElementPlus, { ElLoading } from 'element-plus'
import 'element-plus/dist/index.css'
import './styles/tokens.css'
import './styles/theme.css'
import './styles/public.css'
import zhCn from 'element-plus/dist/locale/zh-cn.mjs'
import App from './App.vue'
import router from './router'
import { registerIcons } from './plugins/icons'

// __EP_ON_DEMAND__ 是 vite.config.js 在构建期注入的常量：
// 装了 unplugin-vue-components 时为 true（组件按需引入），否则为 false（全量引入）。
// 用 typeof 兜底，万一没有经过 Vite 处理也不会 ReferenceError。
const ON_DEMAND_EP = typeof __EP_ON_DEMAND__ !== 'undefined' && __EP_ON_DEMAND__

const app = createApp(App)
app.use(createPinia())
app.use(router)

if (!ON_DEMAND_EP) {
  // 全量注册 Element Plus（默认路径）
  app.use(ElementPlus, { locale: zhCn })
}

// v-loading 是自定义指令，不走模板组件解析，按需模式下必须手动注册。
// 全量模式下 app.use(ElementPlus) 已经注册过，重复注册会告警，所以只在按需模式注册。
if (ON_DEMAND_EP) {
  app.directive('loading', ElLoading.directive)
}

// 只注册用到的图标（30 个），不再全量导入 293 个
registerIcons(app)

app.mount('#app')
