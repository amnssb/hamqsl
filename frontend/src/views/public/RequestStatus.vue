<template>
  <div class="public-page">
    <div class="public-container">
      <header class="public-header">
        <div class="logo-section">
          <div class="logo-icon">QSL</div>
          <div>
            <h1>{{ t('status.title') }}</h1>
            <p class="subtitle" v-if="!info">{{ t('status.subtitle') }}</p>
          </div>
        </div>
        <button class="home-back-link" @click="$router.push('/')">
          <el-icon><ArrowLeft /></el-icon>
          <span>{{ t('common.backHome') }}</span>
        </button>
      </header>

      <main class="public-main">
        <!-- 手动输入（URL 无编号时） -->
        <div v-if="!info && !loading && !error" class="card modern-card">
          <el-form label-position="top" @submit.prevent>
            <el-form-item :label="t('status.requestCode')">
              <el-input v-model="manualCode" :placeholder="t('status.requestCodePlaceholder')" size="large" @keyup.enter="lookup" />
            </el-form-item>
            <div class="form-actions">
              <el-button type="primary" size="large" :loading="loading" @click="lookup" class="submit-btn">
                <el-icon><Search /></el-icon> {{ t('status.lookupBtn') }}
              </el-button>
            </div>
          </el-form>
        </div>

        <!-- 加载中 -->
        <div v-if="loading && !info" class="card modern-card" style="text-align:center;padding:60px;">
          <el-icon class="is-loading" :size="32"><Loading /></el-icon>
          <p style="margin-top:12px;color:var(--qsl-muted);">{{ t('status.loading') }}</p>
        </div>

        <!-- 查询失败 -->
        <div v-if="error" class="card modern-card" style="text-align:center;padding:40px;">
          <el-icon :size="48" color="var(--qsl-danger)"><CircleClose /></el-icon>
          <h2 style="color:var(--qsl-danger);margin:16px 0 8px;">{{ t('status.failedTitle') }}</h2>
          <p style="color:var(--qsl-muted);">{{ error }}</p>
          <el-button @click="reset" style="margin-top:16px;">{{ t('status.reenter') }}</el-button>
        </div>

        <!-- 进度详情 -->
        <div v-if="info" class="card modern-card">
          <!-- 站点公告（管理员在设置中配置，留空不显示） -->
          <div v-if="siteNotice" class="notice-block">
            <div class="notice-head"><el-icon><BellFilled /></el-icon><b>{{ t('status.notice') }}</b></div>
            <pre class="notice-text">{{ siteNotice }}</pre>
          </div>

          <el-descriptions :column="2" border style="margin-bottom:28px;">
            <el-descriptions-item :label="t('status.requestCode')"><span class="mono-val">{{ info.request_code }}</span></el-descriptions-item>
            <el-descriptions-item :label="t('status.callSign')"><span class="callsign-badge">{{ info.call_sign }}</span></el-descriptions-item>
            <el-descriptions-item :label="t('status.scene')">{{ sceneText(info.scene_type) }}</el-descriptions-item>
            <el-descriptions-item :label="t('status.requestTime')"><span class="mono-val">{{ fmtTime(info.created_at) }}</span></el-descriptions-item>
          </el-descriptions>

          <el-steps :active="activeStep" align-center :process-status="rejected ? 'error' : 'process'" finish-status="success">
            <el-step v-for="(s, idx) in stepsData" :key="idx" :title="s.title" :description="s.desc" />
          </el-steps>

          <el-alert v-if="rejected" type="error" :closable="false" style="margin-top:28px;">
            {{ t('status.rejectedPrefix') }}{{ info.review_reason || t('status.noReason') }}
          </el-alert>
          <div v-else-if="info.tracking_number && info.flow_status !== 'SIGNED'" class="mail-block">
            <div class="mail-head">
              <b>{{ t('status.mailTracking') }}</b>
              <el-tag size="small" class="mono-val">{{ info.tracking_number }}</el-tag>
            </div>
            <p class="mail-tip">{{ t('status.kuaidiTip') }}</p>
            <div class="mail-actions">
              <el-button type="primary" @click="openKuaidi100">{{ t('status.openKuaidi') }}</el-button>
              <el-button @click="copyTracking">{{ t('status.copyTracking') }}</el-button>
            </div>
          </div>

          <!-- SWL 反寄：回寄地址（管理员发送后展示） -->
          <div v-if="info.return_address_text && !info.return_mailed_at" class="mail-block">
            <div class="mail-head"><b>{{ t('status.sendToAddress') }}</b></div>
            <pre class="addr-text">{{ info.return_address_text }}</pre>
          </div>

          <!-- SWL 反寄：对方寄出后登记单号/平信 -->
          <div v-if="info.scene_type === 'SWL' && info.review_status === 'APPROVED'" class="mail-block">
            <template v-if="!info.return_mailed_at">
              <div class="mail-head"><b>{{ t('status.registerMailHead') }}</b></div>
              <p class="mail-tip">{{ t('status.registerMailTip') }}</p>
              <el-form label-position="top" style="margin-top:8px;">
                <el-form-item :label="t('status.mailType')">
                  <el-radio-group v-model="returnForm.mail_type">
                    <el-radio value="REGISTERED">{{ t('status.registeredMail') }}</el-radio>
                    <el-radio value="ORDINARY">{{ t('status.ordinaryMail') }}</el-radio>
                  </el-radio-group>
                </el-form-item>
                <el-form-item v-if="returnForm.mail_type === 'REGISTERED'" :label="t('status.trackingNo')">
                  <el-input v-model="returnForm.tracking_number" :placeholder="t('status.trackingPlaceholder')" />
                </el-form-item>
                <el-button type="primary" :loading="returnSaving" @click="submitReturnMail">{{ t('status.submitRegister') }}</el-button>
              </el-form>
            </template>
            <template v-else>
              <div class="mail-head">
                <b>{{ t('status.registeredHead') }}</b>
                <el-tag size="small" :type="info.return_mail_type === 'REGISTERED' ? '' : 'info'">{{ info.return_mail_type === 'REGISTERED' ? t('status.registeredMail') : t('status.ordinaryMail') }}</el-tag>
              </div>
              <p v-if="info.return_tracking" class="mail-tip">{{ t('status.trackingPrefix') }}<span class="mono-val">{{ info.return_tracking }}</span></p>
              <p class="mail-tip">{{ t('status.registeredAtPrefix') }}{{ fmtTime(info.return_mailed_at) }}</p>
              <el-button v-if="info.return_tracking" size="small" @click="openReturnTracking">{{ t('status.openKuaidiShort') }}</el-button>
            </template>
          </div>

          <!-- 邮件未收到引导：申请通过后各节点均有邮件提醒，教对方查垃圾箱+加白名单 -->
          <div v-if="info && info.review_status === 'APPROVED' && senderEmail" class="mail-block">
            <div class="mail-head"><b>{{ t('status.mailNotReceived') }}</b></div>
            <p class="mail-tip">
              {{ t('status.whitelistTipBefore') }}
              <span class="sender-mail">{{ senderEmail }}</span>
              {{ t('status.whitelistTipAfter') }}
            </p>
            <div class="mail-actions">
              <el-button size="small" @click="copySenderEmail">{{ t('status.copySender') }}</el-button>
            </div>
          </div>

          <p class="refresh-note">{{ t('status.cardCodePrefix') }}{{ info.card_code || t('status.cardNotGenerated') }}<el-button link type="primary" size="small" style="margin-left:8px;" @click="refresh">{{ t('status.manualRefresh') }}</el-button></p>
        </div>
      </main>

      <footer class="public-footer">
        <p>{{ t('common.footer') }}</p>
      </footer>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import api from '../../api'
