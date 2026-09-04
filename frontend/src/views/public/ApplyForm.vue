<template>
  <div class="public-page">
    <div class="public-container">
      <!-- Header -->
      <header class="public-header">
        <div class="logo-section">
          <div class="logo-icon">QSL</div>
          <div>
            <h1>QSL 卡片换卡申请</h1>
            <p class="subtitle">填写信息，申请与对方台站交换 QSL 卡片</p>
          </div>
        </div>
      </header>

      <!-- Main Content -->
      <main class="public-main">
        <div class="card modern-card" v-if="!submitted">
          <div class="card-header">
            <h2>申请信息</h2>
            <p>请按步骤填写换卡申请</p>
          </div>

          <el-form :model="form" :rules="rules" ref="formRef" label-position="top" class="modern-form">

            <!-- 第一步：选择场景 -->
            <el-divider content-position="left">第一步 · 选择换卡场景</el-divider>
            <el-form-item prop="scene_type" class="form-item-full">
              <el-radio-group v-model="form.scene_type" class="scene-radio-group">
                <el-radio-button value="QSO">
                  <el-icon><Connection /></el-icon>
                  <span class="scene-label">QSO 通联换卡</span>
                </el-radio-button>
                <el-radio-button value="SWL">
                  <el-icon><Headset /></el-icon>
                  <span class="scene-label">SWL 收听换卡</span>
                </el-radio-button>
                <el-radio-button value="EYEBALL">
                  <el-icon><Location /></el-icon>
                  <span class="scene-label">EYEBALL 见面换卡</span>
                </el-radio-button>
              </el-radio-group>
            </el-form-item>

            <!-- 第二步：场景证据（动态） -->
            <el-divider content-position="left">第二步 · 填写场景证据</el-divider>

            <!-- QSO 场景 -->
            <div v-if="form.scene_type === 'QSO'" class="form-grid">
              <el-form-item label="通联日期 (UTC+8)" prop="qso_date">
                <el-date-picker v-model="form.qso_date" type="date" value-format="YYYY-MM-DD" placeholder="选择日期" size="large" style="width:100%" :disabled-date="disableFuture" />
              </el-form-item>
              <el-form-item label="通联时间 (UTC+8)">
                <el-time-select v-model="form.qso_time" start="00:00" step="00:05" end="23:55" placeholder="精确到分钟，如 19:42" size="large" style="width:100%" />
              </el-form-item>
              <el-form-item label="频率" prop="qso_freq">
                <el-input v-model="form.qso_freq" placeholder="14.270" size="large" />
              </el-form-item>
              <el-form-item label="频段">
                <el-input v-model="form.qso_band" placeholder="20m" size="large" />
              </el-form-item>
              <el-form-item label="模式" prop="qso_mode">
                <el-input v-model="form.qso_mode" placeholder="SSB" size="large" />
              </el-form-item>

            </div>

            <!-- EYEBALL 场景 -->
            <div v-if="form.scene_type === 'EYEBALL'" class="form-grid">
              <el-form-item label="见面类型" prop="eyeball_type" class="form-item-full">
                <el-radio-group v-model="form.eyeball_type">
                  <el-radio-button value="OFFLINE">线下补换</el-radio-button>
                  <el-radio-button value="ONLINE">网络EYE</el-radio-button>
                </el-radio-group>
              </el-form-item>
              <el-form-item label="见面日期 (UTC+8)" prop="eyeball_date">
                <el-date-picker v-model="form.eyeball_date" type="date" value-format="YYYY-MM-DD" placeholder="选择日期" size="large" style="width:100%" :disabled-date="disableFuture" />
              </el-form-item>
              <el-form-item label="见面时间 (UTC+8)">
                <el-time-select v-model="form.eyeball_time" start="00:00" step="00:05" end="23:55" placeholder="精确到分钟，如 14:05" size="large" style="width:100%" />
              </el-form-item>
              <el-form-item label="活动名称" prop="eyeball_activity" class="form-item-full">
                <el-input v-model="form.eyeball_activity" placeholder="例如: 2024业余无线电节" size="large" />
              </el-form-item>
              <el-form-item label="地点" class="form-item-full">
                <el-input v-model="form.eyeball_location" placeholder="例如: 北京展览馆" size="large" />
              </el-form-item>
            </div>

            <!-- SWL 场景 -->
            <div v-if="form.scene_type === 'SWL'" class="form-grid">
              <el-form-item label="收听日期 (UTC+8)" prop="swl_date">
                <el-date-picker v-model="form.swl_date" type="date" value-format="YYYY-MM-DD" placeholder="选择日期" size="large" style="width:100%" :disabled-date="disableFuture" />
              </el-form-item>
              <el-form-item label="收听时间 (UTC+8)" prop="swl_time">
                <el-time-picker v-model="form.swl_time" format="HH:mm" value-format="HH:mm" placeholder="精确到分钟，如 19:42" size="large" style="width:100%" />
              </el-form-item>
              <el-form-item label="收听频率" prop="swl_freq">
                <el-input v-model="form.swl_freq" placeholder="14.270" size="large" />
              </el-form-item>
              <el-form-item label="频段">
                <el-input v-model="form.swl_band" placeholder="20m" size="large" />
              </el-form-item>
              <el-form-item label="收听模式" prop="swl_mode">
                <el-input v-model="form.swl_mode" placeholder="SSB" size="large" />
              </el-form-item>
            </div>

            <!-- 第三步：换卡理由 -->
            <el-divider content-position="left">第三步 · 换卡理由（选填）</el-divider>
            <el-form-item class="form-item-full">
              <el-input
                v-model="form.application_reason"
                type="textarea"
                :rows="3"
                placeholder="可选，填写您申请换卡的理由"
              />
            </el-form-item>

            <!-- 期望卡片版本（SWL 由对方先寄卡，无需选择版本） -->
            <template v-if="form.scene_type !== 'SWL'">
            <el-divider content-position="left">{{ "第" + stepNo.version + "步 · 期望卡片版本" }}</el-divider>
            <el-form-item prop="card_version" class="form-item-full">
              <div class="version-block">
                <p class="version-label">期望卡片版本（可多选）</p>
                <div class="version-grid">
                  <div
                    v-for="v in versions"
                    :key="v.card_version"
                    class="version-card"
                    :class="{ selected: form.card_version.includes(v.card_version) }"
                    @click="toggleVersion(v.card_version)"
                  >
                    <div class="version-thumb">
                      <img v-if="v.image_path && !brokenThumb[v.card_version]" :src="v.image_path" @error="brokenThumb[v.card_version] = true" />
                      <span v-else class="version-placeholder">QSL</span>
                    </div>
                    <div class="version-name">{{ v.card_version }}</div>
                    <el-icon v-if="form.card_version.includes(v.card_version)" class="version-check"><CircleCheck /></el-icon>
                  </div>
                </div>
              </div>
            </el-form-item>
            </template>

            <!-- 收卡方式 -->
            <el-divider content-position="left">{{ "第" + stepNo.mail + "步 · 收卡方式" }}</el-divider>
            <p class="field-tip" style="margin-bottom:12px;">
              <template v-if="form.scene_type === 'SWL'">此选项决定我们的<b>回寄卡片</b>如何寄给您（待我们收到并确认您的收听卡后制卡回寄）；选择「通过卡片局」时，请在下方填写该卡片局的收件地址。</template>
              <template v-else>此选项决定我们的卡片如何寄给您；选择「通过卡片局」时，请在下方填写该卡片局的收件地址。</template>
            </p>
            <el-form-item class="form-item-full">
              <el-radio-group v-model="form.use_bureau" class="radio-group-modern">
                <el-radio-button :value="false">
                  <el-icon><Postcard /></el-icon>
                  直接邮寄
                </el-radio-button>
                <el-radio-button :value="true">
                  <el-icon><OfficeBuilding /></el-icon>
                  通过卡片局
                </el-radio-button>
              </el-radio-group>
            </el-form-item>
            <el-form-item v-if="form.use_bureau" label="您的卡片局名称" prop="bureau_name" class="form-item-full">
              <el-input v-model="form.bureau_name" placeholder="例如: JARL QSL Bureau" size="large" />
              <p class="field-tip">由您提供卡片局信息，我们会把卡片寄往该局；收件信息中请填写该局的地址和邮编</p>
            </el-form-item>

            <!-- 收件信息 -->
            <el-divider content-position="left">{{ "第" + stepNo.info + "步 · 收件信息" }}</el-divider>
            <el-alert v-if="form.scene_type === 'SWL'" type="info" :closable="false" style="margin-bottom:16px;">
              <p style="margin:0 0 4px;">SWL 换卡流程说明：</p>
              <p style="margin:0 0 2px;">1. 审核通过后，我们会通过邮件发送<b>我们的收卡地址</b>给您；</p>
              <p style="margin:0 0 2px;">2. 您将收听卡寄出，并在申请进度页登记邮寄方式与单号；</p>
              <p style="margin:0 0 2px;">3. 我们确认收卡（自动生成收卡记录）并邮件通知您；</p>
              <p style="margin:0 0 2px;">4. 我们按需制卡回寄到您下方填写的地址，进度可在状态链接随时查询。</p>
              <p style="margin:2px 0 0;">下方地址<b>仅用于接收我们的回寄卡片</b>，请如实填写。</p>
            </el-alert>
            <div class="form-grid">
              <el-form-item label="您的呼号" prop="call_sign">
                <el-input
                  v-model="form.call_sign"
                  placeholder="例如: BV2AAA"
                  size="large"
                  prefix-icon="User"
                  @blur="form.call_sign = form.call_sign.toUpperCase()"
                />
              </el-form-item>

              <el-form-item label="收件人姓名" prop="name">
                <el-input v-model="form.name" placeholder="您的姓名" size="large" />
              </el-form-item>

              <el-form-item label="电子邮箱" prop="email">
                <el-input v-model="form.email" placeholder="用于接收通知" size="large" />
              </el-form-item>

              <el-form-item label="联系电话">
                <el-input v-model="form.telephone" placeholder="可选" size="large" />
              </el-form-item>

              <el-form-item label="邮政编码" prop="postal_code">
                <el-input v-model="form.postal_code" placeholder="邮编" size="large" />
              </el-form-item>

              <el-form-item label="详细地址" prop="address" class="form-item-full">
                <el-input
                  v-model="form.address"
                  type="textarea"
                  :rows="3"
                  :placeholder="form.use_bureau ? '您卡片局的收件地址' : '您的收件地址'"
                  size="large"
                />
              </el-form-item>
            </div>

            <!-- 第七步：提交 -->
            <div class="form-actions">
              <el-button
                type="primary"
                size="large"
                :loading="submitting"
                @click="handleSubmit"
                class="submit-btn"
              >
                <el-icon><Check /></el-icon>
                提交申请
              </el-button>
            </div>
          </el-form>
        </div>

        <!-- Success Message -->
        <div v-if="submitted" class="card modern-card success-card">
          <div class="success-content">
            <div class="success-icon">
              <el-icon :size="64" color="#10b981"><CircleCheck /></el-icon>
            </div>
            <h2>申请已提交</h2>
            <p>您的换卡申请已成功提交，编号: <strong>{{ result.request_code }}</strong></p>
            <p class="success-note">我们会尽快审核您的申请，审核结果将通过邮件通知您。</p>
            <div class="share-section">
              <p>申请编号：<strong>{{ result.request_code }}</strong></p>
              <p class="share-tip">复制链接可实时查看申请进度（审核中 / 制卡中 / 邮寄中 / 已签收）：</p>
              <div class="share-link">
                <el-input v-model="statusLink" readonly size="large">
                  <template #append>
                    <el-button @click="copyLink">复制</el-button>
                  </template>
                </el-input>
              </div>
            </div>
            <div class="mail-section" v-if="mailInfo && mailInfo.address">
              <p class="mail-title">请将您的卡片寄至以下地址（呼号: {{ mailInfo.call_sign }}）：</p>
              <p>{{ mailInfo.name }} 收</p>
              <p>{{ mailInfo.postal_code }} {{ mailInfo.address }}</p>
              <p class="mail-note">建议随卡附上本页截图或注明您的呼号，便于我们登记回卡；若您选择了卡片局渠道，也可经卡片局寄出。</p>
            </div>
          </div>
        </div>
      </main>

      <!-- Footer -->
      <footer class="public-footer">
        <p>QSL 卡片管理系统 · 业余无线电</p>
      </footer>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import api from '../../api'
