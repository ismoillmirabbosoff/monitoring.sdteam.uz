<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { api, parseFifoUsers, isExtension, companyForQueue, companyName, COMPANIES } from '../api.js'
import { t } from '../i18n.js'

const fifoOnline = ref({})    // ext -> online bool
const names = ref({})         // ext -> ism
const compMap = ref({})       // ext -> kompaniya
const live = ref({})          // ext -> 'talking'|'ringing'|'dnd'
const opStats = ref({})       // ext -> {incoming, outgoing, survey_unfilled, servers}
const hidden = ref(new Set())
const company = ref('')
const now = ref(new Date())
const wsState = ref('connecting')
const theme = ref(localStorage.getItem('theme') || 'light')
function toggleTheme() {
  theme.value = theme.value === 'dark' ? 'light' : 'dark'
  localStorage.setItem('theme', theme.value)
  document.documentElement.setAttribute('data-theme', theme.value)
}
let ws = null, pollTimer = null, clockTimer = null, reconnectTimer = null

const companies = COMPANIES

// Status: online(green) / offline(red) / talking(blue) / ringing(amber) / dnd(orange)
const STATUS = {
  online:  { key: 'tv.onLine',  color: '#10b981' },
  offline: { key: 'tv.offLine', color: '#ef4444' },
  talking: { key: 'tv.talking', color: '#3b82f6' },
  ringing: { key: 'tv.ringing', color: '#f59e0b' },
  dnd:     { key: 'tv.dnd',     color: '#f97316' },
}

const operators = computed(() => {
  const exts = new Set([...Object.keys(fifoOnline.value), ...Object.keys(names.value)])
  let list = [...exts]
    .filter((e) => !hidden.value.has(e))
    .map((ext) => {
      let status = fifoOnline.value[ext] ? 'online' : 'offline'
      if (live.value[ext]) status = live.value[ext]
      const st = opStats.value[ext] || {}
      return {
        ext,
        name: names.value[ext] || `Operator ${ext}`,
        company: compMap.value[ext] || '',
        status,
        incoming: st.incoming || 0,
        outgoing: st.outgoing || 0,
        unfilled: st.survey_unfilled || 0,
        servers: st.servers || 0,
      }
    })
  if (company.value) list = list.filter((o) => o.company === company.value)
  const rank = { talking: 0, ringing: 1, dnd: 2, online: 3, offline: 4 }
  list.sort((a, b) => rank[a.status] - rank[b.status] || Number(a.ext) - Number(b.ext))
  return list
})

const counts = computed(() => {
  const c = { online: 0, offline: 0, talking: 0, ringing: 0, dnd: 0 }
  for (const o of operators.value) c[o.status]++
  return c
})
// Kanban ustunlari (holat bo'yicha)
const COLS = [
  { status: 'talking', key: 'tv.talking', color: '#3b82f6' },
  { status: 'ringing', key: 'tv.ringing', color: '#f59e0b' },
  { status: 'dnd', key: 'tv.dnd', color: '#f97316' },
  { status: 'online', key: 'tv.onLine', color: '#10b981' },
  { status: 'offline', key: 'tv.offLine', color: '#f43f5e' },
]
const columns = computed(() => COLS.map((c) => ({ ...c, ops: operators.value.filter((o) => o.status === c.status) })))

const clock = computed(() => now.value.toLocaleTimeString('ru-RU'))
const dateLabel = computed(() => now.value.toLocaleDateString('uz-UZ', { weekday: 'long', day: 'numeric', month: 'long', year: 'numeric' }))

async function loadFifo() {
  try {
    const r = await api.fifo()
    const map = {}
    for (const q of r.data || []) for (const u of parseFifoUsers(q.users)) map[u.ext] = u.online || map[u.ext]
    fifoOnline.value = map
  } catch {}
}
async function loadUsers() {
  try {
    const users = await api.users()
    const nm = {}, cm = {}
    for (const u of users || []) { if (u.num) { nm[String(u.num)] = u.name; cm[String(u.num)] = companyForQueue(u.tr1) } }
    names.value = nm; compMap.value = cm
  } catch {}
}
async function loadHidden() {
  try { hidden.value = new Set((await api.hidden()) || []) } catch {}
}
async function loadStats() {
  try {
    const s = await api.stats()
    const map = {}
    for (const o of s.operators || []) map[String(o.ext)] = o
    opStats.value = map
  } catch {}
}
async function refresh() { await Promise.all([loadFifo(), loadUsers(), loadHidden(), loadStats()]) }

