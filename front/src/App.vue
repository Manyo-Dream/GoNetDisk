<script setup>
import { isAuthenticated, userApi, clearTokens } from './api/index.js'
import { ref, onMounted, provide, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useI18n } from './i18n/index.js'
import AppLayout from './components/AppLayout.vue'
import ConfirmDialog from './components/ConfirmDialog.vue'
import Toast from './components/Toast.vue'

const router = useRouter()
const route = useRoute()
const { t } = useI18n()

const userInfo = ref(null)
const toastRef = ref(null)
provide('userInfo', userInfo)
provide('toast', (msg) => toastRef.value?.show(msg))
const loading = ref(true)
const navItems = computed(() => [
  { label: t('nav.files'),  icon: '[ ]', to: '/files' },
  { label: t('nav.trash'),  icon: '[x]', to: '/trash' },
  { label: t('nav.shares'), icon: '< >', to: '/shares' },
  { label: t('nav.settings'), icon: '[~]', to: '/settings' },
])

onMounted(async () => {
  if (isAuthenticated()) {
    try {
      userInfo.value = await userApi.getInfo()
    } catch (e) {
      if (!localStorage.getItem('access_token')) {
        clearTokens()
        router.push('/login')
      }
    }
  }
  loading.value = false
})

const isGuestRoute = () => ['login', 'register', 'share-download'].includes(route.name)

const confirmData = ref({
  visible: false,
  title: '',
  message: '',
  danger: false,
  confirmText: 'CONFIRM',
  onConfirm: null,
})

function showConfirm(title, message, danger, confirmText, cb) {
  confirmData.value = {
    visible: true,
    title,
    message,
    danger,
    confirmText: confirmText || 'CONFIRM',
    onConfirm: cb,
  }
}

function onConfirm() {
  const cb = confirmData.value.onConfirm
  confirmData.value.visible = false
  if (cb) cb()
}

function handleLogout() {
  showConfirm(
    t('app.logoutTitle'),
    t('app.logoutMessage'),
    false,
    t('common.exit'),
    () => {
      localStorage.removeItem('access_token')
      localStorage.removeItem('refresh_token')
      userInfo.value = null
      router.push('/login')
    }
  )
}
</script>

<template>
  <div v-if="loading" class="loading-screen">
    <span class="cursor-blink">_</span> {{ t('app.initializing') }}
  </div>

  <template v-else-if="isGuestRoute()">
    <router-view />
  </template>

  <AppLayout
    v-else
    :user-info="userInfo"
    :nav-items="navItems"
    @logout="handleLogout"
  >
    <router-view />
  </AppLayout>

  <ConfirmDialog
    :visible="confirmData.visible"
    :title="confirmData.title"
    :message="confirmData.message"
    :danger="confirmData.danger"
    :confirm-text="confirmData.confirmText"
    @confirm="onConfirm"
    @close="confirmData.visible = false"
  />
  <Toast ref="toastRef" />
</template>

<style scoped>
.loading-screen {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: var(--accent);
  font-size: var(--fs-lg);
  gap: var(--gap-2);
}
</style>
