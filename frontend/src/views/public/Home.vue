<template>
  <div class="portal-page">
    <header class="portal-header">
      <div class="portal-brand">
        <span class="brand-mark">Q</span>
        <div class="brand-info">
          <strong>{{ siteName }}</strong>
          <small>{{ t('home.brandSubtitle') }}</small>
        </div>
      </div>
      <el-button class="admin-link" text @click="$router.push('/login')">
        <span>{{ t('home.adminLink') }}</span>
        <el-icon><ArrowRight /></el-icon>
      </el-button>
    </header>

    <main class="portal-main">
      <!-- 站点公告 -->
      <section v-if="siteNotice" class="site-notice">
        <div class="notice-badge"><el-icon><BellFilled /></el-icon></div>
        <div class="notice-body">
          <b>{{ t('home.noticeBadge') }}</b>
          <pre>{{ siteNotice }}</pre>
        </div>
      </section>

      <!-- 首屏：左文案 + 右数据面板 -->
      <section class="hero">
        <div class="hero-copy">
          <div class="eyebrow">AMATEUR RADIO QSL PORTAL</div>
          <h1 class="hero-title">
            {{ t('home.heroTitle1') }}<br>
            <span class="highlight">{{ t('home.heroTitle2') }}</span>
          </h1>
          <p class="hero-desc">{{ t('home.heroDesc') }}</p>
          <div class="hero-actions">
            <el-button type="primary" size="large" class="hero-btn-primary" @click="$router.push('/apply')">
              <span>{{ t('home.btnApply') }}</span>
              <el-icon><ArrowRight /></el-icon>
            </el-button>
            <el-button size="large" class="hero-btn-secondary" @click="$router.push('/status')">
              <span>{{ t('home.btnStatus') }}</span>
            </el-button>
          </div>
        </div>

        <!-- 系统实时数据 -->
        <div v-if="statsLoaded" class="stats-panel">
          <div class="stats-head">
            <div class="stats-indicator">
              <span class="stats-dot"></span>
              <span>{{ t('home.statsTitle') }}</span>
            </div>
            <span class="live-pill">LIVE</span>
          </div>
          <div class="stats-grid">
            <div class="stat-item">
              <b>{{ stats.cards_sent }}</b>
              <span>{{ t('home.statsSent') }}</span>
            </div>
            <div class="stat-item">
              <b>{{ stats.cards_signed }}</b>
              <span>{{ t('home.statsSigned') }}</span>
            </div>
            <div class="stat-item">
              <b>{{ stats.pending_requests }}</b>
              <span>{{ t('home.statsPending') }}</span>
            </div>
          </div>
          <p class="stats-note">{{ t('home.statsNote') }}</p>
        </div>
      </section>

      <!-- 服务入口 -->
      <section class="services">
        <button class="service-card" @click="$router.push('/apply')">
          <div class="service-icon blue"><el-icon><Promotion /></el-icon></div>
          <div class="service-text">
            <b>{{ t('home.svcApply') }}</b>
            <small>{{ t('home.svcApplyDesc') }}</small>
          </div>
          <el-icon class="service-arrow"><ArrowRight /></el-icon>
        </button>
        <button class="service-card" @click="$router.push('/status')">
          <div class="service-icon indigo"><el-icon><Search /></el-icon></div>
          <div class="service-text">
            <b>{{ t('home.svcStatus') }}</b>
            <small>{{ t('home.svcStatusDesc') }}</small>
          </div>
          <el-icon class="service-arrow"><ArrowRight /></el-icon>
        </button>
        <button class="service-card" @click="$router.push('/track')">
          <div class="service-icon emerald"><el-icon><Van /></el-icon></div>
          <div class="service-text">
            <b>{{ t('home.svcTrack') }}</b>
            <small>{{ t('home.svcTrackDesc') }}</small>
          </div>
          <el-icon class="service-arrow"><ArrowRight /></el-icon>
        </button>
      </section>
    </main>

    <footer class="portal-footer">
      <div class="footer-inner">
        <span>{{ siteName }} · {{ t('home.footerTagline') }}</span>
        <span class="footer-sub">73 DE BI1KBU · AMATEUR RADIO STATION</span>
      </div>
    </footer>
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
.portal-page {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  color: var(--qsl-ink, #0f172a);
  background: var(--qsl-paper, #f8fafc);
  background-image: radial-gradient(at 10% 10%, rgba(59, 130, 246, 0.05) 0px, transparent 50%),
                    radial-gradient(at 90% 15%, rgba(16, 185, 129, 0.04) 0px, transparent 50%);
}

.portal-header {
  height: 72px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  max-width: 1120px;
  width: 100%;
  margin: 0 auto;
  padding: 0 24px;
  border-bottom: 1px solid #e2e8f0;
}

.portal-brand {
  display: flex;
  align-items: center;
  gap: 12px;
}
.brand-mark {
  display: grid;
  place-items: center;
  width: 36px;
  height: 36px;
  color: #ffffff;
  background: linear-gradient(135deg, #3b82f6 0%, #2563eb 100%);
  border-radius: 10px;
  font-size: 18px;
  font-weight: 850;
  box-shadow: 0 4px 10px rgba(37, 99, 235, 0.28);
}
.brand-info strong {
  display: block;
  color: #0f172a;
  font-size: 15px;
  font-weight: 750;
  letter-spacing: 0.06em;
}
.brand-info small {
  display: block;
  margin-top: 2px;
  color: #64748b;
  font-size: 11px;
}

.admin-link {
  color: #475569;
  font-size: 13px;
  font-weight: 600;
  padding: 6px 14px;
  border-radius: 9999px;
  background: #ffffff;
  border: 1px solid #e2e8f0;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.04);
  transition: all 0.2s ease;
}
.admin-link:hover {
  color: #2563eb;
  border-color: #bfdbfe;
  background: #eff6ff;
}

.portal-main {
  flex: 1;
  width: 100%;
  max-width: 1120px;
  margin: 0 auto;
  padding: 40px 24px 50px;
}

/* 公告条 */
.site-notice {
  display: flex;
  gap: 14px;
  align-items: flex-start;
  margin-bottom: 32px;
  padding: 16px 20px;
  background: #fffbeb;
  border: 1px solid #fef3c7;
  border-left: 4px solid #f59e0b;
  border-radius: 12px;
  box-shadow: 0 1px 3px rgba(245, 158, 11, 0.08);
}
.notice-badge {
  display: grid;
  place-items: center;
  flex: none;
  width: 28px;
  height: 28px;
  color: #d97706;
  background: #fef3c7;
  border-radius: 8px;
  font-size: 14px;
}
.notice-body b {
  display: block;
  color: #92400e;
  font-size: 13px;
  font-weight: 700;
}
.notice-body pre {
  margin: 6px 0 0;
  font-family: inherit;
  font-size: 13px;
  line-height: 1.7;
  white-space: pre-wrap;
  word-break: break-all;
  color: #78350f;
}

/* 首屏双栏 */
.hero {
  display: grid;
  grid-template-columns: 1.15fr 0.85fr;
  gap: 48px;
  align-items: center;
  padding: 24px 0 54px;
}
.eyebrow {
  color: #2563eb;
  font-size: 11px;
  font-weight: 800;
  letter-spacing: 0.16em;
}
.hero-title {
  margin: 14px 0 18px;
  color: #0f172a;
  font-size: clamp(30px, 4vw, 44px);
  line-height: 1.15;
  letter-spacing: -0.03em;
  font-weight: 850;
}
.hero-title .highlight {
  background: linear-gradient(135deg, #2563eb 0%, #4f46e5 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
}
.hero-desc {
  max-width: 440px;
  margin: 0;
  color: #64748b;
  font-size: 15px;
  line-height: 1.75;
}
.hero-actions {
  display: flex;
  gap: 12px;
  margin-top: 30px;
}
.hero-btn-primary {
  height: 46px;
  padding: 0 24px;
  font-size: 14.5px;
  border-radius: 10px;
  box-shadow: 0 4px 14px rgba(37, 99, 235, 0.3);
}
.hero-btn-secondary {
  height: 46px;
  padding: 0 22px;
  font-size: 14.5px;
  border-radius: 10px;
  background: #ffffff;
  border: 1px solid #e2e8f0;
  color: #334155;
}
.hero-btn-secondary:hover {
  border-color: #cbd5e1;
  background: #f8fafc;
}

/* 系统实时数据面板 */
.stats-panel {
  position: relative;
  width: 100%;
  max-width: 360px;
  margin: 0 auto;
  padding: 24px;
  background: #ffffff;
  border: 1px solid rgba(226, 232, 240, 0.9);
  border-radius: 16px;
  box-shadow: 0 10px 25px -5px rgba(0, 0, 0, 0.05), 0 8px 10px -6px rgba(0, 0, 0, 0.03);
}
.stats-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 20px;
}
.stats-indicator {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #0f172a;
  font-size: 13px;
  font-weight: 700;
  letter-spacing: 0.04em;
}
.stats-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #10b981;
  box-shadow: 0 0 0 3px rgba(16, 185, 129, 0.25);
  animation: pulse-dot 2s infinite;
}
.live-pill {
  color: #059669;
  background: #ecfdf5;
  font-size: 10px;
  font-weight: 800;
  letter-spacing: 0.12em;
  padding: 2px 8px;
  border-radius: 9999px;
  border: 1px solid #d1fae5;
}
@keyframes pulse-dot {
  0%, 100% { box-shadow: 0 0 0 3px rgba(16, 185, 129, 0.25); }
  50% { box-shadow: 0 0 0 6px rgba(16, 185, 129, 0.1); }
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 8px;
}
.stat-item {
  text-align: center;
  padding: 4px;
}
.stat-item + .stat-item {
  border-left: 1px solid #f1f5f9;
}
.stat-item b {
  display: block;
  color: #0f172a;
  font-size: 28px;
  font-weight: 850;
  line-height: 1.1;
  letter-spacing: -0.03em;
  font-variant-numeric: tabular-nums;
}
.stat-item span {
  display: block;
  margin-top: 6px;
  color: #64748b;
  font-size: 12px;
}
.stats-note {
  margin: 18px 0 0;
  color: #94a3b8;
  font-size: 11px;
  text-align: center;
}

/* 服务入口 */
.services {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
  padding-top: 36px;
  border-top: 1px solid #e2e8f0;
}
.service-card {
  display: flex;
  align-items: center;
  gap: 14px;
  width: 100%;
  padding: 18px 20px;
  color: #0f172a;
  text-align: left;
  border: 1px solid #e2e8f0;
  border-radius: 14px;
  background: #ffffff;
  cursor: pointer;
  transition: all 0.22s cubic-bezier(0.4, 0, 0.2, 1);
  font-family: inherit;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.03);
}
.service-card:hover {
  transform: translateY(-3px);
  box-shadow: 0 10px 15px -3px rgba(0, 0, 0, 0.06), 0 4px 6px -4px rgba(0, 0, 0, 0.03);
  border-color: #93c5fd;
}
.service-card:hover .service-arrow {
  transform: translateX(4px);
  color: #2563eb;
}

.service-icon {
  display: grid;
  place-items: center;
  flex: none;
  width: 40px;
  height: 40px;
  border-radius: 10px;
  font-size: 18px;
}
.service-icon.blue { color: #2563eb; background: #eff6ff; }
.service-icon.indigo { color: #4f46e5; background: #eef2ff; }
.service-icon.emerald { color: #059669; background: #ecfdf5; }

.service-text {
  flex: 1;
  min-width: 0;
}
.service-text b {
  display: block;
  font-size: 14.5px;
  font-weight: 700;
  color: #0f172a;
}
.service-text small {
  display: block;
  margin-top: 3px;
  color: #64748b;
  font-size: 12px;
}
.service-arrow {
  flex: none;
  color: #94a3b8;
  font-size: 15px;
  transition: all 0.2s ease;
}

.portal-footer {
  max-width: 1120px;
  width: 100%;
  margin: 0 auto;
  padding: 24px;
  border-top: 1px solid #e2e8f0;
}
.footer-inner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  color: #94a3b8;
  font-size: 12px;
}
.footer-sub {
  font-size: 11px;
  letter-spacing: 0.06em;
}

@media (max-width: 880px) {
  .hero { grid-template-columns: 1fr; gap: 36px; padding-bottom: 36px; }
  .stats-panel { max-width: 100%; }
  .footer-inner { flex-direction: column; gap: 6px; text-align: center; }
}
@media (max-width: 720px) {
  .portal-main { padding-top: 24px; }
  .services { grid-template-columns: 1fr; }
}
</style>
