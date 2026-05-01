<template>
  <div class="card">
    <h3>内网网段</h3>
    <p style="font-size: 13px; color: var(--text-muted); margin: 4px 0 16px;">定义内网 IP 地址范围，影响策略判断</p>

    <div class="net-list">
      <div v-for="(net, i) in networks" :key="i" class="net-item">
        <span class="net-cidr">{{ net }}</span>
        <button class="btn btn-ghost btn-sm" @click="removeNet(i)">删除</button>
      </div>
      <div v-if="!networks.length" class="empty-state" style="padding: 20px;">暂无网段</div>
    </div>

    <div class="add-row">
      <input v-model="newNet" class="input" placeholder="例：172.16.0.0/12" @keyup.enter="addNet">
      <button class="btn btn-primary btn-sm" @click="addNet">添加</button>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import api from '../api'
import { useToast } from '../composables/toast'

const toast = useToast()
const networks = ref([])
const newNet = ref('')

onMounted(async () => {
  try {
    const data = await api.getConfig()
    networks.value = [...(data.internal_nets || [])]
  } catch {}
})

function removeNet(i) {
  networks.value.splice(i, 1)
  save()
}

function addNet() {
  const cidr = newNet.value.trim()
  if (!cidr) return
  if (!cidr.includes('/')) {
    toast.error('请输入有效的 CIDR 格式')
    return
  }
  networks.value.push(cidr)
  newNet.value = ''
  save()
}

async function save() {
  try {
    await api.updateNetworks(networks.value)
    toast.success('网段已更新')
  } catch (e) {
    toast.error(e.message || '保存失败')
  }
}
</script>

<style scoped>
.net-list {
  margin-bottom: 16px;
}

.net-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  border-bottom: 1px solid var(--bg-hover);
}

.net-item:last-child { border-bottom: none; }

.net-cidr {
  font-family: var(--font-mono);
  font-size: 14px;
  color: var(--text-primary);
}

.add-row {
  display: flex;
  gap: 8px;
}

.add-row .input { flex: 1; }
</style>
