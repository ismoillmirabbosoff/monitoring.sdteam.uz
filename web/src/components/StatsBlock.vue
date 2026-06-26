<script setup>
import { computed } from 'vue'
import { fmtDuration } from '../api.js'
import { t } from '../i18n.js'

const props = defineProps({
  stats: { type: Object, default: null },
  names: { type: Object, default: () => ({}) },
})

const periods = ['today', 'week', 'month']
const opList = computed(() =>
  [...(props.stats?.operators || [])].sort((a, b) => (b.incoming + b.outgoing) - (a.incoming + a.outgoing))
)
const maxOp = computed(() => Math.max(1, ...opList.value.map((o) => o.incoming + o.outgoing)))
function nm(ext) { return props.names[ext] || `#${ext}` }
function survPct(p) {
  const s = props.stats?.surveys?.[p]
  if (!s || !s[1]) return 0
  return Math.round(s[0] / s[1] * 100)
}
</script>

<template>
  <div v-if="stats" class="sb">
    <div class="sb__tables">
      <!-- KIRUVCHI -->
      <div class="card panel">
        <div class="panel__head">
          <div class="ico in"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M17 7 7 17M7 17h8M7 17V9"/></svg></div>
          <div class="panel__t">{{ t('dash.incoming') }}</div>
          <div class="panel__big"><b>{{ stats.incoming.today.total }}</b><span>{{ t('common.today') }}</span></div>
        </div>
        <div class="rows">
          <div class="rh"><span></span><span>{{ t('common.today') }}</span><span>{{ t('common.week') }}</span><span>{{ t('common.month') }}</span></div>
          <div class="r"><span class="r__l">{{ t('st.total') }}</span><span class="hi">{{ stats.incoming.today.total }}</span><span>{{ stats.incoming.week.total }}</span><span>{{ stats.incoming.month.total }}</span></div>
          <div class="r"><span class="r__l">{{ t('st.answered') }}</span><span class="hi"><b class="pill-ok">{{ stats.incoming.today.answered }}</b></span><span class="ok">{{ stats.incoming.week.answered }}</span><span class="ok">{{ stats.incoming.month.answered }}</span></div>
          <div class="r"><span class="r__l">{{ t('st.missed') }}</span><span class="hi"><b class="pill-bad">{{ stats.incoming.today.missed }}</b></span><span class="bad">{{ stats.incoming.week.missed }}</span><span class="bad">{{ stats.incoming.month.missed }}</span></div>
          <div class="r"><span class="r__l">{{ t('st.surveys') }}</span><span class="hi">{{ survPct('today') }}%</span><span>{{ survPct('week') }}%</span><span>{{ survPct('month') }}%</span></div>
          <div class="r"><span class="r__l">{{ t('st.avgTalk') }}</span><span class="hi mono">{{ fmtDuration(stats.incoming.today.avg) }}</span><span class="mono dim">{{ fmtDuration(stats.incoming.week.avg) }}</span><span class="mono dim">{{ fmtDuration(stats.incoming.month.avg) }}</span></div>
          <div class="r"><span class="r__l">{{ t('st.totalTalk') }}</span><span class="hi mono">{{ fmtDuration(stats.incoming.today.talk) }}</span><span class="mono dim">{{ fmtDuration(stats.incoming.week.talk) }}</span><span class="mono dim">{{ fmtDuration(stats.incoming.month.talk) }}</span></div>
        </div>
      </div>

      <!-- CHIQUVCHI -->
      <div class="card panel">
        <div class="panel__head">
          <div class="ico out"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M7 17 17 7M17 7H9M17 7v8"/></svg></div>
          <div class="panel__t">{{ t('dash.outgoing') }}</div>
          <div class="panel__big"><b>{{ stats.outgoing.today.total }}</b><span>{{ t('common.today') }}</span></div>
        </div>
        <div class="rows">
          <div class="rh"><span></span><span>{{ t('common.today') }}</span><span>{{ t('common.week') }}</span><span>{{ t('common.month') }}</span></div>
          <div class="r"><span class="r__l">{{ t('st.total') }}</span><span class="hi">{{ stats.outgoing.today.total }}</span><span>{{ stats.outgoing.week.total }}</span><span>{{ stats.outgoing.month.total }}</span></div>
          <div class="r"><span class="r__l">{{ t('st.success') }}</span><span class="hi"><b class="pill-ok">{{ stats.outgoing.today.answered }}</b></span><span class="ok">{{ stats.outgoing.week.answered }}</span><span class="ok">{{ stats.outgoing.month.answered }}</span></div>
          <div class="r"><span class="r__l">{{ t('st.failed') }}</span><span class="hi"><b class="pill-bad">{{ stats.outgoing.today.missed }}</b></span><span class="bad">{{ stats.outgoing.week.missed }}</span><span class="bad">{{ stats.outgoing.month.missed }}</span></div>
          <div class="r"><span class="r__l">{{ t('st.avgTalk') }}</span><span class="hi mono">{{ fmtDuration(stats.outgoing.today.avg) }}</span><span class="mono dim">{{ fmtDuration(stats.outgoing.week.avg) }}</span><span class="mono dim">{{ fmtDuration(stats.outgoing.month.avg) }}</span></div>
          <div class="r"><span class="r__l">{{ t('st.totalTalk') }}</span><span class="hi mono">{{ fmtDuration(stats.outgoing.today.talk) }}</span><span class="mono dim">{{ fmtDuration(stats.outgoing.week.talk) }}</span><span class="mono dim">{{ fmtDuration(stats.outgoing.month.talk) }}</span></div>
        </div>
      </div>
    </div>

    <!-- OPERATOR JADVALI -->
    <div class="card ops">
      <div class="ops__head"><h3>{{ t('st.opTable') }}</h3><span class="badge">{{ t('common.today') }}</span></div>
      <div class="ops__scroll">
        <div class="ops__row ops__row--head">
          <span class="c-name">{{ t('st.name') }}</span>
          <span class="c-n">{{ t('st.inCalls') }}</span><span class="c-t">{{ t('st.time') }}</span>
          <span class="c-n">{{ t('st.outCalls') }}</span><span class="c-t">{{ t('st.time') }}</span>
          <span class="c-n">{{ t('st.missed') }}</span><span class="c-t">{{ t('st.totalTalk') }}</span>
          <span class="c-pct">%</span>
        </div>
        <div v-for="o in opList" :key="o.ext" class="ops__row">
          <span class="c-name"><span class="av">{{ nm(o.ext).slice(0,2).toUpperCase() }}</span>
            <span class="who"><b>{{ nm(o.ext) }}</b><i class="mono">#{{ o.ext }}</i></span></span>
          <span class="c-n ok mono">{{ o.incoming }}</span>
          <span class="c-t mono dim">{{ fmtDuration(o.incoming_time) }}</span>
          <span class="c-n mono" style="color:var(--accent-2)">{{ o.outgoing }}</span>
          <span class="c-t mono dim">{{ fmtDuration(o.outgoing_time) }}</span>
          <span class="c-n mono" :class="{ bad: o.missed }">{{ o.missed }}</span>
          <span class="c-t mono">{{ fmtDuration(o.total_time) }}</span>
          <span class="c-pct">
            <span class="bar"><i :style="{ width: ((o.incoming+o.outgoing)/maxOp*100)+'%' }"></i></span>
            <em>{{ o.pct.toFixed(1) }}%</em>
          </span>
        </div>
        <div v-if="!opList.length" class="ops__empty">—</div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.sb { display: flex; flex-direction: column; gap: 22px; margin-bottom: 24px; }
