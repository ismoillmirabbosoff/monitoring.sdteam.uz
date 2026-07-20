<script setup>
import { RouterView, RouterLink, useRoute, useRouter } from 'vue-router'
import { ref, computed } from 'vue'
import { auth } from './auth.js'
import { api } from './api.js'
import { t, locale, setLocale, LOCALES } from './i18n.js'

const route = useRoute()
const router = useRouter()
const theme = ref(localStorage.getItem('theme') || 'light')
const collapsed = ref(localStorage.getItem('sidebar_collapsed') === '1')

function applyTheme() { document.documentElement.setAttribute('data-theme', theme.value) }
function toggleTheme() {
  theme.value = theme.value === 'dark' ? 'light' : 'dark'
  localStorage.setItem('theme', theme.value); applyTheme()
}
applyTheme()

function toggleSidebar() {
  collapsed.value = !collapsed.value
  localStorage.setItem('sidebar_collapsed', collapsed.value ? '1' : '0')
}

async function logout() {
  try { await api.logout() } catch {}
  auth.clear()
  router.replace('/login')
}

const fullscreen = computed(() => route.meta.fullscreen)
const showSidebar = computed(() => !fullscreen.value && route.name !== 'login')

const allLinks = [
  { to: '/', key: 'nav.dashboard', admin: false, match: (p) => p === '/',
    icon: '<rect x="3" y="3" width="7" height="9"/><rect x="14" y="3" width="7" height="5"/><rect x="14" y="12" width="7" height="9"/><rect x="3" y="16" width="7" height="5"/>' },
  { to: '/survey', key: 'nav.survey', admin: false, match: (p) => p === '/survey',
    icon: '<path d="M9 11l3 3L22 4"/><path d="M21 12v7a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11"/>' },
  { to: '/calls', key: 'nav.calls', admin: false, match: (p) => p === '/calls',
    icon: '<path d="M22 16.92v3a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07 19.5 19.5 0 0 1-6-6 19.79 19.79 0 0 1-3.07-8.67A2 2 0 0 1 4.11 2h3a2 2 0 0 1 2 1.72c.13.96.36 1.9.7 2.81a2 2 0 0 1-.45 2.11L8.09 9.91a16 16 0 0 0 6 6l1.27-1.27a2 2 0 0 1 2.11-.45c.91.34 1.85.57 2.81.7A2 2 0 0 1 22 16.92z"/>' },
  { to: '/admin/survey', key: 'nav.surveyReport', admin: true, match: (p) => p === '/admin/survey',
    icon: '<path d="M3 3v18h18"/><rect x="7" y="12" width="3" height="6"/><rect x="12" y="8" width="3" height="10"/><rect x="17" y="5" width="3" height="13"/>' },
  { to: '/admin', key: 'nav.servers', admin: true, match: (p) => p === '/admin' || p.startsWith('/admin/employee'),
    icon: '<path d="M12 2l8 4v6c0 5-3.5 8-8 10-4.5-2-8-5-8-10V6z"/><path d="M9 12l2 2 4-4"/>' },
  { to: '/admin/users', key: 'nav.staff', admin: true, match: (p) => p === '/admin/users',
    icon: '<path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/>' },
  { to: '/admin/sessions', key: 'nav.sessions', admin: true, match: (p) => p === '/admin/sessions',
    icon: '<rect x="2" y="3" width="20" height="14" rx="2"/><path d="M8 21h8M12 17v4"/>' },
  { to: '/admin/analytics', key: 'nav.analytics', admin: true, match: (p) => p === '/admin/analytics',
    icon: '<path d="M3 3v18h18"/><path d="M7 14l3-3 3 3 5-6"/>' },
  { to: '/admin/feedback', key: 'nav.feedback', admin: true, match: (p) => p === '/admin/feedback',
    icon: '<path d="M12 2l2.4 5 5.6.8-4 4 1 5.6L12 20l-5 2.4 1-5.6-4-4 5.6-.8z"/>' },
  { to: '/admin/audit-log', key: 'nav.auditLog', admin: true, match: (p) => p === '/admin/audit-log',
    icon: '<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><path d="M14 2v6h6M9 13h6M9 17h6"/>' },
  { to: '/tv', key: 'nav.tv', admin: false, match: (p) => p === '/tv',
    icon: '<rect x="2" y="3" width="20" height="14" rx="2"/><path d="M8 21h8M12 17v4"/>' },
]
const links = computed(() => allLinks.filter((l) => !l.admin || auth.isAdmin))
const isActive = (l) => l.match(route.path)
</script>

