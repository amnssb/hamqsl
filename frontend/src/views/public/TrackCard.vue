<template>
  <div class="public-page">
    <div class="public-container">
      <header class="public-header">
        <div class="logo-section">
          <div class="logo-icon">QSL</div>
          <div>
            <h1>快递追踪</h1>
            <p class="subtitle">查询挂号信物流状态</p>
          </div>
        </div>
      </header>

      <main class="public-main">
        <div class="card modern-card">
          <div class="card-header">
            <h2>输入单号</h2>
          </div>

          <el-form :model="form" label-position="top" class="modern-form" @submit.prevent>
            <el-form-item label="挂号信 / 快递单号">
              <el-input
                v-model="form.tracking_number"
                placeholder="例如: RR123456789CN"
                size="large"
                clearable
                @keyup.enter="handleTrack"
              />
            </el-form-item>

            <p class="carrier-tip">点击查询将直接跳转到快递100 官网查询实时物流，单号自动识别承运商（中国邮政 / EMS / 顺丰 / 圆通 / 中通 / 申通 / 韵达 / 京东等）。</p>

            <div class="form-actions">
              <el-button
                type="primary"
                size="large"
                :disabled="!form.tracking_number"
                @click="handleTrack"
                class="submit-btn"
              >
                <el-icon><Search /></el-icon>
                查询物流
              </el-button>
            </div>
          </el-form>
        </div>

        <div class="card modern-card tip-card">
          <b>没有单号？</b>
          <p>挂号信号码会随寄出邮件发送到您的邮箱；也可以在申请进度页查看物流信息。</p>
          <el-button link type="primary" @click="$router.push('/status')">前往申请进度查询</el-button>
        </div>
      </main>

      <footer class="public-footer">
        <p>QSL 卡片管理系统 · 业余无线电</p>
      </footer>
    </div>
  </div>
</template>

<script setup>
import { reactive, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'

const route = useRoute()

const form = reactive({
  tracking_number: ''
})

// 直接跳转快递100：单号自动识别承运商，无需本地中转查询
function handleTrack() {
  const nu = (form.tracking_number || '').trim()
  if (!nu) {
    ElMessage.warning('请输入单号')
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
  padding: 20px;
}

.public-container {
  max-width: 600px;
  margin: 0 auto;
}

.public-header {
  padding: 40px 0 30px;
}

.logo-section {
  display: flex;
  align-items: center;
  gap: 16px;
}

.logo-icon {
  width: 42px;
  height: 42px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
  font-weight: bold;
}

.public-header h1 {
  margin: 0;
  font-size: 28px;
  font-weight: 600;
}

.subtitle {
  margin: 8px 0 0;
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

.carrier-tip {
  margin: 0;
  color: #909399;
  font-size: 13px;
  line-height: 1.7;
}

.form-actions {
  margin-top: 24px;
  text-align: center;
}

.submit-btn {
  width: 200px;
  height: 50px;
  font-size: 16px;
  border-radius: 12px;
}

.tip-card b { color: #1a1a1a; font-size: 14px; }
.tip-card p { margin: 8px 0 10px; color: #909399; font-size: 13px; line-height: 1.7; }

.public-footer {
  text-align: center;
  padding: 30px 0;
  font-size: 14px;
}

/* Kumo visual layer */
.public-page { background:var(--qsl-paper); color:var(--qsl-ink); }.public-header { color:var(--qsl-navy); border-bottom:1px solid var(--qsl-line); text-align:left; padding:34px 0 28px; }.logo-section { justify-content:flex-start; }.logo-icon { width:42px; height:42px; border-radius:0; color:var(--qsl-navy); background:var(--qsl-yellow); font-size:14px; }.public-header h1 { color:var(--qsl-navy); font-size:24px; }.subtitle { color:var(--qsl-muted); font-size:13px; }.modern-card { border:1px solid var(--qsl-line); border-radius:2px; box-shadow:3px 3px 0 rgba(24,45,61,.07); }.card-header { border-bottom-color:#eeeae2; }.card-header h2 { color:var(--qsl-navy); }.public-footer { color:#9c978d; border-top:1px solid var(--qsl-line); }
</style>
