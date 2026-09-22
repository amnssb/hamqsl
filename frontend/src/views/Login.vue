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

        <el-form :model="form" @submit.prevent="handleLogin" label-position="top" class="login-form">
          <el-form-item label="管理员账号">
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
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { ElMessage } from 'element-plus'

const router = useRouter()
const auth = useAuthStore()
const loading = ref(false)
const form = reactive({ username: '', password: '' })

async function handleLogin() {
  if (!form.username || !form.password) {
    ElMessage.warning('请输入用户名和密码喵')
    return
  }
  loading.value = true
  try {
    await auth.login(form.username, form.password)
    ElMessage.success('登录成功喵')
    router.push('/admin/dashboard')
  } catch {
    ElMessage.error('用户名或密码错误喵')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-container {
  min-height: 100vh;
  display: grid;
  grid-template-columns: minmax(360px, 1fr) minmax(460px, 1.25fr);
  background: var(--qsl-paper, #f8fafc);
}

.login-aside {
  position: relative;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  padding: 48px;
  overflow: hidden;
  color: #fff;
  background: linear-gradient(145deg, #0b132b 0%, #1c2541 60%, #1e3a8a 100%);
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
  border: 1px solid rgba(96, 165, 250, 0.15);
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
  background: #3b82f6;
  box-shadow: 0 0 16px #60a5fa;
}
.station-pulse {
  position: absolute;
  top: -6px;
  left: -6px;
  width: 24px;
  height: 24px;
  border-radius: 50%;
  border: 2px solid #60a5fa;
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
  color: #ffffff;
  background: linear-gradient(135deg, #3b82f6 0%, #2563eb 100%);
  border-radius: 10px;
  font-size: 20px;
  font-weight: 850;
  box-shadow: 0 4px 14px rgba(37, 99, 235, 0.4);
}
.brand-text strong {
  display: block;
  font-size: 16px;
  letter-spacing: 0.08em;
  color: #f8fafc;
}
.brand-text span {
  font-size: 10px;
  letter-spacing: 0.12em;
  color: #93c5fd;
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
  color: #93c5fd;
  background: rgba(59, 130, 246, 0.15);
  border: 1px solid rgba(147, 197, 253, 0.3);
  padding: 4px 12px;
  border-radius: 9999px;
  margin-bottom: 20px;
}
.aside-kicker {
  margin: 0 0 18px;
  color: #ffffff;
  font-size: clamp(28px, 3.5vw, 42px);
  font-weight: 850;
  line-height: 1.1;
  letter-spacing: -0.03em;
}
.aside-copy {
  max-width: 380px;
  margin: 0;
  color: #cbd5e1;
  font-size: 14.5px;
  line-height: 1.75;
}

.aside-footer-note {
  position: relative;
  z-index: 2;
  display: flex;
  align-items: center;
  gap: 10px;
  color: #64748b;
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.06em;
}
.freq-tag {
  color: #60a5fa;
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
  border-radius: 16px;
  background: #ffffff;
  border: 1px solid rgba(226, 232, 240, 0.9);
  padding: 36px 32px;
  box-shadow: 0 10px 25px -5px rgba(0, 0, 0, 0.05), 0 8px 10px -6px rgba(0, 0, 0, 0.03);
}

.login-header {
  margin-bottom: 28px;
}
.welcome-badge {
  color: #2563eb;
  font-size: 11px;
  font-weight: 800;
  letter-spacing: 0.12em;
  margin-bottom: 8px;
}
.login-header h1 {
  margin: 0 0 6px;
  color: #0f172a;
  font-size: 26px;
  font-weight: 800;
  letter-spacing: -0.02em;
}
.login-header p {
  margin: 0;
  color: #64748b;
  font-size: 13.5px;
  line-height: 1.5;
}

.login-form :deep(.el-form-item__label) {
  color: #334155;
  font-weight: 650;
  font-size: 13.5px;
  margin-bottom: 6px;
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
  border-top: 1px solid #f1f5f9;
  font-size: 12px;
}
.system-status {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #64748b;
}
.status-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: #10b981;
  box-shadow: 0 0 0 2px rgba(16, 185, 129, 0.25);
}
.back-home-btn {
  color: #2563eb;
  font-size: 12px;
  padding: 0;
}
.back-home-btn:hover {
  color: #1d4ed8;
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
