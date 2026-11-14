<!--
对外展示插件的自有界面。
插件页上启用后多一个「取展示代码」按钮，点开是贴到 QRZ / 博客的 HTML 和 JSON 地址。
-->
<template>
  <el-button size="small" text type="primary" @click="open">取展示代码</el-button>
  <el-dialog v-model="visible" title="对外展示" width="640px">
    <el-alert
      v-if="!embed.site_url"
      type="warning"
      :closable="false"
      show-icon
      title="还没有可用的站点地址"
      description="到设置里把站点地址写成 https://你的域名。不填的话，贴出去的图片指不回这台服务器。"
    />
    <el-alert
      v-else-if="!embed.https"
      type="warning"
      :closable="false"
      show-icon
      title="当前站点地址是 http"
      description="QRZ 简介是 https 页面，http 图片会被浏览器丢掉。博客如果也是 https，同样显示不出来。"
    />
    <template v-if="embed.html">
      <p class="embed-note">QRZ 简介和博客都会丢掉脚本和内嵌页，图片和链接留得住。下面这一段两边都能贴。数字大约 60 秒更新一次，不含地址、邮箱和备注。</p>
      <el-input :model-value="embed.html" type="textarea" readonly :rows="4" />
      <p class="embed-note">自己排版时直接 GET 下面的地址，不要加自定义请求头。</p>
      <el-input :model-value="embed.feed_url" readonly />
      <div class="embed-actions">
        <el-button size="small" @click="copyText(embed.html)">复制 HTML</el-button>
        <el-button size="small" @click="copyText(embed.feed_url)">复制 JSON 地址</el-button>
        <el-button v-if="!embed.https" size="small" @click="close(); goSettings()">去改成 https</el-button>
      </div>
    </template>
    <div v-else class="embed-actions">
      <el-button type="primary" @click="close(); goSettings()">去填写站点地址</el-button>
    </div>
  </el-dialog>
</template>

<script setup>
// --- 打开弹窗并取 embed 数据 ---
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import api from '../../api'

const router = useRouter()
const visible = ref(false)
const embed = ref({ site_url: '', html: '', feed_url: '', https: false })

async function open() {
  const res = await api.get('/ext/showcase/embed')
  embed.value = res || { site_url: '', html: '', feed_url: '', https: false }
  visible.value = true
}

function close() {
  visible.value = false
}

function goSettings() {
  router.push('/admin/settings') // https 提示的跳转目标：设置页改站点地址
}

// --- 复制到剪贴板 ---
async function copyText(text) {
  if (!text) return
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success('已复制')
  } catch {
    ElMessage.warning('复制失败，请手动选择文本')
  }
}
</script>

<style scoped>
.embed-note { margin: 12px 0 8px; color: var(--qsl-muted); font-size: 13px; line-height: 1.7; }
.embed-actions { display: flex; gap: 8px; margin-top: 12px; }
</style>
