<script setup>
import { ref, computed, onMounted } from 'vue'
import { api, COMPANIES, companyForQueue } from '../api.js'
import { auth } from '../auth.js'
import { t } from '../i18n.js'

const users = ref([])
const operators = ref([])
const hidden = ref(new Set())   // yashiringan operator ext'lari
const search = ref('')
const company = ref('')          // '' | 'salesdoc' | 'ibox'
const companies = COMPANIES
const msg = ref('')
const showForm = ref(false)
const nf = ref({ email: '', password: '', name: '', role: 'operator', ext: '' })
const editing = ref(null)
const ef = ref({ name: '', role: 'operator', ext: '', password: '', active: true })

function flash(t) { msg.value = t; setTimeout(() => (msg.value = ''), 3000) }

async function load() {
  try {
    users.value = await api.userList()
    operators.value = await api.users() // OnlinePBX operatorlari (ext->name)
    try { hidden.value = new Set((await api.hidden()) || []) } catch {}
  } catch (e) { flash(t('common.errorPrefix') + e.message) }
}

// ext -> kompaniya (OnlinePBX operatorlarining tr1 navbatidan)
const extCompany = computed(() => {
  const m = {}
  for (const o of operators.value) if (o.num) m[String(o.num)] = companyForQueue(o.tr1)
  return m
})
function userCompany(u) { return u.ext ? (extCompany.value[String(u.ext)] || '') : '' }

const filtered = computed(() => {
  const q = search.value.trim().toLowerCase()
  let list = users.value
  if (company.value) list = list.filter((u) => userCompany(u) === company.value)
  if (q) list = list.filter((u) =>
    (u.name || '').toLowerCase().includes(q) ||
    (u.email || '').toLowerCase().includes(q) ||
    String(u.ext || '').includes(q))
  return list
})

function isHidden(u) { return !!u.ext && hidden.value.has(String(u.ext)) }
async function toggleHide(u) {
  if (!u.ext) { flash(t('users.noExtHide')); return }
  const next = !isHidden(u)
  try {
    await api.setHiddenByExt(u.ext, next)
    const s = new Set(hidden.value)
    next ? s.add(String(u.ext)) : s.delete(String(u.ext))
    hidden.value = s
    flash(`${u.name || u.ext} ${next ? t('admin.hidden') : t('admin.shown')}`)
  } catch (e) { flash(t('common.errorPrefix') + e.message) }
}

async function create() {
  if (!nf.value.email || !nf.value.password) { flash(t('users.emailPasswordRequired')); return }
  try {
    await api.userCreate({ ...nf.value })
    flash(t('users.created'))
    nf.value = { email: '', password: '', name: '', role: 'operator', ext: '' }
    showForm.value = false
    await load()
  } catch (e) { flash(t('common.errorPrefix') + e.message) }
}

function startEdit(u) {
  editing.value = u.id
  ef.value = { name: u.name, role: u.role, ext: u.ext || '', password: '', active: u.active }
}
async function saveEdit(u) {
  try {
    const payload = { name: ef.value.name, role: ef.value.role, ext: ef.value.ext, active: ef.value.active }
    if (ef.value.password) payload.password = ef.value.password
    await api.userUpdate(u.id, payload)
    flash(t('common.saved'))
    editing.value = null
    await load()
  } catch (e) { flash(t('common.errorPrefix') + e.message) }
}
function copyPw(pw) {
  try { navigator.clipboard.writeText(pw); flash(t('users.passwordCopied')) } catch {}
}
async function remove(u) {
  if (u.id === auth.user?.id) { flash(t('users.cannotDeleteSelf')); return }
  try { await api.userDelete(u.id); await load() } catch (e) { flash(t('common.errorPrefix') + e.message) }
}

onMounted(load)
</script>

