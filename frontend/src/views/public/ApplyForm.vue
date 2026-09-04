<template>
  <div class="public-page">
    <div class="public-container">
      <!-- Header -->
      <header class="public-header">
        <div class="logo-section">
          <div class="logo-icon">QSL</div>
          <div>
            <h1>{{ t('apply.title') }}</h1>
            <p class="subtitle">{{ t('apply.subtitle') }}</p>
          </div>
        </div>
      </header>

      <!-- Main Content -->
      <main class="public-main">
        <div class="card modern-card" v-if="!submitted">
          <div class="card-header">
            <h2>{{ t('apply.formTitle') }}</h2>
            <p>{{ t('apply.formSub') }}</p>
          </div>

          <el-form :model="form" :rules="rules" ref="formRef" label-position="top" class="modern-form">

            <!-- 第一步：选择场景 -->
            <el-divider content-position="left">{{ t('apply.stepScene') }}</el-divider>
            <el-form-item prop="scene_type" class="form-item-full">
              <el-radio-group v-model="form.scene_type" class="scene-radio-group">
                <el-radio-button value="QSO">
                  <el-icon><Connection /></el-icon>
                  <span class="scene-label">{{ t('apply.sceneQso') }}</span>
                </el-radio-button>
                <el-radio-button value="SWL">
                  <el-icon><Headset /></el-icon>
                  <span class="scene-label">{{ t('apply.sceneSwl') }}</span>
                </el-radio-button>
                <el-radio-button value="EYEBALL">
                  <el-icon><Location /></el-icon>
                  <span class="scene-label">{{ t('apply.sceneEyeball') }}</span>
                </el-radio-button>
              </el-radio-group>
            </el-form-item>

            <!-- 第二步：场景证据（动态） -->
            <el-divider content-position="left">{{ t('apply.stepEvidence') }}</el-divider>

            <!-- QSO 场景 -->
            <div v-if="form.scene_type === 'QSO'" class="form-grid">
              <el-form-item :label="t('apply.qsoDate')" prop="qso_date">
                <el-date-picker v-model="form.qso_date" type="date" value-format="YYYY-MM-DD" :placeholder="t('common.pickDate')" size="large" style="width:100%" :disabled-date="disableFuture" />
              </el-form-item>
              <el-form-item :label="t('apply.qsoTime')">
                <el-time-select v-model="form.qso_time" start="00:00" step="00:05" end="23:55" :placeholder="t('apply.timePlaceholder')" size="large" style="width:100%" />
              </el-form-item>
              <el-form-item :label="t('apply.freq')" prop="qso_freq">
                <el-input v-model="form.qso_freq" placeholder="14.270" size="large" />
              </el-form-item>
              <el-form-item :label="t('apply.band')">
                <el-input v-model="form.qso_band" placeholder="20m" size="large" />
              </el-form-item>
              <el-form-item :label="t('apply.mode')" prop="qso_mode">
                <el-input v-model="form.qso_mode" placeholder="SSB" size="large" />
              </el-form-item>

            </div>

            <!-- EYEBALL 场景 -->
            <div v-if="form.scene_type === 'EYEBALL'" class="form-grid">
              <el-form-item :label="t('apply.eyeballType')" prop="eyeball_type" class="form-item-full">
                <el-radio-group v-model="form.eyeball_type">
                  <el-radio-button value="OFFLINE">{{ t('apply.eyeballOffline') }}</el-radio-button>
                  <el-radio-button value="ONLINE">{{ t('apply.eyeballOnline') }}</el-radio-button>
                </el-radio-group>
              </el-form-item>
              <el-form-item :label="t('apply.eyeballDate')" prop="eyeball_date">
                <el-date-picker v-model="form.eyeball_date" type="date" value-format="YYYY-MM-DD" :placeholder="t('common.pickDate')" size="large" style="width:100%" :disabled-date="disableFuture" />
              </el-form-item>
              <el-form-item :label="t('apply.eyeballTime')">
                <el-time-select v-model="form.eyeball_time" start="00:00" step="00:05" end="23:55" :placeholder="t('apply.timePlaceholder')" size="large" style="width:100%" />
              </el-form-item>
              <el-form-item :label="t('apply.eyeballActivity')" prop="eyeball_activity" class="form-item-full">
                <el-input v-model="form.eyeball_activity" :placeholder="t('apply.eyeballActivityPh')" size="large" />
              </el-form-item>
              <el-form-item :label="t('apply.eyeballLocation')" class="form-item-full">
                <el-input v-model="form.eyeball_location" :placeholder="t('apply.eyeballLocationPh')" size="large" />
              </el-form-item>
            </div>

            <!-- SWL 场景 -->
            <div v-if="form.scene_type === 'SWL'" class="form-grid">
              <el-form-item :label="t('apply.swlDate')" prop="swl_date">
                <el-date-picker v-model="form.swl_date" type="date" value-format="YYYY-MM-DD" :placeholder="t('common.pickDate')" size="large" style="width:100%" :disabled-date="disableFuture" />
              </el-form-item>
              <el-form-item :label="t('apply.swlTime')" prop="swl_time">
                <el-time-picker v-model="form.swl_time" format="HH:mm" value-format="HH:mm" :placeholder="t('apply.timePlaceholder')" size="large" style="width:100%" />
              </el-form-item>
              <el-form-item :label="t('apply.swlFreq')" prop="swl_freq">
                <el-input v-model="form.swl_freq" placeholder="14.270" size="large" />
              </el-form-item>
              <el-form-item :label="t('apply.band')">
                <el-input v-model="form.swl_band" placeholder="20m" size="large" />
              </el-form-item>
              <el-form-item :label="t('apply.swlMode')" prop="swl_mode">
                <el-input v-model="form.swl_mode" placeholder="SSB" size="large" />
              </el-form-item>
            </div>

            <!-- 第三步：换卡理由 -->
            <el-divider content-position="left">{{ t('apply.stepReason') }}</el-divider>
            <el-form-item class="form-item-full">
              <el-input
                v-model="form.application_reason"
                type="textarea"
                :rows="3"
                :placeholder="t('apply.reasonPh')"
              />
            </el-form-item>

            <!-- 期望卡片版本（SWL 由对方先寄卡，无需选择版本） -->
            <template v-if="form.scene_type !== 'SWL'">
            <el-divider content-position="left">{{ t('apply.stepVersion', { n: stepNo.version }) }}</el-divider>
            <el-form-item prop="card_version" class="form-item-full">
              <div class="version-block">
                <p class="version-label">{{ t('apply.versionLabel') }}</p>
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
            <el-divider content-position="left">{{ t('apply.stepMail', { n: stepNo.mail }) }}</el-divider>
            <p class="field-tip" style="margin-bottom:12px;">
              <template v-if="form.scene_type === 'SWL'">{{ t('apply.mailTipSwl') }}</template>
              <template v-else>{{ t('apply.mailTip') }}</template>
            </p>
            <el-form-item class="form-item-full">
              <el-radio-group v-model="form.use_bureau" class="radio-group-modern">
                <el-radio-button :value="false">
                  <el-icon><Postcard /></el-icon>
                  {{ t('apply.directMail') }}
                </el-radio-button>
                <el-radio-button :value="true">
                  <el-icon><OfficeBuilding /></el-icon>
                  {{ t('apply.viaBureau') }}
                </el-radio-button>
              </el-radio-group>
            </el-form-item>
            <el-form-item v-if="form.use_bureau" :label="t('apply.bureauName')" prop="bureau_name" class="form-item-full">
              <el-input v-model="form.bureau_name" :placeholder="t('apply.bureauNamePh')" size="large" />
              <p class="field-tip">{{ t('apply.bureauTip') }}</p>
            </el-form-item>

            <!-- 收件信息 -->
            <el-divider content-position="left">{{ t('apply.stepInfo', { n: stepNo.info }) }}</el-divider>
            <el-alert v-if="form.scene_type === 'SWL'" type="info" :closable="false" style="margin-bottom:16px;">
              <p style="margin:0 0 4px;">{{ t('apply.flowTitle') }}</p>
              <p style="margin:0 0 2px;">{{ t('apply.flow1a') }}<b>{{ t('apply.flow1b') }}</b>{{ t('apply.flow1c') }}</p>
              <p style="margin:0 0 2px;">{{ t('apply.flow2') }}</p>
              <p style="margin:0 0 2px;">{{ t('apply.flow3') }}</p>
              <p style="margin:0 0 2px;">{{ t('apply.flow4') }}</p>
              <p style="margin:2px 0 0;">{{ t('apply.flowNoteA') }}<b>{{ t('apply.flowNoteB') }}</b>{{ t('apply.flowNoteC') }}</p>
            </el-alert>
            <div class="form-grid">
              <el-form-item :label="t('apply.callsign')" prop="call_sign">
                <el-input
                  v-model="form.call_sign"
                  :placeholder="t('apply.callsignPh')"
                  size="large"
                  prefix-icon="User"
                  @blur="form.call_sign = form.call_sign.toUpperCase()"
                />
              </el-form-item>

              <el-form-item :label="t('apply.name')" prop="name">
                <el-input v-model="form.name" :placeholder="t('apply.namePh')" size="large" />
              </el-form-item>

              <el-form-item :label="t('apply.email')" prop="email">
                <el-input v-model="form.email" :placeholder="t('apply.emailPh')" size="large" />
              </el-form-item>

              <el-form-item :label="t('apply.phone')">
                <el-input v-model="form.telephone" :placeholder="t('apply.phonePh')" size="large" />
              </el-form-item>

              <el-form-item :label="t('apply.postcode')" prop="postal_code">
                <el-input v-model="form.postal_code" :placeholder="t('apply.postcodePh')" size="large" />
              </el-form-item>

              <el-form-item :label="t('apply.address')" prop="address" class="form-item-full">
                <el-input
                  v-model="form.address"
                  type="textarea"
                  :rows="3"
                  :placeholder="form.use_bureau ? t('apply.addressBureauPh') : t('apply.addressPh')"
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
                {{ t('apply.submit') }}
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
            <h2>{{ t('apply.successTitle') }}</h2>
            <p>{{ t('apply.successLine', { code: result.request_code }) }}</p>
            <p class="success-note">{{ t('apply.successNote') }}</p>
            <div class="share-section">
              <p>{{ t('apply.refLabel') }}<strong>{{ result.request_code }}</strong></p>
              <p class="share-tip">{{ t('apply.shareTip') }}</p>
              <div class="share-link">
                <el-input v-model="statusLink" readonly size="large">
                  <template #append>
                    <el-button @click="copyLink">{{ t('common.copy') }}</el-button>
                  </template>
                </el-input>
              </div>
            </div>
            <div class="mail-section" v-if="mailInfo && mailInfo.address">
              <p class="mail-title">{{ t('apply.mailTitle', { call: mailInfo.call_sign }) }}</p>
              <p>{{ t('apply.mailAttn', { name: mailInfo.name }) }}</p>
              <p>{{ mailInfo.postal_code }} {{ mailInfo.address }}</p>
              <p class="mail-note">{{ t('apply.mailNote') }}</p>
            </div>
          </div>
        </div>
      </main>

      <!-- Footer -->
      <footer class="public-footer">
        <p>{{ t('common.footer') }}</p>
      </footer>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import api from '../../api'
