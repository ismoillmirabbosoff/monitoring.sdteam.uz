<script setup>
import { ref, watch } from 'vue'
import { t } from '../i18n.js'

const props = defineProps({
  questions: { type: Array, default: () => [] },
  modelValue: { type: Object, default: () => ({}) },
})
const emit = defineEmits(['update:modelValue'])

const answers = ref({ ...props.modelValue })
watch(answers, (v) => emit('update:modelValue', v), { deep: true })

function opts(q) {
  try { return Array.isArray(q.options) ? q.options : JSON.parse(q.options || '[]') }
  catch { return [] }
}
</script>

<template>
  <div class="sf">
    <div v-for="q in questions" :key="q.id" class="sf__q">
      <label class="sf__label">{{ q.label }} <span v-if="q.required" class="sf__req">*</span></label>

      <input v-if="q.type === 'text'" v-model="answers[q.id]" :placeholder="t('surveyForm.answer')" class="sf__input" />

      <div v-else-if="q.type === 'yesno'" class="sf__chips">
        <button type="button" class="sf__chip" :class="{ on: answers[q.id] === 'Ha' }" @click="answers[q.id] = 'Ha'">{{ t('surveyForm.yes') }}</button>
        <button type="button" class="sf__chip no" :class="{ on: answers[q.id] === 'Yo\'q' }" @click="answers[q.id] = 'Yo\'q'">{{ t('surveyForm.no') }}</button>
      </div>

      <div v-else-if="q.type === 'rating'" class="sf__stars">
        <button v-for="n in 5" :key="n" type="button" class="sf__star" :class="{ on: answers[q.id] >= n }"
                @click="answers[q.id] = n">★</button>
      </div>

      <div v-else-if="q.type === 'choice'" class="sf__chips">
        <button v-for="o in opts(q)" :key="o" type="button" class="sf__chip" :class="{ on: answers[q.id] === o }"
                @click="answers[q.id] = o">{{ o }}</button>
      </div>
    </div>
    <div v-if="!questions.length" class="sf__empty">{{ t('surveyForm.noQuestions') }}</div>
  </div>
</template>

<style scoped>
.sf { display: flex; flex-direction: column; gap: 18px; }
.sf__q { display: flex; flex-direction: column; gap: 9px; }
.sf__label { font-size: 13.5px; font-weight: 600; }
.sf__req { color: var(--red); }
.sf__input { width: 100%; }
.sf__chips { display: flex; gap: 8px; flex-wrap: wrap; }
.sf__chip { background: var(--surface-2); color: var(--text-dim); border: 1px solid var(--border-strong);
  padding: 8px 16px; font-size: 13px; }
.sf__chip:hover { transform: none; box-shadow: none; color: var(--text); }
.sf__chip.on { background: var(--grad); color: #fff; border-color: transparent; }
.sf__chip.no.on { background: var(--red); }
.sf__stars { display: flex; gap: 5px; }
.sf__star { background: none; border: none; color: var(--border-strong); font-size: 30px; padding: 0; line-height: 1; }
.sf__star:hover { transform: none; box-shadow: none; }
.sf__star.on { color: var(--amber); }
.sf__empty { color: var(--text-faint); font-size: 13px; text-align: center; padding: 20px; }
</style>
