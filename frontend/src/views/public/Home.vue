<template>
  <div class="portal-page">
    <header class="portal-header">
      <div class="portal-brand"><span class="brand-mark">Q</span><div><strong>{{ siteName }}</strong><small>{{ t('home.brandSubtitle') }}</small></div></div>
      <el-button class="admin-link" text @click="$router.push('/login')">{{ t('home.adminLink') }} <el-icon><ArrowRight /></el-icon></el-button>
    </header>

    <main class="portal-main">
      <!-- 站点公告（管理员在设置中配置，留空不显示） -->
      <section v-if="siteNotice" class="site-notice">
        <span class="notice-badge"><BellFilled /></span>
        <div class="notice-body">
          <b>{{ t('home.noticeBadge') }}</b>
          <pre>{{ siteNotice }}</pre>
        </div>
      </section>

      <!-- 首屏：左文案 + 右 QSL 卡片样本 -->
      <section class="hero">
        <div class="hero-copy">
          <span class="eyebrow">PUBLIC QSL PORTAL</span>
          <h1>{{ t('home.heroTitle1') }}<br><em>{{ t('home.heroTitle2') }}</em></h1>
          <p>{{ t('home.heroDesc') }}</p>
          <div class="hero-actions">
            <el-button type="primary" size="large" @click="$router.push('/apply')">{{ t('home.btnApply') }} <el-icon><ArrowRight /></el-icon></el-button>
            <el-button size="large" @click="$router.push('/status')">{{ t('home.btnStatus') }}</el-button>
          </div>
        </div>

        <!-- 系统实时数据（公开聚合统计） -->
        <div v-if="statsLoaded" class="stats-panel">
          <div class="stats-head"><span class="stats-dot"></span>{{ t('home.statsTitle') }}</div>
          <div class="stats-grid">
            <div class="stat-item"><b>{{ stats.cards_sent }}</b><span>{{ t('home.statsSent') }}</span></div>
            <div class="stat-item"><b>{{ stats.cards_signed }}</b><span>{{ t('home.statsSigned') }}</span></div>
            <div class="stat-item"><b>{{ stats.pending_requests }}</b><span>{{ t('home.statsPending') }}</span></div>
          </div>
          <p class="stats-note">{{ t('home.statsNote') }}</p>
        </div>
      </section>

      <!-- 服务入口 -->
      <section class="services">
        <button class="service-card" @click="$router.push('/apply')">
          <span class="service-icon orange"><Promotion /></span>
          <span class="service-text"><b>{{ t('home.svcApply') }}</b><small>{{ t('home.svcApplyDesc') }}</small></span>
          <el-icon class="service-arrow"><ArrowRight /></el-icon>
        </button>
        <button class="service-card" @click="$router.push('/status')">
          <span class="service-icon blue"><Search /></span>
          <span class="service-text"><b>{{ t('home.svcStatus') }}</b><small>{{ t('home.svcStatusDesc') }}</small></span>
          <el-icon class="service-arrow"><ArrowRight /></el-icon>
        </button>
        <button class="service-card" @click="$router.push('/track')">
          <span class="service-icon green"><Van /></span>
          <span class="service-text"><b>{{ t('home.svcTrack') }}</b><small>{{ t('home.svcTrackDesc') }}</small></span>
          <el-icon class="service-arrow"><ArrowRight /></el-icon>
        </button>
      </section>
    </main>

    <footer>{{ siteName }} · {{ t('home.footerTagline') }}</footer>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import api from '../../api'
import { t } from '../../i18n'

const siteName = ref('QSL / HUB')
const siteNotice = ref('')
const stats = reactive({ cards_sent: 0, cards_signed: 0, pending_requests: 0 })
const statsLoaded = ref(false)
onMounted(async () => {
  try {
    const res = await api.get('/public/site-info')
    if (res?.site_name) {
      siteName.value = res.site_name
      document.title = res.site_name
    }
    siteNotice.value = (res && res.site_notice) || ''
  } catch (e) { /* 使用默认名称 */ }
  try {
    const s = await api.get('/public/stats')
    if (s) {
      stats.cards_sent = s.cards_sent ?? 0
      stats.cards_signed = s.cards_signed ?? 0
      stats.pending_requests = s.pending_requests ?? 0
      statsLoaded.value = true
    }
  } catch (e) { /* 统计面板缺省隐藏 */ }
})
</script>

<style scoped>
.portal-page { min-height:100vh; display:flex; flex-direction:column; color:var(--qsl-ink); background:var(--qsl-paper); }
.portal-header { height:64px; display:flex; align-items:center; justify-content:space-between; max-width:1080px; width:100%; margin:0 auto; padding:0 24px; border-bottom:1px solid var(--qsl-line); }
.portal-brand { display:flex; align-items:center; gap:11px; }
.brand-mark { display:grid; place-items:center; width:32px; height:32px; color:var(--qsl-navy); background:var(--qsl-yellow); font-size:18px; font-weight:850; }
.portal-brand strong { display:block; color:var(--qsl-navy); font-size:13px; letter-spacing:.1em; }
.portal-brand small { display:block; margin-top:3px; color:var(--qsl-muted); font-size:10px; }
.admin-link { color:var(--qsl-navy); }

