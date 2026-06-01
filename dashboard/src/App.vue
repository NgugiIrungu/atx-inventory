<template>
  <div class="app">
    <!-- Header -->
    <header class="header">
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
      </div>
    </header>

    <!-- Page content -->
    <main class="main">
      <RouterView />
    </main>
  </div>
</template>

<script setup>
import { RouterLink, RouterView } from 'vue-router'
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
}

.brand { display: flex; align-items: center; gap: 12px; }
.brand-icon { font-size: 28px; }
.brand-name { font-size: 18px; font-weight: 700; letter-spacing: 0.5px; }
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

.main { max-width: 1400px; margin: 0 auto; padding: 2rem; }
</style>