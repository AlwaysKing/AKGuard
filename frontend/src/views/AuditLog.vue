<template>
  <div class="card">
    <h3 style="margin-bottom: 16px;">操作审计</h3>

    <div v-if="loading" class="loading">加载中...</div>
    <template v-else>
      <table v-if="logs.length">
        <thead>
          <tr>
            <th>时间</th>
            <th>操作</th>
            <th>详情</th>
            <th>来源 IP</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="log in logs" :key="log.id">
            <td>{{ formatTime(log.created_at) }}</td>
            <td>
              <span :class="['badge', actionBadge(log.action)]">{{ actionLabel(log.action) }}</span>
            </td>
            <td class="detail-cell">{{ formatDetail(log.action, log.detail) }}</td>
            <td>{{ log.client_ip || '-' }}</td>
          </tr>
        </tbody>
      </table>
      <div v-else class="empty-state">暂无审计记录</div>

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

const totalPages = computed(() => Math.ceil(total.value / pageSize))

onMounted(() => loadLogs(1))

async function loadLogs(p) {
  loading.value = true
  page.value = p
  try {
    const data = await api.getAuditLogs(p, pageSize)
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

const actionLabels = {
  change_auth_password: '修改认证密码',
  change_admin_password: '修改管理密码',
  update_bark: '更新 Bark 推送',
  update_networks: '更新内网网段',
  update_domains: '更新域名策略',
  update_default_policy: '更新默认策略',
  update_admin_login_methods: '更新管理登录方式',
  update_auth_login_methods: '更新认证登录方式',
  update_auth_ban_config: '更新认证封禁配置',
  update_admin_ban_config: '更新管理封禁配置',
  add_blacklist: '添加黑名单',
  remove_blacklist: '移除黑名单',
  update_site_title: '更新站点标题',
  delete_session: '注销会话',
  admin_login: '管理员登录',
  admin_logout: '管理员登出',
  auth_login: '用户登录',
  auth_logout: '用户登出',
}

function actionLabel(action) {
  return actionLabels[action] || action
}

function actionBadge(action) {
  if (action.includes('password')) return 'badge-warning'
  if (action.includes('delete')) return 'badge-error'
  if (action.includes('login')) return 'badge-success'
  return 'badge-info'
}

function formatDetail(action, detail) {
  if (!detail) return '-'
  let obj
  try { obj = JSON.parse(detail) } catch { return detail }

  switch (action) {
    case 'update_bark':
      return obj.bark_url || detail
    case 'update_networks':
      return (obj.networks || []).join('、')
    case 'update_domains': {
      const domains = obj.domains || []
      return domains.map(d => d.host).join('、')
    }
    case 'update_default_policy':
      return `内网: ${policyLabel(obj.internal)}，外网: ${policyLabel(obj.external)}`
    case 'delete_session':
      return `会话 ${obj.session_id ? obj.session_id.substring(0, 8) + '...' : ''}`
    case 'update_admin_login_methods':
    case 'update_auth_login_methods':
      return `密码: ${obj.password_login ? '开' : '关'}，Bark: ${obj.bark_login ? '开' : '关'}`
    case 'add_blacklist':
      return `${obj.ip || ''}（${obj.type === 'admin' ? '管理' : '认证'}${obj.duration_sec ? '，' + obj.duration_sec + '秒' : '，永久'}）`
    case 'remove_blacklist':
      return `${obj.ip || ''}（${obj.type === 'admin' ? '管理' : '认证'}）`
    case 'update_auth_ban_config':
    case 'update_admin_ban_config':
      return `${obj.enabled ? '启用' : '关闭'}，窗口 ${obj.window_sec}秒，${obj.max_attempts}次，封禁 ${obj.duration_sec}秒`
    case 'update_site_title':
      return obj.title || detail
    default:
      return detail
  }
}

function policyLabel(val) {
  const map = { pass: '放行', reject: '拒绝', auth: '认证' }
  return map[val] || val
}
</script>

<style scoped>
.detail-cell {
  max-width: 360px;
  word-break: break-all;
}
</style>
