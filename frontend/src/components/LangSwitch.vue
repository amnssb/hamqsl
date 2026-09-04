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
.lang-switch { position: fixed; right: 16px; bottom: 16px; z-index: 3000; display: flex; overflow: hidden; background: #fff; border: 1px solid var(--qsl-line, #e5e0d5); border-radius: 999px; box-shadow: 2px 3px 0 rgba(24,45,61,.12); }
.lang-switch button { padding: 6px 13px; color: var(--qsl-navy, #182d3d); font-family: inherit; font-size: 12px; border: 0; background: transparent; cursor: pointer; }
.lang-switch button + button { border-left: 1px solid var(--qsl-line, #e5e0d5); }
.lang-switch button.active { color: var(--qsl-navy, #182d3d); background: var(--qsl-yellow, #f5c518); font-weight: 800; }
.lang-switch button:hover:not(.active) { background: #f7f4ec; }
</style>
