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
            <div class="eyebrow">HAM RADIO STATION</div>
            <el-breadcrumb separator="/">
              <el-breadcrumb-item :to="{ path: '/admin/dashboard' }">工作台</el-breadcrumb-item>
              <el-breadcrumb-item v-if="route.meta.title">{{ route.meta.title }}</el-breadcrumb-item>
            </el-breadcrumb>
          </div>
        </div>
        <div class="header-right">
          <div class="user-chip">
            <span class="avatar">{{ (auth.user?.display_name || auth.user?.username || 'A').slice(0, 1).toUpperCase() }}</span>
            <span class="user-name">{{ auth.user?.display_name || auth.user?.username }}</span>
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
</script>

<style scoped>
.layout-container { min-height: 100vh; background: var(--qsl-paper, #f8fafc); }
.layout-aside {
  position: relative;
  display: flex;
  flex-direction: column;
  background: #0b132b;
  border-right: 1px solid rgba(255, 255, 255, 0.08);
  transition: width .25s cubic-bezier(0.4, 0, 0.2, 1);
  overflow: hidden;
  box-shadow: 2px 0 12px rgba(0, 0, 0, 0.08);
}
.brand {
  height: 76px;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 0 20px;
  color: #fff;
  cursor: pointer;
  border-bottom: 1px solid rgba(255, 255, 255, 0.07);
  transition: background .2s;
}
.brand:hover { background: rgba(255, 255, 255, 0.03); }
.brand-mark {
  width: 38px;
  height: 38px;
  display: grid;
  place-items: center;
  flex: none;
  background: linear-gradient(135deg, #3b82f6 0%, #2563eb 100%);
  border-radius: 10px;
  box-shadow: 0 4px 12px rgba(37, 99, 235, 0.35);
}
.brand-glyph {
  color: #ffffff;
  font-size: 20px;
  font-weight: 850;
  letter-spacing: -0.02em;
}
.brand-copy { display: flex; flex-direction: column; line-height: 1.2; white-space: nowrap; }
.brand-title {
  font-size: 16px;
  font-weight: 700;
  letter-spacing: 0.06em;
  color: #f8fafc;
}
.brand-title strong {
  color: #60a5fa;
  font-weight: 850;
  margin-left: 2px;
}
.brand-sub { margin-top: 3px; color: #64748b; font-size: 10px; letter-spacing: 0.1em; font-weight: 600; }

.nav-caption {
  padding: 20px 20px 8px;
  color: #475569;
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
  color: #94a3b8;
  border-radius: 8px;
  font-size: 13.5px;
  font-weight: 550;
  transition: all 0.18s ease;
}
:deep(.side-menu .el-menu-item:hover),
:deep(.side-menu .el-sub-menu__title:hover) {
  color: #ffffff;
  background: rgba(255, 255, 255, 0.07);
}
:deep(.side-menu .el-menu-item.is-active) {
  color: #ffffff;
  background: linear-gradient(135deg, #3b82f6 0%, #2563eb 100%);
  box-shadow: 0 4px 10px rgba(37, 99, 235, 0.35);
  font-weight: 650;
}
:deep(.side-menu .el-menu) { background: #070d1f; border-radius: 8px; margin: 2px 8px; padding: 4px 0; }
:deep(.side-menu .el-menu .el-menu-item) { margin-left: 16px; height: 38px; }
:deep(.side-menu .el-icon) { color: inherit; font-size: 16px; margin-right: 8px; }

.aside-footer {
  display: flex;
  align-items: center;
  gap: 10px;
  margin: auto 16px 18px;
  padding: 12px 14px;
  color: #64748b;
  background: rgba(255, 255, 255, 0.04);
  border-radius: 8px;
  font-size: 12px;
  white-space: nowrap;
}
.aside-footer.collapsed { justify-content: center; margin: auto 8px 18px; padding: 10px 0; }
.brand.collapsed { justify-content: center; padding: 0; }

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #10b981;
  box-shadow: 0 0 0 3px rgba(16, 185, 129, 0.25);
  animation: pulse-dot 2.5s infinite;
}
@keyframes pulse-dot {
  0%, 100% { box-shadow: 0 0 0 3px rgba(16, 185, 129, 0.25); }
  50% { box-shadow: 0 0 0 6px rgba(16, 185, 129, 0.1); }
}

.main-container {
  min-height: 100vh;
  background: var(--qsl-paper, #f8fafc);
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
  background: rgba(255, 255, 255, 0.88);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  border-bottom: 1px solid #e2e8f0;
  position: sticky;
  top: 0;
  z-index: 20;
}
.header-left { display: flex; flex-direction: column; gap: 4px; }
.eyebrow {
  color: #3b82f6;
  font-size: 10px;
  font-weight: 750;
  letter-spacing: 0.14em;
}
:deep(.el-breadcrumb__inner) { color: #64748b; font-size: 13.5px; font-weight: 500; }
:deep(.el-breadcrumb__inner.is-link:hover) { color: #2563eb; }
:deep(.el-breadcrumb__item:last-child .el-breadcrumb__inner) { color: #0f172a; font-weight: 650; }

.header-right { display: flex; align-items: center; gap: 16px; }
.user-chip {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 4px 10px 4px 4px;
  background: #f1f5f9;
  border-radius: 9999px;
  border: 1px solid #e2e8f0;
  color: #0f172a;
  font-size: 13px;
  font-weight: 600;
}
.avatar {
  display: grid;
  place-items: center;
  width: 28px;
  height: 28px;
  color: #fff;
  background: linear-gradient(135deg, #3b82f6 0%, #2563eb 100%);
  border-radius: 50%;
  font-size: 12px;
  font-weight: 700;
}
.logout-btn {
  color: #64748b;
  font-size: 13px;
  padding: 6px 12px;
  border-radius: 8px;
}
.logout-btn:hover { color: #ef4444; background: #fef2f2; }
.logout-text { margin-left: 4px; }

.layout-main {
  flex: 1;
  padding: 28px 32px;
  background: var(--qsl-paper, #f8fafc);
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

.nav-burger { margin-right: 8px; color: #0f172a; }
.header-title { display: flex; flex-direction: column; gap: 4px; }
.nav-mask { position: fixed; inset: 0; z-index: 65; background: rgba(15, 23, 42, 0.5); backdrop-filter: blur(4px); }
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
