<template>
  <div class="layout">
    <aside class="sidebar">
      <div class="brand">
        <h2>SimpleC2</h2>
      </div>
      <nav class="nav">
        <router-link to="/" class="nav-item" active-class="active">
          <span class="icon">📊</span>
          Dashboard
        </router-link>
        <router-link to="/listeners" class="nav-item" active-class="active">
          <span class="icon">🎧</span>
          Listeners
        </router-link>
        <router-link to="/beacons" class="nav-item" active-class="active">
          <span class="icon">📡</span>
          Beacons
        </router-link>
      </nav>
    </aside>
    <main class="main-content">
      <header class="header">
        <div class="header-left">
          <!-- Breadcrumbs or Title could go here -->
        </div>
        <div class="header-right">
          <div class="user-profile">
            <span>{{ authStore.user }}</span>
            <Button variant="ghost" size="sm" @click="handleLogout" class="logout-btn">Logout</Button>
          </div>
        </div>
      </header>
      <div class="content-wrapper">
        <router-view></router-view>
      </div>
    </main>
  </div>
</template>

<script setup lang="ts">
import { useAuthStore } from '../stores/auth'
import Button from './ui/Button.vue'

const authStore = useAuthStore()

const handleLogout = () => {
  authStore.logout()
}
</script>

<style scoped>
.layout {
  display: flex;
  height: 100vh;
  width: 100vw;
  overflow: hidden;
  background-color: var(--color-bg-primary);
}

.sidebar {
  width: 260px;
  background-color: var(--color-bg-secondary);
  border-right: 1px solid var(--color-border);
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
}

.brand {
  height: 64px;
  display: flex;
  align-items: center;
  padding: 0 var(--spacing-lg);
  border-bottom: 1px solid var(--color-border);
}

.brand h2 {
  font-size: var(--font-size-lg);
  font-weight: 600;
  color: var(--color-primary);
}

.nav {
  padding: var(--spacing-md);
  display: flex;
  flex-direction: column;
  gap: var(--spacing-xs);
}

.nav-item {
  display: flex;
  align-items: center;
  padding: var(--spacing-sm) var(--spacing-md);
  color: var(--color-text-secondary);
  border-radius: var(--radius-md);
  font-weight: 500;
  transition: all 0.2s;
}

.nav-item:hover {
  background-color: var(--color-bg-tertiary);
  color: var(--color-text-primary);
}

.nav-item.active {
  background-color: var(--color-primary);
  color: var(--color-text-light);
}

.nav-item .icon {
  margin-right: var(--spacing-sm);
}

.main-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.header {
  height: 64px;
  background-color: var(--color-bg-secondary);
  border-bottom: 1px solid var(--color-border);
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 var(--spacing-xl);
}

.user-profile {
  display: flex;
  align-items: center;
  gap: var(--spacing-md);
}

.logout-btn {
  color: var(--color-danger);
}

.content-wrapper {
  flex: 1;
  overflow-y: auto;
  padding: var(--spacing-xl);
}

@media (max-width: 768px) {
  .sidebar {
    display: none; /* Mobile menu to be implemented */
  }
}
</style>
