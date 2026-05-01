<template>
  <div class="dashboard">
    <!-- 统计卡片 -->
    <div class="stats-grid">
      <div class="stat-card">
        <div class="stat-value">{{ stats.total_today }}</div>
        <div class="stat-label">今日鉴权</div>
      </div>
      <div class="stat-card">
        <div class="stat-value" style="color: var(--error)">{{ stats.reject_count }}</div>
        <div class="stat-label">拦截次数</div>
      </div>
      <div class="stat-card">
        <div class="stat-value" style="color: var(--success)">{{ activeSessions }}</div>
        <div class="stat-label">活跃会话</div>
      </div>
      <div class="stat-card">
        <div class="stat-value" style="color: var(--info)">{{ domainsCount }}</div>
        <div class="stat-label">域名策略</div>
      </div>
    </div>

    <!-- 最近活动 -->
    <div class="card" style="margin-top: 24px">
      <h3 style="margin-bottom: 16px; font-size: 15px;">最近活动</h3>
      <div v-if="loading" class="loading">加载中...</div>
      <table v-else-if="recentLogs.length">
        <thead>
          <tr>
            <th>时间</th>
            <th>域名</th>
            <th>来源 IP</th>
            <th>来源</th>
            <th>动作</th>
            <th>结果</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="log in recentLogs" :key="log.id">
            <td>{{ formatTime(log.created_at) }}</td>
            <td>{{ log.host }}</td>
            <td>{{ log.client_ip }}</td>
            <td>
              <span :class="['badge', log.source === 'internal' ? 'badge-info' : 'badge-warning']">
                {{ log.source === 'internal' ? '内网' : '外网' }}
              </span>
            </td>
            <td>
              <span :class="['badge', actionBadge(log.action)]">{{ log.action }}</span>
            </td>
            <td>
              <span :class="['badge', log.result === 'allowed' ? 'badge-success' : 'badge-error']">
                {{ log.result === 'allowed' ? '放行' : '拦截' }}
              </span>
            </td>
          </tr>
        </tbody>
      </table>
      <div v-else class="empty-state">暂无访问记录</div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import api from '../api'

const stats = ref({ total_today: 0, reject_count: 0, pass_count: 0, auth_count: 0 })
const activeSessions = ref(0)
const domainsCount = ref(0)
const recentLogs = ref([])
const loading = ref(true)

onMounted(async () => {
  try {
    const data = await api.getDashboard()
    stats.value = data.stats || stats.value
    activeSessions.value = data.active_sessions || 0
    domainsCount.value = data.domains_count || 0
    recentLogs.value = data.recent_logs || []
  } catch (e) {
    console.error('Failed to load dashboard:', e)
  } finally {
    loading.value = false
  }
})

function formatTime(t) {
  if (!t) return ''
  return t.replace('T', ' ').substring(0, 19)
}

function actionBadge(action) {
  const map = { pass: 'badge-success', reject: 'badge-error', auth: 'badge-info' }
  return map[action] || 'badge-info'
}
</script>

<style scoped>
.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 16px;
}

.stat-card {
  background: var(--bg-card);
  border-radius: var(--radius-md);
  padding: 24px;
  text-align: center;
  box-shadow: var(--shadow-sm);
}

.stat-value {
  font-size: 32px;
  font-weight: 700;
  color: var(--primary-400);
  line-height: 1;
}

.stat-label {
  margin-top: 8px;
  font-size: 13px;
  color: var(--text-muted);
}
</style>
