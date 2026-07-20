<script setup>
import { ref, computed, onMounted } from 'vue'
import { api, isExtension, fmtDuration, fmtTime, todayStr } from '../api.js'
import { auth } from '../auth.js'
import AnketaForm from '../components/AnketaForm.vue'

const calls = ref([])
const responded = ref(new Map()) // call_uuid -> response
const names = ref({})            // ext -> operator ismi
const config = ref({ reasons: [], common_modules: [], payment_topics: [], statuses: [] })
function opName(c) { const e = opExt(c); return names.value[e] || ('Operator ' + e) }
const loading = ref(true)
const tab = ref('todo') // todo | done | all
const day = ref(todayStr())
const msg = ref('')

// modal
const active = ref(null) // joriy call
const answers = ref({})
const saving = ref(false)

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

async function load() {
  loading.value = true
  try {
    const [from, to] = dayRange()
    const [cs, rs, cfg, us] = await Promise.all([
      api.data('', from, to),
      api.surveyResponses(from, to),
      api.surveyConfig(),
      api.users().catch(() => []),
    ])
    config.value = cfg || config.value
    const nm = {}; for (const u of us || []) { if (u.num) nm[String(u.num)] = u.name }
    names.value = nm
    const map = new Map()
    for (const r of rs) map.set(r.call_uuid, r)
    responded.value = map
    // faqat suhbatli (javob berilgan) qo'ng'iroqlar anketaga muhtoj
    let list = cs.filter((c) => (c.user_talk_time || 0) > 0 && opExt(c))
    if (!auth.isAdmin && auth.user?.ext) list = list.filter((c) => opExt(c) === auth.user.ext)
    calls.value = list.sort((a, b) => b.start_stamp - a.start_stamp)
  } catch (e) { flash('Xato: ' + e.message) }
  finally { loading.value = false }
}

const filtered = computed(() => {
  if (tab.value === 'todo') return calls.value.filter((c) => !responded.value.has(c.uuid))
  if (tab.value === 'done') return calls.value.filter((c) => responded.value.has(c.uuid))
  return calls.value
})
const counts = computed(() => ({
  todo: calls.value.filter((c) => !responded.value.has(c.uuid)).length,
  done: calls.value.filter((c) => responded.value.has(c.uuid)).length,
  all: calls.value.length,
}))
const coverage = computed(() => calls.value.length ? Math.round(counts.value.done / calls.value.length * 100) : 0)

function openFill(c) {
  active.value = c
  const existing = responded.value.get(c.uuid)
  answers.value = existing ? { ...(typeof existing.answers === 'string' ? JSON.parse(existing.answers) : existing.answers) } : {}
}
function close() { active.value = null }

async function submit() {
  const a = answers.value
  if (!a.reason_key) { flash('Причина обращения tanlang'); return }
  if (!a.status) { flash('Статус tanlang'); return }
  const reason = (config.value.reasons || []).find((r) => r.key === a.reason_key)
  if (reason && reason.required && !(a.comment || '').trim()) { flash('Комментарий majburiy'); return }
  saving.value = true
  try {
    await api.surveySubmit({ call_uuid: active.value.uuid, operator_ext: opExt(active.value), answers: answers.value })
    flash('Anketa saqlandi')
    close()
    await load()
  } catch (e) { flash('Xato: ' + e.message) }
  finally { saving.value = false }
}

onMounted(load)
</script>

