<template>
  <div>
    <el-card>
      <template #header>
        <div style="display:flex;justify-content:space-between;align-items:center;">
          <span>线下换卡活动</span>
          <el-button type="primary" @click="showAdd">创建活动</el-button>
        </div>
      </template>

      <el-table :data="list" border stripe v-loading="loading">
        <el-table-column prop="activity_code" label="活动编号" width="120" />
        <el-table-column prop="activity_name" label="活动名称" min-width="150" />
        <el-table-column prop="activity_location" label="地点" width="150" show-overflow-tooltip />
        <el-table-column prop="activity_date" label="日期" width="110" />
        <el-table-column prop="activity_time" label="时间" width="80" />
        <el-table-column prop="workflow_status" label="状态" width="80">
          <template #default="{ row }">
            <el-tag :type="row.workflow_status === 'ACTIVE' ? 'success' : 'info'" size="small">{{ row.workflow_status }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="80">
          <template #default="{ row }">
            <el-button link type="primary" size="small" @click="showEdit(row)">编辑</el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-pagination style="margin-top:16px;justify-content:flex-end;" v-model:current-page="page"
        v-model:page-size="size" :total="total" layout="total, prev, pager, next" @current-change="loadData" />
    </el-card>

    <el-dialog v-model="dialogVisible" :title="isEdit ? '编辑活动' : '创建活动'" width="500px" destroy-on-close>
      <el-form :model="form" label-width="100px">
        <el-form-item label="活动名称" required><el-input v-model="form.activity_name" /></el-form-item>
        <el-form-item label="活动地点"><el-input v-model="form.activity_location" /></el-form-item>
        <el-form-item label="活动日期"><el-date-picker v-model="form.activity_date" type="date" value-format="YYYY-MM-DD" style="width:100%" /></el-form-item>
        <el-form-item label="活动时间"><el-input v-model="form.activity_time" placeholder="HH:mm" /></el-form-item>
        <el-form-item label="卡片备注"><el-input v-model="form.card_remarks" type="textarea" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="handleSave">保存</el-button>
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
const page = ref(1)
const size = ref(20)
const dialogVisible = ref(false)
const isEdit = ref(false)
const editId = ref(null)
const form = reactive({ activity_name: '', activity_location: '', activity_date: '', activity_time: '', card_remarks: '' })

async function loadData() {
  loading.value = true
  try {
    const res = await api.get('/exchange/offline/activities', { params: { page: page.value, size: size.value } })
    list.value = res.items
    total.value = res.total
  } finally { loading.value = false }
}

function showAdd() {
  isEdit.value = false; editId.value = null
  Object.assign(form, { activity_name: '', activity_location: '', activity_date: '', activity_time: '', card_remarks: '' })
  dialogVisible.value = true
}

function showEdit(row) {
  isEdit.value = true; editId.value = row.id
  Object.assign(form, row)
  dialogVisible.value = true
}

async function handleSave() {
  if (!form.activity_name) { ElMessage.warning('活动名称必填'); return }
  saving.value = true
  try {
    if (isEdit.value) await api.put(`/exchange/offline/activities/${editId.value}`, form)
    else await api.post('/exchange/offline/activities', form)
    ElMessage.success('保存成功')
    dialogVisible.value = false
    loadData()
  } finally { saving.value = false }
}

onMounted(loadData)
</script>