.portal-main { flex:1; width:100%; max-width:1080px; margin:0 auto; padding:36px 24px 44px; }

/* 公告条 */
.site-notice { display:flex; gap:12px; align-items:flex-start; margin-bottom:30px; padding:13px 16px; background:#fdf6e8; border:1px solid #ecd9b0; border-left:4px solid var(--qsl-orange); }
.notice-badge { display:grid; place-items:center; flex:none; width:24px; height:24px; color:var(--qsl-navy); background:var(--qsl-yellow); font-size:13px; }
.notice-body b { display:block; color:var(--qsl-navy); font-size:12px; letter-spacing:.08em; }
.notice-body pre { margin:5px 0 0; font-family:inherit; font-size:12.5px; line-height:1.75; white-space:pre-wrap; word-break:break-all; color:var(--qsl-ink); }

/* 首屏双栏 */
.hero { display:grid; grid-template-columns:1.05fr .95fr; gap:44px; align-items:center; padding:22px 0 50px; }
.eyebrow { color:var(--qsl-orange); font-size:11px; font-weight:800; letter-spacing:.18em; }
.hero-copy h1 { margin:14px 0 16px; color:var(--qsl-navy); font-size:clamp(28px,4vw,40px); line-height:1.18; letter-spacing:-.02em; }
.hero-copy h1 em { color:var(--qsl-orange); font-style:normal; }
.hero-copy p { max-width:420px; margin:0; color:var(--qsl-muted); font-size:14.5px; line-height:1.75; }
.hero-actions { display:flex; gap:10px; margin-top:26px; }

/* 系统实时数据面板 */
.stats-panel { position:relative; width:min(340px,100%); margin:0 auto; padding:22px 22px 16px; background:#fff; border:2px solid var(--qsl-navy); box-shadow:6px 6px 0 rgba(24,45,61,.12); }
.stats-panel::after { content:"LIVE"; position:absolute; top:-9px; right:14px; padding:2px 8px; color:var(--qsl-navy); background:var(--qsl-yellow); font-size:9px; font-weight:850; letter-spacing:.14em; }
.stats-head { display:flex; align-items:center; gap:8px; color:var(--qsl-navy); font-size:12px; font-weight:800; letter-spacing:.14em; margin-bottom:16px; }
.stats-dot { flex:none; width:8px; height:8px; background:var(--qsl-orange); }
.stats-grid { display:grid; grid-template-columns:repeat(3,1fr); }
.stat-item { text-align:center; padding:0 4px; }
.stat-item + .stat-item { border-left:1px dotted #b9b2a4; }
.stat-item b { display:block; color:var(--qsl-navy); font-size:30px; font-weight:850; line-height:1.1; }
.stat-item span { display:block; margin-top:7px; color:var(--qsl-muted); font-size:11px; }
.stats-note { margin:14px 0 0; color:#b3ada0; font-size:10px; text-align:center; }

/* 服务入口 */
.services { display:grid; grid-template-columns:repeat(3,1fr); gap:12px; padding-top:30px; border-top:1px solid var(--qsl-line); }
.service-card { display:flex; align-items:center; gap:11px; width:100%; padding:13px 15px; color:var(--qsl-navy); text-align:left; border:1px solid var(--qsl-line); background:#fff; cursor:pointer; transition:transform .18s, box-shadow .18s; font-family:inherit; line-height:1.5; }
.service-card:hover { transform:translateY(-2px); box-shadow:3px 4px 0 rgba(24,45,61,.1); border-color:var(--qsl-navy); }
.service-card:hover .service-arrow { transform:translateX(3px); color:var(--qsl-orange); }
.service-icon { display:grid; place-items:center; flex:none; width:30px; height:30px; font-size:14px; }
.service-icon.orange { color:var(--qsl-orange); background:#fff0eb; }
.service-icon.blue { color:#286486; background:#e8f2f7; }
.service-icon.green { color:var(--qsl-green); background:#e9f5f1; }
.service-text { flex:1; min-width:0; }
.service-text b, .service-text small { display:block; }
.service-text b { font-size:13px; }
.service-text small { margin-top:3px; color:var(--qsl-muted); font-size:11px; }
.service-arrow { flex:none; color:var(--qsl-muted); font-size:13px; transition:transform .18s, color .18s; }

footer { max-width:1080px; width:100%; margin:0 auto; padding:22px 24px; color:#9c978d; border-top:1px solid var(--qsl-line); font-size:11px; }

@media(max-width:880px){
  .hero { grid-template-columns:1fr; gap:34px; padding-bottom:36px; }
  .stats-panel { width:min(320px,92%); }
}
@media(max-width:700px){
  .portal-main { padding-top:26px; }
  .services { grid-template-columns:1fr; }
}
</style>
