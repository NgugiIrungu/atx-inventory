<template>
  <div class="app">
    <!-- Header — only show when logged in -->
    <header v-if="auth.isLoggedIn" class="header">
      <div class="header-inner">
        <div class="brand">
          <span class="brand-icon">📡</span>
          <div>
            <div class="brand-name">ATX Technology</div>
            <div class="brand-sub">Inventory Management System</div>
          </div>
        </div>
        <nav class="nav">
          <RouterLink to="/" class="nav-link">📦 Inventory</RouterLink>
          <RouterLink to="/ai" class="nav-link">🤖 AI Agent</RouterLink>
        </nav>
        <div class="user-info">
          <div class="user-details">
            <span class="user-name">{{ auth.username }}</span>
            <span class="user-role" :class="auth.isAdmin ? 'role-admin' : 'role-user'">
              {{ auth.isAdmin ? 'Admin' : 'User' }}
            </span>
          </div>
          <button class="btn-logout" @click="logout">Sign Out</button>
        </div>
      </div>
    </header>

    <main :class="auth.isLoggedIn ? 'main' : ''">
      <RouterView />
    </main>
  </div>
</template>

<script setup>
import { RouterLink, RouterView } from 'vue-router'
import { useRouter } from 'vue-router'
import { useAuthStore } from './stores/auth'

const auth = useAuthStore()
const router = useRouter()

// Restore token on page load
auth.init()

function logout() {
  auth.logout()
  router.push('/login')
}
</script>

<style>
* { box-sizing: border-box; margin: 0; padding: 0; }

body {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
  background: #f4f6f9;
  color: #1a1a2e;
  min-height: 100vh;
}

.app { min-height: 100vh; }

.header {
  background: #1a1a2e;
  color: white;
  padding: 0 2rem;
  position: sticky;
  top: 0;
  z-index: 100;
  box-shadow: 0 2px 8px rgba(0,0,0,0.3);
}

.header-inner {
  max-width: 1400px;
  margin: 0 auto;
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 64px;
  gap: 1rem;
}

.brand { display: flex; align-items: center; gap: 12px; flex-shrink: 0; }
.brand-icon { font-size: 28px; }
.brand-name { font-size: 18px; font-weight: 700; }
.brand-sub { font-size: 11px; color: #8892b0; }

.nav { display: flex; gap: 8px; }
.nav-link {
  color: #8892b0;
  text-decoration: none;
  padding: 6px 16px;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 500;
  transition: all 0.2s;
}
.nav-link:hover { background: rgba(255,255,255,0.1); color: white; }
.nav-link.router-link-active { background: #0070f3; color: white; }

.user-info { display: flex; align-items: center; gap: 12px; flex-shrink: 0; }
.user-details { display: flex; flex-direction: column; align-items: flex-end; }
.user-name { font-size: 13px; font-weight: 500; color: white; }
.user-role {
  font-size: 10px;
  font-weight: 600;
  padding: 1px 6px;
  border-radius: 10px;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}
.role-admin { background: #f6e05e; color: #744210; }
.role-user { background: #90cdf4; color: #2c5282; }

.btn-logout {
  padding: 6px 14px;
  background: rgba(255,255,255,0.1);
  color: white;
  border: 1px solid rgba(255,255,255,0.2);
  border-radius: 8px;
  font-size: 13px;
  cursor: pointer;
  transition: all 0.2s;
}
.btn-logout:hover { background: rgba(255,255,255,0.2); }

.main { max-width: 1400px; margin: 0 auto; padding: 2rem; }
</style>