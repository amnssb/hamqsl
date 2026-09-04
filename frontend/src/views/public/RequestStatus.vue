<template>
  <div class="public-page">
    <div class="public-container">
      <header class="public-header">
        <div class="logo-section">
          <div class="logo-icon">QSL</div>
          <div>
            <h1>申请进度查询</h1>
            <p class="subtitle" v-if="!info">输入申请编号，查看审核、制卡、邮寄与签收状态</p>
          </div>
        </div>
      </header>

      <main class="public-main">
        <!-- 手动输入（URL 无编号时） -->
        <div v-if="!info && !loading && !error" class="card modern-card">
          <el-form label-position="top" @submit.prevent>
            <el-form-item label="申请编号">
              <el-input v-model="manualCode" placeholder="例如: EXAB23CD9" size="large" @keyup.enter="lookup" />
            </el-form-item>
            <div class="form-actions">
              <el-button type="primary" size="large" :loading="loading" @click="lookup" class="submit-btn">
                <el-icon><Search /></el-icon> 查询进度
              </el-button>
            </div>
          </el-form>
        </div>

        <!-- 加载中 -->
        <div v-if="loading && !info" class="card modern-card" style="text-align:center;padding:60px;">
          <el-icon class="is-loading" :size="32"><Loading /></el-icon>
          <p style="margin-top:12px;color:#666;">正在查询申请进度...</p>
        </div>

        <!-- 查询失败 -->
        <div v-if="error" class="card modern-card" style="text-align:center;padding:40px;">
          <el-icon :size="48" color="#f56c6c"><CircleClose /></el-icon>
          <h2 style="color:#f56c6c;margin:16px 0 8px;">查询失败</h2>
          <p style="color:#666;">{{ error }}</p>
          <el-button @click="reset" style="margin-top:16px;">重新输入</el-button>
        </div>

        <!-- 进度详情 -->
        <div v-if="info" class="card modern-card">
          <!-- 站点公告（管理员在设置中配置，留空不显示） -->
          <div v-if="siteNotice" class="notice-block">
            <div class="notice-head"><el-icon><BellFilled /></el-icon><b>公告</b></div>
            <pre class="notice-text">{{ siteNotice }}</pre>
          </div>

          <el-descriptions :column="2" border style="margin-bottom:28px;">
            <el-descriptions-item label="申请编号">{{ info.request_code }}</el-descriptions-item>
            <el-descriptions-item label="呼号">{{ info.call_sign }}</el-descriptions-item>
            <el-descriptions-item label="申请场景">{{ sceneText(info.scene_type) }}</el-descriptions-item>
            <el-descriptions-item label="申请时间">{{ fmtTime(info.created_at) }}</el-descriptions-item>
          </el-descriptions>

          <el-steps :active="activeStep" align-center :process-status="rejected ? 'error' : 'process'" finish-status="success">
            <el-step v-for="(s, idx) in stepsData" :key="idx" :title="s.title" :description="s.desc" />
          </el-steps>

          <el-alert v-if="rejected" type="error" :closable="false" style="margin-top:28px;">
            申请未通过：{{ info.review_reason || '未填写原因' }}
          </el-alert>
          <div v-else-if="info.tracking_number && info.flow_status !== 'SIGNED'" class="mail-block">
            <div class="mail-head">
              <b>物流查询</b>
              <el-tag size="small" style="font-family:monospace;">{{ info.tracking_number }}</el-tag>
            </div>
            <p class="mail-tip">复制下方单号，到快递100 官网即可查询实时物流：</p>
            <div class="mail-actions">
              <el-button type="primary" @click="openKuaidi100">在快递100 查询物流</el-button>
              <el-button @click="copyTracking">复制单号</el-button>
            </div>
          </div>

          <!-- SWL 反寄：回寄地址（管理员发送后展示） -->
          <div v-if="info.return_address_text && !info.return_mailed_at" class="mail-block">
            <div class="mail-head"><b>请将您的卡片寄至以下地址</b></div>
            <pre class="addr-text">{{ info.return_address_text }}</pre>
          </div>

          <!-- SWL 反寄：对方寄出后登记单号/平信 -->
          <div v-if="info.scene_type === 'SWL' && info.review_status === 'APPROVED'" class="mail-block">
            <template v-if="!info.return_mailed_at">
              <div class="mail-head"><b>登记您的寄出信息</b></div>
              <p class="mail-tip">您寄出收听卡后在此登记，我们会收到记录并尽快处理回寄。</p>
              <el-form label-position="top" style="margin-top:8px;">
                <el-form-item label="邮寄方式">
                  <el-radio-group v-model="returnForm.mail_type">
                    <el-radio value="REGISTERED">挂号信</el-radio>
                    <el-radio value="ORDINARY">平信</el-radio>
                  </el-radio-group>
                </el-form-item>
                <el-form-item v-if="returnForm.mail_type === 'REGISTERED'" label="单号">
                  <el-input v-model="returnForm.tracking_number" placeholder="您的挂号信/快递单号" />
                </el-form-item>
                <el-button type="primary" :loading="returnSaving" @click="submitReturnMail">提交登记</el-button>
              </el-form>
            </template>
            <template v-else>
              <div class="mail-head">
                <b>您的寄出登记</b>
                <el-tag size="small" :type="info.return_mail_type === 'REGISTERED' ? '' : 'info'">{{ info.return_mail_type === 'REGISTERED' ? '挂号信' : '平信' }}</el-tag>
              </div>
              <p v-if="info.return_tracking" class="mail-tip">单号：<span style="font-family:monospace;">{{ info.return_tracking }}</span></p>
              <p class="mail-tip">登记时间：{{ fmtTime(info.return_mailed_at) }}</p>
              <el-button v-if="info.return_tracking" size="small" @click="openReturnTracking">在快递100 查询</el-button>
            </template>
          </div>

          <!-- 邮件未收到引导：申请通过后各节点均有邮件提醒，教对方查垃圾箱+加白名单 -->
          <div v-if="info && info.review_status === 'APPROVED' && senderEmail" class="mail-block">
            <div class="mail-head"><b>邮件未收到？</b></div>
            <p class="mail-tip">
              审核通过、收卡、回寄卡片寄出等节点都会向您的邮箱发送提醒。若未收到，请检查垃圾邮件箱（Spam），并将发件邮箱
              <span class="sender-mail">{{ senderEmail }}</span>
              添加到白名单，以便后续通知正常送达。
            </p>
            <div class="mail-actions">
              <el-button size="small" @click="copySenderEmail">复制发件邮箱</el-button>
            </div>
          </div>

          <p class="refresh-note">卡片编号：{{ info.card_code || '尚未生成' }}<el-button link type="primary" size="small" style="margin-left:8px;" @click="refresh">手动刷新</el-button></p>
        </div>
      </main>

      <footer class="public-footer">
        <p>QSL 卡片管理系统 · 业余无线电</p>
      </footer>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import api from '../../api'
