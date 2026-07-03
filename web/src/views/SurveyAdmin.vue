<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import { api, isExtension, todayStr, fmtDuration, fmtTime, COMPANIES, companyForGateway, companyForQueue } from '../api.js'
import SurveyForm from '../components/SurveyForm.vue'

const tab = ref('report')
const calls = ref([])
const responses = ref(new Set())
const names = ref({})
const extCompany = ref({})       // ext -> kompaniya (tr1 navbatdan)
const loading = ref(true)
const msg = ref('')

// --- sana oralig'i ---
const preset = ref('week')
const fromInput = ref('')
const toInput = ref('')

// --- filtrlar (client-side) ---
const fOperator = ref('')
const fCompany = ref('')
const fDirection = ref('')
const fMinTalk = ref(0)
const fPhone = ref('')

// --- sahifalash ---
const PAGE_SIZE = 50
const page = ref(1)

// --- anketa savollari (sozlamalar + to'ldirish) ---
const questions = ref([])        // barcha (sozlamalar tab)
const activeQuestions = ref([])  // faol (to'ldirish modali)
const nf = ref({ label: '', type: 'text', options: '', required: false })

// --- to'ldirish modali ---
const fillActive = ref(null)
const answers = ref({})
const saving = ref(false)

function flash(t) { msg.value = t; setTimeout(() => (msg.value = ''), 3000) }
function pad(n) { return String(n).padStart(2, '0') }

