import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// ---------------------------------------------------------------------------
// Element Plus 按需引入（可选，默认关闭）
// ---------------------------------------------------------------------------
// 装了 `unplugin-vue-components` 就自动启用；没装就保持全量引入。
// 用 try/catch + 顶层 await 做运行时探测，保证「没装也能正常 build」，
// 不会因为少一个依赖把项目卡死。
//
//   开启：npm i -D unplugin-vue-components      然后重新 npm run build
//   回退：npm uninstall unplugin-vue-components 配置自动降级，无需改代码
//
// 开启后 main.js 里的 __EP_ON_DEMAND__ 会被 define 成 true，
// 于是不再 app.use(ElementPlus)，组件由 resolver 在编译期按需注入。
// 样式仍然整包引入（element-plus/dist/index.css），CSS 体积要等确认组件
// 全部解析正确之后再考虑拆分。
// ---------------------------------------------------------------------------
let Components = null
let ElementPlusResolver = null
try {
  ;({ default: Components } = await import('unplugin-vue-components/vite'))
  ;({ ElementPlusResolver } = await import('unplugin-vue-components/resolvers'))
} catch {
  /* 未安装：走全量引入 */
}
const onDemandEP = Boolean(Components && ElementPlusResolver)

export default defineConfig({
  plugins: [
    vue(),
    onDemandEP &&
      Components({
        resolvers: [ElementPlusResolver()],
        // 不生成 components.d.ts，项目没有 TypeScript
        dts: false,
      }),
  ].filter(Boolean),

  define: {
    // 构建期常量：main.js 据此决定是否 app.use(ElementPlus)
    __EP_ON_DEMAND__: JSON.stringify(onDemandEP),
  },

  server: {
    port: 3000,
    watch: {
      // 编辑器原子保存产生的临时文件（如 Home.vue~RFxxxxx.TMP）不触发 HMR
      ignored: ['**/*.TMP', '**/*.tmp'],
    },
    proxy: {
      '/api': {
        target: 'http://localhost:8000',
        changeOrigin: true,
      },
      '/uploads': {
        target: 'http://localhost:8000',
        changeOrigin: true,
      },
    },
  },

  build: {
    outDir: 'dist',
    assetsDir: 'assets',
    target: 'es2020',
    cssCodeSplit: true,
    // 单个 chunk 超过 600KB 才告警（Element Plus 本体较大，默认 500KB 会一直刷警告）
    chunkSizeWarningLimit: 600,
    rollupOptions: {
      output: {
        // 把不常变的依赖拆成独立 chunk：业务代码改动时这几个 chunk 的 hash 不变，
        // 浏览器缓存不会因一次小改动全部失效。
        manualChunks(id) {
          if (!id.includes('node_modules')) return
          if (id.includes('element-plus') || id.includes('@element-plus')) return 'element-plus'
          if (id.includes('/@vue/') || id.includes('/vue/') || id.includes('vue-router') || id.includes('pinia')) return 'vue'
          if (id.includes('axios')) return 'axios'
          if (id.includes('qrcode')) return 'qrcode'
          return 'vendor'
        },
      },
    },
  },
})
