<template>
  <div>
    <el-card>
      <template #header>
        <div style="display:flex;justify-content:space-between;align-items:center;">
          <span>卡片局管理</span>
          <el-button type="primary" @click="showAdd">新增卡片局</el-button>
        </div>
      </template>

      <el-table :data="list" border stripe v-loading="loading">
        <el-table-column prop="bureau_name" label="卡片局名称" width="150" />
        <el-table-column prop="telephone" label="电话" width="120" />
        <el-table-column prop="postal_code" label="邮编" width="80" />
        <el-table-column prop="destination_country" label="去向国" width="80" />
        <el-table-column prop="address" label="地址" min-width="200" show-overflow-tooltip />
        <el-table-column label="操作" width="120">
          <template #default="{ row }">
            <el-button link type="primary" size="small" @click="showEdit(row)">编辑</el-button>
            <el-popconfirm title="确定删除？" @confirm="handleDelete(row.id)">
              <template #reference><el-button link type="danger" size="small">删除</el-button></template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog v-model="dialogVisible" :title="isEdit ? '编辑卡片局' : '新增卡片局'" width="600px" destroy-on-close>
      <el-form :model="form" label-width="100px">
        <el-form-item label="卡片局名称" required><el-input v-model="form.bureau_name" /></el-form-item>
        <el-row :gutter="16">
          <el-col :span="12"><el-form-item label="电话"><el-input v-model="form.telephone" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="邮编"><el-input v-model="form.postal_code" /></el-form-item></el-col>
        </el-row>
        <el-form-item label="去向国"><el-input v-model="form.destination_country" /></el-form-item>
        <el-form-item label="地址"><el-input v-model="form.address" type="textarea" /></el-form-item>
        <el-form-item label="备注"><el-input v-model="form.address_remarks" type="textarea" /></el-form-item>
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
const dialogVisible = ref(false)
const isEdit = ref(false)
const editId = ref(null)
const defaultForm = { bureau_name: '', telephone: '', postal_code: '', destination_country: '', address: '', address_remarks: '' }
const form = reactive({ ...defaultForm })

async function loadData() {
  loading.value = true
  try { const res = await api.get('/address/bureaus'); list.value = res || [] }
  finally { loading.value = false }
}

function showAdd() { isEdit.value = false; editId.value = null; Object.assign(form, defaultForm); dialogVisible.value = true }
function showEdit(row) { isEdit.value = true; editId.value = row.id; Object.assign(form, row); dialogVisible.value = true }

async function handleSave() {
  if (!form.bureau_name) { ElMessage.warning('名称必填'); return }
  saving.value = true
  try {
    if (isEdit.value) await api.put(`/address/bureaus/${editId.value}`, form)
    else await api.post('/address/bureaus', form)
    ElMessage.success('保存成功'); dialogVisible.value = false; loadData()
  } finally { saving.value = false }
}

async function handleDelete(id) { await api.delete(`/address/bureaus/${id}`); ElMessage.success('删除成功'); loadData() }
onMounted(loadData)
</script>
