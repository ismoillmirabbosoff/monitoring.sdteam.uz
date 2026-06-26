<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { api, COMPANIES, companyName } from '../api.js'
import ServerCard from '../components/ServerCard.vue'

const router = useRouter()
const route = useRoute()

const employees = ref([])
const servers = ref([])
const loading = ref(false)
const msg = ref('')

// Server formasi
const sf = ref({ name: '', company: 'salesdoc', employee_id: '', assigned_at: '' })

// Tanlangan kompaniyaga tegishli xodimlar (server biriktirish uchun)
const availableEmployees = computed(() =>
  employees.value.filter((e) => !sf.value.company || e.company === sf.value.company)
)
// Kompaniya o'zgarsa, mos kelmaydigan tanlangan xodimni tozalaymiz
watch(() => sf.value.company, () => {
  if (sf.value.employee_id && !availableEmployees.value.some((e) => e.id === Number(sf.value.employee_id))) {
    sf.value.employee_id = ''
  }
})

const columns = [
  { col: 1, title: '1-oy', hint: '0–1 oy ishlamoqda' },
  { col: 2, title: '2-oy', hint: '1–2 oy ishlamoqda' },
  { col: 3, title: '3-oy+', hint: '2 oydan ortiq' },
]
const byColumn = (c) => servers.value.filter((s) => s.column === c)
const companiesAll = COMPANIES

async function loadAll() {
  loading.value = true
  try {
    const [emps, srvs] = await Promise.all([api.employees(), api.servers()])
    employees.value = emps
    servers.value = srvs
  } catch (e) {
    if (e.message !== '401') { /* boshqa xato */ }
  } finally { loading.value = false }
}

async function addServer() {
  if (!sf.value.name.trim()) return
  try {
    const payload = {
      name: sf.value.name.trim(),
      company: sf.value.company,
      employee_id: sf.value.employee_id ? Number(sf.value.employee_id) : null,
      assigned_at: sf.value.assigned_at || '',
    }
    await api.addServer(payload)
    flash(`Server "${payload.name}" qo'shildi`)
    sf.value = { name: '', company: 'salesdoc', employee_id: '', assigned_at: '' }
    await loadAll()
  } catch (e) { flash('Xato: ' + e.message) }
}

async function removeServer(id) {
  try { await api.deleteServer(id); await loadAll() } catch (e) { flash('Xato: ' + e.message) }
}

// --- Serverlar ro'yxati (jadval) ---
const tab = ref(route.query.tab === 'list' ? 'list' : 'board') // 'board' | 'list'

// Server kompaniyasiga tegishli xodimlar (+ joriy biriktirilgan xodim doim ko'rinadi)
function employeesFor(comp, keepId) {
  return employees.value.filter((e) => !comp || e.company === comp || e.id === keepId)
}

async function attachEmployee(server, value) {
  try {
    const r = await api.updateServer(server.id, { employee_id: value ? Number(value) : null })
    Object.assign(server, r)
    await loadAll()
  } catch (e) { flash('Xato: ' + e.message) }
}

async function toggleActive(server) {
  try {
    const r = await api.updateServer(server.id, { active: !server.active })
    Object.assign(server, r)
    flash(`"${server.name}" ${r.active ? 'faollashtirildi' : 'nofaol qilindi'}`)
  } catch (e) { flash('Xato: ' + e.message) }
}

const sortedServers = computed(() =>
  [...servers.value].sort((a, b) => Number(b.active) - Number(a.active) || a.name.localeCompare(b.name))
)

async function toggleHidden(e) {
  try {
    const r = await api.updateEmployee(e.id, { hidden: !e.hidden })
    Object.assign(e, r)
    flash(`${e.name} ${e.hidden ? 'yashirildi' : 'ko\'rsatildi'}`)
  } catch (err) { flash('Xato: ' + err.message) }
}

function flash(t) { msg.value = t; setTimeout(() => (msg.value = ''), 3000) }
function openEmployee(id) { router.push(`/admin/employee/${id}`) }

const totalServers = computed(() => servers.value.length)
const activeServers = computed(() => servers.value.filter((s) => s.active).length)

onMounted(loadAll)
</script>

