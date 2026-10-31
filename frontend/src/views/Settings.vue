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
      <template #header>
        <div style="display:flex;justify-content:space-between;align-items:center;">
          <span>账号安全</span>
          <el-tag v-if="security.username" type="info">{{ security.username }}</el-tag>
        </div>
      </template>

      <el-alert v-if="security.mfa_enabled" type="success" :closable="false" style="margin-bottom:18px;">
        已绑定二步验证：下次登录在口令之后，还要过一次动态码或通行密钥。
      </el-alert>
      <el-alert v-else type="warning" :closable="false" style="margin-bottom:18px;">
        还没绑定二步验证。口令一旦泄露就没有第二道门了，建议现在就绑一个。
      </el-alert>

      <el-divider content-position="left">修改口令</el-divider>
      <el-form label-width="100px" style="max-width:480px;">
        <el-form-item label="原密码" required><el-input v-model="pwdForm.old_password" type="password" show-password autocomplete="current-password" /></el-form-item>
        <el-form-item label="新密码" required><el-input v-model="pwdForm.new_password" type="password" show-password autocomplete="new-password" placeholder="至少 8 位" /></el-form-item>
        <el-form-item label="确认新密码" required><el-input v-model="pwdForm.confirm" type="password" show-password autocomplete="new-password" /></el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="pwdSaving" @click="handleChangePassword">修改密码</el-button>
        </el-form-item>
      </el-form>

      <el-divider content-position="left">用户名</el-divider>
      <el-form label-width="100px" style="max-width:480px;">
        <el-form-item label="新用户名">
          <el-input v-model="renameForm.username" placeholder="3-32 位字母、数字、下划线、点或连字符" />
        </el-form-item>
        <el-form-item label="当前口令">
          <el-input v-model="renameForm.password" type="password" show-password autocomplete="current-password" placeholder="确认身份" />
        </el-form-item>
        <el-form-item>
          <el-button :loading="renaming" @click="handleRename">修改用户名</el-button>
          <span class="hint-text">改名后其他设备会被强制下线，本机保持登录。</span>
        </el-form-item>
      </el-form>

      <el-divider content-position="left">二步验证（身份验证器）</el-divider>
      <div v-if="!security.totp_enabled" class="security-block">
        <el-button v-if="!totp.otpauth" :loading="totpLoading" @click="startTotp">生成密钥</el-button>
        <template v-else>
          <div class="totp-setup">
            <img v-if="totp.qr" :src="totp.qr" alt="扫码绑定" class="totp-qr" />
            <div class="totp-tips">
              <p>用身份验证器（Google Authenticator / 1Password / 微软 Authenticator）扫描二维码，或手动输入：</p>
              <code class="totp-secret">{{ totp.secret }}</code>
              <p class="hint-text">扫完把验证器里的 6 位数字填到下面，确认后才算启用。</p>
              <div style="display:flex;gap:8px;margin-top:10px;">
                <el-input v-model="totp.code" placeholder="6 位数字" maxlength="6" style="max-width:160px;" />
                <el-button type="primary" :loading="totpLoading" @click="enableTotp">确认启用</el-button>
                <el-button text @click="cancelTotp">取消</el-button>
              </div>
            </div>
          </div>
        </template>
      </div>
      <div v-else class="security-block">
        <p class="hint-text">剩余恢复码：{{ security.recovery_left }} 个（用一次少一个，用完记得重新生成）。</p>
        <div style="display:flex;gap:8px;flex-wrap:wrap;margin-top:10px;">
          <el-input v-model="totp.code" placeholder="当前 6 位动态码" maxlength="6" style="max-width:160px;" />
          <el-input v-model="totp.password" type="password" show-password placeholder="当前口令" style="max-width:200px;" />
          <el-button type="danger" plain :loading="totpLoading" @click="disableTotp">关闭二步验证</el-button>
        </div>
      </div>

      <el-divider content-position="left">通行密钥</el-divider>
      <el-alert v-if="!passkeyAvailable" type="info" :closable="false" style="margin-bottom:12px;">
        当前环境不支持通行密钥：需要 HTTPS，且服务器要配置 WEBAUTHN_RP_ID / WEBAUTHN_ORIGINS。
      </el-alert>
      <template v-else>
        <el-table v-if="(security.passkeys || []).length" :data="security.passkeys" size="small" style="max-width:640px;">
          <el-table-column prop="name" label="名称" />
          <el-table-column label="上次使用" width="180">
            <template #default="{ row }">{{ formatTime(row.last_used_at) }}</template>
          </el-table-column>
          <el-table-column label="操作" width="100">
            <template #default="{ row }">
              <el-button link type="danger" @click="removePasskey(row)">解绑</el-button>
            </template>
          </el-table-column>
        </el-table>
        <p v-else class="hint-text">还没有绑定通行密钥。绑定后可以用指纹 / 人脸 / 安全密钥直接登录，不用输口令。</p>
        <el-button style="margin-top:12px;" :loading="passkeyLoading" @click="addPasskey">绑定新通行密钥</el-button>
      </template>

      <el-divider content-position="left">其他设备</el-divider>
      <p class="hint-text">修改口令、改用户名、增删通行密钥都会自动让其他设备下线。这里可以一次性全部踢掉。</p>
      <el-button style="margin-top:10px;" @click="revokeSessions">强制下线所有设备</el-button>
    </el-card>

    <!-- 恢复码只显示这一次，关掉就再也看不到 -->
    <el-dialog v-model="recoveryVisible" title="恢复码（只显示这一次）" width="420px">
      <el-alert type="warning" :closable="false" style="margin-bottom:14px;">
        手机丢了、验证器没了时，用其中任意一个就能登录。每个只能用一次。请现在抄下来。
      </el-alert>
      <div class="recovery-grid">
        <code v-for="c in recoveryCodes" :key="c">{{ c }}</code>
      </div>
      <template #footer>
        <el-button @click="copyRecovery">复制全部</el-button>
        <el-button type="primary" @click="recoveryVisible = false">我已保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, watch, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import api from '../api'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useAuthStore } from '../stores/auth'
