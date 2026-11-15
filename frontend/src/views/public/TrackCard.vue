<template>
  <div class="public-page">
    <div class="public-container">
      <PublicHeader :title="t('track.title')" :subtitle="t('track.subtitle')" />

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
import PublicHeader from '../../components/PublicHeader.vue'

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
.public-container { max-width: 600px; }

.carrier-tip {
  margin: 0;
  color: var(--qsl-muted);
  font-size: 13px;
  line-height: 1.7;
}

.tip-card b { color: var(--qsl-heading); font-size: 14px; }
.tip-card p { margin: 8px 0 10px; color: var(--qsl-muted); font-size: 13px; line-height: 1.7; }

</style>
