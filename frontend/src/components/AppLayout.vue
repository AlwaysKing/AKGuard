<template>
  <div class="app-layout">
    <Sidebar :current-route="$route.name" />
    <div class="main-area">
      <header class="top-bar">
        <h2 class="page-title">{{ $route.meta.title }}</h2>
        <div class="top-actions">
          <button class="btn btn-ghost btn-sm" @click="toggleTheme">
            {{ isDark ? '☀️' : '🌙' }}
          </button>
          <button class="btn btn-ghost btn-sm" @click="handleLogout">退出</button>
        </div>
      </header>
      <main class="content">
        <router-view />
      </main>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import Sidebar from './Sidebar.vue'
import api from '../api'
import { siteTitle } from '../router'

const router = useRouter()
const isDark = ref(true)

onMounted(async () => {
  const saved = localStorage.getItem('ak-theme')
  if (saved === 'light') {
    isDark.value = false
    document.documentElement.setAttribute('data-theme', 'light')
  }
  try {
    const data = await api.getConfig()
    if (data.site_title) {
      siteTitle.value = data.site_title
      localStorage.setItem('ak-site-title', data.site_title)
    }
    document.title = siteTitle.value
  } catch {}
})

function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.setAttribute('data-theme', isDark.value ? '' : 'light')
  localStorage.setItem('ak-theme', isDark.value ? 'dark' : 'light')
}

async function handleLogout() {
  try {
    await api.adminLogout()
  } catch {}
  router.push('/adminlogin')
}
</script>

<style scoped>
.app-layout {
  display: flex;
  height: 100vh;
  overflow: hidden;
}

.main-area {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.top-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 24px;
  background: var(--bg-secondary);
  border-bottom: 1px solid var(--bg-hover);
  flex-shrink: 0;
}

.page-title {
  font-size: 18px;
  font-weight: 600;
  color: var(--text-primary);
}

.top-actions {
  display: flex;
  gap: 8px;
}

.content {
  flex: 1;
  overflow-y: auto;
  padding: 24px;
}
</style>
