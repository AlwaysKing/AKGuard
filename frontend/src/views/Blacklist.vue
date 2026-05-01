<template>
  <div class="blacklist-page">
    <div class="ban-row">
    <!-- 认证登录封禁 -->
    <div class="card">
      <div class="card-header">
        <div>
          <h3>认证登录封禁</h3>
          <p class="desc">普通用户登录失败达到阈值后自动封禁 IP</p>
        </div>
        <div class="toggle-row">
          <label class="toggle">
            <input type="checkbox" :checked="authBan.enabled" @change="toggleAuthBan">
            <span class="toggle-slider"></span>
          </label>
          <span class="toggle-label">{{ authBan.enabled ? '已启用' : '未启用' }}</span>
        </div>
      </div>
      <div class="ban-config">
        <div v-if="authBan.enabled" class="config-fields">
          <div class="field">
            <label>统计窗口</label>
            <select v-model.number="authBan.window_sec" @change="saveAuthBan">
              <option :value="60">1 分钟</option>
              <option :value="180">3 分钟</option>
              <option :value="300">5 分钟</option>
              <option :value="600">10 分钟</option>
              <option :value="900">15 分钟</option>
            </select>
          </div>
          <div class="field">
            <label>最大失败次数</label>
            <input v-model.number="authBan.max_attempts" type="number" min="1" max="100" class="input sm" @change="saveAuthBan">
          </div>
          <div class="field">
            <label>封禁时长</label>
            <select v-model.number="authBan.duration_sec" @change="saveAuthBan">
              <option :value="300">5 分钟</option>
              <option :value="900">15 分钟</option>
              <option :value="1800">30 分钟</option>
              <option :value="3600">1 小时</option>
              <option :value="21600">6 小时</option>
              <option :value="86400">24 小时</option>
            </select>
          </div>
        </div>
      </div>
    </div>

    <!-- 管理登录封禁 -->
    <div class="card">
      <div class="card-header">
        <div>
          <h3>管理登录封禁</h3>
          <p class="desc">管理员登录失败达到阈值后自动封禁 IP</p>
        </div>
        <div class="toggle-row">
          <label class="toggle">
            <input type="checkbox" :checked="adminBan.enabled" @change="toggleAdminBan">
            <span class="toggle-slider"></span>
          </label>
          <span class="toggle-label">{{ adminBan.enabled ? '已启用' : '未启用' }}</span>
        </div>
      </div>
      <div class="ban-config">
        <div v-if="adminBan.enabled" class="config-fields">
          <div class="field">
            <label>统计窗口</label>
            <select v-model.number="adminBan.window_sec" @change="saveAdminBan">
              <option :value="60">1 分钟</option>
              <option :value="180">3 分钟</option>
              <option :value="300">5 分钟</option>
              <option :value="600">10 分钟</option>
            </select>
          </div>
          <div class="field">
            <label>最大失败次数</label>
            <input v-model.number="adminBan.max_attempts" type="number" min="1" max="100" class="input sm" @change="saveAdminBan">
          </div>
          <div class="field">
            <label>封禁时长</label>
            <select v-model.number="adminBan.duration_sec" @change="saveAdminBan">
              <option :value="300">5 分钟</option>
              <option :value="900">15 分钟</option>
              <option :value="1800">30 分钟</option>
              <option :value="3600">1 小时</option>
              <option :value="86400">24 小时</option>
            </select>
          </div>
        </div>
      </div>
    </div>
    </div>

    <!-- 黑名单列表 -->
    <div class="card">
      <h3>黑名单列表</h3>
      <div class="add-row">
        <input v-model="newIP" class="input" placeholder="输入 IP 地址，如 1.2.3.4" @keyup.enter="addManual">
        <select v-model="newType" class="select-sm">
          <option value="auth">认证</option>
          <option value="admin">管理</option>
        </select>
        <select v-model.number="manualDuration" class="select-sm">
          <option :value="0">永久</option>
          <option :value="3600">1 小时</option>
          <option :value="86400">24 小时</option>
          <option :value="604800">7 天</option>
        </select>
        <button class="btn btn-primary" :disabled="!newIP || loading" @click="addManual">封禁</button>
      </div>

      <div v-if="entries.length" class="list-container">
        <div v-for="e in entries" :key="e.id" class="list-item">
          <div class="item-info">
            <span class="item-ip">{{ e.ip }}</span>
            <span class="badge badge-type">{{ e.type === 'admin' ? '管理' : '认证' }}</span>
            <span :class="['badge', e.reason === 'auto' ? 'badge-info' : 'badge-warning']">
              {{ e.reason === 'auto' ? '自动' : '手动' }}
            </span>
            <span class="item-time">
              {{ e.expires_at ? '剩余 ' + formatRemaining(e.expires_at) : '永久' }}
            </span>
          </div>
          <button class="btn btn-ghost btn-sm" @click="remove(e.id)">解封</button>
        </div>
      </div>
      <div v-else class="empty-state">暂无封禁记录</div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import api from '../api'
import { useToast } from '../composables/toast'

const toast = useToast()
const loading = ref(false)
const entries = ref([])
const newIP = ref('')
const newType = ref('auth')
const manualDuration = ref(0)

const authBan = reactive({
  enabled: false,
  window_sec: 300,
  max_attempts: 5,
  duration_sec: 3600,
})

const adminBan = reactive({
  enabled: false,
  window_sec: 300,
  max_attempts: 5,
  duration_sec: 3600,
})

