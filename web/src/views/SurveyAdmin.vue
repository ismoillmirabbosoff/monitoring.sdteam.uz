<script setup>
import { ref, computed, onMounted } from 'vue'
import { api, isExtension, todayStr, fmtDuration, fmtTime } from '../api.js'

const tab = ref('report')
const day = ref(todayStr())
const calls = ref([])
const responses = ref(new Set())
const names = ref({})
const loading = ref(true)
const msg = ref('')
const expanded = ref(null)   // ochilgan operator ext (drill-down)

// savollar
const questions = ref([])
const nf = ref({ label: '', type: 'text', options: '', required: false })

function flash(t) { msg.value = t; setTimeout(() => (msg.value = ''), 3000) }
function opExt(c) {
  if (c.direction === 'outbound') return isExtension(c.caller_id_number) ? c.caller_id_number : ''
  return isExtension(c.destination_number) ? c.destination_number : ''
}
function dayRange() {
  const d = new Date(day.value + 'T00:00:00')
  const from = Math.floor(d.getTime() / 1000)
  return [from, from + 86400 - 1]
}

async function loadReport() {
  loading.value = true
  try {
    const [from, to] = dayRange()
    const [cs, rs, us] = await Promise.all([api.data('', from, to), api.surveyResponses(from, to), api.users()])
    const nm = {}; for (const u of us || []) nm[String(u.num)] = u.name
    names.value = nm
    responses.value = new Set(rs.map((r) => r.call_uuid))
    calls.value = cs.filter((c) => (c.user_talk_time || 0) > 0 && opExt(c))
  } catch (e) { flash('Xato: ' + e.message) }
  finally { loading.value = false }
}

async function loadQuestions() {
  try { questions.value = await api.qList() } catch (e) { flash('Xato: ' + e.message) }
}

const stats = computed(() => {
  const total = calls.value.length
  const done = calls.value.filter((c) => responses.value.has(c.uuid)).length
  return { total, done, missing: total - done, coverage: total ? Math.round(done / total * 100) : 0 }
})

// kim ko'p to'ldirmaydi (operator bo'yicha anketasiz)
const byOperator = computed(() => {
  const m = {}
  for (const c of calls.value) {
    const e = opExt(c); if (!e) continue
    m[e] ||= { ext: e, total: 0, missing: 0 }
    m[e].total++
    if (!responses.value.has(c.uuid)) m[e].missing++
  }
  return Object.values(m).filter((o) => o.missing > 0).sort((a, b) => b.missing - a.missing)
})

// Bitta operatorning anketasiz qo'ng'iroqlari (drill-down)
function opUnfilled(ext) {
  return calls.value
    .filter((c) => opExt(c) === ext && !responses.value.has(c.uuid))
    .sort((a, b) => b.start_stamp - a.start_stamp)
}
function toggleOp(ext) { expanded.value = expanded.value === ext ? null : ext }

async function addQuestion() {
  if (!nf.value.label.trim()) { flash('Savol matni kerak'); return }
  try {
    const options = nf.value.type === 'choice'
      ? nf.value.options.split(',').map((s) => s.trim()).filter(Boolean) : []
    await api.qCreate({ label: nf.value.label.trim(), type: nf.value.type, options, required: nf.value.required, position: questions.value.length })
    nf.value = { label: '', type: 'text', options: '', required: false }
    await loadQuestions()
  } catch (e) { flash('Xato: ' + e.message) }
}
async function toggleQ(q) { try { await api.qUpdate(q.id, { active: !q.active }); await loadQuestions() } catch (e) { flash(e.message) } }
async function delQ(q) { try { await api.qDelete(q.id); await loadQuestions() } catch (e) { flash(e.message) } }

const TYPES = { text: 'Matn', choice: 'Tanlov', rating: 'Reyting', yesno: 'Ha/Yo\'q' }

onMounted(() => { loadReport(); loadQuestions() })
</script>

