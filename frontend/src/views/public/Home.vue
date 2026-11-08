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
          <div class="telemetry-pill hero-telemetry">
            <span class="pulse-indicator"></span>
            <span>AMATEUR RADIO DX SERVICE · UTC+8</span>
          </div>
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
        <button class="service-card" @click="$router.push('/confirm')">
          <div class="service-icon amber"><el-icon><Check /></el-icon></div>
          <div class="service-text">
            <b>{{ t('home.svcConfirm') }}</b>
            <small>{{ t('home.svcConfirmDesc') }}</small>
          </div>
          <el-icon class="service-arrow"><ArrowRight /></el-icon>
        </button>
      </section>

      <!-- 换卡全流程说明 -->
      <section class="workflow-section">
        <div class="section-title-wrap">
          <span class="telemetry-pill">HOW IT WORKS</span>
          <h2>{{ t('home.workflowTitle') }}</h2>
          <p>{{ t('home.workflowSub') }}</p>
        </div>
        <div class="workflow-grid">
          <div class="workflow-step">
            <div class="step-num">01</div>
            <div class="step-icon blue"><el-icon><Document /></el-icon></div>
            <b>{{ t('home.step1Title') }}</b>
            <p>{{ t('home.step1Desc') }}</p>
          </div>
          <div class="workflow-step">
            <div class="step-num">02</div>
            <div class="step-icon indigo"><el-icon><CircleCheck /></el-icon></div>
            <b>{{ t('home.step2Title') }}</b>
            <p>{{ t('home.step2Desc') }}</p>
          </div>
          <div class="workflow-step">
            <div class="step-num">03</div>
            <div class="step-icon amber"><el-icon><Postcard /></el-icon></div>
            <b>{{ t('home.step3Title') }}</b>
            <p>{{ t('home.step3Desc') }}</p>
          </div>
          <div class="workflow-step">
            <div class="step-num">04</div>
            <div class="step-icon emerald"><el-icon><Box /></el-icon></div>
            <b>{{ t('home.step4Title') }}</b>
            <p>{{ t('home.step4Desc') }}</p>
          </div>
        </div>
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
  // 站名和统计互不依赖，一起发。其中一个失败不挡住另一个。
  const [info, s] = await Promise.all([
    api.get('/public/site-info').catch(() => null),
    api.get('/public/stats').catch(() => null),
  ])
  if (info?.site_name) {
    siteName.value = info.site_name
    document.title = info.site_name
  }
  siteNotice.value = (info && info.site_notice) || ''
  if (s) {
    stats.cards_sent = s.cards_sent ?? 0
    stats.cards_signed = s.cards_signed ?? 0
    stats.pending_requests = s.pending_requests ?? 0
    statsLoaded.value = true
  }
})
</script>

<style scoped>
/* 分类强调色只定义色相（--qsl-tone-* 在 tokens.css），
   底色用 color-mix 叠在主题表面之上，暗色主题下也不会刺眼 */
.portal-page {
  --tone-blue: var(--qsl-tone-blue);
  --tone-indigo: var(--qsl-tone-indigo);
  --tone-emerald: var(--qsl-tone-emerald);

  min-height: 100vh;
  display: flex;
  flex-direction: column;
  color: var(--qsl-ink);
  background: var(--qsl-bg);
  background-image: radial-gradient(at 10% 10%, color-mix(in srgb, var(--qsl-primary) 6%, transparent) 0px, transparent 50%),
                    radial-gradient(at 90% 15%, color-mix(in srgb, var(--qsl-green) 5%, transparent) 0px, transparent 50%);
}

.portal-header {
  position: sticky;
  top: 0;
  z-index: 10;
  height: 72px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  max-width: 1120px;
  width: 100%;
  margin: 0 auto;
  padding: 0 24px;
  background: color-mix(in srgb, var(--qsl-bg) 84%, transparent);
  backdrop-filter: blur(14px);
  border-bottom: 1px solid var(--qsl-line);
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
  color: var(--qsl-on-primary);
  background: var(--qsl-primary);
  border-radius: 10px;
  font-size: 18px;
  font-weight: 850;
  box-shadow: 0 4px 10px color-mix(in srgb, var(--qsl-ink) 20%, transparent);
}
.brand-info strong {
  display: block;
  color: var(--qsl-navy);
  font-size: 15px;
  font-weight: 750;
  letter-spacing: 0.06em;
}
.brand-info small {
  display: block;
  margin-top: 2px;
  color: var(--qsl-muted);
  font-size: 11px;
}

