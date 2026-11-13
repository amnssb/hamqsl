<template>
  <div class="theme-lab">
    <div class="page-heading">
      <div>
        <h1>主题巡检</h1>
        <p>
          一页铺满项目实际用到的 Element Plus 组件与自有组件，用来逐个主题肉眼验收。
          这里的切换<strong>只影响本页预览</strong>，不会改动站点设置。
        </p>
      </div>
      <el-button @click="restore">退出并恢复</el-button>
    </div>

    <el-card class="lab-card">
      <template #header>
        <div class="lab-head">
          <span>预览主题</span>
          <span class="lab-hint">当前：{{ previewName }}</span>
        </div>
      </template>
      <div class="theme-picker">
        <button
          v-for="opt in themeOptions"
          :key="opt.name"
          class="theme-chip"
          :class="{ active: opt.name === current }"
          @click="preview(opt.name)"
        >
          <span class="chip-dots">
            <i v-for="c in opt.swatches" :key="c" :style="{ background: c }" />
          </span>
          <span class="chip-label">{{ opt.title }}</span>
        </button>
      </div>
    </el-card>

    <!-- 1. 按钮 -->
    <el-card class="lab-card">
      <template #header><span>按钮</span></template>
      <div class="row">
        <el-button>默认</el-button>
        <el-button type="primary">主要</el-button>
        <el-button type="success">成功</el-button>
        <el-button type="warning">警告</el-button>
        <el-button type="danger">危险</el-button>
        <el-button type="info">信息</el-button>
        <el-button disabled>禁用</el-button>
        <el-button type="primary" plain>朴素</el-button>
        <el-button type="primary" :loading="true">加载中</el-button>
        <el-button link type="primary">文字按钮</el-button>
      </div>
    </el-card>

    <!-- 2. 表单控件 -->
    <el-card class="lab-card">
      <template #header><span>表单控件</span></template>
      <el-form label-width="96px" class="lab-form">
        <el-form-item label="输入框">
          <el-input v-model="demo.text" placeholder="占位符颜色也要跟着主题走" clearable />
        </el-form-item>
        <el-form-item label="多行文本">
          <el-input v-model="demo.textarea" type="textarea" :rows="2" />
        </el-form-item>
        <el-form-item label="禁用态">
          <el-input v-model="demo.disabled" disabled />
        </el-form-item>
        <el-form-item label="下拉选择">
          <el-select v-model="demo.select" placeholder="展开看面板" style="width: 240px">
            <el-option label="选项一" value="a" />
            <el-option label="选项二" value="b" />
            <el-option label="选项三" value="c" />
          </el-select>
        </el-form-item>
        <el-form-item label="日期">
          <el-date-picker v-model="demo.date" type="date" value-format="YYYY-MM-DD" style="width: 240px" />
        </el-form-item>
        <el-form-item label="数字">
          <el-input-number v-model="demo.number" :min="0" />
        </el-form-item>
        <el-form-item label="单选">
          <el-radio-group v-model="demo.radio">
            <el-radio value="1">挂号信</el-radio>
            <el-radio value="2">平信</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="按钮组">
          <el-radio-group v-model="demo.radioButton">
            <el-radio-button value="a">QSO</el-radio-button>
            <el-radio-button value="b">SWL</el-radio-button>
            <el-radio-button value="c">EYEBALL</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="多选">
          <el-checkbox-group v-model="demo.checks">
            <el-checkbox value="x" label="建卡通知" />
            <el-checkbox value="y" label="发信通知" />
          </el-checkbox-group>
        </el-form-item>
        <el-form-item label="开关">
          <el-switch v-model="demo.toggle" />
        </el-form-item>
        <el-form-item label="上传">
          <el-upload :show-file-list="false" :auto-upload="false">
            <el-button>选择文件</el-button>
          </el-upload>
        </el-form-item>
      </el-form>
    </el-card>

    <!-- 3. 标签 -->
    <el-card class="lab-card">
      <template #header><span>标签 / 徽章</span></template>
      <div class="row">
        <el-tag>默认</el-tag>
        <el-tag type="success">成功</el-tag>
        <el-tag type="warning">警告</el-tag>
        <el-tag type="danger">危险</el-tag>
        <el-tag type="info">信息</el-tag>
        <el-tag effect="plain">描边</el-tag>
        <el-tag effect="dark" type="primary">深色</el-tag>
      </div>
    </el-card>

    <!-- 4. 表格 + 分页 -->
    <el-card class="lab-card">
      <template #header><span>表格 / 分页</span></template>
      <el-table :data="demo.rows" border stripe>
        <el-table-column prop="code" label="编号" width="110" />
        <el-table-column prop="call" label="呼号" width="110" />
        <el-table-column prop="date" label="日期" width="120" />
        <el-table-column label="状态" width="110">
          <template #default="{ row }">
            <el-tag size="small" :type="row.type">{{ row.status }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="remark" label="备注" min-width="160" show-overflow-tooltip />
        <el-table-column label="操作" width="130" fixed="right">
          <template #default>
            <el-button link type="primary" size="small">详情</el-button>
            <el-button link type="danger" size="small">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-pagination
        v-model:current-page="demo.page"
        :page-size="10"
        :total="86"
        layout="total, prev, pager, next, jumper"
        style="margin-top: 16px; justify-content: flex-end"
      />
    </el-card>

    <!-- 5. 浮层：折叠 / 提示 / 下拉 / 气泡确认 -->
    <el-card class="lab-card">
      <template #header><span>浮层</span></template>
      <div class="row">
        <el-tooltip content="提示气泡" placement="top">
          <el-button>悬停提示</el-button>
        </el-tooltip>
        <el-dropdown trigger="click">
          <el-button>下拉菜单<el-icon class="el-icon--right"><ArrowDown /></el-icon></el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item>建卡通知</el-dropdown-item>
              <el-dropdown-item>发信通知</el-dropdown-item>
              <el-dropdown-item divided>快递更新</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
        <el-popconfirm title="确定删除？" @confirm="noop">
          <template #reference><el-button type="danger" plain>气泡确认</el-button></template>
        </el-popconfirm>
        <el-button @click="dialogVisible = true">打开弹窗</el-button>
        <el-button @click="openMessageBox">确认框</el-button>
        <el-button @click="openMessage">全局提示</el-button>
      </div>
    </el-card>

    <!-- 6. 数据展示 -->
    <el-card class="lab-card">
      <template #header><span>数据展示</span></template>
      <el-descriptions :column="2" border size="small" style="margin-bottom: 20px">
        <el-descriptions-item label="卡片编号">C4XQ7M2P</el-descriptions-item>
        <el-descriptions-item label="呼号">BI1KBU</el-descriptions-item>
        <el-descriptions-item label="状态">
          <el-tag size="small" type="success">已签收</el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="邮寄方式">挂号信</el-descriptions-item>
      </el-descriptions>

      <el-steps :active="2" align-center style="margin-bottom: 20px">
        <el-step title="提交" description="已提交" />
        <el-step title="审核" description="已通过" />
        <el-step title="制卡" description="进行中" />
        <el-step title="签收" description="待完成" />
      </el-steps>

      <div class="row">
        <el-alert type="info" :closable="false" title="信息提示" style="max-width: 260px" />
        <el-alert type="success" :closable="false" title="成功提示" style="max-width: 260px" />
        <el-alert type="warning" :closable="false" title="警告提示" style="max-width: 260px" />
        <el-alert type="error" :closable="false" title="错误提示" style="max-width: 260px" />
      </div>

      <div class="row" style="margin-top: 16px">
        <el-empty description="暂无数据" :image-size="72" />
      </div>
    </el-card>

    <!-- 7. 自有组件：移动端字段卡片 -->
    <el-card class="lab-card">
      <template #header><span>自有组件 · 移动端字段卡片（ResponsiveTable 的 .rcard）</span></template>
      <div class="rcard-list">
        <div class="rcard">
          <div class="rcard-head"><span>C4XQ7M2P</span><el-tag size="small" type="success">已签收</el-tag></div>
          <div class="rcard-row"><span class="rcard-label">呼号</span><span class="rcard-value">BI1KBU</span></div>
          <div class="rcard-row"><span class="rcard-label">日期</span><span class="rcard-value">2026-01-31</span></div>
          <div class="rcard-actions">
            <el-button link type="primary" size="small">详情</el-button>
            <el-button link type="danger" size="small">删除</el-button>
          </div>
        </div>
      </div>
    </el-card>

    <el-dialog v-model="dialogVisible" title="弹窗标题" width="420px">
      <p>弹窗背景、标题、底部分隔线、遮罩色都应该跟随主题。</p>
      <el-alert type="info" :closable="false" title="弹窗里的提示条" />
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="dialogVisible = false">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, reactive, ref, onMounted, onBeforeUnmount } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { ArrowDown } from '@element-plus/icons-vue'
import { applyTheme } from '../plugins/registry'
import api from '../api'

