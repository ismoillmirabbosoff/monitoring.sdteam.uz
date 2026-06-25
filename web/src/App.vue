<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import StatCard from './components/StatCard.vue'
import OperatorCard from './components/OperatorCard.vue'
import HourlyChart from './components/HourlyChart.vue'
import CallsFeed from './components/CallsFeed.vue'
import { api, todayStr, parseFifoUsers, isExtension, fmtDuration } from './api.js'

const ICONS = {
  phone: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 16.92v3a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07 19.5 19.5 0 0 1-6-6 19.79 19.79 0 0 1-3.07-8.67A2 2 0 0 1 4.11 2h3a2 2 0 0 1 2 1.72c.13.96.36 1.9.7 2.81a2 2 0 0 1-.45 2.11L8.09 9.91a16 16 0 0 0 6 6l1.27-1.27a2 2 0 0 1 2.11-.45c.91.34 1.85.57 2.81.7A2 2 0 0 1 22 16.92z"/></svg>',
  in: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M17 7 7 17M7 17h8M7 17V9"/></svg>',
  out: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M7 17 17 7M17 7H9M17 7v8"/></svg>',
  clock: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="9"/><path d="M12 7v5l3 2"/></svg>',
  users: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87M16 3.13a4 4 0 0 1 0 7.75"/></svg>',
}

const calls = ref([])
const fifoOnline = ref({})       // ext -> online bool
const userNames = ref({})        // ext -> operator ismi
const liveStatus = ref({})       // ext -> 'ringing'|'talking' (websocket'dan)
const loading = ref(true)
const error = ref('')
const wsState = ref('connecting') // connecting|live|polling
const now = ref(new Date())
let ws = null, pollTimer = null, clockTimer = null, reconnectTimer = null

// ---- Hosil bo'lgan ma'lumotlar ----

const kpi = computed(() => {
  let inc = 0, out = 0, talk = 0, answered = 0
  for (const c of calls.value) {
    if (c.direction === 'outbound') out++; else inc++
    if (c.user_talk_time > 0) { talk += c.user_talk_time; answered++ }
  }
  return { total: calls.value.length, inc, out, avgTalk: answered ? talk / answered : 0 }
})

// Har extension bo'yicha statistika (bigData'dan)
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

const operators = ref([])
function rebuildOperators() {
  const exts = new Set([
    ...Object.keys(fifoOnline.value),
    ...Object.keys(extStats.value),
    ...Object.keys(userNames.value),
  ])
  const list = [...exts].sort((a, b) => Number(a) - Number(b)).map((ext) => {
    const s = extStats.value[ext] || { incoming: 0, outgoing: 0, talk: 0, answered: 0 }
    let status = 'offline'
    if (fifoOnline.value[ext]) status = 'online'
    if (liveStatus.value[ext]) status = liveStatus.value[ext]
    const avgTalk = s.answered > 0 ? Math.round(s.talk / s.answered) : 0
    return {
      ext,
      name: userNames.value[ext] || `Operator ${ext}`,
      status,
      incoming: s.incoming,
      outgoing: s.outgoing,
      talk: s.talk,
      avgTalk,
    }
  })
  // jonli (ringing/talking) yuqorida, keyin online, keyin oflayn
  const rank = { ringing: 0, talking: 1, online: 2, offline: 3 }
  list.sort((a, b) => rank[a.status] - rank[b.status] || Number(a.ext) - Number(b.ext))
  operators.value = list
}

const activeOps = computed(() => operators.value.filter((o) => o.status !== 'offline').length)
const recentCalls = computed(() =>
  [...calls.value].sort((a, b) => b.start_stamp - a.start_stamp).slice(0, 40)
)
const clock = computed(() => now.value.toLocaleTimeString('ru-RU'))
const dateLabel = computed(() =>
  now.value.toLocaleDateString('uz-UZ', { weekday: 'long', day: 'numeric', month: 'long' })
)

// ---- Ma'lumot yuklash ----

async function loadFifo() {
  try {
    const r = await api.fifo()
    const map = {}
    for (const q of r.data || []) {
      for (const u of parseFifoUsers(q.users)) map[u.ext] = u.online || map[u.ext]
    }
    fifoOnline.value = map
  } catch (e) { /* fifo ixtiyoriy — xato bo'lsa o'tkazib yuboramiz */ }
}

async function loadUsers() {
  try {
    const users = await api.users()
    const map = {}
    for (const u of users || []) if (u.num && u.name) map[String(u.num)] = u.name
    userNames.value = map
  } catch (e) { /* ismlar ixtiyoriy */ }
}

async function loadCalls() {
  calls.value = await api.bigData(todayStr())
}

async function refresh() {
  try {
    await Promise.all([loadCalls(), loadFifo(), loadUsers()])
    error.value = ''
  } catch (e) {
    error.value = e.message || 'Yuklashda xato'
  } finally {
    loading.value = false
    rebuildOperators()
  }
}

