<template>
  <div class="public-page">
    <div class="public-container">
      <header class="public-header">
        <div class="logo-section">
          <div class="logo-icon">QSL</div>
          <div>
            <h1>{{ t('confirm.title') }}</h1>
            <p class="subtitle">{{ t('confirm.subtitle') }}</p>
          </div>
        </div>
      </header>

      <main class="public-main">
        <!-- 手动输入（URL 无编号时） -->
        <div v-if="!cardInfo && !loading && !error" class="card modern-card">
          <div class="card-header">
            <h2>{{ t('confirm.manualTitle') }}</h2>
          </div>
          <el-form label-position="top" class="modern-form">
            <el-form-item :label="t('confirm.codeLabel')">
              <el-input
                v-model="manualCode"
                :placeholder="t('confirm.codePlaceholder')"
                size="large"
                prefix-icon="Ticket"
                @keyup.enter="lookupCard"
              />
            </el-form-item>
            <div class="form-actions">
              <el-button type="primary" size="large" :loading="loading" @click="lookupCard" class="submit-btn">
                <el-icon><Search /></el-icon> {{ t('confirm.lookup') }}
              </el-button>
            </div>
          </el-form>
        </div>

        <!-- 加载中 -->
        <div v-if="loading" class="card modern-card" style="text-align:center;padding:60px;">
          <el-icon class="is-loading" :size="32"><Loading /></el-icon>
          <p style="margin-top:12px;color:#666;">{{ t('confirm.loading') }}</p>
        </div>

        <!-- 查询失败 -->
        <div v-if="error" class="card modern-card" style="text-align:center;padding:40px;">
          <el-icon :size="48" color="#f56c6c"><CircleClose /></el-icon>
          <h2 style="color:#f56c6c;margin:16px 0 8px;">{{ t('confirm.notFound') }}</h2>
          <p style="color:#666;">{{ error }}</p>
          <el-button type="primary" size="large" @click="resetAll" style="margin-top:20px;">{{ t('confirm.retry') }}</el-button>
        </div>

        <!-- 卡片信息 + 确认表单 -->
        <div v-if="cardInfo && !viewConfirmed" class="card modern-card">
          <div class="card-header">
            <h2>{{ t('confirm.infoTitle') }}</h2>
          </div>

          <div class="card-details">
            <div class="detail-row">
              <span class="label">{{ t('confirm.labelCode') }}</span>
              <span class="value highlight">{{ cardInfo.card_code }}</span>
            </div>
            <div class="detail-row">
              <span class="label">{{ t('confirm.labelCall') }}</span>
              <span class="value">{{ cardInfo.call_sign }}</span>
            </div>
            <div class="detail-row">
              <span class="label">{{ t('confirm.labelVersion') }}</span>
              <span class="value">{{ cardInfo.card_version || '—' }}</span>
            </div>
            <div class="detail-row">
              <span class="label">{{ t('confirm.labelType') }}</span>
              <span class="value">{{ cardInfo.card_type || '—' }}</span>
            </div>
            <div class="detail-row">
              <span class="label">{{ t('confirm.labelStatus') }}</span>
              <span class="value">
                <el-tag :type="statusTagType(cardInfo.flow_status)" size="small">{{ statusText(cardInfo.flow_status) }}</el-tag>
              </span>
            </div>
          </div>

          <el-alert v-if="!cardInfo.card_sent" type="warning" :closable="false" style="margin-bottom:16px;">
            {{ t('confirm.notSent') }}
          </el-alert>
          <el-alert v-if="cardInfo.receipt_confirmed" type="success" :closable="false" style="margin-bottom:16px;">
            {{ t('confirm.alreadySigned') }}
          </el-alert>

          <el-divider>{{ t('confirm.dividerConfirm') }}</el-divider>

          <el-form :model="confirmForm" label-position="top">
            <el-form-item :label="t('confirm.receivedDateLabel')">
              <el-date-picker
                v-model="confirmForm.received_date"
                type="date"
                value-format="YYYY-MM-DD"
                :placeholder="t('confirm.receivedDatePlaceholder')"
                size="large"
                style="width:100%"
              />
            </el-form-item>
            <el-form-item :label="t('confirm.remarksLabel')">
              <el-input
                v-model="confirmForm.remarks"
                type="textarea"
                :rows="2"
                :placeholder="t('confirm.remarksPlaceholder')"
              />
            </el-form-item>
            <div class="form-actions">
              <el-button
                type="success"
                size="large"
                :loading="confirming"
                :disabled="cardInfo.receipt_confirmed || !cardInfo.card_sent"
                @click="handleConfirm"
                class="submit-btn"
              >
                <el-icon><Check /></el-icon> {{ t('confirm.btnConfirm') }}
              </el-button>
            </div>
          </el-form>
        </div>

        <!-- 确认成功（含回寄引导） -->
        <div v-if="viewConfirmed" class="card modern-card success-card">
          <div class="success-content">
            <div class="success-icon">
              <el-icon :size="80" color="#10b981"><CircleCheck /></el-icon>
            </div>
            <h2>{{ t('confirm.successTitle') }}</h2>
            <p>{{ t('confirm.successPrefix') }} <strong>{{ cardInfo?.card_code }}</strong> {{ t('confirm.successSuffix') }}</p>
            <p v-if="confirmed" class="success-time">{{ t('confirm.confirmTime') }} {{ new Date().toLocaleString('zh-CN') }}</p>
            <div class="success-details">
              <div class="detail-item">
                <span>{{ t('confirm.successLabelCode') }}</span>
                <strong>{{ cardInfo?.card_code }}</strong>
              </div>
              <div class="detail-item">
                <span>{{ t('confirm.successLabelCall') }}</span>
                <strong>{{ cardInfo?.call_sign }}</strong>
              </div>
              <div v-if="confirmed" class="detail-item">
                <span>{{ t('confirm.successLabelDate') }}</span>
                <strong>{{ confirmForm.received_date }}</strong>
              </div>
            </div>

            <!-- 回寄引导：仅后台为本卡开通后显示；获取本台地址 → 登记回寄 -->
            <div v-if="cardInfo?.return_mail_enabled || returnDone" class="return-box">
              <template v-if="returnDone">
                <h3>{{ t('confirm.returnDoneTitle') }}</h3>
                <p class="return-done-line">
                  {{ cardInfo?.return_mail_type === 'REGISTERED' ? t('confirm.registered') : t('confirm.ordinary') }}
                  <template v-if="cardInfo?.return_tracking"> · {{ t('confirm.trackingLabel') }} <strong>{{ cardInfo.return_tracking }}</strong></template>
                  <template v-if="cardInfo?.return_mailed_at"> · {{ cardInfo.return_mailed_at }}</template>
                </p>
                <p class="return-tip">{{ t('confirm.returnDoneTip') }}</p>
                <el-form label-position="top" style="max-width:420px;margin:0 auto;text-align:left;">
                  <el-form-item :label="t('confirm.fixTypeLabel')">
                    <el-radio-group v-model="returnForm.mail_type">
                      <el-radio value="REGISTERED">{{ t('confirm.registered') }}</el-radio>
                      <el-radio value="ORDINARY">{{ t('confirm.ordinary') }}</el-radio>
                    </el-radio-group>
                  </el-form-item>
                  <el-form-item v-if="returnForm.mail_type === 'REGISTERED'" :label="t('confirm.fixTrackingLabel')">
                    <el-input v-model="returnForm.tracking_number" :placeholder="t('confirm.trackingPlaceholder')" clearable />
                  </el-form-item>
                  <el-button type="primary" :loading="returnSubmitting" @click="submitReturn" style="width:100%;">{{ t('confirm.btnUpdateReturn') }}</el-button>
                </el-form>
              </template>
              <template v-else>
                <h3>{{ t('confirm.returnTitle') }}</h3>
                <p class="return-tip">{{ t('confirm.returnIntro') }}</p>
                <div v-if="stationAddress" class="station-address"><pre>{{ stationAddress }}</pre></div>
                <el-alert v-else type="info" :closable="false" style="text-align:left;">
                  {{ t('confirm.noStationAddress') }}
                </el-alert>
                <el-form label-position="top" style="max-width:420px;margin:16px auto 0;text-align:left;">
                  <el-form-item :label="t('confirm.yourMailType')">
                    <el-radio-group v-model="returnForm.mail_type">
                      <el-radio value="REGISTERED">{{ t('confirm.registered') }}</el-radio>
                      <el-radio value="ORDINARY">{{ t('confirm.ordinary') }}</el-radio>
                    </el-radio-group>
                  </el-form-item>
                  <el-form-item v-if="returnForm.mail_type === 'REGISTERED'" :label="t('confirm.trackingNumberLabel')" required>
                    <el-input v-model="returnForm.tracking_number" :placeholder="t('confirm.trackingPlaceholder')" clearable />
                  </el-form-item>
                  <el-button type="success" :loading="returnSubmitting" @click="submitReturn" style="width:100%;">{{ t('confirm.btnRegisterReturn') }}</el-button>
                </el-form>
                <p class="return-tip" style="margin-top:10px;">{{ t('confirm.returnRegisterTip') }}</p>
              </template>
            </div>
          </div>
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
import { t, locale } from '../../i18n'

