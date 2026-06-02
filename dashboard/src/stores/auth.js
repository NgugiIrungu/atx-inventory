import { defineStore } from 'pinia'
import axios from 'axios'

const API = 'http://localhost:8080'

export const useAuthStore = defineStore('auth', {
  state: () => ({
    user: JSON.parse(localStorage.getItem('atx_user') || 'null'),
    token: localStorage.getItem('atx_token') || null,
    error: null,
  }),

  getters: {
    isLoggedIn: (state) => !!state.token,
    isAdmin: (state) => state.user?.role === 'admin',
    username: (state) => state.user?.username || '',
  },

  actions: {
    async login(username, password) {
      this.error = null
      try {
        const res = await axios.post(`${API}/login`, { username, password })
        if (!res.data.success) {
          this.error = res.data.message
          return false
        }
        this.token = res.data.token
        this.user = res.data.user
        localStorage.setItem('atx_token', this.token)
        localStorage.setItem('atx_user', JSON.stringify(this.user))
        axios.defaults.headers.common['Authorization'] = `Bearer ${this.token}`
        return true
      } catch (e) {
        this.error = 'Invalid username or password'
        return false
      }
    },

    async register(username, email, password, role) {
      this.error = null
      try {
        const res = await axios.post(`${API}/register`, { username, email, password, role })
        if (!res.data.success) {
          this.error = res.data.message
          return false
        }
        this.token = res.data.token
        this.user = res.data.user
        localStorage.setItem('atx_token', this.token)
        localStorage.setItem('atx_user', JSON.stringify(this.user))
        axios.defaults.headers.common['Authorization'] = `Bearer ${this.token}`
        return true
      } catch (e) {
        this.error = 'Registration failed'
        return false
      }
    },

    logout() {
      this.token = null
      this.user = null
      this.error = null
      localStorage.removeItem('atx_token')
      localStorage.removeItem('atx_user')
      delete axios.defaults.headers.common['Authorization']
    },

    // Restore token on page reload
    init() {
      if (this.token) {
        axios.defaults.headers.common['Authorization'] = `Bearer ${this.token}`
      }
    },
  },
})