.admin-link {
  color: var(--qsl-navy);
  font-size: 13px;
  font-weight: 600;
  padding: 6px 14px;
  border-radius: 9999px;
  background: var(--qsl-panel);
  border: 1px solid var(--qsl-line);
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.04);
  transition: all 0.2s ease;
}
.admin-link:hover {
  color: var(--el-color-primary);
  border-color: var(--el-color-primary);
  background: var(--qsl-paper);
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
  background: var(--qsl-panel);
  border: 1px solid var(--qsl-line);
  border-left: 4px solid var(--qsl-yellow);
  border-radius: 12px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);
}
.notice-badge {
  display: grid;
  place-items: center;
  flex: none;
  width: 28px;
  height: 28px;
  color: var(--qsl-yellow);
  background: color-mix(in srgb, var(--qsl-yellow) 12%, transparent);
  border-radius: 8px;
  font-size: 14px;
}
.notice-body b {
  display: block;
  color: var(--qsl-navy);
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
  color: var(--qsl-ink);
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
  color: var(--el-color-primary);
  font-size: 11px;
  font-weight: 800;
  letter-spacing: 0.16em;
}
.hero-telemetry {
  margin-bottom: 16px;
}

.hero-title {
  margin: 14px 0 18px;
  color: var(--qsl-navy);
  font-size: clamp(30px, 4vw, 44px);
  line-height: 1.15;
  letter-spacing: -0.03em;
  font-weight: 850;
}
.hero-title .highlight {
  color: var(--el-color-primary);
}
.hero-desc {
  max-width: 440px;
  margin: 0;
  color: var(--qsl-muted);
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
  background: var(--el-color-primary);
  border-color: var(--el-color-primary);
  box-shadow: 0 4px 14px rgba(0, 0, 0, 0.18);
}
.hero-btn-secondary {
  height: 46px;
  padding: 0 22px;
  font-size: 14.5px;
  border-radius: 10px;
  background: var(--qsl-panel);
  border: 1px solid var(--qsl-line);
  color: var(--qsl-navy);
}
.hero-btn-secondary:hover {
  border-color: var(--el-color-primary);
  background: var(--qsl-paper);
}

/* 系统实时数据面板 */
.stats-panel {
  position: relative;
  width: 100%;
  max-width: 360px;
  margin: 0 auto;
  padding: 24px;
  background: var(--qsl-panel);
  border: 1px solid var(--qsl-line);
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
  color: var(--qsl-navy);
  font-size: 13px;
  font-weight: 700;
  letter-spacing: 0.04em;
}
.stats-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--qsl-green);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--qsl-green) 25%, transparent);
  animation: pulse-dot 2s infinite;
}
.live-pill {
  color: var(--qsl-green);
  background: color-mix(in srgb, var(--qsl-green) 12%, transparent);
  font-size: 10px;
  font-weight: 800;
  letter-spacing: 0.12em;
  padding: 2px 8px;
  border-radius: 9999px;
  border: 1px solid var(--qsl-line);
}
@keyframes pulse-dot {
  0%, 100% { box-shadow: 0 0 0 3px color-mix(in srgb, var(--qsl-green) 25%, transparent); }
  50% { box-shadow: 0 0 0 6px color-mix(in srgb, var(--qsl-green) 10%, transparent); }
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
  border-left: 1px solid var(--qsl-line);
}
.stat-item b {
  display: block;
  color: var(--qsl-navy);
  font-family: var(--qsl-font-mono);
  font-size: 28px;
  font-weight: 850;
  line-height: 1.1;
  letter-spacing: -0.03em;
  font-variant-numeric: tabular-nums;
}
.stat-item span {
  display: block;
  margin-top: 6px;
  color: var(--qsl-muted);
  font-size: 12px;
}
.stats-note {
  margin: 18px 0 0;
  color: var(--qsl-muted);
  font-size: 11px;
  text-align: center;
}