import { ElMessage } from 'element-plus'
import { t } from '../../i18n'

const route = useRoute()
const router = useRouter()
const loading = ref(false)
const info = ref(null)
const error = ref('')
const manualCode = ref('')
const returnForm = reactive({ mail_type: 'REGISTERED', tracking_number: '' })
const returnSaving = ref(false)
// 站点公开信息：公告 + 发件邮箱（白名单引导），跟随设置页配置
const siteNotice = ref('')
const senderEmail = ref('')

async function loadSiteInfo() {
  try {
    const res = await api.get('/public/site-info')
    siteNotice.value = (res && res.site_notice) || ''
    senderEmail.value = (res && res.sender_email) || ''
  } catch (e) { /* 公告与白名单提示缺省隐藏 */ }
}

async function copySenderEmail() {
  if (!senderEmail.value) return
  try {
    await navigator.clipboard.writeText(senderEmail.value)
    ElMessage.success(t('status.senderCopied') + senderEmail.value)
  } catch (e) { ElMessage.error(t('status.copySenderFailed')) }
}

async function submitReturnMail() {
  if (!info.value?.request_code) return
  if (returnForm.mail_type === 'REGISTERED' && !returnForm.tracking_number) {
    ElMessage.warning(t('status.trackingRequired'))
    return
  }
  returnSaving.value = true
  try {
    await api.post('/public/exchange-return-mail', {
      request_code: info.value.request_code,
      mail_type: returnForm.mail_type,
      tracking_number: returnForm.tracking_number
    })
    ElMessage.success(t('status.registerSuccess'))
    await refresh()
  } catch (e) { /* 拦截器已提示 */ } finally { returnSaving.value = false }
}