const route = useRoute()
const router = useRouter()
const loading = ref(false)
const confirming = ref(false)
const confirmed = ref(false)
const cardInfo = ref(null)
const error = ref('')
const manualCode = ref('')

// 已确认视图：本次确认成功，或卡片本就处于已签收状态（二次扫码也能登记回寄）
const viewConfirmed = computed(() => confirmed.value || !!(cardInfo.value && cardInfo.value.receipt_confirmed))

const confirmForm = reactive({
  received_date: new Date().toISOString().split('T')[0],
  remarks: ''
})

// ---- 回寄登记 ----
const station = ref(null)
const returnSubmitting = ref(false)
const returnForm = reactive({ mail_type: 'REGISTERED', tracking_number: '' })
const returnDone = computed(() => !!(cardInfo.value && cardInfo.value.return_mailed_at))
const stationAddress = computed(() => {
  const s = station.value
  if (!s || (!s.address && !s.address_en)) return ''
  const lines = []
  if (s.call_sign) lines.push(t('confirm.addrCall') + s.call_sign)
  if (s.name) lines.push(t('confirm.addrName') + s.name)
  if (s.postal_code) lines.push(t('confirm.addrPostcode') + s.postal_code)
  if (s.address) lines.push(t('confirm.addrAddress') + s.address)
  if (s.address_en) lines.push(t('confirm.addrAddressEn') + s.address_en)
  return lines.join('\n')
})
async function loadStation() {
  try { station.value = await api.get('/public/station-mail-info') } catch { station.value = null }
}
async function submitReturn() {
  if (returnForm.mail_type === 'REGISTERED' && !returnForm.tracking_number.trim()) {
    ElMessage.warning(t('confirm.warnRegisteredNeedNumber'))
    return
  }
  returnSubmitting.value = true
  try {
    const res = await api.post('/public/return-mail', {
      card_code: cardInfo.value.card_code,
      mail_type: returnForm.mail_type,
      tracking_number: returnForm.tracking_number.trim()
    })
    cardInfo.value = { ...cardInfo.value, return_mail_type: res.return_mail_type, return_tracking: res.return_tracking, return_mailed_at: res.return_mailed_at }
    ElMessage.success(t('confirm.returnSuccessMsg'))
  } catch (e) {
    ElMessage.error(e.response?.data?.message || t('confirm.returnFailMsg'))
  } finally {
    returnSubmitting.value = false
  }
}

