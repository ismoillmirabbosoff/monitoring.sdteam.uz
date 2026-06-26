import { reactive } from 'vue'

// Global auth holati (sessiya token + joriy foydalanuvchi).
export const auth = reactive({
  token: localStorage.getItem('session_token') || '',
  user: null,
  ready: false,

  get authed() { return !!this.token },
  get isAdmin() { return this.user?.role === 'admin' },

  setSession(token, user) {
    this.token = token
    this.user = user
    localStorage.setItem('session_token', token)
  },
  clear() {
    this.token = ''
    this.user = null
    localStorage.removeItem('session_token')
  },
})
