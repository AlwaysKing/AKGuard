<template>
  <div class="login-page">
    <button class="theme-toggle" @click="toggleTheme">
      {{ isDark ? '☀️' : '🌙' }}
    </button>
    <div class="login-card">
      <h1 class="login-title">{{ siteTitle }}</h1>
      <p class="login-subtitle">管理员登录</p>

      <div v-if="error" class="login-error">{{ error }}</div>

      <!-- 只有一种方式时不显示 Tab -->
      <div v-if="showTabs" class="login-tabs">
        <button v-if="methods.passwordLogin" :class="['tab', { active: mode === 'password' }]" @click="mode = 'password'">密码登录</button>
        <button v-if="methods.barkLogin" :class="['tab', { active: mode === 'otp' }]" @click="mode = 'otp'">Bark 推送</button>
      </div>

      <!-- 密码登录 -->
      <form v-if="mode === 'password'" @submit.prevent="loginWithPassword" :class="{ 'form-no-tabs': !showTabs }">
        <div class="form-group">
          <input v-model="password" type="password" class="input" placeholder="请输入管理密码" ref="pwdInput">
        </div>
        <button class="btn btn-primary login-btn" :disabled="loading">
          {{ loading ? '登录中...' : '登录' }}
        </button>
      </form>

      <!-- OTP 登录 -->
      <form v-if="mode === 'otp'" @submit.prevent="loginWithOTP" :class="{ 'form-no-tabs': !showTabs }">
        <div class="form-group">
          <div class="otp-row">
            <input v-model="otpCode" class="input" placeholder="请输入6位验证码" maxlength="6">
            <button class="btn btn-ghost btn-sm" @click.prevent="sendCode" :disabled="cooldown > 0">
              {{ cooldown > 0 ? `${cooldown}s` : '发送验证码' }}
            </button>
          </div>
        </div>
        <button class="btn btn-primary login-btn" :disabled="loading">
          {{ loading ? '登录中...' : '登录' }}
        </button>
      </form>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import api from '../api'
import { adminVerified, siteTitle } from '../router'

const router = useRouter()
const password = ref('')
const otpCode = ref('')
const mode = ref('password')
const error = ref('')
const loading = ref(false)
const cooldown = ref(0)
let cooldownTimer = null

const pwdInput = ref(null)
const isDark = ref(true)

const methods = reactive({ passwordLogin: true, barkLogin: false })
const showTabs = computed(() => methods.passwordLogin && methods.barkLogin)

onMounted(async () => {
  const saved = localStorage.getItem('ak-theme')
  if (saved === 'light') {
    isDark.value = false
    document.documentElement.setAttribute('data-theme', 'light')
  }

  try {
    const data = await api.getLoginMethods()
    methods.passwordLogin = data.admin_password_login ?? true
    methods.barkLogin = data.admin_bark_login ?? false
    if (data.site_title) {
      siteTitle.value = data.site_title
      localStorage.setItem('ak-site-title', data.site_title)
    }
    document.title = siteTitle.value
    // 自动选择可用方式
    if (!methods.passwordLogin && methods.barkLogin) {
      mode.value = 'otp'
    } else {
      mode.value = 'password'
    }
  } catch {
    // 获取失败时默认显示密码登录
  }

  pwdInput.value?.focus()
})

function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.setAttribute('data-theme', isDark.value ? '' : 'light')
  localStorage.setItem('ak-theme', isDark.value ? 'dark' : 'light')
}

async function loginWithPassword() {
  if (!password.value) return
  loading.value = true
  error.value = ''
  try {
    await api.adminLogin(password.value)
    adminVerified.value = true
    router.push('/dashboard')
  } catch (e) {
    error.value = e.message || '登录失败'
  } finally {
    loading.value = false
  }
}

async function loginWithOTP() {
  if (!otpCode.value) return
  loading.value = true
  error.value = ''
  try {
    await api.adminLogin('', otpCode.value)
    adminVerified.value = true
    router.push('/dashboard')
  } catch (e) {
    error.value = e.message || '登录失败'
  } finally {
    loading.value = false
  }
}

async function sendCode() {
  try {
    await api.sendOTP('admin')
    cooldown.value = 30
    cooldownTimer = setInterval(() => {
      cooldown.value--
      if (cooldown.value <= 0) clearInterval(cooldownTimer)
    }, 1000)
  } catch (e) {
    error.value = e.message || '发送失败'
  }
}
</script>

<style scoped>
.login-page {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  background: linear-gradient(135deg, var(--bg-primary) 0%, var(--bg-secondary) 50%, var(--bg-hover) 100%);
}

.theme-toggle {
  position: fixed;
  top: 16px;
  right: 16px;
  background: var(--bg-card);
  border: 1px solid var(--bg-hover);
  border-radius: 50%;
  width: 40px;
  height: 40px;
  font-size: 18px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background var(--transition-fast);
}

.theme-toggle:hover {
  background: var(--bg-hover);
}

.login-card {
  width: 380px;
  background: var(--bg-card);
  border-radius: var(--radius-lg);
  padding: 40px 32px;
  box-shadow: var(--shadow-lg);
}

.login-title {
  font-size: 28px;
  font-weight: 700;
  text-align: center;
  background: linear-gradient(135deg, var(--primary-400), var(--primary-300));
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.login-subtitle {
  text-align: center;
  color: var(--text-muted);
  margin-top: 4px;
  font-size: 14px;
}

.login-error {
  margin: 16px 0;
  padding: 10px 14px;
  background: rgba(248, 113, 113, 0.1);
  border: 1px solid rgba(248, 113, 113, 0.2);
  border-radius: var(--radius-sm);
  color: var(--error);
  font-size: 13px;
}

.login-tabs {
  display: flex;
  gap: 0;
  margin-bottom: 20px;
  border-bottom: 1px solid var(--bg-hover);
}

.tab {
  flex: 1;
  padding: 10px;
  background: none;
  border: none;
  border-bottom: 2px solid transparent;
  color: var(--text-muted);
  font-size: 14px;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.tab:hover { color: var(--text-secondary); }
.tab.active {
  color: var(--primary-400);
  border-bottom-color: var(--primary-400);
}

.login-btn {
  width: 100%;
  margin-top: 8px;
}

.otp-row {
  display: flex;
  gap: 8px;
}

.otp-row .input { flex: 1; }
.form-no-tabs { margin-top: 24px; }
</style>
