<template>
  <div>
    <el-card style="margin-bottom:20px;">
      <template #header>
        <div style="display:flex;justify-content:space-between;align-items:center;">
          <span>站点设置</span>
          <el-button type="primary" @click="saveSite" :loading="savingSite">保存</el-button>
        </div>
      </template>
      <el-alert type="info" :closable="false" style="margin-bottom:20px;">
        站点名称显示在公开门户首页与浏览器标签页；站点地址用于审批通知等邮件中的进度查询链接；通知邮箱用于新申请实时提醒（需先配置 SMTP）；公告栏显示在对方查看申请进度的页面顶部。留空时站点名称使用默认值、公告栏不显示。
      </el-alert>
      <el-form label-width="100px" style="max-width:640px;">
        <el-form-item label="站点名称"><el-input v-model="siteForm.site_name" placeholder="QSL 卡片管理系统" maxlength="50" show-word-limit /></el-form-item>
        <el-form-item label="站点地址"><el-input v-model="siteForm.site_url" placeholder="https://qsl.example.com" /></el-form-item>
        <el-form-item label="通知邮箱"><el-input v-model="siteForm.notify_email" placeholder="有人提交新申请时，邮件通知此邮箱（留空则不通知）" /></el-form-item>
        <el-form-item label="公告栏"><el-input v-model="siteForm.site_notice" type="textarea" :rows="4" maxlength="500" show-word-limit placeholder="显示在申请进度页顶部，如：近期寄卡高峰，回寄周期约 2-3 周；留空则不显示" /></el-form-item>
      </el-form>
    </el-card>

    <el-card style="margin-bottom:20px;">
      <template #header>
        <div style="display:flex;justify-content:space-between;align-items:center;">
          <span>SMTP 邮件配置</span>
          <div>
            <el-button @click="testSmtp" :loading="testLoading" :disabled="!smtpForm.smtp_host">发送测试邮件</el-button>
            <el-button type="primary" @click="saveSmtp()" :loading="savingSmtp">保存配置</el-button>
          </div>
        </div>
      </template>
      <el-alert type="info" :closable="false" style="margin-bottom:20px;">
        配置 SMTP 后，SWL 审批通过、回寄地址发送等通知将实际发送到对方邮箱。密码/授权码保存后不再回显（显示为空），留空保存即保持原密码；点击「发送测试邮件」会先自动保存当前配置再发送。
      </el-alert>
      <el-form :model="smtpForm" label-width="120px" style="max-width:640px;">
        <el-form-item label="SMTP 服务器"><el-input v-model="smtpForm.smtp_host" placeholder="smtp.qq.com / smtp.163.com / smtp.gmail.com" /></el-form-item>
        <el-form-item label="端口"><el-input v-model="smtpForm.smtp_port" placeholder="465(SSL) / 587(TLS) / 25" /></el-form-item>
        <el-form-item label="用户名"><el-input v-model="smtpForm.smtp_user" placeholder="发件邮箱地址" /></el-form-item>
        <el-form-item label="密码/授权码"><el-input v-model="smtpForm.smtp_password" type="password" show-password placeholder="留空则保持已保存的密码不变" /></el-form-item>
        <el-form-item label="发件人地址"><el-input v-model="smtpForm.smtp_from" placeholder="留空则使用用户名" /></el-form-item>
        <el-form-item label="发件人名称"><el-input v-model="smtpForm.smtp_from_name" placeholder="如 BI1KBU QSL管理" /></el-form-item>
        <el-form-item label="加密方式">
          <el-radio-group v-model="smtpEncryption">
            <el-radio value="ssl">SSL</el-radio>
            <el-radio value="tls">TLS</el-radio>
            <el-radio value="none">无加密</el-radio>
          </el-radio-group>
        </el-form-item>
      </el-form>
      <el-divider>常见邮箱配置参考</el-divider>
      <el-descriptions :column="1" border size="small">
        <el-descriptions-item label="QQ邮箱">smtp.qq.com | 端口 465(SSL) | 授权码需在QQ邮箱设置中获取</el-descriptions-item>
        <el-descriptions-item label="163邮箱">smtp.163.com | 端口 465(SSL) | 授权码需在163邮箱设置中获取</el-descriptions-item>
        <el-descriptions-item label="Gmail">smtp.gmail.com | 端口 587(TLS) | 需开启应用专用密码</el-descriptions-item>
        <el-descriptions-item label="Outlook">smtp.office365.com | 端口 587(TLS) | 使用账户密码</el-descriptions-item>
      </el-descriptions>
    </el-card>

    <el-card style="margin-bottom:20px;">
      <template #header><span>数据备份</span></template>
      <el-alert type="info" :closable="false" style="margin-bottom:16px;">
        导出系统全部数据（台站、通联日志、卡片记录、地址簿、卡片局、线上换卡申请、收卡记录、系统设置等）为 JSON 文件；敏感配置（SMTP 密码）不包含在内。
      </el-alert>
      <div style="display:flex;gap:12px;align-items:center;flex-wrap:wrap;">
        <el-button type="primary" :loading="exporting" @click="handleExport">导出全部数据（JSON）</el-button>
      </div>

      <el-divider content-position="left">导入恢复</el-divider>
      <el-alert type="warning" :closable="false" style="margin-bottom:14px;">
        导入会用备份文件<b>覆盖当前全部业务数据</b>（通联、卡片、申请、收卡、地址簿、设置等），此操作不可撤销；登录账号与 SMTP 密码保持当前值不变。建议先导出一份当前数据再操作。
      </el-alert>
      <div style="display:flex;gap:12px;align-items:center;flex-wrap:wrap;">
        <el-upload
          ref="backupUploadRef"
          :auto-upload="false"
          :limit="1"
          accept=".json,application/json"
          :on-change="(f) => (backupFile = f.raw)"
          :on-remove="() => (backupFile = null)"
          :on-exceed="handleBackupExceed"
        >
          <el-button>选择备份文件（.json）</el-button>
        </el-upload>
        <el-button type="danger" :disabled="!backupFile" :loading="importing" @click="handleImport">导入并覆盖当前数据</el-button>
      </div>
    </el-card>

    <el-card>
      <template #header><span>账号安全</span></template>
      <el-form label-width="100px" style="max-width:480px;">
        <el-form-item label="原密码" required><el-input v-model="pwdForm.old_password" type="password" show-password autocomplete="current-password" /></el-form-item>
        <el-form-item label="新密码" required><el-input v-model="pwdForm.new_password" type="password" show-password autocomplete="new-password" placeholder="至少 6 位" /></el-form-item>
        <el-form-item label="确认新密码" required><el-input v-model="pwdForm.confirm" type="password" show-password autocomplete="new-password" /></el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="pwdSaving" @click="handleChangePassword">修改密码</el-button>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<script setup>