function openReturnTracking() {
  if (info.value?.return_tracking) {
    window.open('https://www.kuaidi100.com/chaxun?nu=' + encodeURIComponent(info.value.return_tracking), '_blank')
  }
}

const rejected = computed(() => info.value && info.value.review_status === 'REJECTED')

const activeStep = computed(() => {
  const i = info.value
  if (!i) return 0
  if (i.review_status === 'REJECTED') return 1
  if (i.scene_type === 'SWL') {
    if (i.flow_status === 'SIGNED') return 6
    if (['SENT', 'RECEIVED'].includes(i.flow_status)) return 5
    if (i.return_received_at) return 4
    if (i.return_mailed_at) return 3
    if (i.review_status === 'APPROVED') return 2
    return 1
  }
  if (i.flow_status === 'SIGNED') return 5
  if (i.flow_status === 'SENT' || i.flow_status === 'RECEIVED') return 4
  if (i.card_created) return 3
  if (i.review_status === 'APPROVED') return 2
  return 1
})

const reviewDesc = computed(() => {
  const i = info.value
  if (!i) return ''
  if (i.review_status === 'PENDING') return t('status.reviewPending')
  if (i.review_status === 'APPROVED') return t('status.reviewApproved')
  if (i.review_status === 'REJECTED') return t('status.reviewRejected')
  return ''
})
const cardDesc = computed(() => {
  const i = info.value
  if (!i) return ''
  return i.card_created ? (t('status.cardCreatedPrefix') + (i.card_code || '')) : t('status.cardWaiting')
})
const mailDesc = computed(() => {
  const i = info.value
  if (!i) return ''
  if (['SENT', 'RECEIVED', 'SIGNED'].includes(i.flow_status)) return t('status.mailed')
  if (i.card_created) return t('status.preparing')
  return t('status.waitingMail')
})
const signedDesc = computed(() => info.value && info.value.flow_status === 'SIGNED' ? t('status.signed') : t('status.waitingSign'))

// SWL 反寄：第三步为"您寄出卡"，其后才是我方制卡回寄
const swlSentDesc = computed(() => {
  const i = info.value
  if (!i) return ''
  if (i.return_mailed_at) return t('status.sentAtPrefix') + fmtTime(i.return_mailed_at)
  if (i.address_sent_at) return t('status.addressSent')
  return t('status.waitingAddress')
})
const swlReceivedDesc = computed(() => {
  const i = info.value
  if (!i) return ''
  if (i.return_received_at) return t('status.receivedAtPrefix') + fmtTime(i.return_received_at)
  if (i.return_mailed_at) return t('status.waitingArrival')
  return '—'
})

