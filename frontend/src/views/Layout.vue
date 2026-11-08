<template>
  <el-container class="layout-container">
    <transition name="nav-fade">
      <div v-if="isMobile && mobileNav" class="nav-mask" @click="mobileNav = false"></div>
    </transition>
    <el-aside :width="asideWidth" class="layout-aside" :class="{ 'is-mobile': isMobile, 'mobile-open': isMobile && mobileNav }">
      <div class="brand" :class="{ collapsed: !isMobile && isCollapse }" @click="toggleAside">
        <div class="brand-mark">
          <span class="brand-glyph">Q</span>
        </div>
        <div v-if="!(!isMobile && isCollapse)" class="brand-copy">
          <div class="brand-title">QSL<strong>HUB</strong></div>
          <span class="brand-sub">HAM RADIO STATION</span>
        </div>
      </div>

      <div v-if="!(!isMobile && isCollapse)" class="nav-caption">工作区</div>
      <el-menu :default-active="route.path" router :collapse="!isMobile && isCollapse" class="side-menu">
        <el-menu-item index="/admin/dashboard"><el-icon><DataBoard /></el-icon><template #title>总览</template></el-menu-item>
        <el-sub-menu index="qsl-biz">
          <template #title><el-icon><Document /></el-icon><span>通联业务</span></template>
          <el-menu-item index="/admin/qso">通联日志</el-menu-item>
          <el-menu-item index="/admin/cards">卡片记录</el-menu-item>
        </el-sub-menu>
        <el-sub-menu index="exchange-biz">
          <template #title><el-icon><Switch /></el-icon><span>换卡业务</span></template>
          <el-menu-item index="/admin/exchange/online">线上换卡</el-menu-item>
          <el-menu-item index="/admin/exchange/offline">线下换卡</el-menu-item>
        </el-sub-menu>
        <el-menu-item index="/admin/receive"><el-icon><Box /></el-icon><template #title>收卡记录</template></el-menu-item>
      </el-menu>

      <div v-if="!(!isMobile && isCollapse)" class="nav-caption nav-caption-bottom">系统管理</div>
      <el-menu :default-active="route.path" router :collapse="!isMobile && isCollapse" class="side-menu">
        <el-menu-item index="/admin/address"><el-icon><Notebook /></el-icon><template #title>我的地址</template></el-menu-item>
        <el-menu-item index="/admin/station"><el-icon><Postcard /></el-icon><template #title>卡片版本</template></el-menu-item>
        <el-menu-item index="/admin/settings"><el-icon><Tools /></el-icon><template #title>设置</template></el-menu-item>
        <el-menu-item index="/admin/plugins"><el-icon><MagicStick /></el-icon><template #title>主题与插件</template></el-menu-item>
      </el-menu>

      <div class="aside-footer" :class="{ collapsed: !isMobile && isCollapse }">
        <span class="status-dot"></span>
        <span v-if="!(!isMobile && isCollapse)" class="status-text">系统正常运行</span>
      </div>
    </el-aside>

    <el-container class="main-container">
      <el-header class="layout-header">
        <div class="header-left">
          <el-button v-if="isMobile" class="nav-burger" text @click="mobileNav = true"><el-icon :size="20"><Menu /></el-icon></el-button>
          <div class="header-title">
            <div class="eyebrow"><span class="pulse-indicator"></span>HAM RADIO STATION WORKBENCH</div>
            <el-breadcrumb separator="/">
              <el-breadcrumb-item :to="{ path: '/admin/dashboard' }">工作台</el-breadcrumb-item>
              <el-breadcrumb-item v-if="route.meta.title">{{ route.meta.title }}</el-breadcrumb-item>
            </el-breadcrumb>
          </div>
        </div>
        <div class="header-right">
          <el-button class="portal-link-btn" text @click="openPortal">
            <el-icon><Promotion /></el-icon>
            <span class="portal-link-text">公开门户</span>
          </el-button>
          <div class="user-chip">
            <span class="avatar">{{ (auth.user?.display_name || auth.user?.username || 'A').slice(0, 1).toUpperCase() }}</span>
            <span class="user-name">{{ auth.user?.display_name || auth.user?.username }}</span>
            <span class="status-online-dot"></span>
          </div>
          <el-button class="logout-btn" text @click="handleLogout">
            <el-icon><SwitchButton /></el-icon>
            <span class="logout-text">退出</span>
          </el-button>
        </div>
      </el-header>
      <el-main class="layout-main">
        <div class="content-shell">
          <router-view />
        </div>
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup>
import { ref, computed, watch, onMounted, onBeforeUnmount } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const isCollapse = ref(false)

const isMobile = ref(false)
const mobileNav = ref(false)
function syncMobile() {
  isMobile.value = window.innerWidth <= 860
  if (!isMobile.value) mobileNav.value = false
}
onMounted(() => { syncMobile(); window.addEventListener('resize', syncMobile) })
onBeforeUnmount(() => window.removeEventListener('resize', syncMobile))
watch(() => route.path, () => { mobileNav.value = false })

