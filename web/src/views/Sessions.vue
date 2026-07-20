<script setup>
import { ref, onMounted } from 'vue'
import { api } from '../api.js'
import { auth } from '../auth.js'
import { t } from '../i18n.js'

const sessions = ref([])
const msg = ref('')

function flash(t) { msg.value = t; setTimeout(() => (msg.value = ''), 3000) }
function ago(d) {
  const s = Math.floor((Date.now() - new Date(d).getTime()) / 1000)
  if (s < 60) return s + 's'
  if (s < 3600) return Math.floor(s/60) + 'm'
  if (s < 86400) return Math.floor(s/3600) + 'h'
  return Math.floor(s/86400) + 'd'
}
function device(ua) {
  if (/iphone|android|mobile/i.test(ua)) return t('sessions.mobile')
  if (/chrome/i.test(ua)) return 'Chrome'
  if (/firefox/i.test(ua)) return 'Firefox'
  if (/safari/i.test(ua)) return 'Safari'
  return ua ? ua.slice(0, 24) : '—'
}

async function load() {
  try { sessions.value = await api.sessionList() } catch (e) { flash(t('common.errorPrefix') + e.message) }
}
async function revoke(s) {
  try { await api.sessionRevoke(s.token); flash(t('sessions.revoked')); await load() }
  catch (e) { flash(t('common.errorPrefix') + e.message) }
}
const isMine = (s) => s.user_id === auth.user?.id

onMounted(load)
</script>

<template>
  <div class="sessions">
    <div class="top">
      <div>
        <h1>{{ t('sessions.title') }}</h1>
        <p>{{ sessions.length }} {{ t('sessions.devicesLoggedIn') }}</p>
      </div>
      <button class="btn-ghost" @click="load">↻ {{ t('common.refresh') }}</button>
    </div>

    <Transition name="page"><div v-if="msg" class="toast">{{ msg }}</div></Transition>

    <div class="grid">
      <div v-for="s in sessions" :key="s.token" class="sess card" :class="{ mine: isMine(s) }">
        <div class="sess__top">
          <div class="sess__av">{{ (s.user_name || s.user_email).slice(0,2).toUpperCase() }}</div>
          <div class="sess__id">
            <div class="sess__name">{{ s.user_name || '—' }} <span v-if="isMine(s)" class="you">{{ t('sessions.you') }}</span></div>
            <div class="sess__email">{{ s.user_email }}</div>
          </div>
          <span class="role" :class="s.user_role">{{ s.user_role === 'admin' ? t('role.adminShort') : t('role.operator') }}</span>
        </div>
        <div class="sess__meta">
          <span><b>{{ t('sessions.device') }}:</b> {{ device(s.user_agent) }}</span>
          <span><b>IP:</b> <span class="mono">{{ (s.ip || '').split(':')[0] }}</span></span>
          <span><b>{{ t('sessions.activity') }}:</b> {{ ago(s.last_seen) }} {{ t('sessions.ago') }}</span>
        </div>
        <button class="sess__revoke" @click="revoke(s)">{{ t('sessions.revoke') }}</button>
      </div>
      <div v-if="!sessions.length" class="empty card">{{ t('sessions.empty') }}</div>
    </div>
  </div>
</template>

<style scoped>
.sessions { animation: fade-up 0.4s both; }
.top { display: flex; justify-content: space-between; align-items: flex-start; margin: 16px 0 20px; }
.top h1 { font-size: 24px; font-weight: 800; }
.top p { color: var(--text-dim); font-size: 13px; margin-top: 4px; }
.toast { position: fixed; top: 22px; left: 50%; transform: translateX(-50%); z-index: 50;
  background: var(--grad); color: #fff; padding: 11px 22px; border-radius: 12px; font-size: 13.5px; font-weight: 600; box-shadow: var(--glow); }
.grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(330px, 1fr)); gap: 14px; }
.sess { padding: 18px; }
.sess.mine { border-color: var(--accent); }
.sess__top { display: flex; align-items: center; gap: 12px; }
.sess__av { width: 40px; height: 40px; border-radius: 11px; background: var(--grad-soft); color: var(--accent); display: grid; place-items: center; font-weight: 700; }
.sess__id { flex: 1; min-width: 0; }
.sess__name { font-size: 14px; font-weight: 600; }
.you { font-size: 10px; background: var(--accent); color: #fff; padding: 1px 7px; border-radius: 999px; margin-left: 4px; }
.sess__email { font-size: 12px; color: var(--text-dim); }
.role { font-size: 10.5px; font-weight: 600; padding: 3px 9px; border-radius: 999px; }
.role.admin { background: rgba(109,94,252,0.16); color: var(--accent); }
.role.operator { background: var(--surface-2); color: var(--text-dim); }
.sess__meta { display: flex; flex-direction: column; gap: 6px; margin: 14px 0; font-size: 12.5px; color: var(--text-dim); }
.sess__meta b { color: var(--text-faint); font-weight: 600; }
.sess__revoke { width: 100%; justify-content: center; background: rgba(239,68,68,0.14); color: var(--red); }
.sess__revoke:hover { background: rgba(239,68,68,0.22); box-shadow: none; }
.empty { grid-column: 1/-1; padding: 50px; text-align: center; color: var(--text-faint); }
</style>
