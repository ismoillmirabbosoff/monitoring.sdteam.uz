<script setup>
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { GridLayout, GridItem } from 'grid-layout-plus'
import { api, companyForQueue, companyName, COMPANIES } from '../api.js'
import { t } from '../i18n.js'

const statusMap = ref({})     // ext -> holat (bridge'dan: online/offline/talking/ringing/dnd)
const bridgeConnected = ref(false)
const names = ref({})         // ext -> ism
const compMap = ref({})       // ext -> kompaniya
const opStats = ref({})       // ext -> {incoming, outgoing, survey_unfilled, servers}
const hidden = ref(new Set())
const company = ref('')
const now = ref(new Date())
const theme = ref(localStorage.getItem('theme') || 'light')
function toggleTheme() {
  theme.value = theme.value === 'dark' ? 'light' : 'dark'
  localStorage.setItem('theme', theme.value)
  document.documentElement.setAttribute('data-theme', theme.value)
}
let pollTimer = null, liveTimer = null, clockTimer = null

const companies = COMPANIES

// Status: online(green) / offline(red) / talking(blue) / ringing(amber) / dnd(orange) / unknown(grey)
const STATUS = {
  online:  { key: 'tv.onLine',  color: '#15803d' },
  offline: { key: 'tv.offLine', color: '#dc2626' },
  talking: { key: 'tv.talking', color: '#1d4ed8' },
  ringing: { key: 'tv.ringing', color: '#b45309' },
  dnd:     { key: 'tv.dnd',     color: '#6d28d9' },
  unknown: { key: 'tv.unknown', color: '#94a3b8' },
}

const operators = computed(() => {
  const exts = new Set([...Object.keys(statusMap.value), ...Object.keys(names.value)])
  let list = [...exts]
    .filter((e) => !hidden.value.has(e))
    .map((ext) => {
      const status = statusMap.value[ext] || 'unknown'
      const st = opStats.value[ext] || {}
      return {
        ext,
        name: names.value[ext] || `Operator ${ext}`,
        company: compMap.value[ext] || '',
        status,
        incoming: st.incoming || 0,
        outgoing: st.outgoing || 0,
        missed: st.missed || 0,
        unfilled: st.survey_unfilled || 0,
        servers: st.servers || 0,
      }
    })
  if (company.value) list = list.filter((o) => o.company === company.value)
  const rank = { talking: 0, ringing: 1, dnd: 2, online: 3, unknown: 4, offline: 5 }
  list.sort((a, b) => rank[a.status] - rank[b.status] || Number(a.ext) - Number(b.ext))
  return list
})

// ---- Grafana uslubidagi grid (surish + o'lchamini o'zgartirish) ----
const GRID_COLS = 12, DEF_W = 3, DEF_H = 2
const editable = ref(false)
const layout = ref([])              // [{i:ext, x, y, w, h}]
const opByExt = computed(() => Object.fromEntries(operators.value.map((o) => [o.ext, o])))
// Ekranda ko'rinadigan grid katakchalari (layout + operator ma'lumoti)
const gridItems = computed(() =>
  layout.value.map((it) => ({ ...it, op: opByExt.value[it.i] })).filter((c) => c.op)
)

function loadPos() { try { return JSON.parse(localStorage.getItem('tv_layout') || '{}') } catch { return {} } }
let savedPos = loadPos()
function savePos() { localStorage.setItem('tv_layout', JSON.stringify(savedPos)) }

// Ko'rinadigan operatorlar bilan layoutni moslashtiradi (yangi qo'shadi, yo'qolganini olib tashlaydi).
function reconcile() {
  const visible = operators.value.map((o) => o.ext)
  const visibleSet = new Set(visible)
  const next = layout.value.filter((it) => visibleSet.has(it.i))
  const have = new Set(next.map((it) => it.i))
  let idx = next.length
  for (const ext of visible) {
    if (have.has(ext)) continue
    const p = savedPos[ext]
    if (p) next.push({ i: ext, x: p.x, y: p.y, w: p.w, h: p.h })
    else {
      next.push({ i: ext, x: (idx % 4) * DEF_W, y: Math.floor(idx / 4) * DEF_H, w: DEF_W, h: DEF_H })
      idx++
    }
  }
  layout.value = next
}
watch(operators, reconcile, { immediate: true })