const statusMapZh = {
  PENDING_ISSUE: '待制卡', ISSUED: '已制卡', PACKED: '已打包',
  SENT: '已发卡', RECEIVED: '已收卡', SIGNED: '已签收', ERROR: '异常'
}
const statusMapEn = {
  PENDING_ISSUE: 'Awaiting print', ISSUED: 'Printed', PACKED: 'Packed',
  SENT: 'Shipped', RECEIVED: 'Delivered', SIGNED: 'Signed for', ERROR: 'Error'
}
const statusText = s => (locale.current === 'en' ? statusMapEn[s] : statusMapZh[s]) || s || '—'
const statusTagType = s => ({
  PENDING_ISSUE: 'info', ISSUED: 'success', PACKED: 'warning',
  SENT: 'warning', RECEIVED: '', SIGNED: 'success', ERROR: 'danger'
}[s] || 'info')

async function loadCard(code) {
  loading.value = true
  error.value = ''
  cardInfo.value = null
  try {
    const res = await api.get('/public/cards/' + code)
    cardInfo.value = res
    if (res && res.receipt_confirmed) loadStation()
  } catch (e) {
    error.value = e.response?.data?.message || t('confirm.loadFail')
  } finally {
    loading.value = false
  }
}

function lookupCard() {
  if (!manualCode.value) {
    ElMessage.warning(t('confirm.warnPickCode'))
    return
  }
  loadCard(manualCode.value.toUpperCase())
}

async function handleConfirm() {
  if (!confirmForm.received_date) {
    ElMessage.warning(t('confirm.warnPickDate'))
    return
  }
  confirming.value = true
  try {
    await api.post('/public/confirm-receipt', {
      card_code: cardInfo.value.card_code,
      received_date: confirmForm.received_date,
      remarks: confirmForm.remarks
    })
    confirmed.value = true
    loadStation()
    ElMessage.success(t('confirm.successMsg'))
  } catch (e) {
    ElMessage.error(e.response?.data?.message || t('confirm.failMsg'))
  } finally {
    confirming.value = false
  }
}

function resetAll() {
  cardInfo.value = null
  error.value = ''
  manualCode.value = ''
  confirmed.value = false
  if (route.params.code) {
    router.push('/confirm')
  }
}

