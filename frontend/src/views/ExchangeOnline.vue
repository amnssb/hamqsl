<template>
  <div>
    <el-card>
      <template #header>
        <el-tabs v-model="sceneTab" style="margin-top:-8px;">
          <el-tab-pane label="QSL 通联" name="QSO" />
          <el-tab-pane label="SWL 收听" name="SWL" />
          <el-tab-pane label="EYEBALL 见面" name="EYEBALL" />
        </el-tabs>
      </template>

      <el-form :inline="true" style="margin-bottom:16px;">
        <el-form-item label="呼号">
          <el-input v-model="query.call_sign" placeholder="对方呼号" clearable style="width:160px" @clear="loadData" />
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="query.review_status" placeholder="全部" clearable style="width:120px" @change="loadData">
            <el-option label="待审核" value="PENDING" /><el-option label="已通过" value="APPROVED" /><el-option label="已拒绝" value="REJECTED" />
          </el-select>
        </el-form-item>
        <el-form-item><el-button type="primary" @click="loadData">查询</el-button></el-form-item>
      </el-form>

      <el-table :data="list" border stripe v-loading="loading" @row-click="showDetail">
        <el-table-column prop="request_code" label="申请编号" width="110" />
        <el-table-column prop="call_sign" label="对方呼号" width="110" />
        <el-table-column label="证据摘要" min-width="200" show-overflow-tooltip>
          <template #default="{ row }">
            <el-tag :type="sceneTagType(row.scene_type)" size="small" style="margin-right:6px;">{{ sceneLabel(row.scene_type) }}</el-tag>{{ evidenceSummary(row) }}
          </template>
        </el-table-column>
        <el-table-column prop="application_reason" label="换卡理由" min-width="150" show-overflow-tooltip />
        <el-table-column prop="card_version" label="期望版本" width="130" show-overflow-tooltip />
        <el-table-column prop="email" label="邮箱" width="180" show-overflow-tooltip />
        <el-table-column prop="name" label="姓名" width="100" />
        <el-table-column prop="postal_code" label="邮编" width="80" />
        <el-table-column prop="address" label="地址" min-width="150" show-overflow-tooltip />
        <el-table-column label="审核状态" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="{ PENDING: 'warning', APPROVED: 'success', REJECTED: 'danger' }[row.review_status]">
              {{ { PENDING: '待审核', APPROVED: '已通过', REJECTED: '已拒绝' }[row.review_status] }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="已建卡" width="80" align="center">
          <template #default="{ row }">
            <el-icon v-if="row.card_created" color="#67C23A"><Check /></el-icon>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="300" fixed="right">
          <template #default="{ row }">
            <el-button link type="info" size="small" @click.stop="showDetail(row)">详情</el-button>
            <template v-if="row.review_status === 'PENDING'">
              <el-button link type="success" size="small" @click.stop="handleApprove(row)">通过</el-button>
              <el-button link type="danger" size="small" @click.stop="showReject(row)">拒绝</el-button>
            </template>
            <!-- SWL：对方寄卡到达并登记收卡后，才能创建回寄卡片 -->
            <template v-if="row.review_status === 'APPROVED' && !row.card_created && !(row.scene_type === 'SWL' && !row.return_received_at)">
              <el-button link type="primary" size="small" @click.stop="handleCreateCard(row)">创建卡片</el-button>
            </template>
            <el-button v-if="row.scene_type === 'SWL' && row.return_mailed_at && !row.return_received_at" link type="success" size="small" @click.stop="handleReceiveReturn(row)">登记收卡</el-button>
            <el-button v-if="row.review_status === 'APPROVED' && row.scene_type === 'SWL' && !row.return_mailed_at" link type="warning" size="small" @click.stop="showSendAddress(row)">重发地址</el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-pagination style="margin-top:16px;justify-content:flex-end;" v-model:current-page="query.page"
        v-model:page-size="query.size" :total="total" :page-sizes="[20,50,100]" layout="total, sizes, prev, pager, next"
        @size-change="loadData" @current-change="loadData" />
    </el-card>

    <!-- 拒绝对话框 -->
    <el-dialog v-model="rejectDialog" title="拒绝申请" width="400px" destroy-on-close>
      <el-form label-width="80px">
        <el-form-item label="拒绝原因">
          <el-input v-model="rejectReason" type="textarea" :rows="3" placeholder="请填写拒绝原因" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="rejectDialog = false">取消</el-button>
        <el-button type="danger" @click="handleReject">确认拒绝</el-button>
      </template>
    </el-dialog>

    <!-- 申请详情对话框 -->
    <el-dialog v-model="detailDialog" title="申请详情" width="720px" destroy-on-close>
      <template v-if="detailRow">
        <el-descriptions :column="2" border>
          <el-descriptions-item label="申请编号">{{ detailRow.request_code }}</el-descriptions-item>
          <el-descriptions-item label="申请场景">
            <el-tag :type="sceneTagType(detailRow.scene_type)" size="small">{{ sceneLabel(detailRow.scene_type) }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="对方呼号">{{ detailRow.call_sign }}</el-descriptions-item>
          <el-descriptions-item label="期望版本">{{ detailRow.card_version }}</el-descriptions-item>
          <el-descriptions-item label="姓名">{{ detailRow.name }}</el-descriptions-item>
          <el-descriptions-item label="邮箱">{{ detailRow.email }}</el-descriptions-item>
          <el-descriptions-item label="电话">{{ detailRow.telephone || '—' }}</el-descriptions-item>
          <el-descriptions-item label="邮编">{{ detailRow.postal_code }}</el-descriptions-item>
          <el-descriptions-item label="地址" :span="2">{{ detailRow.address }}{{ detailRow.scene_type === 'SWL' ? '（用于接收回寄卡片）' : '' }}</el-descriptions-item>
          <el-descriptions-item v-if="detailRow.use_bureau" label="对方卡片局" :span="2">{{ detailRow.bureau_name || '—' }}</el-descriptions-item>
        </el-descriptions>

        <!-- SWL 反寄流程：对方先寄卡，我方发送地址、对方登记单号 -->
        <template v-if="detailRow.scene_type === 'SWL'">
          <el-divider content-position="left">SWL 反寄登记</el-divider>
          <el-descriptions :column="2" border>
            <el-descriptions-item label="回寄地址">{{ detailRow.address_sent_at ? '已发送对方邮箱 ' + fmtTime(detailRow.address_sent_at) : '未发送' }}</el-descriptions-item>
            <el-descriptions-item label="对方寄出">{{ fmtTime(detailRow.return_mailed_at) }}</el-descriptions-item>
            <el-descriptions-item label="对方邮寄方式">{{ mailTypeText(detailRow.return_mail_type) }}</el-descriptions-item>
            <el-descriptions-item label="对方单号">{{ detailRow.return_tracking || '—' }}</el-descriptions-item>
            <el-descriptions-item label="我方收卡">{{ fmtTime(detailRow.return_received_at) }}</el-descriptions-item>
          </el-descriptions>
          <div v-if="detailRow.return_tracking" style="margin-top:10px;">
            <el-button type="primary" plain size="small" @click="openReturnTracking(detailRow)">在快递100 查询对方单号</el-button>
          </div>
          <div v-if="detailRow.return_address_text" class="reason-box" style="margin-top:12px;">{{ detailRow.return_address_text }}</div>
        </template>

        <!-- 场景证据 -->
        <el-divider content-position="left">场景证据</el-divider>
        <el-descriptions v-if="detailRow.scene_type === 'QSO'" :column="2" border>
          <el-descriptions-item label="通联日期">{{ detailRow.qso_date }}</el-descriptions-item>
          <el-descriptions-item label="通联时间">{{ detailRow.qso_time || '—' }}</el-descriptions-item>
          <el-descriptions-item label="频率">{{ detailRow.qso_freq }}</el-descriptions-item>
          <el-descriptions-item label="频段">{{ detailRow.qso_band || '—' }}</el-descriptions-item>
          <el-descriptions-item label="模式">{{ detailRow.qso_mode }}</el-descriptions-item>
          <el-descriptions-item label="RST 发送">{{ detailRow.qso_rst_sent || '—' }}</el-descriptions-item>
          <el-descriptions-item label="RST 接收">{{ detailRow.qso_rst_rcvd || '—' }}</el-descriptions-item>
        </el-descriptions>
        <el-descriptions v-else-if="detailRow.scene_type === 'EYEBALL'" :column="2" border>
          <el-descriptions-item label="见面类型">{{ detailRow.eyeball_type === 'ONLINE' ? '网络EYE' : '线下补换' }}</el-descriptions-item>
          <el-descriptions-item label="见面日期">{{ detailRow.eyeball_date }}</el-descriptions-item>
          <el-descriptions-item label="见面时间">{{ detailRow.eyeball_time || '—' }}</el-descriptions-item>
          <el-descriptions-item label="活动名称" :span="2">{{ detailRow.eyeball_activity }}</el-descriptions-item>
          <el-descriptions-item label="地点" :span="2">{{ detailRow.eyeball_location || '—' }}</el-descriptions-item>
        </el-descriptions>
        <el-descriptions v-else-if="detailRow.scene_type === 'SWL'" :column="2" border>
          <el-descriptions-item label="收听日期">{{ detailRow.swl_date }}</el-descriptions-item>
          <el-descriptions-item label="收听时间 (UTC+8)">{{ detailRow.swl_time || '—' }}</el-descriptions-item>
          <el-descriptions-item label="收听频率">{{ detailRow.swl_freq }}</el-descriptions-item>
          <el-descriptions-item label="频段">{{ detailRow.swl_band || '—' }}</el-descriptions-item>
          <el-descriptions-item label="收听模式">{{ detailRow.swl_mode }}</el-descriptions-item>
        </el-descriptions>

        <!-- 换卡理由 -->
        <el-divider content-position="left">换卡理由</el-divider>
        <div class="reason-box">{{ detailRow.application_reason }}</div>

        <!-- 备注 -->
        <template v-if="detailRow.remarks">
          <el-divider content-position="left">附加备注</el-divider>
          <div class="reason-box">{{ detailRow.remarks }}</div>
        </template>

        <!-- 审核信息 -->
        <el-divider content-position="left">审核信息</el-divider>
        <el-descriptions :column="2" border>
          <el-descriptions-item label="审核状态">
            <el-tag :type="{ PENDING: 'warning', APPROVED: 'success', REJECTED: 'danger' }[detailRow.review_status]">
              {{ { PENDING: '待审核', APPROVED: '已通过', REJECTED: '已拒绝' }[detailRow.review_status] }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="审核人">{{ detailRow.reviewed_by || '—' }}</el-descriptions-item>
          <el-descriptions-item label="审核时间">{{ detailRow.reviewed_at || '—' }}</el-descriptions-item>
          <el-descriptions-item label="已建卡">{{ detailRow.card_created ? '是' : '否' }}</el-descriptions-item>
          <el-descriptions-item v-if="detailRow.review_reason" label="审核意见" :span="2">{{ detailRow.review_reason }}</el-descriptions-item>
        </el-descriptions>
      </template>
    </el-dialog>

    <!-- 发送我的地址对话框（SWL 反寄） -->
    <el-dialog v-model="addrDialog" title="发送我的地址到对方邮箱" width="640px" destroy-on-close>
      <el-alert type="info" :closable="false" style="margin-bottom:12px;">
        审批通过时系统已自动发送一条默认回寄地址（台站档案地址优先，未配置时取地址簿第一条）到对方邮箱并显示在公开进度页。此处可重新勾选地址再次发送，覆盖此前内容（{{ currentRow?.email }}）。
      </el-alert>
      <el-table ref="addrTableRef" :data="myAddresses" border stripe max-height="320" @selection-change="addrSel = $event">
        <el-table-column type="selection" width="45" />
        <el-table-column prop="name" label="地址名称" width="130" />
        <el-table-column prop="address" label="详细地址" min-width="200" show-overflow-tooltip />
        <el-table-column prop="postal_code" label="邮编" width="90" />
      </el-table>
      <template #footer>
        <el-button @click="addrDialog = false">取消</el-button>
        <el-button type="primary" :loading="addrSending" @click="handleSendAddress">发送（已选 {{ addrSel.length }} 条）</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, watch, onMounted } from 'vue'
import api from '../api'
import { ElMessage, ElMessageBox } from 'element-plus'

const loading = ref(false)
const list = ref([])
const total = ref(0)
// 分家管理：QSL / SWL / EYEBALL 三个 Tab 各自独立列表
const sceneTab = ref('QSO')
const query = reactive({ call_sign: '', review_status: '', scene_type: 'QSO', page: 1, size: 20 })
const rejectDialog = ref(false)
const rejectId = ref(null)
const rejectReason = ref('')
const detailDialog = ref(false)
const detailRow = ref(null)

watch(sceneTab, t => { query.scene_type = t; query.page = 1; loadData() })

const sceneLabels = { QSO: 'QSL 通联', SWL: 'SWL 收听', EYEBALL: 'EYEBALL 见面' }
const sceneColors = { QSO: 'primary', SWL: 'info', EYEBALL: 'warning' }

function sceneLabel(t) { return sceneLabels[t] || t || '—' }
function sceneTagType(t) { return sceneColors[t] || 'info' }
function eyeballTypeText(row) {
  if (!row || row.scene_type !== 'EYEBALL') return ''
  return row.eyeball_type === 'ONLINE' ? ' · 网络EYE' : ' · 线下补换'
}
// 显示时间：精确到分钟（后端已存 UTC+8）
function fmtTime(s) { return s ? String(s).slice(0, 16) : '—' }

function evidenceSummary(row) {
  if (!row || !row.scene_type) return '—'
  switch (row.scene_type) {
    case 'QSO':
      return [row.qso_date, row.qso_time, row.qso_band, row.qso_mode].filter(Boolean).join(' ') || '—'
    case 'EYEBALL':
      return [row.eyeball_type === 'ONLINE' ? '网络EYE' : '线下补换', row.eyeball_date, row.eyeball_activity].filter(Boolean).join(' · ') || '—'
    case 'SWL':
      return [row.swl_date, row.swl_time, row.swl_band, row.swl_mode].filter(Boolean).join(' ') || '—'
    default:
      return '—'
  }
}

async function loadData() {
  loading.value = true
  try {
    const res = await api.get('/exchange/online/requests', { params: query })
    list.value = res.items
    total.value = res.total
  } finally { loading.value = false }
}

function showDetail(row) {
  detailRow.value = row
  detailDialog.value = true
}

async function handleApprove(row) {
  await api.post(`/exchange/online/requests/${row.id}/approve`)
  ElMessage.success('审核通过')
  loadData()
}

function showReject(row) {
  rejectId.value = row.id
  rejectReason.value = ''
  rejectDialog.value = true
}

async function handleReject() {
  await api.post(`/exchange/online/requests/${rejectId.value}/reject`, { review_status: 'REJECTED', review_reason: rejectReason.value })
  ElMessage.success('已拒绝')
  rejectDialog.value = false
  loadData()
}

async function handleCreateCard(row) {
  await api.post(`/exchange/online/requests/${row.id}/create-card`)
  ElMessage.success('卡片创建成功')
  loadData()
}

const mailTypeText = t => ({ REGISTERED: '挂号信', ORDINARY: '平信' }[t] || '—')

async function handleReceiveReturn(row) {
  const { value } = await ElMessageBox.prompt('确认已收到对方寄来的卡片？确认后将自动创建回寄卡片记录。可填写备注（如实际收到日期/状态）', '登记收卡', {
    inputPlaceholder: '备注（可选）', inputValue: '', confirmButtonText: '确认收卡', cancelButtonText: '取消',
  }).catch(() => ({ value: null }))
  if (value === null) return
  const res = await api.post(`/exchange/online/requests/${row.id}/receive-return`, { remarks: value })
  let msg = '收卡已登记（收卡记录 ' + res.receive_code + '）'
  if (res.card_code) msg += '，回寄卡片已自动建单（' + res.card_code + '）'
  if (res.mail_error) ElMessage.warning(msg + '，但通知邮件发送失败：' + res.mail_error)
  else ElMessage.success(msg + '，对方将收到邮件通知')
  loadData()
}
function openReturnTracking(row) {
  window.open('https://www.kuaidi100.com/chaxun?nu=' + encodeURIComponent(row.return_tracking), '_blank')
}

// ---- 发送我的地址（SWL 反寄） ----
const addrDialog = ref(false)
const addrTableRef = ref(null)
const addrSel = ref([])
const addrSending = ref(false)
const myAddresses = ref([])
const currentRow = ref(null)

async function showSendAddress(row) {
  currentRow.value = row
  if (myAddresses.value.length === 0) {
    try {
      const res = await api.get('/address/book', { params: { page: 1, size: 100 } })
      myAddresses.value = res.items || []
    } catch (e) { /* 拦截器已提示 */ }
  }
  if (myAddresses.value.length === 0) {
    ElMessage.warning('请先在「我的地址」中添加您的回寄地址')
    return
  }
  addrSel.value = []
  addrDialog.value = true
}

async function handleSendAddress() {
  if (addrSel.value.length === 0) { ElMessage.warning('请至少勾选一条地址'); return }
  addrSending.value = true
  try {
    const res = await api.post(`/exchange/online/requests/${currentRow.value.id}/send-address`, { address_ids: addrSel.value.map(a => a.id) })
    if (res.mail_error) ElMessage.warning('地址已留档，但邮件发送失败：' + res.mail_error)
    else ElMessage.success('地址已发送到对方邮箱')
    addrDialog.value = false
    loadData()
  } finally { addrSending.value = false }
}

onMounted(loadData)
</script>

<style scoped>
.reason-box {
  background: #f9f9f4;
  border: 1px solid #e8e4dc;
  border-radius: 4px;
  padding: 12px 16px;
  font-size: 14px;
  line-height: 1.7;
  color: #444;
  white-space: pre-wrap;
  word-break: break-all;
}
</style>
