// Backend bilan ishlash qatlami. Same-origin (Go server) yoki dev proxy orqali.

const BASE = import.meta.env.VITE_API_BASE || ''

async function get(path, params) {
  const url = new URL(BASE + path, window.location.origin)
  if (params) Object.entries(params).forEach(([k, v]) => url.searchParams.set(k, v))
  const res = await fetch(url, { headers: { Accept: 'application/json' } })
  if (!res.ok) throw new Error(`${path} → ${res.status}`)
  return res.json()
}

export const api = {
  config: () => get('/api/config'),
  keys: () => get('/api/monitoring/keys'),
  fifo: () => get('/api/monitoring/fifo'),
  users: () => get('/api/monitoring/users'),
  bigData: (date) => get('/api/monitoring/bigData', { date }),
  operatorTime: (date) => get('/api/monitoring/operatorTime', { date }),
  data: (gateway, from, to) => get('/api/monitoring/data', { gateway, from, to }),
}

// YYYY-MM-DD (mahalliy vaqt)
export function todayStr(d = new Date()) {
  const p = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`
}

// Ichki extension (operator) raqamimi? (3-4 xonali, 5xxx navbatlar emas)
export function isExtension(num) {
  if (!num) return false
  const s = String(num).trim()
  return /^[1-4]\d{2,3}$/.test(s) && !s.startsWith('5')
}

// fifo "users" satrini parse qiladi: "201:1;202:0;..." → [{ext, online}]
export function parseFifoUsers(users) {
  if (!users) return []
  return users
    .split(';')
    .map((s) => s.trim())
    .filter(Boolean)
    .map((pair) => {
      const [ext, on] = pair.split(':')
      return { ext: String(ext).trim(), online: Number(on) === 1 }
    })
}

export function fmtDuration(sec) {
  sec = Math.max(0, Math.floor(sec || 0))
  const h = Math.floor(sec / 3600)
  const m = Math.floor((sec % 3600) / 60)
  const s = sec % 60
  if (h > 0) return `${h}:${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`
  return `${m}:${String(s).padStart(2, '0')}`
}

export function timeAgo(stamp) {
  const diff = Math.floor(Date.now() / 1000 - stamp)
  if (diff < 60) return `${diff}s`
  if (diff < 3600) return `${Math.floor(diff / 60)}m`
  if (diff < 86400) return `${Math.floor(diff / 3600)}h`
  return `${Math.floor(diff / 86400)}d`
}

export function fmtTime(stamp) {
  const d = new Date(stamp * 1000)
  const p = (n) => String(n).padStart(2, '0')
  return `${p(d.getHours())}:${p(d.getMinutes())}`
}