<template>
  <RouterView v-if="!showSidebar" />

  <div v-else class="layout" :class="{ collapsed }">
    <!-- ochish tugmasi (yopiq holatda) -->
    <button v-if="collapsed" class="reopen" @click="toggleSidebar" title="Menyu">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M3 12h18M3 6h18M3 18h18"/></svg>
    </button>

    <aside class="sidebar">
      <div class="sb-head">
        <RouterLink to="/" class="brand">
          <div class="brand__logo">
            <svg viewBox="0 0 32 32"><defs><linearGradient id="lg" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stop-color="#6d5efc"/><stop offset="1" stop-color="#14b8c4"/></linearGradient></defs><rect width="32" height="32" rx="9" fill="url(#lg)"/><path d="M9 20c0-3.9 3.1-7 7-7s7 3.1 7 7" fill="none" stroke="#fff" stroke-width="2.4" stroke-linecap="round"/><circle cx="16" cy="22" r="2.2" fill="#fff"/></svg>
          </div>
          <div class="brand__txt"><strong>Monitoring</strong><span>{{ t('app.subtitle') }}</span></div>
        </RouterLink>
        <button class="sb-toggle" @click="toggleSidebar" title="Yopish">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M15 18l-6-6 6-6"/></svg>
        </button>
      </div>

      <nav class="menu">
        <RouterLink v-for="l in links" :key="l.to" :to="l.to" class="menu__item" :class="{ active: isActive(l) }">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" v-html="l.icon"></svg>
          <span>{{ t(l.key) }}</span>
        </RouterLink>
      </nav>

      <!-- til switcher -->
      <div class="lang">
        <button v-for="l in LOCALES" :key="l.id" class="lang__btn" :class="{ on: locale === l.id }" @click="setLocale(l.id)">
          {{ l.flag }} <span>{{ l.id.toUpperCase() }}</span>
        </button>
      </div>

      <div v-if="auth.user" class="me">
        <div class="me__av">{{ (auth.user.name || auth.user.email).slice(0,2).toUpperCase() }}</div>
        <div class="me__info">
          <div class="me__name">{{ auth.user.name || auth.user.email }}</div>
          <div class="me__role">{{ auth.isAdmin ? t('role.admin') : t('role.operator') }}</div>
        </div>
        <button class="me__out" @click="logout" :title="t('common.logout')">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4M16 17l5-5-5-5M21 12H9"/></svg>
        </button>
      </div>

      <button class="theme-toggle" @click="toggleTheme">
        <span class="theme-toggle__track" :class="theme">
          <span class="theme-toggle__thumb">
            <svg v-if="theme === 'dark'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/></svg>
            <svg v-else viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="5"/><path d="M12 1v2M12 21v2M4.2 4.2l1.4 1.4M18.4 18.4l1.4 1.4M1 12h2M21 12h2M4.2 19.8l1.4-1.4M18.4 5.6l1.4-1.4"/></svg>
          </span>
        </span>
        <span class="theme-toggle__label">{{ theme === 'dark' ? t('common.theme.dark') : t('common.theme.light') }}</span>
      </button>
    </aside>

    <main class="content">
      <RouterView v-slot="{ Component }">
        <Transition name="page" mode="out-in"><component :is="Component" /></Transition>
      </RouterView>
    </main>
  </div>
</template>

<style scoped>
.layout { display: grid; grid-template-columns: 248px 1fr; min-height: 100vh; transition: grid-template-columns 0.3s cubic-bezier(.2,.8,.2,1); }
.layout.collapsed { grid-template-columns: 0 1fr; }

.sidebar {
  position: sticky; top: 0; height: 100vh; display: flex; flex-direction: column;
  padding: 20px 16px; background: var(--surface); border-right: 1px solid var(--border);
  transition: transform 0.3s cubic-bezier(.2,.8,.2,1); overflow: hidden; min-width: 248px;
}
.layout.collapsed .sidebar { transform: translateX(-100%); }

.reopen { position: fixed; top: 18px; left: 18px; z-index: 40; width: 42px; height: 42px; padding: 0;
  background: var(--surface); color: var(--text); border: 1px solid var(--border); box-shadow: var(--shadow); }
.reopen:hover { box-shadow: var(--shadow-lg); transform: none; }
.reopen svg { width: 20px; height: 20px; }

