<template>
  <div class="auth-management">
    <!-- 默认密码警告 -->
    <div v-if="isDefaultAuth" class="card warning-card">
      <p>⚠️ 您正在使用默认认证密码，请立即修改。</p>
    </div>

    <!-- 密码认证 -->
    <div class="card">
      <div class="method-header">
        <div class="method-info">
          <h3>密码认证</h3>
          <p class="method-desc">用户通过密码登录获取访问权限</p>
        </div>
        <label class="toggle">
          <input type="checkbox" :checked="passwordEnabled" :disabled="passwordEnabled && !barkEnabled" @change="toggleMethod('password')">
          <span class="toggle-slider"></span>
        </label>
      </div>

      <div v-if="passwordEnabled" class="method-body">
        <form @submit.prevent="updateAuthPwd" class="single-row">
          <input v-model="authPwd" type="text" class="input" placeholder="输入新密码">
          <button class="btn btn-primary" :disabled="loading">更新密码</button>
        </form>
      </div>
    </div>

    <!-- Bark 推送认证 -->
    <div class="card">
      <div class="method-header">
        <div class="method-info">
          <h3>Bark 推送认证</h3>
          <p class="method-desc">通过 Bark 推送一次性验证码完成登录</p>
        </div>
        <label class="toggle">
          <input type="checkbox" :checked="barkEnabled" :disabled="(!barkVerified && !barkEnabled) || (barkEnabled && !passwordEnabled)" @change="toggleMethod('bark')">
          <span class="toggle-slider"></span>
        </label>
      </div>

      <div class="method-body">
        <div class="single-row">
          <input v-model="barkUrl" type="text" class="input" placeholder="例：https://api.day.app/yourkey">
          <button class="btn btn-primary" :disabled="loading || !barkUrl" @click="startBarkVerify">
            {{ barkVerified ? '重新验证' : '保存并验证' }}
          </button>
        </div>
        <p v-if="barkVerified" class="verified-hint">✓ 地址已验证</p>
      </div>
    </div>

    <!-- API Key 认证 -->
    <div class="card">
      <div class="method-header">
        <div class="method-info">
          <h3>API Key 认证</h3>
          <p class="method-desc">供程序调用使用，通过 X-AK-Token header 或 ak_token URL 参数直接通过鉴权</p>
        </div>
        <label class="toggle">
          <input type="checkbox" :checked="apikeyEnabled" @change="toggleMethod('apikey')">
          <span class="toggle-slider"></span>
        </label>
      </div>

      <div v-if="apikeyEnabled" class="method-body">
        <div class="single-row">
          <input :value="apiKey" type="text" class="input apikey-display" readonly ref="apiKeyInput">
          <button class="btn btn-ghost" @click="copyApiKey">复制</button>
          <button class="btn btn-primary" :disabled="loading" @click="regenerateKey">重新生成</button>
        </div>
        <p class="method-desc" style="margin-top:8px">程序调用时添加 header：<code>X-AK-Token: &lt;api_key&gt;</code> 或 URL 参数：<code>?ak_token=&lt;api_key&gt;</code></p>
      </div>
    </div>

    <!-- Token 自动续签 -->
    <div class="card">
      <div class="method-header">
        <div class="method-info">
          <h3>Token 自动续签</h3>
          <p class="method-desc">token 过期后的宽限期内，访问登录页可自动续签，无需重新输入密码</p>
        </div>
      </div>
      <div class="method-body">
        <form @submit.prevent="updateGracePeriod" class="single-row">
          <input v-model.number="gracePeriodHours" type="number" class="input" placeholder="0" min="0" step="1">
          <span class="input-suffix">小时</span>
          <button class="btn btn-primary" :disabled="loading">保存</button>
        </form>
        <p class="method-desc" style="margin-top:8px">设为 0 表示关闭自动续签功能</p>
      </div>
    </div>

    <!-- Bark 验证对话框 -->
    <Teleport to="body">
      <div v-if="showVerifyDialog" class="dialog-overlay" @click.self="showVerifyDialog = false">
        <div class="dialog">
          <h3>验证 Bark 推送</h3>
          <p class="dialog-desc">验证码已发送到您的手机，请输入收到的6位数字</p>
          <input
            v-model="verifyCode"
            class="input dialog-input"
            placeholder="输入验证码"
            maxlength="6"
            ref="verifyInput"
            @keyup.enter="confirmBark"
          >
          <div class="dialog-actions">
            <button class="btn btn-ghost" @click="showVerifyDialog = false">取消</button>
            <button class="btn btn-primary" :disabled="!verifyCode || loading" @click="confirmBark">确认</button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<script setup>
import { ref, onMounted, nextTick } from 'vue'
import api from '../api'
import { useToast } from '../composables/toast'

const toast = useToast()
const loading = ref(false)
const isDefaultAuth = ref(false)
const passwordEnabled = ref(true)
const barkEnabled = ref(false)
const barkUrl = ref('')
const barkVerified = ref(false)
const authPwd = ref('')
const gracePeriodHours = ref(0)
const apikeyEnabled = ref(false)
const apiKey = ref('')
const apiKeyInput = ref(null)

// 验证对话框
const showVerifyDialog = ref(false)
const verifyCode = ref('')
const verifyInput = ref(null)

onMounted(async () => {
  try {
    const data = await api.getConfig()
    isDefaultAuth.value = data.is_default_auth || false
    barkUrl.value = data.bark_url || ''
    barkVerified.value = !!barkUrl.value
    passwordEnabled.value = data.auth_password_login ?? true
    barkEnabled.value = data.auth_bark_login ?? false
    gracePeriodHours.value = Math.round((data.token_grace_period || 0) / 3600)
    apikeyEnabled.value = data.auth_apikey_login ?? false
    apiKey.value = data.api_key || ''
  } catch {}
})