onMounted(() => {
  if (route.params.code) {
    loadCard(route.params.code)
  }
})
</script>

<style scoped>
.public-page {
  min-height: 100vh;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  padding: 20px;
}
.public-container { max-width: 600px; margin: 0 auto; }
.public-header { text-align: center; padding: 40px 0 30px; color: white; }
.logo-section { display: flex; align-items: center; justify-content: center; gap: 16px; }
.logo-icon { width: 60px; height: 60px; background: rgba(255,255,255,0.2); border-radius: 16px; display: flex; align-items: center; justify-content: center; font-size: 24px; font-weight: bold; backdrop-filter: blur(10px); }
.public-header h1 { margin: 0; font-size: 28px; font-weight: 600; }
.subtitle { margin: 8px 0 0; opacity: 0.9; font-size: 16px; }
.modern-card { background: white; border-radius: 20px; padding: 32px; box-shadow: 0 20px 60px rgba(0,0,0,0.15); margin-bottom: 24px; }
.card-header { margin-bottom: 24px; padding-bottom: 16px; border-bottom: 1px solid #f0f0f0; }
.card-header h2 { margin: 0 0 8px; font-size: 22px; color: #1a1a1a; }
.card-details { margin-bottom: 24px; }
.detail-row { display: flex; justify-content: space-between; align-items: center; padding: 12px 0; border-bottom: 1px solid #f5f5f5; }
.detail-row .label { color: #666; font-size: 14px; }
.detail-row .value { font-weight: 500; color: #1a1a1a; }
.detail-row .value.highlight { color: #667eea; font-size: 18px; font-weight: 600; }
.form-actions { margin-top: 24px; text-align: center; }
.submit-btn { width: 200px; height: 50px; font-size: 16px; border-radius: 12px; }
.success-card { text-align: center; }
.success-content { padding: 20px; }
.success-icon { margin-bottom: 20px; }
.success-content h2 { color: #10b981; margin: 0 0 12px; font-size: 28px; }
.success-content p { color: #666; margin: 8px 0; }
.success-time { color: #999; font-size: 14px; }
.success-details { display: flex; justify-content: center; gap: 40px; margin-top: 24px; padding: 20px; background: #f8f9fa; border-radius: 12px; }
.success-details .detail-item { text-align: center; }
.success-details .detail-item span { display: block; color: #666; font-size: 14px; margin-bottom: 4px; }
.success-details .detail-item strong { color: #1a1a1a; font-size: 16px; }
.public-footer { text-align: center; padding: 30px 0; color: rgba(255,255,255,0.7); font-size: 14px; }
@media (max-width: 640px) {
  .public-header h1 { font-size: 22px; }
  .modern-card { padding: 20px; }
  .success-details { flex-direction: column; gap: 16px; }
}
/* Kumo visual layer */
.public-page { background:var(--qsl-paper); color:var(--qsl-ink); }
.public-header { color:var(--qsl-navy); border-bottom:1px solid var(--qsl-line); text-align:left; padding:34px 0 28px; }
.logo-section { justify-content:flex-start; }
.logo-icon { width:42px; height:42px; border-radius:0; color:var(--qsl-navy); background:var(--qsl-yellow); font-size:14px; }
.public-header h1 { color:var(--qsl-navy); font-size:24px; }
.subtitle { color:var(--qsl-muted); opacity:1; font-size:13px; }
.modern-card { border:1px solid var(--qsl-line); border-radius:2px; box-shadow:3px 3px 0 rgba(24,45,61,.07); }
.card-header { border-bottom-color:#eeeae2; }
.card-header h2 { color:var(--qsl-navy); }
.public-footer { color:#9c978d; border-top:1px solid var(--qsl-line); }

/* 回寄引导 */
.return-box { margin:24px auto 0; max-width:520px; padding:20px 22px; text-align:center; background:#fdf9ee; border:1px dashed var(--qsl-orange); }
.return-box h3 { margin:0 0 8px; color:var(--qsl-navy); font-size:16px; }
.return-tip { margin:6px 0 0; color:var(--qsl-muted); font-size:12.5px; line-height:1.7; }
.return-done-line { margin:6px 0; color:var(--qsl-ink); font-size:14px; }
.station-address { margin:10px auto 0; max-width:420px; padding:12px 16px; text-align:left; background:#fff; border:1px solid var(--qsl-line); }
.station-address pre { margin:0; font-family:inherit; font-size:13.5px; line-height:1.9; white-space:pre-wrap; word-break:break-all; color:var(--qsl-navy); font-weight:600; }
</style>