.sb-head { display: flex; align-items: center; justify-content: space-between; padding: 4px 6px 22px; }
.brand { display: flex; align-items: center; gap: 11px; text-decoration: none; color: var(--text); min-width: 0; }
.brand__logo svg { width: 36px; height: 36px; display: block; flex-shrink: 0; }
.brand__txt strong { display: block; font-size: 16px; font-weight: 800; letter-spacing: -0.02em; white-space: nowrap; }
.brand__txt span { font-size: 11px; color: var(--text-dim); white-space: nowrap; }
.sb-toggle { background: var(--surface-2); color: var(--text-dim); width: 30px; height: 30px; padding: 0; border-radius: 9px; flex-shrink: 0; }
.sb-toggle:hover { box-shadow: none; transform: none; color: var(--text); }
.sb-toggle svg { width: 16px; height: 16px; }

.menu { display: flex; flex-direction: column; gap: 4px; flex: 1; }
.menu__item { display: flex; align-items: center; gap: 12px; padding: 11px 14px; border-radius: 11px;
  font-size: 14px; font-weight: 600; text-decoration: none; color: var(--text-dim); transition: all 0.2s; white-space: nowrap; }
.menu__item svg { width: 18px; height: 18px; flex-shrink: 0; }
.menu__item:hover { background: var(--surface-2); color: var(--text); }
.menu__item.active { background: var(--grad); color: #fff; box-shadow: var(--glow); }

.lang { display: flex; gap: 4px; margin: 14px 0 10px; }
.lang__btn { flex: 1; background: var(--surface-2); color: var(--text-dim); border: 1px solid var(--border);
  padding: 7px 4px; font-size: 11px; font-weight: 600; border-radius: 9px; gap: 3px; }
.lang__btn:hover { box-shadow: none; transform: none; color: var(--text); }
.lang__btn.on { background: var(--grad-soft); color: var(--accent); border-color: var(--border-strong); }
.lang__btn span { font-size: 10px; }

.me { display: flex; align-items: center; gap: 10px; padding: 10px; margin-bottom: 10px;
  border-radius: 12px; background: var(--surface-2); border: 1px solid var(--border); }
.me__av { width: 34px; height: 34px; border-radius: 10px; flex-shrink: 0; display: grid; place-items: center;
  background: var(--grad); color: #fff; font-weight: 700; font-size: 12px; }
.me__info { min-width: 0; flex: 1; }
.me__name { font-size: 13px; font-weight: 600; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.me__role { font-size: 11px; color: var(--text-dim); }
.me__out { background: none; color: var(--text-faint); width: 30px; height: 30px; padding: 0; flex-shrink: 0; }
.me__out:hover { color: var(--red); background: rgba(239,68,68,0.1); box-shadow: none; transform: none; }
.me__out svg { width: 16px; height: 16px; }

.theme-toggle { display: flex; align-items: center; gap: 11px; background: var(--surface-2); border: 1px solid var(--border);
  color: var(--text-dim); padding: 8px 10px; border-radius: 12px; font-size: 12.5px; font-weight: 600; }
.theme-toggle:hover { transform: none; box-shadow: none; background: var(--surface-3); color: var(--text); }
.theme-toggle__track { width: 42px; height: 24px; border-radius: 999px; flex-shrink: 0; background: var(--bg);
  border: 1px solid var(--border-strong); position: relative; transition: background 0.3s; }
.theme-toggle__thumb { position: absolute; top: 2px; left: 2px; width: 18px; height: 18px; border-radius: 50%;
  background: var(--grad); color: #fff; display: grid; place-items: center; transition: transform 0.3s cubic-bezier(.2,.8,.2,1); }
.theme-toggle__track.light .theme-toggle__thumb { transform: translateX(18px); }
.theme-toggle__thumb svg { width: 11px; height: 11px; }

.content { min-width: 0; padding: 24px 30px 50px; }
.layout.collapsed .content { padding-left: 76px; }

.page-enter-active, .page-leave-active { transition: opacity 0.25s, transform 0.25s; }
.page-enter-from { opacity: 0; transform: translateY(10px); }
.page-leave-to { opacity: 0; transform: translateY(-6px); }

@media (max-width: 920px) {
  .layout { grid-template-columns: 0 1fr; }
  .sidebar { position: fixed; z-index: 45; }
  .layout:not(.collapsed) .sidebar { transform: translateX(0); box-shadow: var(--shadow-lg); }
  .layout.collapsed .sidebar { transform: translateX(-100%); }
  .content, .layout.collapsed .content { padding-left: 76px; }
}
</style>