import { ref, reactive, watch, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import api from '../api'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useAuthStore } from '../stores/auth'

const router = useRouter()
const auth = useAuthStore()

// ---- 站点设置 ----
const savingSite = ref(false)
const siteForm = reactive({ site_name: '', site_url: '', notify_email: '', site_notice: '' })
async function loadSite() {
  try { const res = await api.get('/settings/site'); Object.assign(siteForm, res) } catch {}
}
async function saveSite() {
  savingSite.value = true
  try { await api.post('/settings/site', siteForm); ElMessage.success('站点设置已保存') } finally { savingSite.value = false }
}

// ---- 数据备份 ----
const exporting = ref(false)
async function handleExport() {
  exporting.value = true
  try {
    const res = await api.get('/admin/export', { responseType: 'blob' })
    const url = URL.createObjectURL(res)
    const a = document.createElement('a')
    a.href = url
    a.download = 'qsl-backup-' + new Date().toISOString().slice(0, 10) + '.json'
    a.click()
    URL.revokeObjectURL(url)
    ElMessage.success('备份已导出')
  } catch (e) { /* 拦截器已提示 */ } finally { exporting.value = false }
}

// ---- 导入恢复 ----
const backupUploadRef = ref(null)
const backupFile = ref(null)
const importing = ref(false)
function handleBackupExceed(files) {
  const f = files[0] || files
  if (f && f.raw !== undefined) backupFile.value = f.raw
  else if (f instanceof File) backupFile.value = f
  if (backupUploadRef.value) {
    backupUploadRef.value.clearFiles()
    if (f) backupUploadRef.value.handleStart(f instanceof File ? f : (f.raw || f))
  }
}
async function handleImport() {
  if (!backupFile.value) { ElMessage.warning('请先选择备份文件'); return }
  try {
    await ElMessageBox.confirm(
      '导入将覆盖当前全部业务数据（通联、卡片、申请、收卡、地址簿、设置等），此操作不可撤销；登录账号与 SMTP 密码保持不变。建议先导出一份当前备份。确定继续？',
      '危险操作：导入并覆盖数据',
      { type: 'warning', confirmButtonText: '仍然导入', cancelButtonText: '取消' }
    )
  } catch (e) { return }
  importing.value = true
  try {
    const fd = new FormData()
    fd.append('file', backupFile.value)
    const res = await api.post('/admin/import', fd, { headers: { 'Content-Type': 'multipart/form-data' } })
    const parts = []
    if (res && res.restored) {
      for (const k of Object.keys(res.restored)) parts.push(k + ' ' + res.restored[k] + ' 条')
    }
    ElMessage.success('导入完成：' + (parts.length ? parts.join('，') : '备份为空表'))
    backupFile.value = null
    if (backupUploadRef.value) backupUploadRef.value.clearFiles()
    loadSite(); loadSmtp()
  } catch (e) { /* 拦截器已提示（含回滚说明） */ } finally { importing.value = false }
}

