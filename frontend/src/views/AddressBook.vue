<template>
  <div>
    <el-card>
      <template #header>
        <div style="display:flex;justify-content:space-between;align-items:center;">
          <span>我的地址</span>
          <el-button type="primary" @click="showAdd">新增地址</el-button>
        </div>
      </template>

      <el-alert type="info" :closable="false" style="margin-bottom:16px;">
        这里维护您自己的回寄地址（可添加多条）。在线上换卡的 SWL 申请中，可多选地址发送到对方邮箱，对方寄出后在进度页登记单号即可留档。
      </el-alert>

      <el-table :data="list" border stripe v-loading="loading">
        <el-table-column prop="name" label="地址名称" width="140" />
        <el-table-column prop="address" label="详细地址" min-width="220" show-overflow-tooltip />
        <el-table-column prop="postal_code" label="邮编" width="90" />
        <el-table-column prop="destination_country" label="国家/地区" width="100" />
        <el-table-column prop="telephone" label="电话" width="130" />
        <el-table-column prop="email" label="邮箱" width="180" show-overflow-tooltip />
        <el-table-column label="操作" width="120" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" size="small" @click="showEdit(row)">编辑</el-button>
            <el-popconfirm title="确定删除？" @confirm="handleDelete(row.id)">
              <template #reference><el-button link type="danger" size="small">删除</el-button></template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>

      <el-pagination style="margin-top:16px;justify-content:flex-end;" v-model:current-page="query.page"
        v-model:page-size="query.size" :total="total" layout="total, prev, pager, next" @current-change="loadData" />
    </el-card>

    <el-dialog v-model="dialogVisible" :title="isEdit ? '编辑地址' : '新增地址'" width="600px" destroy-on-close>
      <el-form :model="form" label-width="100px">
        <el-row :gutter="16">
          <el-col :span="12"><el-form-item label="地址名称" required><el-input v-model="form.name" placeholder="如：家庭地址 / 单位地址" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="电话"><el-input v-model="form.telephone" /></el-form-item></el-col>
        </el-row>
        <el-row :gutter="16">
          <el-col :span="12"><el-form-item label="邮编"><el-input v-model="form.postal_code" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="国家/地区"><el-input v-model="form.destination_country" placeholder="如：China" /></el-form-item></el-col>
        </el-row>
        <el-form-item label="详细地址" required><el-input v-model="form.address" type="textarea" /></el-form-item>
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
const total = ref(0)
const query = reactive({ page: 1, size: 20 })
const dialogVisible = ref(false)
const isEdit = ref(false)
const editId = ref(null)
const defaultForm = { call_sign: '', name: '', telephone: '', postal_code: '', destination_country: '', address: '', email: '', address_remarks: '' }
const form = reactive({ ...defaultForm })

async function loadData() {
  loading.value = true
  try {
    const res = await api.get('/address/book', { params: query })
    list.value = res.items
    total.value = res.total
  } finally { loading.value = false }
}

function showAdd() { isEdit.value = false; editId.value = null; Object.assign(form, defaultForm); dialogVisible.value = true }
function showEdit(row) { isEdit.value = true; editId.value = row.id; Object.assign(form, row); dialogVisible.value = true }

async function handleSave() {
  if (!form.name) { ElMessage.warning('地址名称必填'); return }
  if (!form.address) { ElMessage.warning('详细地址必填'); return }
  saving.value = true
  try {
    if (isEdit.value) await api.put('/address/book/' + editId.value, form)
    else await api.post('/address/book', form)
    ElMessage.success('保存成功'); dialogVisible.value = false; loadData()
  } finally { saving.value = false }
}

async function handleDelete(id) {
  await api.delete('/address/book/' + id)
  ElMessage.success('删除成功'); loadData()
}

onMounted(loadData)
</script>
