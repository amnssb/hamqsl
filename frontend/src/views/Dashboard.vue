<template>
  <div class="dashboard">
    <div class="page-heading">
      <div><h1>业务总览</h1><p>追踪你的通联、卡片与换卡工作流。</p></div>
      <el-button type="primary" @click="$router.push('/admin/qso')"><el-icon><Plus /></el-icon>新增通联</el-button>
    </div>

    <div class="stat-grid">
      <el-card v-for="card in statCards" :key="card.label" class="stat-card" shadow="never">
        <div class="stat-top"><span class="stat-label">{{ card.label }}</span><span class="stat-mark" :class="card.tone"><component :is="card.icon" /></span></div>
        <div class="stat-value">{{ card.value }}</div>
        <div class="stat-foot">较上一周期 <b>{{ card.note }}</b></div>
      </el-card>
    </div>

    <div class="dashboard-grid">
      <el-card class="workflow-card" shadow="never">
        <template #header><div class="section-heading"><div><h2>待处理事项</h2><p>需要你关注的工作</p></div><el-tag type="warning" effect="plain">{{ pendingTotal }} 项</el-tag></div></template>
        <div v-for="item in pendingItems" :key="item.label" class="workflow-row">
          <span class="workflow-icon" :class="item.tone"><component :is="item.icon" /></span>
          <div class="workflow-copy"><strong>{{ item.label }}</strong><span>{{ item.hint }}</span></div>
          <b class="workflow-value">{{ item.value }}</b>
        </div>
      </el-card>
      <el-card class="workflow-card" shadow="never">
        <template #header><div class="section-heading"><div><h2>业务统计</h2><p>当前账户数据概览</p></div><span class="section-code">LIVE</span></div></template>
        <div class="metric-list">
          <div v-for="item in metrics" :key="item.label" class="metric-row"><span>{{ item.label }}</span><strong>{{ item.value }}</strong></div>
        </div>
        <div class="insight"><span class="insight-bar"></span><div><strong>保持记录完整</strong><p>每一次通联都让下一张 QSL 更容易抵达。</p></div></div>
      </el-card>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import api from '../api'
const summary = ref({ total_qso:0, total_cards:0, total_sent:0, total_received:0, pending_issue:0, pending_sent:0, pending_receive:0, pending_exchange_review:0, total_exchange_requests:0, total_offline_activities:0 })
const statCards = computed(() => [
  { label:'通联记录', value:summary.value.total_qso, icon:'Document', tone:'orange', note:'持续增长' },
  { label:'卡片总数', value:summary.value.total_cards, icon:'Postcard', tone:'green', note:'库存同步' },
  { label:'已发卡', value:summary.value.total_sent, icon:'Promotion', tone:'yellow', note:'流程进行中' },
  { label:'已收卡', value:summary.value.total_received, icon:'Box', tone:'blue', note:'档案已更新' },
])
const pendingItems = computed(() => [
  { label:'待制卡', hint:'通联已完成，等待制作', value:summary.value.pending_issue, icon:'Postcard', tone:'orange' },
  { label:'待发信', hint:'已制卡，等待邮寄', value:summary.value.pending_sent, icon:'Promotion', tone:'yellow' },
  { label:'待收卡', hint:'已寄出，等待确认', value:summary.value.pending_receive, icon:'Box', tone:'blue' },
  { label:'待审核换卡申请', hint:'线上申请需要处理', value:summary.value.pending_exchange_review, icon:'Bell', tone:'green' },
])
const pendingTotal = computed(() => pendingItems.value.reduce((sum, item) => sum + Number(item.value || 0), 0))
const metrics = computed(() => [
  { label:'通联记录总数', value:summary.value.total_qso }, { label:'卡片总数', value:summary.value.total_cards },
  { label:'换卡申请', value:summary.value.total_exchange_requests }, { label:'线下活动', value:summary.value.total_offline_activities },
])
onMounted(async () => { try { summary.value = await api.get('/dashboard/summary') } catch (e) { console.error(e) } })
</script>

<style scoped>
.dashboard { animation: enter .35s ease both; }
@keyframes enter { from { opacity:0; transform:translateY(7px); } to { opacity:1; transform:none; } }
.stat-grid { display:grid; grid-template-columns:repeat(4, minmax(0,1fr)); gap:16px; }
.stat-card { min-height:154px; transition:transform .2s, box-shadow .2s; }
.stat-card:hover { transform:translateY(-3px); box-shadow:6px 8px 0 rgba(24,45,61,.08); }
.stat-top, .section-heading, .workflow-row, .metric-row { display:flex; align-items:center; justify-content:space-between; }
.stat-label { color:var(--qsl-muted); font-size:13px; font-weight:650; }
.stat-mark, .workflow-icon { display:grid; place-items:center; width:34px; height:34px; font-size:17px; }
.stat-mark.orange, .workflow-icon.orange { color:var(--qsl-orange); background:#fff0eb; }.stat-mark.green, .workflow-icon.green { color:var(--qsl-green); background:#e9f5f1; }.stat-mark.yellow, .workflow-icon.yellow { color:#9a6b00; background:#fff5d9; }.stat-mark.blue, .workflow-icon.blue { color:#286486; background:#e8f2f7; }
.stat-value { margin-top:19px; color:var(--qsl-navy); font-size:38px; font-weight:800; letter-spacing:-.06em; }
.stat-foot { margin-top:7px; color:#98948b; font-size:11px; }.stat-foot b { color:var(--qsl-green); font-weight:700; }
.dashboard-grid { display:grid; grid-template-columns:1.1fr .9fr; gap:16px; margin-top:16px; }.section-heading h2 { margin:0; color:var(--qsl-navy); font-size:17px; }.section-heading p { margin:5px 0 0; color:var(--qsl-muted); font-size:12px; }.section-code { color:var(--qsl-green); font-size:10px; font-weight:800; letter-spacing:.14em; }
.workflow-row { justify-content:flex-start; gap:12px; padding:16px 0; border-bottom:1px solid #eeeae2; }.workflow-row:last-child { border-bottom:0; }.workflow-copy { display:flex; flex:1; flex-direction:column; gap:4px; }.workflow-copy strong { color:var(--qsl-navy); font-size:13px; }.workflow-copy span { color:var(--qsl-muted); font-size:11px; }.workflow-value { color:var(--qsl-navy); font-size:20px; }
.metric-list { padding:4px 0 14px; }.metric-row { padding:13px 0; border-bottom:1px solid #eeeae2; color:var(--qsl-muted); font-size:13px; }.metric-row strong { color:var(--qsl-navy); font-size:16px; }.insight { display:flex; gap:12px; margin-top:10px; padding:14px; background:#f1f7f5; }.insight-bar { width:4px; background:var(--qsl-green); }.insight strong { color:var(--qsl-navy); font-size:12px; }.insight p { margin:4px 0 0; color:var(--qsl-muted); font-size:11px; }
@media (max-width:1000px) { .stat-grid { grid-template-columns:repeat(2,1fr); }.dashboard-grid { grid-template-columns:1fr; } }
@media (max-width:560px) { .stat-grid { grid-template-columns:1fr; } }
</style>