import QRCode from 'qrcode'
import {
  passkeySupported,
  publicKeyToCreationOptions,
  credentialToJSON,
} from '../utils/webauthn'

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
  if (pwdForm.new_password.length < 8) { ElMessage.warning('新密码至少 8 位'); return }
  if (pwdForm.new_password !== pwdForm.confirm) { ElMessage.warning('两次输入的新密码不一致'); return }
  pwdSaving.value = true
  try {
    await api.post('/auth/change-password', { old_password: pwdForm.old_password, new_password: pwdForm.new_password })
    ElMessage.success('密码已修改，请重新登录')
    auth.logout()
    router.push('/login')
  } catch (e) { /* 拦截器已提示 */ } finally { pwdSaving.value = false }
}

// ---- 账号安全总览 ----
const security = reactive({
  username: '',
  is_admin: false,
  totp_enabled: false,
  recovery_left: 0,
  passkeys: [],
  has_password: true,
})
const serverPasskey = ref(false)
// 服务端启用 + 浏览器在安全上下文里，两个条件都满足才显示通行密钥区块
const passkeyAvailable = computed(() => serverPasskey.value && passkeySupported())

async function loadSecurity() {
  try {
    Object.assign(security, await api.get('/auth/security'))
  } catch { /* 未登录时静默 */ }
  try {
    const cap = await api.get('/auth/capabilities')
    serverPasskey.value = !!cap?.passkey
  } catch { serverPasskey.value = false }
}

function formatTime(v) {
  if (!v) return '从未'
  const d = new Date(v)
  if (Number.isNaN(d.getTime())) return '从未'
  return d.toLocaleString('zh-CN')
}

// ---- 改用户名 ----
const renaming = ref(false)
const renameForm = reactive({ username: '', password: '' })
async function handleRename() {
  const name = (renameForm.username || '').trim()
  if (!name) { ElMessage.warning('请输入新用户名'); return }
  if (security.has_password && !renameForm.password) { ElMessage.warning('请输入当前口令确认身份'); return }
  renaming.value = true
  try {
    const res = await api.post('/auth/username', { username: name, password: renameForm.password })
    auth.adoptTokens(res) // 后端会作废旧令牌并换发新的，本机保持登录
    renameForm.password = ''
    ElMessage.success('用户名已修改')
    await loadSecurity()
  } catch (e) { /* 拦截器已提示（含占用 409） */ } finally { renaming.value = false }
}

// ---- 二步验证（TOTP） ----
const totpLoading = ref(false)
const totp = reactive({ secret: '', otpauth: '', qr: '', code: '', password: '' })

async function startTotp() {
  totpLoading.value = true
  try {
    const res = await api.post('/auth/totp/setup', {})
    totp.secret = res.secret
    totp.otpauth = res.otpauth_url
    // 二维码在本地渲染，secret 不经过任何第三方
    totp.qr = await QRCode.toDataURL(res.otpauth_url, { margin: 1, width: 200 })
  } catch (e) { /* 拦截器已提示 */ } finally { totpLoading.value = false }
}

function cancelTotp() {
  totp.secret = ''
  totp.otpauth = ''
  totp.qr = ''
  totp.code = ''
}

async function enableTotp() {
  if (!/^\d{6}$/.test((totp.code || '').trim())) { ElMessage.warning('请输入 6 位数字'); return }
  totpLoading.value = true
  try {
    const res = await api.post('/auth/totp/enable', { code: totp.code.trim() })
    recoveryCodes.value = res.recovery_codes || []
    recoveryVisible.value = true
    cancelTotp()
    ElMessage.success('二步验证已启用')
    await loadSecurity()
  } catch (e) { /* 拦截器已提示 */ } finally { totpLoading.value = false }
}