<template>
  <div class="sa">
    <div class="top">
      <div><h1>Anketa hisoboti</h1><p>Qoplanish va sozlamalar</p></div>
      <input v-if="tab === 'report'" type="date" v-model="day" @change="loadReport" />
    </div>
    <Transition name="page"><div v-if="msg" class="toast">{{ msg }}</div></Transition>

    <div class="tabs">
      <button :class="{ active: tab === 'report' }" @click="tab = 'report'">Hisobot</button>
      <button :class="{ active: tab === 'settings' }" @click="tab = 'settings'">Sozlamalar</button>
    </div>

    <!-- HISOBOT -->
    <div v-if="tab === 'report'">
      <div class="kpis">
        <div class="kpi card"><div class="kpi__v">{{ stats.total }}</div><div class="kpi__l">Suhbatli qo'ng'iroqlar</div></div>
        <div class="kpi card"><div class="kpi__v" style="color:var(--green)">{{ stats.done }}</div><div class="kpi__l">Anketa to'ldirilgan</div></div>
        <div class="kpi card"><div class="kpi__v" style="color:var(--red)">{{ stats.missing }}</div><div class="kpi__l">Anketasiz</div></div>
        <div class="kpi card"><div class="kpi__v" style="color:var(--accent)">{{ stats.coverage }}%</div><div class="kpi__l">Qoplanish</div></div>
      </div>

      <h2 class="sub">Kim ko'p to'ldirmaydi</h2>
      <div v-if="loading" class="loading"><i class="spin"></i></div>
      <div v-else class="ops card">
        <div v-for="o in byOperator" :key="o.ext" class="opwrap">
          <div class="op" :class="{ open: expanded === o.ext }" @click="toggleOp(o.ext)">
            <div class="op__av">{{ (names[o.ext] || o.ext).slice(0,2).toUpperCase() }}</div>
            <div class="op__info">
              <div class="op__name">{{ names[o.ext] || ('Operator ' + o.ext) }}</div>
              <div class="op__ext mono">#{{ o.ext }}</div>
            </div>
            <div class="op__bar"><div class="op__fill" :style="{ width: (o.missing/o.total*100)+'%' }"></div></div>
            <div class="op__cnt"><b>{{ o.missing }}</b> / {{ o.total }}</div>
            <svg class="op__caret" :class="{ rot: expanded === o.ext }" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round" stroke-linejoin="round"><path d="M6 9l6 6 6-6"/></svg>
          </div>
          <div v-if="expanded === o.ext" class="drill">
            <div class="drill__head">Anketasiz qo'ng'iroqlar ({{ opUnfilled(o.ext).length }})</div>
            <div v-for="c in opUnfilled(o.ext)" :key="c.uuid" class="drow">
              <span class="drow__dir" :class="c.direction === 'outbound' ? 'out' : 'in'">
                <svg v-if="c.direction === 'outbound'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><path d="M7 17 17 7M17 7H9M17 7v8"/></svg>
                <svg v-else viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><path d="M17 7 7 17M7 17h8M7 17V9"/></svg>
              </span>
              <span class="drow__num mono">{{ c.caller_id_number }} → {{ c.destination_number }}</span>
              <span class="drow__meta mono">{{ fmtTime(c.start_stamp) }} · {{ fmtDuration(c.user_talk_time) }}</span>
            </div>
            <div v-if="!opUnfilled(o.ext).length" class="drill__empty">Bu operatorda anketasiz qo'ng'iroq yo'q 🎉</div>
          </div>
        </div>
        <div v-if="!byOperator.length" class="empty">Hammasi to'ldirilgan 🎉</div>
      </div>
    </div>

    <!-- SOZLAMALAR -->
    <div v-else>
      <form class="card qform" @submit.prevent="addQuestion">
        <input v-model="nf.label" placeholder="Savol matni" class="grow" />
        <select v-model="nf.type">
          <option v-for="(l,k) in TYPES" :key="k" :value="k">{{ l }}</option>
        </select>
        <input v-if="nf.type === 'choice'" v-model="nf.options" placeholder="variantlar: ha, yo'q, balki" class="grow" />
        <label class="req"><input type="checkbox" v-model="nf.required" /> majburiy</label>
        <button type="submit">+ Qo'shish</button>
      </form>

      <div class="card qlist">
        <div v-for="q in questions" :key="q.id" class="q" :class="{ off: !q.active }">
          <span class="q__type">{{ TYPES[q.type] }}</span>
          <span class="q__label">{{ q.label }}<span v-if="q.required" class="q__req">*</span></span>
          <span class="q__opts" v-if="q.type==='choice'">{{ (Array.isArray(q.options)?q.options:JSON.parse(q.options||'[]')).join(' · ') }}</span>
          <button class="q__btn" @click="toggleQ(q)">{{ q.active ? 'Yashirish' : 'Yoqish' }}</button>
          <button class="q__btn del" @click="delQ(q)">×</button>
        </div>
        <div v-if="!questions.length" class="empty">Savol qo'shilmagan</div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.sa { animation: fade-up 0.4s both; }
