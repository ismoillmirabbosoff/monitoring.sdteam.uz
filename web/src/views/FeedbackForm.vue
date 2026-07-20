<script setup>
import { ref } from 'vue'
import { useRoute } from 'vue-router'
import { api } from '../api.js'
import { t } from '../i18n.js'

const route = useRoute()
const score = ref(0)
const comment = ref('')
const sending = ref(false)
const done = ref(false)
const error = ref('')

async function submit() {
  if (!score.value) { error.value = t('feedback.pickScore'); return }
  sending.value = true; error.value = ''
  try {
    await api.feedbackSubmit({ call_uuid: route.params.uuid || '', score: score.value, comment: comment.value.trim() })
    done.value = true
  } catch (e) { error.value = t('common.errorPrefix') + (e.message || '') }
  finally { sending.value = false }
}
</script>

<template>
  <div class="fb">
    <div class="fb__card card">
      <div class="fb__logo">
        <svg viewBox="0 0 32 32"><defs><linearGradient id="flg" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stop-color="#6d5efc"/><stop offset="1" stop-color="#14b8c4"/></linearGradient></defs><rect width="32" height="32" rx="9" fill="url(#flg)"/><path d="M9 20c0-3.9 3.1-7 7-7s7 3.1 7 7" fill="none" stroke="#fff" stroke-width="2.4" stroke-linecap="round"/><circle cx="16" cy="22" r="2.2" fill="#fff"/></svg>
      </div>

      <template v-if="!done">
        <h1>{{ t('feedback.rateTitle') }}</h1>
        <p class="fb__sub">{{ t('feedback.rateSub') }}</p>
        <div class="fb__stars">
          <button v-for="n in 5" :key="n" type="button" class="fb__star" :class="{ on: score >= n }" @click="score = n">★</button>
        </div>
        <textarea v-model="comment" class="fb__ta" rows="3" :placeholder="t('feedback.commentOptional')"></textarea>
        <div v-if="error" class="fb__err">{{ error }}</div>
        <button class="fb__btn" @click="submit" :disabled="sending">{{ sending ? '...' : t('common.send') }}</button>
      </template>

      <template v-else>
        <div class="fb__ok">✓</div>
        <h1>{{ t('feedback.thanks') }}</h1>
        <p class="fb__sub">{{ t('feedback.received') }}</p>
      </template>
    </div>
  </div>
</template>

<style scoped>
.fb { min-height: 100vh; display: grid; place-items: center; padding: 20px; background: var(--bg); }
.fb__card { width: 400px; max-width: 100%; padding: 38px 34px; text-align: center; animation: fade-up 0.5s both; }
.fb__logo svg { width: 56px; height: 56px; margin-bottom: 16px; }
.fb h1 { font-size: 22px; font-weight: 800; }
.fb__sub { font-size: 13.5px; color: var(--text-dim); margin: 6px 0 22px; }
.fb__stars { display: flex; justify-content: center; gap: 8px; margin-bottom: 20px; }
.fb__star { background: none; border: none; color: var(--border-strong); font-size: 46px; padding: 0; line-height: 1; cursor: pointer; transition: transform 0.15s, color 0.15s; }
.fb__star:hover { transform: scale(1.15); }
.fb__star.on { color: var(--amber); }
.fb__ta { width: 100%; resize: vertical; font-family: inherit; margin-bottom: 16px; }
.fb__err { color: var(--red); font-size: 13px; margin-bottom: 12px; }
.fb__btn { width: 100%; justify-content: center; padding: 13px; }
.fb__ok { width: 64px; height: 64px; margin: 0 auto 16px; border-radius: 50%; background: var(--green-soft, rgba(16,185,129,0.15)); color: var(--green); display: grid; place-items: center; font-size: 32px; font-weight: 800; }
</style>
