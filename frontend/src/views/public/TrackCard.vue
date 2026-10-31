<template>
  <div class="public-page">
    <div class="public-container">
      <header class="public-header">
        <div class="logo-section">
          <div class="logo-icon">QSL</div>
          <div>
            <h1>{{ t('track.title') }}</h1>
            <p class="subtitle">{{ t('track.subtitle') }}</p>
          </div>
        </div>
      </header>

      <main class="public-main">
        <div class="card modern-card">
          <div class="card-header">
            <h2>{{ t('track.formTitle') }}</h2>
          </div>

          <el-form :model="form" label-position="top" class="modern-form" @submit.prevent>
            <el-form-item :label="t('track.numberLabel')">
              <el-input
                v-model="form.tracking_number"
                :placeholder="t('track.numberPlaceholder')"
                size="large"
                clearable
                @keyup.enter="handleTrack"
              />
            </el-form-item>

            <p class="carrier-tip">{{ t('track.carrierTip') }}</p>

            <div class="form-actions">
              <el-button
                type="primary"
                size="large"
                :disabled="!form.tracking_number"
                @click="handleTrack"
                class="submit-btn"
              >
                <el-icon><Search /></el-icon>
                {{ t('track.submit') }}
              </el-button>
            </div>
          </el-form>
        </div>

        <div class="card modern-card tip-card">
          <b>{{ t('track.noNumberTitle') }}</b>
          <p>{{ t('track.noNumberDesc') }}</p>
          <el-button link type="primary" @click="$router.push('/status')">{{ t('track.goStatus') }}</el-button>
        </div>
      </main>

      <footer class="public-footer">
        <p>{{ t('common.footer') }}</p>
      </footer>
    </div>
  </div>
</template>

<script setup>
import { reactive, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { t } from '../../i18n'

const route = useRoute()

const form = reactive({
  tracking_number: ''
})

// 直接跳转快递100：单号自动识别承运商，无需本地中转查询
function handleTrack() {
  const nu = (form.tracking_number || '').trim()
  if (!nu) {
    ElMessage.warning(t('track.emptyNumber'))
    return
  }
  window.open('https://www.kuaidi100.com/chaxun?nu=' + encodeURIComponent(nu), '_blank')
}

onMounted(() => {
  // /track/:code 带单号访问时直接跳转快递100
  if (route.params.code) {
    form.tracking_number = route.params.code
    handleTrack()
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

.public-container {
  max-width: 600px;
  margin: 0 auto;
}

.public-header {
  padding: 36px 0 28px;
}

.logo-section {
  display: flex;
  align-items: center;
  gap: 16px;
}

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

.subtitle {
  margin: 6px 0 0;
  color: var(--qsl-muted);
  font-size: 14px;
}

.modern-card {
  background: var(--qsl-card-bg);
  border: 1px solid var(--qsl-line);
  border-radius: 18px;
  padding: 36px 32px;
  box-shadow: var(--qsl-shadow-2);
  margin-bottom: 24px;
}

.card-header {
  margin-bottom: 24px;
  padding-bottom: 16px;
  border-bottom: 1px solid var(--qsl-line);
}

.card-header h2 {
  margin: 0 0 6px;
  font-size: 20px;
  font-weight: 700;
  color: var(--qsl-heading);
}

.carrier-tip {
  margin: 0;
  color: var(--qsl-muted);
  font-size: 13px;
  line-height: 1.7;
}

.form-actions {
  margin-top: 24px;
  text-align: center;
}

.submit-btn {
  width: 200px;
  height: 48px;
  font-size: 15.5px;
  font-weight: 650;
  border-radius: 10px;
}

.tip-card b { color: var(--qsl-heading); font-size: 14px; }
.tip-card p { margin: 8px 0 10px; color: var(--qsl-muted); font-size: 13px; line-height: 1.7; }

.public-footer {
  text-align: center;
  padding: 24px 0 36px;
  color: var(--qsl-muted);
  font-size: 13px;
}

</style>
