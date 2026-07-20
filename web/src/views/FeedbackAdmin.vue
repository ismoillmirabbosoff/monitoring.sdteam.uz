<script setup>
import { ref, computed, onMounted } from 'vue'
import { api, todayStr } from '../api.js'
import { t } from '../i18n.js'

const items = ref([])
const loading = ref(true)
const msg = ref('')
function flash(t) { msg.value = t; setTimeout(() => (msg.value = ''), 3000) }

const preset = ref('week')
const fromInput = ref('')
const toInput = ref('')
const fScore = ref('')

function setRange(f, t) { fromInput.value = todayStr(f); toInput.value = todayStr(t) }
function applyPreset(id) {
  preset.value = id; const n = new Date()
  if (id === 'today') setRange(n, n)
  else if (id === 'week') { const w = new Date(n); w.setDate(n.getDate() - 6); setRange(w, n) }
  else if (id === 'month') { setRange(new Date(n.getFullYear(), n.getMonth(), 1), n) }
  load()
}
function rangeUnix() {
  const f = new Date(fromInput.value + 'T00:00:00'), t = new Date(toInput.value + 'T23:59:59')
  return [Math.floor(f.getTime() / 1000), Math.floor(t.getTime() / 1000)]
}
function fmt(ts) { const d = new Date(ts); const p = (n) => String(n).padStart(2, '0'); return `${p(d.getDate())}.${p(d.getMonth() + 1)} ${p(d.getHours())}:${p(d.getMinutes())}` }

async function load() {
  loading.value = true
  try {
    const [from, to] = rangeUnix()
    items.value = await api.feedbackList({ from, to, score: fScore.value }) || []
  } catch (e) { flash(t('common.errorPrefix') + e.message) }
  finally { loading.value = false }
}

const stats = computed(() => {
  const n = items.value.length
  const avg = n ? (items.value.reduce((a, b) => a + b.score, 0) / n) : 0
  const promoters = items.value.filter((x) => x.score >= 4).length
  const detractors = items.value.filter((x) => x.score <= 2).length
  return { n, avg: avg.toFixed(2), promoters, detractors }
})

onMounted(() => applyPreset('week'))
</script>

<template>
  <div class="fa">
    <div class="top"><div><h1>{{ t('nav.feedback') }}</h1><p>{{ t('feedbackAdmin.subtitle') }}</p></div></div>
    <Transition name="page"><div v-if="msg" class="toast">{{ msg }}</div></Transition>

    <div class="kpis">
      <div class="kpi card"><div class="kpi__v">{{ stats.n }}</div><div class="kpi__l">{{ t('feedbackAdmin.count') }}</div></div>
      <div class="kpi card"><div class="kpi__v" style="color:var(--amber)">{{ stats.avg }} ★</div><div class="kpi__l">{{ t('feedbackAdmin.avg') }}</div></div>
      <div class="kpi card"><div class="kpi__v" style="color:var(--green)">{{ stats.promoters }}</div><div class="kpi__l">{{ t('feedbackAdmin.good') }}</div></div>
      <div class="kpi card"><div class="kpi__v" style="color:var(--red)">{{ stats.detractors }}</div><div class="kpi__l">{{ t('feedbackAdmin.bad') }}</div></div>
    </div>

    <div class="card filters">
      <div class="fl-presets">
        <button v-for="p in [['today','common.today'],['week','common.week'],['month','common.month']]" :key="p[0]"
                class="preset" :class="{ active: preset === p[0] }" @click="applyPreset(p[0])">{{ t(p[1]) }}</button>
      </div>
      <div class="fl-row">
        <label class="fld"><span>{{ t('common.from') }}</span><input type="date" v-model="fromInput" @change="preset='custom'; load()" /></label>
        <label class="fld"><span>{{ t('common.to') }}</span><input type="date" v-model="toInput" @change="preset='custom'; load()" /></label>
        <label class="fld"><span>{{ t('feedbackAdmin.score') }}</span><select v-model="fScore" @change="load"><option value="">{{ t('common.all') }}</option><option v-for="n in 5" :key="n" :value="n">{{ n }} ★</option></select></label>
      </div>
    </div>

    <div v-if="loading" class="loading"><i class="spin"></i></div>
    <div v-else class="card tbl-wrap">
      <table class="tbl">
        <thead><tr><th>{{ t('st.time') }}</th><th>{{ t('common.phone') }}</th><th class="ta-c">{{ t('feedbackAdmin.score') }}</th><th>{{ t('survey.comment') }}</th></tr></thead>
        <tbody>
          <tr v-for="f in items" :key="f.id">
            <td class="mono dim">{{ fmt(f.created_at) }}</td>
            <td class="mono">{{ f.phone || '—' }}</td>
            <td class="ta-c"><span class="stars" :class="'s' + f.score">{{ '★'.repeat(f.score) }}{{ '☆'.repeat(5 - f.score) }}</span></td>
            <td>{{ f.comment || '—' }}</td>
          </tr>
          <tr v-if="!items.length"><td colspan="4" class="empty">{{ t('feedbackAdmin.noRatings') }}</td></tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<style scoped>