.sb__tables { display: grid; grid-template-columns: 1fr 1fr; gap: 22px; }

.panel { padding: 24px 26px; }
.panel__head { display: flex; align-items: center; gap: 14px; margin-bottom: 20px; }
.ico { width: 44px; height: 44px; border-radius: 13px; display: grid; place-items: center; flex-shrink: 0; }
.ico svg { width: 22px; height: 22px; }
.ico.in { background: var(--green-soft); color: var(--green); }
.ico.out { background: rgba(6,182,212,0.14); color: var(--accent-2); }
.panel__t { font-size: 17px; font-weight: 700; }
.panel__big { margin-left: auto; text-align: right; line-height: 1; }
.panel__big b { font-size: 30px; font-weight: 800; font-family: var(--mono); }
.panel__big span { display: block; font-size: 11px; color: var(--text-faint); margin-top: 4px; text-transform: uppercase; letter-spacing: 0.05em; }

.rows { display: flex; flex-direction: column; }
.rh, .r { display: grid; grid-template-columns: 1.5fr 1fr 1fr 1fr; align-items: center; }
.rh { padding: 0 0 10px; }
.rh span { text-align: right; font-size: 10.5px; font-weight: 600; color: var(--text-faint); text-transform: uppercase; letter-spacing: 0.04em; }
.rh span:first-child { text-align: left; }
.r { padding: 13px 0; border-top: 1px solid var(--border); font-size: 14px; }
.r span { text-align: right; }
.r__l { text-align: left !important; color: var(--text-dim); font-size: 13px; font-weight: 500; }
.r .hi { font-weight: 700; padding-right: 6px; }
.r .ok { color: var(--green); }
.r .bad { color: var(--red); }
.dim { color: var(--text-faint); }