function opExt(c) {
  if (c.direction === 'outbound') return isExtension(c.caller_id_number) ? c.caller_id_number : ''
  return isExtension(c.destination_number) ? c.destination_number : ''
}
function clientNumber(c) { return c.direction === 'outbound' ? c.destination_number : c.caller_id_number }
function callCompany(c) {
  const g = companyForGateway(c.gateway)
  if (g) return g
  const e = opExt(c)
  return e ? (extCompany.value[e] || '') : ''
}
function compBadge(id) { return id === 'salesdoc' ? 'SD' : id === 'ibox' ? 'iBox' : '—' }
function fmtDateTime(stamp) {
  const d = new Date(stamp * 1000)
  return `${pad(d.getDate())}.${pad(d.getMonth() + 1)} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

// --- sana presetlari ---
function setRange(fromDate, toDate) {
  fromInput.value = todayStr(fromDate)
  toInput.value = todayStr(toDate)
}
function applyPreset(id) {
  preset.value = id
  const n = new Date()
  if (id === 'today') setRange(n, n)
  else if (id === 'yesterday') { const y = new Date(n); y.setDate(n.getDate() - 1); setRange(y, y) }
  else if (id === 'week') { const w = new Date(n); w.setDate(n.getDate() - 6); setRange(w, n) }
  else if (id === 'month') { setRange(new Date(n.getFullYear(), n.getMonth(), 1), n) }
  if (id !== 'custom') loadReport()
}
function rangeUnix() {
  const f = new Date(fromInput.value + 'T00:00:00')
  const t = new Date(toInput.value + 'T23:59:59')
  return [Math.floor(f.getTime() / 1000), Math.floor(t.getTime() / 1000)]
}
const rangeLabel = computed(() => `${fromInput.value} — ${toInput.value}`)

async function loadReport() {
  loading.value = true
  try {
    const [from, to] = rangeUnix()
    const [cs, rs, us] = await Promise.all([api.data('', from, to), api.surveyResponses(from, to), api.users()])
    const nm = {}, cm = {}
    for (const u of us || []) { if (u.num) { nm[String(u.num)] = u.name; cm[String(u.num)] = companyForQueue(u.tr1) } }
    names.value = nm; extCompany.value = cm
    responses.value = new Set(rs.map((r) => r.call_uuid))
    calls.value = cs.filter((c) => (c.user_talk_time || 0) > 0 && opExt(c))  // faqat suhbatli + operatorli
    page.value = 1
  } catch (e) { flash('Xato: ' + e.message) }
  finally { loading.value = false }
}

async function loadQuestions() {
  try { questions.value = await api.qList() } catch (e) { flash('Xato: ' + e.message) }
}
async function loadActive() {
  try { activeQuestions.value = await api.surveyQuestions() } catch {}
}

// --- filtrlangan suhbatli qo'ng'iroqlar ---
const filtered = computed(() => calls.value.filter((c) => {
  if (fCompany.value && callCompany(c) !== fCompany.value) return false
  if (fOperator.value && opExt(c) !== fOperator.value) return false
  if (fDirection.value && c.direction !== fDirection.value) return false
  if (fMinTalk.value && (c.user_talk_time || 0) < Number(fMinTalk.value)) return false
  if (fPhone.value && !String(clientNumber(c) || '').includes(fPhone.value.trim())) return false
  return true
}))
const unfilledCalls = computed(() =>
  filtered.value.filter((c) => !responses.value.has(c.uuid)).sort((a, b) => b.start_stamp - a.start_stamp)
)
const stats = computed(() => {
  const total = filtered.value.length
  const done = filtered.value.filter((c) => responses.value.has(c.uuid)).length
  return { total, done, missing: total - done, coverage: total ? Math.round(done / total * 100) : 0 }
})

// operator dropdown ro'yxati
const operatorOptions = computed(() => {
  const set = new Set(calls.value.map(opExt).filter(Boolean))
  return [...set].sort((a, b) => Number(a) - Number(b)).map((ext) => ({ ext, name: names.value[ext] || `Operator ${ext}` }))
})

// kim ko'p to'ldirmaydi (filtrga bo'ysunadi)
const byOperator = computed(() => {
  const m = {}
  for (const c of filtered.value) {
    const e = opExt(c); if (!e) continue
    m[e] ||= { ext: e, total: 0, missing: 0 }
    m[e].total++
    if (!responses.value.has(c.uuid)) m[e].missing++
  }
  return Object.values(m).filter((o) => o.missing > 0).sort((a, b) => b.missing - a.missing)
})

// sahifalangan qo'ng'iroqlar ro'yxati (anketasiz)
const totalPages = computed(() => Math.max(1, Math.ceil(unfilledCalls.value.length / PAGE_SIZE)))
const pagedCalls = computed(() => unfilledCalls.value.slice((page.value - 1) * PAGE_SIZE, page.value * PAGE_SIZE))

// filtr o'zgarsa — sahifani boshiga
watch([fOperator, fCompany, fDirection, fMinTalk, fPhone], () => { page.value = 1 })

function selectOperator(ext) { fOperator.value = fOperator.value === ext ? '' : ext }
function resetFilters() {
  fOperator.value = ''; fCompany.value = ''; fDirection.value = ''; fMinTalk.value = 0; fPhone.value = ''
}

// --- to'ldirish modali ---
function openFill(c) {
  fillActive.value = c
  answers.value = {}
}
function closeFill() { fillActive.value = null }
async function submitFill() {
  for (const q of activeQuestions.value) {
    if (q.required && !answers.value[q.id]) { flash(`"${q.label}" majburiy`); return }
  }
  saving.value = true
  try {
    await api.surveySubmit({ call_uuid: fillActive.value.uuid, operator_ext: opExt(fillActive.value), answers: answers.value })
    responses.value = new Set([...responses.value, fillActive.value.uuid])
    flash('Anketa saqlandi')
    closeFill()
  } catch (e) { flash('Xato: ' + e.message) }
  finally { saving.value = false }
}

// --- sozlamalar (savol builder) ---
async function addQuestion() {
  if (!nf.value.label.trim()) { flash('Savol matni kerak'); return }
  try {
    const options = nf.value.type === 'choice'
      ? nf.value.options.split(',').map((s) => s.trim()).filter(Boolean) : []
    await api.qCreate({ label: nf.value.label.trim(), type: nf.value.type, options, required: nf.value.required, position: questions.value.length })
    nf.value = { label: '', type: 'text', options: '', required: false }
    await Promise.all([loadQuestions(), loadActive()])
  } catch (e) { flash('Xato: ' + e.message) }
}
async function toggleQ(q) { try { await api.qUpdate(q.id, { active: !q.active }); await Promise.all([loadQuestions(), loadActive()]) } catch (e) { flash(e.message) } }
async function delQ(q) { try { await api.qDelete(q.id); await Promise.all([loadQuestions(), loadActive()]) } catch (e) { flash(e.message) } }

const TYPES = { text: 'Matn', choice: 'Tanlov', rating: 'Reyting', yesno: 'Ha/Yo\'q' }
const companies = COMPANIES

onMounted(() => {
  applyPreset('week')   // fromInput/toInput + loadReport
  loadQuestions(); loadActive()
})
</script>

<template>
  <div class="sa">
    <div class="top">
      <div><h1>Anketa hisoboti</h1><p>Anketasiz qo'ng'iroqlar · qoplanish · sozlamalar</p></div>
    </div>
    <Teleport to="body"><Transition name="page"><div v-if="msg" class="toast">{{ msg }}</div></Transition></Teleport>

    <div class="tabs">
      <button :class="{ active: tab === 'report' }" @click="tab = 'report'">Hisobot</button>
      <button :class="{ active: tab === 'settings' }" @click="tab = 'settings'">Sozlamalar</button>
    </div>

    <!-- ================= HISOBOT ================= -->
    <div v-if="tab === 'report'">
      <!-- KPI -->
      <div class="kpis">
        <div class="kpi card"><div class="kpi__ico warn">!</div><div><div class="kpi__v">{{ stats.missing }}</div><div class="kpi__l">Anketasiz qo'ng'iroqlar</div></div></div>
        <div class="kpi card"><div class="kpi__ico ok">✓</div><div><div class="kpi__v" style="color:var(--green)">{{ stats.done }}</div><div class="kpi__l">Anketa to'ldirilgan</div></div></div>
        <div class="kpi card"><div class="kpi__ico blue">☎</div><div><div class="kpi__v">{{ stats.total }}</div><div class="kpi__l">Suhbatli qo'ng'iroqlar</div></div></div>
        <div class="kpi card"><div class="kpi__ico acc">%</div><div><div class="kpi__v" style="color:var(--accent)">{{ stats.coverage }}%</div><div class="kpi__l">Anketa qoplanishi</div></div></div>
      </div>

      <!-- FILTRLAR -->
      <div class="card filters">
        <div class="fl-presets">
          <button v-for="p in [['today','Bugun'],['yesterday','Kecha'],['week','Hafta'],['month','Oy']]" :key="p[0]"
                  class="preset" :class="{ active: preset === p[0] }" @click="applyPreset(p[0])">{{ p[1] }}</button>
          <span class="fl-range mono">{{ rangeLabel }}</span>
        </div>
        <div class="fl-grid">
          <label class="fld"><span>Dan</span><input type="date" v-model="fromInput" @change="preset='custom'; loadReport()" /></label>
          <label class="fld"><span>Gacha</span><input type="date" v-model="toInput" @change="preset='custom'; loadReport()" /></label>
          <label class="fld"><span>Operator</span>
            <select v-model="fOperator"><option value="">Hammasi</option>
              <option v-for="o in operatorOptions" :key="o.ext" :value="o.ext">{{ o.name }} · #{{ o.ext }}</option>
            </select>
          </label>
          <label class="fld"><span>Kanal</span>
            <select v-model="fCompany"><option value="">Hammasi</option>
              <option v-for="c in companies.filter(x=>x.id)" :key="c.id" :value="c.id">{{ c.name }}</option>
            </select>
          </label>
          <label class="fld"><span>Yo'nalish</span>
            <select v-model="fDirection"><option value="">Hammasi</option><option value="inbound">Kiruvchi</option><option value="outbound">Chiquvchi</option></select>
          </label>
          <label class="fld"><span>Min. suhbat (sek)</span><input type="number" min="0" v-model="fMinTalk" placeholder="0" /></label>
          <label class="fld"><span>Telefon</span><input v-model="fPhone" placeholder="998…" /></label>
          <button class="fl-reset" @click="resetFilters">Tozalash</button>
        </div>
      </div>

      <!-- KIM KO'P TO'LDIRMAYDI -->
      <div class="section-h"><h2>Kim ko'p to'ldirmaydi</h2><span class="muted">tanlangan davr bo'yicha · bosing = filtrlaydi</span></div>
      <div v-if="loading" class="loading"><i class="spin"></i></div>
      <div v-else class="opgrid">
        <button v-for="o in byOperator" :key="o.ext" class="opc" :class="{ active: fOperator === o.ext }" @click="selectOperator(o.ext)">
          <div class="opc__av">{{ (names[o.ext] || o.ext).slice(0,2).toUpperCase() }}</div>
          <div class="opc__info">
            <div class="opc__name">{{ names[o.ext] || ('Operator ' + o.ext) }}</div>
            <div class="opc__meta"><b>{{ o.missing }}</b> anketasiz · #{{ o.ext }}</div>
          </div>
          <svg class="opc__arr" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><path d="M9 6l6 6-6 6"/></svg>
        </button>
        <div v-if="!byOperator.length" class="empty">Hammasi to'ldirilgan 🎉</div>
      </div>

      <!-- QO'NG'IROQLAR RO'YXATI -->
      <div class="section-h">
        <h2>Anketasiz qo'ng'iroqlar <span class="count">{{ unfilledCalls.length }}</span></h2>
        <div class="pager" v-if="totalPages > 1">
          <button :disabled="page<=1" @click="page--">←</button>
          <span class="mono">{{ page }} / {{ totalPages }}</span>
          <button :disabled="page>=totalPages" @click="page++">→</button>
        </div>
      </div>
      <div class="card tbl-wrap">
        <table class="tbl">
          <thead><tr>
            <th>Sana</th><th>Kanal</th><th>Klient</th><th>Operator</th><th class="ta-c">Yo'nalish</th><th class="ta-r">Suhbat</th><th class="ta-r">Amal</th>
          </tr></thead>
          <tbody>
            <tr v-for="c in pagedCalls" :key="c.uuid">
              <td class="mono dim">{{ fmtDateTime(c.start_stamp) }}</td>
              <td><span class="cbadge" :class="callCompany(c)">{{ compBadge(callCompany(c)) }}</span></td>
              <td class="mono">{{ clientNumber(c) }}</td>
              <td>
                <div class="top-op">
                  <span class="tav">{{ (names[opExt(c)] || opExt(c)).slice(0,2).toUpperCase() }}</span>
                  <span>{{ names[opExt(c)] || ('Operator ' + opExt(c)) }}</span>
                </div>
              </td>
              <td class="ta-c">
                <span class="dir" :class="c.direction === 'outbound' ? 'out' : 'in'">{{ c.direction === 'outbound' ? 'Chiq.' : 'Kir.' }}</span>
              </td>
              <td class="ta-r mono">{{ fmtDuration(c.user_talk_time) }}</td>
              <td class="ta-r"><button class="fill-btn" @click="openFill(c)">To'ldirish</button></td>
            </tr>
            <tr v-if="!pagedCalls.length"><td colspan="7" class="empty">Anketasiz qo'ng'iroq yo'q 🎉</td></tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- ================= SOZLAMALAR ================= -->
    <div v-else>
      <form class="card qform" @submit.prevent="addQuestion">
        <input v-model="nf.label" placeholder="Savol matni" class="grow" />
        <select v-model="nf.type"><option v-for="(l,k) in TYPES" :key="k" :value="k">{{ l }}</option></select>
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

    <!-- TO'LDIRISH MODALI -->
    <Teleport to="body">
      <Transition name="modal">
        <div v-if="fillActive" class="modal" @click.self="closeFill">
          <div class="modal__card">
            <div class="modal__head">
              <div>
                <h3>Anketa to'ldirish</h3>
                <p class="mono">{{ clientNumber(fillActive) }} · #{{ opExt(fillActive) }} · {{ fmtDateTime(fillActive.start_stamp) }}</p>
              </div>
              <button class="modal__x" @click="closeFill">×</button>
            </div>
            <div class="modal__body"><SurveyForm :questions="activeQuestions" v-model="answers" /></div>
            <div class="modal__foot">
              <button class="btn-ghost" @click="closeFill">Bekor</button>
              <button @click="submitFill" :disabled="saving">{{ saving ? '...' : 'Saqlash' }}</button>
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<style scoped>
.sa { animation: fade-up 0.4s both; }
.top { display: flex; justify-content: space-between; align-items: flex-start; margin: 16px 0 18px; }
.top h1 { font-size: 24px; font-weight: 800; }
.top p { color: var(--text-dim); font-size: 13px; margin-top: 4px; }
.toast { position: fixed; top: 22px; left: 50%; transform: translateX(-50%); z-index: 120; background: var(--grad); color: #fff; padding: 11px 22px; border-radius: 12px; font-weight: 600; box-shadow: var(--glow); }
.tabs { display: inline-flex; gap: 4px; background: var(--surface); padding: 5px; border-radius: 12px; border: 1px solid var(--border); margin-bottom: 20px; }
.tabs button { background: transparent; color: var(--text-dim); padding: 9px 18px; font-size: 13px; }
.tabs button:hover { transform: none; box-shadow: none; color: var(--text); }
.tabs button.active { background: var(--grad); color: #fff; }

/* KPI */
.kpis { display: grid; grid-template-columns: repeat(4, 1fr); gap: 14px; margin-bottom: 18px; }
.kpi { padding: 18px 20px; display: flex; align-items: center; gap: 14px; }
.kpi__ico { width: 42px; height: 42px; border-radius: 12px; display: grid; place-items: center; font-size: 18px; font-weight: 800; flex-shrink: 0; }
.kpi__ico.warn { background: rgba(245,158,11,0.15); color: var(--amber); }
.kpi__ico.ok { background: var(--green-soft, rgba(16,185,129,0.15)); color: var(--green); }
.kpi__ico.blue { background: rgba(59,130,246,0.15); color: var(--blue, #3b82f6); }
.kpi__ico.acc { background: var(--accent-soft, rgba(109,94,252,0.16)); color: var(--accent); }
.kpi__v { font-size: 28px; font-weight: 800; font-family: var(--mono); line-height: 1; }
.kpi__l { font-size: 12px; color: var(--text-dim); margin-top: 5px; }

/* Filtrlar */
.filters { padding: 16px 18px; margin-bottom: 22px; }
.fl-presets { display: flex; align-items: center; gap: 6px; margin-bottom: 14px; flex-wrap: wrap; }
.preset { background: var(--surface-2); color: var(--text-dim); padding: 7px 14px; font-size: 12.5px; border: 1px solid var(--border); }
.preset:hover { transform: none; box-shadow: none; color: var(--text); }
.preset.active { background: var(--grad); color: #fff; border-color: transparent; }
.fl-range { margin-left: auto; font-size: 12px; color: var(--text-faint); }
.fl-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 12px; align-items: end; }
.fld { display: flex; flex-direction: column; gap: 6px; }
.fld span { font-size: 11.5px; font-weight: 600; color: var(--text-dim); }
.fld input, .fld select { width: 100%; }
.fl-reset { background: var(--surface-2); color: var(--text-dim); border: 1px solid var(--border); height: 38px; }
.fl-reset:hover { transform: none; box-shadow: none; color: var(--text); }

.section-h { display: flex; align-items: center; justify-content: space-between; margin: 22px 0 14px; }
.section-h h2 { font-size: 16px; font-weight: 700; display: flex; align-items: center; gap: 10px; }
.count { font-size: 12px; font-weight: 600; color: var(--text-dim); background: var(--surface-2); padding: 2px 9px; border-radius: 999px; }
.muted { font-size: 12px; color: var(--text-faint); }

/* Operator grid */
.opgrid { display: grid; grid-template-columns: repeat(auto-fill, minmax(240px, 1fr)); gap: 12px; margin-bottom: 8px; }
.opc { display: flex; align-items: center; gap: 12px; padding: 12px 14px; text-align: left;
  background: var(--surface); border: 1px solid var(--border); border-radius: 14px; }
.opc:hover { transform: none; box-shadow: var(--shadow); border-color: var(--border-strong); }
.opc.active { border-color: var(--accent); background: var(--accent-soft, rgba(109,94,252,0.1)); }
.opc__av { width: 40px; height: 40px; border-radius: 11px; background: var(--grad-soft); color: var(--accent); display: grid; place-items: center; font-weight: 700; font-size: 13px; flex-shrink: 0; }
.opc__info { min-width: 0; flex: 1; }
.opc__name { font-size: 13.5px; font-weight: 600; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.opc__meta { font-size: 11.5px; color: var(--text-faint); margin-top: 3px; }
.opc__meta b { color: var(--red); font-size: 13px; }
.opc__arr { width: 16px; height: 16px; color: var(--text-faint); flex-shrink: 0; }

/* Jadval */
.tbl-wrap { padding: 6px 8px; overflow-x: auto; }
.tbl { width: 100%; border-collapse: collapse; }
.tbl th { text-align: left; font-size: 11px; font-weight: 600; color: var(--text-faint); text-transform: uppercase; padding: 12px 14px; white-space: nowrap; }
.tbl td { padding: 11px 14px; border-top: 1px solid var(--border); font-size: 13px; white-space: nowrap; }
.ta-c { text-align: center; } .ta-r { text-align: right; }
.dim { color: var(--text-faint); }
.cbadge { font-size: 11px; font-weight: 700; padding: 3px 9px; border-radius: 7px; background: var(--surface-2); color: var(--text-dim); }
.cbadge.salesdoc { background: rgba(16,185,129,0.15); color: var(--green); }
.cbadge.ibox { background: rgba(6,182,212,0.15); color: var(--accent-2); }
.top-op { display: flex; align-items: center; gap: 9px; }
.tav { width: 28px; height: 28px; border-radius: 8px; background: var(--grad-soft); color: var(--accent); display: grid; place-items: center; font-size: 11px; font-weight: 700; flex-shrink: 0; }
.dir { font-size: 11px; font-weight: 600; padding: 3px 10px; border-radius: 999px; }
.dir.in { background: rgba(16,185,129,0.14); color: var(--green); }
.dir.out { background: rgba(6,182,212,0.14); color: var(--accent-2); }
.fill-btn { padding: 7px 14px; font-size: 12.5px; }
.pager { display: flex; align-items: center; gap: 10px; }
.pager button { width: 34px; height: 34px; padding: 0; background: var(--surface-2); border: 1px solid var(--border); color: var(--text-dim); }
.pager button:hover:not(:disabled) { color: var(--text); transform: none; box-shadow: none; }
.pager button:disabled { opacity: 0.4; cursor: not-allowed; }

/* Sozlamalar */
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

.empty { text-align: center; color: var(--text-faint); padding: 30px; }
.loading { display: flex; justify-content: center; padding: 40px; }
.spin { width: 18px; height: 18px; border: 2.5px solid var(--border-strong); border-top-color: var(--accent); border-radius: 50%; animation: spin 0.8s linear infinite; }

/* Modal */
.modal { position: fixed; inset: 0; z-index: 100; background: rgba(8,10,18,0.55); backdrop-filter: blur(4px); display: grid; place-items: center; padding: 20px; }
.modal__card { width: 480px; max-width: 100%; max-height: 90vh; display: flex; flex-direction: column; background: var(--surface); border: 1px solid var(--border); border-radius: var(--radius-lg); box-shadow: var(--shadow-lg); overflow: hidden; }
.modal__head { display: flex; justify-content: space-between; align-items: center; gap: 12px; padding: 20px 22px; border-bottom: 1px solid var(--border); }
.modal__head h3 { font-size: 17px; font-weight: 700; }
.modal__head p { font-size: 12px; color: var(--text-dim); margin-top: 3px; }
.modal__x { background: var(--surface-2); color: var(--text-faint); font-size: 22px; width: 34px; height: 34px; padding: 0; border-radius: 10px; line-height: 1; }
.modal__x:hover { color: var(--text); transform: none; box-shadow: none; background: var(--surface-3); }
.modal__body { padding: 22px; overflow-y: auto; }
.modal__foot { display: flex; justify-content: flex-end; gap: 10px; padding: 18px 22px; border-top: 1px solid var(--border); }
.modal-enter-active, .modal-leave-active { transition: opacity 0.25s; }
.modal-enter-from, .modal-leave-to { opacity: 0; }

@media (max-width: 1080px) { .kpis, .fl-grid { grid-template-columns: repeat(2, 1fr); } }
</style>
