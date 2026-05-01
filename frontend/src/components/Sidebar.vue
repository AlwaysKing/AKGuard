<template>
  <aside class="sidebar">
    <div class="sidebar-header">
      <span class="logo-icon">⚙</span>
      <span class="logo-text">{{ siteTitle }}</span>
    </div>

    <nav class="sidebar-nav">
      <!-- 仪表盘 -->
      <router-link to="/dashboard" class="nav-item" :class="{ active: currentRoute === 'Dashboard' }">
        <span class="nav-icon">📊</span>
        <span class="nav-label">仪表盘</span>
      </router-link>

      <!-- 认证管理 -->
      <div class="nav-group">
        <div class="nav-group-label">认证管理</div>
        <router-link to="/auth" class="nav-item" :class="{ active: currentRoute === 'Auth' }">
          <span class="nav-icon">🔑</span>
          <span class="nav-label">认证方式</span>
        </router-link>
        <router-link to="/networks" class="nav-item" :class="{ active: currentRoute === 'Networks' }">
          <span class="nav-icon">🌐</span>
          <span class="nav-label">网段管理</span>
        </router-link>
        <router-link to="/domains" class="nav-item" :class="{ active: currentRoute === 'Domains' }">
          <span class="nav-icon">🛡</span>
          <span class="nav-label">域名策略</span>
        </router-link>
      </div>

      <!-- 监控 -->
      <div class="nav-group">
        <div class="nav-group-label">监控</div>
        <router-link to="/logs" class="nav-item" :class="{ active: currentRoute === 'AccessLog' }">
          <span class="nav-icon">📋</span>
          <span class="nav-label">访问日志</span>
        </router-link>
        <router-link to="/audit" class="nav-item" :class="{ active: currentRoute === 'AuditLog' }">
          <span class="nav-icon">📝</span>
          <span class="nav-label">操作审计</span>
        </router-link>
        <router-link to="/sessions" class="nav-item" :class="{ active: currentRoute === 'Sessions' }">
          <span class="nav-icon">👤</span>
          <span class="nav-label">会话管理</span>
        </router-link>
        <router-link to="/blacklist" class="nav-item" :class="{ active: currentRoute === 'Blacklist' }">
          <span class="nav-icon">🚫</span>
          <span class="nav-label">黑名单</span>
        </router-link>
      </div>

      <!-- 底部推到最下面 -->
      <div class="nav-spacer"></div>

      <!-- 管理设置（底部） -->
      <router-link to="/admin-settings" class="nav-item" :class="{ active: currentRoute === 'AdminSettings' }">
        <span class="nav-icon">🔐</span>
        <span class="nav-label">管理设置</span>
      </router-link>
    </nav>
  </aside>
</template>

<script setup>
import { siteTitle } from '../router'
defineProps({
  currentRoute: String,
})
</script>

<style scoped>
.sidebar {
  width: var(--sidebar-width);
  height: 100vh;
  background: var(--bg-secondary);
  border-right: 1px solid var(--bg-hover);
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
  overflow: hidden;
}

.sidebar-header {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 18px 20px;
  border-bottom: 1px solid var(--bg-hover);
  flex-shrink: 0;
}

.logo-icon {
  font-size: 22px;
}

.logo-text {
  font-size: 16px;
  font-weight: 700;
  color: var(--primary-400);
}

.sidebar-nav {
  padding: 12px 8px;
  flex: 1;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
}

.nav-group {
  margin-top: 16px;
}

.nav-group:first-child {
  margin-top: 0;
}

.nav-group-label {
  padding: 4px 12px 6px;
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--text-muted);
}

.nav-spacer {
  flex: 1;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 9px 12px;
  border-radius: var(--radius-sm);
  color: var(--text-secondary);
  font-size: 14px;
  transition: all var(--transition-fast);
  text-decoration: none;
}

.nav-item:hover {
  background: var(--bg-hover);
  color: var(--text-primary);
}

.nav-item.active {
  background: rgba(99, 102, 241, 0.15);
  color: var(--primary-400);
}

.nav-icon {
  font-size: 16px;
  width: 20px;
  text-align: center;
  flex-shrink: 0;
}

.nav-label {
  white-space: nowrap;
}
</style>
