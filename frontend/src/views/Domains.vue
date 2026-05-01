<template>
  <div>
    <!-- 默认策略 -->
    <div class="card" style="margin-bottom: 20px">
      <h3>默认策略</h3>
      <p style="font-size: 13px; color: var(--text-muted); margin: 4px 0 16px;">未配置域名策略时的默认行为</p>
      <div class="policy-row">
        <div class="form-group" style="margin-bottom: 0">
          <label>内网</label>
          <select v-model="defaultPolicy.internal" class="input" @change="saveDefaultPolicy">
            <option value="reject">拒绝</option>
            <option value="pass">放行</option>
            <option value="auth">认证</option>
          </select>
        </div>
        <div class="form-group" style="margin-bottom: 0">
          <label>外网</label>
          <select v-model="defaultPolicy.external" class="input" @change="saveDefaultPolicy">
            <option value="reject">拒绝</option>
            <option value="pass">放行</option>
            <option value="auth">认证</option>
          </select>
        </div>
      </div>
    </div>

    <!-- 域名策略列表 -->
    <div class="card">
      <h3 style="margin-bottom: 16px;">域名策略</h3>
      <table v-if="domains.length">
        <thead>
          <tr>
            <th>域名</th>
            <th>内网策略</th>
            <th>外网策略</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(d, i) in domains" :key="i">
            <td><code>{{ d.host }}</code></td>
            <td>
              <select v-model="d.internal" class="input" style="width: auto; padding: 4px 30px 4px 8px;" @change="saveDomains">
                <option value="reject">拒绝</option>
                <option value="pass">放行</option>
                <option value="auth">认证</option>
              </select>
            </td>
            <td>
              <select v-model="d.external" class="input" style="width: auto; padding: 4px 30px 4px 8px;" @change="saveDomains">
                <option value="reject">拒绝</option>
                <option value="pass">放行</option>
                <option value="auth">认证</option>
              </select>
            </td>
            <td>
              <button class="btn btn-ghost btn-sm" @click="removeDomain(i)">删除</button>
            </td>
          </tr>
        </tbody>
      </table>
      <div v-else class="empty-state">暂无域名策略</div>

      <div class="add-domain" style="margin-top: 24px; padding-top: 24px; border-top: 1px solid var(--bg-hover);">
        <div style="display: flex; gap: 8px; align-items: end;">
          <div class="form-group" style="flex: 1; margin-bottom: 0">
            <label>域名</label>
            <input v-model="newDomain.host" class="input" placeholder="例：app.example.com" @keyup.enter="addDomain">
          </div>
          <div class="form-group" style="margin-bottom: 0">
            <label>内网</label>
            <select v-model="newDomain.internal" class="input" style="width: auto;">
              <option value="reject">拒绝</option>
              <option value="pass">放行</option>
              <option value="auth">认证</option>
            </select>
          </div>
          <div class="form-group" style="margin-bottom: 0">
            <label>外网</label>
            <select v-model="newDomain.external" class="input" style="width: auto;">
              <option value="reject">拒绝</option>
              <option value="pass">放行</option>
              <option value="auth">认证</option>
            </select>
          </div>
          <button class="btn btn-primary" @click="addDomain">添加</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import api from '../api'
import { useToast } from '../composables/toast'

const toast = useToast()
const domains = ref([])
const defaultPolicy = reactive({ internal: 'auth', external: 'auth' })
const newDomain = reactive({ host: '', internal: 'pass', external: 'auth' })

onMounted(async () => {
  try {
    const data = await api.getConfig()
    domains.value = (data.domains || []).map(d => ({
      host: d.host,
      internal: d.internal,
      external: d.external,
    }))
    if (data.default_policy) {
      defaultPolicy.internal = data.default_policy.internal
      defaultPolicy.external = data.default_policy.external
    }
  } catch {}
})

async function saveDefaultPolicy() {
  try {
    await api.updateDefaultPolicy(defaultPolicy.internal, defaultPolicy.external)
    toast.success('默认策略已更新')
  } catch (e) {
    toast.error(e.message || '更新失败')
  }
}

async function removeDomain(i) {
  domains.value.splice(i, 1)
  await saveDomains()
}

async function addDomain() {
  if (!newDomain.host) {
    toast.error('请输入域名')
    return
  }
  domains.value.push({ ...newDomain })
  newDomain.host = ''
  await saveDomains()
}

async function saveDomains() {
  try {
    await api.updateDomains(domains.value)
    toast.success('域名策略已更新')
  } catch (e) {
    toast.error(e.message || '更新失败')
  }
}
</script>

<style scoped>
.policy-row {
  display: flex;
  align-items: end;
  gap: 16px;
}

.policy-row .form-group { min-width: 120px; }
</style>