async function initWS() {
  try {
    const [cfg, keys] = await Promise.all([api.config(), api.keys()])
    if (!cfg.domain || !keys.auth_key) { wsState.value = 'offline'; return }
    ws = new WebSocket(`wss://${cfg.domain}:${cfg.wsPort || 3342}/?key=${keys.auth_key}`)
    ws.onopen = () => {
      wsState.value = 'live'
      ws.send(JSON.stringify({ command: 'subscribe', reqId: 'tv-' + Date.now(),
        data: { eventGroups: ['call_status', 'call_start', 'call_end', 'user_status', 'registration'] } }))
    }
    ws.onmessage = (ev) => { try { handleWS(JSON.parse(ev.data)) } catch {} }
    ws.onclose = () => { wsState.value = 'offline'; clearTimeout(reconnectTimer); reconnectTimer = setTimeout(initWS, 5000) }
    ws.onerror = () => { try { ws.close() } catch {} }
  } catch { wsState.value = 'offline'; reconnectTimer = setTimeout(initWS, 5000) }
}

function initials(op) {
  const parts = (op.name || '').trim().split(/\s+/).filter(Boolean)
  if (parts.length >= 2 && !/^operator$/i.test(parts[0])) return (parts[0][0] + parts[1][0]).toUpperCase()
  if (parts.length === 1 && !/^operator$/i.test(parts[0])) return parts[0].slice(0, 2).toUpperCase()
  return String(op.ext).slice(-2)
}

function handleWS(msg) {
  const d = msg?.data || msg
  const type = msg?.event || msg?.type || d?.type
  const st = (d?.status || '').toLowerCase()
  const exts = []
  for (const k of ['caller', 'callee', 'src', 'dst', 'number', 'user', 'extension', 'destination_number', 'caller_id_number'])
    if (isExtension(d?.[k])) exts.push(String(d[k]))
  if (!exts.length) return
  const next = { ...live.value }
  if (type === 'call_end' || st === 'hangup' || st === 'unregistered' || st === 'unregister') {
    for (const e of exts) delete next[e]
  } else if (st === 'dnd') {
    for (const e of exts) next[e] = 'dnd'
  } else if (st === 'answered' || type === 'call_status') {
    for (const e of exts) next[e] = 'talking'
  } else if (type === 'call_start' || st === 'ringing') {
    for (const e of exts) next[e] = 'ringing'
  } else if (st === 'registered' || st === 'register') {
    for (const e of exts) delete next[e]
  }
  live.value = next
}

onMounted(async () => {
  await refresh()
  initWS()
  pollTimer = setInterval(refresh, 10000)
  clockTimer = setInterval(() => (now.value = new Date()), 1000)
})
onUnmounted(() => {
  clearInterval(pollTimer); clearInterval(clockTimer); clearTimeout(reconnectTimer)
  try { ws && ws.close() } catch {}
})
</script>

<template>
  <div class="tv">
    <header class="tv__top">
      <div class="tv__brand">
        <div class="tv__logo">
          <svg viewBox="0 0 32 32"><defs><linearGradient id="tlg" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stop-color="#6d5efc"/><stop offset="1" stop-color="#14b8c4"/></linearGradient></defs><rect width="32" height="32" rx="9" fill="url(#tlg)"/><path d="M9 20c0-3.9 3.1-7 7-7s7 3.1 7 7" fill="none" stroke="#fff" stroke-width="2.4" stroke-linecap="round"/><circle cx="16" cy="22" r="2.2" fill="#fff"/></svg>
        </div>
        <div>
          <h1>{{ t('tv.title') }}</h1>
          <p>{{ dateLabel }}</p>
        </div>
      </div>

      <div class="tv__filter">
        <button v-for="c in companies" :key="c.id" :class="{ active: company === c.id }" @click="company = c.id">
          {{ c.id ? c.name : t('common.all') }}
        </button>
      </div>

      <div class="tv__right">
        <button class="tv__theme" @click="toggleTheme" :title="t('common.theme.' + (theme==='dark'?'light':'dark'))">
          <svg v-if="theme === 'dark'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="5"/><path d="M12 1v2M12 21v2M4.2 4.2l1.4 1.4M18.4 18.4l1.4 1.4M1 12h2M21 12h2M4.2 19.8l1.4-1.4M18.4 5.6l1.4-1.4"/></svg>
          <svg v-else viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/></svg>
        </button>
        <div class="tv__clock mono">{{ clock }}</div>
      </div>
    </header>

    <div class="tv__list">
      <TransitionGroup name="list">
        <div v-for="op in operators" :key="op.ext" class="item" :class="`s-${op.status}`">
          <div class="item__top">
            <span class="item__ext mono">{{ op.ext }}</span>
            <span class="item__dot"></span>
          </div>
          <div class="item__name">{{ op.name }}</div>
          <div class="item__st">{{ t(STATUS[op.status].key) }}</div>
          <div class="item__metrics">
            <div class="m m--in" :title="t('st.inCalls')">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M17 7 7 17M7 17h8M7 17V9"/></svg>
              <span>{{ op.incoming }}</span>
            </div>
            <div class="m m--out" :title="t('st.outCalls')">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M7 17 17 7M17 7H9M17 7v8"/></svg>
              <span>{{ op.outgoing }}</span>
            </div>
            <div class="m m--surv" :class="{ warn: op.unfilled > 0 }" :title="t('tv.unfilled')">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M9 5H7a2 2 0 0 0-2 2v12a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V7a2 2 0 0 0-2-2h-2"/><rect x="9" y="3" width="6" height="4" rx="1"/><path d="M9 12h6M9 16h4"/></svg>
              <span>{{ op.unfilled }}</span>
            </div>
            <div class="m m--srv" :title="t('tv.servers')">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="4" width="18" height="7" rx="2"/><rect x="3" y="13" width="18" height="7" rx="2"/><path d="M7 7.5h.01M7 16.5h.01"/></svg>
              <span>{{ op.servers }}</span>
            </div>
          </div>
        </div>
      </TransitionGroup>
      <div v-if="!operators.length" class="tv__empty">—</div>
    </div>
  </div>