import { ElMessage } from 'element-plus'

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
    ElMessage.success('发件邮箱已复制：' + senderEmail.value)
  } catch (e) { ElMessage.error('复制失败，请手动选择邮箱复制') }
}

async function submitReturnMail() {
  if (!info.value?.request_code) return
  if (returnForm.mail_type === 'REGISTERED' && !returnForm.tracking_number) {
    ElMessage.warning('挂号信请填写单号')
    return
  }
  returnSaving.value = true
  try {
    await api.post('/public/exchange-return-mail', {
      request_code: info.value.request_code,
      mail_type: returnForm.mail_type,
      tracking_number: returnForm.tracking_number
    })
    ElMessage.success('登记成功，我们已收到您的寄出记录')
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
  if (i.review_status === 'PENDING') return '审核中'
  if (i.review_status === 'APPROVED') return '已通过'
  if (i.review_status === 'REJECTED') return '未通过'
  return ''
})
const cardDesc = computed(() => {
  const i = info.value
  if (!i) return ''
  return i.card_created ? ('已建卡 ' + (i.card_code || '')) : '等待建卡'
})
const mailDesc = computed(() => {
  const i = info.value
  if (!i) return ''
  if (['SENT', 'RECEIVED', 'SIGNED'].includes(i.flow_status)) return '已寄出'
  if (i.card_created) return '准备中'
  return '等待'
})
const signedDesc = computed(() => info.value && info.value.flow_status === 'SIGNED' ? '对方已签收' : '等待签收')