<template>
  <div class="admin">
    <div class="admin__top">
      <div>
        <h1>Serverlar boshqaruvi</h1>
        <p>{{ totalServers }} ta server · {{ activeServers }} faol · {{ employees.length }} ta xodim</p>
      </div>
    </div>

    <Transition name="page"><div v-if="msg" class="toast">{{ msg }}</div></Transition>

    <!-- Yangi server registratsiyasi -->
    <form class="card form" @submit.prevent="addServer">
      <h3><span class="form__dot" style="background:var(--accent)"></span>Yangi server qo'shish</h3>
      <div class="form__grid">
        <label class="field">
          <span>Server nomi</span>
          <input v-model="sf.name" placeholder="mehkaz.salesdoc.io" />
        </label>
        <label class="field">
          <span>Kompaniya</span>
          <select v-model="sf.company">
            <option v-for="c in companiesAll.filter(x=>x.id)" :key="c.id" :value="c.id">{{ c.name }}</option>
          </select>
        </label>
        <label class="field">
          <span>Xodim <small>({{ availableEmployees.length }} ta {{ companyName(sf.company) }})</small></span>
          <select v-model="sf.employee_id">
            <option value="">— Xodim tanlang —</option>
            <option v-for="e in availableEmployees" :key="e.id" :value="e.id">{{ e.name }}{{ e.ext ? ` (#${e.ext})` : '' }}</option>
          </select>
        </label>
        <label class="field">
          <span>Ish boshlangan sana <small>(ixtiyoriy)</small></span>
          <input v-model="sf.assigned_at" type="date" />
        </label>
      </div>
      <button type="submit">+ Ro'yxatga olish</button>
    </form>

    <!-- Tab toggle -->
    <div class="tabs">
      <button class="tab" :class="{ active: tab === 'board' }" @click="tab = 'board'">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="6" height="18"/><rect x="10" y="3" width="6" height="12"/><rect x="17" y="3" width="4" height="8"/></svg>
        Yosh bo'yicha
      </button>
      <button class="tab" :class="{ active: tab === 'list' }" @click="tab = 'list'">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="8" y1="6" x2="21" y2="6"/><line x1="8" y1="12" x2="21" y2="12"/><line x1="8" y1="18" x2="21" y2="18"/><line x1="3" y1="6" x2="3.01" y2="6"/><line x1="3" y1="12" x2="3.01" y2="12"/><line x1="3" y1="18" x2="3.01" y2="18"/></svg>
        Ro'yxat
      </button>
    </div>

    <!-- 3-kalonkali board -->
    <div v-if="tab === 'board'" class="board">
      <div v-for="c in columns" :key="c.col" class="board__col">
        <div class="board__head">
          <span class="board__num">{{ c.col }}</span>
          <div>
            <div class="board__title">{{ c.title }}</div>
            <div class="board__hint">{{ c.hint }}</div>
          </div>
          <span class="board__count">{{ byColumn(c.col).length }}</span>
        </div>
        <div class="board__list">
          <TransitionGroup name="list">
            <ServerCard v-for="(s, i) in byColumn(c.col)" :key="s.id" :server="s"
                        :style="{ animationDelay: i * 50 + 'ms' }"
                        @open="openEmployee" @remove="removeServer" />
          </TransitionGroup>
          <div v-if="!byColumn(c.col).length" class="board__empty">Bo'sh</div>
        </div>
      </div>
    </div>

    <!-- Serverlar ro'yxati (jadval) -->
    <div v-else class="card tbl-wrap">
      <table class="tbl">
        <thead>
          <tr>
            <th>Server</th>
            <th>Kompaniya</th>
            <th>Biriktirilgan xodim</th>
            <th>Yoshi</th>
            <th class="ta-c">Holat</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="s in sortedServers" :key="s.id" :class="{ inactive: !s.active }">
            <td>
              <div class="tbl__name">
                <span class="tbl__bar" :class="s.company"></span>{{ s.name }}
              </div>
            </td>
            <td><span v-if="s.company" class="pill" :class="s.company">{{ companyName(s.company) }}</span></td>
            <td>
              <select class="tbl__sel" :value="s.employee_id || ''"
                      @change="attachEmployee(s, $event.target.value)">
                <option value="">— Biriktirilmagan —</option>
                <option v-for="e in employeesFor(s.company, s.employee_id)" :key="e.id" :value="e.id">{{ e.name }}{{ e.ext ? ` (#${e.ext})` : '' }}</option>
              </select>
            </td>
            <td><span class="mono tbl__age">{{ s.days < 1 ? 'Bugun' : s.days < 30 ? s.days + ' kun' : Math.floor(s.days/30) + ' oy' }}</span>
              <span class="tbl__col">{{ s.column }}-kalonka</span></td>
            <td class="ta-c">
              <button class="switch" :class="{ on: s.active }" @click="toggleActive(s)" :title="s.active ? 'Faol' : 'Nofaol'">
                <span class="switch__thumb"></span>
              </button>
              <div class="tbl__status" :class="{ on: s.active }">{{ s.active ? 'Faol' : 'Nofaol' }}</div>
            </td>
            <td><button class="tbl__del" @click="removeServer(s.id)" title="O'chirish">×</button></td>
          </tr>
          <tr v-if="!servers.length"><td colspan="6" class="tbl__empty">Hali server yo'q</td></tr>
        </tbody>
      </table>
    </div>

    <!-- Xodimlar -->
    <h2 class="section-title">Xodimlar</h2>
    <div class="emps">
      <div v-for="e in employees" :key="e.id" class="emp card" :class="{ hidden: e.hidden }" @click="openEmployee(e.id)">
        <div class="emp__avatar">{{ e.name.slice(0,2).toUpperCase() }}</div>
        <div class="emp__info">
          <div class="emp__name">{{ e.name }}</div>
          <div class="emp__meta">
            <span v-if="e.ext" class="mono">#{{ e.ext }}</span>
            <span v-if="e.company" class="emp__company" :class="e.company">{{ companyName(e.company) }}</span>
            <span class="emp__src">{{ e.source === 'operator' ? 'Operator' : 'Qo\'lda' }}</span>
          </div>
        </div>
        <button class="emp__eye" :class="{ off: e.hidden }" @click.stop="toggleHidden(e)"
                :title="e.hidden ? 'Ko\'rsatish' : 'Yashirish (dashboard/TV)'">
          <svg v-if="!e.hidden" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/></svg>
          <svg v-else viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24M1 1l22 22"/></svg>
        </button>
        <div class="emp__count">
          <span class="mono">{{ e.server_count }}</span>
          <small>server</small>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* login */
