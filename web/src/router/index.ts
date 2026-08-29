import { createRouter, createWebHistory } from 'vue-router'
import { useUserStore } from '../stores/user'

declare module 'vue-router' {
  interface RouteMeta {
    public?: boolean
    requiresAuth?: boolean // 仅 admin
    requiresAuthUser?: boolean // 任何登录用户(非匿名)
  }
}

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
    path: '/space',
    component: () => import('../layouts/SpaceLayout.vue'),
    meta: { requiresAuthUser: true },
    children: [
      { path: '', redirect: '/space/notes' },
      { path: 'notes', component: () => import('../views/space/MyNoteList.vue') },
      { path: 'notes/new', component: () => import('../views/space/MyNoteEditor.vue') },
      { path: 'notes/:id/edit', component: () => import('../views/space/MyNoteEditor.vue') },
      { path: 'profile', component: () => import('../views/space/Profile.vue') },
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
  const needAuth = to.meta.requiresAuth || to.meta.requiresAuthUser

  if (needAuth && !userStore.token) {
    return { path: '/login', query: { redirect: to.fullPath } }
  }
  if (to.meta.requiresAuth && userStore.token && !userStore.isAdmin) {
    // 仅 admin 路由: 非 admin 踢到 /space 或 /blog(优先 /space, 登录用户至少能进个人空间)
    return { path: userStore.token ? '/space/notes' : '/blog' }
  }
  if (to.path === '/login' && userStore.token) {
    return userStore.isAdmin ? { path: '/admin/articles' } : { path: '/space/notes' }
  }
})

export default router