<template>
  <div class="login-container">
    <div class="login-aside">
      <div class="radio-waves">
        <div class="wave wave-1"></div>
        <div class="wave wave-2"></div>
        <div class="wave wave-3"></div>
        <div class="station-center">
          <div class="station-pulse"></div>
          <div class="station-core"></div>
        </div>
      </div>

      <div class="login-brand">
        <div class="brand-badge">Q</div>
        <div class="brand-text">
          <strong>QSL / HUB</strong>
          <span>HAM RADIO CLOUD</span>
        </div>
      </div>

      <div class="aside-content">
        <div class="aside-pill">73 & GOOD DX</div>
        <h2 class="aside-kicker">CONNECT ACROSS<br>THE ETHER.</h2>
        <p class="aside-copy">记录每一次空中通联的信号与电波，将呼号与友情凝结为一张张值得永久珍藏的实体 QSL 卡片。</p>
      </div>

      <div class="aside-footer-note">
        <span class="freq-tag">HF / VHF / UHF</span>
        <span class="dot-sep">·</span>
        <span>AMATEUR RADIO STATION WORKBENCH</span>
      </div>
    </div>

    <div class="login-card-wrapper">
      <el-card class="login-card" shadow="never">
        <div class="login-header">
          <div class="welcome-badge">MANAGEMENT PORTAL</div>
          <h1>欢迎登录</h1>
          <p>管理台站日志、QSL 卡片制作与线上换卡业务</p>
        </div>

        <!-- 第一阶段：账号 + 口令 -->
        <el-form
          v-if="step === 'credentials'"
          :model="form"
          @submit.prevent="handleLogin"
          label-position="top"
          class="login-form"
        >
          <el-form-item label="账号">
            <el-input
              v-model="form.username"
              prefix-icon="User"
              placeholder="请输入用户名"
              size="large"
            />
          </el-form-item>
          <el-form-item label="安全密码">
            <el-input
              v-model="form.password"
              type="password"
              prefix-icon="Lock"
              placeholder="请输入登录密码"
              size="large"
              show-password
              @keyup.enter="handleLogin"
            />
          </el-form-item>
          <el-button
            type="primary"
            size="large"
            :loading="loading"
            @click="handleLogin"
            class="login-button"
          >
            <span>进入管理工作台</span>
            <el-icon><Right /></el-icon>
          </el-button>

          <!-- 免密登录：只在服务端启用了通行密钥、且当前是 HTTPS（安全上下文）时出现 -->
          <template v-if="passkeyReady">
            <el-divider class="login-divider">或</el-divider>
            <el-button
              plain
              size="large"
              class="login-button"
              :loading="passkeyLoading"
              @click="handlePasskeyLogin"
            >
              <span>使用通行密钥登录</span>
            </el-button>
          </template>
        </el-form>

        <!-- 第二阶段：口令已经过了，补第二因子 -->
        <el-form
          v-else
          :model="mfa"
          @submit.prevent="handleVerify"
          label-position="top"
          class="login-form"
        >
          <el-alert type="success" :closable="false" class="mfa-hint">
            口令已验证，再完成一步即可进入。
          </el-alert>

          <el-form-item :label="mfa.useRecovery ? '恢复码' : '动态验证码'">
            <el-input
              v-model="mfa.code"
              :placeholder="mfa.useRecovery ? 'ABCDE-FGHJK' : '身份验证器里的 6 位数字'"
              size="large"
              :maxlength="mfa.useRecovery ? 20 : 6"
              @keyup.enter="handleVerify"
            />
          </el-form-item>

          <el-button
            type="primary"
            size="large"
            :loading="loading"
            @click="handleVerify"
            class="login-button"
          >
            <span>完成验证</span>
            <el-icon><Right /></el-icon>
          </el-button>

          <div class="mfa-actions">
            <el-button text @click="mfa.useRecovery = !mfa.useRecovery">
              {{ mfa.useRecovery ? '改用动态验证码' : '手机丢了？用恢复码' }}
            </el-button>
            <el-button v-if="canUsePasskeyMfa" text :loading="passkeyLoading" @click="handlePasskeyMfa">
              改用通行密钥
            </el-button>
            <el-button text @click="backToCredentials">换个账号</el-button>
          </div>
        </el-form>

        <div class="login-footer">
          <div class="system-status">
            <span class="status-dot"></span>
            <span>QSL 本地化管理台站系统在线</span>
          </div>
          <el-button text class="back-home-btn" @click="$router.push('/')">
            <span>返回公开门户</span>
            <el-icon><ArrowRight /></el-icon>
          </el-button>
        </div>
      </el-card>
    </div>
  </div>
