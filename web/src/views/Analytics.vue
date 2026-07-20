<script setup>
import { ref, computed, onMounted } from 'vue'
import { api, isExtension, todayStr, fmtDuration, COMPANIES, companyForGateway, companyForQueue } from '../api.js'
import { t } from '../i18n.js'

const calls = ref([])
const names = ref({})
const extCompany = ref({})
const loading = ref(true)
const msg = ref('')
function flash(t) { msg.value = t; setTimeout(() => (msg.value = ''), 3000) }

const preset = ref('week')
const fromInput = ref('')
const toInput = ref('')
const fCompany = ref('')

function pad(n) { return String(n).padStart(2, '0') }
function opExt(c) {
  if (c.direction === 'outbound') return isExtension(c.caller_id_number) ? c.caller_id_number : ''
  return isExtension(c.destination_number) ? c.destination_number : ''
}
function clientPhone(c) {
  const raw = c.direction === 'outbound' ? c.destination_number : c.caller_id_number
  return String(raw || '').replace(/\D/g, '')
}
function callCompany(c) {
  const g = companyForGateway(c.gateway); if (g) return g
  const e = opExt(c); return e ? (extCompany.value[e] || '') : ''
}
function opName(e) { return names.value[e] || ('Operator ' + e) }
function fmtDateTime(s) { const d = new Date(s * 1000); return `${pad(d.getDate())}.${pad(d.getMonth() + 1)} ${pad(d.getHours())}:${pad(d.getMinutes())}` }
function fmtDelay(sec) { if (sec == null) return '—'; const m = Math.floor(sec / 60); if (m < 60) return `${m} ${t('analytics.min')}`; const h = Math.floor(m / 60); return `${h} ${t('analytics.hour')} ${m % 60} ${t('analytics.min')}` }

function setRange(f, t) { fromInput.value = todayStr(f); toInput.value = todayStr(t) }
function applyPreset(id) {
  preset.value = id; const n = new Date()
  if (id === 'today') setRange(n, n)
  else if (id === 'yesterday') { const y = new Date(n); y.setDate(n.getDate() - 1); setRange(y, y) }
  else if (id === 'week') { const w = new Date(n); w.setDate(n.getDate() - 6); setRange(w, n) }
  else if (id === 'month') { setRange(new Date(n.getFullYear(), n.getMonth(), 1), n) }
  if (id !== 'custom') load()
}
function rangeUnix() {
  const f = new Date(fromInput.value + 'T00:00:00'), t = new Date(toInput.value + 'T23:59:59')
  return [Math.floor(f.getTime() / 1000), Math.floor(t.getTime() / 1000)]
}
const rangeLabel = computed(() => `${fromInput.value} — ${toInput.value}`)

async function load() {
  loading.value = true
  try {
    const [from, to] = rangeUnix()
    const [cs, us] = await Promise.all([api.data('', from, to), api.users().catch(() => [])])
    const nm = {}, cm = {}
    for (const u of us || []) { if (u.num) { nm[String(u.num)] = u.name; cm[String(u.num)] = companyForQueue(u.tr1) } }
    names.value = nm; extCompany.value = cm
    calls.value = [...cs].sort((a, b) => a.start_stamp - b.start_stamp)
  } catch (e) { flash(t('common.errorPrefix') + e.message) }
  finally { loading.value = false }
}

// missed inbound → keyingi javob berilgan qo'ng'iroq (o'sha mijoz) bilan yechilish
const analysis = computed(() => {
  const list = fCompany.value ? calls.value.filter((c) => callCompany(c) === fCompany.value) : calls.value
  // mijoz telefoni bo'yicha indeks (javob berilgan qo'ng'iroqlar)
  const answeredByPhone = {}
  for (const c of list) {
    if ((c.user_talk_time || 0) > 0) {
      const p = clientPhone(c); if (!p) continue
      ;(answeredByPhone[p] ||= []).push(c)
    }
  }
  const rows = []
  for (const c of list) {
    const missed = c.direction !== 'outbound' && (c.user_talk_time || 0) === 0 && (c.duration || 0) > 5
    if (!missed) continue
    const p = clientPhone(c)
    const cand = (answeredByPhone[p] || []).find((x) => x.start_stamp > c.start_stamp)
    rows.push({
      uuid: c.uuid, phone: p, ext: opExt(c), start: c.start_stamp,
      resolved: !!cand, delay: cand ? cand.start_stamp - c.start_stamp : null,
      company: callCompany(c),
    })
  }
  return rows.sort((a, b) => b.start - a.start)
})