import { ElMessage } from 'element-plus'
import { t } from '../../i18n'

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
    scene_type: [{ required: true, message: t('apply.ruleScene'), trigger: 'change' }],
    call_sign: [{ required: true, message: t('apply.ruleCallsign'), trigger: 'blur' }],
    name: [{ required: true, message: t('apply.ruleName'), trigger: 'blur' }],
    email: [
      { required: true, message: t('apply.ruleEmail'), trigger: 'blur' },
      { type: 'email', message: t('apply.ruleEmailFormat'), trigger: 'blur' }
    ],
    postal_code: [{ required: true, message: t('apply.rulePostcode'), trigger: 'blur' }],
    address: [{ required: true, message: t('apply.ruleAddress'), trigger: 'blur' }],
    card_version: [{ type: 'array', required: form.scene_type !== 'SWL', message: t('apply.ruleVersion'), trigger: 'change' }]
  }
  if (form.use_bureau) {
    r.bureau_name = [{ required: true, message: t('apply.ruleBureau'), trigger: 'blur' }]
  }
  // 场景证据动态必填
  if (form.scene_type === 'QSO') {
    r.qso_date = [{ required: true, message: t('apply.ruleQsoDate'), trigger: 'blur' }]
    r.qso_freq = [{ required: true, message: t('apply.ruleQsoFreq'), trigger: 'blur' }]
    r.qso_mode = [{ required: true, message: t('apply.ruleQsoMode'), trigger: 'blur' }]
  } else if (form.scene_type === 'EYEBALL') {
    r.eyeball_date = [{ required: true, message: t('apply.ruleEyeballDate'), trigger: 'blur' }]
    r.eyeball_activity = [{ required: true, message: t('apply.ruleEyeballActivity'), trigger: 'blur' }]
  } else if (form.scene_type === 'SWL') {
    r.swl_date = [{ required: true, message: t('apply.ruleSwlDate'), trigger: 'blur' }]
  r.swl_time = [{ required: true, message: t('apply.ruleSwlTime'), trigger: 'change' }]
    r.swl_freq = [{ required: true, message: t('apply.ruleSwlFreq'), trigger: 'blur' }]
    r.swl_mode = [{ required: true, message: t('apply.ruleSwlMode'), trigger: 'blur' }]
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
    ElMessage.warning(t('apply.warnIncomplete'))
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
    ElMessage.success(t('apply.successMsg'))
  } catch (e) {
    ElMessage.error(e.response?.data?.message || t('apply.failMsg'))
  } finally {
    submitting.value = false
  }
}

function copyLink() {
  navigator.clipboard.writeText(statusLink.value)
  ElMessage.success(t('apply.linkCopied'))
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
