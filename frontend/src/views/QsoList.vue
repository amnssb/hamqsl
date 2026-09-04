<template>
  <div>
    <el-card>
      <template #header>
        <div style="display:flex;justify-content:space-between;align-items:center;">
          <span>通联日志</span>
          <div>
            <el-button type="success" @click="importDialog = true">导入 ADIF</el-button>
            <el-button type="primary" @click="showAdd">新增通联</el-button>
          </div>
        </div>
      </template>

      <el-form :inline="true" style="margin-bottom:16px;">
        <el-form-item label="呼号">
          <el-input v-model="query.call_sign" placeholder="对方呼号" clearable @clear="loadData" />
        </el-form-item>
        <el-form-item label="场景">
          <el-select v-model="query.scene_type" placeholder="全部" clearable @change="loadData">
            <el-option label="QSO" value="QSO" />
            <el-option label="SWL" value="SWL" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="loadData">查询</el-button>
        </el-form-item>
      </el-form>

      <el-table :data="list" border stripe v-loading="loading">
        <el-table-column prop="record_code" label="编号" width="120" />
        <el-table-column prop="date" label="日期" width="110" />
        <el-table-column prop="time" label="时间" width="80" />
        <el-table-column prop="call_sign" label="对方呼号" width="120" />
        <el-table-column prop="freq" label="频率" width="100" />
        <el-table-column prop="band" label="频段" width="80" />
        <el-table-column prop="mode" label="模式" width="80" />
        <el-table-column prop="rst_sent" label="RST发送" width="90" />
        <el-table-column prop="rst_rcvd" label="RST接收" width="90" />
        <el-table-column prop="qth" label="对方QTH" min-width="120" show-overflow-tooltip />
        <el-table-column label="建卡" width="70" align="center">
          <template #default="{ row }">
            <el-tag :type="row.has_card ? 'success' : 'info'" size="small">{{ row.has_card ? '是' : '否' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="120" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" size="small" @click="showEdit(row)">编辑</el-button>
            <el-popconfirm title="确定删除？" @confirm="handleDelete(row.id)">
              <template #reference>
                <el-button link type="danger" size="small" :disabled="row.has_card">删除</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>

      <el-pagination style="margin-top:16px;justify-content:flex-end;" v-model:current-page="query.page"
        v-model:page-size="query.size" :total="total" :page-sizes="[20,50,100]" layout="total, sizes, prev, pager, next"
        @size-change="loadData" @current-change="loadData" />
    </el-card>

    <!-- 新增/编辑对话框 -->
    <el-dialog v-model="dialogVisible" :title="isEdit ? '编辑通联' : '新增通联'" width="700px" destroy-on-close>
      <el-form :model="form" label-width="100px">
        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="场景类型">
              <el-select v-model="form.scene_type">
                <el-option label="QSO" value="QSO" />
                <el-option label="SWL" value="SWL" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="对方呼号" required>
              <el-input v-model="form.call_sign" placeholder="如 BI1KBU" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="通联日期" required>
              <el-date-picker v-model="form.date" type="date" value-format="YYYY-MM-DD" style="width:100%" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="通联时间">
              <el-input v-model="form.time" placeholder="HH:mm" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="16">
          <el-col :span="8"><el-form-item label="频率"><el-input v-model="form.freq" /></el-form-item></el-col>
          <el-col :span="8"><el-form-item label="频段"><el-input v-model="form.band" /></el-form-item></el-col>
          <el-col :span="8"><el-form-item label="模式"><el-input v-model="form.mode" /></el-form-item></el-col>
        </el-row>
        <el-row :gutter="16">
          <el-col :span="8"><el-form-item label="RST发送"><el-input v-model="form.rst_sent" /></el-form-item></el-col>
          <el-col :span="8"><el-form-item label="RST接收"><el-input v-model="form.rst_rcvd" /></el-form-item></el-col>
          <el-col :span="8"><el-form-item label="操作员"><el-input v-model="form.operator" /></el-form-item></el-col>
        </el-row>
        <el-row :gutter="16">
          <el-col :span="12"><el-form-item label="本台设备"><el-input v-model="form.my_rig" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="对方设备"><el-input v-model="form.rig" /></el-form-item></el-col>
        </el-row>
        <el-row :gutter="16">
          <el-col :span="12"><el-form-item label="本台QTH"><el-input v-model="form.my_qth" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="对方QTH"><el-input v-model="form.qth" placeholder="网格或地址" /></el-form-item></el-col>
        </el-row>
        <el-form-item label="备注">
          <el-input v-model="form.remarks" type="textarea" :rows="2" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="handleSave">保存</el-button>
      </template>
    </el-dialog>

    <!-- ADIF 导入对话框 -->
    <el-dialog v-model="importDialog" title="导入 ADIF 通联日志" width="480px" destroy-on-close>
      <el-alert type="info" :closable="false" style="margin-bottom:16px;">
        支持标准 ADIF（.adi / .adif）文件，5MB 以内；按「呼号 + 日期 + 频率 + 模式」自动去重，重复记录将跳过。
      </el-alert>
      <el-upload drag accept=".adi,.adif,.txt" :show-file-list="false" :http-request="handleImport">
        <el-icon :size="40" style="color:#c0c4cc;"><UploadFilled /></el-icon>
        <div style="margin-top:8px;color:#666;">点击或拖拽 ADIF 文件到此处</div>
      </el-upload>
      <p v-if="importResult" style="margin-top:12px;color:#666;">
        导入成功 <b>{{ importResult.imported }}</b> 条，跳过 <b>{{ importResult.skipped }}</b> 条（重复或缺少呼号/日期）
      </p>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import api from '../api'
import { ElMessage } from 'element-plus'

const loading = ref(false)
const saving = ref(false)
const list = ref([])
const total = ref(0)
const dialogVisible = ref(false)
const isEdit = ref(false)
const editId = ref(null)

const query = reactive({ call_sign: '', scene_type: '', page: 1, size: 20 })
const defaultForm = {
  scene_type: 'QSO', date: '', time: '', timezone: 'UTC+8', freq: '', band: '', mode: '',
  my_rig: '', my_rig_mode: '', my_rig_ant: '', my_rig_pwr: '', my_qth: '', operator: '',
  call_sign: '', rig: '', ant: '', pwr: '', qth: '', rst_sent: '', rst_rcvd: '', remarks: '',
}
const form = reactive({ ...defaultForm })

async function loadData() {
  loading.value = true
  try {
    const res = await api.get('/qso-records', { params: query })
    list.value = res.items
    total.value = res.total
  } finally { loading.value = false }
}

function showAdd() {
  isEdit.value = false
  editId.value = null
  Object.assign(form, defaultForm)
  dialogVisible.value = true
}

function showEdit(row) {
  isEdit.value = true
  editId.value = row.id
  Object.assign(form, row)
  dialogVisible.value = true
}

async function handleSave() {
  if (!form.call_sign || !form.date) {
    ElMessage.warning('呼号和日期必填')
    return
  }
  saving.value = true
  try {
    if (isEdit.value) {
      await api.put(`/qso-records/${editId.value}`, form)
    } else {
      await api.post('/qso-records', form)
    }
    ElMessage.success('保存成功')
    dialogVisible.value = false
    loadData()
  } finally { saving.value = false }
}

async function handleDelete(id) {
  await api.delete(`/qso-records/${id}`)
  ElMessage.success('删除成功')
  loadData()
}

// ---- ADIF 导入 ----
const importDialog = ref(false)
const importResult = ref(null)
async function handleImport({ file }) {
  const fd = new FormData()
  fd.append('file', file)
  try {
    const res = await api.post('/qso-records/import', fd)
    importResult.value = res
    ElMessage.success('导入完成：新增 ' + res.imported + ' 条，跳过 ' + res.skipped + ' 条')
    loadData()
  } catch (e) { /* 拦截器已提示 */ }
}

onMounted(loadData)
</script>