<template>
  <div class="users">
    <div class="top">
      <div>
        <h1>{{ t('nav.staff') }}</h1>
        <p>{{ users.length }} · {{ t('users.canLogin') }}</p>
      </div>
      <div class="top__actions">
        <div class="cfilter">
          <button v-for="c in companies" :key="c.id" class="cfilter__btn"
                  :class="{ active: company === c.id }" @click="company = c.id"
                  :style="company === c.id ? { '--c': c.color } : {}">
            <i v-if="c.id" :style="{ background: c.color }"></i>{{ c.id ? c.name : t('common.all') }}
          </button>
        </div>
        <input v-model="search" class="search" :placeholder="t('users.searchPlaceholder')" />
        <button @click="showForm = !showForm">{{ showForm ? t('common.close') : t('users.addStaff') }}</button>
      </div>
    </div>

    <Transition name="page"><div v-if="msg" class="toast">{{ msg }}</div></Transition>

    <Transition name="page">
      <form v-if="showForm" class="card cform" @submit.prevent="create">
        <label class="fld"><span>Email</span><input v-model="nf.email" type="email" placeholder="operator@salesdoc.io" /></label>
        <label class="fld"><span>{{ t('users.password') }}</span><input v-model="nf.password" type="text" :placeholder="t('users.initialPassword')" /></label>
        <label class="fld"><span>{{ t('st.name') }}</span><input v-model="nf.name" :placeholder="t('users.fullName')" /></label>
        <label class="fld"><span>{{ t('users.role') }}</span>
          <select v-model="nf.role"><option value="operator">{{ t('role.operator') }}</option><option value="admin">{{ t('role.adminShort') }}</option></select>
        </label>
        <label class="fld"><span>{{ t('users.operatorExt') }}</span>
          <select v-model="nf.ext">
            <option value="">{{ t('users.noneDash') }}</option>
            <option v-for="o in operators" :key="o.num" :value="o.num">{{ o.num }} · {{ o.name }}</option>
          </select>
        </label>
        <button type="submit">{{ t('common.create') }}</button>
      </form>
    </Transition>

    <div class="card tbl-wrap">
      <table class="tbl">
        <thead><tr><th>{{ t('common.user') }}</th><th>Email</th><th>{{ t('users.role') }}</th><th>{{ t('common.operator') }}</th><th>{{ t('users.password') }}</th><th class="ta-c">{{ t('tv.status') }}</th><th class="ta-c">{{ t('users.tvPanel') }}</th><th></th></tr></thead>
        <tbody>
          <tr v-for="u in filtered" :key="u.id" :class="{ inactive: !u.active }">
            <td>
              <div class="u-name">
                <span class="u-av">{{ (u.name || u.email).slice(0,2).toUpperCase() }}</span>
                <template v-if="editing === u.id"><input v-model="ef.name" class="mini" /></template>
                <template v-else>{{ u.name || '—' }}</template>
              </div>
            </td>
            <td class="u-email">{{ u.email }}</td>
            <td>
              <select v-if="editing === u.id" v-model="ef.role" class="mini"><option value="operator">{{ t('role.operator') }}</option><option value="admin">{{ t('role.adminShort') }}</option></select>
              <span v-else class="role" :class="u.role">{{ u.role === 'admin' ? t('role.adminShort') : t('role.operator') }}</span>
            </td>
            <td>
              <select v-if="editing === u.id" v-model="ef.ext" class="mini">
                <option value="">—</option><option v-for="o in operators" :key="o.num" :value="o.num">{{ o.num }}</option>
              </select>
              <span v-else class="mono">{{ u.ext || '—' }}</span>
            </td>
            <td class="u-pw">
              <span v-if="u.initial_password" class="pw-badge mono" @click="copyPw(u.initial_password)" :title="t('users.copy')">{{ u.initial_password }}</span>
              <span v-else class="mono dim">—</span>
            </td>
            <td class="ta-c">
              <label v-if="editing === u.id" class="chk"><input type="checkbox" v-model="ef.active" /> {{ t('users.active') }}</label>
              <span v-else class="dot" :class="{ on: u.active }"></span>
            </td>
            <td class="ta-c">
              <button class="eye" :class="{ off: isHidden(u) }" @click="toggleHide(u)" :disabled="!u.ext"
                      :title="isHidden(u) ? t('admin.show') : t('users.hideTvPanel')">
                <svg v-if="!isHidden(u)" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/></svg>
                <svg v-else viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24"/><path d="M1 1l22 22"/></svg>
              </button>
            </td>
            <td class="u-act">
              <template v-if="editing === u.id">
                <input v-model="ef.password" type="text" :placeholder="t('users.newPasswordOpt')" class="mini pw" />
                <button class="mini-btn" @click="saveEdit(u)">{{ t('common.save') }}</button>
                <button class="mini-btn ghost" @click="editing = null">×</button>
              </template>
              <template v-else>
                <button class="mini-btn ghost" @click="startEdit(u)">✎</button>
                <button class="mini-btn del" @click="remove(u)">×</button>
              </template>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<style scoped>
