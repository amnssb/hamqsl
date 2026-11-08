<template>
  <div class="dashboard">
    <div class="page-heading">
      <div>
        <h1>业务总览</h1>
        <p>实时追踪无线电通联、QSL 卡片制作与换卡进度。</p>
      </div>
      <div class="page-actions">
        <el-button type="primary" @click="$router.push('/admin/qso')">
          <el-icon><Plus /></el-icon>
          <span>新增通联记录</span>
        </el-button>
      </div>
    </div>

    <div class="stat-grid">
      <el-card v-for="card in statCards" :key="card.label" class="stat-card" shadow="never" @click="$router.push(card.path)">
        <div class="stat-top">
          <span class="stat-label">{{ card.label }}</span>
          <div class="stat-mark" :class="card.tone">
            <component :is="card.icon" />
          </div>
        </div>
        <div class="stat-value">{{ card.value }}</div>
        <div class="stat-foot">
          <span class="stat-tag" :class="card.tone">{{ card.note }}</span>
          <span class="stat-hint">点击直达列表</span>
        </div>
      </el-card>
    </div>

    <div class="dashboard-grid">
      <el-card class="workflow-card" shadow="never">
        <template #header>
          <div class="section-heading">
            <div>
              <h2>待处理事项</h2>
              <p>需要跟进的卡片与交换任务（点击卡片直达处理）</p>
            </div>
            <span class="pending-badge">{{ pendingTotal }} 项待办</span>
          </div>
        </template>
        <div class="workflow-list">
          <div v-for="item in pendingItems" :key="item.label" class="workflow-row" @click="$router.push(item.path)">
            <div class="workflow-icon" :class="item.tone">
              <component :is="item.icon" />
            </div>
            <div class="workflow-copy">
              <strong>{{ item.label }}</strong>
              <span>{{ item.hint }}</span>
            </div>
            <div class="workflow-value-box">
              <span class="workflow-value" :class="{ 'has-val': item.value > 0 }">{{ item.value }}</span>
              <el-icon class="workflow-arrow"><ArrowRight /></el-icon>
            </div>
          </div>
        </div>
      </el-card>

      <el-card class="workflow-card" shadow="never">
        <template #header>
          <div class="section-heading">
            <div>
              <h2>业务统计</h2>
              <p>系统核心累计指标概览</p>
            </div>
            <span class="section-code"><span class="live-dot"></span>LIVE</span>
          </div>
        </template>
        <div class="metric-list">
          <div v-for="item in metrics" :key="item.label" class="metric-row">
            <span class="metric-label">{{ item.label }}</span>
            <strong class="metric-value">{{ item.value }}</strong>
          </div>
        </div>
        <div class="insight">
          <div class="insight-icon"><el-icon><InfoFilled /></el-icon></div>
          <div class="insight-body">
            <strong>保持通联档案完整</strong>
            <p>每一次规范记录，都让下一张承载友谊的 QSL 卡片更稳妥抵达。</p>
          </div>
        </div>
      </el-card>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import api from '../api'
import { icons } from '../plugins/icons'

const summary = ref({
  total_qso: 0,
  total_cards: 0,
  total_sent: 0,
  total_received: 0,
  pending_issue: 0,
  pending_sent: 0,
  pending_receive: 0,
  pending_exchange_review: 0,
  total_exchange_requests: 0,
  total_offline_activities: 0
})

// icon 直接传组件对象，不再用字符串让 <component :is> 去全局注册表里找
const statCards = computed(() => [
  { label: '通联日志', value: summary.value.total_qso, icon: icons.Document, tone: 'blue', note: '日志档案', path: '/admin/qso' },
  { label: '卡片总数', value: summary.value.total_cards, icon: icons.Postcard, tone: 'indigo', note: '卡片库存', path: '/admin/cards' },
  { label: '已发卡片', value: summary.value.total_sent, icon: icons.Promotion, tone: 'amber', note: '已递送', path: '/admin/cards?flow_status=SENT' },
  { label: '已收卡片', value: summary.value.total_received, icon: icons.Box, tone: 'emerald', note: '回卡已入库', path: '/admin/receive' },
])

const pendingItems = computed(() => [
  { label: '待制卡', hint: '通联已完成，等待制作卡片', value: summary.value.pending_issue, icon: icons.Postcard, tone: 'blue', path: '/admin/cards?flow_status=PENDING_ISSUE' },
  { label: '待发信', hint: '卡片已制作，等待封装寄出', value: summary.value.pending_sent, icon: icons.Promotion, tone: 'amber', path: '/admin/cards?flow_status=ISSUED' },
  { label: '待收卡', hint: '卡片已寄出，等待对方收件确认', value: summary.value.pending_receive, icon: icons.Box, tone: 'indigo', path: '/admin/cards?flow_status=SENT' },
  { label: '待审核换卡申请', hint: '公开门户提交的线上换卡请求', value: summary.value.pending_exchange_review, icon: icons.Bell, tone: 'emerald', path: '/admin/exchange/online' },
])