<template>
  <div class="surv">
    <div class="top">
      <div>
        <h1>Anketa</h1>
        <p>{{ auth.isAdmin ? 'Barcha operatorlar' : 'Mening qo\'ng\'iroqlarim' }} · qoplanish {{ coverage }}%</p>
      </div>
      <input type="date" v-model="day" @change="load" />
    </div>

    <Teleport to="body"><Transition name="modal"><div v-if="msg" class="toast">{{ msg }}</div></Transition></Teleport>

    <div class="tabs">
      <button :class="{ active: tab === 'todo' }" @click="tab = 'todo'">Anketasiz <b>{{ counts.todo }}</b></button>
      <button :class="{ active: tab === 'done' }" @click="tab = 'done'">To'ldirilgan <b>{{ counts.done }}</b></button>
      <button :class="{ active: tab === 'all' }" @click="tab = 'all'">Hammasi <b>{{ counts.all }}</b></button>
    </div>

    <div v-if="loading" class="loading"><i class="spin"></i> Yuklanmoqda…</div>
    <div v-else class="card list">
      <div v-for="c in filtered" :key="c.uuid" class="row" :class="{ done: responded.has(c.uuid) }">
        <div class="row__dir" :class="c.direction === 'outbound' ? 'out' : 'in'">
          <svg v-if="c.direction === 'outbound'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><path d="M7 17 17 7M17 7H9M17 7v8"/></svg>
          <svg v-else viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><path d="M17 7 7 17M7 17h8M7 17V9"/></svg>
        </div>
        <div class="row__main">
          <div class="row__num mono">{{ c.caller_id_number }} → {{ c.destination_number }}</div>
          <div class="row__meta"><b>{{ opName(c) }}</b> · #{{ opExt(c) }} · {{ fmtDuration(c.user_talk_time) }} · {{ fmtTime(c.start_stamp) }}</div>
        </div>
        <button v-if="responded.has(c.uuid)" class="row__btn done-btn" @click="openFill(c)">✓ Ko'rish</button>
        <button v-else class="row__btn" @click="openFill(c)">To'ldirish</button>
      </div>
      <div v-if="!filtered.length" class="empty">{{ tab === 'todo' ? 'Barcha anketalar to\'ldirilgan 🎉' : 'Yozuv yo\'q' }}</div>
    </div>

    <!-- Modal (Teleport -> body, transform-containing-block muammosidan qochish) -->
    <Teleport to="body">
      <Transition name="modal">
        <div v-if="active" class="modal" @click.self="close">
          <div class="modal__card">
            <div class="modal__head">
              <div class="modal__hl">
                <div class="modal__ico" :class="active.direction === 'outbound' ? 'out' : 'in'">
                  <svg v-if="active.direction === 'outbound'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><path d="M7 17 17 7M17 7H9M17 7v8"/></svg>
                  <svg v-else viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><path d="M17 7 7 17M7 17h8M7 17V9"/></svg>
                </div>
                <div>
                  <h3>Anketa to'ldirish</h3>
                  <p class="opline">👤 <b>{{ opName(active) }}</b> · #{{ opExt(active) }}</p>
                  <p class="mono nums">{{ active.caller_id_number }} → {{ active.destination_number }}</p>
                </div>
              </div>
              <button class="modal__x" @click="close">×</button>
            </div>
            <div class="modal__body">
              <AnketaForm :config="config" v-model="answers" />
            </div>
            <div class="modal__foot">
              <button class="btn-ghost" @click="close">Bekor</button>
              <button @click="submit" :disabled="saving">{{ saving ? '...' : 'Saqlash' }}</button>
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<style scoped>
.surv { animation: fade-up 0.4s both; }
.top { display: flex; justify-content: space-between; align-items: flex-start; margin: 16px 0 20px; }
.top h1 { font-size: 24px; font-weight: 800; }
.top p { color: var(--text-dim); font-size: 13px; margin-top: 4px; }
.toast { position: fixed; top: 22px; left: 50%; transform: translateX(-50%); z-index: 60;
  background: var(--grad); color: #fff; padding: 11px 22px; border-radius: 12px; font-weight: 600; box-shadow: var(--glow); }

.tabs { display: inline-flex; gap: 4px; background: var(--surface); padding: 5px; border-radius: 12px; border: 1px solid var(--border); margin-bottom: 18px; }
.tabs button { background: transparent; color: var(--text-dim); padding: 9px 16px; font-size: 13px; }
.tabs button:hover { transform: none; box-shadow: none; color: var(--text); }
.tabs button.active { background: var(--grad); color: #fff; }
.tabs b { margin-left: 4px; }

.list { padding: 6px 10px; }
.row { display: flex; align-items: center; gap: 14px; padding: 12px 8px; border-top: 1px solid var(--border); }
.row:first-child { border-top: none; }
.row.done { opacity: 0.7; }
.row__dir { width: 34px; height: 34px; border-radius: 10px; display: grid; place-items: center; flex-shrink: 0; }
.row__dir svg { width: 17px; height: 17px; }
.modal__ico svg { width: 20px; height: 20px; }
.row__dir.in { background: rgba(16,185,129,0.14); color: var(--green); }
.row__dir.out { background: rgba(20,184,196,0.14); color: var(--accent-2); }
.row__main { flex: 1; min-width: 0; }
.row__num { font-size: 14px; font-weight: 600; }
.row__meta { font-size: 11.5px; color: var(--text-faint); margin-top: 2px; }
.row__btn { padding: 8px 16px; font-size: 12.5px; }
.row__btn.done-btn { background: var(--surface-2); color: var(--green); }
.row__btn.done-btn:hover { box-shadow: none; }
.empty { text-align: center; color: var(--text-faint); padding: 44px; font-size: 14px; }
.loading { display: flex; align-items: center; justify-content: center; gap: 10px; padding: 50px; color: var(--text-dim); }
.spin { width: 18px; height: 18px; border: 2.5px solid var(--border-strong); border-top-color: var(--accent); border-radius: 50%; animation: spin 0.8s linear infinite; }

.modal { position: fixed; inset: 0; z-index: 100; background: rgba(8,10,18,0.55); backdrop-filter: blur(4px);
  display: grid; place-items: center; padding: 20px; }
.modal__card { width: 480px; max-width: 100%; max-height: 90vh; display: flex; flex-direction: column;
  background: var(--surface); border: 1px solid var(--border); border-radius: var(--radius-lg);
  box-shadow: var(--shadow-lg); overflow: hidden; }
.modal__head { display: flex; justify-content: space-between; align-items: center; gap: 12px;
  padding: 20px 22px; border-bottom: 1px solid var(--border); }
.modal__hl { display: flex; align-items: center; gap: 13px; }
.modal__ico { width: 42px; height: 42px; border-radius: 12px; display: grid; place-items: center; font-size: 20px; flex-shrink: 0; }
.modal__ico.in { background: var(--green-soft); color: var(--green); }
.modal__ico.out { background: rgba(6,182,212,0.14); color: var(--accent-2); }
.modal__head h3 { font-size: 17px; font-weight: 700; }
.modal__head p { font-size: 12px; color: var(--text-dim); margin-top: 3px; }
.modal__x { background: var(--surface-2); color: var(--text-faint); font-size: 22px; width: 34px; height: 34px;
  padding: 0; border-radius: 10px; flex-shrink: 0; line-height: 1; }
.modal__x:hover { color: var(--text); transform: none; box-shadow: none; background: var(--surface-3); }
.modal__body { padding: 22px; overflow-y: auto; }
.modal__foot { display: flex; justify-content: flex-end; gap: 10px; padding: 18px 22px; border-top: 1px solid var(--border); }

.modal-enter-active, .modal-leave-active { transition: opacity 0.25s; }
.modal-enter-active .modal__card, .modal-leave-active .modal__card { transition: transform 0.3s cubic-bezier(.2,.9,.3,1.2), opacity 0.25s; }
.modal-enter-from, .modal-leave-to { opacity: 0; }
.modal-enter-from .modal__card, .modal-leave-to .modal__card { transform: scale(0.92) translateY(20px); opacity: 0; }
</style>