function onLayoutUpdated(newLayout) {
  for (const it of newLayout) savedPos[it.i] = { x: it.x, y: it.y, w: it.w, h: it.h }
  savePos()
}
function resetLayout() {
  savedPos = {}
  savePos()
  layout.value = []
  reconcile()
}

const counts = computed(() => {
  const c = { online: 0, offline: 0, talking: 0, ringing: 0, dnd: 0, unknown: 0 }
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
async function refresh() { await Promise.all([loadUsers(), loadHidden(), loadStats()]) }

// Jonli holat — backend bridge snapshot'ini olamiz (bridge OnlinePBX WS'ni 24/7 tinglaydi).
async function loadLiveState() {
  try {
    const r = await api.liveState()
    statusMap.value = r.operators || {}
    bridgeConnected.value = !!r.connected
  } catch {}
}

function initials(op) {
  const parts = (op.name || '').trim().split(/\s+/).filter(Boolean)
  if (parts.length >= 2 && !/^operator$/i.test(parts[0])) return (parts[0][0] + parts[1][0]).toUpperCase()
  if (parts.length === 1 && !/^operator$/i.test(parts[0])) return parts[0].slice(0, 2).toUpperCase()
  return String(op.ext).slice(-2)
}

onMounted(async () => {
  await Promise.all([refresh(), loadLiveState()])
  liveTimer = setInterval(loadLiveState, 2500) // jonli holat (bridge snapshot)
  pollTimer = setInterval(refresh, 10000)
  clockTimer = setInterval(() => (now.value = new Date()), 1000)
})
onUnmounted(() => {
  clearInterval(pollTimer); clearInterval(liveTimer); clearInterval(clockTimer)
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

    <div class="tv__toolbar">
      <button class="tv__editbtn" :class="{ active: editable }" @click="editable = !editable">
        {{ editable ? '🔓 Tahrir rejimi yoniq' : '🔒 Joylashuvni tahrirlash' }}
      </button>
      <template v-if="editable">
        <button class="tv__resetbtn" @click="resetLayout">↺ Tartibni tiklash</button>
        <span class="tv__hint">Kartani suring · burchakdan cho'zib kattalashtiring</span>
      </template>
    </div>

    <GridLayout
      v-if="gridItems.length"
      v-model:layout="layout"
      :col-num="GRID_COLS"
      :row-height="72"
      :margin="[16, 16]"
      :is-draggable="editable"
      :is-resizable="editable"
      :vertical-compact="true"
      @layout-updated="onLayoutUpdated"
    >
      <GridItem
        v-for="cell in gridItems"
        :key="cell.i"
        :i="cell.i"
        :x="cell.x" :y="cell.y" :w="cell.w" :h="cell.h"
        :min-w="2" :min-h="1"
      >
        <div class="item" :class="[`s-${cell.op.status}`, { 'is-edit': editable }]">
          <div class="item__top">
            <span class="item__ext mono">{{ cell.op.ext }}</span>
            <span class="item__phone" :title="t(STATUS[cell.op.status].key)">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 16.92v3a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07 19.5 19.5 0 0 1-6-6 19.79 19.79 0 0 1-3.07-8.67A2 2 0 0 1 4.11 2h3a2 2 0 0 1 2 1.72c.13.96.36 1.9.7 2.81a2 2 0 0 1-.45 2.11L8.09 9.91a16 16 0 0 0 6 6l1.27-1.27a2 2 0 0 1 2.11-.45c.91.34 1.85.57 2.81.7A2 2 0 0 1 22 16.92z"/></svg>
            </span>
          </div>
          <div class="item__name">{{ cell.op.name }}</div>
          <div class="item__st">{{ t(STATUS[cell.op.status].key) }}</div>
          <div class="item__metrics">
            <div class="m m--in" :title="t('st.inCalls')">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="16 2 16 8 22 8"/><line x1="22" y1="2" x2="16" y2="8"/><path d="M22 16.92v3a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07 19.5 19.5 0 0 1-6-6 19.79 19.79 0 0 1-3.07-8.67A2 2 0 0 1 4.11 2h3a2 2 0 0 1 2 1.72c.13.96.36 1.9.7 2.81a2 2 0 0 1-.45 2.11L8.09 9.91a16 16 0 0 0 6 6l1.27-1.27a2 2 0 0 1 2.11-.45c.91.34 1.85.57 2.81.7A2 2 0 0 1 22 16.92z"/></svg>
              <span>{{ cell.op.incoming }}</span>
            </div>
            <div class="m m--out" :title="t('st.outCalls')">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="22 8 22 2 16 2"/><line x1="16" y1="8" x2="22" y2="2"/><path d="M22 16.92v3a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07 19.5 19.5 0 0 1-6-6 19.79 19.79 0 0 1-3.07-8.67A2 2 0 0 1 4.11 2h3a2 2 0 0 1 2 1.72c.13.96.36 1.9.7 2.81a2 2 0 0 1-.45 2.11L8.09 9.91a16 16 0 0 0 6 6l1.27-1.27a2 2 0 0 1 2.11-.45c.91.34 1.85.57 2.81.7A2 2 0 0 1 22 16.92z"/></svg>
              <span>{{ cell.op.outgoing }}</span>
            </div>
            <div class="m m--miss" :class="{ bad: cell.op.missed > 0 }" :title="t('tv.missed')">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 16.92v3a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07 19.5 19.5 0 0 1-6-6 19.79 19.79 0 0 1-3.07-8.67A2 2 0 0 1 4.11 2h3a2 2 0 0 1 2 1.72c.13.96.36 1.9.7 2.81a2 2 0 0 1-.45 2.11L8.09 9.91a16 16 0 0 0 6 6l1.27-1.27a2 2 0 0 1 2.11-.45c.91.34 1.85.57 2.81.7A2 2 0 0 1 22 16.92z"/><line x1="23" y1="1" x2="17" y2="7"/><line x1="17" y1="1" x2="23" y2="7"/></svg>
              <span>{{ cell.op.missed }}</span>
            </div>
            <div class="m m--surv" :class="{ warn: cell.op.unfilled > 0 }" :title="t('tv.unfilled')">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M9 5H7a2 2 0 0 0-2 2v12a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V7a2 2 0 0 0-2-2h-2"/><rect x="9" y="3" width="6" height="4" rx="1"/><path d="M9 12h6M9 16h4"/></svg>
              <span>{{ cell.op.unfilled }}</span>
            </div>
            <div class="m m--srv" :title="t('tv.servers')">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="4" width="18" height="7" rx="2"/><rect x="3" y="13" width="18" height="7" rx="2"/><path d="M7 7.5h.01M7 16.5h.01"/></svg>
              <span>{{ cell.op.servers }}</span>
            </div>
          </div>
        </div>
      </GridItem>
    </GridLayout>
    <div v-else class="tv__empty">—</div>
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

.s-online  { --c: #15803d; }
.s-offline { --c: #dc2626; }
.s-talking { --c: #1d4ed8; }
.s-ringing { --c: #b45309; }
.s-dnd     { --c: #6d28d9; }
.s-unknown { --c: #94a3b8; }

/* Grafana uslubidagi grid toolbar */
.tv__toolbar { display: flex; align-items: center; gap: 12px; margin-bottom: 16px; }
.tv__editbtn { background: var(--surface); border: 1px solid var(--border); color: var(--text-dim);
  padding: 9px 16px; font-size: 13px; border-radius: 10px; }
.tv__editbtn:hover { color: var(--text); transform: none; box-shadow: none; }
.tv__editbtn.active { background: var(--accent-soft, rgba(109,94,252,0.16)); color: var(--accent); border-color: transparent; }
.tv__resetbtn { background: var(--surface-2); color: var(--text-dim); padding: 9px 14px; font-size: 13px; border: 1px solid var(--border); border-radius: 10px; }
.tv__resetbtn:hover { color: var(--text); transform: none; box-shadow: none; }
.tv__hint { font-size: 12.5px; color: var(--text-faint); }

/* Grid katakni to'ldiradigan kartalar — status rangi bilan to'q bo'yalgan */
.item {
  height: 100%; display: flex; flex-direction: column; overflow: hidden;
  background: color-mix(in srgb, var(--c) 14%, var(--surface));
  border: 1.5px solid color-mix(in srgb, var(--c) 45%, transparent);
  border-left: 6px solid var(--c);
  border-radius: 14px; padding: 15px 17px; box-shadow: var(--shadow); transition: box-shadow 0.2s;
}
.item.is-edit { cursor: move; }
.item:hover { box-shadow: var(--shadow-lg); }
.item__top { display: flex; align-items: center; justify-content: space-between; }
.item__ext { font-size: 15px; color: var(--text-faint); font-weight: 500; }
.item__dot { width: 11px; height: 11px; border-radius: 50%; background: var(--c); flex-shrink: 0; }
.item__phone { width: 34px; height: 34px; border-radius: 10px; display: grid; place-items: center;
  color: #fff; background: var(--c); flex-shrink: 0; box-shadow: 0 2px 8px color-mix(in srgb, var(--c) 45%, transparent); }
.item__phone svg { width: 17px; height: 17px; }
.s-talking .item__phone, .s-ringing .item__phone, .s-online .item__phone { animation: pulse-dot 1.8s infinite; }
.item__name { font-size: 17px; font-weight: 600; color: var(--text); margin-top: 12px;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.item__st { display: inline-block; align-self: flex-start; font-size: 11px; color: #fff; background: var(--c);
  text-transform: uppercase; letter-spacing: 0.04em; margin-top: 9px; font-weight: 700; padding: 3px 10px; border-radius: 6px; }
.item__metrics { display: grid; grid-template-columns: repeat(5, 1fr); gap: 5px; margin-top: auto;
  padding-top: 12px; border-top: 1px solid var(--border); }
.m { display: flex; align-items: center; justify-content: center; gap: 5px;
  font-size: 14px; font-weight: 700; color: var(--text); font-family: var(--mono); }
.m svg { width: 15px; height: 15px; flex-shrink: 0; }
.m--in { color: var(--green, #10b981); }
.m--out { color: var(--accent-2, #14b8c4); }
.m--miss { color: var(--text-faint); }
.m--miss.bad { color: #ef4444; }
.m--surv { color: var(--text-faint); }
.m--surv.warn { color: #f59e0b; }
.m--srv { color: var(--text-dim); }
.s-offline { opacity: 0.6; }
.s-offline .item__name { color: var(--text-dim); }
.s-offline .item__phone { box-shadow: none; }
.s-talking .item__dot, .s-ringing .item__dot, .s-online .item__dot { animation: pulse-dot 1.8s infinite; }

.tv__empty { text-align: center; padding: 80px; color: var(--text-faint); font-size: 18px; }

/* grid-layout-plus (bola komponent) uslublari — temaga moslash */
:deep(.vgl-item--placeholder) { background: var(--accent, #6d5efc); opacity: 0.18; border-radius: 16px; }
:deep(.vgl-item__resizer) { z-index: 3; }
:deep(.vgl-item--dragging) { z-index: 5; }
:deep(.vgl-item--dragging .item) { box-shadow: var(--shadow-lg); border-color: var(--accent, #6d5efc); }
</style>

<!-- grid-layout-plus asosiy CSS (paket alohida css fayl bermaydi — shu bois shu yerda) -->
<style>
.vgl-layout { --vgl-placeholder-bg: var(--accent, #6d5efc); --vgl-placeholder-opacity: 18%; --vgl-placeholder-z-index: 2; --vgl-item-resizing-z-index: 3; --vgl-item-resizing-opacity: 60%; --vgl-item-dragging-z-index: 5; --vgl-item-dragging-opacity: 100%; --vgl-resizer-size: 14px; --vgl-resizer-border-color: var(--text-faint, #888); --vgl-resizer-border-width: 2px; position: relative; box-sizing: border-box; transition: height .2s ease; }
.vgl-item { position: absolute; box-sizing: border-box; transition: .2s ease; transition-property: left, top, right; }
.vgl-item--placeholder { z-index: var(--vgl-placeholder-z-index, 2); user-select: none; background-color: var(--vgl-placeholder-bg, red); opacity: var(--vgl-placeholder-opacity, 20%); transition-duration: .1s; border-radius: 16px; }
.vgl-item--no-touch { touch-action: none; }
.vgl-item--transform { right: auto; left: 0; transition-property: transform; }
.vgl-item--resizing { z-index: var(--vgl-item-resizing-z-index, 3); user-select: none; opacity: var(--vgl-item-resizing-opacity, 60%); }
.vgl-item--dragging { z-index: var(--vgl-item-dragging-z-index, 3); user-select: none; opacity: var(--vgl-item-dragging-opacity, 100%); transition: none; }
.vgl-item__resizer { position: absolute; right: 0; bottom: 0; box-sizing: border-box; width: var(--vgl-resizer-size); height: var(--vgl-resizer-size); cursor: se-resize; }
.vgl-item__resizer::before { position: absolute; top: 0; right: 3px; bottom: 3px; left: 0; content: ""; border: 0 solid var(--vgl-resizer-border-color); border-right-width: var(--vgl-resizer-border-width); border-bottom-width: var(--vgl-resizer-border-width); }
</style>