const pendingTotal = computed(() => pendingItems.value.reduce((sum, item) => sum + Number(item.value || 0), 0))

const metrics = computed(() => [
  { label: '通联记录总数 (QSO)', value: summary.value.total_qso },
  { label: '卡片流转总数 (QSL)', value: summary.value.total_cards },
  { label: '线上换卡申请数', value: summary.value.total_exchange_requests },
  { label: '线下活动聚会记录', value: summary.value.total_offline_activities },
])

onMounted(async () => {
  try {
    const res = await api.get('/dashboard/summary')
    if (res) summary.value = res
  } catch (e) {
    console.error(e)
  }
})
</script>

<style scoped>
/* 分类强调色只定义"色相"（--qsl-tone-* 在 tokens.css），底色一律用 color-mix
   叠在主题表面之上，这样暗色主题下不会出现"深色卡片上贴浅色色块"的割裂感。 */
.dashboard {
  --tone-blue: var(--qsl-tone-blue);
  --tone-indigo: var(--qsl-tone-indigo);
  --tone-amber: var(--qsl-tone-amber);
  --tone-emerald: var(--qsl-tone-emerald);

  animation: enter 0.3s cubic-bezier(0.16, 1, 0.3, 1) both;
}
@keyframes enter {
  from { opacity: 0; transform: translateY(8px); }
  to { opacity: 1; transform: translateY(0); }
}

.stat-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 20px;
}

.stat-card {
  border-radius: 14px;
  background: var(--qsl-panel);
  border: 1px solid var(--qsl-line);
  padding: 4px;
  box-shadow: var(--qsl-shadow-1);
  cursor: pointer;
  transition: all 0.22s cubic-bezier(0.4, 0, 0.2, 1);
}
.stat-card:hover {
  transform: translateY(-3px);
  border-color: var(--el-color-primary);
  box-shadow: var(--qsl-shadow-2);
}

.stat-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.stat-label {
  color: var(--qsl-muted);
  font-size: 13.5px;
  font-weight: 600;
}

.stat-mark {
  display: grid;
  place-items: center;
  width: 40px;
  height: 40px;
  border-radius: 10px;
  font-size: 18px;
}
/* 分类强调色走 --qsl-tone-*（定义在 tokens.css）：
   底色用 color-mix 叠在主题表面之上，暗色主题下不会刺眼，
   也不再出现裸十六进制（stylelint color-no-hex 会拦）。 */
.stat-mark.blue { color: var(--qsl-tone-blue); background: color-mix(in srgb, var(--qsl-tone-blue) 12%, transparent); }
.stat-mark.indigo { color: var(--qsl-tone-indigo); background: color-mix(in srgb, var(--qsl-tone-indigo) 12%, transparent); }
.stat-mark.amber { color: var(--qsl-tone-amber); background: color-mix(in srgb, var(--qsl-tone-amber) 12%, transparent); }
.stat-mark.emerald { color: var(--qsl-tone-emerald); background: color-mix(in srgb, var(--qsl-tone-emerald) 12%, transparent); }

.stat-value {
  margin-top: 14px;
  color: var(--qsl-heading);
  font-family: var(--qsl-font-mono);
  font-size: 34px;
  font-weight: 800;
  letter-spacing: -0.04em;
  font-variant-numeric: tabular-nums;
  line-height: 1.1;
}

.stat-foot {
  margin-top: 12px;
  display: flex;
  align-items: center;
  gap: 8px;
}
.stat-tag {
  font-size: 11px;
  font-weight: 650;
  padding: 2px 8px;
  border-radius: 9999px;
}
.stat-tag.blue { background: color-mix(in srgb, var(--qsl-tone-blue) 10%, transparent); color: var(--qsl-tone-blue); }
.stat-tag.indigo { background: color-mix(in srgb, var(--qsl-tone-indigo) 10%, transparent); color: var(--qsl-tone-indigo); }
.stat-tag.amber { background: color-mix(in srgb, var(--qsl-tone-amber) 10%, transparent); color: var(--qsl-tone-amber); }
.stat-tag.emerald { background: color-mix(in srgb, var(--qsl-tone-emerald) 10%, transparent); color: var(--qsl-tone-emerald); }
.stat-hint {
  color: var(--qsl-muted);
  font-size: 11.5px;
}

.dashboard-grid {
  display: grid;
  grid-template-columns: 1.15fr 0.85fr;
  gap: 20px;
  margin-top: 24px;
}