// SWL 反寄：第三步为"您寄出卡"，其后才是我方制卡回寄
const swlSentDesc = computed(() => {
  const i = info.value
  if (!i) return ''
  if (i.return_mailed_at) return '已寄出 ' + fmtTime(i.return_mailed_at)
  if (i.address_sent_at) return '地址已发送，请尽快寄出'
  return '等待回寄地址'
})
const swlReceivedDesc = computed(() => {
  const i = info.value
  if (!i) return ''
  if (i.return_received_at) return '已收到 ' + fmtTime(i.return_received_at)
  if (i.return_mailed_at) return '等待寄达'
  return '—'
})

// EYEBALL：交付方式灵活（见面交付或邮寄），第五步为"完成"
const eyeballDeliverDesc = computed(() => {
  const i = info.value
  if (!i) return ''
  if (i.flow_status === 'SIGNED') return '已交付'
  if (['SENT', 'RECEIVED'].includes(i.flow_status)) return '已寄出'
  if (i.card_created) return '见面交付或邮寄'
  return '等待制卡'
})
const eyeballDoneDesc = computed(() => info.value && info.value.flow_status === 'SIGNED' ? '已完成' : '待确认')

// 按场景生成时间线步骤
const stepsData = computed(() => {
  const i = info.value
  if (!i) return []
  if (i.scene_type === 'SWL') {
    return [
      { title: '提交申请', desc: fmtTime(i.created_at) },
      { title: '审核', desc: reviewDesc.value },
      { title: '您寄出卡', desc: swlSentDesc.value },
      { title: '我方收卡', desc: swlReceivedDesc.value },
      { title: '制卡回寄', desc: cardDesc.value },
      { title: '签收', desc: signedDesc.value },
    ]
  }
  if (i.scene_type === 'EYEBALL') {
    return [
      { title: '提交申请', desc: fmtTime(i.created_at) },
      { title: '审核', desc: reviewDesc.value },
      { title: '制卡', desc: cardDesc.value },
      { title: '交付', desc: eyeballDeliverDesc.value },
      { title: '完成', desc: eyeballDoneDesc.value },
    ]
  }
  return [
    { title: '提交申请', desc: fmtTime(i.created_at) },
    { title: '审核', desc: reviewDesc.value },
    { title: '制卡', desc: cardDesc.value },
    { title: '邮寄', desc: mailDesc.value },
    { title: '签收', desc: signedDesc.value },
  ]
})

const sceneText = (t) => {
  const i = info.value
  const base = ({ QSO: 'QSO 通联', SWL: 'SWL 收听', EYEBALL: 'EYEBALL 见面' }[t] || t || '—')
  if (t === 'EYEBALL' && i && i.eyeball_type === 'ONLINE') return base + '（网络EYE）'
  if (t === 'EYEBALL' && i && i.eyeball_type === 'OFFLINE') return base + '（线下补换）'
  return base
}