const DEFAULT_LABEL = '默认主题'
const DEFAULT_SWATCHES = ['#f8fafc', '#ffffff', '#2563eb', '#10b981']

// 巡检页自己的预览态：只改 <html data-theme> 与皮肤，不碰后端开关
const current = ref('')
const serverTheme = ref('') // 进页面时服务端生效的主题，用于退出并恢复
const themes = ref([]) // 全部主题，含还没启用的，好在这一页逐个看

const themeOptions = computed(() => [
  { name: '', title: DEFAULT_LABEL, swatches: DEFAULT_SWATCHES },
  ...themes.value,
])

const previewName = computed(
  () => themeOptions.value.find(o => o.name === current.value)?.title || DEFAULT_LABEL
)

function preview(name) {
  current.value = name
  applyTheme(name)
}

function restore() {
  current.value = serverTheme.value
  applyTheme(serverTheme.value)
}

const dialogVisible = ref(false)
const demo = reactive({
  text: '',
  textarea: '多行文本',
  disabled: '禁用状态',
  select: 'a',
  date: '',
  number: 3,
  radio: '1',
  radioButton: 'a',
  checks: ['x'],
  toggle: true,
  page: 1,
  rows: [
    { code: 'C4XQ7M2P', call: 'BI1KBU', date: '2026-01-31', status: '已签收', type: 'success', remark: '挂号信 RA123456789CN' },
    { code: 'C9TK3W8N', call: 'BA1AA', date: '2026-01-28', status: '已发卡', type: 'warning', remark: '平信' },
    { code: 'C2PL6R4D', call: 'BG2XYZ', date: '2026-01-22', status: '待制卡', type: 'info', remark: '' },
  ],
})

