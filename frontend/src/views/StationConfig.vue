<template>
  <div>
    <el-card>
      <template #header>
        <div style="display:flex;justify-content:space-between;align-items:center;">
          <span>本台卡片版本</span>
          <el-button type="primary" @click="showAddCard">新增版本</el-button>
        </div>
      </template>
      <el-table :data="cards" border stripe>
        <el-table-column prop="card_version" label="版本名称" width="150" />
        <el-table-column label="图片" width="70">
          <template #default="{ row }">
            <el-image v-if="row.image_path" :src="row.image_path" fit="cover" style="width:40px;height:40px;border-radius:4px;" :preview-src-list="[row.image_path]" preview-teleported />
            <span v-else style="color:#c0c4cc;">-</span>
          </template>
        </el-table-column>
        <el-table-column prop="available_inventory" label="库存" width="80" />
        <el-table-column prop="version_total" label="总量" width="80" />
        <el-table-column prop="qso_only" label="仅QSO" width="80">
          <template #default="{ row }"><el-tag :type="row.qso_only ? 'warning' : 'info'" size="small">{{ row.qso_only ? '是' : '否' }}</el-tag></template>
        </el-table-column>
        <el-table-column prop="is_active" label="启用" width="70">
          <template #default="{ row }"><el-switch v-model="row.is_active" @change="updateCard(row)" /></template>
        </el-table-column>
        <el-table-column prop="remarks" label="备注" min-width="150" show-overflow-tooltip />
        <el-table-column label="操作" width="80">
          <template #default="{ row }"><el-button link type="primary" size="small" @click="showEditCard(row)">编辑</el-button></template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 卡片版本对话框 -->
    <el-dialog v-model="cardDialog" :title="isEditCard ? '编辑卡片版本' : '新增卡片版本'" width="500px" destroy-on-close>
      <el-form :model="cardForm" label-width="100px">
        <el-form-item label="版本名称" required><el-input v-model="cardForm.card_version" /></el-form-item>
        <el-row :gutter="16">
          <el-col :span="12"><el-form-item label="库存余量"><el-input-number v-model="cardForm.available_inventory" :min="0" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="版本总量"><el-input-number v-model="cardForm.version_total" :min="0" /></el-form-item></el-col>
        </el-row>
        <el-form-item label="排序"><el-input-number v-model="cardForm.sort_order" :min="0" /></el-form-item>
        <el-form-item label="仅限QSO"><el-switch v-model="cardForm.qso_only" /></el-form-item>
        <el-form-item label="版本图片">
          <div class="upload-row">
            <el-image v-if="cardForm.image_path" :src="cardForm.image_path" fit="cover" class="version-preview" :preview-src-list="[cardForm.image_path]" preview-teleported />
            <el-upload :show-file-list="false" accept="image/*" :http-request="handleUpload">
              <el-button size="small" type="primary" plain :loading="uploading">{{ cardForm.image_path ? '重新上传' : '上传图片' }}</el-button>
            </el-upload>
            <el-button v-if="cardForm.image_path" size="small" text type="danger" @click="cardForm.image_path = ''">移除</el-button>
          </div>
          <p class="upload-tip">将显示在公开申请页的版本选择卡片中；5MB 内 jpg/png/gif/webp</p>
        </el-form-item>
        <el-form-item label="备注"><el-input v-model="cardForm.remarks" type="textarea" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="cardDialog = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="handleSaveCard">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import api from '../api'
import { ElMessage } from 'element-plus'

const saving = ref(false)
const cards = ref([])

const cardDialog = ref(false)
const isEditCard = ref(false)
const editCardId = ref(null)
const cardDefault = { card_version: '', image_path: '', available_inventory: 0, version_total: 0, sort_order: 0, qso_only: false, remarks: '' }
const cardForm = reactive({ ...cardDefault })

async function loadCards() { const res = await api.get('/station/cards'); cards.value = res || [] }

function showAddCard() { isEditCard.value = false; editCardId.value = null; Object.assign(cardForm, cardDefault); cardDialog.value = true }
function showEditCard(row) { isEditCard.value = true; editCardId.value = row.id; Object.assign(cardForm, row); cardDialog.value = true }
async function handleSaveCard() {
  if (!cardForm.card_version) { ElMessage.warning('版本名称必填'); return }
  saving.value = true
  try {
    if (isEditCard.value) await api.put('/station/cards/' + editCardId.value, cardForm)
    else await api.post('/station/cards', cardForm)
    ElMessage.success('保存成功'); cardDialog.value = false; loadCards()
  } finally { saving.value = false }
}
async function updateCard(row) { await api.put('/station/cards/' + row.id, { is_active: row.is_active }) }

const uploading = ref(false)
async function handleUpload({ file }) {
  uploading.value = true
  try {
    const fd = new FormData()
    fd.append('file', file)
    const res = await api.post('/upload/image', fd)
    cardForm.image_path = res.url
    ElMessage.success('图片上传成功')
  } catch (e) { /* 拦截器已提示 */ } finally { uploading.value = false }
}

onMounted(() => { loadCards() })
</script>

<style scoped>
.upload-row { display: flex; align-items: center; gap: 12px; }
.version-preview { width: 64px; height: 64px; border-radius: 6px; border: 1px solid #e5e1d8; }
.upload-tip { margin: 6px 0 0; font-size: 12px; color: #999; }
</style>