function openKuaidi100() {
  window.open('https://www.kuaidi100.com/chaxun?nu=' + encodeURIComponent(info.value.tracking_number), '_blank')
}
async function copyTracking() {
  try {
    await navigator.clipboard.writeText(info.value.tracking_number)
    ElMessage.success('单号已复制')
  } catch (e) { ElMessage.error('复制失败，请手动选择单号复制') }
}
const fmtTime = s => {
  if (!s) return '—'
  const t = String(s)
  // 后端时间已存 UTC+8 的直接截取到分钟；RFC3339（含 Z）转本地时区
  if (t.includes('T')) {
    const d = new Date(t)
    if (!isNaN(d.getTime())) {
      const p = n => String(n).padStart(2, '0')
      return d.getFullYear() + '-' + p(d.getMonth() + 1) + '-' + p(d.getDate()) + ' ' + p(d.getHours()) + ':' + p(d.getMinutes())
    }
  }
  return t.slice(0, 16)
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
    error.value = e.response?.data?.message || '申请不存在或编号错误'
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
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  padding: 20px;
}
.public-container { max-width: 720px; margin: 0 auto; }
.public-header { text-align: center; padding: 40px 0 30px; color: white; }
.logo-section { display: flex; align-items: center; justify-content: center; gap: 16px; }
.logo-icon { width: 60px; height: 60px; background: rgba(255,255,255,0.2); border-radius: 16px; display: flex; align-items: center; justify-content: center; font-size: 24px; font-weight: bold; backdrop-filter: blur(10px); }
.public-header h1 { margin: 0; font-size: 28px; font-weight: 600; }
.subtitle { margin: 8px 0 0; opacity: 0.9; font-size: 16px; }
.modern-card { background: white; border-radius: 20px; padding: 32px; box-shadow: 0 20px 60px rgba(0,0,0,0.15); margin-bottom: 24px; }
.form-actions { margin-top: 24px; text-align: center; }
.submit-btn { width: 200px; height: 50px; font-size: 16px; border-radius: 12px; }
.refresh-note { margin-top: 24px; text-align: center; color: #909399; font-size: 13px; }
.public-footer { text-align: center; padding: 30px 0; color: rgba(255,255,255,0.7); font-size: 14px; }

.mail-block { margin-top: 28px; padding: 16px 20px; background: #f8f7f2; border: 1px solid #e8e4dc; border-radius: 4px; }
.mail-head { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
.mail-head b { color: var(--qsl-navy, #1a1a1a); }
.addr-text { margin: 10px 0 0; padding: 12px 14px; background: #fff; border: 1px dashed #d8d2c6; border-radius: 4px; font-family: inherit; font-size: 14px; line-height: 1.8; white-space: pre-wrap; word-break: break-all; color: var(--qsl-ink, #333); }
.sender-mail { font-family: monospace; font-weight: 700; color: var(--qsl-navy, #1a2d3d); background: #fff; border: 1px dashed #d8d2c6; padding: 1px 6px; border-radius: 3px; word-break: break-all; }
.mail-actions { margin-top: 10px; }

.notice-block { margin: -8px 0 24px; padding: 14px 18px; background: #fdf6e8; border: 1px solid #ecd9b0; border-left: 4px solid #f5a623; border-radius: 4px; }
.notice-head { display: flex; align-items: center; gap: 7px; color: var(--qsl-navy, #1a2d3d); font-size: 14px; margin-bottom: 8px; }
.notice-text { margin: 0; font-family: inherit; font-size: 14px; line-height: 1.8; white-space: pre-wrap; word-break: break-all; color: var(--qsl-ink, #333); }

/* Kumo visual layer */
.public-page { background:var(--qsl-paper); color:var(--qsl-ink); }
.public-header { color:var(--qsl-navy); border-bottom:1px solid var(--qsl-line); text-align:left; padding:34px 0 28px; }
.logo-section { justify-content:flex-start; }
.logo-icon { width:42px; height:42px; border-radius:0; color:var(--qsl-navy); background:var(--qsl-yellow); font-size:14px; }
.public-header h1 { color:var(--qsl-navy); font-size:24px; }
.subtitle { color:var(--qsl-muted); opacity:1; font-size:13px; }
.modern-card { border:1px solid var(--qsl-line); border-radius:2px; box-shadow:3px 3px 0 rgba(24,45,61,.07); }
.public-footer { color:#9c978d; border-top:1px solid var(--qsl-line); }
</style>