.fa { animation: fade-up 0.4s both; }
.top { margin: 16px 0 18px; }
.top h1 { font-size: 24px; font-weight: 800; }
.top p { color: var(--text-dim); font-size: 13px; margin-top: 4px; }
.toast { position: fixed; top: 22px; left: 50%; transform: translateX(-50%); z-index: 60; background: var(--grad); color: #fff; padding: 11px 22px; border-radius: 12px; font-weight: 600; box-shadow: var(--glow); }
.kpis { display: grid; grid-template-columns: repeat(4, 1fr); gap: 14px; margin-bottom: 18px; }
.kpi { padding: 18px 20px; }
.kpi__v { font-size: 28px; font-weight: 800; font-family: var(--mono); line-height: 1; }
.kpi__l { font-size: 12px; color: var(--text-dim); margin-top: 5px; }
.filters { padding: 16px 18px; margin-bottom: 22px; }
.fl-presets { display: flex; gap: 6px; margin-bottom: 14px; flex-wrap: wrap; }
.preset { background: var(--surface-2); color: var(--text-dim); padding: 7px 14px; font-size: 12.5px; border: 1px solid var(--border); }
.preset:hover { transform: none; box-shadow: none; color: var(--text); }
.preset.active { background: var(--grad); color: #fff; border-color: transparent; }
.fl-row { display: flex; gap: 12px; }
.fld { display: flex; flex-direction: column; gap: 6px; }
.fld span { font-size: 11.5px; font-weight: 600; color: var(--text-dim); }
.tbl-wrap { padding: 6px 8px; overflow-x: auto; }
.tbl { width: 100%; border-collapse: collapse; }
.tbl th { text-align: left; font-size: 11px; font-weight: 600; color: var(--text-faint); text-transform: uppercase; padding: 12px 14px; }
.tbl td { padding: 10px 14px; border-top: 1px solid var(--border); font-size: 13px; }
.ta-c { text-align: center; }
.dim { color: var(--text-faint); }
.stars { font-size: 14px; letter-spacing: 1px; }
.stars.s1, .stars.s2 { color: var(--red); }
.stars.s3 { color: var(--amber); }
.stars.s4, .stars.s5 { color: var(--green); }
.empty { text-align: center; color: var(--text-faint); padding: 30px; }
.loading { display: flex; justify-content: center; padding: 40px; }
.spin { width: 18px; height: 18px; border: 2.5px solid var(--border-strong); border-top-color: var(--accent); border-radius: 50%; animation: spin 0.8s linear infinite; }
@media (max-width: 1080px) { .kpis { grid-template-columns: repeat(2, 1fr); } }
</style>