// EYEBALL：交付方式灵活（见面交付或邮寄），第五步为"完成"
const eyeballDeliverDesc = computed(() => {
  const i = info.value
  if (!i) return ''
  if (i.flow_status === 'SIGNED') return t('status.delivered')
  if (['SENT', 'RECEIVED'].includes(i.flow_status)) return t('status.mailed')
  if (i.card_created) return t('status.eyeballDeliverMethods')
  return t('status.waitingCard')
})
const eyeballDoneDesc = computed(() => info.value && info.value.flow_status === 'SIGNED' ? t('status.completed') : t('status.pendingConfirm'))

// 按场景生成时间线步骤
const stepsData = computed(() => {
  const i = info.value
  if (!i) return []
  if (i.scene_type === 'SWL') {
    return [
      { title: t('status.stepSubmit'), desc: fmtTime(i.created_at) },
      { title: t('status.stepReview'), desc: reviewDesc.value },
      { title: t('status.stepSwlYouSend'), desc: swlSentDesc.value },
      { title: t('status.stepSwlWeReceive'), desc: swlReceivedDesc.value },
      { title: t('status.stepSwlMakeReturn'), desc: cardDesc.value },
      { title: t('status.stepSign'), desc: signedDesc.value },
    ]
  }
  if (i.scene_type === 'EYEBALL') {
    return [
      { title: t('status.stepSubmit'), desc: fmtTime(i.created_at) },
      { title: t('status.stepReview'), desc: reviewDesc.value },
      { title: t('status.stepMakeCard'), desc: cardDesc.value },
      { title: t('status.stepDeliver'), desc: eyeballDeliverDesc.value },
      { title: t('status.stepDone'), desc: eyeballDoneDesc.value },
    ]
  }
  return [
    { title: t('status.stepSubmit'), desc: fmtTime(i.created_at) },
    { title: t('status.stepReview'), desc: reviewDesc.value },
    { title: t('status.stepMakeCard'), desc: cardDesc.value },
    { title: t('status.stepMail'), desc: mailDesc.value },
    { title: t('status.stepSign'), desc: signedDesc.value },
  ]
})

const sceneText = (type) => {
  const i = info.value
  const base = ({ QSO: t('status.sceneQso'), SWL: t('status.sceneSwl'), EYEBALL: t('status.sceneEyeball') }[type] || type || '—')
  if (type === 'EYEBALL' && i && i.eyeball_type === 'ONLINE') return base + t('status.eyeballOnline')
  if (type === 'EYEBALL' && i && i.eyeball_type === 'OFFLINE') return base + t('status.eyeballOffline')
  return base
}

function openKuaidi100() {
  window.open('https://www.kuaidi100.com/chaxun?nu=' + encodeURIComponent(info.value.tracking_number), '_blank')
}
async function copyTracking() {
  try {
    await navigator.clipboard.writeText(info.value.tracking_number)
    ElMessage.success(t('status.trackingCopied'))
  } catch (e) { ElMessage.error(t('status.copyTrackingFailed')) }
}
const fmtTime = s => {
  if (!s) return '—'
  const raw = String(s)
  // 后端时间已存 UTC+8 的直接截取到分钟；RFC3339（含 Z）转本地时区
  if (raw.includes('T')) {
    const d = new Date(raw)
    if (!isNaN(d.getTime())) {
      const p = n => String(n).padStart(2, '0')
      return d.getFullYear() + '-' + p(d.getMonth() + 1) + '-' + p(d.getDate()) + ' ' + p(d.getHours()) + ':' + p(d.getMinutes())
    }
  }
  return raw.slice(0, 16)
}

async function load(code) {
  loading.value = true
  error.value = ''
  info.value = null
  try {
    const res = await api.get('/public/exchange-status/' + encodeURIComponent(code))
    info.value = res
    error.value = ''
  } catch (e) {
    error.value = e.response?.data?.message || t('status.notFound')
  } finally {
    loading.value = false
  }
}

async function refresh() {
  const code = info.value?.request_code || route.params.code || manualCode.value
  if (code) await load(code)
}

