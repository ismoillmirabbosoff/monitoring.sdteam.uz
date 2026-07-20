<script setup>
import { ref, onMounted } from 'vue'
import { api } from '../api.js'

const entries = ref([])
const actions = ref([])
const fAction = ref('')
const fQ = ref('')
const loading = ref(true)
const msg = ref('')
function flash(t) { msg.value = t; setTimeout(() => (msg.value = ''), 3000) }

async function load() {
  loading.value = true
  try {
    const r = await api.auditLog({ action: fAction.value, q: fQ.value, limit: 300 })
    entries.value = r.entries || []
    actions.value = r.actions || []
  } catch (e) { flash('Xato: ' + e.message) }
  finally { loading.value = false }
}

function fmt(ts) {
  const d = new Date(ts)
  const p = (n) => String(n).padStart(2, '0')
  return `${p(d.getDate())}.${p(d.getMonth() + 1)}.${d.getFullYear()} ${p(d.getHours())}:${p(d.getMinutes())}:${p(d.getSeconds())}`
}
const methodColor = { POST: 'ok', PATCH: 'warn', PUT: 'warn', DELETE: 'bad' }

onMounted(load)
</script>

<template>
  <div class="al">
    <div class="top">
      <div><h1>Amallar jurnali</h1><p>{{ entries.length }} ta yozuv · admin o'zgartirishlari</p></div>
    </div>
    <Transition name="page"><div v-if="msg" class="toast">{{ msg }}</div></Transition>

    <div class="card filters">
      <label class="fld"><span>Amal</span>
        <select v-model="fAction" @change="load"><option value="">Hammasi</option><option v-for="a in actions" :key="a" :value="a">{{ a }}</option></select>
      </label>
      <label class="fld grow"><span>Qidiruv (foydalanuvchi / yo'l)</span>
        <input v-model="fQ" @keyup.enter="load" placeholder="ism yoki /api/admin/…" />
      </label>
      <button @click="load">Qidirish</button>
    </div>

    <div v-if="loading" class="loading"><i class="spin"></i></div>
    <div v-else class="card tbl-wrap">
      <table class="tbl">
        <thead><tr><th>Vaqt</th><th>Foydalanuvchi</th><th>Amal</th><th class="ta-c">Metod</th><th>Yo'l</th><th>IP</th></tr></thead>
        <tbody>
          <tr v-for="e in entries" :key="e.id">
            <td class="mono dim">{{ fmt(e.created_at) }}</td>
            <td>{{ e.user_name || '—' }}</td>
            <td><span class="act">{{ e.action }}</span></td>
            <td class="ta-c"><span class="mth" :class="methodColor[e.method]">{{ e.method }}</span></td>
            <td class="mono path">{{ e.path }}</td>
            <td class="mono dim">{{ e.ip }}</td>
          </tr>
          <tr v-if="!entries.length"><td colspan="6" class="empty">Yozuv yo'q</td></tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<style scoped>
.al { animation: fade-up 0.4s both; }
.top { margin: 16px 0 18px; }
.top h1 { font-size: 24px; font-weight: 800; }
.top p { color: var(--text-dim); font-size: 13px; margin-top: 4px; }
.toast { position: fixed; top: 22px; left: 50%; transform: translateX(-50%); z-index: 60; background: var(--grad); color: #fff; padding: 11px 22px; border-radius: 12px; font-weight: 600; box-shadow: var(--glow); }
.filters { display: flex; gap: 12px; align-items: end; padding: 16px 18px; margin-bottom: 20px; }
.fld { display: flex; flex-direction: column; gap: 6px; }
.fld.grow { flex: 1; }
.fld span { font-size: 11.5px; font-weight: 600; color: var(--text-dim); }
.fld input, .fld select { width: 100%; }
.tbl-wrap { padding: 6px 8px; overflow-x: auto; }
.tbl { width: 100%; border-collapse: collapse; }
.tbl th { text-align: left; font-size: 11px; font-weight: 600; color: var(--text-faint); text-transform: uppercase; padding: 12px 14px; white-space: nowrap; }
.tbl td { padding: 10px 14px; border-top: 1px solid var(--border); font-size: 13px; }
.ta-c { text-align: center; }
.dim { color: var(--text-faint); }
.path { font-size: 12px; color: var(--text-dim); }
.act { font-size: 11.5px; font-weight: 600; color: var(--accent); background: var(--grad-soft); padding: 3px 10px; border-radius: 999px; }
.mth { font-size: 10.5px; font-weight: 700; padding: 3px 9px; border-radius: 6px; background: var(--surface-2); color: var(--text-dim); }
.mth.ok { background: rgba(16,185,129,0.15); color: var(--green); }
.mth.warn { background: rgba(245,158,11,0.15); color: var(--amber); }
.mth.bad { background: rgba(239,68,68,0.15); color: var(--red); }
.empty { text-align: center; color: var(--text-faint); padding: 30px; }
.loading { display: flex; justify-content: center; padding: 40px; }
.spin { width: 18px; height: 18px; border: 2.5px solid var(--border-strong); border-top-color: var(--accent); border-radius: 50%; animation: spin 0.8s linear infinite; }
</style>
