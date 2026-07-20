<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { api, COMPANIES, companyName } from '../api.js'
import { t } from '../i18n.js'

const props = defineProps({ id: [String, Number] })
const router = useRouter()

const employee = ref(null)
const servers = ref([])
const loading = ref(true)
const error = ref('')

// Server biriktirish formasi
const showForm = ref(false)
const nf = ref({ name: '', company: '', assigned_at: '' })
const msg = ref('')
const companiesAll = COMPANIES.filter((c) => c.id)

function toggleForm() {
  showForm.value = !showForm.value
  if (showForm.value) nf.value = { name: '', company: employee.value?.company || 'salesdoc', assigned_at: '' }
}

async function assignServer() {
  if (!nf.value.name.trim()) return
  try {
    await api.addServer({
      name: nf.value.name.trim(),
      company: nf.value.company,
      employee_id: Number(props.id),
      assigned_at: nf.value.assigned_at || '',
    })
    msg.value = `"${nf.value.name}" ${t('emp.assigned')}`
    setTimeout(() => (msg.value = ''), 3000)
    showForm.value = false
    await reload()
  } catch (e) { msg.value = t('common.errorPrefix') + e.message }
}

async function reload() {
  const r = await api.employee(props.id)
  employee.value = r.employee
  servers.value = r.servers
}

function fmtAge(days) {
  if (days < 1) return t('emp.startedToday')
  if (days < 30) return `${days} ${t('admin.day')}`
  const m = Math.floor(days / 30)
  const rem = days % 30
  return rem > 0 ? `${m} ${t('admin.month')} ${rem} ${t('admin.day')}` : `${m} ${t('admin.month')}`
}
function colLabel(c) { return c >= 3 ? t('admin.col3') : `${c}-${t('admin.month')}` }

const totalDays = computed(() => servers.value.reduce((a, s) => a + s.days, 0))
const activeCount = computed(() => servers.value.length)

onMounted(async () => {
  try {
    await reload()
  } catch (e) {
    error.value = e.message === '401' ? t('emp.authRequired') : t('emp.notFound')
  } finally { loading.value = false }
})
</script>

<template>
  <div class="ed">
    <RouterLink to="/admin" class="back">← {{ t('common.back') }}</RouterLink>

    <div v-if="loading" class="loading"><i class="spin"></i> {{ t('common.loading') }}</div>
    <div v-else-if="error" class="banner">{{ error }}</div>

    <template v-else-if="employee">
      <!-- Header -->
      <div class="hero card">
        <div class="hero__avatar">{{ employee.name.slice(0,2).toUpperCase() }}</div>
        <div class="hero__info">
          <h1>{{ employee.name }}</h1>
          <div class="hero__meta">
            <span v-if="employee.ext" class="mono">#{{ employee.ext }}</span>
            <span v-if="employee.company" class="tag" :class="employee.company">{{ companyName(employee.company) }}</span>
            <span class="tag-src">{{ employee.source === 'operator' ? t('role.operator') : t('emp.manualAdded') }}</span>
          </div>
        </div>
        <div class="hero__stats">
          <div class="hero__stat">
            <span class="mono">{{ activeCount }}</span>
            <small>{{ t('emp.activeServers') }}</small>
          </div>
          <div class="hero__stat">
            <span class="mono">{{ Math.floor(totalDays/30) }}<em>{{ t('admin.month') }}</em></span>
            <small>{{ t('emp.totalTenure') }}</small>
          </div>
        </div>
      </div>

      <Transition name="page"><div v-if="msg" class="toast">{{ msg }}</div></Transition>

      <!-- Servers -->
      <div class="srv-head">
        <h2 class="section-title">{{ t('emp.assignedServers') }} <span class="count">{{ servers.length }}</span></h2>
        <button class="btn-ghost btn-sm" @click="toggleForm">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 5v14M5 12h14"/></svg>
          {{ showForm ? t('common.close') : t('emp.assignServer') }}
        </button>
      </div>

      <!-- biriktirish formasi -->
      <Transition name="page">
        <form v-if="showForm" class="assign card" @submit.prevent="assignServer">
          <label class="field">
            <span>{{ t('admin.serverName') }}</span>
            <input v-model="nf.name" :placeholder="t('emp.serverNameExample')" autofocus />
          </label>
          <label class="field">
            <span>{{ t('admin.company') }}</span>
            <select v-model="nf.company">
              <option v-for="c in companiesAll" :key="c.id" :value="c.id">{{ c.name }}</option>
            </select>
          </label>
          <label class="field">
            <span>{{ t('admin.startDate') }}</span>
            <input v-model="nf.assigned_at" type="date" />
          </label>
          <button type="submit">{{ t('emp.assign') }}</button>
        </form>
      </Transition>

      <div v-if="!servers.length && !showForm" class="empty card">
        <p>{{ t('emp.noServers') }}</p>
        <button @click="toggleForm">{{ t('emp.addNewServer') }}</button>
      </div>
      <div v-else-if="servers.length" class="srv-list">
        <div v-for="(s, i) in servers" :key="s.id" class="row card" :style="{ animationDelay: i*50+'ms' }">
          <div class="row__bar" :class="s.company"></div>
          <div class="row__main">
            <div class="row__name">{{ s.name }}</div>
            <div class="row__company" v-if="s.company">{{ companyName(s.company) }}</div>
          </div>
          <div class="row__col">
            <span class="row__col-badge">{{ colLabel(s.column) }}</span>
            <small>{{ s.column }}-{{ t('admin.column') }}</small>
          </div>
          <div class="row__dur">
            <span class="mono">{{ fmtAge(s.days) }}</span>
            <small>{{ t('emp.working') }}</small>
          </div>
          <!-- progress: nechta kalonka (oy) -->
          <div class="row__months">
            <i v-for="m in 3" :key="m" :class="{ on: m <= s.column }"></i>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<style scoped>
