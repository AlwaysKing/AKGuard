import { createRouter, createWebHistory } from 'vue-router'
import { ref } from 'vue'
import AppLayout from '../components/AppLayout.vue'
import api from '../api'

const adminRoutes = ['Dashboard', 'Auth', 'Networks', 'Domains', 'Blacklist', 'AccessLog', 'AuditLog', 'Sessions', 'AdminSettings']

export const adminVerified = ref(false)
export const siteTitle = ref(localStorage.getItem('ak-site-title') || 'AKGuard')

const routes = [
  {
    path: '/login',
    name: 'AuthLogin',
    component: () => import('../views/AuthLogin.vue'),
  },
  {
    path: '/adminlogin',
    name: 'AdminLogin',
    component: () => import('../views/AdminLogin.vue'),
  },
  {
    path: '/',
    component: AppLayout,
    redirect: '/dashboard',
    children: [
      {
        path: 'dashboard',
        name: 'Dashboard',
        component: () => import('../views/Dashboard.vue'),
        meta: { icon: '📊', title: '仪表盘' },
      },
      // 认证管理
      {
        path: 'auth',
        name: 'Auth',
        component: () => import('../views/AuthManagement.vue'),
        meta: { icon: '🔑', title: '认证方式', group: '认证管理' },
      },
      {
        path: 'networks',
        name: 'Networks',
        component: () => import('../views/Networks.vue'),
        meta: { icon: '🌐', title: '网段管理', group: '认证管理' },
      },
      {
        path: 'domains',
        name: 'Domains',
        component: () => import('../views/Domains.vue'),
        meta: { icon: '🛡', title: '域名策略', group: '认证管理' },
      },
      {
        path: 'blacklist',
        name: 'Blacklist',
        component: () => import('../views/Blacklist.vue'),
        meta: { icon: '🚫', title: '黑名单', group: '监控' },
      },
      // 监控
      {
        path: 'logs',
        name: 'AccessLog',
        component: () => import('../views/AccessLog.vue'),
        meta: { icon: '📋', title: '访问日志', group: '监控' },
      },
      {
        path: 'audit',
        name: 'AuditLog',
        component: () => import('../views/AuditLog.vue'),
        meta: { icon: '📝', title: '操作审计', group: '监控' },
      },
      {
        path: 'sessions',
        name: 'Sessions',
        component: () => import('../views/Sessions.vue'),
        meta: { icon: '👤', title: '会话管理', group: '监控' },
      },
      // 系统设置（底部）
      {
        path: 'admin-settings',
        name: 'AdminSettings',
        component: () => import('../views/AdminSettings.vue'),
        meta: { icon: '🔐', title: '管理设置' },
      },
    ],
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.beforeEach(async (to) => {
  // 登录页不需要验证
  if (to.name === 'AdminLogin' || to.name === 'AuthLogin') return true

  // 管理页面需要验证 admin session
  if (adminRoutes.includes(to.name)) {
    if (!adminVerified.value) {
      try {
        await api.getDashboard()
        adminVerified.value = true
      } catch {
        return { name: 'AdminLogin' }
      }
    }
  }

  return true
})

export default router
