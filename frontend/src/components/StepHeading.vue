<template>
  <div class="step-heading">
    <span v-if="n != null && n !== ''" class="step-badge" aria-hidden="true">{{ n }}</span>
    <h3>{{ title }}</h3>
  </div>
</template>

<script setup>
defineProps({
  n: { type: [Number, String], default: undefined },
  title: { type: String, required: true },
})
</script>