.ops { padding: 22px 24px; }
.ops__head { display: flex; align-items: center; gap: 12px; margin-bottom: 16px; }
.ops__head h3 { font-size: 17px; font-weight: 700; }
.badge { font-size: 11px; font-weight: 600; color: var(--accent); background: var(--accent-soft); padding: 4px 11px; border-radius: 999px; }
.ops__scroll { overflow-x: auto; }
.ops__row {
  display: grid; align-items: center;
  grid-template-columns: minmax(190px,2.4fr) 70px 90px 70px 90px 80px 110px minmax(120px,1.3fr);
  gap: 8px; padding: 12px 10px; border-radius: 12px;
}
.ops__row:not(.ops__row--head):hover { background: var(--surface-2); }
.ops__row--head { padding: 0 10px 12px; }
.ops__row--head span { font-size: 10.5px; font-weight: 600; color: var(--text-faint); text-transform: uppercase; letter-spacing: 0.04em; }
.ops__row:not(.ops__row--head) { border-top: 1px solid var(--border); border-radius: 0; }
.c-name { display: flex; align-items: center; gap: 12px; }
.c-n, .c-t, .ops__row--head .c-n, .ops__row--head .c-t { text-align: right; }
.c-pct { text-align: right; }
.av { width: 34px; height: 34px; border-radius: 10px; background: var(--grad-soft); color: var(--accent); display: grid; place-items: center; font-size: 12px; font-weight: 700; flex-shrink: 0; }
.who { display: flex; flex-direction: column; min-width: 0; }
.who b { font-size: 13.5px; font-weight: 600; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.who i { font-size: 11px; color: var(--text-faint); font-style: normal; }
.c-n { font-size: 14px; font-weight: 600; }
.c-t { font-size: 13px; }
.c-pct { display: flex; align-items: center; justify-content: flex-end; gap: 10px; }
.bar { width: 70px; height: 6px; border-radius: 999px; background: var(--surface-3); overflow: hidden; flex-shrink: 0; }
.bar i { display: block; height: 100%; background: var(--grad); border-radius: 999px; }
.c-pct em { font-style: normal; font-weight: 700; color: var(--accent); font-size: 13px; width: 44px; text-align: right; }
.ops__empty { text-align: center; color: var(--text-faint); padding: 30px; }

@media (max-width: 1080px) { .sb__tables { grid-template-columns: 1fr; } }
</style>
