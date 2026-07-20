// Backend bilan ishlash qatlami. Same-origin (Go server) yoki dev proxy orqali.
import { auth } from './auth.js'

const BASE = import.meta.env.VITE_API_BASE || ''

function authHeaders(extra) {
  const h = { ...(extra || {}) }
  if (auth.token) h['Authorization'] = 'Bearer ' + auth.token
  return h
}

async function get(path, params) {
  const url = new URL(BASE + path, window.location.origin)
  if (params) Object.entries(params).forEach(([k, v]) => url.searchParams.set(k, v))
  const res = await fetch(url, { headers: authHeaders({ Accept: 'application/json' }) })
  if (!res.ok) throw new Error(`${path} → ${res.status}`)
  return res.json()
}

// Sessiya token bilan so'rov (admin/auth endpointlari).
async function req(method, path, body) {
  const res = await fetch(BASE + path, {
    method,
    headers: authHeaders({ 'Content-Type': 'application/json' }),
    body: body ? JSON.stringify(body) : undefined,
  })
  if (res.status === 401) { throw new Error('401') }
  if (!res.ok) {
    let msg = `${path} → ${res.status}`
    try { msg = (await res.json()).error || msg } catch {}
    throw new Error(msg)
  }
  return res.status === 204 ? null : res.json()
}
const adminReq = req

export const api = {
  config: () => get('/api/config'),
  keys: () => get('/api/monitoring/keys'),
  fifo: () => get('/api/monitoring/fifo'),
  users: () => get('/api/monitoring/users'),
  hidden: () => get('/api/monitoring/hidden'),
  stats: (company) => get('/api/monitoring/stats', company ? { company } : undefined),
  bigData: (date) => get('/api/monitoring/bigData', { date }),
  operatorTime: (date) => get('/api/monitoring/operatorTime', { date }),
  data: (gateway, from, to) => get('/api/monitoring/data', { gateway, from, to }),
  // qo'ng'iroq yozuvi (mp3) — <audio src> uchun bevosita URL
  recordingUrl: (uuid) => `${BASE}/api/monitoring/recording?uuid=${encodeURIComponent(uuid)}`,

  // auth
  authLogin: (email, password) => req('POST', '/api/auth/login', { email, password }),
  authVerify: (email, code) => req('POST', '/api/auth/verify', { email, code }),
  me: () => req('GET', '/api/auth/me'),
  logout: () => req('POST', '/api/auth/logout'),

  // admin: foydalanuvchilar va sessiyalar
  userList: () => req('GET', '/api/admin/users'),
  userCreate: (payload) => req('POST', '/api/admin/users', payload),
  userUpdate: (id, payload) => req('PATCH', `/api/admin/users/${id}`, payload),
  userDelete: (id) => req('DELETE', `/api/admin/users/${id}`),
  sessionList: () => req('GET', '/api/admin/sessions'),
  sessionRevoke: (token) => req('DELETE', `/api/admin/sessions/${token}`),

  // anketa
  surveyQuestions: () => req('GET', '/api/survey/questions'),
  surveySubmit: (payload) => req('POST', '/api/survey/responses', payload),
  surveyResponses: (from, to) => get('/api/survey/responses', { from, to }),
  // anketa konfiguratsiyasi (reason/modules/status/comment)
  surveyConfig: () => req('GET', '/api/survey/config'),
  qConfig: () => req('GET', '/api/admin/survey/config'),
  qConfigSave: (cfg) => req('PUT', '/api/admin/survey/config', cfg),
  qList: () => req('GET', '/api/admin/survey/questions'),
  qCreate: (payload) => req('POST', '/api/admin/survey/questions', payload),
  qUpdate: (id, payload) => req('PATCH', `/api/admin/survey/questions/${id}`, payload),
  qDelete: (id) => req('DELETE', `/api/admin/survey/questions/${id}`),

  employees: () => adminReq('GET', '/api/admin/employees'),
  addEmployee: (name, company) => adminReq('POST', '/api/admin/employees', { name, company }),
  employee: (id) => adminReq('GET', `/api/admin/employees/${id}`),
  updateEmployee: (id, payload) => adminReq('PATCH', `/api/admin/employees/${id}`, payload),
  setHiddenByExt: (ext, hidden) => adminReq('PATCH', `/api/admin/employees/by-ext/${ext}`, { hidden }),
  // mijoz baholari (otzyvlar)
  feedbackSubmit: (payload) => req('POST', '/api/feedback', payload),
  feedbackList: (params) => {
    const qs = new URLSearchParams(Object.entries(params || {}).filter(([, v]) => v)).toString()
    return adminReq('GET', '/api/admin/feedback' + (qs ? '?' + qs : ''))
  },

  // operator ballari (avtomatizatsiya)
  scores: () => adminReq('GET', '/api/admin/scores'),
  scoreAdd: (payload) => adminReq('POST', '/api/admin/scores', payload),
  scoreDelete: (id) => adminReq('DELETE', `/api/admin/scores/${id}`),

  // ish vaqti + audit log
  workHours: () => adminReq('GET', '/api/admin/work-hours'),
  workHoursSave: (rows) => adminReq('POST', '/api/admin/work-hours', rows),
  auditLog: (params) => {
    const qs = new URLSearchParams(Object.entries(params || {}).filter(([, v]) => v)).toString()
    return adminReq('GET', '/api/admin/audit-log' + (qs ? '?' + qs : ''))
  },

  servers: () => adminReq('GET', '/api/admin/servers'),
  addServer: (payload) => adminReq('POST', '/api/admin/servers', payload),
  updateServer: (id, payload) => adminReq('PATCH', `/api/admin/servers/${id}`, payload),
  deleteServer: (id) => adminReq('DELETE', `/api/admin/servers/${id}`),
}

// OnlinePBX WebSocket manzili.
//   - production (https, nginx orqasida): same-origin /onpbx-ws/ proxy orqali (443 port)
//   - dev (http localhost): to'g'ridan-to'g'ri OnlinePBX :3342 ga
export function wsUrl(cfg, key) {
  if (typeof window !== 'undefined' && window.location.protocol === 'https:') {
    return `wss://${window.location.host}/onpbx-ws/?key=${key}`
  }
  return `wss://${cfg.domain}:${cfg.wsPort || 3342}/?key=${key}`
}

// --- Kompaniya (Salesdoc / Ibox) ---
export const COMPANIES = [
  { id: '', name: 'Hammasi', color: 'var(--accent)' },
  { id: 'salesdoc', name: 'Salesdoc', color: 'var(--green)' },
  { id: 'ibox', name: 'Ibox', color: 'var(--accent-2)' },
]

export function companyForGateway(g) {
  g = String(g || '')
  if (g.startsWith('712')) return 'salesdoc'
  if (g.startsWith('781')) return 'ibox'
  return ''
}
export function companyForQueue(q) {
  q = String(q || '').trim()
  if (q === '5201') return 'salesdoc'
  if (q === '5202') return 'ibox'
  return ''
}
export function companyName(id) {
  return (COMPANIES.find((c) => c.id === id) || {}).name || id
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