function lookup() {
  if (!manualCode.value) return
  load(manualCode.value.toUpperCase())
}

function reset() {
  info.value = null
  error.value = ''
  manualCode.value = ''
  if (route.params.code) router.push('/status')
}

onMounted(() => {
  loadSiteInfo()
  if (route.params.code) {
    manualCode.value = route.params.code
    load(route.params.code)
  }
})
</script>

<style scoped>
.public-page {
  min-height: 100vh;
  color: var(--qsl-ink);
  background: var(--qsl-bg);
  background-image: radial-gradient(at 10% 10%, color-mix(in srgb, var(--qsl-primary) 6%, transparent) 0px, transparent 50%),
                    radial-gradient(at 90% 15%, color-mix(in srgb, var(--qsl-green) 5%, transparent) 0px, transparent 50%);
  padding: 24px 20px;
}
.public-container { max-width: 720px; margin: 0 auto; }
.public-header { text-align: center; padding: 36px 0 28px; }
.logo-section { display: flex; align-items: center; justify-content: center; gap: 14px; }
.logo-icon {
  width: 48px;
  height: 48px;
  background: var(--qsl-primary);
  color: var(--qsl-on-primary);
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 20px;
  font-weight: 850;
  box-shadow: 0 4px 14px color-mix(in srgb, var(--qsl-primary) 30%, transparent);
}
.public-header h1 {
  margin: 0;
  color: var(--qsl-heading);
  font-size: 26px;
  font-weight: 750;
  letter-spacing: -0.02em;
}
.subtitle { margin: 6px 0 0; color: var(--qsl-muted); font-size: 14px; }
.modern-card {
  background: var(--qsl-card-bg);
  border: 1px solid var(--qsl-line);
  border-radius: 18px;
  padding: 36px 32px;
  box-shadow: var(--qsl-shadow-2);
  margin-bottom: 24px;
}
.form-actions { margin-top: 24px; text-align: center; }
.submit-btn { width: 200px; height: 48px; font-size: 15.5px; font-weight: 650; border-radius: 10px; }
.refresh-note { margin-top: 24px; text-align: center; color: var(--qsl-muted); font-size: 13px; }
.public-footer { text-align: center; padding: 24px 0 36px; color: var(--qsl-muted); font-size: 13px; }

.mail-block {
  margin-top: 28px;
  padding: 18px 22px;
  background: var(--qsl-panel-2);
  border: 1px solid var(--qsl-line);
  border-radius: var(--qsl-radius);
}
.mail-head { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
.mail-head b { color: var(--qsl-heading); font-weight: 700; }
.addr-text {
  margin: 12px 0 0;
  padding: 14px 16px;
  background: var(--qsl-panel);
  border: 1px dashed var(--qsl-line);
  border-radius: var(--qsl-radius);
  font-family: inherit;
  font-size: 14px;
  line-height: 1.8;
  white-space: pre-wrap;
  word-break: break-all;
  color: var(--qsl-ink);
}
.sender-mail {
  font-family: monospace;
  font-weight: 700;
  color: var(--qsl-primary);
  background: var(--qsl-active-tint);
  border: 1px solid var(--qsl-line);
  padding: 2px 8px;
  border-radius: 6px;
  word-break: break-all;
}
.mail-actions { margin-top: 12px; }

.notice-block {
  margin: -8px 0 24px;
  padding: 16px 20px;
  background: color-mix(in srgb, var(--qsl-yellow) 12%, var(--qsl-panel));
  border: 1px solid color-mix(in srgb, var(--qsl-yellow) 30%, var(--qsl-line));
  border-left: 4px solid var(--qsl-yellow);
  border-radius: var(--qsl-radius);
}
.notice-head {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--qsl-heading);
  font-size: 14px;
  font-weight: 700;
  margin-bottom: 8px;
}
.notice-text {
  margin: 0;
  font-family: inherit;
  font-size: 13.5px;
  line-height: 1.8;
  white-space: pre-wrap;
  word-break: break-all;
  color: var(--qsl-ink);
}
</style>
