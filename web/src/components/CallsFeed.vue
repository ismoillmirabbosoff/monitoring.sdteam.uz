<script setup>
import { fmtDuration, fmtTime } from '../api.js'

defineProps({ calls: { type: Array, default: () => [] } })

function answered(c) { return (c.user_talk_time || 0) > 0 }
</script>

<template>
  <div class="feed">
    <TransitionGroup name="list" tag="div">
      <div v-for="c in calls" :key="c.uuid" class="row">
        <div class="row__dir" :class="c.direction === 'outbound' ? 'out' : 'in'">
          <svg v-if="c.direction === 'outbound'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M7 17 17 7M17 7H9M17 7v8"/>
          </svg>
          <svg v-else viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M17 7 7 17M7 17h8M7 17V9"/>
          </svg>
        </div>
        <div class="row__main">
          <div class="row__num mono">{{ c.caller_id_number || '—' }}</div>
          <div class="row__sub">→ {{ c.destination_number || '—' }}</div>
        </div>
        <div class="row__meta">
          <span class="row__badge" :class="answered(c) ? 'ok' : 'miss'">
            {{ answered(c) ? fmtDuration(c.user_talk_time) : 'Javobsiz' }}
          </span>
          <span class="row__time mono">{{ fmtTime(c.start_stamp) }}</span>
        </div>
      </div>
    </TransitionGroup>
    <div v-if="!calls.length" class="empty">Bugun hali qo'ng'iroqlar yo'q</div>
  </div>
</template>

<style scoped>
.feed { display: flex; flex-direction: column; max-height: 520px; overflow-y: auto; }
.row {
  display: flex; align-items: center; gap: 13px;
  padding: 11px 6px;
  border-bottom: 1px solid var(--border);
}
.row:last-child { border-bottom: none; }
.row__dir {
  display: grid; place-items: center;
  width: 34px; height: 34px; border-radius: 10px; flex-shrink: 0;
}
.row__dir svg { width: 17px; height: 17px; }
.row__dir.in { background: rgba(52,211,153,0.14); color: var(--green); }
.row__dir.out { background: rgba(34,211,238,0.14); color: var(--accent-2); }
.row__main { flex: 1; min-width: 0; }
.row__num { font-size: 14px; font-weight: 600; }
.row__sub { font-size: 11.5px; color: var(--text-faint); margin-top: 1px; }
.row__meta { display: flex; flex-direction: column; align-items: flex-end; gap: 4px; }
.row__badge {
  font-size: 11px; font-weight: 600; padding: 2px 8px; border-radius: 7px;
}
.row__badge.ok { background: rgba(52,211,153,0.14); color: var(--green); }
.row__badge.miss { background: rgba(248,113,113,0.14); color: var(--red); }
.row__time { font-size: 11px; color: var(--text-faint); }
.empty { padding: 40px; text-align: center; color: var(--text-faint); font-size: 13px; }
</style>
