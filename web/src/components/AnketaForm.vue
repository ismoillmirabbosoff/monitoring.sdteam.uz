<script setup>
import { computed } from 'vue'

// phone.sdteam uslubidagi anketa: Sabab → Modullar → Status → Izoh
const props = defineProps({
  config: { type: Object, default: () => ({ reasons: [], common_modules: [], payment_topics: [], statuses: [] }) },
  modelValue: { type: Object, default: () => ({}) },
})
const emit = defineEmits(['update:modelValue'])

const val = computed({
  get: () => props.modelValue || {},
  set: (v) => emit('update:modelValue', v),
})

const reasons = computed(() => props.config?.reasons || [])
const statuses = computed(() => props.config?.statuses || [])
const currentReason = computed(() => reasons.value.find((r) => r.key === val.value.reason_key) || null)

// tanlangan sababga qarab modullar ro'yxati
const moduleList = computed(() => {
  const r = currentReason.value
  if (!r) return []
  if (r.module_set === 'payment') return props.config?.payment_topics || []
  if (r.module_set === 'custom') return r.custom_modules || []
  return props.config?.common_modules || []
})
const statusList = computed(() => {
  const r = currentReason.value
  if (r && r.module_set === 'custom' && Array.isArray(r.custom_statuses) && r.custom_statuses.length) return r.custom_statuses
  return statuses.value
})
const moduleTitle = computed(() => currentReason.value?.module_title || 'Модули')
const commentRequired = computed(() => !!currentReason.value?.required)

function patch(p) { emit('update:modelValue', { ...val.value, ...p }) }
function selectReason(r) {
  patch({ reason_key: r.key, reason_label: r.label, modules: [], status: val.value.status || '' })
}
function toggleModule(m) {
  const cur = Array.isArray(val.value.modules) ? [...val.value.modules] : []
  const i = cur.indexOf(m)
  if (i >= 0) cur.splice(i, 1); else cur.push(m)
  patch({ modules: cur })
}
function hasModule(m) { return Array.isArray(val.value.modules) && val.value.modules.includes(m) }
</script>

<template>
  <div class="af">
    <!-- SABAB -->
    <div class="af__q">
      <label class="af__label">Причина обращения <span class="af__req">*</span></label>
      <div class="af__chips">
        <button v-for="r in reasons" :key="r.key" type="button" class="af__chip"
                :class="{ on: val.reason_key === r.key }" @click="selectReason(r)">{{ r.label }}</button>
      </div>
    </div>

    <template v-if="currentReason">
      <!-- MODULLAR -->
      <div class="af__q" v-if="moduleList.length">
        <label class="af__label">{{ moduleTitle }}</label>
        <div class="af__chips">
          <button v-for="m in moduleList" :key="m" type="button" class="af__chip sm"
                  :class="{ on: hasModule(m) }" @click="toggleModule(m)">{{ m }}</button>
        </div>
      </div>

      <!-- STATUS -->
      <div class="af__q">
        <label class="af__label">Статус <span class="af__req">*</span></label>
        <div class="af__chips">
          <button v-for="st in statusList" :key="st" type="button" class="af__chip"
                  :class="{ on: val.status === st }" @click="patch({ status: st })">{{ st }}</button>
        </div>
      </div>

      <!-- IZOH -->
      <div class="af__q">
        <label class="af__label">Комментарий <span v-if="commentRequired" class="af__req">*</span></label>
        <textarea class="af__ta" rows="3" :value="val.comment || ''"
                  @input="patch({ comment: $event.target.value })" placeholder="Комментарий…"></textarea>
      </div>
    </template>
    <div v-else class="af__hint">Сначала выберите причину обращения</div>
  </div>
</template>

<style scoped>
.af { display: flex; flex-direction: column; gap: 18px; }
.af__q { display: flex; flex-direction: column; gap: 9px; }
.af__label { font-size: 13.5px; font-weight: 600; }
.af__req { color: var(--red); }
.af__chips { display: flex; gap: 8px; flex-wrap: wrap; }
.af__chip { background: var(--surface-2); color: var(--text-dim); border: 1px solid var(--border-strong);
  padding: 8px 15px; font-size: 13px; border-radius: 10px; }
.af__chip.sm { padding: 6px 12px; font-size: 12px; }
.af__chip:hover { transform: none; box-shadow: none; color: var(--text); }
.af__chip.on { background: var(--grad); color: #fff; border-color: transparent; }
.af__ta { width: 100%; resize: vertical; font-family: inherit; }
.af__hint { color: var(--text-faint); font-size: 13px; padding: 10px 0; }
</style>