const kpi = computed(() => {
  const r = analysis.value
  const resolved = r.filter((x) => x.resolved)
  const delays = resolved.map((x) => x.delay)
  const sla15 = resolved.filter((x) => x.delay <= 900).length
  const avg = delays.length ? Math.round(delays.reduce((a, b) => a + b, 0) / delays.length) : 0
  return {
    total: r.length, resolved: resolved.length, unresolved: r.length - resolved.length,
    sla: r.length ? Math.round(sla15 / r.length * 100) : 0, avg,
  }
})
const buckets = computed(() => {
  const b = { m5: 0, m15: 0, m30: 0, m30p: 0, none: 0 }
  for (const x of analysis.value) {
    if (!x.resolved) { b.none++; continue }
    if (x.delay <= 300) b.m5++
    else if (x.delay <= 900) b.m15++
    else if (x.delay <= 1800) b.m30++
    else b.m30p++
  }
  return b
})

const companies = COMPANIES
onMounted(() => applyPreset('week'))
</script>

<template>
  <div class="an">
    <div class="top"><div><h1>{{ t('analytics.title') }}</h1><p>{{ t('analytics.subtitle') }}</p></div></div>
    <Transition name="page"><div v-if="msg" class="toast">{{ msg }}</div></Transition>

    <div class="kpis">
      <div class="kpi card"><div class="kpi__v" style="color:var(--red)">{{ kpi.total }}</div><div class="kpi__l">{{ t('st.missed') }}</div></div>
      <div class="kpi card"><div class="kpi__v" style="color:var(--green)">{{ kpi.resolved }}</div><div class="kpi__l">{{ t('analytics.reconnected') }}</div></div>
      <div class="kpi card"><div class="kpi__v" style="color:var(--amber)">{{ kpi.unresolved }}</div><div class="kpi__l">{{ t('analytics.notReconnected') }}</div></div>
      <div class="kpi card"><div class="kpi__v" style="color:var(--accent)">{{ kpi.sla }}%</div><div class="kpi__l">{{ t('analytics.sla15') }}</div></div>
      <div class="kpi card"><div class="kpi__v mono">{{ fmtDelay(kpi.avg) }}</div><div class="kpi__l">{{ t('analytics.avgDelay') }}</div></div>
    </div>

    <div class="card filters">
      <div class="fl-presets">
        <button v-for="p in [['today','common.today'],['yesterday','common.yesterday'],['week','common.week'],['month','common.month']]" :key="p[0]"
                class="preset" :class="{ active: preset === p[0] }" @click="applyPreset(p[0])">{{ t(p[1]) }}</button>
        <span class="fl-range mono">{{ rangeLabel }}</span>
      </div>
      <div class="fl-row">
        <label class="fld"><span>{{ t('common.from') }}</span><input type="date" v-model="fromInput" @change="preset='custom'; load()" /></label>
        <label class="fld"><span>{{ t('common.to') }}</span><input type="date" v-model="toInput" @change="preset='custom'; load()" /></label>
        <label class="fld"><span>{{ t('common.channel') }}</span><select v-model="fCompany"><option value="">{{ t('common.all') }}</option><option v-for="c in companies.filter(x=>x.id)" :key="c.id" :value="c.id">{{ c.name }}</option></select></label>
      </div>
    </div>

    <div class="section-h"><h2>{{ t('analytics.delayDist') }}</h2></div>
    <div class="card buckets">
      <div class="bk"><span class="bk__v" style="color:var(--green)">{{ buckets.m5 }}</span><span class="bk__l">{{ t('analytics.b5') }}</span></div>
      <div class="bk"><span class="bk__v" style="color:#22c55e">{{ buckets.m15 }}</span><span class="bk__l">{{ t('analytics.b15') }}</span></div>
      <div class="bk"><span class="bk__v" style="color:var(--amber)">{{ buckets.m30 }}</span><span class="bk__l">{{ t('analytics.b30') }}</span></div>
      <div class="bk"><span class="bk__v" style="color:#f97316">{{ buckets.m30p }}</span><span class="bk__l">{{ t('analytics.b30p') }}</span></div>
      <div class="bk"><span class="bk__v" style="color:var(--red)">{{ buckets.none }}</span><span class="bk__l">{{ t('analytics.notReconnected') }}</span></div>
    </div>

    <div class="section-h"><h2>{{ t('st.missed') }} <span class="count">{{ analysis.length }}</span></h2></div>
    <div v-if="loading" class="loading"><i class="spin"></i></div>
    <div v-else class="card tbl-wrap">
      <table class="tbl">
        <thead><tr><th>{{ t('st.time') }}</th><th>{{ t('common.client') }}</th><th>{{ t('common.operator') }}</th><th class="ta-c">{{ t('tv.status') }}</th><th class="ta-r">{{ t('analytics.delay') }}</th></tr></thead>
        <tbody>
          <tr v-for="r in analysis.slice(0, 300)" :key="r.uuid">
            <td class="mono dim">{{ fmtDateTime(r.start) }}</td>
            <td class="mono">{{ r.phone }}</td>
            <td>{{ r.ext ? opName(r.ext) : '—' }}</td>
            <td class="ta-c"><span class="st" :class="r.resolved ? 'ok' : 'bad'">{{ r.resolved ? t('analytics.reconnectedShort') : t('analytics.notReconnected') }}</span></td>
            <td class="ta-r mono">{{ fmtDelay(r.delay) }}</td>
          </tr>
          <tr v-if="!analysis.length"><td colspan="5" class="empty">{{ t('analytics.noMissed') }} 🎉</td></tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<style scoped>