import { ElMessage } from 'element-plus'

const route = useRoute()
const formRef = ref(null)
const submitting = ref(false)
const submitted = ref(false)
const versions = ref([])
const result = ref(null)
const statusLink = ref('')
const mailInfo = ref(null)
// 场景日期不允许选择未来
const disableFuture = (d) => d.getTime() > Date.now()

const form = reactive({
  scene_type: 'QSO',
  call_sign: '',
  card_version: [],
  use_bureau: false,
  bureau_name: '',
  name: '',
  email: '',
  telephone: '',
  postal_code: '',
  address: '',
  remarks: '',
  application_reason: '',
  // QSO
  qso_date: '',
  qso_time: '',
  qso_freq: '',
  qso_band: '',
  qso_mode: '',
  // EYEBALL
  eyeball_date: '',
  eyeball_time: '',
  eyeball_activity: '',
  eyeball_location: '',
  eyeball_type: 'OFFLINE',
  // SWL
  swl_date: '',
  swl_time: '',
  swl_freq: '',
  swl_band: '',
  swl_mode: ''
})

const brokenThumb = ref({})

function toggleVersion(v) {
  const i = form.card_version.indexOf(v)
  if (i >= 0) {
    form.card_version.splice(i, 1)
  } else {
    form.card_version.push(v)
  }
}