onMounted(async () => {
  try {
    const data = await api.getConfig()
    const ab = data.auth_ban || {}
    authBan.enabled = ab.enabled ?? false
    authBan.window_sec = ab.window_sec ?? 300
    authBan.max_attempts = ab.max_attempts ?? 5
    authBan.duration_sec = ab.duration_sec ?? 3600
    const bb = data.admin_ban || {}
    adminBan.enabled = bb.enabled ?? false
    adminBan.window_sec = bb.window_sec ?? 300
    adminBan.max_attempts = bb.max_attempts ?? 5
    adminBan.duration_sec = bb.duration_sec ?? 3600
  } catch {}
  loadList()
})

async function loadList() {
  try {
    const [authData, adminData] = await Promise.all([
      api.getBlacklist('auth'),
      api.getBlacklist('admin'),
    ])
    const authEntries = (authData.entries || []).map(e => ({ ...e, type: 'auth' }))
    const adminEntries = (adminData.entries || []).map(e => ({ ...e, type: 'admin' }))
    entries.value = [...authEntries, ...adminEntries]
  } catch {}
}

async function toggleAuthBan() {
  authBan.enabled = !authBan.enabled
  await saveAuthBan()
}

async function toggleAdminBan() {
  adminBan.enabled = !adminBan.enabled
  await saveAdminBan()
}

async function saveAuthBan() {
  loading.value = true
  try {
    await api.updateAuthBanConfig({ ...authBan })
    toast.success('认证封禁配置已更新')
  } catch (e) {
    toast.error(e.message || '保存失败')
  } finally {
    loading.value = false
  }
}

async function saveAdminBan() {
  loading.value = true
  try {
    await api.updateAdminBanConfig({ ...adminBan })
    toast.success('管理封禁配置已更新')
  } catch (e) {
    toast.error(e.message || '保存失败')
  } finally {
    loading.value = false
  }
}

async function addManual() {
  if (!newIP.value) return
  loading.value = true
  try {
    await api.addBlacklist(newIP.value, newType.value, manualDuration.value)
    toast.success(`已封禁 ${newIP.value}（${newType.value === 'admin' ? '管理' : '认证'}）`)
    newIP.value = ''
    loadList()
  } catch (e) {
    toast.error(e.message || '封禁失败')
  } finally {
    loading.value = false
  }
}

async function remove(id) {
  try {
    await api.deleteBlacklist(id)
    toast.success('已解封')
    loadList()
  } catch (e) {
    toast.error(e.message || '解封失败')
  }
}

function formatRemaining(expiresAt) {
  const diff = new Date(expiresAt) - new Date()
  if (diff <= 0) return '即将到期'
  const mins = Math.floor(diff / 60000)
  if (mins < 60) return `${mins} 分钟`
  const hours = Math.floor(mins / 60)
  if (hours < 24) return `${hours} 小时`
  return `${Math.floor(hours / 24)} 天`
}
</script>

<style scoped>
.blacklist-page {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.ban-row {
  display: flex;
  gap: 20px;
}

.ban-row .card {
  flex: 1;
  min-width: 0;
}

.desc {
  font-size: 13px;
  color: var(--text-muted);
  margin: 4px 0 0;
}

.card-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
}

.card-header h3 {
  margin: 0;
}

.ban-config {
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid var(--bg-hover);
}

.toggle-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 16px;
}

.toggle-label {
  font-size: 14px;
  color: var(--text-secondary);
}

/* Toggle 开关 */
.toggle {
  position: relative;
  display: inline-block;
  width: 44px;
  height: 24px;
  cursor: pointer;
  flex-shrink: 0;
}

.toggle input {
  opacity: 0;
  width: 0;
  height: 0;
}

.toggle input:disabled + .toggle-slider {
  opacity: 0.4;
  cursor: not-allowed;
}

.toggle-slider {
  position: absolute;
  inset: 0;
  background: var(--bg-hover);
  border-radius: 12px;
  transition: var(--transition-fast);
}

.toggle-slider::before {
  content: '';
  position: absolute;
  left: 3px;
  top: 3px;
  width: 18px;
  height: 18px;
  background: var(--text-muted);
  border-radius: 50%;
  transition: var(--transition-fast);
}

.toggle input:checked + .toggle-slider {
  background: var(--primary-500);
}

.toggle input:checked + .toggle-slider::before {
  transform: translateX(20px);
  background: white;
}

.config-fields {
  display: flex;
  gap: 20px;
}

.field {
  display: flex;
  align-items: center;
  gap: 8px;
}

.field label {
  font-size: 14px;
  color: var(--text-secondary);
  white-space: nowrap;
}

.field select {
  background: var(--bg-secondary);
  border: 1px solid var(--bg-hover);
  border-radius: var(--radius-sm);
  color: var(--text-primary);
  padding: 6px 10px;
  font-size: 14px;
}

.input.sm {
  width: 80px;
  text-align: center;
}

.add-row {
  display: flex;
  gap: 8px;
  margin: 16px 0;
}

.add-row .input {
  flex: 1;
}

.select-sm {
  background: var(--bg-secondary);
  border: 1px solid var(--bg-hover);
  border-radius: var(--radius-sm);
  color: var(--text-primary);
  padding: 6px 10px;
  font-size: 14px;
}

.list-container {
  display: flex;
  flex-direction: column;
}

.list-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 0;
  border-bottom: 1px solid var(--bg-hover);
}

.list-item:last-child {
  border-bottom: none;
}

.item-info {
  display: flex;
  align-items: center;
  gap: 10px;
}

.item-ip {
  font-family: monospace;
  font-size: 14px;
}

.badge-type {
  background: rgba(139, 92, 246, 0.15);
  color: #a78bfa;
}

.item-time {
  font-size: 12px;
  color: var(--text-muted);
}
</style>
