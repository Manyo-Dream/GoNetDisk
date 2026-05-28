import { createRouter, createWebHashHistory } from 'vue-router'
import { isAuthenticated } from '../api/index.js'

import LoginView from '../views/LoginView.vue'
import RegisterView from '../views/RegisterView.vue'
import ShareDownloadView from '../views/ShareDownloadView.vue'
import FilesView from '../views/FilesView.vue'
import TrashView from '../views/TrashView.vue'
import SharesView from '../views/SharesView.vue'
import SettingsView from '../views/SettingsView.vue'

const routes = [
  { path: '/login', name: 'login', component: LoginView, meta: { guest: true } },
  { path: '/register', name: 'register', component: RegisterView, meta: { guest: true } },
  { path: '/share', name: 'share-download', component: ShareDownloadView, meta: { public: true } },
  { path: '/files', name: 'files', component: FilesView },
  { path: '/trash', name: 'trash', component: TrashView },
  { path: '/shares', name: 'shares', component: SharesView },
  { path: '/settings', name: 'settings', component: SettingsView },
  { path: '/', redirect: '/files' },
  { path: '/:pathMatch(.*)*', redirect: '/files' },
]

const router = createRouter({
  history: createWebHashHistory(),
  routes,
})

router.beforeEach((to, from, next) => {
  if (to.meta.public) {
    return next()
  }
  if (to.meta.guest && isAuthenticated()) {
    return next('/files')
  }
  if (!to.meta.guest && !isAuthenticated()) {
    return next({ path: '/login', query: { redirect: to.fullPath } })
  }
  next()
})

export default router