.login-wrap { display: grid; place-items: center; min-height: 70vh; }
.login { width: 360px; padding: 36px; text-align: center; animation: fade-up 0.5s both; }
.login__icon { width: 56px; height: 56px; margin: 0 auto 18px; display: grid; place-items: center;
  border-radius: 16px; background: var(--grad-soft); color: var(--accent); }
.login__icon svg { width: 26px; height: 26px; }
.login h2 { font-size: 20px; font-weight: 800; }
.login p { color: var(--text-dim); font-size: 13px; margin: 6px 0 22px; }
.login input { width: 100%; }
.login__err { color: var(--red); font-size: 12.5px; margin-top: 10px; }
.login button { width: 100%; margin-top: 16px; }

/* admin */
.admin { animation: fade-up 0.4s both; }
.admin__top { display: flex; justify-content: space-between; align-items: flex-start; margin: 22px 0 20px; }
.admin__top h1 { font-size: 24px; font-weight: 800; letter-spacing: -0.02em; }
.admin__top p { color: var(--text-dim); font-size: 13px; margin-top: 4px; }

.toast { position: fixed; top: 22px; left: 50%; transform: translateX(-50%); z-index: 50;
  background: var(--grad); color: #fff; padding: 11px 22px; border-radius: 12px;
  font-size: 13.5px; font-weight: 600; box-shadow: 0 12px 30px -8px rgba(124,92,255,0.6); }

.form { padding: 22px; margin-bottom: 30px; }
.form h3 { font-size: 14px; font-weight: 700; display: flex; align-items: center; gap: 9px; margin-bottom: 18px; }
.form__dot { width: 9px; height: 9px; border-radius: 50%; }
.form__grid { display: grid; grid-template-columns: 2fr 1fr 1.5fr 1fr; gap: 14px; margin-bottom: 18px; }
.field { display: flex; flex-direction: column; gap: 7px; }
.field span { font-size: 12px; font-weight: 600; color: var(--text-dim); }
.field small { font-weight: 500; color: var(--text-faint); }
.field input, .field select { width: 100%; }

.section-title { font-size: 16px; font-weight: 700; margin: 8px 0 16px; }

/* Tabs */
.tabs { display: inline-flex; gap: 4px; background: var(--surface); padding: 5px;
  border-radius: 12px; border: 1px solid var(--border); margin-bottom: 22px; }