const asideWidth = computed(() => (isMobile.value ? '264px' : (isCollapse.value ? '72px' : '248px')))
function toggleAside() {
  if (isMobile.value) mobileNav.value = false
  else isCollapse.value = !isCollapse.value
}

function handleLogout() { auth.logout(); router.push('/login') }
function openPortal() { window.open('/', '_blank') }
</script>

<style scoped>
/* 侧栏/品牌区用的是 --qsl-chrome-*（深色"外壳"），与页面正体的 --qsl-* 分开：
   这样玻璃/纸墨这类"深色侧栏 + 浅色正文"的主题不需要再硬编码侧栏文字颜色。 */
.layout-container { min-height: 100vh; background: var(--qsl-bg); }
.layout-aside {
  position: relative;
  display: flex;
  flex-direction: column;
  background: var(--qsl-chrome-bg);
  border-right: 1px solid color-mix(in srgb, var(--qsl-chrome-text) 10%, transparent);
  transition: width .25s cubic-bezier(0.4, 0, 0.2, 1);
  overflow: hidden;
  box-shadow: 2px 0 12px color-mix(in srgb, var(--qsl-chrome-bg) 30%, transparent);
}
.brand {
  height: 76px;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 0 20px;
  color: var(--qsl-chrome-text);
  cursor: pointer;
  border-bottom: 1px solid color-mix(in srgb, var(--qsl-chrome-text) 8%, transparent);
  transition: background .2s;
}
.brand:hover { background: color-mix(in srgb, var(--qsl-chrome-text) 5%, transparent); }
.brand-mark {
  width: 38px;
  height: 38px;
  display: grid;
  place-items: center;
  flex: none;
  background: var(--qsl-primary);
  border-radius: 10px;
  box-shadow: 0 4px 12px color-mix(in srgb, var(--qsl-ink) 25%, transparent);
}
.brand-glyph {
  color: var(--qsl-on-primary);
  font-size: 20px;
  font-weight: 850;
  letter-spacing: -0.02em;
}
.brand-copy { display: flex; flex-direction: column; line-height: 1.2; white-space: nowrap; }
.brand-title {
  font-size: 16px;
  font-weight: 700;
  letter-spacing: 0.06em;
  color: var(--qsl-chrome-text);
}
.brand-title strong {
  color: var(--qsl-poster-accent-2);
  font-weight: 850;
  margin-left: 2px;
}
.brand-sub { margin-top: 3px; color: var(--qsl-chrome-muted); font-size: 10px; letter-spacing: 0.1em; font-weight: 600; }

.nav-caption {
  padding: 20px 20px 8px;
  color: var(--qsl-chrome-muted);
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.12em;
  text-transform: uppercase;
}
.nav-caption-bottom { padding-top: 24px; }
.side-menu { border: 0; background: transparent; }

:deep(.side-menu:not(.el-menu--collapse) .el-menu-item),
:deep(.side-menu:not(.el-menu--collapse) .el-sub-menu__title) {
  height: 42px;
  margin: 3px 10px;
  padding-left: 12px !important;
  color: var(--qsl-chrome-muted);
  border-radius: var(--qsl-radius);
  font-size: 13.5px;
  font-weight: 550;
  transition: all 0.18s ease;
}
:deep(.side-menu .el-menu-item:hover),
:deep(.side-menu .el-sub-menu__title:hover) {
  color: var(--qsl-chrome-text);
  background: color-mix(in srgb, var(--qsl-chrome-text) 8%, transparent);
}
:deep(.side-menu .el-menu-item.is-active) {
  color: var(--qsl-chrome-text);
  background: color-mix(in srgb, var(--qsl-primary) 22%, transparent);
  box-shadow: inset 3px 0 0 var(--qsl-primary);
  font-weight: 650;
}
:deep(.side-menu .el-menu) {
  background: color-mix(in srgb, var(--qsl-chrome-bg) 82%, black);
  border-radius: var(--qsl-radius);
  margin: 2px 8px;
  padding: 4px 0;
}
:deep(.side-menu .el-menu .el-menu-item) { margin-left: 16px; height: 38px; }
:deep(.side-menu .el-icon) { color: inherit; font-size: 16px; margin-right: 8px; }

.aside-footer {
  display: flex;
  align-items: center;
  gap: 10px;
  margin: auto 16px 18px;
  padding: 12px 14px;
  color: var(--qsl-chrome-muted);
  background: color-mix(in srgb, var(--qsl-chrome-text) 5%, transparent);
  border-radius: var(--qsl-radius);
  font-size: 12px;
  white-space: nowrap;
}
.aside-footer.collapsed { justify-content: center; margin: auto 8px 18px; padding: 10px 0; }
.brand.collapsed { justify-content: center; padding: 0; }

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--qsl-green);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--qsl-green) 25%, transparent);
  animation: pulse-dot 2.5s infinite;
}
@keyframes pulse-dot {
  0%, 100% { box-shadow: 0 0 0 3px color-mix(in srgb, var(--qsl-green) 25%, transparent); }
  50% { box-shadow: 0 0 0 6px color-mix(in srgb, var(--qsl-green) 10%, transparent); }
}

