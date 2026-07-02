<script setup>
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import StatCard from '../components/StatCard.vue'
import OperatorCard from '../components/OperatorCard.vue'
import HourlyChart from '../components/HourlyChart.vue'
import CallsFeed from '../components/CallsFeed.vue'
import StatsBlock from '../components/StatsBlock.vue'
import {
  api, todayStr, parseFifoUsers, isExtension, fmtDuration,
  COMPANIES, companyForGateway, companyForQueue,
} from '../api.js'
import { t } from '../i18n.js'

const ICONS = {
  phone: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 16.92v3a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07 19.5 19.5 0 0 1-6-6 19.79 19.79 0 0 1-3.07-8.67A2 2 0 0 1 4.11 2h3a2 2 0 0 1 2 1.72c.13.96.36 1.9.7 2.81a2 2 0 0 1-.45 2.11L8.09 9.91a16 16 0 0 0 6 6l1.27-1.27a2 2 0 0 1 2.11-.45c.91.34 1.85.57 2.81.7A2 2 0 0 1 22 16.92z"/></svg>',
  in: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M17 7 7 17M7 17h8M7 17V9"/></svg>',
  out: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M7 17 17 7M17 7H9M17 7v8"/></svg>',
  clock: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="9"/><path d="M12 7v5l3 2"/></svg>',
  users: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87M16 3.13a4 4 0 0 1 0 7.75"/></svg>',
}

const allCalls = ref([])
const fifoOnline = ref({})       // ext -> online bool
const userNames = ref({})        // ext -> operator ismi
const userCompany = ref({})      // ext -> kompaniya (tr1 navbat bo'yicha)
const liveStatus = ref({})       // ext -> 'ringing'|'talking'
const hiddenExts = ref(new Set())
const loading = ref(true)
const error = ref('')
const wsState = ref('connecting')
const now = ref(new Date())
const company = ref('')          // '' | 'salesdoc' | 'ibox'
const stats = ref(null)
let ws = null, pollTimer = null, clockTimer = null, reconnectTimer = null

const companies = COMPANIES

async function loadStats() {
  try { stats.value = await api.stats(company.value) } catch { /* ixtiyoriy */ }
}
watch(company, loadStats)

// --- Sana/vaqt filtri ---
function pad(n) { return String(n).padStart(2, '0') }
function toLocalInput(d) { return `${d.getFullYear()}-${pad(d.getMonth()+1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}` }
function startOfDay(d) { const x = new Date(d); x.setHours(0,0,0,0); return x }

const preset = ref('today')
const fromInput = ref(toLocalInput(startOfDay(new Date())))
const toInput = ref(toLocalInput(new Date()))

const PRESETS = [
  { id: 'today', key: 'common.today' },
  { id: 'yesterday', key: 'common.yesterday' },
  { id: 'week', key: 'common.week' },
  { id: 'month', key: 'common.month' },
  { id: 'custom', key: 'common.custom' },
]

function applyPreset(id) {
  preset.value = id
  const n = new Date()
  if (id === 'today') { fromInput.value = toLocalInput(startOfDay(n)); toInput.value = toLocalInput(n) }
  else if (id === 'yesterday') { const y = new Date(n); y.setDate(n.getDate()-1); fromInput.value = toLocalInput(startOfDay(y)); const e = startOfDay(y); e.setHours(23,59); toInput.value = toLocalInput(e) }
  else if (id === 'week') { const w = new Date(n); w.setDate(n.getDate()-6); fromInput.value = toLocalInput(startOfDay(w)); toInput.value = toLocalInput(n) }
  else if (id === 'month') { fromInput.value = toLocalInput(new Date(n.getFullYear(), n.getMonth(), 1)); toInput.value = toLocalInput(n) }
  if (id !== 'custom') refresh()
}
function applyCustom() { preset.value = 'custom'; refresh() }