.users { animation: fade-up 0.4s both; }
.top { display: flex; justify-content: space-between; align-items: flex-start; margin: 16px 0 20px; }
.top h1 { font-size: 24px; font-weight: 800; }
.top p { color: var(--text-dim); font-size: 13px; margin-top: 4px; }
.top__actions { display: flex; gap: 10px; align-items: center; flex-wrap: wrap; justify-content: flex-end; }
.search { width: 280px; max-width: 40vw; padding: 9px 13px; font-size: 13px; }
.cfilter { display: flex; gap: 4px; background: var(--surface); padding: 5px; border-radius: 12px; border: 1px solid var(--border); }
.cfilter__btn { display: flex; align-items: center; gap: 7px; padding: 8px 14px; border-radius: 9px;
  font-size: 12.5px; font-weight: 600; background: transparent; border: none; color: var(--text-dim); }
.cfilter__btn i { width: 8px; height: 8px; border-radius: 50%; }
.cfilter__btn:hover { color: var(--text); transform: none; box-shadow: none; }
.cfilter__btn.active { background: var(--surface-2); color: var(--text); }
.toast { position: fixed; top: 22px; left: 50%; transform: translateX(-50%); z-index: 50;
  background: var(--grad); color: #fff; padding: 11px 22px; border-radius: 12px; font-size: 13.5px; font-weight: 600; box-shadow: var(--glow); }
.cform { display: grid; grid-template-columns: repeat(5, 1fr) auto; gap: 14px; align-items: end; padding: 18px; margin-bottom: 18px; }
.fld { display: flex; flex-direction: column; gap: 6px; }
.fld span { font-size: 11.5px; font-weight: 600; color: var(--text-dim); }
.fld input, .fld select { width: 100%; }

.tbl-wrap { padding: 6px 8px; overflow-x: auto; }
.tbl { width: 100%; border-collapse: collapse; }
.tbl th { text-align: left; font-size: 11px; font-weight: 600; color: var(--text-faint); text-transform: uppercase; padding: 13px 14px; }
.tbl td { padding: 11px 14px; border-top: 1px solid var(--border); font-size: 13.5px; }
.tbl tr.inactive { opacity: 0.5; }
.ta-c { text-align: center; }
.u-name { display: flex; align-items: center; gap: 10px; font-weight: 600; }
.u-av { width: 32px; height: 32px; border-radius: 9px; background: var(--grad-soft); color: var(--accent); display: grid; place-items: center; font-size: 12px; font-weight: 700; }
.u-email { color: var(--text-dim); font-size: 12.5px; }
.role { font-size: 11px; font-weight: 600; padding: 3px 10px; border-radius: 999px; }
.role.admin { background: rgba(109,94,252,0.16); color: var(--accent); }
.role.operator { background: var(--surface-2); color: var(--text-dim); }
.dot { display: inline-block; width: 10px; height: 10px; border-radius: 50%; background: var(--gray); }
.dot.on { background: var(--green); }
.u-pw .pw-badge { font-size: 12px; background: var(--surface-2); color: var(--text-dim);
  padding: 3px 9px; border-radius: 7px; cursor: pointer; border: 1px solid var(--border); }
.u-pw .pw-badge:hover { color: var(--text); background: var(--surface-3); }
.dim { color: var(--text-faint); }
.eye { width: 32px; height: 32px; padding: 0; background: var(--surface-2); border: 1px solid var(--border);
  color: var(--text-dim); display: inline-grid; place-items: center; }
.eye:hover { color: var(--text); background: var(--surface-3); transform: none; box-shadow: none; }
.eye.off { color: var(--amber); }
.eye:disabled { opacity: 0.4; cursor: not-allowed; }
.eye svg { width: 15px; height: 15px; }
.u-act { display: flex; gap: 6px; align-items: center; justify-content: flex-end; }
.mini { padding: 6px 9px; font-size: 12.5px; }
.mini.pw { width: 140px; }
.mini-btn { padding: 6px 10px; font-size: 12px; }
.mini-btn.ghost { background: var(--surface-2); color: var(--text-dim); }
.mini-btn.del { background: rgba(239,68,68,0.14); color: var(--red); }
.mini-btn.ghost:hover, .mini-btn.del:hover { box-shadow: none; transform: none; }
.chk { font-size: 12px; color: var(--text-dim); display: inline-flex; gap: 5px; align-items: center; }
</style>
