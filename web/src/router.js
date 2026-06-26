import { createRouter, createWebHistory } from 'vue-router'
import Dashboard from './views/Dashboard.vue'
import Admin from './views/Admin.vue'
import EmployeeDetail from './views/EmployeeDetail.vue'
import TvDashboard from './views/TvDashboard.vue'
import Login from './views/Login.vue'
import Users from './views/Users.vue'
import Sessions from './views/Sessions.vue'
import Surveys from './views/Surveys.vue'
import SurveyAdmin from './views/SurveyAdmin.vue'
import { auth } from './auth.js'
import { api } from './api.js'

const routes = [
  { path: '/login', name: 'login', component: Login, meta: { public: true } },
  { path: '/tv', name: 'tv', component: TvDashboard, meta: { fullscreen: true, public: true } },
  { path: '/', name: 'dashboard', component: Dashboard },
  { path: '/survey', name: 'survey', component: Surveys },
  { path: '/admin', name: 'admin', component: Admin, meta: { admin: true } },
  { path: '/admin/survey', name: 'survey-admin', component: SurveyAdmin, meta: { admin: true } },
  { path: '/admin/users', name: 'users', component: Users, meta: { admin: true } },
  { path: '/admin/sessions', name: 'sessions', component: Sessions, meta: { admin: true } },
  { path: '/admin/employee/:id', name: 'employee', component: EmployeeDetail, props: true, meta: { admin: true } },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
  scrollBehavior: () => ({ top: 0 }),
})

// Auth guard — birinchi navigatsiyada /me yuklanadi
let loaded = false
router.beforeEach(async (to) => {
  if (auth.token && !auth.user && !loaded) {
    loaded = true
    try { auth.user = await api.me() } catch { auth.clear() }
    auth.ready = true
  }
  if (to.meta.public) return true
  if (!auth.authed) return { path: '/login', query: { redirect: to.fullPath } }
  if (to.meta.admin && !auth.isAdmin) return { path: '/' }
  return true
})

export default router
