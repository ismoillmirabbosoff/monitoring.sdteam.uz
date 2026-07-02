<script setup>
import { ref, nextTick } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { api } from '../api.js'
import { auth } from '../auth.js'

const router = useRouter()
const route = useRoute()

const step = ref('login') // 'login' | 'code'
const email = ref('')
const password = ref('')
const code = ref('')
const devCode = ref('')
const error = ref('')
const loading = ref(false)
const codeInput = ref(null)

async function submitLogin() {
  error.value = ''
  loading.value = true
  try {
    const r = await api.authLogin(email.value.trim(), password.value)
    // Operator: token darhol qaytadi (kod bosqichi yo'q)
    if (r.token) {
      auth.setSession(r.token, r.user)
      const dest = route.query.redirect || (r.user.role === 'admin' ? '/admin' : '/')
      router.replace(dest)
      return
    }
    // Admin: email kod bosqichi
    devCode.value = r.dev_code || ''
    step.value = 'code'
    await nextTick(); codeInput.value?.focus()
  } catch (e) {
    error.value = e.message === '401' ? 'Login yoki parol noto\'g\'ri' : (e.message || 'Xato')
  } finally { loading.value = false }
}

async function submitCode() {
  error.value = ''
  loading.value = true
  try {
    const r = await api.authVerify(email.value.trim(), code.value.trim())
    auth.setSession(r.token, r.user)
    const dest = route.query.redirect || (r.user.role === 'admin' ? '/admin' : '/')
    router.replace(dest)
  } catch (e) {
    error.value = e.message === '401' ? 'Kod noto\'g\'ri yoki muddati o\'tgan' : (e.message || 'Xato')
  } finally { loading.value = false }
}

function back() { step.value = 'login'; code.value = ''; error.value = '' }
</script>

<template>
  <div class="auth">
    <div class="auth__card card">
      <div class="auth__logo">
        <svg viewBox="0 0 32 32"><defs><linearGradient id="alg" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stop-color="#6d5efc"/><stop offset="1" stop-color="#14b8c4"/></linearGradient></defs><rect width="32" height="32" rx="9" fill="url(#alg)"/><path d="M9 20c0-3.9 3.1-7 7-7s7 3.1 7 7" fill="none" stroke="#fff" stroke-width="2.4" stroke-linecap="round"/><circle cx="16" cy="22" r="2.2" fill="#fff"/></svg>
      </div>
      <h1>Monitoring</h1>

      <!-- 1-bosqich: email + parol -->
      <form v-if="step === 'login'" @submit.prevent="submitLogin" class="auth__form">
        <p class="auth__sub">Tizimga kirish</p>
        <label class="fld"><span>Email yoki ext</span>
          <input v-model="email" type="text" placeholder="email@salesdoc.io yoki 201" autofocus autocomplete="username" />
        </label>
        <label class="fld"><span>Parol</span>
          <input v-model="password" type="password" placeholder="••••••••" autocomplete="current-password" />
        </label>
        <div v-if="error" class="auth__err">{{ error }}</div>
        <button type="submit" :disabled="loading">{{ loading ? '...' : 'Davom etish' }}</button>
      </form>

      <!-- 2-bosqich: kod -->
      <form v-else @submit.prevent="submitCode" class="auth__form">
        <p class="auth__sub"><b>{{ email }}</b> ga yuborilgan kodni kiriting</p>
        <div v-if="devCode" class="auth__dev">DEV kod: <b class="mono">{{ devCode }}</b></div>
        <label class="fld"><span>Tasdiqlash kodi</span>
          <input ref="codeInput" v-model="code" inputmode="numeric" maxlength="6" placeholder="000000" class="auth__code mono" />
        </label>
        <div v-if="error" class="auth__err">{{ error }}</div>
        <button type="submit" :disabled="loading">{{ loading ? '...' : 'Kirish' }}</button>
        <button type="button" class="btn-ghost auth__back" @click="back">← Orqaga</button>
      </form>
    </div>
  </div>
</template>

<style scoped>
.auth { min-height: 100vh; display: grid; place-items: center; padding: 20px; }
.auth__card { width: 380px; padding: 38px 34px; text-align: center; animation: fade-up 0.5s both; }
.auth__logo svg { width: 56px; height: 56px; margin-bottom: 14px; }
.auth h1 { font-size: 24px; font-weight: 800; letter-spacing: -0.02em; margin-bottom: 4px; }
.auth__sub { font-size: 13.5px; color: var(--text-dim); margin: 6px 0 22px; }
.auth__form { display: flex; flex-direction: column; gap: 14px; text-align: left; }
.fld { display: flex; flex-direction: column; gap: 7px; }
.fld span { font-size: 12px; font-weight: 600; color: var(--text-dim); }
.fld input { width: 100%; }
.auth__code { font-size: 24px; letter-spacing: 0.3em; text-align: center; padding: 12px; }
.auth__err { color: var(--red); font-size: 12.5px; text-align: center; }
.auth__dev { font-size: 12px; color: var(--amber); background: rgba(245,158,11,0.12);
  padding: 8px; border-radius: 8px; text-align: center; }
.auth__form button[type=submit] { width: 100%; justify-content: center; padding: 12px; margin-top: 4px; }
.auth__back { width: 100%; justify-content: center; }
</style>