async function disableTotp() {
  if (!totp.password) { ElMessage.warning('请输入当前口令'); return }
  if (!/^\d{6}$/.test((totp.code || '').trim())) { ElMessage.warning('请输入当前 6 位动态码'); return }
  try {
    await ElMessageBox.confirm('关闭后只剩口令这一道门，确认关闭？', '关闭二步验证', { type: 'warning' })
  } catch { return }
  totpLoading.value = true
  try {
    const res = await api.post('/auth/totp/disable', { password: totp.password, code: totp.code.trim() })
    auth.adoptTokens(res)
    totp.password = ''
    totp.code = ''
    ElMessage.success('已关闭二步验证')
    await loadSecurity()
  } catch (e) { /* 拦截器已提示 */ } finally { totpLoading.value = false }
}

const recoveryVisible = ref(false)
const recoveryCodes = ref([])
async function copyRecovery() {
  try {
    await navigator.clipboard.writeText(recoveryCodes.value.join('\n'))
    ElMessage.success('已复制')
  } catch {
    ElMessage.warning('浏览器不允许自动复制，请手动抄写')
  }
}

// ---- 通行密钥 ----
const passkeyLoading = ref(false)
async function addPasskey() {
  if (!passkeyAvailable.value) return
  let name = '通行密钥'
  try {
    const r = await ElMessageBox.prompt('给这把钥匙起个名字，方便以后分辨', '绑定通行密钥', {
      inputValue: '通行密钥',
      inputPattern: /\S+/,
      inputErrorMessage: '请填写名称',
    })
    name = r.value
  } catch { return }

  passkeyLoading.value = true
  try {
    const begin = await api.post('/auth/passkey/register/begin', { name })
    const cred = await navigator.credentials.create({
      publicKey: publicKeyToCreationOptions(begin.publicKey),
    })
    await api.post(
      '/auth/passkey/register/finish?ticket=' + encodeURIComponent(begin.ticket) + '&name=' + encodeURIComponent(name),
      credentialToJSON(cred)
    )
    ElMessage.success('通行密钥已绑定')
    await loadSecurity()
  } catch (e) {
    if (e?.name === 'NotAllowedError') ElMessage.warning('已取消')
    else if (!e?.response) ElMessage.error('绑定失败，请确认浏览器支持通行密钥')
  } finally { passkeyLoading.value = false }
}

async function removePasskey(row) {
  let password = ''
  if (security.has_password) {
    try {
      const r = await ElMessageBox.prompt(`解绑「${row.name}」后它就不能再登录了，输入当前口令确认：`, '解绑通行密钥', {
        inputType: 'password',
        inputErrorMessage: '请输入口令',
      })
      password = r.value
    } catch { return }
  } else {
    try {
      await ElMessageBox.confirm(`解绑「${row.name}」后它就不能再登录了，确认？`, '解绑通行密钥', { type: 'warning' })
    } catch { return }
  }

  passkeyLoading.value = true
  try {
    const res = await api.delete('/auth/passkey/' + row.id, { data: { password } })
    auth.adoptTokens(res)
    ElMessage.success('已解绑')
    await loadSecurity()
  } catch (e) { /* 拦截器已提示 */ } finally { passkeyLoading.value = false }
}

// ---- 强制下线 ----
async function revokeSessions() {
  let password = ''
  try {
    const r = await ElMessageBox.prompt('其他设备上的登录会立即失效，本机保持登录。输入当前口令确认：', '强制下线', {
      inputType: 'password',
      inputErrorMessage: '请输入口令',
    })
    password = r.value
  } catch { return }
  try {
    const res = await api.post('/auth/sessions/revoke', { password })
    auth.adoptTokens(res)
    ElMessage.success('已让其他设备下线')
  } catch (e) { /* 拦截器已提示 */ }
}

onMounted(() => { loadSite(); loadSmtp(); loadSecurity() })
</script>

<style scoped>
.hint-text {
  color: var(--qsl-muted);
  font-size: 12.5px;
  margin-left: 8px;
}
.security-block {
  max-width: 720px;
}
.totp-setup {
  display: flex;
  gap: 18px;
  align-items: flex-start;
  flex-wrap: wrap;
}
.totp-qr {
  width: 180px;
  height: 180px;
  border: 1px solid var(--qsl-line);
  border-radius: var(--qsl-radius);
  padding: 6px;
  background: var(--qsl-qr-bg);
}
.totp-tips {
  flex: 1;
  min-width: 260px;
  font-size: 13px;
  line-height: 1.7;
}
.totp-tips p {
  margin: 0 0 8px;
}
.totp-secret {
  display: inline-block;
  padding: 4px 8px;
  background: var(--qsl-panel-2);
  border-radius: var(--qsl-radius-sm);
  font-size: 13px;
  letter-spacing: 0.05em;
  word-break: break-all;
}
.recovery-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 8px;
}
.recovery-grid code {
  padding: 6px 10px;
  background: var(--qsl-panel-2);
  border-radius: var(--qsl-radius-sm);
  font-size: 14px;
  letter-spacing: 0.08em;
}
</style>
