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
  background: var(--qsl-card-bg);
  border: 1px solid var(--qsl-line);
  border-radius: 9999px;
  box-shadow: var(--qsl-shadow-1);
  transition: transform 0.2s ease, box-shadow 0.2s ease;
}
.lang-switch:hover {
  transform: translateY(-1px);
  box-shadow: var(--qsl-shadow-2);
}
.lang-switch button {
  padding: 6px 14px;
  color: var(--qsl-muted);
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
  color: var(--qsl-on-primary);
  background: var(--qsl-primary);
  box-shadow: 0 2px 6px var(--qsl-focus-ring);
}
.lang-switch button:hover:not(.active) {
  color: var(--qsl-ink);
  background: var(--qsl-panel-2);
}
</style>
