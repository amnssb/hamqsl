<template>
  <el-container class="layout-container">
    <transition name="nav-fade">
      <div v-if="isMobile && mobileNav" class="nav-mask" @click="mobileNav = false"></div>
    </transition>
    <el-aside :width="asideWidth" class="layout-aside" :class="{ 'is-mobile': isMobile, 'mobile-open': isMobile && mobileNav }">
      <div class="brand" :class="{ collapsed: !isMobile && isCollapse }" @click="toggleAside">
        <div class="brand-mark">Q</div>
        <div v-if="!(!isMobile && isCollapse)" class="brand-copy">
          <strong>QSL / HUB</strong>
          <span>卡片工作台</span>
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

      <div v-if="!(!isMobile && isCollapse)" class="nav-caption nav-caption-bottom">管理</div>
      <el-menu :default-active="route.path" router :collapse="!isMobile && isCollapse" class="side-menu">
        <el-menu-item index="/admin/address"><el-icon><Notebook /></el-icon><template #title>我的地址</template></el-menu-item>
        <el-menu-item index="/admin/station"><el-icon><Postcard /></el-icon><template #title>卡片版本</template></el-menu-item>
        <el-menu-item index="/admin/settings"><el-icon><Tools /></el-icon><template #title>设置</template></el-menu-item>
        <el-menu-item index="/admin/plugins"><el-icon><MagicStick /></el-icon><template #title>插件</template></el-menu-item>
      </el-menu>

      <div class="aside-footer" :class="{ collapsed: !isMobile && isCollapse }">
        <span class="status-dot"></span>
        <span v-if="!(!isMobile && isCollapse)">系统运行正常</span>
      </div>
    </el-aside>

    <el-container>
      <el-header class="layout-header">
        <div class="header-left">
          <el-button v-if="isMobile" class="nav-burger" text @click="mobileNav = true"><el-icon :size="22"><Menu /></el-icon></el-button>
          <div class="header-title">
            <span class="eyebrow">QSL MANAGEMENT</span>
            <el-breadcrumb separator="/">
              <el-breadcrumb-item :to="{ path: '/admin/dashboard' }">工作台</el-breadcrumb-item>
              <el-breadcrumb-item v-if="route.meta.title">{{ route.meta.title }}</el-breadcrumb-item>
            </el-breadcrumb>
          </div>
        </div>
        <div class="header-right">
          <div class="user-chip"><span class="avatar">{{ (auth.user?.display_name || auth.user?.username || 'A').slice(0, 1).toUpperCase() }}</span><span>{{ auth.user?.display_name || auth.user?.username }}</span></div>
          <el-button class="logout-btn" text @click="handleLogout">退出登录</el-button>
        </div>
      </el-header>
      <el-main class="layout-main"><div class="content-shell"><router-view /></div></el-main>
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

// 手机端抽屉导航：≤860px 侧栏变为固定定位抽屉，汉堡按钮唤出，路由跳转后自动收起
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
// 手机端点品牌区收起抽屉；桌面端保持原折叠逻辑
function toggleAside() {
  if (isMobile.value) mobileNav.value = false
  else isCollapse.value = !isCollapse.value
}

function handleLogout() { auth.logout(); router.push('/login') }
</script>