// 步骤编号：SWL 无"期望卡片版本"步骤，后续步骤顺延
const stepNo = computed(() => {
  const skip = form.scene_type === 'SWL'
  return { version: skip ? 0 : 4, mail: skip ? 4 : 5, info: skip ? 5 : 6 }
})

// 动态校验规则：根据场景类型动态生成必填项
const rules = computed(() => {
  const r = {
    scene_type: [{ required: true, message: '请选择换卡场景', trigger: 'change' }],
    call_sign: [{ required: true, message: '请输入呼号', trigger: 'blur' }],
    name: [{ required: true, message: '请输入姓名', trigger: 'blur' }],
    email: [
      { required: true, message: '请输入邮箱', trigger: 'blur' },
      { type: 'email', message: '邮箱格式不正确', trigger: 'blur' }
    ],
    postal_code: [{ required: true, message: '请输入邮编', trigger: 'blur' }],
    address: [{ required: true, message: '请输入地址', trigger: 'blur' }],
    card_version: [{ type: 'array', required: form.scene_type !== 'SWL', message: '请至少选择一个卡片版本', trigger: 'change' }]
  }
  if (form.use_bureau) {
    r.bureau_name = [{ required: true, message: '请填写您的卡片局名称', trigger: 'blur' }]
  }
  // 场景证据动态必填
  if (form.scene_type === 'QSO') {
    r.qso_date = [{ required: true, message: '请输入通联日期', trigger: 'blur' }]
    r.qso_freq = [{ required: true, message: '请输入频率', trigger: 'blur' }]
    r.qso_mode = [{ required: true, message: '请输入模式', trigger: 'blur' }]
  } else if (form.scene_type === 'EYEBALL') {
    r.eyeball_date = [{ required: true, message: '请输入见面日期', trigger: 'blur' }]
    r.eyeball_activity = [{ required: true, message: '请输入活动名称', trigger: 'blur' }]
  } else if (form.scene_type === 'SWL') {
    r.swl_date = [{ required: true, message: '请输入收听日期', trigger: 'blur' }]
  r.swl_time = [{ required: true, message: '请选择收听时间', trigger: 'change' }]
    r.swl_freq = [{ required: true, message: '请输入收听频率', trigger: 'blur' }]
    r.swl_mode = [{ required: true, message: '请输入收听模式', trigger: 'blur' }]
  }
  return r
})

