<template>
  <div class="login-page">
    <div class="login-card">

      <!-- Logo -->
      <div class="login-logo">
        <span class="logo-icon">📡</span>
        <div class="logo-name">ATX Technology</div>
        <div class="logo-sub">Inventory Management System</div>
      </div>

      <!-- Tabs -->
      <div class="auth-tabs">
        <button :class="['tab', { active: mode === 'login' }]" @click="mode = 'login'">
          Sign In
        </button>
        <button :class="['tab', { active: mode === 'register' }]" @click="mode = 'register'">
          Register
        </button>
      </div>

      <!-- Error -->
      <div v-if="auth.error" class="error-box">
        ⚠ {{ auth.error }}
      </div>

      <!-- Login Form -->
      <div v-if="mode === 'login'">
        <div class="form-group">
          <label>Username</label>
          <input v-model="username" placeholder="Enter your username" @keydown.enter="submit" />
        </div>
        <div class="form-group">
          <label>Password</label>
          <input v-model="password" type="password" placeholder="Enter your password" @keydown.enter="submit" />
        </div>
        <button class="btn-submit" @click="submit" :disabled="loading">
          {{ loading ? 'Signing in...' : 'Sign In' }}
        </button>
        <div class="hint">
          Default admin: <strong>admin</strong> / <strong>atx@admin2024</strong>
        </div>
      </div>

      <!-- Register Form -->
      <div v-if="mode === 'register'">
        <div class="form-group">
          <label>Username</label>
          <input v-model="username" placeholder="Choose a username" />
        </div>
        <div class="form-group">
          <label>Email</label>
          <input v-model="email" type="email" placeholder="your@email.com" />
        </div>
        <div class="form-group">
          <label>Password</label>
          <input v-model="password" type="password" placeholder="Choose a password" />
        </div>
        <div class="form-group">
          <label>Role</label>
          <select v-model="role">
            <option value="user">User — can view and update stock</option>
            <option value="admin">Admin — full access</option>
          </select>
        </div>
        <button class="btn-submit" @click="submit" :disabled="loading">
          {{ loading ? 'Creating account...' : 'Create Account' }}
        </button>
      </div>

    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const auth = useAuthStore()
const router = useRouter()

const mode = ref('login')
const username = ref('')
const email = ref('')
const password = ref('')
const role = ref('user')
const loading = ref(false)

async function submit() {
  if (!username.value || !password.value) {
    auth.error = 'Please fill in all fields'
    return
  }

  loading.value = true
  let success = false

  if (mode.value === 'login') {
    success = await auth.login(username.value, password.value)
  } else {
    if (!email.value) {
      auth.error = 'Please enter your email'
      loading.value = false
      return
    }
    success = await auth.register(username.value, email.value, password.value, role.value)
  }

  loading.value = false
  if (success) {
    router.push('/')
  }
}
</script>

<style scoped>
.login-page {
  min-height: 100vh;
  background: #1a1a2e;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 1rem;
}

.login-card {
  background: white;
  border-radius: 20px;
  padding: 2.5rem;
  width: 100%;
  max-width: 420px;
  box-shadow: 0 20px 60px rgba(0,0,0,0.4);
}

.login-logo {
  text-align: center;
  margin-bottom: 2rem;
}
.logo-icon { font-size: 48px; display: block; margin-bottom: 8px; }
.logo-name { font-size: 22px; font-weight: 700; color: #1a1a2e; }
.logo-sub { font-size: 13px; color: #888; margin-top: 4px; }

.auth-tabs {
  display: flex;
  background: #f4f6f9;
  border-radius: 10px;
  padding: 4px;
  margin-bottom: 1.5rem;
}
.tab {
  flex: 1;
  padding: 8px;
  border: none;
  background: transparent;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 500;
  color: #666;
  cursor: pointer;
  transition: all 0.2s;
}
.tab.active {
  background: white;
  color: #1a1a2e;
  box-shadow: 0 1px 4px rgba(0,0,0,0.1);
}

.error-box {
  background: #fff5f5;
  color: #e53e3e;
  border: 1px solid #fed7d7;
  border-radius: 8px;
  padding: 10px 14px;
  font-size: 13px;
  margin-bottom: 1rem;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 5px;
  margin-bottom: 1rem;
}
.form-group label {
  font-size: 12px;
  font-weight: 600;
  color: #555;
  text-transform: uppercase;
  letter-spacing: 0.04em;
}
.form-group input,
.form-group select {
  padding: 10px 14px;
  border: 1.5px solid #e2e8f0;
  border-radius: 10px;
  font-size: 14px;
  outline: none;
  transition: border 0.2s;
}
.form-group input:focus,
.form-group select:focus {
  border-color: #0070f3;
}

.btn-submit {
  width: 100%;
  padding: 12px;
  background: #0070f3;
  color: white;
  border: none;
  border-radius: 10px;
  font-size: 15px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
  margin-top: 0.5rem;
}
.btn-submit:hover:not(:disabled) { background: #0060d3; }
.btn-submit:disabled { opacity: 0.6; cursor: not-allowed; }

.hint {
  text-align: center;
  font-size: 12px;
  color: #888;
  margin-top: 1rem;
}
</style>