<script setup>
import { computed } from 'vue'
import { fmtDuration } from '../api.js'
import { t } from '../i18n.js'

const props = defineProps({ op: Object })

const STATUS = {
  talking:  { label: 'tv.talking', color: 'var(--blue)' },
  ringing:  { label: 'tv.ringing', color: 'var(--amber)' },
  online:   { label: 'st.online',  color: 'var(--green)' },
  offline:  { label: 'st.offline', color: 'var(--gray)' },
}

const st = computed(() => STATUS[props.op.status] || STATUS.offline)
const initials = computed(() => {
  const n = (props.op.name || '').trim()
  const parts = n.split(/\s+/).filter(Boolean)
  if (parts.length >= 2) return (parts[0][0] + parts[1][0]).toUpperCase()
  if (parts.length === 1 && !/^operator$/i.test(parts[0])) return parts[0].slice(0, 2).toUpperCase()
  return String(props.op.ext).slice(-2)
})
const total = computed(() => props.op.incoming + props.op.outgoing)
</script>

<template>
  <div class="op" :class="`op--${op.status}`">
    <div class="op__sheen"></div>
    <div class="op__top">
      <div class="op__avatar" :style="{ '--c': st.color }">
        <span>{{ initials }}</span>
        <i class="op__dot" :style="{ background: st.color }"></i>
      </div>
      <div class="op__id">
        <div class="op__name" :title="op.name">{{ op.name }}</div>
        <div class="op__meta">
          <span class="op__ext mono">#{{ op.ext }}</span>
          <span class="op__status" :style="{ color: st.color }">
            <i :style="{ background: st.color }"></i>{{ t(st.label) }}
          </span>
        </div>
      </div>
    </div>

    <div class="op__stats">
      <div class="op__stat op__stat--in">
        <span class="op__stat-v mono">{{ op.incoming }}</span>
        <span class="op__stat-l">{{ t('dash.incoming') }}</span>
      </div>
      <div class="op__stat op__stat--out">
        <span class="op__stat-v mono">{{ op.outgoing }}</span>
        <span class="op__stat-l">{{ t('dash.outgoing') }}</span>
      </div>
      <div class="op__stat op__stat--avg">
        <span class="op__stat-v mono">{{ fmtDuration(op.avgTalk) }}</span>
        <span class="op__stat-l">{{ t('common.average') }}</span>
      </div>
    </div>

    <div class="op__foot">
      <span class="op__foot-l">{{ t('st.totalTalk') }}</span>
      <span class="op__foot-v mono">{{ fmtDuration(op.talk) }}</span>
    </div>
  </div>
</template>

<style scoped>
.op {
  position: relative;
  background: linear-gradient(160deg, var(--surface-2), var(--surface));
  border: 1px solid var(--border);
  border-radius: 16px;
  padding: 16px;
  overflow: hidden;
  transition: transform 0.3s cubic-bezier(.2,.8,.2,1), border-color 0.3s, box-shadow 0.3s;
}
.op:hover {
  transform: translateY(-4px);
  border-color: var(--border-strong);
  box-shadow: 0 18px 40px -20px rgba(0,0,0,0.7);
}
.op__sheen {
  position: absolute; inset: 0;
  background: linear-gradient(115deg, transparent 30%, rgba(255,255,255,0.06) 50%, transparent 70%);
  transform: translateX(-120%);
  pointer-events: none;
}
.op:hover .op__sheen { animation: sheen 0.9s ease; }
@keyframes sheen { to { transform: translateX(120%); } }

.op--ringing { border-color: rgba(251,191,36,0.5); animation: pulse-ring 1.5s infinite; }
.op--talking { border-color: rgba(96,165,250,0.45); }
.op--online  { border-color: rgba(52,211,153,0.30); }
.op--offline { opacity: 0.5; }

.op__top { display: flex; align-items: center; gap: 12px; }
.op__avatar {
  position: relative;
  display: grid; place-items: center;
  width: 44px; height: 44px; border-radius: 13px;
  font-weight: 700; font-size: 15px;
  color: var(--c);
  background: color-mix(in srgb, var(--c) 18%, transparent);
  border: 1px solid color-mix(in srgb, var(--c) 38%, transparent);
  flex-shrink: 0;
}
.op__dot {
  position: absolute; right: -3px; bottom: -3px;
  width: 13px; height: 13px; border-radius: 50%;
  border: 2.5px solid var(--bg-2);
}
.op--online .op__dot, .op--talking .op__dot, .op--ringing .op__dot { animation: pulse-dot 1.8s infinite; }

.op__id { min-width: 0; flex: 1; }
.op__name {
  font-size: 14px; font-weight: 600; color: var(--text);
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}
.op__meta { display: flex; align-items: center; gap: 8px; margin-top: 3px; }
.op__ext { font-size: 11.5px; color: var(--text-faint); }
.op__status { display: flex; align-items: center; gap: 4px; font-size: 11px; font-weight: 600; }
.op__status i { width: 6px; height: 6px; border-radius: 50%; }

.op__stats {
  display: grid; grid-template-columns: repeat(3, 1fr);
  gap: 6px; margin-top: 15px;
}
.op__stat {
  display: flex; flex-direction: column; gap: 3px;
  padding: 9px 8px; border-radius: 10px;
  background: rgba(255,255,255,0.03);
  border: 1px solid var(--border);
}
.op__stat--in .op__stat-v { color: var(--green); }
.op__stat--out .op__stat-v { color: var(--accent-2); }
.op__stat--avg .op__stat-v { color: var(--amber); }
.op__stat-v { font-size: 15px; font-weight: 700; }
.op__stat-l { font-size: 10px; color: var(--text-faint); font-weight: 500; }

.op__foot {
  display: flex; justify-content: space-between; align-items: center;
  margin-top: 11px; padding-top: 11px;
  border-top: 1px solid var(--border);
}
.op__foot-l { font-size: 11px; color: var(--text-dim); }
.op__foot-v { font-size: 13px; font-weight: 600; color: var(--text); }
</style>