.tab { background: transparent; color: var(--text-dim); padding: 9px 16px; border-radius: 9px; font-size: 13px; }
.tab:hover { transform: none; box-shadow: none; color: var(--text); }
.tab svg { width: 16px; height: 16px; }
.tab.active { background: var(--grad); color: #fff; }

/* Jadval */
.tbl-wrap { padding: 6px 8px; margin-bottom: 34px; overflow-x: auto; }
.tbl { width: 100%; border-collapse: collapse; }
.tbl th { text-align: left; font-size: 11.5px; font-weight: 600; color: var(--text-faint);
  text-transform: uppercase; letter-spacing: 0.04em; padding: 14px 16px; }
.tbl td { padding: 12px 16px; border-top: 1px solid var(--border); font-size: 13.5px; vertical-align: middle; }
.tbl tr.inactive { opacity: 0.55; }
.ta-c { text-align: center; }
.tbl__name { display: flex; align-items: center; gap: 11px; font-weight: 600; }
.tbl__bar { width: 4px; height: 26px; border-radius: 3px; background: var(--accent); }
.tbl__bar.salesdoc { background: var(--green); }
.tbl__bar.ibox { background: var(--accent-2); }
.pill { font-size: 11px; font-weight: 600; padding: 3px 10px; border-radius: 999px; }
.pill.salesdoc { background: rgba(16,185,129,0.15); color: var(--green); }
.pill.ibox { background: rgba(20,184,196,0.15); color: var(--accent-2); }
.tbl__sel { padding: 7px 10px; font-size: 12.5px; min-width: 170px; }
.tbl__age { font-size: 13px; font-weight: 600; }
.tbl__col { display: block; font-size: 10.5px; color: var(--text-faint); margin-top: 2px; }
.tbl__del { background: none; color: var(--text-faint); font-size: 19px; padding: 0 6px; }
.tbl__del:hover { color: var(--red); transform: none; box-shadow: none; }
.tbl__empty { text-align: center; color: var(--text-faint); padding: 40px; }

/* Switch (active/inactive) */
.switch { width: 42px; height: 24px; border-radius: 999px; padding: 0; background: var(--surface-3);
  border: 1px solid var(--border-strong); position: relative; transition: background 0.25s; }
.switch:hover { transform: none; box-shadow: none; }
.switch.on { background: var(--green); border-color: transparent; }
.switch__thumb { position: absolute; top: 2px; left: 2px; width: 18px; height: 18px; border-radius: 50%;
  background: #fff; transition: transform 0.25s cubic-bezier(.2,.8,.2,1); box-shadow: 0 1px 3px rgba(0,0,0,0.3); }
.switch.on .switch__thumb { transform: translateX(18px); }
.tbl__status { font-size: 10.5px; font-weight: 600; color: var(--text-faint); margin-top: 4px; }
.tbl__status.on { color: var(--green); }

.board { display: grid; grid-template-columns: repeat(3, 1fr); gap: 16px; margin-bottom: 34px; }
.board__col { background: rgba(255,255,255,0.02); border: 1px solid var(--border); border-radius: 16px; padding: 14px; }
.board__head { display: flex; align-items: center; gap: 11px; margin-bottom: 14px; padding-bottom: 12px; border-bottom: 1px solid var(--border); }
.board__num { width: 30px; height: 30px; flex-shrink: 0; display: grid; place-items: center; border-radius: 9px;
  background: var(--grad-soft); color: var(--accent); font-weight: 800; font-size: 14px; }
.board__title { font-size: 14px; font-weight: 700; }
.board__hint { font-size: 11px; color: var(--text-faint); margin-top: 1px; }
.board__count { margin-left: auto; font-size: 12px; font-weight: 700; color: var(--text-dim);
  background: var(--surface-2); padding: 3px 10px; border-radius: 999px; }
.board__list { display: flex; flex-direction: column; gap: 10px; min-height: 60px; }
.board__empty { text-align: center; color: var(--text-faint); font-size: 12px; padding: 24px 0; }

.emps { display: grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); gap: 12px; }
.emp { display: flex; align-items: center; gap: 13px; padding: 14px 16px; cursor: pointer;
  transition: transform 0.2s, border-color 0.2s; }
.emp:hover { transform: translateY(-2px); border-color: var(--border-strong); }
.emp.hidden { opacity: 0.5; }
.emp__eye { background: var(--surface-2); border: 1px solid var(--border); color: var(--text-dim);
  width: 34px; height: 34px; padding: 0; border-radius: 9px; display: grid; place-items: center; flex-shrink: 0; }
.emp__eye:hover { transform: none; box-shadow: none; color: var(--text); background: var(--surface-3); }
.emp__eye.off { color: var(--amber); }
.emp__eye svg { width: 16px; height: 16px; }
.emp__avatar { width: 42px; height: 42px; border-radius: 12px; flex-shrink: 0; display: grid; place-items: center;
  background: var(--grad-soft); color: var(--accent); font-weight: 700; font-size: 14px; }
.emp__info { flex: 1; min-width: 0; }
.emp__name { font-size: 14px; font-weight: 600; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.emp__meta { display: flex; align-items: center; gap: 8px; margin-top: 3px; font-size: 11px; color: var(--text-faint); }
.emp__company { padding: 1px 7px; border-radius: 999px; font-weight: 600; }
.emp__company.salesdoc { background: rgba(52,211,153,0.15); color: var(--green); }
.emp__company.ibox { background: rgba(34,211,238,0.15); color: var(--accent-2); }
.emp__count { text-align: center; }
.emp__count .mono { font-size: 20px; font-weight: 700; display: block; }
.emp__count small { font-size: 10px; color: var(--text-faint); }

@media (max-width: 1080px) {
  .form__grid { grid-template-columns: 1fr 1fr; }
  .board { grid-template-columns: 1fr; }
}
</style>
