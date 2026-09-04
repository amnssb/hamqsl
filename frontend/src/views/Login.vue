<template>
  <div class="login-container">
    <div class="login-aside">
      <div class="login-brand"><span>Q</span><strong>QSL / HUB</strong></div>
      <div class="signal-art"><i></i><i></i><i></i><i></i><i></i></div>
      <p class="aside-kicker">KEEP EVERY<br>CONNECTION</p>
      <p class="aside-copy">把每一次通联，变成一张可以抵达的卡片。</p>
    </div>
    <el-card class="login-card" shadow="never">
      <div class="login-header"><span class="eyebrow">WELCOME BACK</span><h1>进入工作台</h1><p>管理你的通联与 QSL 卡片。</p></div>
      <el-form :model="form" @submit.prevent="handleLogin" label-position="top" class="login-form">
        <el-form-item label="用户名"><el-input v-model="form.username" prefix-icon="User" placeholder="请输入用户名" size="large" /></el-form-item>
        <el-form-item label="密码"><el-input v-model="form.password" type="password" prefix-icon="Lock" placeholder="请输入密码" size="large" show-password @keyup.enter="handleLogin" /></el-form-item>
        <el-button type="primary" size="large" :loading="loading" @click="handleLogin" class="login-button">登录工作台 <el-icon><Right /></el-icon></el-button>
      </el-form>
      <div class="login-footer"><span class="status-dot"></span> 本地 QSL 管理系统</div>
    </el-card>
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
    ElMessage.warning('请输入用户名和密码')
    return
  }
  loading.value = true
  try {
    await auth.login(form.username, form.password)
    ElMessage.success('登录成功')
    router.push('/admin/dashboard')
  } catch {
    ElMessage.error('用户名或密码错误')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-container { min-height:100vh; display:grid; grid-template-columns:minmax(280px, .85fr) minmax(400px, 1fr); background:var(--qsl-paper); }
.login-aside { position:relative; display:flex; flex-direction:column; justify-content:flex-end; padding:58px; overflow:hidden; color:#fff; background:var(--qsl-navy); }
.login-brand { position:absolute; top:42px; left:58px; display:flex; align-items:center; gap:12px; font-size:15px; letter-spacing:.1em; }.login-brand span { display:grid; place-items:center; width:35px; height:35px; color:var(--qsl-navy); background:var(--qsl-yellow); font-size:20px; font-weight:850; }.login-brand strong { font-size:14px; }
.signal-art { position:absolute; top:29%; left:18%; width:300px; height:300px; border:1px solid rgba(255,255,255,.16); border-radius:50%; }.signal-art:before,.signal-art:after { content:""; position:absolute; inset:28px; border:1px solid rgba(255,255,255,.12); border-radius:50%; }.signal-art:after { inset:78px; }.signal-art i { position:absolute; top:50%; left:50%; width:8px; height:8px; border-radius:50%; background:var(--qsl-orange); box-shadow:0 0 0 8px rgba(255,107,53,.15); transform:translate(-50%,-50%); }.signal-art i:nth-child(2) { transform:translate(-50%,-50%) rotate(45deg) translateY(-105px); }.signal-art i:nth-child(3) { transform:translate(-50%,-50%) rotate(135deg) translateY(-105px); }.signal-art i:nth-child(4) { transform:translate(-50%,-50%) rotate(225deg) translateY(-105px); }.signal-art i:nth-child(5) { transform:translate(-50%,-50%) rotate(315deg) translateY(-105px); }
.aside-kicker { margin:0 0 17px; color:var(--qsl-yellow); font-size:clamp(30px,4vw,54px); font-weight:850; line-height:.95; letter-spacing:-.06em; }.aside-copy { max-width:280px; margin:0; color:#b8c7cf; font-size:14px; line-height:1.7; }
.login-card { align-self:center; width:min(430px, calc(100% - 48px)); margin:auto; border:0; background:transparent; box-shadow:none; }.login-header { margin-bottom:36px; }.eyebrow { color:var(--qsl-orange); font-size:10px; font-weight:800; letter-spacing:.16em; }.login-header h1 { margin:12px 0 8px; color:var(--qsl-navy); font-size:34px; letter-spacing:-.06em; }.login-header p { margin:0; color:var(--qsl-muted); font-size:14px; }.login-form :deep(.el-form-item__label) { color:var(--qsl-navy); font-weight:700; }.login-button { width:100%; margin-top:10px; height:46px; }.login-footer { display:flex; align-items:center; gap:8px; margin-top:30px; color:#a09c94; font-size:11px; }.status-dot { width:7px; height:7px; border-radius:50%; background:var(--qsl-green); }
@media (max-width:700px) { .login-container { display:block; padding:24px 0; }.login-aside { min-height:220px; padding:30px; }.login-brand { top:28px; left:30px; }.signal-art { top:20px; right:-70px; left:auto; width:230px; height:230px; }.aside-kicker { margin-top:80px; font-size:33px; }.login-card { width:min(430px, calc(100% - 40px)); padding:30px 0; } }
</style>