async function toggleMethod(type) {
  let newPwd = passwordEnabled.value
  let newBark = barkEnabled.value
  let newApikey = apikeyEnabled.value

  if (type === 'password') {
    newPwd = !passwordEnabled.value
  } else if (type === 'bark') {
    newBark = !barkEnabled.value
  } else if (type === 'apikey') {
    newApikey = !apikeyEnabled.value
  }

  loading.value = true
  try {
    await api.updateAuthLoginMethods(newPwd, newBark, newApikey)
    passwordEnabled.value = newPwd
    barkEnabled.value = newBark
    apikeyEnabled.value = newApikey
    const labels = { password: '密码认证', bark: 'Bark 推送认证', apikey: 'API Key 认证' }
    const enabled = { password: newPwd, bark: newBark, apikey: newApikey }
    toast.success(enabled[type] ? `${labels[type]}已启用` : `${labels[type]}已关闭`)
  } catch (e) {
    toast.error(e.message || '操作失败')
  } finally {
    loading.value = false
  }
}

async function updateAuthPwd() {
  if (!authPwd.value) {
    toast.error('请输入密码')
    return
  }
  loading.value = true
  try {
    await api.updateAuthPassword(authPwd.value, authPwd.value)
    toast.success('认证密码已更新')
    authPwd.value = ''
    isDefaultAuth.value = false
  } catch (e) {
    toast.error(e.message || '更新失败')
  } finally {
    loading.value = false
  }
}

async function startBarkVerify() {
  if (!barkUrl.value) return
  loading.value = true
  try {
    await api.testBark(barkUrl.value)
    toast.success('验证码已发送')
    showVerifyDialog.value = true
    verifyCode.value = ''
    await nextTick()
    verifyInput.value?.focus()
  } catch (e) {
    toast.error(e.message || '发送失败，请检查地址')
  } finally {
    loading.value = false
  }
}

async function confirmBark() {
  if (!verifyCode.value) return
  loading.value = true
  try {
    await api.confirmBark(barkUrl.value, verifyCode.value)
    toast.success('Bark 地址验证通过，已保存')
    barkVerified.value = true
    showVerifyDialog.value = false
    verifyCode.value = ''
  } catch (e) {
    toast.error(e.message || '验证码错误')
  } finally {
    loading.value = false
  }
}

async function updateGracePeriod() {
  loading.value = true
  try {
    const seconds = Math.max(0, (gracePeriodHours.value || 0)) * 3600
    await api.updateTokenGracePeriod(seconds)
    toast.success(seconds > 0 ? `宽限期已设为 ${gracePeriodHours.value} 小时` : '自动续签已关闭')
  } catch (e) {
    toast.error(e.message || '保存失败')
  } finally {
    loading.value = false
  }
}

function copyApiKey() {
  navigator.clipboard.writeText(apiKey.value).then(() => {
    toast.success('API Key 已复制到剪贴板')
  }).catch(() => {
    apiKeyInput.value?.select()
    document.execCommand('copy')
    toast.success('API Key 已复制到剪贴板')
  })
}

async function regenerateKey() {
  loading.value = true
  try {
    const data = await api.regenerateApiKey()
    apiKey.value = data.api_key
    toast.success('API Key 已重新生成')
  } catch (e) {
    toast.error(e.message || '操作失败')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.auth-management {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.warning-card {
  border: 1px solid rgba(251, 191, 36, 0.3);
  background: rgba(251, 191, 36, 0.05);
}

.warning-card p {
  color: var(--warning);
  font-size: 14px;
}

.method-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.method-info h3 {
  margin: 0;
  font-size: 16px;
}

.method-desc {
  font-size: 13px;
  color: var(--text-muted);
  margin: 4px 0 0;
}

.method-body {
  margin-top: 20px;
  padding-top: 20px;
  border-top: 1px solid var(--bg-hover);
}

.single-row {
  display: flex;
  gap: 10px;
  align-items: center;
}

.single-row .input {
  flex: 1;
}

.input-suffix {
  font-size: 14px;
  color: var(--text-muted);
  white-space: nowrap;
}

/* 隐藏 number 输入框的原生上下箭头 */
input[type="number"]::-webkit-inner-spin-button,
input[type="number"]::-webkit-outer-spin-button {
  -webkit-appearance: none;
  margin: 0;
}
input[type="number"] {
  -moz-appearance: textfield;
}

.verified-hint {
  margin-top: 8px;
  font-size: 13px;
  color: var(--success);
}

.apikey-display {
  font-family: monospace;
  font-size: 14px;
  letter-spacing: 1px;
}

.apikey-display[readonly] {
  opacity: 0.85;
  cursor: default;
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

/* 对话框 */
.dialog-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 9000;
}

.dialog {
  background: var(--bg-card);
  border-radius: var(--radius-lg);
  padding: 28px 32px;
  width: 380px;
  box-shadow: var(--shadow-lg);
}

.dialog h3 {
  margin: 0 0 8px;
  font-size: 18px;
}

.dialog-desc {
  font-size: 13px;
  color: var(--text-muted);
  margin-bottom: 20px;
}

.dialog-input {
  font-size: 20px;
  text-align: center;
  letter-spacing: 8px;
  padding: 14px;
}

.dialog-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 20px;
}
</style>
