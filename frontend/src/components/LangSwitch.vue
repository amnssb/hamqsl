<template>
  <div class="lang-switch" role="group" aria-label="Language / 语言">
    <button :class="{ active: locale.current === 'zh' }" @click="setLocale('zh')">中文</button>
    <button :class="{ active: locale.current === 'en' }" @click="setLocale('en')">EN</button>
  </div>
</template>

<script setup>
import { locale, setLocale } from '../i18n'
</script>

<style scoped>
.lang-switch {
  position: fixed;
  right: 20px;
  bottom: 20px;
  z-index: 3000;
  display: flex;
  padding: 3px;
  background: #ffffff;
  border: 1px solid #e2e8f0;
  border-radius: 9999px;
  box-shadow: 0 4px 14px rgba(0, 0, 0, 0.08);
  transition: transform 0.2s ease, box-shadow 0.2s ease;
}
.lang-switch:hover {
  transform: translateY(-1px);
  box-shadow: 0 6px 20px rgba(0, 0, 0, 0.12);
}
.lang-switch button {
  padding: 6px 14px;
  color: #475569;
  font-family: inherit;
  font-size: 12px;
  font-weight: 600;
  border: 0;
  border-radius: 9999px;
  background: transparent;
  cursor: pointer;
  transition: all 0.18s ease;
}
.lang-switch button.active {
  color: #ffffff;
  background: linear-gradient(135deg, #3b82f6 0%, #2563eb 100%);
  box-shadow: 0 2px 6px rgba(37, 99, 235, 0.3);
}
.lang-switch button:hover:not(.active) {
  color: #0f172a;
  background: #f1f5f9;
}
</style>