// ---- SMTP ----
const savingSmtp = ref(false)
const testLoading = ref(false)
const smtpForm = reactive({ smtp_host: '', smtp_port: '465', smtp_user: '', smtp_password: '', smtp_from: '', smtp_from_name: '', smtp_use_tls: false, smtp_use_ssl: true })
const smtpEncryption = ref('ssl')
watch(smtpEncryption, (val) => {
  smtpForm.smtp_use_ssl = val === 'ssl'
  smtpForm.smtp_use_tls = val === 'tls'
  if (val === 'ssl') smtpForm.smtp_port = '465'
  else if (val === 'tls') smtpForm.smtp_port = '587'
  else smtpForm.smtp_port = '25'
})
async function loadSmtp() {
  try { const res = await api.get('/settings/smtp'); Object.assign(smtpForm, res)
    smtpEncryption.value = smtpForm.smtp_use_ssl ? 'ssl' : (smtpForm.smtp_use_tls ? 'tls' : 'none')
  } catch {}
}
async function saveSmtp(silent) {
  savingSmtp.value = true
  try {
    await api.post('/settings/smtp', smtpForm)
    if (!silent) ElMessage.success('SMTP配置已保存')
    return true
  } catch (e) {
    if (!silent) throw e
    return false
  } finally { savingSmtp.value = false }
}
async function testSmtp() {
  const { value } = await ElMessage.prompt('输入测试收件邮箱（默认发给自己，注意垃圾邮件箱）', '测试邮件', {
    inputValue: smtpForm.smtp_user || '', inputPattern: /@/, inputErrorMessage: '请输入有效邮箱',
  })
  // 先保存当前表单，确保测试使用的是页面上看到的配置
  const ok = await saveSmtp(true)
  if (!ok) return
  testLoading.value = true
  try {
    await api.post('/settings/smtp/test?to_email=' + encodeURIComponent(value))
    ElMessage.success('测试邮件已发送至 ' + value + '，请查收（注意垃圾邮件箱）')
  } catch (e) { /* 拦截器已显示具体失败原因 */ } finally { testLoading.value = false }
}

// ---- 修改密码 ----
const pwdSaving = ref(false)
const pwdForm = reactive({ old_password: '', new_password: '', confirm: '' })
async function handleChangePassword() {
  if (!pwdForm.old_password) { ElMessage.warning('请输入原密码'); return }
  if (pwdForm.new_password.length < 6) { ElMessage.warning('新密码至少 6 位'); return }
  if (pwdForm.new_password !== pwdForm.confirm) { ElMessage.warning('两次输入的新密码不一致'); return }
  pwdSaving.value = true
  try {
    await api.post('/auth/change-password', { old_password: pwdForm.old_password, new_password: pwdForm.new_password })
    ElMessage.success('密码已修改，请重新登录')
    auth.logout()
    router.push('/login')
  } catch (e) { /* 拦截器已提示 */ } finally { pwdSaving.value = false }
}

onMounted(() => { loadSite(); loadSmtp() })
</script>