// ---- WebSocket (jonli qo'ng'iroq holatlari) ----

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

// OnlinePBX hodisalaridan extension holatini yangilash (best-effort)
function handleWS(msg) {
  const d = msg?.data || msg
  const type = msg?.event || msg?.type || d?.type
  const exts = []
  for (const k of ['caller', 'callee', 'src', 'dst', 'number', 'destination_number', 'caller_id_number']) {
    const v = d?.[k]
    if (isExtension(v)) exts.push(String(v))
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
  rebuildOperators()
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
  <div class="app">
    <!-- Header -->
    <header class="hdr">
      <div class="hdr__brand">
        <div class="logo">
          <svg viewBox="0 0 32 32"><defs><linearGradient id="lg" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stop-color="#7c5cff"/><stop offset="1" stop-color="#22d3ee"/></linearGradient></defs><rect width="32" height="32" rx="9" fill="url(#lg)"/><path d="M9 20c0-3.9 3.1-7 7-7s7 3.1 7 7" fill="none" stroke="#fff" stroke-width="2.4" stroke-linecap="round"/><circle cx="16" cy="22" r="2.2" fill="#fff"/></svg>
        </div>
        <div>
          <h1>Monitoring</h1>
          <p>Call Center · Real-time</p>
        </div>
      </div>
      <div class="hdr__right">
        <div class="conn" :class="wsState">
          <i></i>
          <span>{{ wsState === 'live' ? 'Jonli ulanish' : wsState === 'polling' ? 'Yangilanmoqda' : 'Ulanmoqda…' }}</span>
        </div>
        <div class="clock">
          <div class="clock__time mono">{{ clock }}</div>
          <div class="clock__date">{{ dateLabel }}</div>
        </div>
      </div>
    </header>

    <div v-if="error" class="banner">⚠️ {{ error }}</div>

    <!-- KPI -->
    <section class="kpis">
      <div class="kpi-wrap" style="animation-delay:0ms"><StatCard label="Bugungi qo'ng'iroqlar" :value="kpi.total" :icon="ICONS.phone" accent="var(--accent)" /></div>
      <div class="kpi-wrap" style="animation-delay:70ms"><StatCard label="Kiruvchi" :value="kpi.inc" :icon="ICONS.in" accent="var(--green)" /></div>
      <div class="kpi-wrap" style="animation-delay:140ms"><StatCard label="Chiquvchi" :value="kpi.out" :icon="ICONS.out" accent="var(--accent-2)" /></div>
      <div class="kpi-wrap" style="animation-delay:210ms"><StatCard label="O'rtacha suhbat" :value="kpi.avgTalk" :format="fmtDuration" :icon="ICONS.clock" accent="var(--amber)" /></div>
      <div class="kpi-wrap" style="animation-delay:280ms"><StatCard label="Faol operatorlar" :value="activeOps" :icon="ICONS.users" accent="var(--blue)" /></div>
    </section>

    <!-- Main grid -->
    <div class="grid">
      <main class="col">
        <div class="card panel">
          <div class="panel__head">
            <h2>Soatlik faollik</h2>
            <div class="legend">
              <span><i style="background:#7c5cff"></i>Kiruvchi</span>
              <span><i style="background:#22d3ee"></i>Chiquvchi</span>
            </div>
          </div>
          <HourlyChart :calls="calls" />
        </div>

        <div class="card panel">
          <div class="panel__head">
            <h2>Operatorlar <span class="count">{{ operators.length }}</span></h2>
            <div class="legend">
              <span><i style="background:var(--green)"></i>Onlayn</span>
              <span><i style="background:var(--blue)"></i>Suhbat</span>
              <span><i style="background:var(--amber)"></i>Jiringlash</span>
            </div>
          </div>
          <div v-if="loading" class="loading"><i class="spin"></i> Yuklanmoqda…</div>
          <div v-else-if="!operators.length" class="loading">Operatorlar topilmadi</div>
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
          <h2>So'nggi qo'ng'iroqlar</h2>
          <span class="count">{{ recentCalls.length }}</span>
        </div>
        <CallsFeed :calls="recentCalls" />
      </aside>
    </div>

    <footer class="foot">monitoring.sddev.uz · OnlinePBX · {{ todayStr() }}</footer>
  </div>
</template>

<style scoped>
.app { max-width: 1400px; margin: 0 auto; padding: 28px 28px 60px; }

.hdr { display: flex; justify-content: space-between; align-items: center; margin-bottom: 28px; }
.hdr__brand { display: flex; align-items: center; gap: 14px; }
.logo svg { width: 44px; height: 44px; display: block; filter: drop-shadow(0 6px 16px rgba(124,92,255,0.4)); }
.hdr h1 { font-size: 22px; font-weight: 800; letter-spacing: -0.02em; }
.hdr p { font-size: 12.5px; color: var(--text-dim); margin-top: 1px; }
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
}
</style>