/* 服务入口 */
.services {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
  padding-top: 36px;
  border-top: 1px solid var(--qsl-line);
}
.service-card {
  display: flex;
  align-items: center;
  gap: 14px;
  width: 100%;
  padding: 18px 20px;
  color: var(--qsl-navy);
  text-align: left;
  border: 1px solid var(--qsl-line);
  border-radius: 14px;
  background: var(--qsl-panel);
  cursor: pointer;
  transition: all 0.22s cubic-bezier(0.4, 0, 0.2, 1);
  font-family: inherit;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.03);
}
.service-card:hover {
  transform: translateY(-3px);
  box-shadow: 0 10px 15px -3px rgba(0, 0, 0, 0.06), 0 4px 6px -4px rgba(0, 0, 0, 0.03);
  border-color: var(--el-color-primary);
}
.service-card:hover .service-arrow {
  transform: translateX(4px);
  color: var(--el-color-primary);
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
.service-icon.blue { color: var(--qsl-tone-blue); background: color-mix(in srgb, var(--qsl-tone-blue) 12%, transparent); }
.service-icon.indigo { color: var(--qsl-tone-indigo); background: color-mix(in srgb, var(--qsl-tone-indigo) 12%, transparent); }
.service-icon.emerald { color: var(--qsl-tone-emerald); background: color-mix(in srgb, var(--qsl-tone-emerald) 12%, transparent); }
.service-icon.amber { color: var(--qsl-tone-amber); background: color-mix(in srgb, var(--qsl-tone-amber) 12%, transparent); }

/* 换卡全流程说明 */
.workflow-section {
  margin-top: 48px;
  padding-top: 36px;
  border-top: 1px solid var(--qsl-line);
}
.section-title-wrap {
  text-align: center;
  margin-bottom: 30px;
}
.section-title-wrap h2 {
  margin: 10px 0 6px;
  color: var(--qsl-heading);
  font-size: 24px;
  font-weight: 800;
  letter-spacing: -0.02em;
}
.section-title-wrap p {
  margin: 0;
  color: var(--qsl-muted);
  font-size: 14px;
}
.workflow-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
}
.workflow-step {
  position: relative;
  padding: 24px 20px;
  border-radius: 14px;
  background: var(--qsl-panel);
  border: 1px solid var(--qsl-line);
  box-shadow: var(--qsl-shadow-1);
  transition: all 0.22s cubic-bezier(0.4, 0, 0.2, 1);
}
.workflow-step:hover {
  transform: translateY(-2px);
  border-color: var(--el-color-primary);
  box-shadow: var(--qsl-shadow-2);
}
.step-num {
  position: absolute;
  top: 18px;
  right: 18px;
  font-family: var(--qsl-font-mono);
  font-size: 15px;
  font-weight: 850;
  color: color-mix(in srgb, var(--qsl-muted) 40%, transparent);
}
.step-icon {
  width: 38px;
  height: 38px;
  display: grid;
  place-items: center;
  border-radius: 10px;
  font-size: 18px;
  margin-bottom: 14px;
}
.step-icon.blue { color: var(--qsl-tone-blue); background: color-mix(in srgb, var(--qsl-tone-blue) 12%, transparent); }
.step-icon.indigo { color: var(--qsl-tone-indigo); background: color-mix(in srgb, var(--qsl-tone-indigo) 12%, transparent); }
.step-icon.amber { color: var(--qsl-tone-amber); background: color-mix(in srgb, var(--qsl-tone-amber) 12%, transparent); }
.step-icon.emerald { color: var(--qsl-tone-emerald); background: color-mix(in srgb, var(--qsl-tone-emerald) 12%, transparent); }
.workflow-step b {
  display: block;
  font-size: 15px;
  font-weight: 750;
  color: var(--qsl-heading);
  margin-bottom: 6px;
}
.workflow-step p {
  margin: 0;
  color: var(--qsl-muted);
  font-size: 12.5px;
  line-height: 1.6;
}

.service-text {
  flex: 1;
  min-width: 0;
}
.service-text b {
  display: block;
  font-size: 14.5px;
  font-weight: 700;
  color: var(--qsl-navy);
}
.service-text small {
  display: block;
  margin-top: 3px;
  color: var(--qsl-muted);
  font-size: 12px;
}
.service-arrow {
  flex: none;
  color: var(--qsl-muted);
  font-size: 15px;
  transition: all 0.2s ease;
}

.portal-footer {
  max-width: 1120px;
  width: 100%;
  margin: 0 auto;
  padding: 24px;
  border-top: 1px solid var(--qsl-line);
}
.footer-inner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  color: var(--qsl-muted);
  font-size: 12px;
}
.footer-sub {
  font-size: 11px;
  letter-spacing: 0.06em;
}

@media (max-width: 990px) {
  .services { grid-template-columns: repeat(2, 1fr); }
  .workflow-grid { grid-template-columns: repeat(2, 1fr); }
}
@media (max-width: 880px) {
  .hero { grid-template-columns: 1fr; gap: 36px; padding-bottom: 36px; }
  .stats-panel { max-width: 100%; }
  .footer-inner { flex-direction: column; gap: 6px; text-align: center; }
}
@media (max-width: 720px) {
  .portal-main { padding-top: 24px; }
  .services { grid-template-columns: 1fr; }
  .workflow-grid { grid-template-columns: 1fr; }
}
</style>
