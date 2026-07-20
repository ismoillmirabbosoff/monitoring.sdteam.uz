<script setup>
import { computed } from 'vue'
import { companyName } from '../api.js'

const props = defineProps({ server: Object })
const emit = defineEmits(['open', 'remove', 'edit'])

const age = computed(() => {
  const d = props.server.days
  if (d < 1) return 'Bugun'
  if (d < 30) return `${d} kun`
  const m = Math.floor(d / 30)
  const rem = d % 30
  return rem > 0 ? `${m} oy ${rem} kun` : `${m} oy`
})
</script>

<template>
  <div class="srv" :class="server.company">
    <div class="srv__bar"></div>
    <div class="srv__body">
      <div class="srv__top">
        <div class="srv__name" :title="server.name">{{ server.name }}</div>
        <button class="srv__edit" @click.stop="emit('edit', server)" title="Tahrirlash">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 20h9"/><path d="M16.5 3.5a2.12 2.12 0 0 1 3 3L7 19l-4 1 1-4z"/></svg>
        </button>
        <button class="srv__x" @click.stop="emit('remove', server.id)" title="O'chirish">×</button>
      </div>
      <div class="srv__emp" v-if="server.employee_name" @click="emit('open', server.employee_id)">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>
        {{ server.employee_name }}
      </div>
      <div class="srv__emp srv__emp--none" v-else>Biriktirilmagan</div>
      <div class="srv__foot">
        <span v-if="server.company" class="srv__tag">{{ companyName(server.company) }}</span>
        <span class="srv__age mono">{{ age }}</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.srv {
  position: relative; display: flex; overflow: hidden;
  background: var(--surface); border: 1px solid var(--border);
  border-radius: 12px;
  animation: fade-up 0.4s both;
  transition: transform 0.2s, border-color 0.2s;
}
.srv:hover { transform: translateX(2px); border-color: var(--border-strong); }
.srv__bar { width: 4px; flex-shrink: 0; background: var(--accent); }
.srv.salesdoc .srv__bar { background: var(--green); }
.srv.ibox .srv__bar { background: var(--accent-2); }
.srv__body { flex: 1; min-width: 0; padding: 12px 13px; }
.srv__top { display: flex; align-items: center; gap: 8px; }
.srv__name { flex: 1; min-width: 0; font-size: 13.5px; font-weight: 600;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.srv__x { background: none; border: none; color: var(--text-faint); font-size: 18px; cursor: pointer;
  line-height: 1; padding: 0 2px; transition: color 0.2s; }
.srv__x:hover { color: var(--red); }
.srv__edit { background: none; border: none; color: var(--text-faint); cursor: pointer; padding: 0 2px; transition: color 0.2s; }
.srv__edit svg { width: 14px; height: 14px; }
.srv__edit:hover { color: var(--accent); }
.srv__emp { display: flex; align-items: center; gap: 6px; margin-top: 7px;
  font-size: 12px; color: var(--text-dim); cursor: pointer; }
.srv__emp svg { width: 13px; height: 13px; }
.srv__emp:hover { color: var(--accent-2); }
.srv__emp--none { color: var(--text-faint); cursor: default; font-style: italic; }
.srv__foot { display: flex; align-items: center; justify-content: space-between; margin-top: 10px; }
.srv__tag { font-size: 10.5px; font-weight: 600; color: var(--text-dim);
  background: var(--surface-2); padding: 2px 8px; border-radius: 999px; }
.srv__age { font-size: 11.5px; font-weight: 600; color: var(--text); }
</style>