function noop() {}
function openMessageBox() {
  ElMessageBox.confirm('这是确认框的正文，看看背景与按钮配色。', '确认操作', { type: 'warning' })
    .catch(() => {})
}
function openMessage() {
  ElMessage.success('这是一条全局提示')
}

onMounted(async () => {
  // 名单、标题、色点都从插件接口来。公开接口只给已启用的，巡检要把没启用的也摆出来。
  try {
    const res = await api.get('/plugins')
    const items = res?.items || []
    themes.value = items
      .filter(p => p.kind === 'theme')
      .map(p => ({
        name: p.name,
        title: p.title,
        swatches: p.swatches && p.swatches.length ? p.swatches : DEFAULT_SWATCHES,
      }))
    const enabled = items.filter(p => p.kind === 'theme' && p.enabled).map(p => p.name)
    serverTheme.value = enabled.length ? enabled[enabled.length - 1] : '' // 和服务端一样，后登记的生效
  } catch {
    serverTheme.value = ''
  }
  current.value = serverTheme.value
})

onBeforeUnmount(() => {
  // 离开页面时把预览态还原，避免影响其它页面的观感
  applyTheme(serverTheme.value)
})
</script>

<style scoped>
.theme-lab {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.lab-card {
  margin: 0;
}

.lab-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.lab-hint {
  color: var(--qsl-muted);
  font-size: 12px;
  font-weight: 500;
}

.theme-picker {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.theme-chip {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 14px;
  border: 1px solid var(--qsl-line);
  border-radius: var(--qsl-radius);
  background: var(--qsl-panel);
  color: var(--qsl-ink);
  font-family: inherit;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.18s ease;
}

.theme-chip:hover {
  border-color: var(--qsl-primary);
}

.theme-chip.active {
  border-color: var(--qsl-primary);
  background: var(--qsl-active-tint);
  color: var(--qsl-primary);
}

.chip-dots {
  display: flex;
  gap: 2px;
}

.chip-dots i {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  border: 1px solid var(--qsl-line);
}

.row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
}

.lab-form {
  max-width: 620px;
}

/* 与 ResponsiveTable.vue 的 .rcard 保持一致的视觉，便于对照 */
.rcard-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
  max-width: 420px;
}
.rcard {
  border: 1px solid var(--qsl-line);
  background: var(--qsl-card-bg);
  border-radius: var(--qsl-radius-card);
  padding: 16px;
  box-shadow: var(--qsl-shadow-1);
}
.rcard-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding-bottom: 10px;
  margin-bottom: 8px;
  border-bottom: 1px solid var(--qsl-line);
  color: var(--qsl-heading);
  font-weight: 750;
  font-size: 14.5px;
}
.rcard-row {
  display: flex;
  gap: 12px;
  padding: 4px 0;
  font-size: 13px;
  line-height: 1.6;
}
.rcard-label {
  flex: none;
  width: 78px;
  color: var(--qsl-muted);
  font-weight: 500;
}
.rcard-value {
  flex: 1;
  min-width: 0;
  word-break: break-all;
  color: var(--qsl-ink);
}
.rcard-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 6px 10px;
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px dashed var(--qsl-line);
}
</style>
