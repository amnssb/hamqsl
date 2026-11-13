<template>
  <div>
    <div class="page-heading">
      <div>
        <h1>本台资料</h1>
        <p>本台呼号和中英文名称。对外展示页只读这三项。对方回寄用的地址在「我的地址」，不在这里。</p>
      </div>
    </div>

    <el-card>
      <template #header>
        <div class="card-head">
          <span>台站档案</span>
          <el-button type="primary" :loading="saving" @click="save">保存</el-button>
        </div>
      </template>
      <el-form :model="form" label-width="88px" class="profile-form" @submit.prevent>
        <el-form-item label="呼号" required>
          <el-input v-model="form.call_sign" placeholder="如 BH1ABC" maxlength="20" @blur="form.call_sign = (form.call_sign || '').trim().toUpperCase()" />
        </el-form-item>
        <el-form-item label="名称"><el-input v-model="form.name" maxlength="100" /></el-form-item>
        <el-form-item label="英文名"><el-input v-model="form.name_en" maxlength="100" /></el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<script setup>
import { reactive, ref, onMounted } from 'vue'
import api from '../api'
import { ElMessage } from 'element-plus'

const saving = ref(false)
const form = reactive({
  id: 0,
  call_sign: '',
  name: '',
  name_en: '',
})

function payload() {
  return {
    call_sign: (form.call_sign || '').trim().toUpperCase(),
    name: form.name,
    name_en: form.name_en,
  }
}

async function load() {
  const res = await api.get('/station/profile')
  if (!res || !res.id) return
  form.id = res.id
  form.call_sign = res.call_sign || ''
  form.name = res.name || ''
  form.name_en = res.name_en || ''
}

async function save() {
  const body = payload()
  if (!body.call_sign) {
    ElMessage.warning('呼号必填')
    return
  }
  form.call_sign = body.call_sign
  saving.value = true
  try {
    if (form.id) {
      await api.put('/station/profile/' + form.id, body)
    } else {
      const created = await api.post('/station/profile', body)
      if (created && created.id) form.id = created.id
    }
    ElMessage.success('已保存')
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>

<style scoped>
.card-head { display: flex; align-items: center; justify-content: space-between; }
.profile-form { max-width: 480px; }
</style>
