<template>
  <div class="admin-password-page">
    <!-- 站点标题 -->
    <div class="card">
      <h3>站点标题</h3>
      <p class="method-desc">设置登录页和管理面板中显示的名称</p>
      <div class="method-body">
        <div class="single-row">
          <input v-model="siteTitle" type="text" class="input" placeholder="输入站点标题" @keyup.enter="saveSiteTitle">
          <button class="btn btn-primary" :disabled="loading || !siteTitle" @click="saveSiteTitle">保存</button>
        </div>
      </div>
    </div>

    <!-- 密码认证 -->
    <div class="card">
      <div class="method-header">
        <div class="method-info">
          <h3>密码认证</h3>
          <p class="method-desc">管理员通过密码登录管理面板</p>
        </div>
        <label class="toggle">
          <input type="checkbox" :checked="passwordEnabled" :disabled="passwordEnabled && !barkEnabled" @change="toggleMethod('password')">
          <span class="toggle-slider"></span>
        </label>
      </div>

      <div v-if="passwordEnabled" class="method-body">
        <form @submit.prevent="updateAdminPwd">
          <div class="form-group">
            <label>当前密码</label>
            <input v-model="adminPwd.current" type="password" class="input">
          </div>
          <div class="form-group">
            <label>新密码</label>
            <input v-model="adminPwd.new" type="password" class="input">
          </div>
          <div class="form-group">
            <label>确认新密码</label>
            <input v-model="adminPwd.confirm" type="password" class="input">
          </div>
          <div class="form-actions">
            <button class="btn btn-primary" :disabled="loading">更新</button>
          </div>
        </form>
      </div>
    </div>

    <!-- Bark 推送认证 -->
    <div class="card">
      <div class="method-header">
        <div class="method-info">
          <h3>Bark 推送认证</h3>
          <p class="method-desc">通过 Bark 推送一次性验证码完成管理员登录</p>
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
import { ref, reactive, onMounted, nextTick } from 'vue'
import api from '../api'
import { useToast } from '../composables/toast'
import { siteTitle } from '../router'

const toast = useToast()
const loading = ref(false)
const passwordEnabled = ref(true)
const barkEnabled = ref(false)
const barkUrl = ref('')
const barkVerified = ref(false)
const adminPwd = reactive({ current: '', new: '', confirm: '' })

// 验证对话框
const showVerifyDialog = ref(false)
const verifyCode = ref('')
const verifyInput = ref(null)

onMounted(async () => {
  try {
    const data = await api.getConfig()
    barkUrl.value = data.bark_url || ''
    barkVerified.value = !!barkUrl.value
    siteTitle.value = data.site_title || 'AKGuard'
    passwordEnabled.value = data.admin_password_login ?? true
    barkEnabled.value = data.admin_bark_login ?? false
  } catch {}
})

async function saveSiteTitle() {
  if (!siteTitle.value) return
  loading.value = true
  try {
    await api.updateSiteTitle(siteTitle.value)
    localStorage.setItem('ak-site-title', siteTitle.value)
    document.title = siteTitle.value
    toast.success('站点标题已更新')
  } catch (e) {
    toast.error(e.message || '更新失败')
  } finally {
    loading.value = false
  }
}

async function toggleMethod(type) {
  let newPwd = passwordEnabled.value
  let newBark = barkEnabled.value

  if (type === 'password') {
    newPwd = !passwordEnabled.value
  } else {
    newBark = !barkEnabled.value
  }

  loading.value = true
  try {
    await api.updateAdminLoginMethods(newPwd, newBark)
    passwordEnabled.value = newPwd
    barkEnabled.value = newBark
    toast.success(type === 'password'
      ? (newPwd ? '密码认证已启用' : '密码认证已关闭')
      : (newBark ? 'Bark 推送认证已启用' : 'Bark 推送认证已关闭')
    )
  } catch (e) {
    toast.error(e.message || '操作失败')
  } finally {
    loading.value = false
  }
}

async function updateAdminPwd() {
  if (!adminPwd.current || !adminPwd.new || !adminPwd.confirm) {
    toast.error('请填写所有字段')
    return
  }
  if (adminPwd.new !== adminPwd.confirm) {
    toast.error('两次输入的密码不一致')
    return
  }
  loading.value = true
  try {
    await api.updateAdminPassword(adminPwd.current, adminPwd.new, adminPwd.confirm)
    toast.success('管理密码已更新')
    adminPwd.current = ''
    adminPwd.new = ''
    adminPwd.confirm = ''
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
</script>

<style scoped>
.admin-password-page {
  display: flex;
  flex-direction: column;
  gap: 20px;
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

.verified-hint {
  margin-top: 8px;
  font-size: 13px;
  color: var(--success);
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