.ed { animation: fade-up 0.4s both; padding-top: 16px; }
.back { display: inline-block; color: var(--text-dim); text-decoration: none; font-size: 13px;
  font-weight: 600; margin-bottom: 18px; transition: color 0.2s; }
.back:hover { color: var(--text); }

.hero { display: flex; align-items: center; gap: 22px; padding: 26px; margin-bottom: 28px; }
.hero__avatar { width: 70px; height: 70px; border-radius: 20px; flex-shrink: 0; display: grid; place-items: center;
  background: var(--grad); color: #fff; font-weight: 800; font-size: 24px;
  box-shadow: 0 10px 26px -8px rgba(124,92,255,0.6); }
.hero__info { flex: 1; min-width: 0; }
.hero__info h1 { font-size: 26px; font-weight: 800; letter-spacing: -0.02em; }
.hero__meta { display: flex; align-items: center; gap: 10px; margin-top: 8px; font-size: 12.5px; color: var(--text-dim); }
.tag { padding: 2px 10px; border-radius: 999px; font-weight: 600; }
.tag.salesdoc { background: rgba(52,211,153,0.15); color: var(--green); }
.tag.ibox { background: rgba(34,211,238,0.15); color: var(--accent-2); }
.tag-src { color: var(--text-faint); }

.hero__stats { display: flex; gap: 14px; }
.hero__stat { text-align: center; padding: 14px 22px; border-radius: 14px;
  background: var(--surface); border: 1px solid var(--border); }
.hero__stat .mono { font-size: 28px; font-weight: 800; display: block; line-height: 1; }
.hero__stat .mono em { font-size: 14px; font-style: normal; color: var(--text-dim); margin-left: 3px; }
.hero__stat small { font-size: 11px; color: var(--text-faint); display: block; margin-top: 7px; max-width: 110px; }

.srv-head { display: flex; align-items: center; justify-content: space-between; margin-bottom: 16px; }
.section-title { font-size: 16px; font-weight: 700; display: flex; align-items: center; gap: 10px; }
.count { font-size: 12px; font-weight: 600; color: var(--text-dim);
  background: var(--surface-2); padding: 2px 9px; border-radius: 999px; }
.btn-sm { padding: 8px 14px; font-size: 12.5px; }
.btn-sm svg { width: 15px; height: 15px; }

.assign { display: grid; grid-template-columns: 2fr 1fr 1fr auto; gap: 14px; align-items: end; padding: 18px; margin-bottom: 16px; }
.assign .field { display: flex; flex-direction: column; gap: 7px; }
.assign .field span { font-size: 12px; font-weight: 600; color: var(--text-dim); }
.assign input, .assign select { width: 100%; }

.toast { position: fixed; top: 22px; left: 50%; transform: translateX(-50%); z-index: 50;
  background: var(--grad); color: #fff; padding: 11px 22px; border-radius: 12px;
  font-size: 13.5px; font-weight: 600; box-shadow: var(--glow); }

.empty { padding: 44px; text-align: center; color: var(--text-faint); font-size: 13px;
  display: flex; flex-direction: column; align-items: center; gap: 16px; }

.srv-list { display: flex; flex-direction: column; gap: 12px; }
.row { display: flex; align-items: center; gap: 18px; padding: 0; overflow: hidden; animation: fade-up 0.4s both; }
.row__bar { width: 4px; align-self: stretch; background: var(--accent); }
.row__bar.salesdoc { background: var(--green); }
.row__bar.ibox { background: var(--accent-2); }
.row__main { flex: 1; min-width: 0; padding: 16px 0; }
.row__name { font-size: 15px; font-weight: 600; }
.row__company { font-size: 11.5px; color: var(--text-faint); margin-top: 3px; }
.row__col, .row__dur { text-align: center; }
.row__col-badge { display: inline-block; font-size: 12px; font-weight: 700; color: var(--accent);
  background: var(--grad-soft); padding: 3px 11px; border-radius: 999px; }
.row__col small, .row__dur small { display: block; font-size: 10px; color: var(--text-faint); margin-top: 5px; }
.row__dur .mono { font-size: 14px; font-weight: 700; }
.row__months { display: flex; gap: 5px; padding-right: 20px; }
.row__months i { width: 22px; height: 6px; border-radius: 3px; background: var(--surface-2); transition: background 0.3s; }
.row__months i.on { background: var(--grad); }

.loading { display: flex; align-items: center; justify-content: center; gap: 10px; padding: 60px; color: var(--text-dim); }
.spin { width: 18px; height: 18px; border: 2.5px solid var(--border-strong); border-top-color: var(--accent); border-radius: 50%; animation: spin 0.8s linear infinite; }
.banner { background: rgba(248,113,113,0.12); border: 1px solid rgba(248,113,113,0.3); color: #fca5a5; padding: 14px; border-radius: 12px; }

@media (max-width: 1080px) {
  .hero { flex-wrap: wrap; }
  .row__months { display: none; }
  .assign { grid-template-columns: 1fr 1fr; }
}
</style>
