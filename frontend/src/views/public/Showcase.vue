<template>
  <div class="public-page">
    <div class="public-container">
      <header class="public-header">
        <div class="logo-section">
          <div class="logo-icon">QSL</div>
          <div>
            <h1>{{ heading }}</h1>
            <p class="subtitle">{{ subheading }}</p>
          </div>
        </div>
        <button class="home-back-link" @click="$router.push('/')">
          <el-icon><ArrowLeft /></el-icon>
          <span>{{ t('common.backHome') }}</span>
        </button>
      </header>

      <main class="public-main">
        <div v-if="state === 'off'" class="modern-card">
          <h2>{{ t('showcase.offTitle') }}</h2>
          <p class="subtitle">{{ t('showcase.offDesc') }}</p>
        </div>
        <div v-else-if="state === 'fail'" class="modern-card">
          <h2>{{ t('showcase.fail') }}</h2>
        </div>
        <div v-else-if="state === 'loading'" class="modern-card">
          <p class="subtitle">{{ t('common.loading') }}</p>
        </div>
        <template v-else-if="state === 'ready'">
          <section class="counts">
            <div class="count"><b>{{ counts.qso }}</b><span>{{ t('showcase.qso') }}</span></div>
            <div class="count"><b>{{ counts.cards_sent }}</b><span>{{ t('showcase.sent') }}</span></div>
            <div class="count"><b>{{ counts.cards_signed }}</b><span>{{ t('showcase.signed') }}</span></div>
            <div class="count"><b>{{ counts.cards_received }}</b><span>{{ t('showcase.received') }}</span></div>
          </section>

          <section class="modern-card">
            <div class="card-header">
              <h2>{{ t('showcase.recent') }}</h2>
            </div>
            <p v-if="!recent.length" class="subtitle">{{ t('showcase.emptyQso') }}</p>
            <div v-for="row in recent" :key="row.call_sign + row.date + row.time" class="qso-row">
              <b>{{ row.call_sign }}</b>
              <span>{{ row.date }} {{ row.time }}</span>
              <span>{{ row.band }} {{ row.mode }}</span>
              <span>{{ row.freq }}</span>
              <span v-if="row.rst_sent || row.rst_rcvd">{{ t('showcase.rst') }} {{ row.rst_sent }}/{{ row.rst_rcvd }}</span>
            </div>
          </section>

          <section class="modern-card">
            <div class="card-header">
              <h2>{{ t('showcase.cards') }}</h2>
            </div>
            <p v-if="!cards.length" class="subtitle">{{ t('showcase.emptyCards') }}</p>
            <div v-else class="card-grid">
              <figure v-for="card in cards" :key="card.card_version">
                <img v-if="card.image_path" :src="card.image_path" :alt="card.card_version">
                <figcaption>{{ card.card_version }}</figcaption>
              </figure>
            </div>
          </section>
        </template>
      </main>
    </div>
    <LangSwitch />
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import LangSwitch from '../../components/LangSwitch.vue'
import { t } from '../../i18n'

const state = ref('loading')
const station = ref({ call_sign: '', name: '', name_en: '' })
const counts = ref({ qso: 0, cards_sent: 0, cards_signed: 0, cards_received: 0 })
const recent = ref([])
const cards = ref([])

const heading = computed(() => station.value.call_sign || t('showcase.title'))
const subheading = computed(() => station.value.name || station.value.name_en || t('showcase.subtitle'))

onMounted(async () => {
  let res
  try {
    res = await fetch('/api/public/ext/showcase?limit=8')
  } catch {
    state.value = 'fail'
    return
  }
  if (res.status === 404) {
    state.value = 'off'
    return
  }
  if (!res.ok) {
    state.value = 'fail'
    return
  }
  const body = await res.json()
  const data = body.data || {}
  station.value = data.station || station.value
  counts.value = data.counts || counts.value
  recent.value = data.recent || []
  cards.value = data.cards || []
  if (station.value.call_sign) document.title = station.value.call_sign
  state.value = 'ready'
})
</script>

<style scoped>
.counts {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 10px;
  margin-bottom: 20px;
}
.count {
  padding: 14px 16px;
  background: var(--qsl-card-bg);
  border: 1px solid var(--qsl-line);
  border-radius: var(--qsl-radius-card);
}
.count b {
  display: block;
  color: var(--qsl-heading);
  font-size: 22px;
  font-weight: 750;
}
.count span {
  color: var(--qsl-muted);
  font-size: 12px;
}
.qso-row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px 16px;
  padding: 10px 0;
  border-bottom: 1px solid var(--qsl-line);
  color: var(--qsl-ink);
  font-size: 13px;
}
.qso-row b {
  min-width: 7em;
  color: var(--qsl-heading);
}
.card-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
  gap: 12px;
}
.card-grid figure {
  margin: 0;
}
.card-grid img {
  width: 100%;
  height: 100px;
  object-fit: cover;
  border-radius: 8px;
  background: var(--qsl-panel);
}
.card-grid figcaption {
  margin-top: 6px;
  color: var(--qsl-muted);
  font-size: 12px;
}
@media (max-width: 640px) {
  .counts { grid-template-columns: repeat(2, 1fr); }
}
</style>