async function loadData() {
  try {
    const [vRes, mRes] = await Promise.all([
      api.get('/public/station-cards'),
      api.get('/public/station-mail-info')
    ])
    versions.value = (vRes || []).filter(v => !v.qso_only)
    mailInfo.value = mRes || {}
  } catch (e) {
    console.error('加载失败:', e)
  }
}

async function handleSubmit() {
  if (!formRef.value) return
  try {
    await formRef.value.validate()
  } catch {
    ElMessage.warning('请完善必填项')
    return
  }

  submitting.value = true
  try {
    const payload = {
      scene_type: form.scene_type,
      call_sign: form.call_sign.toUpperCase(),
      card_version: form.card_version.join(', '),
      use_bureau: form.use_bureau,
      bureau_name: form.bureau_name,
      email: form.email,
      name: form.name,
      telephone: form.telephone,
      postal_code: form.postal_code,
      address: form.address,
      remarks: form.remarks,
      application_reason: form.application_reason,
      qso_date: form.qso_date,
      qso_time: form.qso_time,
      qso_freq: form.qso_freq,
      qso_band: form.qso_band,
      qso_mode: form.qso_mode,
      eyeball_date: form.eyeball_date,
      eyeball_time: form.eyeball_time,
      eyeball_activity: form.eyeball_activity,
      eyeball_location: form.eyeball_location,
      eyeball_type: form.eyeball_type,
      swl_date: form.swl_date,
      swl_time: form.swl_time,
      swl_freq: form.swl_freq,
      swl_band: form.swl_band,
      swl_mode: form.swl_mode
    }
    const res = await api.post('/public/exchange-online', payload)
    result.value = res
    statusLink.value = window.location.origin + '/status/' + res.request_code
    submitted.value = true
    ElMessage.success('申请提交成功')
  } catch (e) {
    ElMessage.error(e.response?.data?.message || '提交失败')
  } finally {
    submitting.value = false
  }
}

