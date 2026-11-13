<template>
  <div class="row-ops">
    <el-button link type="primary" size="small" @click="emit('detail', row)">详情</el-button>
    <el-button link type="primary" size="small" @click="emit('edit', row)">编辑</el-button>
    <el-dropdown trigger="click" @command="onCommand">
      <el-button link size="small">更多<el-icon class="el-icon--right"><ArrowDown /></el-icon></el-button>
      <template #dropdown>
        <el-dropdown-menu>
          <el-dropdown-item v-if="!row.card_issued" command="issue">制卡</el-dropdown-item>
          <el-dropdown-item v-if="row.card_issued && !row.card_sent" command="sent">发信</el-dropdown-item>
          <el-dropdown-item v-if="row.card_sent && !row.card_received" command="received">对方收卡</el-dropdown-item>
          <el-dropdown-item v-if="row.mail_type === 'REGISTERED' && row.tracking_number" command="track">查快递</el-dropdown-item>
          <el-dropdown-item command="qr">二维码</el-dropdown-item>
          <el-dropdown-item v-if="row.mail_target_email" command="mail-created" divided>建卡通知</el-dropdown-item>
          <el-dropdown-item v-if="row.mail_target_email" command="mail-sent">发信通知</el-dropdown-item>
          <el-dropdown-item v-if="row.mail_target_email" command="mail-received">收卡通知</el-dropdown-item>
          <el-dropdown-item v-if="row.mail_target_email" command="mail-tracking">快递更新</el-dropdown-item>
          <el-dropdown-item command="delete" divided><span class="op-danger">删除</span></el-dropdown-item>
        </el-dropdown-menu>
      </template>
    </el-dropdown>
  </div>
</template>

<script setup>
defineProps({ row: { type: Object, required: true } })
const emit = defineEmits(['detail', 'edit', 'command'])

function onCommand(cmd) {
  emit('command', cmd)
}
</script>

<style scoped>
.row-ops {
  display: inline-flex;
  align-items: center;
  flex-wrap: nowrap;
  white-space: nowrap;
}
.row-ops :deep(.el-button + .el-button) { margin-left: 0; }
</style>

<style>
.op-danger { color: var(--qsl-danger); }
</style>
