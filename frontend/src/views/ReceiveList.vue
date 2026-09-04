<template>
  <div>
    <el-card>
      <template #header>
        <div style="display:flex;justify-content:space-between;align-items:center;">
          <span>收卡记录</span>
          <el-button type="primary" @click="showAdd">收卡确认</el-button>
        </div>
      </template>

      <el-form :inline="true" style="margin-bottom:16px;">
        <el-form-item label="呼号">
          <el-input v-model="query.call_sign" placeholder="对方呼号" clearable @clear="loadData" />
        </el-form-item>
        <el-form-item><el-button type="primary" @click="loadData">查询</el-button></el-form-item>
      </el-form>

      <el-table :data="list" border stripe v-loading="loading">
        <el-table-column prop="receive_code" label="收卡编号" width="160" />
        <el-table-column prop="call_sign" label="对方呼号" width="110" />
        <el-table-column prop="card_type" label="卡片类型" width="90" />
        <el-table-column prop="business_type" label="业务类型" width="110" />
        <el-table-column prop="received_date" label="收卡日期" width="110" />
        <el-table-column label="匹配状态" width="90" align="center">
          <template #default="{ row }">
            <el-tag :type="row.match_status === 'MATCHED' ? 'success' : 'warning'" size="small">
              {{ row.match_status === 'MATCHED' ? '已匹配' : '未匹配' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="remarks" label="备注" min-width="150" show-overflow-tooltip />
      </el-table>

      <el-pagination style="margin-top:16px;justify-content:flex-end;" v-model:current-page="query.page"
        v-model:page-size="query.size" :total="total" layout="total, prev, pager, next" @current-change="loadData" />
    </el-card>

    <el-dialog v-model="dialogVisible" title="收卡确认" width="500px" destroy-on-close>
      <el-form :model="form" label-width="100px">
        <el-form-item label="对方呼号" required><el-input v-model="form.call_sign" /></el-form-item>
        <el-form-item label="卡片类型">
          <el-select v-model="form.card_type">
            <el-option label="QSO" value="QSO" /><el-option label="SWL" value="SWL" /><el-option label="EYEBALL" value="EYEBALL" />
          </el-select>
        </el-form-item>
        <el-form-item label="业务类型">
          <el-select v-model="form.business_type">
            <el-option label="QSO" value="QSO" /><el-option label="SWL" value="SWL" /><el-option label="EYEBALL" value="EYEBALL" />
          </el-select>
        </el-form-item>
        <el-form-item label="收卡日期" required>
          <el-date-picker v-model="form.received_date" type="date" value-format="YYYY-MM-DD" style="width:100%" />
        </el-form-item>
        <el-form-item label="关联发卡ID">
          <el-input v-model.number="form.outbound_card_id" placeholder="可选，关联发卡记录ID" />
        </el-form-item>
        <el-form-item label="备注"><el-input v-model="form.remarks" type="textarea" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="handleSave">确认收卡</el-button>
      </template>
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
const query = reactive({ call_sign: '', page: 1, size: 20 })
const dialogVisible = ref(false)
const form = reactive({ call_sign: '', card_type: 'QSO', business_type: 'QSO', received_date: '', outbound_card_id: null, remarks: '' })

async function loadData() {
  loading.value = true
  try {
    const res = await api.get('/receive-records', { params: query })
    list.value = res.items
    total.value = res.total
  } finally { loading.value = false }
}

function showAdd() {
  Object.assign(form, { call_sign: '', card_type: 'QSO', business_type: 'QSO', received_date: '', outbound_card_id: null, remarks: '' })
  dialogVisible.value = true
}

async function handleSave() {
  if (!form.call_sign || !form.received_date) { ElMessage.warning('呼号和收卡日期必填'); return }
  saving.value = true
  try {
    await api.post('/receive-records', form)
    ElMessage.success('收卡确认成功')
    dialogVisible.value = false
    loadData()
  } finally { saving.value = false }
}

onMounted(loadData)
</script>
