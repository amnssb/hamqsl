<template>
  <div>
    <el-card>
      <template #header>
        <div style="display:flex;justify-content:space-between;align-items:center;">
          <span>卡片记录</span>
          <div>
            <el-button type="success" @click="showFromQso">通联换卡</el-button>
            <el-button type="primary" @click="showCreateCard">创建卡片</el-button>
          </div>
        </div>
      </template>

      <el-form :inline="true" style="margin-bottom:16px;">
        <el-form-item label="呼号">
          <el-input v-model="query.call_sign" placeholder="对方呼号" clearable style="width:160px" @clear="loadData" />
        </el-form-item>
        <el-form-item label="场景">
          <el-select v-model="query.scene_type" placeholder="全部" clearable style="width:120px" @change="loadData">
            <el-option label="QSO" value="QSO" /><el-option label="SWL" value="SWL" /><el-option label="EYEBALL" value="EYEBALL" />
          </el-select>
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="query.flow_status" placeholder="全部" clearable style="width:120px" @change="loadData">
            <el-option label="待制卡" value="PENDING_ISSUE" /><el-option label="已制卡" value="ISSUED" />
            <el-option label="已打包" value="PACKED" /><el-option label="已发卡" value="SENT" />
            <el-option label="对方已收卡" value="RECEIVED" /><el-option label="已签收" value="SIGNED" />
          </el-select>
        </el-form-item>
        <el-form-item label="邮寄方式">
          <el-select v-model="query.mail_type" placeholder="全部" clearable style="width:120px" @change="loadData">
            <el-option label="挂号信" value="REGISTERED" /><el-option label="平信" value="ORDINARY" />
          </el-select>
        </el-form-item>
        <el-form-item label="回寄状态">
          <el-select v-model="query.return_status" placeholder="全部" clearable style="width:130px" @change="loadData">
            <el-option label="回寄已开启" value="enabled" />
            <el-option label="对方已寄出" value="mailed" />
            <el-option label="回寄已收" value="received" />
          </el-select>
        </el-form-item>
        <el-form-item><el-button type="primary" @click="loadData">查询</el-button></el-form-item>
      </el-form>

      <el-table :data="list" border stripe v-loading="loading">
        <el-table-column prop="card_code" label="编号" width="90" />
        <el-table-column prop="call_sign" label="呼号" width="100" />
        <el-table-column prop="owner_name" label="姓名" width="90">
          <template #default="{ row }">{{ row.owner_name || '-' }}</template>
        </el-table-column>
        <el-table-column prop="card_type" label="类型" width="70" />
        <el-table-column prop="scene_type" label="场景" width="90" />
        <el-table-column label="状态" width="80" align="center">
          <template #default="{ row }">
            <el-tag :type="statusType(row.flow_status)" size="small">{{ statusText(row.flow_status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="邮寄方式" width="90" align="center">
          <template #default="{ row }">
            <el-tag :type="row.mail_type === 'REGISTERED' ? '' : 'info'" size="small">
              {{ row.mail_type === 'REGISTERED' ? '挂号' : '平信' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="挂号信号码" width="140">
          <template #default="{ row }">
            <span v-if="row.tracking_number" style="font-size:12px;">{{ row.tracking_number }}</span>
            <span v-else style="color:#c0c4cc;">-</span>
          </template>
        </el-table-column>
        <el-table-column label="快递状态" width="120">
          <template #default="{ row }">
            <el-tag v-if="row.tracking_status" :type="trackingTagType(row.tracking_status)" size="small" effect="plain">
              {{ row.tracking_status.substring(0, 10) }}
            </el-tag>
            <span v-else style="color:#c0c4cc;">-</span>
          </template>
        </el-table-column>
        <el-table-column label="邮件" width="70" align="center">
          <template #default="{ row }">
            <el-tooltip v-if="row.mail_target_email" :content="row.mail_target_email" placement="top">
              <el-icon color="#67C23A"><Message /></el-icon>
            </el-tooltip>
            <el-icon v-else color="#c0c4cc"><Message /></el-icon>
          </template>
        </el-table-column>
        <el-table-column label="对方回寄" width="96" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.return_received_at" type="success" size="small">已收 {{ row.return_record_code }}</el-tag>
            <el-tooltip v-else-if="row.return_mailed_at" placement="top"
              :content="(row.return_mail_type === 'REGISTERED' ? '挂号信 ' : '平信') + (row.return_tracking ? ' ' + row.return_tracking : '') + ' · ' + row.return_mailed_at + ' · 点击查件'">
              <el-tag type="warning" size="small" style="cursor:pointer;" @click="openReturnTracking(row)">已回寄</el-tag>
            </el-tooltip>
            <el-tooltip v-else-if="row.return_mail_enabled" content="回寄已开启，等待对方确认收件并回寄" placement="top">
              <el-tag type="info" size="small" effect="plain">回寄中</el-tag>
            </el-tooltip>
            <span v-else style="color:#c0c4cc;">-</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="260" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" size="small" @click="showCardDetail(row)">详情</el-button>
            <el-button link type="success" size="small" v-if="!row.card_issued" @click="handleIssue(row)">制卡</el-button>
            <el-button link type="warning" size="small" v-if="row.card_issued && !row.card_sent" @click="showSentDialog(row)">发信</el-button>
            <el-button link type="info" size="small" v-if="row.card_sent && !row.card_received" @click="showReceivedDialog(row)">对方收卡</el-button>
            <el-button link size="small" v-if="row.mail_type === 'REGISTERED' && row.tracking_number" @click="openTracking(row)">快递</el-button>
            <el-button link type="info" size="small" @click="showQRCode(row)">二维码</el-button>
            <el-dropdown v-if="row.mail_target_email" trigger="click" @command="(cmd) => handleSendMail(row, cmd)">
              <el-button link type="info" size="small">邮件<el-icon class="el-icon--right"><ArrowDown /></el-icon></el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="created">建卡通知</el-dropdown-item>
                  <el-dropdown-item command="sent">发信通知</el-dropdown-item>
                  <el-dropdown-item command="received">收卡通知</el-dropdown-item>
                  <el-dropdown-item command="tracking">快递更新</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </template>
        </el-table-column>
      </el-table>

      <el-pagination style="margin-top:16px;justify-content:flex-end;" v-model:current-page="query.page"
        v-model:page-size="query.size" :total="total" :page-sizes="[20,50,100]" layout="total, sizes, prev, pager, next"
        @size-change="loadData" @current-change="loadData" />
    </el-card>

    <!-- 创建卡片对话框 -->
    <el-dialog v-model="createDialog" title="创建卡片" width="550px" destroy-on-close>
      <el-form :model="createForm" label-width="100px">
        <el-form-item label="对方呼号" required><el-input v-model="createForm.call_sign" /></el-form-item>
        <el-form-item label="对方姓名"><el-input v-model="createForm.owner_name" placeholder="邮件称呼会带上姓名" /></el-form-item>
        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="卡片类型">
              <el-select v-model="createForm.card_type">
                <el-option label="QSO" value="QSO" /><el-option label="SWL" value="SWL" /><el-option label="EYEBALL" value="EYEBALL" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="场景">
              <el-select v-model="createForm.scene_type">
                <el-option label="QSO" value="QSO" /><el-option label="SWL" value="SWL" /><el-option label="EYEBALL" value="EYEBALL" />
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="卡片版本">
          <el-select v-model="createForm.card_version" clearable>
            <el-option v-for="v in versions" :key="v.card_version" :label="v.card_version" :value="v.card_version" />
          </el-select>
        </el-form-item>
        <el-form-item label="邮寄方式">
          <el-radio-group v-model="createForm.mail_type">
            <el-radio value="REGISTERED">挂号信</el-radio>
            <el-radio value="ORDINARY">平信</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="收件邮箱">
          <el-input v-model="createForm.mail_target_email" placeholder="用于发送状态通知" />
        </el-form-item>
        <el-form-item label="业务备注">
          <el-input v-model="createForm.business_remarks" type="textarea" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createDialog = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="handleCreateCard">创建</el-button>
      </template>
    </el-dialog>

    <!-- 发信确认对话框 -->
    <el-dialog v-model="sentDialog" title="发信确认" width="500px" destroy-on-close>
      <el-form :model="sentForm" label-width="100px">
        <el-form-item label="邮寄方式">
          <el-radio-group v-model="sentForm.mail_type">
            <el-radio value="REGISTERED">挂号信</el-radio>
            <el-radio value="ORDINARY">平信</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item v-if="sentForm.mail_type === 'REGISTERED'" label="挂号信号码">
          <el-input v-model="sentForm.tracking_number" placeholder="输入挂号信/快递单号" />
        </el-form-item>
        <el-form-item v-if="sentForm.mail_type === 'REGISTERED'" label="承运商">
          <el-select v-model="sentForm.tracking_carrier">
            <el-option label="中国邮政" value="CHINA_POST" /><el-option label="EMS" value="EMS" />
            <el-option label="顺丰" value="SF" /><el-option label="圆通" value="YT" />
            <el-option label="中通" value="ZTO" /><el-option label="申通" value="STO" />
            <el-option label="韵达" value="YUNDA" /><el-option label="京东" value="JD" />
            <el-option label="其他" value="OTHER" />
          </el-select>
        </el-form-item>
        <el-form-item label="发信备注">
          <el-input v-model="sentForm.sent_remarks" type="textarea" />
        </el-form-item>
        <el-alert v-if="sentForm.mail_target_email" type="info" :closable="false" style="margin-bottom:12px;">
          发信后将自动发送邮件通知到: {{ sentForm.mail_target_email }}
        </el-alert>
      </el-form>
      <template #footer>
        <el-button @click="sentDialog = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="handleSent">确认发信</el-button>
      </template>
    </el-dialog>

    <!-- 对方收卡确认对话框（发出去的卡由对方签收） -->
    <el-dialog v-model="receivedDialog" title="对方收卡确认" width="400px" destroy-on-close>
      <el-form :model="receivedForm" label-width="90px">
        <el-form-item label="收到日期" required>
          <el-date-picker v-model="receivedForm.received_date" type="date" value-format="YYYY-MM-DD" style="width:100%" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="receivedForm.received_remarks" type="textarea" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="receivedDialog = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="handleReceived">确认对方已收卡</el-button>
      </template>
    </el-dialog>

    <!-- 通联换卡对话框：从通联日志直接建卡（主动寄卡场景） -->
    <el-dialog v-model="fromQsoDialog" title="通联换卡" width="560px" destroy-on-close>
      <el-alert type="info" :closable="false" style="margin-bottom:14px;">
        从通联日志中选择一条尚未建卡的 QSO 记录直接创建卡片，方便管理您主动寄出的卡片。
      </el-alert>
      <el-form label-width="100px">
        <el-form-item label="通联记录" required>
          <el-select v-model="fromQso.qso_record_id" filterable placeholder="按呼号/编号/日期搜索" style="width:100%">
            <el-option v-for="q in qsoOptions" :key="q.id" :value="q.id"
              :label="q.record_code + ' · ' + q.call_sign + ' · ' + q.date + (q.band ? ' · ' + q.band : '')" />
          </el-select>
        </el-form-item>
        <el-form-item label="卡片版本">
          <el-select v-model="fromQso.card_version" clearable placeholder="选择版本（可选）" style="width:100%">
            <el-option v-for="v in versions" :key="v.card_version" :label="v.card_version" :value="v.card_version" />
          </el-select>
        </el-form-item>
        <el-form-item label="邮寄方式">
          <el-radio-group v-model="fromQso.mail_type">
            <el-radio value="REGISTERED">挂号信</el-radio>
            <el-radio value="ORDINARY">平信</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="收件邮箱">
          <el-input v-model="fromQso.mail_target_email" placeholder="对方邮箱，用于状态通知（可选）" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="fromQso.card_remarks" type="textarea" :rows="2" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="fromQsoDialog = false">取消</el-button>
        <el-button type="primary" :loading="fromQsoSaving" @click="handleFromQso">创建卡片</el-button>
      </template>
    </el-dialog>

    <!-- 详情对话框 -->
    <el-dialog v-model="detailDialog" title="卡片详情" width="600px" destroy-on-close>
      <el-descriptions v-if="detailCard" :column="2" border>
        <el-descriptions-item label="卡片编号">{{ detailCard.card_code }}</el-descriptions-item>
        <el-descriptions-item label="呼号">{{ detailCard.call_sign }}</el-descriptions-item>
        <el-descriptions-item label="对方姓名" :span="2">{{ detailCard.owner_name || '-' }}</el-descriptions-item>
        <el-descriptions-item label="类型">{{ detailCard.card_type }}</el-descriptions-item>
        <el-descriptions-item label="场景">{{ detailCard.scene_type }}</el-descriptions-item>
        <el-descriptions-item label="版本">{{ detailCard.card_version || '-' }}</el-descriptions-item>
        <el-descriptions-item label="建卡日期">{{ detailCard.card_date }}</el-descriptions-item>
        <el-descriptions-item label="状态"><el-tag :type="statusType(detailCard.flow_status)">{{ statusText(detailCard.flow_status) }}</el-tag></el-descriptions-item>
        <el-descriptions-item label="邮寄方式">
          <el-tag :type="detailCard.mail_type === 'REGISTERED' ? '' : 'info'">{{ detailCard.mail_type === 'REGISTERED' ? '挂号信' : '平信' }}</el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="挂号信号码" :span="2">
          {{ detailCard.tracking_number || '-' }}
          <el-button v-if="detailCard.tracking_number" link type="primary" @click="openTracking(detailCard)">在快递100 查询</el-button>
        </el-descriptions-item>
        <el-descriptions-item label="快递状态" :span="2">{{ detailCard.tracking_status || '-' }}</el-descriptions-item>
        <el-descriptions-item label="收件邮箱" :span="2">{{ detailCard.mail_target_email || '-' }}</el-descriptions-item>
        <el-descriptions-item label="制卡">{{ detailCard.card_issued ? '是 ' + detailCard.card_issued_at : '否' }}</el-descriptions-item>
        <el-descriptions-item label="发信">{{ detailCard.card_sent ? '是 ' + detailCard.sent_at : '否' }}</el-descriptions-item>
        <el-descriptions-item label="对方收卡">{{ detailCard.card_received ? '是 ' + detailCard.received_at : '否' }}</el-descriptions-item>
        <el-descriptions-item label="签收">{{ detailCard.receipt_confirmed ? '是' : '否' }}</el-descriptions-item>
        <el-descriptions-item label="对方回寄" :span="2">
          <template v-if="detailCard.return_mailed_at">
            {{ detailCard.return_mail_type === 'REGISTERED' ? '挂号信 ' + (detailCard.return_tracking || '') : '平信' }}
            <span style="color:#909399;">（登记于 {{ detailCard.return_mailed_at }}）</span>
            <el-tag v-if="detailCard.return_received_at" type="success" size="small" style="margin-left:8px;">已收 {{ detailCard.return_record_code }}</el-tag>
            <el-button v-if="detailCard.return_tracking" link type="primary" size="small" @click="openReturnTracking(detailCard)">在快递100 查询</el-button>
          </template>
          <template v-else>—</template>
        </el-descriptions-item>
        <el-descriptions-item label="业务备注" :span="2">{{ detailCard.business_remarks || '-' }}</el-descriptions-item>
        <el-descriptions-item label="卡片备注" :span="2">{{ detailCard.card_remarks || '-' }}</el-descriptions-item>
      </el-descriptions>

      <!-- 回寄管理：开关 + 回寄地址勾选 + 确认收到回寄 -->
      <div class="return-admin">
        <div class="return-admin-row">
          <span class="ra-label">回寄功能</span>
          <el-switch v-model="detailCard.return_mail_enabled" @change="toggleReturnMail" />
          <span class="ra-hint">开启后，对方确认收件成功页会显示「回寄您的卡片」引导（获取回寄地址并登记回寄单号）</span>
        </div>
        <div class="return-admin-row">
          <span class="ra-label">回寄地址</span>
          <el-select v-model="returnAddressId" placeholder="选择回寄地址来源" style="min-width:260px;" @change="saveReturnAddress" :loading="addrLoading">
            <el-option label="台站信息（设置→台站信息）" value="" />
            <el-option v-for="a in addressBook" :key="a.id" :value="String(a.id)" :label="addressLabel(a)" />
          </el-select>
          <span class="ra-hint">勾选后对方看到的回寄地址以此为准；没有合适的请先到「地址簿」添加</span>
        </div>
        <div v-if="detailCard.return_mailed_at && !detailCard.return_received_at" class="return-admin-row">
          <el-button type="success" @click="confirmReturnReceived">确认收到回寄（自动记入收卡记录）</el-button>
        </div>
      </div>

      <template #footer>
        <el-button @click="detailDialog = false">关闭</el-button>
      </template>
    </el-dialog>

    <!-- \u6536\u5361\u786e\u8ba4\u4e8c\u7ef4\u7801\u5bf9\u8bdd\u6846 -->
    <el-dialog v-model="qrDialog" title="收卡确认二维码" width="400px" align-center @opened="renderQR">
      <div v-if="qrCardInfo" style="text-align:center;">
        <el-descriptions :column="1" border size="small" style="margin-bottom:20px;">
          <el-descriptions-item label="卡片编号">{{ qrCardInfo.card_code }}</el-descriptions-item>
          <el-descriptions-item label="对方呼号">{{ qrCardInfo.call_sign }}</el-descriptions-item>
          <el-descriptions-item label="卡片版本">{{ qrCardInfo.card_version || '—' }}</el-descriptions-item>
        </el-descriptions>
        <div style="display:inline-block;padding:16px;background:#fff;border:1px solid #e0e0e0;border-radius:8px;">
          <canvas ref="qrCanvasRef"></canvas>
        </div>
        <p style="margin-top:12px;color:#909399;font-size:13px;">对方扫码后可确认收卡；二维码需随卡装入信封，请在发信前打印。</p>
        <div style="margin-top:12px;">
          <el-button type="primary" plain size="small" @click="copyConfirmLink">复制确认链接</el-button>
          <el-button size="small" @click="printQR">打印</el-button>
          <el-button type="success" size="small" @click="exportQRImage">导出图片</el-button>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import api from '../api'
import { ElMessage, ElMessageBox } from 'element-plus'
import QRCode from 'qrcode'

const loading = ref(false)
const saving = ref(false)
const list = ref([])
const total = ref(0)
const versions = ref([])
const query = reactive({ call_sign: '', scene_type: '', flow_status: '', mail_type: '', return_status: '', page: 1, size: 20 })

const createDialog = ref(false)
const createForm = reactive({ call_sign: '', owner_name: '', card_type: 'QSO', scene_type: 'QSO', card_version: '', business_remarks: '', mail_type: 'REGISTERED', mail_target_email: '' })
const sentDialog = ref(false)
const sentForm = reactive({ id: null, sent_remarks: '', mail_type: 'REGISTERED', tracking_number: '', tracking_carrier: 'CHINA_POST', mail_target_email: '' })
const receivedDialog = ref(false)
const receivedForm = reactive({ id: null, received_date: '', received_remarks: '' })
const detailDialog = ref(false)
const detailCard = ref(null)
const qrDialog = ref(false)
const qrCardInfo = ref(null)
const qrCanvasRef = ref(null)

const statusMap = { PENDING_ISSUE: '待制卡', ISSUED: '已制卡', PACKED: '已打包', SENT: '已发卡', RECEIVED: '对方已收卡', SIGNED: '已签收', ERROR: '异常' }
const statusText = s => statusMap[s] || s
const statusType = s => ({ PENDING_ISSUE: 'info', ISSUED: 'success', SENT: 'warning', RECEIVED: '', SIGNED: 'success', ERROR: 'danger' }[s] || 'info')
const carrierMap = { CHINA_POST: '中国邮政', EMS: 'EMS', SF: '顺丰', YT: '圆通', ZTO: '中通', STO: '申通', YUNDA: '韵达', JD: '京东', OTHER: '其他' }
const carrierText = c => carrierMap[c] || c
const trackingTagType = s => {
  if (!s) return 'info'
  if (s.includes('签收') || s.includes('妥投')) return 'success'
  if (s.includes('派件') || s.includes('投递')) return 'warning'
  if (s.includes('运输') || s.includes('中转')) return ''
  return 'info'
}

async function loadData() {
  loading.value = true
  try {
    const res = await api.get('/card-records', { params: query })
    list.value = res.items
    total.value = res.total
  } finally { loading.value = false }
}

async function loadVersions() {
  const res = await api.get('/station/cards')
  versions.value = res || []
}

function showCreateCard() {
  Object.assign(createForm, { call_sign: '', owner_name: '', card_type: 'QSO', scene_type: 'QSO', card_version: '', business_remarks: '', mail_type: 'REGISTERED', mail_target_email: '' })
  createDialog.value = true
}

async function handleCreateCard() {
  if (!createForm.call_sign) { ElMessage.warning('呼号必填'); return }
  saving.value = true
  try {
    await api.post('/card-records', createForm)
    ElMessage.success('创建成功')
    createDialog.value = false
    loadData()
  } finally { saving.value = false }
}

async function handleIssue(row) {
  await api.post('/card-records/' + row.id + '/issue')
  ElMessage.success('制卡成功')
  loadData()
}

function showSentDialog(row) {
  sentForm.id = row.id
  sentForm.sent_remarks = ''
  sentForm.mail_type = row.mail_type || 'REGISTERED'
  sentForm.tracking_number = row.tracking_number || ''
  sentForm.tracking_carrier = row.tracking_carrier || 'CHINA_POST'
  sentForm.mail_target_email = row.mail_target_email || ''
  sentDialog.value = true
}

async function handleSent() {
  saving.value = true
  try {
    await api.post('/card-records/' + sentForm.id + '/sent', sentForm)
    ElMessage.success('发信确认成功')
    sentDialog.value = false
    loadData()
  } finally { saving.value = false }
}

function showReceivedDialog(row) {
  receivedForm.id = row.id
  receivedForm.received_date = ''
  receivedForm.received_remarks = ''
  receivedDialog.value = true
}

async function handleReceived() {
  if (!receivedForm.received_date) { ElMessage.warning('收卡日期必填'); return }
  saving.value = true
  try {
    await api.post('/card-records/' + receivedForm.id + '/received', receivedForm)
    ElMessage.success('对方收卡确认成功')
    receivedDialog.value = false
    loadData()
  } finally { saving.value = false }
}

// 快递查询统一跳转快递100：单号自动识别承运商，无需选择
function openTracking(row) {
  if (!row.tracking_number) { ElMessage.warning('该卡片暂无单号'); return }
  window.open('https://www.kuaidi100.com/chaxun?nu=' + encodeURIComponent(row.tracking_number), '_blank')
}

// 对方回寄单号一键查件（列表「已回寄」tag 与详情按钮共用）
function openReturnTracking(row) {
  if (!row.return_tracking) { ElMessage.warning('对方未登记回寄单号'); return }
  window.open('https://www.kuaidi100.com/chaxun?nu=' + encodeURIComponent(row.return_tracking), '_blank')
}

// ---- 通联换卡：从通联记录直接建卡 ----
const fromQsoDialog = ref(false)
const fromQsoSaving = ref(false)
const qsoOptions = ref([])
const fromQso = reactive({ qso_record_id: null, card_version: '', mail_type: 'REGISTERED', mail_target_email: '', card_remarks: '' })

async function showFromQso() {
  try {
    const res = await api.get('/qso-records', { params: { page: 1, size: 100, scene_type: 'QSO' } })
    qsoOptions.value = (res.items || []).filter(q => !q.has_card)
  } catch (e) { /* 拦截器已提示 */ }
  fromQso.qso_record_id = null
  fromQso.card_version = ''
  fromQso.mail_type = 'REGISTERED'
  fromQso.mail_target_email = ''
  fromQso.card_remarks = ''
  fromQsoDialog.value = true
  if (qsoOptions.value.length === 0) ElMessage.info('暂无未建卡的通联记录，请先在通联日志中新增')
}

async function handleFromQso() {
  if (!fromQso.qso_record_id) { ElMessage.warning('请选择通联记录'); return }
  fromQsoSaving.value = true
  try {
    await api.post('/card-records/from-qso', fromQso)
    ElMessage.success('卡片已创建，可在列表中继续制卡流程')
    fromQsoDialog.value = false
    loadData()
  } finally { fromQsoSaving.value = false }
}

function showCardDetail(row) {
  detailCard.value = row
  detailDialog.value = true
  loadReturnAddressOptions()
}

// ---- 回寄地址来源（全局配置：地址簿条目或台站信息） ----
const addressBook = ref([])
const returnAddressId = ref('')
const addrLoading = ref(false)
function addressLabel(a) {
  return [a.call_sign, a.name, a.address].filter(Boolean).join(' · ')
}
async function loadReturnAddressOptions() {
  addrLoading.value = true
  try {
    const [book, site] = await Promise.all([
      api.get('/address/book', { params: { page: 1, size: 100 } }).catch(() => null),
      api.get('/settings/site')
    ])
    addressBook.value = (book && (book.items || book)) || []
    returnAddressId.value = (site && site.return_address_id) || ''
  } catch {} finally { addrLoading.value = false }
}
async function saveReturnAddress() {
  try {
    await api.post('/settings/site', { return_address_id: returnAddressId.value || '' })
    ElMessage.success('回寄地址已更新，对后续回寄引导立即生效')
  } catch (e) { /* 拦截器已提示 */ }
}

// 回寄功能开关：开启后对方确认收件页才显示回寄引导
async function toggleReturnMail() {
  const card = detailCard.value
  if (!card) return
  try {
    await api.post('/card-records/' + card.id + '/return-toggle', { enabled: card.return_mail_enabled })
    ElMessage.success(card.return_mail_enabled ? '已开启回寄功能' : '已关闭回寄功能')
  } catch (e) {
    card.return_mail_enabled = !card.return_mail_enabled
  }
}

// 确认收到对方回寄的卡：自动写入收卡记录
async function confirmReturnReceived() {
  const card = detailCard.value
  if (!card) return
  try {
    await ElMessageBox.confirm(
      '确认已收到 ' + card.call_sign + ' 回寄的卡片？将自动写入一条收卡记录（' + (card.return_mail_type === 'REGISTERED' ? '挂号信 ' + (card.return_tracking || '') : '平信') + '），此操作不可撤销。',
      '确认收到回寄',
      { type: 'warning', confirmButtonText: '确认收卡', cancelButtonText: '取消' }
    )
  } catch (e) { return }
  try {
    const res = await api.post('/card-records/' + card.id + '/return-receive')
    card.return_received_at = res.return_received_at
    card.return_record_code = res.receive_code
    ElMessage.success('已确认收到回寄，收卡记录 ' + res.receive_code + ' 已生成')
  } catch (e) { /* 拦截器已提示 */ }
}

function showQRCode(row) {
  qrCardInfo.value = row
  qrDialog.value = true
}

function renderQR() {
  if (qrCanvasRef.value && qrCardInfo.value) {
    const url = window.location.origin + '/confirm/' + qrCardInfo.value.card_code
    QRCode.toCanvas(qrCanvasRef.value, url, {
      width: 220, margin: 2,
      color: { dark: '#1a1a1a', light: '#ffffff' }
    })
  }
}

function copyConfirmLink() {
  if (!qrCardInfo.value) return
  const url = window.location.origin + '/confirm/' + qrCardInfo.value.card_code
  navigator.clipboard.writeText(url)
  ElMessage.success('确认链接已复制')
}

function printQR() {
  if (!qrCardInfo.value) return
  const url = window.location.origin + '/confirm/' + qrCardInfo.value.card_code
  QRCode.toDataURL(url, { width: 200, margin: 2 }).then(dataUrl => {
    const win = window.open('', '', 'width=400,height=520')
    win.document.write('<html><head><title>QSL \u6536\u5361\u786e\u8ba4 - ' + qrCardInfo.value.card_code + '</title></head>' +
      '<body style="text-align:center;font-family:sans-serif;padding:30px;">' +
      '<h2>QSL \u5361\u7247\u6536\u5361\u786e\u8ba4</h2>' +
      '<p>\u5361\u7247\u7f16\u53f7: ' + qrCardInfo.value.card_code + '</p>' +
      '<p>\u5bf9\u65b9\u547c\u53f7: ' + qrCardInfo.value.call_sign + '</p>' +
      '<img src="' + dataUrl + '" style="width:200px;height:200px;" />' +
      '<p style="margin-top:16px;color:#666;">\u8bf7\u626b\u63cf\u4e0a\u65b9\u4e8c\u7ef4\u7801\u786e\u8ba4\u6536\u5361</p>' +
      '</body></html>')
    win.document.close()
    setTimeout(() => { win.print() }, 300)
  })
}

// 导出二维码 PNG：白底合成标题/编号/呼号/二维码/提示，尺寸自控（2 倍清晰度）
async function exportQRImage() {
  if (!qrCardInfo.value) return
  const info = qrCardInfo.value
  const url = window.location.origin + '/confirm/' + info.card_code
  const dataUrl = await QRCode.toDataURL(url, {
    width: 400, margin: 2,
    color: { dark: '#1a1a1a', light: '#ffffff' }
  })
  const img = new Image()
  await new Promise((resolve, reject) => { img.onload = resolve; img.onerror = reject; img.src = dataUrl })

  const w = 480, h = 640, scale = 2
  const canvas = document.createElement('canvas')
  canvas.width = w * scale
  canvas.height = h * scale
  const ctx = canvas.getContext('2d')
  ctx.scale(scale, scale)
  ctx.fillStyle = '#ffffff'
  ctx.fillRect(0, 0, w, h)
  ctx.textAlign = 'center'
  ctx.fillStyle = '#1a1a1a'
  ctx.font = 'bold 24px sans-serif'
  ctx.fillText('QSL 卡片收卡确认', w / 2, 56)
  ctx.font = '17px sans-serif'
  ctx.fillText('卡片编号: ' + info.card_code, w / 2, 96)
  ctx.fillText('对方呼号: ' + (info.call_sign || '—'), w / 2, 124)
  ctx.drawImage(img, (w - 400) / 2, 148, 400, 400)
  ctx.fillStyle = '#666666'
  ctx.font = '15px sans-serif'
  ctx.fillText('请扫描上方二维码确认收卡', w / 2, 588)
  ctx.fillText(window.location.host + '/confirm/' + info.card_code, w / 2, 614)

  const a = document.createElement('a')
  a.href = canvas.toDataURL('image/png')
  a.download = 'qsl-confirm-' + info.card_code + '.png'
  a.click()
  ElMessage.success('二维码图片已导出')
}

async function handleSendMail(row, scene) {
  try {
    const res = await api.post('/card-records/' + row.id + '/send-mail', { scene, card_record_id: row.id })
    ElMessage.success(res || '邮件发送成功')
  } catch {}
}

onMounted(() => { loadData(); loadVersions() })
</script>

<style scoped>
.return-admin { margin-top: 16px; padding: 12px 14px; background: #faf7f0; border: 1px dashed var(--qsl-line); }
.return-admin-row { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; }
.return-admin-row + .return-admin-row { margin-top: 10px; }
.ra-label { font-weight: 600; color: var(--qsl-navy); }
.ra-hint { color: #909399; font-size: 12px; line-height: 1.6; }
</style>