const fromUnix = computed(() => Math.floor(new Date(fromInput.value).getTime() / 1000))
const toUnix = computed(() => Math.floor(new Date(toInput.value).getTime() / 1000))
const rangeLabel = computed(() => {
  const f = new Date(fromInput.value), t = new Date(toInput.value)
  const fmt = (d) => d.toLocaleString('ru-RU', { day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit' })
  return `${fmt(f)} — ${fmt(t)}`
})

// Qo'ng'iroq kompaniyasi: avval gateway (712*/781*), aks holda operator ext bo'yicha
// (kiruvchi qo'ng'iroqlarda gateway kompaniyani ko'rsatmasligi mumkin).
function callCompany(c) {
  const g = companyForGateway(c.gateway)
  if (g) return g
  const ext = c.direction === 'outbound' ? c.caller_id_number : c.destination_number
  if (isExtension(ext)) return userCompany.value[String(ext)] || ''
  return ''
}

// Tanlangan kompaniya bo'yicha filtrlangan qo'ng'iroqlar
const calls = computed(() => {
  if (!company.value) return allCalls.value
  return allCalls.value.filter((c) => callCompany(c) === company.value)
})

const kpi = computed(() => {
  let inc = 0, out = 0, talk = 0, answered = 0
  for (const c of calls.value) {
    if (c.direction === 'outbound') out++; else inc++
    if (c.user_talk_time > 0) { talk += c.user_talk_time; answered++ }
  }
  return { total: calls.value.length, inc, out, avgTalk: answered ? talk / answered : 0 }
})

const extStats = computed(() => {
  const m = {}
  const touch = (e) => (m[e] ||= { ext: e, incoming: 0, outgoing: 0, talk: 0, answered: 0 })
  for (const c of calls.value) {
    const t = c.user_talk_time || 0
    if (c.direction === 'outbound') {
      if (isExtension(c.caller_id_number)) { const o = touch(c.caller_id_number); o.outgoing++; o.talk += t; if (t > 0) o.answered++ }
    } else {
      if (isExtension(c.destination_number)) { const o = touch(c.destination_number); o.incoming++; o.talk += t; if (t > 0) o.answered++ }
    }
  }
  return m
})

const operators = computed(() => {
  const exts = new Set([
    ...Object.keys(fifoOnline.value),
    ...Object.keys(extStats.value),
    ...Object.keys(userNames.value),
  ])
  let list = [...exts].filter((ext) => !hiddenExts.value.has(ext)).map((ext) => {
    const s = extStats.value[ext] || { incoming: 0, outgoing: 0, talk: 0, answered: 0 }
    let status = 'offline'
    if (fifoOnline.value[ext]) status = 'online'
    if (liveStatus.value[ext]) status = liveStatus.value[ext]
    return {
      ext,
      name: userNames.value[ext] || `Operator ${ext}`,
      company: userCompany.value[ext] || '',
      status,
      incoming: s.incoming,
      outgoing: s.outgoing,
      talk: s.talk,
      avgTalk: s.answered > 0 ? Math.round(s.talk / s.answered) : 0,
    }
  })
  if (company.value) list = list.filter((o) => o.company === company.value)
  const rank = { ringing: 0, talking: 1, online: 2, offline: 3 }
  list.sort((a, b) => rank[a.status] - rank[b.status] || Number(a.ext) - Number(b.ext))
  return list
})

const activeOps = computed(() => operators.value.filter((o) => o.status !== 'offline').length)
const recentCalls = computed(() =>
  [...calls.value].sort((a, b) => b.start_stamp - a.start_stamp).slice(0, 40)
)
const clock = computed(() => now.value.toLocaleTimeString('ru-RU'))
const dateLabel = computed(() =>
  now.value.toLocaleDateString('uz-UZ', { weekday: 'long', day: 'numeric', month: 'long' })
)

async function loadFifo() {
  try {
    const r = await api.fifo()
    const map = {}
    for (const q of r.data || []) {
      for (const u of parseFifoUsers(q.users)) map[u.ext] = u.online || map[u.ext]
    }
    fifoOnline.value = map
  } catch (e) { /* fifo ixtiyoriy */ }
}

async function loadUsers() {
  try {
    const users = await api.users()
    const names = {}, comp = {}
    for (const u of users || []) {
      if (u.num && u.name) names[String(u.num)] = u.name
      comp[String(u.num)] = companyForQueue(u.tr1)
    }
    userNames.value = names
    userCompany.value = comp
  } catch (e) { /* ixtiyoriy */ }
}

async function loadCalls() {
  allCalls.value = await api.data('', fromUnix.value, toUnix.value)
}

async function loadHidden() {
  try { hiddenExts.value = new Set((await api.hidden()) || []) } catch { /* ixtiyoriy */ }
}

async function refresh() {
  try {
    await Promise.all([loadCalls(), loadFifo(), loadUsers(), loadHidden(), loadStats()])
    error.value = ''
  } catch (e) {
    error.value = e.message || 'Yuklashda xato'
  } finally {
    loading.value = false
  }
}

async function initWS() {
  try {
    const [cfg, keys] = await Promise.all([api.config(), api.keys()])
    if (!cfg.domain || !keys.auth_key) { wsState.value = 'polling'; return }
    ws = new WebSocket(`wss://${cfg.domain}:${cfg.wsPort || 3342}/?key=${keys.auth_key}`)
    ws.onopen = () => {
      wsState.value = 'live'
      ws.send(JSON.stringify({
        command: 'subscribe', reqId: 'mon-' + Date.now(),
        data: { eventGroups: ['call_status', 'call_start', 'call_end'] },
      }))
    }
    ws.onmessage = (ev) => { try { handleWS(JSON.parse(ev.data)) } catch {} }
    ws.onclose = () => { wsState.value = 'polling'; scheduleReconnect() }
    ws.onerror = () => { try { ws.close() } catch {} }
  } catch (e) { wsState.value = 'polling'; scheduleReconnect() }
}

function scheduleReconnect() {
  clearTimeout(reconnectTimer)
  reconnectTimer = setTimeout(initWS, 5000)
}

function handleWS(msg) {
  const d = msg?.data || msg
  const type = msg?.event || msg?.type || d?.type
  const exts = []
  for (const k of ['caller', 'callee', 'src', 'dst', 'number', 'destination_number', 'caller_id_number']) {
    if (isExtension(d?.[k])) exts.push(String(d[k]))
  }
  if (!exts.length) return
  const next = { ...liveStatus.value }
  if (type === 'call_end' || d?.status === 'hangup') {
    for (const e of exts) delete next[e]
  } else if (type === 'call_start' || d?.status === 'ringing') {
    for (const e of exts) next[e] = 'ringing'
  } else if (d?.status === 'answered' || type === 'call_status') {
    for (const e of exts) next[e] = 'talking'
  }
  liveStatus.value = next
}

onMounted(async () => {
  await refresh()
  initWS()
  pollTimer = setInterval(refresh, 15000)
  clockTimer = setInterval(() => (now.value = new Date()), 1000)
})
onUnmounted(() => {
  clearInterval(pollTimer); clearInterval(clockTimer); clearTimeout(reconnectTimer)
  try { ws && ws.close() } catch {}
})
</script>

<template>
  <div>
    <header class="hdr">
      <div class="filter">
        <button v-for="c in companies" :key="c.id" class="filter__btn"
                :class="{ active: company === c.id }" @click="company = c.id"
                :style="company === c.id ? { '--c': c.color } : {}">
          <i v-if="c.id" :style="{ background: c.color }"></i>{{ c.id ? c.name : t('common.all') }}
        </button>
      </div>
      <div class="hdr__right">
        <div class="conn" :class="wsState">
          <i></i>
          <span>{{ wsState === 'live' ? t('dash.live') : wsState === 'polling' ? t('dash.updating') : t('dash.connecting') }}</span>
        </div>
        <div class="clock">
          <div class="clock__time mono">{{ clock }}</div>
          <div class="clock__date">{{ dateLabel }}</div>
        </div>
      </div>
    </header>

    <div v-if="error" class="banner">⚠️ {{ error }}</div>

    <!-- Sana / vaqt filtri -->
    <div class="daterow card">
      <div class="presets">
        <button v-for="p in PRESETS" :key="p.id" class="preset" :class="{ active: preset === p.id }" @click="applyPreset(p.id)">{{ t(p.key) }}</button>
      </div>
      <div class="dr-custom">
        <input type="datetime-local" v-model="fromInput" @change="applyCustom" />
        <span class="dr-sep">→</span>
        <input type="datetime-local" v-model="toInput" @change="applyCustom" />
      </div>
      <div class="dr-label mono">{{ rangeLabel }}</div>
    </div>

    <section class="kpis">
      <div class="kpi-wrap" style="animation-delay:0ms"><StatCard :label="t('dash.kpi.total')" :value="kpi.total" :icon="ICONS.phone" accent="var(--accent)" /></div>
      <div class="kpi-wrap" style="animation-delay:70ms"><StatCard :label="t('dash.kpi.in')" :value="kpi.inc" :icon="ICONS.in" accent="var(--green)" /></div>
      <div class="kpi-wrap" style="animation-delay:140ms"><StatCard :label="t('dash.kpi.out')" :value="kpi.out" :icon="ICONS.out" accent="var(--accent-2)" /></div>
      <div class="kpi-wrap" style="animation-delay:210ms"><StatCard :label="t('dash.kpi.avg')" :value="kpi.avgTalk" :format="fmtDuration" :icon="ICONS.clock" accent="var(--amber)" /></div>
      <div class="kpi-wrap" style="animation-delay:280ms"><StatCard :label="t('dash.kpi.active')" :value="activeOps" :icon="ICONS.users" accent="var(--blue)" /></div>
    </section>

    <StatsBlock :stats="stats" :names="userNames" />

    <div class="grid">
      <main class="col">
        <div class="card panel">
          <div class="panel__head">
            <h2>{{ t('dash.hourly') }}</h2>
            <div class="legend">
              <span><i style="background:#6d5efc"></i>{{ t('dash.incoming') }}</span>
              <span><i style="background:#14b8c4"></i>{{ t('dash.outgoing') }}</span>
            </div>
          </div>
          <HourlyChart :calls="calls" />
        </div>

        <div class="card panel">
          <div class="panel__head">
            <h2>{{ t('dash.operators') }} <span class="count">{{ operators.length }}</span></h2>
            <div class="legend">
              <span><i style="background:var(--green)"></i>{{ t('tv.onLine') }}</span>
              <span><i style="background:var(--blue)"></i>{{ t('tv.talking') }}</span>
              <span><i style="background:var(--amber)"></i>{{ t('tv.ringing') }}</span>
            </div>
          </div>
          <div v-if="loading" class="loading"><i class="spin"></i></div>
          <div v-else-if="!operators.length" class="loading">—</div>
          <div v-else class="ops">
            <TransitionGroup name="list">
              <div v-for="(op, i) in operators" :key="op.ext" class="op-wrap"
                   :style="{ animationDelay: Math.min(i * 35, 600) + 'ms' }">
                <OperatorCard :op="op" />
              </div>
            </TransitionGroup>
          </div>
        </div>
      </main>

      <aside class="card panel side">
        <div class="panel__head">
          <h2>{{ t('dash.recent') }}</h2>
          <span class="count">{{ recentCalls.length }}</span>
        </div>
        <CallsFeed :calls="recentCalls" />
      </aside>
    </div>

    <footer class="foot">monitoring.sddev.uz · OnlinePBX · {{ todayStr() }}</footer>
  </div>
</template>

<style scoped>
.hdr { display: flex; justify-content: space-between; align-items: center; margin: 18px 0 24px; }

.filter { display: flex; gap: 6px; background: var(--surface); padding: 5px; border-radius: 13px; border: 1px solid var(--border); }
.filter__btn {
  display: flex; align-items: center; gap: 7px;
  padding: 8px 16px; border-radius: 9px;
  font-size: 13px; font-weight: 600; cursor: pointer;
  background: transparent; border: none; color: var(--text-dim);
  transition: all 0.2s;
}
.filter__btn i { width: 8px; height: 8px; border-radius: 50%; }
.filter__btn:hover { color: var(--text); }
.filter__btn.active { background: var(--surface-2); color: var(--text); box-shadow: 0 2px 8px rgba(0,0,0,0.3); }

.hdr__right { display: flex; align-items: center; gap: 22px; }
.conn { display: flex; align-items: center; gap: 8px; font-size: 12.5px; font-weight: 600;
  padding: 8px 14px; border-radius: 999px; border: 1px solid var(--border); background: var(--surface); }
.conn i { width: 8px; height: 8px; border-radius: 50%; background: var(--gray); }
.conn.live i { background: var(--green); box-shadow: 0 0 0 4px rgba(52,211,153,0.18); animation: pulse-dot 1.6s infinite; }
.conn.live { color: var(--green); }
.conn.polling i { background: var(--amber); animation: pulse-dot 1.6s infinite; }
.conn.connecting i { background: var(--accent-2); }
.clock { text-align: right; }
.clock__time { font-size: 19px; font-weight: 700; }
.clock__date { font-size: 11.5px; color: var(--text-dim); text-transform: capitalize; }

.banner { background: rgba(248,113,113,0.12); border: 1px solid rgba(248,113,113,0.3);
  color: #fca5a5; padding: 12px 16px; border-radius: var(--radius-sm); margin-bottom: 20px; font-size: 13px; }

.daterow { display: flex; align-items: center; gap: 18px; flex-wrap: wrap; padding: 12px 16px; margin-bottom: 18px; }
.presets { display: flex; gap: 5px; }
.preset { background: var(--surface-2); color: var(--text-dim); padding: 8px 15px; font-size: 12.5px; border: 1px solid var(--border); }
.preset:hover { transform: none; box-shadow: none; color: var(--text); }
.preset.active { background: var(--grad); color: #fff; border-color: transparent; }
.dr-custom { display: flex; align-items: center; gap: 8px; }
.dr-custom input { padding: 8px 11px; font-size: 12.5px; }
.dr-sep { color: var(--text-faint); }
.dr-label { margin-left: auto; font-size: 12.5px; color: var(--text-dim); font-weight: 600; }

.kpis { display: grid; grid-template-columns: repeat(5, 1fr); gap: 16px; margin-bottom: 22px; }
.kpi-wrap { animation: fade-up 0.6s both; }

.grid { display: grid; grid-template-columns: 1fr 380px; gap: 22px; align-items: start; }
.col { display: flex; flex-direction: column; gap: 22px; }

.panel { padding: 22px; animation: fade-up 0.5s both; }
.panel__head { display: flex; justify-content: space-between; align-items: center; margin-bottom: 18px; }
.panel__head h2 { font-size: 15px; font-weight: 700; display: flex; align-items: center; gap: 10px; }
.count { font-size: 12px; font-weight: 600; color: var(--text-dim);
  background: var(--surface-2); padding: 2px 9px; border-radius: 999px; }
.legend { display: flex; gap: 14px; font-size: 11.5px; color: var(--text-dim); font-weight: 500; }
.legend span { display: flex; align-items: center; gap: 6px; }
.legend i { width: 9px; height: 9px; border-radius: 3px; }

.ops { display: grid; grid-template-columns: repeat(auto-fill, minmax(215px, 1fr)); gap: 14px; }
.op-wrap { animation: fade-up 0.5s both; }

.loading { display: flex; align-items: center; justify-content: center; gap: 10px;
  padding: 50px; color: var(--text-dim); font-size: 13px; }
.spin { width: 18px; height: 18px; border: 2.5px solid var(--border-strong);
  border-top-color: var(--accent); border-radius: 50%; animation: spin 0.8s linear infinite; }

.side { position: sticky; top: 20px; }
.foot { text-align: center; color: var(--text-faint); font-size: 12px; margin-top: 36px; }

@media (max-width: 1080px) {
  .kpis { grid-template-columns: repeat(2, 1fr); }
  .grid { grid-template-columns: 1fr; }
  .side { position: static; }
  .hdr { flex-direction: column; gap: 16px; align-items: stretch; }
}
</style>
