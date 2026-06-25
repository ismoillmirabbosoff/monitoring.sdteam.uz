<script setup>
import { ref, watch, onMounted } from 'vue'

const props = defineProps({
  label: String,
  value: { type: Number, default: 0 },
  suffix: { type: String, default: '' },
  format: { type: Function, default: null },
  icon: String,
  accent: { type: String, default: 'var(--accent)' },
})

const display = ref(0)
let raf = null

function animate(to) {
  const from = display.value
  const start = performance.now()
  const dur = 700
  cancelAnimationFrame(raf)
  const step = (now) => {
    const t = Math.min(1, (now - start) / dur)
    const eased = 1 - Math.pow(1 - t, 3)
    display.value = from + (to - from) * eased
    if (t < 1) raf = requestAnimationFrame(step)
    else display.value = to
  }
  raf = requestAnimationFrame(step)
}

watch(() => props.value, (v) => animate(v))
onMounted(() => animate(props.value))

function shown() {
  if (props.format) return props.format(display.value)
  return Math.round(display.value).toLocaleString('ru-RU')
}
</script>

<template>
  <div class="stat card">
    <div class="stat__icon" :style="{ background: accent + '22', color: accent }">
      <span v-html="icon"></span>
    </div>
    <div class="stat__body">
      <div class="stat__value mono">{{ shown() }}<span class="stat__suffix">{{ suffix }}</span></div>
      <div class="stat__label">{{ label }}</div>
    </div>
    <div class="stat__glow" :style="{ background: accent }"></div>
  </div>
</template>

<style scoped>
.stat {
  position: relative;
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 20px 22px;
  overflow: hidden;
  transition: transform 0.25s, border-color 0.25s;
}
.stat:hover { transform: translateY(-3px); border-color: var(--border-strong); }
.stat__icon {
  display: grid;
  place-items: center;
  width: 48px; height: 48px;
  border-radius: 14px;
  flex-shrink: 0;
}
.stat__icon :deep(svg) { width: 24px; height: 24px; }
.stat__value { font-size: 30px; font-weight: 700; line-height: 1; }
.stat__suffix { font-size: 15px; font-weight: 500; color: var(--text-dim); margin-left: 3px; }
.stat__label { margin-top: 6px; font-size: 12.5px; color: var(--text-dim); font-weight: 500; }
.stat__glow {
  position: absolute; right: -40px; top: -40px;
  width: 120px; height: 120px; border-radius: 50%;
  opacity: 0.10; filter: blur(30px); pointer-events: none;
}
</style>
