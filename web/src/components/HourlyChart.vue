<script setup>
import { computed } from 'vue'

// props.calls: [{start_stamp, direction}]
const props = defineProps({ calls: { type: Array, default: () => [] } })

const W = 760, H = 220, PAD = 28
const hours = computed(() => {
  const inb = Array(24).fill(0)
  const out = Array(24).fill(0)
  for (const c of props.calls) {
    const h = new Date(c.start_stamp * 1000).getHours()
    if (c.direction === 'outbound') out[h]++
    else inb[h]++
  }
  return inb.map((v, i) => ({ h: i, in: v, out: out[i], total: v + out[i] }))
})
const max = computed(() => Math.max(4, ...hours.value.map((d) => d.total)))
const bw = (W - PAD * 2) / 24

function y(v) { return H - PAD - (v / max.value) * (H - PAD * 2) }
const nowH = new Date().getHours()
</script>

<template>
  <svg :viewBox="`0 0 ${W} ${H}`" class="chart" preserveAspectRatio="none">
    <defs>
      <linearGradient id="barIn" x1="0" y1="0" x2="0" y2="1">
        <stop offset="0" stop-color="#6d5efc" stop-opacity="0.95" />
        <stop offset="1" stop-color="#6d5efc" stop-opacity="0.35" />
      </linearGradient>
      <linearGradient id="barOut" x1="0" y1="0" x2="0" y2="1">
        <stop offset="0" stop-color="#14b8c4" stop-opacity="0.9" />
        <stop offset="1" stop-color="#14b8c4" stop-opacity="0.3" />
      </linearGradient>
    </defs>

    <!-- grid -->
    <g class="grid" stroke-width="1">
      <line v-for="g in 4" :key="g" :x1="PAD" :x2="W - PAD"
            :y1="PAD + ((H - PAD * 2) / 4) * g" :y2="PAD + ((H - PAD * 2) / 4) * g" />
    </g>

    <g v-for="d in hours" :key="d.h">
      <rect v-if="d.h === nowH" class="now-col" :x="PAD + d.h * bw" :y="PAD" :width="bw" :height="H - PAD * 2"
            rx="3" />
      <rect :x="PAD + d.h * bw + 3" :width="bw - 6"
            :y="y(d.in)" :height="Math.max(0, H - PAD - y(d.in))"
            fill="url(#barIn)" rx="3" />
      <rect :x="PAD + d.h * bw + 3" :width="bw - 6"
            :y="y(d.total)" :height="Math.max(0, y(d.in) - y(d.total))"
            fill="url(#barOut)" rx="3" />
      <text v-if="d.h % 3 === 0" :x="PAD + d.h * bw + bw / 2" :y="H - 8"
            fill="var(--text-faint)" font-size="10" text-anchor="middle"
            font-family="var(--mono)">{{ String(d.h).padStart(2,'0') }}</text>
    </g>
  </svg>
</template>

<style scoped>
.chart { width: 100%; height: 220px; display: block; }
.grid line { stroke: var(--border); }
.now-col { fill: var(--surface-2); }
.chart text { fill: var(--text-faint); }
.chart rect[fill^="url"] {
  transform-box: fill-box;
  transform-origin: bottom;
  animation: grow-bar 0.7s cubic-bezier(.2,.8,.2,1) both;
}
@keyframes grow-bar { from { transform: scaleY(0); } to { transform: scaleY(1); } }
</style>
