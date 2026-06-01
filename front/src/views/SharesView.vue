<script setup>
import { ref, onMounted, inject } from 'vue'
import { shareApi } from '../api/index.js'
import { formatDate } from '../utils/format.js'
import { useI18n } from '../i18n/index.js'
import ConfirmDialog from '../components/ConfirmDialog.vue'

const toast = inject('toast')
const { t } = useI18n()

const shares = ref([])
const loading = ref(false)

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
    confirmText: confirmText || t('common.confirm'),
    onConfirm: cb,
  }
}

function onConfirm() {
  const cb = confirmData.value.onConfirm
  confirmData.value.visible = false
  if (cb) cb()
}

async function fetchShares() {
  loading.value = true
  try {
    const res = await shareApi.list({ page: 1, page_size: 50 })
    shares.value = res.list || []
  } catch (e) {
    console.error(e)
  } finally {
    loading.value = false
  }
}

async function copyLink(item, withCode = false) {
  let link = `${window.location.origin}/#/share?code=${item.share_code}`
  if (withCode && item.code) {
    link += `&pwd=${encodeURIComponent(item.code)}`
  }
  try {
    await navigator.clipboard.writeText(link)
  } catch (e) {
    /* clipboard blocked */
  }
  toast(`[<] Link copied: …/${item.share_code}${withCode && item.code ? ' [+pwd]' : ''}`)
}

async function revokeShare(item) {
  showConfirm(
    t('shares.revokeTitle'),
    t('shares.revokeMessage', { name: item.file_name || item.share_code }),
    true,
    t('shares.revokeBtn'),
    async () => {
      try {
        await shareApi.revoke(item.share_code)
        fetchShares()
      } catch (e) {
        toast('! ' + e.message)
      }
    }
  )
}

function statusBadge(item) {
  if (item.expire_at) {
    const exp = new Date(item.expire_at)
    if (exp < new Date()) return { text: t('shares.expired'), cls: 'badge-red' }
  }
  return { text: t('shares.active'), cls: 'badge-green' }
}

onMounted(fetchShares)
</script>

<template>
  <div class="shares-view">
    <div class="view-header">
      <h2>&lt;&gt; {{ t('shares.myShares') }}</h2>
      <span class="count">{{ t('shares.links', { count: shares.length }) }}</span>
    </div>

    <div v-if="!loading && shares.length > 0" class="share-list">
      <div v-for="item in shares" :key="item.share_code" class="share-card">
        <div class="share-info">
          <span class="share-name">{{ item.file_name || 'FILE' }}</span>
          <div class="share-meta">
            <span class="code">{{ t('shares.code') }}: {{ item.share_code }}</span>
            <span v-if="item.code" class="pwd">{{ t('shares.pwd') }}: {{ item.code }}</span>
            <span class="views">{{ t('shares.views') }}: {{ item.view_count || 0 }}</span>
          </div>
          <div class="share-date">{{ formatDate(item.created_at) }}</div>
        </div>
        <div class="share-status">
          <span class="badge" :class="statusBadge(item).cls">{{ statusBadge(item).text }}</span>
        </div>
        <div class="share-actions">
          <button class="btn btn-sm" @click="copyLink(item, false)">[&lt;] {{ t('shares.copyBtn') }}</button>
          <button v-if="item.code" class="btn btn-sm" @click="copyLink(item, true)">[&lt;] {{ t('shares.copyPwdBtn') }}</button>
          <button class="btn btn-sm btn-danger" @click="revokeShare(item)">[x] {{ t('shares.revokeBtn') }}</button>
        </div>
      </div>
    </div>

    <div v-else-if="loading" class="empty-state">
      <span class="cursor-blink">_</span> {{ t('common.loading') }}
    </div>

    <div v-else class="empty-state">
      <span class="icon">&lt;&gt;</span>
      <span>{{ t('shares.noShares') }}</span>
      <span style="font-size:var(--fs-xs);color:var(--text-muted)">{{ t('shares.shareHint') }}</span>
    </div>

    <ConfirmDialog
      :visible="confirmData.visible"
      :title="confirmData.title"
      :message="confirmData.message"
      :danger="confirmData.danger"
      :confirm-text="confirmData.confirmText"
      @confirm="onConfirm"
      @close="confirmData.visible = false"
    />
  </div>
</template>

<style scoped>
.shares-view {
  display: flex;
  flex-direction: column;
  gap: var(--gap-4);
}
.view-header {
  display: flex;
  align-items: baseline;
  gap: var(--gap-3);
  padding-bottom: var(--gap-3);
  border-bottom: 1px solid var(--border-faint);
}
h2 {
  font-size: var(--fs-lg);
  font-weight: 600;
  color: var(--cyan);
}
.count {
  font-size: var(--fs-xs);
  color: var(--text-dim);
}
.share-list {
  display: flex;
  flex-direction: column;
  gap: var(--gap-2);
}
.share-card {
  display: flex;
  align-items: center;
  gap: var(--gap-4);
  padding: var(--gap-3) var(--gap-4);
  background: var(--bg-surface);
  border: 1px solid var(--border);
}
.share-info {
  flex: 1;
}
.share-name {
  font-size: var(--fs-sm);
  font-weight: 500;
}
.share-meta {
  display: flex;
  gap: var(--gap-4);
  margin-top: var(--gap-1);
  font-size: var(--fs-xs);
  color: var(--text-dim);
}
.share-date {
  font-size: var(--fs-xs);
  color: var(--text-muted);
  margin-top: 2px;
}
.share-actions {
  display: flex;
  gap: var(--gap-2);
}
</style>