.top { display: flex; justify-content: space-between; align-items: flex-start; margin: 16px 0 18px; }
.top h1 { font-size: 24px; font-weight: 800; }
.top p { color: var(--text-dim); font-size: 13px; margin-top: 4px; }
.toast { position: fixed; top: 22px; left: 50%; transform: translateX(-50%); z-index: 60; background: var(--grad); color: #fff; padding: 11px 22px; border-radius: 12px; font-weight: 600; box-shadow: var(--glow); }
.tabs { display: inline-flex; gap: 4px; background: var(--surface); padding: 5px; border-radius: 12px; border: 1px solid var(--border); margin-bottom: 20px; }
.tabs button { background: transparent; color: var(--text-dim); padding: 9px 18px; font-size: 13px; }
.tabs button:hover { transform: none; box-shadow: none; color: var(--text); }
.tabs button.active { background: var(--grad); color: #fff; }

.kpis { display: grid; grid-template-columns: repeat(4, 1fr); gap: 14px; margin-bottom: 28px; }
.kpi { padding: 20px; }
.kpi__v { font-size: 32px; font-weight: 800; font-family: var(--mono); }
.kpi__l { font-size: 12.5px; color: var(--text-dim); margin-top: 6px; }
.sub { font-size: 16px; font-weight: 700; margin-bottom: 14px; }
.ops { padding: 8px 14px; }
.opwrap { border-top: 1px solid var(--border); }
.opwrap:first-child { border-top: none; }
.op { display: flex; align-items: center; gap: 14px; padding: 12px 4px; cursor: pointer; border-radius: 10px; transition: background 0.15s; }
.op:hover { background: var(--surface-2); }
.op.open { background: var(--surface-2); }
.op__caret { width: 18px; height: 18px; color: var(--text-faint); flex-shrink: 0; transition: transform 0.2s; }
.op__caret.rot { transform: rotate(180deg); color: var(--accent); }
.drill { padding: 6px 4px 14px 52px; animation: fade-up 0.25s both; }
.drill__head { font-size: 11.5px; font-weight: 600; color: var(--text-faint); text-transform: uppercase; letter-spacing: 0.04em; margin: 4px 0 10px; }
.drow { display: flex; align-items: center; gap: 12px; padding: 8px 10px; border-radius: 9px; }
.drow:hover { background: var(--surface); }
.drow__dir { width: 26px; height: 26px; border-radius: 8px; display: grid; place-items: center; flex-shrink: 0; }
.drow__dir svg { width: 13px; height: 13px; }
.drow__dir.in { background: rgba(16,185,129,0.14); color: var(--green); }
.drow__dir.out { background: rgba(20,184,196,0.14); color: var(--accent-2); }
.drow__num { font-size: 13px; font-weight: 600; }
.drow__meta { font-size: 11.5px; color: var(--text-faint); margin-left: auto; }
.drill__empty { color: var(--text-faint); font-size: 12.5px; padding: 10px; }
.op__av { width: 38px; height: 38px; border-radius: 11px; background: var(--grad-soft); color: var(--accent); display: grid; place-items: center; font-weight: 700; font-size: 13px; flex-shrink: 0; }
.op__info { width: 180px; }
.op__name { font-size: 13.5px; font-weight: 600; }
.op__ext { font-size: 11px; color: var(--text-faint); }
.op__bar { flex: 1; height: 8px; border-radius: 999px; background: var(--surface-2); overflow: hidden; }
.op__fill { height: 100%; background: linear-gradient(90deg, var(--amber), var(--red)); border-radius: 999px; }
.op__cnt { font-size: 13px; color: var(--text-dim); width: 64px; text-align: right; }
.op__cnt b { color: var(--red); font-size: 16px; }

.qform { display: flex; gap: 10px; align-items: center; padding: 16px; margin-bottom: 16px; flex-wrap: wrap; }
.qform .grow { flex: 1; min-width: 160px; }
.req { display: inline-flex; align-items: center; gap: 6px; font-size: 12.5px; color: var(--text-dim); white-space: nowrap; }
.qlist { padding: 8px 14px; }
.q { display: flex; align-items: center; gap: 12px; padding: 13px 4px; border-top: 1px solid var(--border); }
.q:first-child { border-top: none; }
.q.off { opacity: 0.5; }
.q__type { font-size: 11px; font-weight: 600; color: var(--accent); background: var(--grad-soft); padding: 3px 10px; border-radius: 999px; white-space: nowrap; }
.q__label { font-weight: 600; font-size: 14px; }
.q__req { color: var(--red); margin-left: 2px; }
.q__opts { color: var(--text-faint); font-size: 12px; margin-left: auto; }
.q__btn { padding: 6px 12px; font-size: 12px; background: var(--surface-2); color: var(--text-dim); }
.q__btn:hover { transform: none; box-shadow: none; color: var(--text); }
.q__btn.del { background: rgba(239,68,68,0.14); color: var(--red); }
.q:not(:has(.q__opts)) .q__btn:first-of-type { margin-left: auto; }
.empty { text-align: center; color: var(--text-faint); padding: 36px; }
.loading { display: flex; justify-content: center; padding: 40px; }
.spin { width: 18px; height: 18px; border: 2.5px solid var(--border-strong); border-top-color: var(--accent); border-radius: 50%; animation: spin 0.8s linear infinite; }
@media (max-width: 1080px) { .kpis { grid-template-columns: repeat(2,1fr); } }
</style>
