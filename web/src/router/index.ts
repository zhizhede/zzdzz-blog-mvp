import { createRouter, createWebHistory } from 'vue-router'
import { useUserStore } from '../stores/user'

const routes = [
  {
    path: '/login',
    component: () => import('../views/Login.vue'),
    meta: { public: true },
  },
  {
    path: '/',
    component: () => import('../layouts/AdminLayout.vue'),
    meta: { requiresAuth: true },
    children: [
      { path: '', redirect: '/admin/articles' },
      { path: 'admin/articles', component: () => import('../views/admin/ArticleList.vue') },
      { path: 'admin/articles/new', component: () => import('../views/admin/ArticleEditor.vue') },
      { path: 'admin/articles/:id/edit', component: () => import('../views/admin/ArticleEditor.vue') },
      { path: 'admin/categories', component: () => import('../views/admin/CategoryManage.vue') },
      { path: 'admin/users', component: () => import('../views/admin/UserManage.vue') },
      { path: 'admin/ai', component: () => import('../views/admin/AICenter.vue') },
    ],
  },
  {
    path: '/blog',
    component: () => import('../views/public/BlogLayout.vue'),
    meta: { public: true },
    children: [
      { path: '', component: () => import('../views/public/BlogList.vue') },
      { path: 'a/:id', component: () => import('../views/public/BlogDetail.vue') },
    ],
  },
  { path: '/:pathMatch(.*)*', redirect: '/blog' },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.beforeEach((to) => {
  const userStore = useUserStore()
  if (to.meta.requiresAuth && !userStore.token) {
    return { path: '/login', query: { redirect: to.fullPath } }
  }
  if (to.meta.requiresAuth && userStore.token && !userStore.isAdmin) {
    // 非 admin 不允许进后台, 踢到公开博客
    return { path: '/blog' }
  }
  if (to.path === '/login' && userStore.token) {
    return userStore.isAdmin ? { path: '/admin/articles' } : { path: '/blog' }
  }
})

export default router