function copyLink() {
  navigator.clipboard.writeText(statusLink.value)
  ElMessage.success('链接已复制')
}

onMounted(() => {
  loadData()
  if (route.query.call_sign) form.call_sign = route.query.call_sign.toUpperCase()
})
</script>

<style scoped>
.public-page {
  min-height: 100vh;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  padding: 20px;
}

.public-container {
  max-width: 800px;
  margin: 0 auto;
}

.public-header {
  text-align: center;
  padding: 40px 0 30px;
  color: white;
}

.logo-section {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 16px;
}

.logo-icon {
  width: 60px;
  height: 60px;
  background: rgba(255,255,255,0.2);
  border-radius: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 24px;
  font-weight: bold;
  backdrop-filter: blur(10px);
}

.public-header h1 {
  margin: 0;
  font-size: 28px;
  font-weight: 600;
}

.subtitle {
  margin: 8px 0 0;
  opacity: 0.9;
  font-size: 16px;
}

.modern-card {
  background: white;
  border-radius: 20px;
  padding: 32px;
  box-shadow: 0 20px 60px rgba(0,0,0,0.15);
  margin-bottom: 24px;
}

.card-header {
  margin-bottom: 24px;
  padding-bottom: 16px;
  border-bottom: 1px solid #f0f0f0;
}

