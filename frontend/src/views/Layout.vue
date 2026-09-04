<template>
  <el-container class="layout-container">
    <el-aside :width="isCollapse ? '72px' : '248px'" class="layout-aside">
      <div class="brand" :class="{ collapsed: isCollapse }" @click="isCollapse = !isCollapse">
        <div class="brand-mark">Q</div>
        <div v-if="!isCollapse" class="brand-copy">
          <strong>QSL / HUB</strong>
          <span>卡片工作台</span>
        </div>
      </div>

      <div v-if="!isCollapse" class="nav-caption">工作区</div>
      <el-menu :default-active="route.path" router :collapse="isCollapse" class="side-menu">
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

      <div v-if="!isCollapse" class="nav-caption nav-caption-bottom">管理</div>
      <el-menu :default-active="route.path" router :collapse="isCollapse" class="side-menu">
        <el-menu-item index="/admin/address"><el-icon><Notebook /></el-icon><template #title>我的地址</template></el-menu-item>
        <el-menu-item index="/admin/station"><el-icon><Postcard /></el-icon><template #title>卡片版本</template></el-menu-item>
        <el-menu-item index="/admin/settings"><el-icon><Tools /></el-icon><template #title>设置</template></el-menu-item>
      </el-menu>

      <div class="aside-footer" :class="{ collapsed: isCollapse }">
        <span class="status-dot"></span>
        <span v-if="!isCollapse">系统运行正常</span>
      </div>
    </el-aside>

    <el-container>
      <el-header class="layout-header">
        <div class="header-left">
          <span class="eyebrow">QSL MANAGEMENT</span>
          <el-breadcrumb separator="/">
            <el-breadcrumb-item :to="{ path: '/admin/dashboard' }">工作台</el-breadcrumb-item>
            <el-breadcrumb-item v-if="route.meta.title">{{ route.meta.title }}</el-breadcrumb-item>
          </el-breadcrumb>
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
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const isCollapse = ref(false)
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
@media (max-width:760px) { .layout-header { padding:0 18px; } .eyebrow { display:none; } .user-chip span:last-child { display:none; } .layout-main { padding:22px 16px; } }
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
