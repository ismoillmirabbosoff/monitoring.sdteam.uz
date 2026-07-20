<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import { api, isExtension, todayStr, fmtDuration, COMPANIES, companyForGateway, companyForQueue } from '../api.js'
import { auth } from '../auth.js'
import AnketaForm from '../components/AnketaForm.vue'
import { t } from '../i18n.js'

const calls = ref([])
const responses = ref(new Set())
const names = ref({})
const extCompany = ref({})
const config = ref({ reasons: [], common_modules: [], payment_topics: [], statuses: [] })
const loading = ref(true)
const msg = ref('')

const preset = ref('week')
const fromInput = ref('')
const toInput = ref('')

const fOperator = ref('')
const fCompany = ref('')
const fDirection = ref('')
const fStatus = ref('')      // '' | answered | missed
const fAnketa = ref('')      // '' | yes | no
const fMinTalk = ref(0)
const fPhone = ref('')

const PAGE_SIZE = 50
const page = ref(1)

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
function opName(c) { const e = opExt(c); return e ? (names.value[e] || ('Operator ' + e)) : '—' }
function isAnswered(c) { return (c.user_talk_time || 0) > 0 }
function isFilled(c) { return responses.value.has(c.uuid) }
function fmtDateTime(stamp) {
  const d = new Date(stamp * 1000)
  return `${pad(d.getDate())}.${pad(d.getMonth() + 1)} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

function setRange(f, t) { fromInput.value = todayStr(f); toInput.value = todayStr(t) }
function applyPreset(id) {
  preset.value = id
  const n = new Date()
  if (id === 'today') setRange(n, n)
  else if (id === 'yesterday') { const y = new Date(n); y.setDate(n.getDate() - 1); setRange(y, y) }
  else if (id === 'week') { const w = new Date(n); w.setDate(n.getDate() - 6); setRange(w, n) }
  else if (id === 'month') { setRange(new Date(n.getFullYear(), n.getMonth(), 1), n) }
  if (id !== 'custom') load()
}
function rangeUnix() {
  const f = new Date(fromInput.value + 'T00:00:00')
  const t = new Date(toInput.value + 'T23:59:59')
  return [Math.floor(f.getTime() / 1000), Math.floor(t.getTime() / 1000)]
}
const rangeLabel = computed(() => `${fromInput.value} — ${toInput.value}`)

async function load() {
  loading.value = true
  try {
    const [from, to] = rangeUnix()
    const [cs, rs, us, cfg] = await Promise.all([
      api.data('', from, to), api.surveyResponses(from, to).catch(() => []), api.users().catch(() => []), api.surveyConfig().catch(() => null),
    ])
    const nm = {}, cm = {}
    for (const u of us || []) { if (u.num) { nm[String(u.num)] = u.name; cm[String(u.num)] = companyForQueue(u.tr1) } }
    names.value = nm; extCompany.value = cm
    responses.value = new Set((rs || []).map((r) => r.call_uuid))
    if (cfg) config.value = cfg
    let list = cs.filter((c) => opExt(c))
    if (!auth.isAdmin && auth.user?.ext) list = list.filter((c) => opExt(c) === auth.user.ext)
    calls.value = list.sort((a, b) => b.start_stamp - a.start_stamp)
    page.value = 1
  } catch (e) { flash(t('common.errorPrefix') + e.message) }
  finally { loading.value = false }
}

const filtered = computed(() => calls.value.filter((c) => {
  if (fCompany.value && callCompany(c) !== fCompany.value) return false
  if (fOperator.value && opExt(c) !== fOperator.value) return false
  if (fDirection.value && c.direction !== fDirection.value) return false
  if (fStatus.value === 'answered' && !isAnswered(c)) return false
  if (fStatus.value === 'missed' && isAnswered(c)) return false
  if (fAnketa.value === 'yes' && !isFilled(c)) return false
  if (fAnketa.value === 'no' && isFilled(c)) return false
  if (fMinTalk.value && (c.user_talk_time || 0) < Number(fMinTalk.value)) return false
  if (fPhone.value && !String(clientNumber(c) || '').includes(fPhone.value.trim())) return false
  return true
}))

const kpi = computed(() => {
  const total = filtered.value.length
  const answered = filtered.value.filter(isAnswered).length
  const filled = filtered.value.filter((c) => isAnswered(c) && isFilled(c)).length
  return { total, answered, missed: total - answered, cov: answered ? Math.round(filled / answered * 100) : 0 }
})
const operatorOptions = computed(() => {
  const set = new Set(calls.value.map(opExt).filter(Boolean))
  return [...set].sort((a, b) => Number(a) - Number(b)).map((ext) => ({ ext, name: names.value[ext] || `Operator ${ext}` }))
})

const totalPages = computed(() => Math.max(1, Math.ceil(filtered.value.length / PAGE_SIZE)))
const paged = computed(() => filtered.value.slice((page.value - 1) * PAGE_SIZE, page.value * PAGE_SIZE))
watch([fOperator, fCompany, fDirection, fStatus, fAnketa, fMinTalk, fPhone], () => { page.value = 1 })
function resetFilters() { fOperator.value = ''; fCompany.value = ''; fDirection.value = ''; fStatus.value = ''; fAnketa.value = ''; fMinTalk.value = 0; fPhone.value = '' }

function copyReview(c) {
  const url = `${window.location.origin}/feedback/${c.uuid}`
  try { navigator.clipboard.writeText(url); flash(t('calls.reviewCopied')) } catch { flash(url) }
}
function openFill(c) { fillActive.value = c; answers.value = {} }
function closeFill() { fillActive.value = null }
async function submitFill() {
  const a = answers.value
  if (!a.reason_key) { flash(t('survey.pickReason')); return }
  if (!a.status) { flash(t('survey.pickStatus')); return }
  const reason = (config.value.reasons || []).find((r) => r.key === a.reason_key)
  if (reason && reason.required && !(a.comment || '').trim()) { flash(t('survey.commentRequired')); return }
  saving.value = true
  try {
    await api.surveySubmit({ call_uuid: fillActive.value.uuid, operator_ext: opExt(fillActive.value), answers: a })
    responses.value = new Set([...responses.value, fillActive.value.uuid])
    flash(t('survey.saved')); closeFill()
  } catch (e) { flash(t('common.errorPrefix') + e.message) }
  finally { saving.value = false }
}

const companies = COMPANIES
onMounted(() => applyPreset('week'))
</script>

<template>
  <div class="calls">
    <div class="top"><div><h1>{{ t('nav.calls') }}</h1><p>{{ t('calls.subtitle') }}</p></div></div>
    <Teleport to="body"><Transition name="page"><div v-if="msg" class="toast">{{ msg }}</div></Transition></Teleport>

    <div class="kpis">
      <div class="kpi card"><div class="kpi__v">{{ kpi.total }}</div><div class="kpi__l">{{ t('st.total') }}</div></div>
      <div class="kpi card"><div class="kpi__v" style="color:var(--green)">{{ kpi.answered }}</div><div class="kpi__l">{{ t('st.answered') }}</div></div>
      <div class="kpi card"><div class="kpi__v" style="color:var(--red)">{{ kpi.missed }}</div><div class="kpi__l">{{ t('st.missed') }}</div></div>
      <div class="kpi card"><div class="kpi__v" style="color:var(--accent)">{{ kpi.cov }}%</div><div class="kpi__l">{{ t('survey.coverage') }}</div></div>
    </div>

    <div class="card filters">
      <div class="fl-presets">
        <button v-for="p in [['today','common.today'],['yesterday','common.yesterday'],['week','common.week'],['month','common.month']]" :key="p[0]"
                class="preset" :class="{ active: preset === p[0] }" @click="applyPreset(p[0])">{{ t(p[1]) }}</button>
        <span class="fl-range mono">{{ rangeLabel }}</span>
      </div>
      <div class="fl-grid">
        <label class="fld"><span>{{ t('common.from') }}</span><input type="date" v-model="fromInput" @change="preset='custom'; load()" /></label>
        <label class="fld"><span>{{ t('common.to') }}</span><input type="date" v-model="toInput" @change="preset='custom'; load()" /></label>
        <label class="fld"><span>{{ t('common.operator') }}</span><select v-model="fOperator"><option value="">{{ t('common.all') }}</option><option v-for="o in operatorOptions" :key="o.ext" :value="o.ext">{{ o.name }} · #{{ o.ext }}</option></select></label>
        <label class="fld"><span>{{ t('common.channel') }}</span><select v-model="fCompany"><option value="">{{ t('common.all') }}</option><option v-for="c in companies.filter(x=>x.id)" :key="c.id" :value="c.id">{{ c.name }}</option></select></label>
        <label class="fld"><span>{{ t('common.direction') }}</span><select v-model="fDirection"><option value="">{{ t('common.all') }}</option><option value="inbound">{{ t('dash.incoming') }}</option><option value="outbound">{{ t('dash.outgoing') }}</option></select></label>
        <label class="fld"><span>{{ t('tv.status') }}</span><select v-model="fStatus"><option value="">{{ t('common.all') }}</option><option value="answered">{{ t('st.answered') }}</option><option value="missed">{{ t('st.missed') }}</option></select></label>
        <label class="fld"><span>{{ t('nav.survey') }}</span><select v-model="fAnketa"><option value="">{{ t('common.all') }}</option><option value="yes">{{ t('survey.filled') }}</option><option value="no">{{ t('survey.unfilled') }}</option></select></label>
        <label class="fld"><span>{{ t('common.phone') }}</span><input v-model="fPhone" placeholder="998…" /></label>
        <label class="fld"><span>{{ t('calls.minTalk') }}</span><input type="number" min="0" v-model="fMinTalk" placeholder="0" /></label>
        <button class="fl-reset" @click="resetFilters">{{ t('common.reset') }}</button>
      </div>
    </div>

    <div class="section-h">
      <h2>{{ t('common.list') }} <span class="count">{{ filtered.length }}</span></h2>
      <div class="pager" v-if="totalPages > 1">
        <button :disabled="page<=1" @click="page--">←</button><span class="mono">{{ page }} / {{ totalPages }}</span><button :disabled="page>=totalPages" @click="page++">→</button>
      </div>
    </div>

    <div v-if="loading" class="loading"><i class="spin"></i></div>
    <div v-else class="card tbl-wrap">
      <table class="tbl">
        <thead><tr><th>{{ t('common.date') }}</th><th>{{ t('common.channel') }}</th><th>{{ t('common.client') }}</th><th>{{ t('common.operator') }}</th><th class="ta-c">{{ t('common.direction') }}</th><th class="ta-r">{{ t('common.talk') }}</th><th class="ta-c">{{ t('nav.survey') }}</th><th>{{ t('common.audio') }}</th><th class="ta-r">{{ t('common.action') }}</th></tr></thead>
        <tbody>
          <tr v-for="c in paged" :key="c.uuid">
            <td class="mono dim">{{ fmtDateTime(c.start_stamp) }}</td>
            <td><span class="cbadge" :class="callCompany(c)">{{ compBadge(callCompany(c)) }}</span></td>
            <td class="mono">{{ clientNumber(c) }}</td>
            <td><div class="top-op"><span class="tav">{{ (names[opExt(c)] || opExt(c)).slice(0,2).toUpperCase() }}</span><span>{{ opName(c) }}</span></div></td>
            <td class="ta-c"><span class="dir" :class="c.direction === 'outbound' ? 'out' : 'in'">{{ c.direction === 'outbound' ? t('common.dirOut') : t('common.dirIn') }}</span></td>
            <td class="ta-r mono" :class="{ miss: !isAnswered(c) }">{{ isAnswered(c) ? fmtDuration(c.user_talk_time) : '—' }}</td>
            <td class="ta-c"><span v-if="isFilled(c)" class="ank ank--yes">✓</span><span v-else-if="isAnswered(c)" class="ank ank--no">—</span><span v-else class="ank">·</span></td>
            <td><audio v-if="isAnswered(c)" class="rec" controls preload="none" :src="api.recordingUrl(c.uuid)"></audio><span v-else class="dim">—</span></td>
            <td class="ta-r acts">
              <button class="link-btn" @click="copyReview(c)" :title="t('calls.copyReviewLink')">🔗</button>
              <button v-if="isAnswered(c) && !isFilled(c)" class="fill-btn" @click="openFill(c)">{{ t('survey.fill') }}</button>
            </td>
          </tr>
          <tr v-if="!paged.length"><td colspan="9" class="empty">{{ t('calls.empty') }}</td></tr>
        </tbody>
      </table>
    </div>

    <!-- to'ldirish modali -->
    <Teleport to="body">
      <Transition name="modal">
        <div v-if="fillActive" class="modal" @click.self="closeFill">
          <div class="modal__card">
            <div class="modal__head">
              <div><h3>{{ t('survey.fillTitle') }}</h3>
                <p class="opline">👤 <b>{{ opName(fillActive) }}</b> · #{{ opExt(fillActive) }}</p>
                <p class="mono nums">{{ clientNumber(fillActive) }} · {{ fmtDateTime(fillActive.start_stamp) }}</p>
              </div>
              <button class="modal__x" @click="closeFill">×</button>
            </div>
            <div class="modal__body"><AnketaForm :config="config" v-model="answers" /></div>
            <div class="modal__foot"><button class="btn-ghost" @click="closeFill">{{ t('common.cancel') }}</button><button @click="submitFill" :disabled="saving">{{ saving ? '...' : t('common.save') }}</button></div>
          </div>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<style scoped>
.calls { animation: fade-up 0.4s both; }
.top { margin: 16px 0 18px; }
.top h1 { font-size: 24px; font-weight: 800; }
.top p { color: var(--text-dim); font-size: 13px; margin-top: 4px; }
.toast { position: fixed; top: 22px; left: 50%; transform: translateX(-50%); z-index: 120; background: var(--grad); color: #fff; padding: 11px 22px; border-radius: 12px; font-weight: 600; box-shadow: var(--glow); }
.kpis { display: grid; grid-template-columns: repeat(4, 1fr); gap: 14px; margin-bottom: 18px; }
.kpi { padding: 18px 20px; }
.kpi__v { font-size: 28px; font-weight: 800; font-family: var(--mono); line-height: 1; }
.kpi__l { font-size: 12px; color: var(--text-dim); margin-top: 5px; }
.filters { padding: 16px 18px; margin-bottom: 22px; }
.fl-presets { display: flex; align-items: center; gap: 6px; margin-bottom: 14px; flex-wrap: wrap; }
.preset { background: var(--surface-2); color: var(--text-dim); padding: 7px 14px; font-size: 12.5px; border: 1px solid var(--border); }
.preset:hover { transform: none; box-shadow: none; color: var(--text); }
.preset.active { background: var(--grad); color: #fff; border-color: transparent; }
.fl-range { margin-left: auto; font-size: 12px; color: var(--text-faint); }
.fl-grid { display: grid; grid-template-columns: repeat(5, 1fr); gap: 12px; align-items: end; }
.fld { display: flex; flex-direction: column; gap: 6px; }
.fld span { font-size: 11.5px; font-weight: 600; color: var(--text-dim); }
.fld input, .fld select { width: 100%; }
.fl-reset { background: var(--surface-2); color: var(--text-dim); border: 1px solid var(--border); height: 38px; }
.fl-reset:hover { transform: none; box-shadow: none; color: var(--text); }
.section-h { display: flex; align-items: center; justify-content: space-between; margin: 8px 0 14px; }
.section-h h2 { font-size: 16px; font-weight: 700; display: flex; align-items: center; gap: 10px; }
.count { font-size: 12px; font-weight: 600; color: var(--text-dim); background: var(--surface-2); padding: 2px 9px; border-radius: 999px; }
.tbl-wrap { padding: 6px 8px; overflow-x: auto; }
.tbl { width: 100%; border-collapse: collapse; }
.tbl th { text-align: left; font-size: 11px; font-weight: 600; color: var(--text-faint); text-transform: uppercase; padding: 12px 14px; white-space: nowrap; }
.tbl td { padding: 10px 14px; border-top: 1px solid var(--border); font-size: 13px; white-space: nowrap; }
.ta-c { text-align: center; } .ta-r { text-align: right; }
.dim { color: var(--text-faint); }
.miss { color: var(--red); }
.cbadge { font-size: 11px; font-weight: 700; padding: 3px 9px; border-radius: 7px; background: var(--surface-2); color: var(--text-dim); }
.cbadge.salesdoc { background: rgba(16,185,129,0.15); color: var(--green); }
.cbadge.ibox { background: rgba(6,182,212,0.15); color: var(--accent-2); }
.top-op { display: flex; align-items: center; gap: 9px; }
.tav { width: 28px; height: 28px; border-radius: 8px; background: var(--grad-soft); color: var(--accent); display: grid; place-items: center; font-size: 11px; font-weight: 700; flex-shrink: 0; }
.dir { font-size: 11px; font-weight: 600; padding: 3px 10px; border-radius: 999px; }
.dir.in { background: rgba(16,185,129,0.14); color: var(--green); }
.dir.out { background: rgba(6,182,212,0.14); color: var(--accent-2); }
.ank { font-weight: 700; }
.ank--yes { color: var(--green); }
.ank--no { color: var(--amber); }
.rec { height: 34px; width: 230px; max-width: 230px; }
.fill-btn { padding: 7px 14px; font-size: 12.5px; }
.acts { display: flex; gap: 6px; align-items: center; justify-content: flex-end; }
.link-btn { background: var(--surface-2); border: 1px solid var(--border); padding: 6px 10px; font-size: 13px; border-radius: 8px; }
.link-btn:hover { transform: none; box-shadow: none; background: var(--surface-3); }
.pager { display: flex; align-items: center; gap: 10px; }
.pager button { width: 34px; height: 34px; padding: 0; background: var(--surface-2); border: 1px solid var(--border); color: var(--text-dim); }
.pager button:hover:not(:disabled) { color: var(--text); transform: none; box-shadow: none; }
.pager button:disabled { opacity: 0.4; cursor: not-allowed; }
.empty { text-align: center; color: var(--text-faint); padding: 30px; }
.loading { display: flex; justify-content: center; padding: 40px; }
.spin { width: 18px; height: 18px; border: 2.5px solid var(--border-strong); border-top-color: var(--accent); border-radius: 50%; animation: spin 0.8s linear infinite; }
.modal { position: fixed; inset: 0; z-index: 100; background: rgba(8,10,18,0.55); backdrop-filter: blur(4px); display: grid; place-items: center; padding: 20px; }
.modal__card { width: 480px; max-width: 100%; max-height: 90vh; display: flex; flex-direction: column; background: var(--surface); border: 1px solid var(--border); border-radius: var(--radius-lg); box-shadow: var(--shadow-lg); overflow: hidden; }
.modal__head { display: flex; justify-content: space-between; align-items: center; gap: 12px; padding: 20px 22px; border-bottom: 1px solid var(--border); }
.modal__head h3 { font-size: 17px; font-weight: 700; }
.opline { font-size: 13px; margin-top: 3px; }
.nums { font-size: 12px; color: var(--text-dim); margin-top: 2px; }
.modal__x { background: var(--surface-2); color: var(--text-faint); font-size: 22px; width: 34px; height: 34px; padding: 0; border-radius: 10px; line-height: 1; }
.modal__x:hover { color: var(--text); transform: none; box-shadow: none; }
.modal__body { padding: 22px; overflow-y: auto; }
.modal__foot { display: flex; justify-content: flex-end; gap: 10px; padding: 18px 22px; border-top: 1px solid var(--border); }
.modal-enter-active, .modal-leave-active { transition: opacity 0.25s; }
.modal-enter-from, .modal-leave-to { opacity: 0; }
@media (max-width: 1080px) { .kpis, .fl-grid { grid-template-columns: repeat(2, 1fr); } }
</style>
