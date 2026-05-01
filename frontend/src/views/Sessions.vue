<template>
  <div class="card">
    <h3 style="margin-bottom: 16px;">会话管理</h3>

    <div v-if="loading" class="loading">加载中...</div>
    <template v-else>
      <table v-if="sessions.length">
        <thead>
          <tr>
            <th>类型</th>
            <th>来源 IP</th>
            <th>User-Agent</th>
            <th>默认密码</th>
            <th>创建时间</th>
            <th>过期时间</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="s in sessions" :key="s.id">
            <td>
              <span :class="['badge', s.token_type === 'admin' ? 'badge-warning' : 'badge-info']">
                {{ s.token_type === 'admin' ? '管理' : '鉴权' }}
              </span>
            </td>
            <td>{{ s.client_ip || '-' }}</td>
            <td style="max-width: 200px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;">
              {{ s.user_agent || '-' }}
            </td>
            <td>
              <span v-if="s.is_init" class="badge badge-warning">是</span>
              <span v-else style="color: var(--text-muted)">-</span>
            </td>
            <td>{{ formatTime(s.created_at) }}</td>
            <td>{{ formatTime(s.expires_at) }}</td>
            <td>
              <button class="btn btn-danger btn-sm" @click="killSession(s.id)">强制下线</button>
            </td>
          </tr>
        </tbody>
      </table>
      <div v-else class="empty-state">暂无活跃会话</div>
    </template>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import api from '../api'
import { useToast } from '../composables/toast'

const toast = useToast()
const sessions = ref([])
const loading = ref(true)

onMounted(() => loadSessions())

async function loadSessions() {
  loading.value = true
  try {
    const data = await api.getSessions()
    sessions.value = data.sessions || []
  } catch (e) {
    console.error(e)
  } finally {
    loading.value = false
  }
}

async function killSession(id) {
  if (!confirm('确定要强制下线此会话吗？')) return
  try {
    await api.deleteSession(id)
    sessions.value = sessions.value.filter(s => s.id !== id)
    toast.success('会话已强制下线')
  } catch (e) {
    toast.error(e.message || '操作失败')
  }
}

function formatTime(t) {
  if (!t) return ''
  return t.replace('T', ' ').substring(0, 19)
}
</script>