</template>

<style scoped>
.tv {
  min-height: 100vh; padding: 28px 38px;
  background: var(--bg);
  color: var(--text); font-family: var(--font);
}
.tv__top { display: flex; align-items: center; justify-content: space-between; gap: 24px; margin-bottom: 30px;
  padding-bottom: 22px; border-bottom: 1px solid var(--border); }
.tv__brand { display: flex; align-items: center; gap: 14px; }
.tv__logo svg { width: 40px; height: 40px; }
.tv__brand h1 { font-size: 22px; font-weight: 700; letter-spacing: -0.01em; }
.tv__brand p { font-size: 13px; color: var(--text-faint); text-transform: capitalize; margin-top: 2px; }

.tv__filter { display: flex; gap: 4px; }
.tv__filter button { background: transparent; color: var(--text-faint); padding: 8px 16px; font-size: 14px; border-radius: 9px; }
.tv__filter button:hover { transform: none; box-shadow: none; color: var(--text); }
.tv__filter button.active { background: var(--surface-2); color: var(--text); }

.tv__right { display: flex; align-items: center; gap: 14px; }
.tv__theme { width: 40px; height: 40px; padding: 0; background: transparent; border: 1px solid var(--border); color: var(--text-faint); }
.tv__theme:hover { color: var(--text); transform: none; box-shadow: none; }
.tv__theme svg { width: 18px; height: 18px; }
.tv__clock { font-size: 30px; font-weight: 700; letter-spacing: 0.01em; color: var(--text); }

.s-online  { --c: #10b981; }
.s-offline { --c: #f43f5e; }
.s-talking { --c: #3b82f6; }
.s-ringing { --c: #f59e0b; }
.s-dnd     { --c: #f97316; }

/* Minimalist kartalar */
.tv__list { display: grid; grid-template-columns: repeat(auto-fill, minmax(200px, 1fr)); gap: 16px; }
.item {
  background: var(--surface); border: 1px solid var(--border); border-radius: 16px;
  padding: 18px 20px; box-shadow: var(--shadow); animation: fade-up 0.35s both;
  transition: transform 0.2s, box-shadow 0.2s;
}
.item:hover { transform: translateY(-3px); box-shadow: var(--shadow-lg); }
.item__top { display: flex; align-items: center; justify-content: space-between; }
.item__ext { font-size: 15px; color: var(--text-faint); font-weight: 500; }
.item__dot { width: 11px; height: 11px; border-radius: 50%; background: var(--c); flex-shrink: 0; }
.item__name { font-size: 17px; font-weight: 600; color: var(--text); margin-top: 12px;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.item__st { font-size: 11.5px; color: var(--c); text-transform: uppercase; letter-spacing: 0.05em; margin-top: 8px; font-weight: 600; }
.item__metrics { display: grid; grid-template-columns: repeat(4, 1fr); gap: 6px; margin-top: 14px;
  padding-top: 12px; border-top: 1px solid var(--border); }
.m { display: flex; align-items: center; justify-content: center; gap: 5px;
  font-size: 14px; font-weight: 700; color: var(--text); font-family: var(--mono); }
.m svg { width: 15px; height: 15px; flex-shrink: 0; }
.m--in { color: var(--green, #10b981); }
.m--out { color: var(--accent-2, #14b8c4); }
.m--surv { color: var(--text-faint); }
.m--surv.warn { color: #f59e0b; }
.m--srv { color: var(--text-dim); }
.s-offline { opacity: 0.7; }
.s-offline .item__name { color: var(--text-dim); }
.s-talking .item__dot, .s-ringing .item__dot, .s-online .item__dot { animation: pulse-dot 1.8s infinite; }

.tv__empty { grid-column: 1/-1; text-align: center; padding: 80px; color: var(--text-faint); font-size: 18px; }
</style>