.main-container {
  min-height: 100vh;
  background:
    radial-gradient(at 0% 0%, color-mix(in srgb, var(--qsl-primary) 7%, transparent), transparent 46%),
    var(--qsl-bg);
  min-width: 0;
  display: flex;
  flex-direction: column;
}
.layout-header {
  height: 72px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 32px;
  background: color-mix(in srgb, var(--qsl-panel) 84%, transparent);
  backdrop-filter: blur(14px);
  border-bottom: 1px solid var(--qsl-line);
  position: sticky;
  top: 0;
  z-index: 20;
}
.header-left { display: flex; flex-direction: column; gap: 4px; }
.eyebrow {
  color: var(--qsl-primary);
  font-size: 10px;
  font-weight: 750;
  letter-spacing: 0.14em;
  display: flex;
  align-items: center;
  gap: 6px;
}
:deep(.el-breadcrumb__inner) { color: var(--qsl-muted); font-size: 13.5px; font-weight: 500; }
:deep(.el-breadcrumb__inner.is-link:hover) { color: var(--qsl-primary); }
:deep(.el-breadcrumb__item:last-child .el-breadcrumb__inner) { color: var(--qsl-heading); font-weight: 650; }

.header-right { display: flex; align-items: center; gap: 14px; }
.portal-link-btn {
  color: var(--qsl-muted);
  font-size: 13px;
  font-weight: 600;
  padding: 6px 12px;
  border-radius: var(--qsl-radius);
  border: 1px solid var(--qsl-line);
  background: var(--qsl-panel);
  transition: all 0.2s ease;
}
.portal-link-btn:hover {
  color: var(--qsl-primary);
  border-color: var(--qsl-primary);
  background: var(--qsl-hover-tint);
}
.portal-link-text { margin-left: 5px; }

.user-chip {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 4px 12px 4px 4px;
  background: var(--qsl-panel);
  border-radius: 9999px;
  border: 1px solid var(--qsl-line);
  color: var(--qsl-ink);
  font-size: 13px;
  font-weight: 600;
  box-shadow: 0 1px 2px color-mix(in srgb, var(--qsl-ink) 4%, transparent);
}
.status-online-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--qsl-green);
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--qsl-green) 30%, transparent);
}
.avatar {
  display: grid;
  place-items: center;
  width: 28px;
  height: 28px;
  color: var(--qsl-on-primary);
  background: var(--qsl-primary);
  border-radius: 50%;
  font-size: 12px;
  font-weight: 700;
}
.logout-btn {
  color: var(--qsl-muted);
  font-size: 13px;
  padding: 6px 12px;
  border-radius: var(--qsl-radius);
}
.logout-btn:hover {
  color: var(--qsl-danger);
  background: color-mix(in srgb, var(--qsl-danger) 10%, transparent);
}
.logout-text { margin-left: 4px; }

.layout-main {
  flex: 1;
  padding: 28px 32px;
  background: transparent;
  min-width: 0;
  overflow-x: hidden;
  padding-bottom: max(28px, env(safe-area-inset-bottom));
}
.content-shell {
  max-width: 1440px;
  margin: 0 auto;
  width: 100%;
  min-width: 0;
}

.nav-burger { margin-right: 8px; color: var(--qsl-ink); }
.header-title { display: flex; flex-direction: column; gap: 4px; }
.nav-mask {
  position: fixed; inset: 0; z-index: 65;
  background: color-mix(in srgb, var(--qsl-ink) 50%, transparent);
  backdrop-filter: blur(4px);
}
.nav-fade-enter-active, .nav-fade-leave-active { transition: opacity .2s ease; }
.nav-fade-enter-from, .nav-fade-leave-to { opacity: 0; }

@media (min-width: 641px) and (max-width: 1024px) {
  .layout-main { padding: 20px 20px !important; }
  .layout-header { padding: 0 20px !important; }
}

@media (max-width: 860px) {
  .layout-aside.is-mobile {
    position: fixed; top: 0; bottom: 0; left: 0; z-index: 70;
    width: 264px !important;
    transform: translateX(-102%);
    transition: transform .25s cubic-bezier(0.4, 0, 0.2, 1);
  }
  .layout-aside.is-mobile.mobile-open { transform: translateX(0); box-shadow: 12px 0 32px rgba(0, 0, 0, 0.3); }
  .layout-header { height: 60px; padding: 0 14px !important; }
  .header-left { flex-direction: row; align-items: center; }
  .eyebrow { display: none; }
  .user-name { display: none; }
  .logout-text { display: none; }
  .layout-main { padding: 16px 12px !important; padding-bottom: max(16px, env(safe-area-inset-bottom)) !important; }
}
</style>

<style>
.el-menu--collapse.el-menu { border-right: 0; }
.side-menu.el-menu--collapse { width: 72px; }
.side-menu.el-menu--collapse .el-menu-item,
.side-menu.el-menu--collapse .el-sub-menu__title { margin: 4px 8px; padding: 0 !important; justify-content: center; border-radius: 8px; }
.el-popper.is-light .el-menu--vertical .el-menu-item { margin: 2px 8px; border-radius: 6px; }
</style>