.an { animation: fade-up 0.4s both; }
.top { margin: 16px 0 18px; }
.top h1 { font-size: 24px; font-weight: 800; }
.top p { color: var(--text-dim); font-size: 13px; margin-top: 4px; }
.toast { position: fixed; top: 22px; left: 50%; transform: translateX(-50%); z-index: 60; background: var(--grad); color: #fff; padding: 11px 22px; border-radius: 12px; font-weight: 600; box-shadow: var(--glow); }
.kpis { display: grid; grid-template-columns: repeat(5, 1fr); gap: 14px; margin-bottom: 18px; }
.kpi { padding: 18px 20px; }
.kpi__v { font-size: 26px; font-weight: 800; font-family: var(--mono); line-height: 1; }
.kpi__l { font-size: 12px; color: var(--text-dim); margin-top: 5px; }
.filters { padding: 16px 18px; margin-bottom: 22px; }
.fl-presets { display: flex; align-items: center; gap: 6px; margin-bottom: 14px; flex-wrap: wrap; }
.preset { background: var(--surface-2); color: var(--text-dim); padding: 7px 14px; font-size: 12.5px; border: 1px solid var(--border); }
.preset:hover { transform: none; box-shadow: none; color: var(--text); }
.preset.active { background: var(--grad); color: #fff; border-color: transparent; }
.fl-range { margin-left: auto; font-size: 12px; color: var(--text-faint); }
.fl-row { display: flex; gap: 12px; }
.fld { display: flex; flex-direction: column; gap: 6px; }
.fld span { font-size: 11.5px; font-weight: 600; color: var(--text-dim); }
.section-h { margin: 20px 0 14px; }
.section-h h2 { font-size: 16px; font-weight: 700; display: flex; align-items: center; gap: 10px; }
.count { font-size: 12px; font-weight: 600; color: var(--text-dim); background: var(--surface-2); padding: 2px 9px; border-radius: 999px; }
.buckets { display: grid; grid-template-columns: repeat(5, 1fr); gap: 12px; padding: 20px; }
.bk { text-align: center; }
.bk__v { display: block; font-size: 26px; font-weight: 800; font-family: var(--mono); }
.bk__l { font-size: 11.5px; color: var(--text-dim); margin-top: 4px; }
.tbl-wrap { padding: 6px 8px; overflow-x: auto; }
.tbl { width: 100%; border-collapse: collapse; }
.tbl th { text-align: left; font-size: 11px; font-weight: 600; color: var(--text-faint); text-transform: uppercase; padding: 12px 14px; }
.tbl td { padding: 10px 14px; border-top: 1px solid var(--border); font-size: 13px; }
.ta-c { text-align: center; } .ta-r { text-align: right; }
.dim { color: var(--text-faint); }
.st { font-size: 11px; font-weight: 600; padding: 3px 10px; border-radius: 999px; }
.st.ok { background: rgba(16,185,129,0.14); color: var(--green); }
.st.bad { background: rgba(239,68,68,0.14); color: var(--red); }
.empty { text-align: center; color: var(--text-faint); padding: 30px; }
.loading { display: flex; justify-content: center; padding: 40px; }
.spin { width: 18px; height: 18px; border: 2.5px solid var(--border-strong); border-top-color: var(--accent); border-radius: 50%; animation: spin 0.8s linear infinite; }
@media (max-width: 1080px) { .kpis, .buckets { grid-template-columns: repeat(2, 1fr); } }
</style>
