<template>
  <div class="card">
    <div class="log-header">
      <h3>访问日志</h3>
      <div class="filter-row">
        <input v-model="hostFilter" class="input" placeholder="按域名筛选" style="width: 200px" @keyup.enter="loadLogs(1)">
        <button class="btn btn-ghost btn-sm" @click="loadLogs(1)">筛选</button>
      </div>
    </div>

    <div v-if="loading" class="loading">加载中...</div>
    <template v-else>
      <table v-if="logs.length">
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
          <tr v-for="log in logs" :key="log.id">
            <td>{{ formatTime(log.created_at) }}</td>
            <td>{{ log.host }}</td>
            <td>{{ log.client_ip }}</td>
            <td>
              <span :class="['badge', log.source === 'internal' ? 'badge-info' : 'badge-warning']">
                {{ log.source === 'internal' ? '内网' : '外网' }}
              </span>
            </td>
            <td>
              <span :class="['badge', actionBadge(log.action)]">{{ actionLabel(log.action) }}</span>
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

      <div class="pagination" v-if="total > pageSize">
        <button :disabled="page <= 1" @click="loadLogs(page - 1)">上一页</button>
        <button class="active">{{ page }} / {{ totalPages }}</button>
        <button :disabled="page >= totalPages" @click="loadLogs(page + 1)">下一页</button>
      </div>
    </template>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import api from '../api'

const logs = ref([])
const loading = ref(true)
const page = ref(1)
const pageSize = 20
const total = ref(0)
const hostFilter = ref('')

const totalPages = computed(() => Math.ceil(total.value / pageSize))

onMounted(() => loadLogs(1))

async function loadLogs(p) {
  loading.value = true
  page.value = p
  try {
    const data = await api.getAccessLogs(p, pageSize, hostFilter.value)
    logs.value = data.logs || []
    total.value = data.total || 0
  } catch (e) {
    console.error(e)
  } finally {
    loading.value = false
  }
}

function formatTime(t) {
  if (!t) return ''
  return t.replace('T', ' ').substring(0, 19)
}

function actionBadge(action) {
  const map = { pass: 'badge-success', reject: 'badge-error', auth: 'badge-info' }
  return map[action] || 'badge-info'
}

function actionLabel(action) {
  const map = { pass: '放行', reject: '拒绝', auth: '认证' }
  return map[action] || action
}
</script>

<style scoped>
.log-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.filter-row {
  display: flex;
  gap: 8px;
}
</style>