</template>

<script setup>
import { reactive, ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { ElMessage } from 'element-plus'
import { passkeySupported } from '../utils/webauthn'
import api from '../api'

const router = useRouter()
const auth = useAuthStore()

const loading = ref(false)
const passkeyLoading = ref(false)
const step = ref('credentials') // credentials → mfa
const form = reactive({ username: '', password: '' })
const mfa = reactive({ ticket: '', code: '', methods: [], useRecovery: false })
const serverPasskey = ref(false)

// 免密登录要求：服务端启用了通行密钥 + 当前是安全上下文（HTTPS）。
// 明文 HTTP 下 navigator.credentials 根本不存在，直接把按钮藏掉，别让用户点了报错。
const passkeyReady = computed(() => serverPasskey.value && passkeySupported())
const canUsePasskeyMfa = computed(() => mfa.methods.includes('passkey'))

onMounted(async () => {
  try {
    const cap = await api.get('/auth/capabilities')
    serverPasskey.value = !!cap?.passkey
  } catch {
    serverPasskey.value = false
  }
})

// 401 由页面自己提示（拦截器不弹）；其他状态码拦截器已经弹过了，避免重复
function show401(e, fallback) {
  if (e?.response?.status === 401) ElMessage.error(fallback)
}

function goDashboard() {
  ElMessage.success('登录成功喵')
  router.push('/admin/dashboard')
}

async function handleLogin() {
  if (!form.username || !form.password) {
    ElMessage.warning('请输入用户名和密码喵')
    return
  }
  loading.value = true
  try {
    const r = await auth.login(form.username, form.password)
    if (r.mfaRequired) {
      mfa.ticket = r.ticket
      mfa.methods = r.methods || []
      mfa.code = ''
      // 没绑动态码就默认切到恢复码输入，省一次点击
      mfa.useRecovery = !mfa.methods.includes('totp')
      step.value = 'mfa'
      return
    }
    goDashboard()
  } catch (e) {
    if (e?.response?.status === 423) return // 已锁定，拦截器已提示
    show401(e, '用户名或密码错误喵')
  } finally {
    loading.value = false
  }
}

async function handleVerify() {
  const code = (mfa.code || '').trim()
  if (!code) {
    ElMessage.warning(mfa.useRecovery ? '请输入恢复码' : '请输入 6 位动态验证码')
    return
  }
  loading.value = true
  try {
    if (mfa.useRecovery) await auth.completeRecovery(mfa.ticket, code)
    else await auth.completeTotp(mfa.ticket, code)
    goDashboard()
  } catch (e) {
    if (e?.response?.status === 410) {
      ElMessage.warning('登录已超时，请重新登录')
      backToCredentials()
      return
    }
    if (e?.response?.status === 429) return // 限速，拦截器已提示
    show401(e, mfa.useRecovery ? '恢复码不正确' : '验证码不正确')
  } finally {
    loading.value = false
  }
}

async function handlePasskeyLogin() {
  if (!passkeyReady.value) {
    ElMessage.warning('当前环境（需 HTTPS）不支持通行密钥')
    return
  }
  passkeyLoading.value = true
  try {
    await auth.loginWithPasskey()
    goDashboard()
  } catch (e) {
    reportPasskeyError(e)
  } finally {
    passkeyLoading.value = false
  }
}

async function handlePasskeyMfa() {
  passkeyLoading.value = true
  try {
    await auth.completePasskey(mfa.ticket)
    goDashboard()
  } catch (e) {
    if (e?.response?.status === 410) {
      ElMessage.warning('登录已超时，请重新登录')
      backToCredentials()
      return
    }
    reportPasskeyError(e)
  } finally {
    passkeyLoading.value = false
  }
}

function reportPasskeyError(e) {
  if (e?.name === 'NotAllowedError') {
    ElMessage.warning('未选择通行密钥，或验证被取消')
    return
  }
  if (e?.name === 'SecurityError' || e?.name === 'NotSupportedError') {
    ElMessage.error('当前浏览器或域名不支持通行密钥（必须 HTTPS）')
    return
  }
  if (!e?.response) ElMessage.error('通行密钥验证失败')
}

function backToCredentials() {
  step.value = 'credentials'
  mfa.ticket = ''
  mfa.code = ''
  mfa.methods = []
  mfa.useRecovery = false
}
</script>

<style scoped>
.login-container {
  min-height: 100vh;
  display: grid;
  grid-template-columns: minmax(360px, 1fr) minmax(460px, 1.25fr);
  background: var(--qsl-bg);
}

/* 左侧品牌板是固定的深色"海报"，不随主题变 —— 它是视觉标识，不是内容面。
   颜色全部收敛到 --qsl-poster-* token（定义在 tokens.css），组件里不写裸色值。 */
.login-aside {
  position: relative;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  padding: 48px;
  overflow: hidden;
  color: var(--qsl-poster-text);
  background: linear-gradient(145deg, var(--qsl-poster-from) 0%, var(--qsl-poster-mid) 60%, var(--qsl-poster-to) 100%);
}

.radio-waves {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  width: 500px;
  height: 500px;
  pointer-events: none;
}

.wave {
  position: absolute;
  top: 50%;
  left: 50%;
  border-radius: 50%;
  border: 1px solid color-mix(in srgb, var(--qsl-poster-accent-2) 15%, transparent);
  transform: translate(-50%, -50%);
}
.wave-1 {
  width: 180px;
  height: 180px;
  animation: pulse-wave 4s cubic-bezier(0.25, 1, 0.5, 1) infinite;
}
.wave-2 {
  width: 320px;
  height: 320px;
  animation: pulse-wave 4s cubic-bezier(0.25, 1, 0.5, 1) infinite 1.3s;
}
.wave-3 {
  width: 480px;
  height: 480px;
  animation: pulse-wave 4s cubic-bezier(0.25, 1, 0.5, 1) infinite 2.6s;
}

@keyframes pulse-wave {
  0% { transform: translate(-50%, -50%) scale(0.8); opacity: 0.8; }
  100% { transform: translate(-50%, -50%) scale(1.3); opacity: 0; }
}

.station-center {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
}
.station-core {
  width: 12px;
  height: 12px;
  border-radius: 50%;
  background: var(--qsl-poster-accent);
  box-shadow: 0 0 16px var(--qsl-poster-accent-2);
}
.station-pulse {
  position: absolute;
  top: -6px;
  left: -6px;
  width: 24px;
  height: 24px;
  border-radius: 50%;
  border: 2px solid var(--qsl-poster-accent-2);
  animation: core-pulse 2s infinite;
}
@keyframes core-pulse {
  0% { transform: scale(0.6); opacity: 1; }
  100% { transform: scale(1.6); opacity: 0; }
}

.login-brand {
  position: relative;
  z-index: 2;
  display: flex;
  align-items: center;
  gap: 12px;
}
.brand-badge {
  display: grid;
  place-items: center;
  width: 40px;
  height: 40px;
  color: var(--qsl-poster-text);
  background: linear-gradient(135deg, var(--qsl-poster-accent) 0%, var(--qsl-poster-to) 100%);
  border-radius: 10px;
  font-size: 20px;
  font-weight: 850;
  box-shadow: 0 4px 14px color-mix(in srgb, var(--qsl-poster-accent) 40%, transparent);
}
.brand-text strong {
  display: block;
  font-size: 16px;
  letter-spacing: 0.08em;
  color: var(--qsl-poster-text);
}
.brand-text span {
  font-size: 10px;
  letter-spacing: 0.12em;
  color: var(--qsl-poster-line);
  font-weight: 600;
}

.aside-content {
  position: relative;
  z-index: 2;
  margin: 60px 0;
}
.aside-pill {
  display: inline-block;
  font-size: 11px;
  font-weight: 750;
  letter-spacing: 0.14em;
  color: var(--qsl-poster-line);
  background: color-mix(in srgb, var(--qsl-poster-accent) 15%, transparent);
  border: 1px solid color-mix(in srgb, var(--qsl-poster-line) 30%, transparent);
  padding: 4px 12px;
  border-radius: 9999px;
  margin-bottom: 20px;
}
.aside-kicker {
  margin: 0 0 18px;
  color: var(--qsl-poster-text);
  font-size: clamp(28px, 3.5vw, 42px);
  font-weight: 850;
  line-height: 1.1;
  letter-spacing: -0.03em;
}
.aside-copy {
  max-width: 380px;
  margin: 0;
  color: var(--qsl-poster-text-2);
  font-size: 14.5px;
  line-height: 1.75;
}

.aside-footer-note {
  position: relative;
  z-index: 2;
  display: flex;
  align-items: center;
  gap: 10px;
  color: var(--qsl-poster-muted);
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.06em;
}
.freq-tag {
  color: var(--qsl-poster-accent-2);
  font-weight: 700;
}
.dot-sep {
  opacity: 0.5;
}

.login-card-wrapper {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 40px 24px;
}
.login-card {
  width: 100%;
  max-width: 440px;
  border-radius: var(--qsl-radius-card);
  background: var(--qsl-card-bg);
  border: 1px solid var(--qsl-line);
  padding: 36px 32px;
  box-shadow: var(--qsl-shadow-1);
}

.login-header {
  margin-bottom: 28px;
}
.welcome-badge {
  color: var(--el-color-primary);
  font-size: 11px;
  font-weight: 800;
  letter-spacing: 0.12em;
  margin-bottom: 8px;
}
.login-header h1 {
  margin: 0 0 6px;
  color: var(--qsl-navy);
  font-size: 26px;
  font-weight: 800;
  letter-spacing: -0.02em;
}
.login-header p {
  margin: 0;
  color: var(--qsl-muted);
  font-size: 13.5px;
  line-height: 1.5;
}

.login-form :deep(.el-form-item__label) {
  color: var(--qsl-navy);
  font-weight: 650;
  font-size: 13.5px;
  margin-bottom: 6px;
}

.mfa-hint {
  margin-bottom: 16px;
}

.mfa-actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 4px;
  margin-top: 12px;
  font-size: 12px;
}

.login-divider {
  margin: 18px 0 14px;
  font-size: 12px;
  color: var(--qsl-muted);
}

.login-button {
  width: 100%;
  margin-top: 12px;
  height: 46px;
  font-size: 15px;
  font-weight: 650;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
}

.login-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 28px;
  padding-top: 20px;
  border-top: 1px solid var(--qsl-line);
  font-size: 12px;
}
.system-status {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--qsl-muted);
}
.status-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--qsl-green);
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--qsl-green) 25%, transparent);
}
.back-home-btn {
  color: var(--el-color-primary);
  font-size: 12px;
  padding: 0;
}
.back-home-btn:hover {
  color: var(--el-color-primary-hover, var(--el-color-primary));
}

@media (max-width: 860px) {
  .login-container {
    grid-template-columns: 1fr;
  }
  .login-aside {
    padding: 36px 24px;
    min-height: 260px;
  }
  .aside-content {
    margin: 24px 0;
  }
  .aside-kicker {
    font-size: 28px;
  }
  .radio-waves {
    display: none;
  }
  .login-card-wrapper {
    padding: 24px 16px;
  }
  .login-card {
    padding: 24px 20px;
  }
}
</style>