.workflow-card {
  border-radius: 14px;
  background: var(--qsl-panel);
  border: 1px solid var(--qsl-line);
  box-shadow: 0 1px 3px 0 rgba(0, 0, 0, 0.04);
}

.section-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.section-heading h2 {
  margin: 0;
  color: var(--qsl-heading);
  font-size: 17px;
  font-weight: 700;
  letter-spacing: -0.01em;
}
.section-heading p {
  margin: 3px 0 0;
  color: var(--qsl-muted);
  font-size: 12.5px;
}

.pending-badge {
  font-size: 12px;
  font-weight: 650;
  color: var(--qsl-primary);
  background: var(--qsl-active-tint);
  padding: 4px 12px;
  border-radius: 9999px;
  border: 1px solid var(--qsl-line);
}

.section-code {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
  font-weight: 750;
  letter-spacing: 0.1em;
  color: var(--qsl-green);
  background: color-mix(in srgb, var(--qsl-green) 12%, transparent);
  padding: 3px 10px;
  border-radius: 9999px;
}
.live-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--qsl-green);
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--qsl-green) 30%, transparent);
}

.workflow-list {
  display: flex;
  flex-direction: column;
}

.workflow-row {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 14px 12px;
  border-radius: var(--qsl-radius);
  transition: background 0.18s ease;
  border-bottom: 1px solid var(--qsl-line);
  cursor: pointer;
}
.workflow-row:last-child {
  border-bottom: 0;
}
.workflow-row:hover {
  background: var(--qsl-panel-2);
}

.workflow-icon {
  display: grid;
  place-items: center;
  width: 38px;
  height: 38px;
  border-radius: 10px;
  font-size: 17px;
  flex: none;
}
.workflow-icon.blue { color: var(--qsl-tone-blue); background: color-mix(in srgb, var(--qsl-tone-blue) 12%, transparent); }
.workflow-icon.amber { color: var(--qsl-tone-amber); background: color-mix(in srgb, var(--qsl-tone-amber) 12%, transparent); }
.workflow-icon.indigo { color: var(--qsl-tone-indigo); background: color-mix(in srgb, var(--qsl-tone-indigo) 12%, transparent); }
.workflow-icon.emerald { color: var(--qsl-tone-emerald); background: color-mix(in srgb, var(--qsl-tone-emerald) 12%, transparent); }

.workflow-copy {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 3px;
  min-width: 0;
}
.workflow-copy strong {
  color: var(--qsl-heading);
  font-size: 13.5px;
  font-weight: 650;
}
.workflow-copy span {
  color: var(--qsl-muted);
  font-size: 12px;
}

.workflow-value-box {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: none;
}
.workflow-arrow {
  color: var(--qsl-muted);
  font-size: 14px;
  transition: transform 0.2s ease, color 0.2s ease;
}
.workflow-row:hover .workflow-arrow {
  transform: translateX(3px);
  color: var(--qsl-primary);
}
.workflow-value {
  display: inline-block;
  min-width: 28px;
  padding: 2px 8px;
  text-align: center;
  border-radius: 9999px;
  background: var(--qsl-panel-2);
  color: var(--qsl-muted);
  font-size: 14px;
  font-weight: 750;
  font-variant-numeric: tabular-nums;
}
.workflow-value.has-val {
  background: var(--qsl-active-tint);
  color: var(--qsl-primary);
}

.metric-list {
  padding: 4px 0 10px;
}
.metric-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 6px;
  border-bottom: 1px solid var(--qsl-line);
}
.metric-row:last-child {
  border-bottom: 0;
}
.metric-label {
  color: var(--qsl-muted);
  font-size: 13.5px;
}
.metric-value {
  color: var(--qsl-heading);
  font-size: 16px;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
}

.insight {
  display: flex;
  gap: 12px;
  margin-top: 14px;
  padding: 14px 16px;
  background: var(--qsl-panel-2);
  border: 1px solid var(--qsl-line);
  border-radius: var(--qsl-radius);
}
.insight-icon {
  color: var(--qsl-green);
  font-size: 18px;
  flex: none;
  margin-top: 1px;
}
.insight-body strong {
  display: block;
  color: var(--qsl-heading);
  font-size: 12.5px;
  font-weight: 700;
}
.insight-body p {
  margin: 3px 0 0;
  color: var(--qsl-muted);
  font-size: 12px;
  line-height: 1.5;
}

@media (max-width: 1060px) {
  .stat-grid { grid-template-columns: repeat(2, 1fr); }
  .dashboard-grid { grid-template-columns: 1fr; }
}
@media (max-width: 600px) {
  .stat-grid { grid-template-columns: 1fr; }
}
</style>
