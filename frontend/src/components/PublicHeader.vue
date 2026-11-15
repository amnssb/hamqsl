<template>
  <header class="public-header">
    <div class="logo-section">
      <div class="logo-icon" aria-hidden="true">QSL</div>
      <div class="logo-copy">
        <h1>{{ title }}</h1>
        <p v-if="subtitle" class="subtitle">{{ subtitle }}</p>
      </div>
    </div>
    <button type="button" class="home-back-link" :aria-label="t('common.backHome')" @click="goHome">
      <el-icon><ArrowLeft /></el-icon>
      <span>{{ t('common.backHome') }}</span>
    </button>
  </header>
</template>

<script setup>
import { useRouter } from 'vue-router'
import { t } from '../i18n'

defineProps({
  title: { type: String, required: true },
  subtitle: { type: String, default: '' },
})

const router = useRouter()
function goHome() {
  router.push('/')
}
</script>
