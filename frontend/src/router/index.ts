import { createRouter, createWebHistory } from 'vue-router'
import { useUserStore } from '@/stores/user'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/login',
      name: 'Login',
      component: () => import('@/views/Login.vue'),
      meta: { requiresAuth: false }
    },
    {
      path: '/init',
      name: 'Init',
      component: () => import('@/views/Init.vue'),
      meta: { requiresAuth: false }
    },
    {
      path: '/',
      component: () => import('@/views/Layout.vue'),
      redirect: '/dashboard',
      children: [
        {
          path: 'dashboard',
          name: 'Dashboard',
          component: () => import('@/views/Dashboard.vue'),
          meta: { title: '仪表盘' }
        },
        {
          path: 'inbounds',
          name: 'Inbounds',
          component: () => import('@/views/Inbounds.vue'),
          meta: { title: '入站管理' }
        },
        {
          path: 'clients',
          name: 'Clients',
          component: () => import('@/views/Clients.vue'),
          meta: { title: '用户管理' }
        },
        {
          path: 'certificates',
          name: 'Certificates',
          component: () => import('@/views/Certificates.vue'),
          meta: { title: '证书管理', requiresAdmin: true }
        },
        {
          path: 'traffic',
          name: 'Traffic',
          component: () => import('@/views/Traffic.vue'),
          meta: { title: '流量统计' }
        },
        {
          path: 'settings',
          name: 'Settings',
          component: () => import('@/views/Settings.vue'),
          meta: { title: '系统设置', requiresAdmin: true }
        },
        {
          path: 'audit',
          name: 'Audit',
          component: () => import('@/views/Audit.vue'),
          meta: { title: '审计日志', requiresAdmin: true }
        }
      ]
    },
    {
      path: '/:pathMatch(.*)*',
      name: 'NotFound',
      redirect: '/dashboard'
    }
  ]
})

// 路由守卫
router.beforeEach((to, _from, next) => {
  const userStore = useUserStore()

  // 不需要认证的路由
  if (to.meta.requiresAuth === false) {
    next()
    return
  }

  // 需要认证但未登录
  if (!userStore.token) {
    next('/login')
    return
  }

  // 需要管理员权限
  if (to.meta.requiresAdmin && !userStore.isAdmin) {
    next('/dashboard')
    return
  }

  next()
})

export default router
