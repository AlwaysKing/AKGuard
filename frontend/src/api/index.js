const BASE = ''

async function request(url, options = {}) {
  const res = await fetch(BASE + url, {
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      ...options.headers,
    },
    ...options,
  })

  const data = await res.json().catch(() => null)

  if (res.status === 401) {
    // 未认证，跳转到登录页（auth-login 页面除外）
    const path = window.location.pathname
    if (path !== '/adminlogin' && path !== '/login') {
      window.location.href = '/adminlogin'
    }
    throw new Error('unauthorized')
  }

  if (!res.ok) {
    throw new Error(data?.error || `HTTP ${res.status}`)
  }

  return data
}

export const api = {
  // 认证
  authLogin: (password, otpCode) =>
    request('/api/auth/login', { method: 'POST', body: JSON.stringify({ password, otp_code: otpCode }) }),
  authLogout: () =>
    request('/api/auth/logout', { method: 'POST' }),
  adminLogin: (password, otpCode) =>
    request('/api/admin/login', { method: 'POST', body: JSON.stringify({ password, otp_code: otpCode }) }),
  adminLogout: () =>
    request('/api/admin/logout', { method: 'POST' }),
  sendOTP: (type) =>
    request('/api/otp/send', { method: 'POST', body: JSON.stringify({ type }) }),

  // 配置
  getLoginMethods: () => request('/api/login-methods'),
  getConfig: () => request('/api/config'),
  updateAuthPassword: (newPwd, confirmPwd) =>
    request('/api/config/auth-password', { method: 'PUT', body: JSON.stringify({ new_password: newPwd, confirm_password: confirmPwd }) }),
  updateAdminPassword: (currentPwd, newPwd, confirmPwd) =>
    request('/api/config/admin-password', { method: 'PUT', body: JSON.stringify({ current_password: currentPwd, new_password: newPwd, confirm_password: confirmPwd }) }),
  updateBark: (barkUrl) =>
    request('/api/config/bark', { method: 'PUT', body: JSON.stringify({ bark_url: barkUrl }) }),
  testBark: (barkUrl) =>
    request('/api/config/test-bark', { method: 'POST', body: JSON.stringify({ bark_url: barkUrl }) }),
  confirmBark: (barkUrl, code) =>
    request('/api/config/confirm-bark', { method: 'POST', body: JSON.stringify({ bark_url: barkUrl, code }) }),
  updateNetworks: (networks) =>
    request('/api/config/networks', { method: 'PUT', body: JSON.stringify({ networks }) }),
  updateDomains: (domains) =>
    request('/api/config/domains', { method: 'PUT', body: JSON.stringify({ domains }) }),
  updateDefaultPolicy: (internal, external) =>
    request('/api/config/default-policy', { method: 'PUT', body: JSON.stringify({ internal, external }) }),
  updateAdminLoginMethods: (passwordLogin, barkLogin) =>
    request('/api/config/admin-login-methods', { method: 'PUT', body: JSON.stringify({ password_login: passwordLogin, bark_login: barkLogin }) }),
  updateAuthLoginMethods: (passwordLogin, barkLogin) =>
    request('/api/config/auth-login-methods', { method: 'PUT', body: JSON.stringify({ password_login: passwordLogin, bark_login: barkLogin }) }),
  updateAuthBanConfig: (config) =>
    request('/api/config/auth-ban', { method: 'PUT', body: JSON.stringify(config) }),
  updateAdminBanConfig: (config) =>
    request('/api/config/admin-ban', { method: 'PUT', body: JSON.stringify(config) }),
  updateSiteTitle: (title) =>
    request('/api/config/site-title', { method: 'PUT', body: JSON.stringify({ title }) }),

  // 黑名单
  getBlacklist: (type) => request(`/api/blacklist?type=${type}`),
  addBlacklist: (ip, type, durationSec) =>
    request('/api/blacklist/add', { method: 'POST', body: JSON.stringify({ ip, type, duration_sec: durationSec }) }),
  deleteBlacklist: (id) =>
    request(`/api/blacklist/${id}`, { method: 'DELETE' }),

  // 仪表盘
  getDashboard: () => request('/api/dashboard'),

  // 日志
  getAccessLogs: (page = 1, pageSize = 20, host = '') =>
    request(`/api/logs/access?page=${page}&page_size=${pageSize}${host ? '&host=' + host : ''}`),
  getAccessLogStats: () => request('/api/logs/access/stats'),
  getAuditLogs: (page = 1, pageSize = 20) =>
    request(`/api/logs/audit?page=${page}&page_size=${pageSize}`),

  // 会话
  getSessions: () => request('/api/sessions'),
  deleteSession: (id) =>
    request(`/api/sessions/${id}`, { method: 'DELETE' }),
}

export default api