<style scoped>
.layout-container { min-height:100vh; background:var(--qsl-paper); }
.layout-aside { position:relative; display:flex; flex-direction:column; background:var(--qsl-navy); transition:width .2s ease; overflow:hidden; }
.brand { height:82px; display:flex; align-items:center; gap:12px; padding:0 22px; color:#fff; cursor:pointer; border-bottom:1px solid rgba(255,255,255,.12); }
.brand-mark { width:34px; height:34px; display:grid; place-items:center; flex:none; color:var(--qsl-navy); background:var(--qsl-yellow); font-size:20px; font-weight:850; }
.brand-copy { display:flex; flex-direction:column; line-height:1.15; white-space:nowrap; }
.brand-copy strong { font-size:15px; letter-spacing:.08em; }
.brand-copy span { margin-top:5px; color:#aebec8; font-size:11px; }
.nav-caption { padding:24px 22px 9px; color:#8096a3; font-size:10px; font-weight:800; letter-spacing:.16em; text-transform:uppercase; }
.nav-caption-bottom { padding-top:30px; }
.side-menu { border:0; background:transparent; }
:deep(.side-menu:not(.el-menu--collapse) .el-menu-item), :deep(.side-menu:not(.el-menu--collapse) .el-sub-menu__title) { height:45px; margin:2px 12px; padding-left:12px !important; color:#b9c9d1; border-radius:2px; }
:deep(.side-menu .el-menu-item:hover), :deep(.side-menu .el-sub-menu__title:hover) { color:#fff; background:var(--qsl-navy-soft); }
:deep(.side-menu .el-menu-item.is-active) { color:#fff; background:var(--qsl-orange); font-weight:700; }
:deep(.side-menu .el-menu) { background:#132632; }
:deep(.side-menu .el-menu .el-menu-item) { margin-left:28px; }
:deep(.side-menu .el-icon) { color:inherit; }
.aside-footer { display:flex; align-items:center; gap:8px; margin:auto 20px 22px; padding-top:16px; color:#91a6b1; border-top:1px solid rgba(255,255,255,.12); font-size:11px; white-space:nowrap; }
.aside-footer.collapsed { justify-content:center; margin:auto 8px 22px; }
.brand.collapsed { justify-content:center; padding:0; }
.brand.collapsed .brand-mark { margin:0; }
.status-dot { width:7px; height:7px; border-radius:50%; background:#54c6a2; box-shadow:0 0 0 3px rgba(84,198,162,.15); }
.layout-header { height:82px; display:flex; align-items:center; justify-content:space-between; padding:0 32px; background:#fff; border-bottom:1px solid var(--qsl-line); }
.header-left { display:flex; flex-direction:column; gap:7px; }
.eyebrow { color:var(--qsl-orange); font-size:10px; font-weight:800; letter-spacing:.16em; }
:deep(.el-breadcrumb__inner) { color:var(--qsl-muted); font-size:13px; }
.header-right { display:flex; align-items:center; gap:18px; }
.user-chip { display:flex; align-items:center; gap:9px; color:var(--qsl-navy); font-size:13px; font-weight:650; }
.avatar { display:grid; place-items:center; width:30px; height:30px; color:#fff; background:var(--qsl-orange); font-size:12px; }
.logout-btn { color:var(--qsl-muted); }
.layout-main { padding:32px; background:var(--qsl-paper); }
.content-shell { max-width:1440px; margin:0 auto; }
/* 手机端：抽屉侧栏 + 紧凑 header */
.nav-burger { margin-right: 8px; color: var(--qsl-navy); }
.header-title { display:flex; flex-direction:column; gap:7px; }
.nav-mask { position:fixed; inset:0; z-index:65; background:rgba(12,22,40,.45); }
.nav-fade-enter-active, .nav-fade-leave-active { transition:opacity .2s ease; }
.nav-fade-enter-from, .nav-fade-leave-to { opacity:0; }
@media (max-width:860px) {
  .layout-aside.is-mobile {
    position:fixed; top:0; bottom:0; left:0; z-index:70;
    width:264px !important;
    transform:translateX(-102%);
    transition:transform .22s ease;
  }
  .layout-aside.is-mobile.mobile-open { transform:translateX(0); box-shadow:8px 0 30px rgba(10,20,40,.35); }
  .layout-header { height:60px; padding:0 14px !important; }
  .header-left { flex-direction:row; align-items:center; }
  .eyebrow { display:none; }
  .user-chip span:last-child { display:none; }
  .layout-main { padding:16px 12px !important; }
}
</style>

<style>
/* 折叠态：子菜单弹层 teleport 到 body，需全局样式；菜单项收起时居中对齐 */
/* EP 折叠菜单默认 64px 宽，与 72px 侧栏不一致导致图标偏左；统一为 72px 并去掉左右边距 */
.el-menu--collapse.el-menu { border-right: 0; }
.side-menu.el-menu--collapse { width: 72px; }
.side-menu.el-menu--collapse .el-menu-item,
.side-menu.el-menu--collapse .el-sub-menu__title { margin: 2px 0; padding: 0 !important; justify-content: center; }
.el-popper.is-light .el-menu--vertical .el-menu-item { margin: 2px 10px; border-radius: 2px; }
</style>