.card-header h2 {
  margin: 0 0 8px;
  font-size: 22px;
  color: #1a1a1a;
}

.card-header p {
  margin: 0;
  color: #666;
  font-size: 14px;
}

.modern-form :deep(.el-form-item__label) {
  font-weight: 500;
  color: #374151;
}

.form-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 20px;
}

.form-item-full {
  grid-column: 1 / -1;
}

.scene-radio-group {
  display: flex;
  width: 100%;
}

.scene-radio-group :deep(.el-radio-button) {
  flex: 1;
}

.scene-radio-group :deep(.el-radio-button__inner) {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  height: 52px;
}

.scene-label {
  font-size: 15px;
  font-weight: 500;
}

.radio-group-modern {
  width: 100%;
}

.radio-group-modern :deep(.el-radio-button) {
  flex: 1;
}

.radio-group-modern :deep(.el-radio-button__inner) {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
}

.form-actions {
  margin-top: 32px;
  text-align: center;
}

.submit-btn {
  width: 200px;
  height: 50px;
  font-size: 16px;
  border-radius: 12px;
}

.success-card {
  text-align: center;
}

.success-content {
  padding: 20px;
}

.success-icon {
  margin-bottom: 20px;
}

.success-content h2 {
  color: #10b981;
  margin: 0 0 12px;
}

.success-content p {
  color: #666;
  margin: 8px 0;
}

.success-note {
  font-size: 14px;
  color: #999;
}

.share-section {
  margin-top: 30px;
  padding-top: 20px;
  border-top: 1px solid #f0f0f0;
}

.share-link {
  max-width: 500px;
  margin: 12px auto 0;
}

.public-footer {
  text-align: center;
  padding: 30px 0;
  color: rgba(255,255,255,0.7);
  font-size: 14px;
}

@media (max-width: 640px) {
  .form-grid {
    grid-template-columns: 1fr;
  }

  .public-header h1 {
    font-size: 22px;
  }

  .modern-card {
    padding: 20px;
  }

  .scene-radio-group {
    flex-direction: column;
  }
}

.mail-section {
  margin-top: 24px;
  padding: 16px 20px;
  background: #f8f7f2;
  border: 1px dashed #c9c3b4;
  text-align: left;
}
.mail-title { font-weight: 600; margin-bottom: 8px; color: var(--qsl-navy, #1a1a1a); }
.mail-note { font-size: 12px; color: #999; margin-top: 8px; }

.version-block { width: 100%; }
.version-label { margin: 0 0 12px; font-weight: 500; color: #374151; }
.version-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 16px; }
.version-card { position: relative; border: 2px solid #e5e1d8; border-radius: 8px; padding: 12px; text-align: center; cursor: pointer; transition: all .15s; background: #fff; }
.version-card:hover { border-color: #b9b29f; }
.version-card.selected { border-color: #10b981; background: #f0fdf7; }
.version-thumb { height: 90px; display: flex; align-items: center; justify-content: center; overflow: hidden; margin-bottom: 8px; background: #faf9f5; border-radius: 4px; }
.version-thumb img { max-height: 90px; max-width: 100%; object-fit: contain; }
.version-placeholder { font-weight: 700; color: #c9c3b4; letter-spacing: 2px; }
.version-name { font-size: 13px; color: #374151; word-break: break-all; }
.version-check { position: absolute; top: 6px; right: 6px; color: #10b981; }
.share-tip { font-size: 13px; color: #666; }

@media (max-width: 640px) {
  .version-grid { grid-template-columns: repeat(2, 1fr); }
}

.field-tip { margin: 6px 0 0; font-size: 12px; color: #999; }

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
.card-header p { color:var(--qsl-muted); }
.public-footer { color:#9c978d; border-top:1px solid var(--qsl-line); }
</style>
